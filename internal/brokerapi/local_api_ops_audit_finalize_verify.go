package brokerapi

import (
	"context"
	"errors"

	"github.com/runecode-ai/runecode/internal/auditd"
)

const (
	auditFinalizeVerifyFailureCodeUnavailable = "audit_finalize_verify_unavailable"
	auditFinalizeVerifyFailureCodeInvalid     = "audit_finalize_verify_invalid"
)

func (s *Service) HandleAuditFinalizeVerify(ctx context.Context, req AuditFinalizeVerifyRequest, meta RequestContext) (AuditFinalizeVerifyResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditFinalizeVerifyRequestSchemaPath)
	if errResp != nil {
		return AuditFinalizeVerifyResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return AuditFinalizeVerifyResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	return s.handleAuditFinalizeVerifyValidated(requestCtx, requestID)
}

func (s *Service) handleAuditFinalizeVerifyValidated(requestCtx context.Context, requestID string) (AuditFinalizeVerifyResponse, *ErrorResponse) {
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return AuditFinalizeVerifyResponse{}, &errOut
	}
	if s == nil || s.auditLedger == nil {
		resp := auditFinalizeVerifyFailedResponse(requestID, auditFinalizeVerifyFailureCodeUnavailable, "audit ledger unavailable")
		return s.validatedAuditFinalizeVerifyResponse(resp)
	}
	result, err := s.auditLedger.VerifyCurrentSegmentAndPersist()
	if err == nil {
		resp := AuditFinalizeVerifyResponse{
			SchemaID:      "runecode.protocol.v0.AuditFinalizeVerifyResponse",
			SchemaVersion: "0.1.0",
			RequestID:     requestID,
			ActionStatus:  "ok",
			SegmentID:     result.SegmentID,
			ReportDigest:  cloneDigestPointer(result.ReportDigest),
		}
		return s.validatedAuditFinalizeVerifyResponse(resp)
	}
	if errors.Is(err, auditd.ErrAnchorReceiptInvalid) {
		resp := auditFinalizeVerifyFailedResponse(requestID, auditFinalizeVerifyFailureCodeInvalid, "audit anchor receipt invalid")
		return s.validatedAuditFinalizeVerifyResponse(resp)
	}
	resp := auditFinalizeVerifyFailedResponse(requestID, auditFinalizeVerifyFailureCodeUnavailable, err.Error())
	return s.validatedAuditFinalizeVerifyResponse(resp)
}

func auditFinalizeVerifyFailedResponse(requestID, failureCode, failureMessage string) AuditFinalizeVerifyResponse {
	return AuditFinalizeVerifyResponse{
		SchemaID:       "runecode.protocol.v0.AuditFinalizeVerifyResponse",
		SchemaVersion:  "0.1.0",
		RequestID:      requestID,
		ActionStatus:   "failed",
		FailureCode:    failureCode,
		FailureMessage: failureMessage,
	}
}

func (s *Service) validatedAuditFinalizeVerifyResponse(resp AuditFinalizeVerifyResponse) (AuditFinalizeVerifyResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, auditFinalizeVerifyResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(resp.RequestID, err)
		return AuditFinalizeVerifyResponse{}, &errOut
	}
	return resp, nil
}
