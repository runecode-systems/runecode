package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) HandleSessionSendMessage(ctx context.Context, req SessionSendMessageRequest, meta RequestContext) (SessionSendMessageResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, sessionSendMessageRequestSchemaPath)
	if errResp != nil {
		return SessionSendMessageResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return SessionSendMessageResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return SessionSendMessageResponse{}, &errOut
	}
	if errResp := s.validateSessionSendMessageRequest(requestID, req); errResp != nil {
		return SessionSendMessageResponse{}, errResp
	}
	if prior, handled, errResp := s.replayedSessionSendMessage(requestID, req); handled {
		return prior, errResp
	}
	session, errResp := s.sessionDetailForSendMessage(requestID, req.SessionID)
	if errResp != nil {
		return SessionSendMessageResponse{}, errResp
	}
	resp, errResp := s.newSessionSendMessageResponse(requestID, req, session)
	if errResp != nil {
		return SessionSendMessageResponse{}, errResp
	}
	s.storeSessionIdempotentInteractionResponse(req.SessionID, strings.TrimSpace(req.IdempotencyKey), resp)
	s.auditSessionSendMessage(requestID, req, resp)
	return resp, nil
}

func sessionInteractionStreamID(sessionID string) string {
	return "session-" + sessionID
}

func normalizedSessionTranscriptLinks(in *SessionTranscriptLinks) SessionTranscriptLinks {
	if in == nil {
		return SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{}, ApprovalIDs: []string{}, ArtifactDigests: []string{}, AuditRecordDigests: []string{}}
	}
	return SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: append([]string{}, in.RunIDs...), ApprovalIDs: append([]string{}, in.ApprovalIDs...), ArtifactDigests: append([]string{}, in.ArtifactDigests...), AuditRecordDigests: append([]string{}, in.AuditRecordDigests...)}
}

func (s *Service) validateSessionSendMessageRequest(requestID string, req SessionSendMessageRequest) *ErrorResponse {
	if strings.TrimSpace(req.SessionID) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "session_id is required")
		return &errOut
	}
	if strings.TrimSpace(req.ContentText) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "content_text is required")
		return &errOut
	}
	if req.Role != "user" && req.Role != "assistant" && req.Role != "system" && req.Role != "tool" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "role is invalid")
		return &errOut
	}
	return nil
}

func (s *Service) replayedSessionSendMessage(requestID string, req SessionSendMessageRequest) (SessionSendMessageResponse, bool, *ErrorResponse) {
	prior, ok := s.sessionIdempotentInteractionResponse(req.SessionID, strings.TrimSpace(req.IdempotencyKey))
	if !ok {
		return SessionSendMessageResponse{}, false, nil
	}
	prior.RequestID = requestID
	if err := s.validateResponse(prior, sessionSendMessageResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionSendMessageResponse{}, true, &errOut
	}
	return prior, true, nil
}

func (s *Service) sessionDetailForSendMessage(requestID, sessionID string) (SessionDetail, *ErrorResponse) {
	session, ok, err := s.sessionDetail(sessionID)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return SessionDetail{}, &errOut
	}
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_session", "storage", false, fmt.Sprintf("session %q not found", sessionID))
		return SessionDetail{}, &errOut
	}
	return session, nil
}

func (s *Service) newSessionSendMessageResponse(requestID string, req SessionSendMessageRequest, session SessionDetail) (SessionSendMessageResponse, *ErrorResponse) {
	createdAt := s.currentTimestampRFC3339()
	turnIndex := s.nextSessionInteractionTurnIndex(req.SessionID, session.Summary.TurnCount)
	turnID := fmt.Sprintf("%s.turn.%06d", req.SessionID, turnIndex)
	message := buildSessionSendMessageMessage(req, turnID, createdAt)
	resp := SessionSendMessageResponse{
		SchemaID:      "runecode.protocol.v0.SessionSendMessageResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		SessionID:     req.SessionID,
		Turn:          buildSessionSendMessageTurn(req.SessionID, turnID, turnIndex, createdAt, message),
		Message:       message,
		EventType:     "session_message_ack",
		StreamID:      sessionInteractionStreamID(req.SessionID),
		Seq:           s.nextSessionInteractionSeq(req.SessionID),
	}
	if err := s.validateResponse(resp, sessionSendMessageResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionSendMessageResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) currentTimestampRFC3339() string {
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	return now.Format(time.RFC3339)
}

func buildSessionSendMessageMessage(req SessionSendMessageRequest, turnID, createdAt string) SessionTranscriptMessage {
	return SessionTranscriptMessage{
		SchemaID:      "runecode.protocol.v0.SessionTranscriptMessage",
		SchemaVersion: "0.1.0",
		MessageID:     fmt.Sprintf("%s.msg.%06d", turnID, 1),
		TurnID:        turnID,
		SessionID:     req.SessionID,
		MessageIndex:  1,
		Role:          req.Role,
		CreatedAt:     createdAt,
		ContentText:   req.ContentText,
		RelatedLinks:  normalizedSessionTranscriptLinks(req.RelatedLinks),
	}
}

func buildSessionSendMessageTurn(sessionID, turnID string, turnIndex int, createdAt string, message SessionTranscriptMessage) SessionTranscriptTurn {
	return SessionTranscriptTurn{
		SchemaID:      "runecode.protocol.v0.SessionTranscriptTurn",
		SchemaVersion: "0.1.0",
		TurnID:        turnID,
		SessionID:     sessionID,
		TurnIndex:     turnIndex,
		StartedAt:     createdAt,
		CompletedAt:   createdAt,
		Status:        "completed",
		Messages:      []SessionTranscriptMessage{message},
	}
}

func (s *Service) auditSessionSendMessage(requestID string, req SessionSendMessageRequest, resp SessionSendMessageResponse) {
	_ = s.AppendTrustedAuditEvent("session_message_recorded", "brokerapi", map[string]interface{}{"session_id": req.SessionID, "turn_id": resp.Turn.TurnID, "message_id": resp.Message.MessageID, "stream_id": resp.StreamID, "seq": resp.Seq, "request_id": requestID, "role": req.Role})
}
