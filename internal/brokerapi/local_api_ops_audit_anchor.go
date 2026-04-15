package brokerapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	auditAnchorFailureCodeReceiptInvalid    = "anchor_receipt_invalid"
	auditAnchorFailureCodeRequestInvalid    = "anchor_request_invalid"
	auditAnchorFailureCodeSignerUnavailable = "anchor_signer_unavailable"
	auditAnchorFailureMessageRequestInvalid = "anchor request validation failed"
	auditAnchorFailureMessageReceiptInvalid = "anchor receipt validation failed"
	auditAnchorGatewayFailureMessage        = "audit anchor ledger operation failed"
)

func (s *Service) HandleAuditAnchorSegment(ctx context.Context, req AuditAnchorSegmentRequest, meta RequestContext) (AuditAnchorSegmentResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditAnchorSegmentRequestSchemaPath)
	if errResp != nil {
		return AuditAnchorSegmentResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return AuditAnchorSegmentResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	return s.handleAuditAnchorSegmentValidated(requestCtx, requestID, req)
}

func (s *Service) handleAuditAnchorSegmentValidated(requestCtx context.Context, requestID string, req AuditAnchorSegmentRequest) (AuditAnchorSegmentResponse, *ErrorResponse) {
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return AuditAnchorSegmentResponse{}, &errOut
	}
	if s.auditLedger == nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit ledger unavailable")
		return AuditAnchorSegmentResponse{}, &errOut
	}
	if s.secretsSvc == nil {
		return s.validatedAuditAnchorSegmentResponse(anchorSegmentFailedResponse(requestID, req.SealDigest, auditAnchorFailureCodeSignerUnavailable, "audit anchor signer unavailable"))
	}
	anchorReq, err := s.buildAnchorSegmentRequest(req)
	if err != nil {
		return s.validatedAuditAnchorSegmentResponse(anchorSegmentFailedResponse(requestID, req.SealDigest, auditAnchorFailureCodeRequestInvalid, auditAnchorFailureMessageRequestInvalid))
	}
	result, err := s.auditLedger.AnchorCurrentSegment(anchorReq)
	if err != nil {
		if errors.Is(err, auditd.ErrAnchorReceiptInvalid) {
			return s.validatedAuditAnchorSegmentResponse(anchorSegmentFailedResponse(requestID, req.SealDigest, auditAnchorFailureCodeReceiptInvalid, auditAnchorFailureMessageReceiptInvalid))
		}
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, auditAnchorGatewayFailureMessage)
		return AuditAnchorSegmentResponse{}, &errOut
	}

	resp, errResp := buildAnchorSegmentResponse(requestID, result)
	if errResp != nil {
		return AuditAnchorSegmentResponse{}, errResp
	}

	if req.ExportReceiptCopy {
		resp.ExportedReceiptRef = strings.TrimSpace(s.exportAnchorReceiptCopy(requestID, result))
	}
	return s.validatedAuditAnchorSegmentResponse(resp)
}

func buildAnchorSegmentResponse(requestID string, result auditd.AnchorSegmentResult) (AuditAnchorSegmentResponse, *ErrorResponse) {
	anchorStatus := strings.TrimSpace(result.AnchorStatus)
	if anchorStatus == "" {
		errOut := toErrorResponse(requestID, "gateway_failure", "internal", false, auditAnchorGatewayFailureMessage)
		return AuditAnchorSegmentResponse{}, &errOut
	}
	resp := AuditAnchorSegmentResponse{
		SchemaID:                 "runecode.protocol.v0.AuditAnchorSegmentResponse",
		SchemaVersion:            "0.1.0",
		RequestID:                requestID,
		SealDigest:               result.SealDigest,
		ReceiptDigest:            digestPtr(result.ReceiptDigest),
		VerificationReportDigest: digestPtr(result.VerificationDigest),
		AnchoringStatus:          anchorStatus,
	}
	return resp, nil
}

func anchorSegmentFailedResponse(requestID string, sealDigest trustpolicy.Digest, failureCode, failureMessage string) AuditAnchorSegmentResponse {
	return AuditAnchorSegmentResponse{
		SchemaID:        "runecode.protocol.v0.AuditAnchorSegmentResponse",
		SchemaVersion:   "0.1.0",
		RequestID:       requestID,
		SealDigest:      sealDigest,
		AnchoringStatus: "failed",
		FailureCode:     strings.TrimSpace(failureCode),
		FailureMessage:  strings.TrimSpace(failureMessage),
	}
}

func (s *Service) validatedAuditAnchorSegmentResponse(resp AuditAnchorSegmentResponse) (AuditAnchorSegmentResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, auditAnchorSegmentResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(resp.RequestID, err)
		return AuditAnchorSegmentResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) exportAnchorReceiptCopy(requestID string, result auditd.AnchorSegmentResult) string {
	receiptDigestID, ok := canExportAnchorReceiptCopy(s, requestID, result)
	if !ok {
		return ""
	}
	ref, err := s.putAnchorReceiptExportArtifact(result.ReceiptDigest, receiptDigestID)
	if err != nil {
		logAuditAnchorExportCopyFailure(requestID, result.ReceiptDigest, "artifact_put_failed", err)
		return ""
	}
	return ref.Digest
}

func canExportAnchorReceiptCopy(s *Service, requestID string, result auditd.AnchorSegmentResult) (string, bool) {
	if s == nil || s.store == nil || s.auditLedger == nil {
		logAuditAnchorExportCopyFailure(requestID, result.ReceiptDigest, "dependencies_unavailable", nil)
		return "", false
	}
	if _, err := result.SealDigest.Identity(); err != nil {
		logAuditAnchorExportCopyFailure(requestID, result.ReceiptDigest, "seal_digest_invalid", nil)
		return "", false
	}
	if strings.TrimSpace(result.AnchorStatus) != "ok" {
		logAuditAnchorExportCopyFailure(requestID, result.ReceiptDigest, "anchoring_status_not_ok", nil)
		return "", false
	}
	receiptDigestID, err := result.ReceiptDigest.Identity()
	if err != nil {
		logAuditAnchorExportCopyFailure(requestID, result.ReceiptDigest, "receipt_digest_invalid", err)
		return "", false
	}
	return receiptDigestID, true
}

func (s *Service) putAnchorReceiptExportArtifact(receiptDigest trustpolicy.Digest, receiptDigestID string) (artifacts.ArtifactReference, error) {
	receiptEnvelope, err := s.auditLedger.ReceiptEnvelopeByDigest(receiptDigest)
	if err != nil {
		return artifacts.ArtifactReference{}, err
	}
	payload, err := json.Marshal(receiptEnvelope)
	if err != nil {
		return artifacts.ArtifactReference{}, err
	}
	return s.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassAuditReceiptExportCopy,
		ProvenanceReceiptHash: receiptDigestID,
		CreatedByRole:         "auditd",
		TrustedSource:         false,
		RunID:                 "audit-anchor",
		StepID:                "anchor-receipt",
	})
}

func logAuditAnchorExportCopyFailure(requestID string, receiptDigest trustpolicy.Digest, reason string, err error) {
	receiptID, _ := receiptDigest.Identity()
	if err == nil {
		log.Printf("brokerapi: audit anchor export copy skipped request_id=%s receipt_digest=%s reason=%s", strings.TrimSpace(requestID), strings.TrimSpace(receiptID), strings.TrimSpace(reason))
		return
	}
	log.Printf("brokerapi: audit anchor export copy skipped request_id=%s receipt_digest=%s reason=%s error_type=%T error=%v", strings.TrimSpace(requestID), strings.TrimSpace(receiptID), strings.TrimSpace(reason), err, err)
}

func digestPtr(d trustpolicy.Digest) *trustpolicy.Digest {
	if _, err := d.Identity(); err != nil {
		return nil
	}
	v := d
	return &v
}
