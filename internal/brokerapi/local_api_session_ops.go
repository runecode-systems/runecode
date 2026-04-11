package brokerapi

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) HandleSessionList(ctx context.Context, req SessionListRequest, meta RequestContext) (SessionListResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, sessionListRequestSchemaPath)
	if errResp != nil {
		return SessionListResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return SessionListResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return SessionListResponse{}, &errOut
	}
	order := req.Order
	if order == "" {
		order = "updated_at_desc"
	}
	summaries, err := s.sessionSummaries(order)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return SessionListResponse{}, &errOut
	}
	limit := normalizeLimit(req.Limit, 50, 200)
	page, next, err := paginate(summaries, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return SessionListResponse{}, &errOut
	}
	resp := SessionListResponse{SchemaID: "runecode.protocol.v0.SessionListResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Sessions: page, NextCursor: next}
	if err := s.validateResponse(resp, sessionListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionListResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleSessionGet(ctx context.Context, req SessionGetRequest, meta RequestContext) (SessionGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, sessionGetRequestSchemaPath)
	if errResp != nil {
		return SessionGetResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return SessionGetResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return SessionGetResponse{}, &errOut
	}
	if strings.TrimSpace(req.SessionID) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "session_id is required")
		return SessionGetResponse{}, &errOut
	}
	detail, ok, err := s.sessionDetail(req.SessionID)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return SessionGetResponse{}, &errOut
	}
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_session", "storage", false, fmt.Sprintf("session %q not found", req.SessionID))
		return SessionGetResponse{}, &errOut
	}
	resp := SessionGetResponse{SchemaID: "runecode.protocol.v0.SessionGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Session: detail}
	if err := s.validateResponse(resp, sessionGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) sessionSummaries(order string) ([]SessionSummary, error) {
	auditBySession, err := s.auditRecordDigestsBySession()
	if err != nil {
		return nil, err
	}
	approvalsBySession := s.approvalIDsBySession()
	byID := s.recordsBySession()
	out := buildSessionSummaries(byID, approvalsBySession, auditBySession)
	sortSessionSummaries(out, order)
	return out, nil
}

func (s *Service) sessionDetail(sessionID string) (SessionDetail, bool, error) {
	summary, ok, err := s.sessionSummaryByID(sessionID)
	if err != nil || !ok {
		return SessionDetail{}, ok, err
	}
	auditBySession, err := s.auditRecordDigestsBySession()
	if err != nil {
		return SessionDetail{}, false, err
	}
	runs, approvals, artifactsByDigest := s.sessionLinkedObjects(sessionID)
	return buildSessionDetail(summary, runs, approvals, artifactsByDigest, auditBySession[sessionID]), true, nil
}

func (s *Service) sessionSummaryByID(sessionID string) (SessionSummary, bool, error) {
	summaries, err := s.sessionSummaries("updated_at_desc")
	if err != nil {
		return SessionSummary{}, false, err
	}
	for _, summary := range summaries {
		if summary.Identity.SessionID == sessionID {
			return summary, true, nil
		}
	}
	return SessionSummary{}, false, nil
}
