package brokerapi

import (
	"context"
	"fmt"
)

func (s *Service) HandleAuditRecordGet(ctx context.Context, req AuditRecordGetRequest, meta RequestContext) (AuditRecordGetResponse, *ErrorResponse) {
	requestID := resolveRequestID(req.RequestID, meta.RequestID)
	if s == nil {
		errOut := toErrorResponse(requestID, "gateway_failure", "internal", false, "audit record service unavailable")
		return AuditRecordGetResponse{}, &errOut
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditRecordGetRequestSchemaPath)
	if errResp != nil {
		return AuditRecordGetResponse{}, errResp
	}
	recordIdentity, errResp := s.validatedAuditRecordGetRequest(ctx, req, requestID, meta)
	if errResp != nil {
		return AuditRecordGetResponse{}, errResp
	}
	record, errResp := s.lookupProjectedAuditRecord(recordIdentity, requestID)
	if errResp != nil {
		return AuditRecordGetResponse{}, errResp
	}
	return s.validatedAuditRecordGetResponse(requestID, record)
}

func (s *Service) validatedAuditRecordGetRequest(ctx context.Context, req AuditRecordGetRequest, requestID string, meta RequestContext) (string, *ErrorResponse) {
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return "", &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return "", &errOut
	}
	recordIdentity, err := req.RecordDigest.Identity()
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, fmt.Sprintf("record_digest: %v", err))
		return "", &errOut
	}
	return recordIdentity, nil
}

func (s *Service) lookupProjectedAuditRecord(recordIdentity string, requestID string) (AuditRecordDetail, *ErrorResponse) {
	if s.auditLedger == nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit ledger unavailable")
		return AuditRecordDetail{}, &errOut
	}
	envelope, found, err := s.auditLedger.SignedEnvelopeByRecordDigest(recordIdentity)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit record lookup failed")
		return AuditRecordDetail{}, &errOut
	}
	if !found {
		errOut := s.makeError(requestID, "broker_not_found_audit_record", "storage", false, fmt.Sprintf("audit record %q not found", recordIdentity))
		return AuditRecordDetail{}, &errOut
	}
	record, err := s.projectAuditRecordDetail(recordIdentity, envelope)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit record projection failed")
		return AuditRecordDetail{}, &errOut
	}
	return record, nil
}

func (s *Service) validatedAuditRecordGetResponse(requestID string, record AuditRecordDetail) (AuditRecordGetResponse, *ErrorResponse) {
	resp := AuditRecordGetResponse{SchemaID: "runecode.protocol.v0.AuditRecordGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Record: record}
	if err := s.validateResponse(resp, auditRecordGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditRecordGetResponse{}, &errOut
	}
	return resp, nil
}
