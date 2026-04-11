package brokerapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
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
	session, errResp := s.sessionStateForSendMessage(requestID, req.SessionID)
	if errResp != nil {
		return SessionSendMessageResponse{}, errResp
	}
	resp, created, errResp := s.newSessionSendMessageResponse(requestID, req, session)
	if errResp != nil {
		return SessionSendMessageResponse{}, errResp
	}
	if created {
		s.auditSessionSendMessage(requestID, req, resp)
	}
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

func (s *Service) sessionStateForSendMessage(requestID, sessionID string) (artifacts.SessionDurableState, *ErrorResponse) {
	session, ok := s.SessionState(sessionID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_session", "storage", false, fmt.Sprintf("session %q not found", sessionID))
		return artifacts.SessionDurableState{}, &errOut
	}
	return session, nil
}

func (s *Service) newSessionSendMessageResponse(requestID string, req SessionSendMessageRequest, session artifacts.SessionDurableState) (SessionSendMessageResponse, bool, *ErrorResponse) {
	createdAt := s.currentTimestamp()
	links := normalizedSessionTranscriptLinks(req.RelatedLinks)
	durableLinks := durableSessionTranscriptLinks(links)
	idempotencyHash, err := artifacts.SessionSendMessageIdempotencyHash(req.SessionID, req.Role, req.ContentText, durableLinks)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return SessionSendMessageResponse{}, false, &errOut
	}
	appendResult, err := s.AppendSessionMessage(buildSessionMessageAppendRequest(req, session, durableLinks, idempotencyHash, createdAt))
	if err != nil {
		if errors.Is(err, artifacts.ErrSessionIdempotencyKeyConflict) {
			errOut := s.makeError(requestID, "broker_idempotency_key_payload_mismatch", "validation", false, err.Error())
			return SessionSendMessageResponse{}, false, &errOut
		}
		errOut := s.errorFromStore(requestID, err)
		return SessionSendMessageResponse{}, false, &errOut
	}
	turn := sessionTranscriptTurnFromDurable(appendResult.Turn)
	message := sessionTranscriptMessageFromDurable(appendResult.Message)
	resp := buildSessionSendMessageAckResponse(requestID, req.SessionID, turn, message, appendResult.Seq)
	if err := s.validateResponse(resp, sessionSendMessageResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionSendMessageResponse{}, false, &errOut
	}
	return resp, appendResult.Created, nil
}

func durableSessionTranscriptLinks(links SessionTranscriptLinks) artifacts.SessionTranscriptLinksDurableState {
	return artifacts.SessionTranscriptLinksDurableState{
		RunIDs:             append([]string{}, links.RunIDs...),
		ApprovalIDs:        append([]string{}, links.ApprovalIDs...),
		ArtifactDigests:    append([]string{}, links.ArtifactDigests...),
		AuditRecordDigests: append([]string{}, links.AuditRecordDigests...),
	}
}

func buildSessionMessageAppendRequest(req SessionSendMessageRequest, session artifacts.SessionDurableState, links artifacts.SessionTranscriptLinksDurableState, idempotencyHash string, occurredAt time.Time) artifacts.SessionMessageAppendRequest {
	return artifacts.SessionMessageAppendRequest{
		SessionID:       req.SessionID,
		WorkspaceID:     session.WorkspaceID,
		CreatedByRunID:  session.CreatedByRunID,
		Role:            req.Role,
		ContentText:     req.ContentText,
		RelatedLinks:    links,
		IdempotencyKey:  strings.TrimSpace(req.IdempotencyKey),
		IdempotencyHash: idempotencyHash,
		OccurredAt:      occurredAt,
	}
}

func buildSessionSendMessageAckResponse(requestID, sessionID string, turn SessionTranscriptTurn, message SessionTranscriptMessage, seq int64) SessionSendMessageResponse {
	return SessionSendMessageResponse{
		SchemaID:      "runecode.protocol.v0.SessionSendMessageResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		SessionID:     sessionID,
		Turn:          turn,
		Message:       message,
		EventType:     "session_message_ack",
		StreamID:      sessionInteractionStreamID(sessionID),
		Seq:           seq,
	}
}

func (s *Service) currentTimestamp() time.Time {
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	return now
}

func sessionTranscriptMessageFromDurable(in artifacts.SessionTranscriptMessageDurableState) SessionTranscriptMessage {
	return SessionTranscriptMessage{
		SchemaID:      "runecode.protocol.v0.SessionTranscriptMessage",
		SchemaVersion: "0.1.0",
		MessageID:     in.MessageID,
		TurnID:        in.TurnID,
		SessionID:     in.SessionID,
		MessageIndex:  in.MessageIndex,
		Role:          in.Role,
		CreatedAt:     in.CreatedAt.UTC().Format(time.RFC3339),
		ContentText:   in.ContentText,
		RelatedLinks: SessionTranscriptLinks{
			SchemaID:           "runecode.protocol.v0.SessionTranscriptLinks",
			SchemaVersion:      "0.1.0",
			RunIDs:             append([]string{}, in.RelatedLinks.RunIDs...),
			ApprovalIDs:        append([]string{}, in.RelatedLinks.ApprovalIDs...),
			ArtifactDigests:    append([]string{}, in.RelatedLinks.ArtifactDigests...),
			AuditRecordDigests: append([]string{}, in.RelatedLinks.AuditRecordDigests...),
		},
	}
}

func sessionTranscriptTurnFromDurable(in artifacts.SessionTranscriptTurnDurableState) SessionTranscriptTurn {
	turn := SessionTranscriptTurn{
		SchemaID:      "runecode.protocol.v0.SessionTranscriptTurn",
		SchemaVersion: "0.1.0",
		TurnID:        in.TurnID,
		SessionID:     in.SessionID,
		TurnIndex:     in.TurnIndex,
		StartedAt:     in.StartedAt.UTC().Format(time.RFC3339),
		Status:        in.Status,
		Messages:      make([]SessionTranscriptMessage, 0, len(in.Messages)),
	}
	if in.CompletedAt != nil {
		turn.CompletedAt = in.CompletedAt.UTC().Format(time.RFC3339)
	}
	for _, message := range in.Messages {
		turn.Messages = append(turn.Messages, sessionTranscriptMessageFromDurable(message))
	}
	return turn
}

func (s *Service) auditSessionSendMessage(requestID string, req SessionSendMessageRequest, resp SessionSendMessageResponse) {
	_ = s.AppendTrustedAuditEvent("session_message_recorded", "brokerapi", map[string]interface{}{"session_id": req.SessionID, "turn_id": resp.Turn.TurnID, "message_id": resp.Message.MessageID, "stream_id": resp.StreamID, "seq": resp.Seq, "request_id": requestID, "role": req.Role})
}
