package brokerapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	auditAnchorFailureCodeInvalid = "anchor_receipt_invalid"
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
		return s.validatedAuditAnchorSegmentResponse(anchorSegmentFailedResponse(requestID, req.SealDigest, auditAnchorFailureCodeInvalid, "audit anchor signer unavailable"))
	}
	result, err := s.anchorCurrentSegment(req)
	if err != nil {
		if errors.Is(err, auditd.ErrAnchorReceiptInvalid) {
			return s.validatedAuditAnchorSegmentResponse(anchorSegmentFailedResponse(requestID, req.SealDigest, auditAnchorFailureCodeInvalid, err.Error()))
		}
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return AuditAnchorSegmentResponse{}, &errOut
	}

	resp := AuditAnchorSegmentResponse{
		SchemaID:                 "runecode.protocol.v0.AuditAnchorSegmentResponse",
		SchemaVersion:            "0.1.0",
		RequestID:                requestID,
		SealDigest:               result.SealDigest,
		ReceiptDigest:            digestPtr(result.ReceiptDigest),
		VerificationReportDigest: digestPtr(result.VerificationDigest),
		AnchoringStatus:          nonEmptyAnchorStatus(result.AnchorStatus),
	}
	if req.ExportReceiptCopy {
		resp.ExportedReceiptRef = strings.TrimSpace(s.exportAnchorReceiptCopy(result.ReceiptDigest, result.SealDigest))
	}
	return s.validatedAuditAnchorSegmentResponse(resp)
}

func (s *Service) anchorCurrentSegment(req AuditAnchorSegmentRequest) (auditd.AnchorSegmentResult, error) {
	anchorReq, err := s.buildAnchorSegmentRequest(req)
	if err != nil {
		return auditd.AnchorSegmentResult{}, fmt.Errorf("%w: %v", auditd.ErrAnchorReceiptInvalid, err)
	}
	return s.auditLedger.AnchorCurrentSegment(anchorReq)
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

func (s *Service) exportAnchorReceiptCopy(receiptDigest trustpolicy.Digest, sealDigest trustpolicy.Digest) string {
	if s == nil || s.store == nil || s.auditLedger == nil {
		return ""
	}
	receiptEnvelope, err := s.auditLedger.ReceiptEnvelopeByDigest(receiptDigest)
	if err != nil {
		return ""
	}
	payload, err := json.Marshal(receiptEnvelope)
	if err != nil {
		return ""
	}
	sealDigestID, _ := sealDigest.Identity()
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassAuditReceiptExportCopy,
		ProvenanceReceiptHash: sealDigestID,
		CreatedByRole:         "auditd",
		TrustedSource:         true,
		RunID:                 "audit-anchor",
		StepID:                "anchor-receipt",
	})
	if err != nil {
		return ""
	}
	return ref.Digest
}

func digestPtr(d trustpolicy.Digest) *trustpolicy.Digest {
	if _, err := d.Identity(); err != nil {
		return nil
	}
	v := d
	return &v
}

func nonEmptyAnchorStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "failed"
	}
	return status
}
