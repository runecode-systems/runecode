package brokerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
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
	order, errResp := s.resolveSessionListOrder(requestID, req.Order)
	if errResp != nil {
		return SessionListResponse{}, errResp
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
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "session_id is required")
		return SessionGetResponse{}, &errOut
	}
	detail, ok, err := s.sessionDetail(sessionID)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return SessionGetResponse{}, &errOut
	}
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_session", "storage", false, fmt.Sprintf("session %q not found", sessionID))
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
	states := s.store.SessionDurableStates()
	runsBySession, approvalsBySession, artifactsBySession := s.sessionLinkIndexes(states)
	auditBySession, err := s.auditRecordDigestsBySession()
	if err != nil {
		return nil, err
	}
	out := buildSessionSummaries(states, runsBySession, approvalsBySession, artifactsBySession, auditBySession)
	sortSessionSummaries(out, order)
	return out, nil
}

func (s *Service) resolveSessionListOrder(requestID, order string) (string, *ErrorResponse) {
	if order == "" {
		return "updated_at_desc", nil
	}
	if order == "updated_at_desc" || order == "updated_at_asc" {
		return order, nil
	}
	errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "order must be updated_at_desc or updated_at_asc")
	return "", &errOut
}

func (s *Service) sessionDetail(sessionID string) (SessionDetail, bool, error) {
	states := s.store.SessionDurableStates()
	runsBySession, approvalsBySession, artifactsBySession := s.sessionLinkIndexes(states)
	auditBySession, err := s.auditRecordDigestsBySession()
	if err != nil {
		return SessionDetail{}, false, err
	}
	var (
		summary SessionSummary
		state   artifacts.SessionDurableState
		ok      bool
	)
	for _, candidate := range states {
		if candidate.SessionID == sessionID {
			state = candidate
			summary = buildSessionSummary(state, len(runsBySession[state.SessionID]), len(approvalsBySession[state.SessionID]), len(artifactsBySession[state.SessionID]), len(auditBySession[state.SessionID]))
			ok = true
			break
		}
	}
	if !ok {
		return SessionDetail{}, false, nil
	}
	return buildSessionDetailFromState(summary, state.TranscriptTurns, runsBySession[sessionID], approvalsBySession[sessionID], artifactsBySession[sessionID], auditBySession[sessionID]), true, nil
}
