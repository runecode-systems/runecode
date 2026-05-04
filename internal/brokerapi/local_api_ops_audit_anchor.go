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

	auditAnchorErrorCodeLedgerUnavailable            = "broker_dependency_audit_ledger_unavailable"
	auditAnchorErrorCodeSignerUnavailable            = "broker_dependency_audit_anchor_signer_unavailable"
	auditAnchorErrorCodePresenceModeUnavailable      = "broker_dependency_audit_anchor_presence_mode_unavailable"
	auditAnchorErrorCodePresenceChallengeUnavailable = "broker_dependency_audit_anchor_presence_challenge_unavailable"
	auditAnchorErrorCodePresenceTokenUnavailable     = "broker_dependency_audit_anchor_presence_token_unavailable"
	auditAnchorErrorCodeAnchorActionUnavailable      = "broker_dependency_audit_anchor_action_unavailable"
	auditAnchorErrorCodeResultInvalid                = "broker_dependency_audit_anchor_result_invalid"
	auditAnchorErrorCodeReadinessUnavailable         = "broker_dependency_audit_readiness_unavailable"
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
		errOut := s.makeError(requestID, auditAnchorErrorCodeLedgerUnavailable, "internal", false, "audit ledger unavailable")
		return AuditAnchorSegmentResponse{}, &errOut
	}
	if s.secretsSvc == nil {
		return s.validatedAuditAnchorSegmentResponse(anchorSegmentFailedResponse(requestID, req.SealDigest, strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest), auditAnchorFailureCodeSignerUnavailable, "audit anchor signer unavailable"))
	}
	anchorReq, err := s.buildAnchorSegmentRequest(req)
	if err != nil {
		return s.validatedAuditAnchorSegmentResponse(anchorSegmentFailedResponse(requestID, req.SealDigest, strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest), auditAnchorFailureCodeRequestInvalid, auditAnchorFailureMessageRequestInvalid))
	}
	result, err := s.auditLedger.AnchorCurrentSegment(anchorReq)
	if err != nil {
		if errors.Is(err, auditd.ErrAnchorReceiptInvalid) {
			s.recordAnchorReceiptFailureAuthoritativePosture(req.SealDigest)
			return s.validatedAuditAnchorSegmentResponse(anchorSegmentFailedResponse(requestID, req.SealDigest, strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest), auditAnchorFailureCodeReceiptInvalid, auditAnchorFailureMessageReceiptInvalid))
		}
		errOut := s.makeError(requestID, auditAnchorErrorCodeAnchorActionUnavailable, "internal", false, auditAnchorGatewayFailureMessage)
		return AuditAnchorSegmentResponse{}, &errOut
	}

	projectContextID := strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
	resp, errResp := buildAnchorSegmentResponse(requestID, result, projectContextID)
	if errResp != nil {
		return AuditAnchorSegmentResponse{}, errResp
	}

	if req.ExportReceiptCopy {
		resp.ExportedReceiptRef = strings.TrimSpace(s.exportAnchorReceiptCopy(requestID, result))
	}
	return s.validatedAuditAnchorSegmentResponse(resp)
}

func buildAnchorSegmentResponse(requestID string, result auditd.AnchorSegmentResult, projectContextID string) (AuditAnchorSegmentResponse, *ErrorResponse) {
	anchorStatus := strings.TrimSpace(result.AnchorStatus)
	if anchorStatus == "" {
		errOut := toErrorResponse(requestID, auditAnchorErrorCodeResultInvalid, "internal", false, auditAnchorGatewayFailureMessage)
		return AuditAnchorSegmentResponse{}, &errOut
	}
	resp := AuditAnchorSegmentResponse{
		SchemaID:                 "runecode.protocol.v0.AuditAnchorSegmentResponse",
		SchemaVersion:            "0.1.0",
		RequestID:                requestID,
		ProjectContextID:         strings.TrimSpace(projectContextID),
		SealDigest:               result.SealDigest,
		ReceiptDigest:            digestPtr(result.ReceiptDigest),
		VerificationReportDigest: digestPtr(result.VerificationDigest),
		AnchoringStatus:          anchorStatus,
		FailureCode:              strings.TrimSpace(result.FailureReasonCode),
		FailureMessage:           strings.TrimSpace(result.FailureReasonMessage),
	}
	return resp, nil
}

func anchorSegmentFailedResponse(requestID string, sealDigest trustpolicy.Digest, projectContextID, failureCode, failureMessage string) AuditAnchorSegmentResponse {
	return AuditAnchorSegmentResponse{
		SchemaID:         "runecode.protocol.v0.AuditAnchorSegmentResponse",
		SchemaVersion:    "0.1.0",
		RequestID:        requestID,
		ProjectContextID: strings.TrimSpace(projectContextID),
		SealDigest:       sealDigest,
		AnchoringStatus:  "failed",
		FailureCode:      strings.TrimSpace(failureCode),
		FailureMessage:   strings.TrimSpace(failureMessage),
	}
}

func (s *Service) validatedAuditAnchorSegmentResponse(resp AuditAnchorSegmentResponse) (AuditAnchorSegmentResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, auditAnchorSegmentResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(resp.RequestID, err)
		return AuditAnchorSegmentResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) recordAnchorReceiptFailureAuthoritativePosture(sealDigest trustpolicy.Digest) {
	if s == nil || s.auditLedger == nil {
		return
	}
	report, err := s.auditLedger.LatestVerificationReport()
	if err != nil {
		log.Printf("brokerapi: audit anchor failure posture persistence skipped reason=latest_report_unavailable error_type=%T error=%v", err, err)
		return
	}
	report.AnchoringStatus = trustpolicy.AuditVerificationStatusFailed
	if !containsReasonCode(report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid) {
		report.HardFailures = append(report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid)
	}
	report.Findings = append(report.Findings, trustpolicy.AuditVerificationFinding{
		Code:                 trustpolicy.AuditVerificationReasonAnchorReceiptInvalid,
		Dimension:            trustpolicy.AuditVerificationDimensionAnchoring,
		Severity:             trustpolicy.AuditVerificationSeverityError,
		Message:              "audit_anchor_segment failed due to invalid anchor receipt evidence",
		SegmentID:            strings.TrimSpace(report.VerificationScope.LastSegmentID),
		SubjectRecordDigest:  cloneDigestPointer(sealDigest),
		RelatedRecordDigests: s.latestVerificationViewDigests(500),
	})
	report.Summary = "Audit verification failed with authoritative anchoring failure recorded."
	if _, err := s.auditLedger.PersistVerificationReport(report); err != nil {
		log.Printf("brokerapi: audit anchor failure posture persistence skipped reason=persist_failed error_type=%T error=%v", err, err)
	}
}

func containsReasonCode(codes []string, code string) bool {
	for idx := range codes {
		if codes[idx] == code {
			return true
		}
	}
	return false
}

func cloneDigestPointer(d trustpolicy.Digest) *trustpolicy.Digest {
	if _, err := d.Identity(); err != nil {
		return nil
	}
	v := d
	return &v
}

func (s *Service) latestVerificationViewDigests(limit int) []trustpolicy.Digest {
	if s == nil || s.auditLedger == nil {
		return nil
	}
	_, views, _, err := s.auditLedger.LatestVerificationSummaryAndViews(limit)
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	digests := make([]trustpolicy.Digest, 0, len(views))
	for _, view := range views {
		id, identityErr := view.RecordDigest.Identity()
		if identityErr != nil || id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		digests = append(digests, view.RecordDigest)
	}
	return digests
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
