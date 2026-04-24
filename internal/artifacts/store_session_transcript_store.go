package artifacts

import (
	"fmt"
	"strings"
	"time"
)

type SessionMessageAppendRequest struct {
	SessionID       string
	WorkspaceID     string
	CreatedByRunID  string
	Role            string
	ContentText     string
	RelatedLinks    SessionTranscriptLinksDurableState
	IdempotencyKey  string
	IdempotencyHash string
	OccurredAt      time.Time
}

type SessionMessageAppendResult struct {
	Created         bool
	Turn            SessionTranscriptTurnDurableState
	Message         SessionTranscriptMessageDurableState
	Seq             int64
	IdempotencyHash string
}

func (s *Store) AppendSessionMessage(req SessionMessageAppendRequest) (SessionMessageAppendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized, err := normalizeSessionMessageAppendRequest(req)
	if err != nil {
		return SessionMessageAppendResult{}, err
	}
	session := loadSessionForAppend(s.state.Sessions, normalized)
	if replay, handled, err := replaySessionAppend(session, normalized); handled {
		return replay, err
	}
	appendResult := createSessionAppendResult(&session, normalized)
	s.state.Sessions[normalized.SessionID] = session
	if err := s.saveStateLocked(); err != nil {
		return SessionMessageAppendResult{}, err
	}
	return appendResult, nil
}

func loadSessionForAppend(states map[string]SessionDurableState, req SessionMessageAppendRequest) SessionDurableState {
	session := states[req.SessionID]
	if session.SessionID == "" {
		return newSessionStateFromAppendRequest(req)
	}
	return session
}

func replaySessionAppend(session SessionDurableState, req SessionMessageAppendRequest) (SessionMessageAppendResult, bool, error) {
	if req.IdempotencyKey == "" {
		return SessionMessageAppendResult{}, false, nil
	}
	if session.IdempotencyByKey == nil {
		session.IdempotencyByKey = map[string]SessionIdempotencyRecord{}
	}
	prior, ok := session.IdempotencyByKey[req.IdempotencyKey]
	if !ok {
		return SessionMessageAppendResult{}, false, nil
	}
	if prior.RequestHash != req.IdempotencyHash {
		return SessionMessageAppendResult{}, true, ErrSessionIdempotencyKeyConflict
	}
	turn, message, found := sessionReplayTurnAndMessage(session, prior)
	if !found {
		return SessionMessageAppendResult{}, true, fmt.Errorf("idempotency replay state missing transcript records")
	}
	return SessionMessageAppendResult{Created: false, Turn: turn, Message: message, Seq: prior.Seq, IdempotencyHash: prior.RequestHash}, true, nil
}

func createSessionAppendResult(session *SessionDurableState, req SessionMessageAppendRequest) SessionMessageAppendResult {
	turn, message, seq := appendSessionTranscriptMessage(session, req)
	storeSessionIdempotencyRecord(session, req, turn, message, seq)
	return SessionMessageAppendResult{
		Created:         true,
		Turn:            copySessionTurnDurableState(turn),
		Message:         copySessionMessageDurableState(message),
		Seq:             seq,
		IdempotencyHash: req.IdempotencyHash,
	}
}

func storeSessionIdempotencyRecord(session *SessionDurableState, req SessionMessageAppendRequest, turn SessionTranscriptTurnDurableState, message SessionTranscriptMessageDurableState, seq int64) {
	if req.IdempotencyKey == "" {
		return
	}
	if session.IdempotencyByKey == nil {
		session.IdempotencyByKey = map[string]SessionIdempotencyRecord{}
	}
	session.IdempotencyByKey[req.IdempotencyKey] = SessionIdempotencyRecord{
		RequestHash: req.IdempotencyHash,
		TurnID:      turn.TurnID,
		MessageID:   message.MessageID,
		Seq:         seq,
	}
}

func normalizeSessionMessageAppendRequest(req SessionMessageAppendRequest) (SessionMessageAppendRequest, error) {
	normalized := req
	normalized.SessionID = strings.TrimSpace(req.SessionID)
	if normalized.SessionID == "" {
		return SessionMessageAppendRequest{}, fmt.Errorf("session id is required")
	}
	normalized.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	if normalized.WorkspaceID == "" {
		normalized.WorkspaceID = "workspace-local"
	}
	normalized.CreatedByRunID = strings.TrimSpace(req.CreatedByRunID)
	normalized.Role = strings.TrimSpace(req.Role)
	if normalized.Role == "" {
		normalized.Role = "user"
	}
	normalized.ContentText = strings.TrimSpace(req.ContentText)
	if normalized.ContentText == "" {
		return SessionMessageAppendRequest{}, fmt.Errorf("content text is required")
	}
	normalized.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	normalized.IdempotencyHash = strings.TrimSpace(req.IdempotencyHash)
	if normalized.IdempotencyKey != "" && normalized.IdempotencyHash == "" {
		return SessionMessageAppendRequest{}, fmt.Errorf("idempotency hash is required when idempotency key is set")
	}
	if req.OccurredAt.IsZero() {
		normalized.OccurredAt = time.Now().UTC()
	} else {
		normalized.OccurredAt = req.OccurredAt.UTC()
	}
	normalized.RelatedLinks = normalizeSessionLinksDurable(req.RelatedLinks)
	return normalized, nil
}

func newSessionStateFromAppendRequest(req SessionMessageAppendRequest) SessionDurableState {
	return SessionDurableState{
		SessionID:                        req.SessionID,
		WorkspaceID:                      req.WorkspaceID,
		CreatedAt:                        req.OccurredAt,
		CreatedByRunID:                   req.CreatedByRunID,
		UpdatedAt:                        req.OccurredAt,
		Status:                           "active",
		WorkPosture:                      "running",
		LastActivityAt:                   req.OccurredAt,
		LastActivityKind:                 "chat_message",
		LastInteractionSequence:          0,
		HasIncompleteTurn:                false,
		IdempotencyByKey:                 map[string]SessionIdempotencyRecord{},
		ExecutionTriggerIdempotencyByKey: map[string]SessionExecutionTriggerIdempotencyRecord{},
		TurnExecutions:                   []SessionTurnExecutionDurableState{},
	}
}

func appendSessionTranscriptMessage(session *SessionDurableState, req SessionMessageAppendRequest) (SessionTranscriptTurnDurableState, SessionTranscriptMessageDurableState, int64) {
	turnIndex := len(session.TranscriptTurns) + 1
	turnID := fmt.Sprintf("%s.turn.%06d", session.SessionID, turnIndex)
	message := SessionTranscriptMessageDurableState{
		MessageID:    fmt.Sprintf("%s.msg.%06d", turnID, 1),
		TurnID:       turnID,
		SessionID:    session.SessionID,
		MessageIndex: 1,
		Role:         req.Role,
		CreatedAt:    req.OccurredAt,
		ContentText:  req.ContentText,
		RelatedLinks: req.RelatedLinks,
	}
	completedAt := req.OccurredAt
	turn := SessionTranscriptTurnDurableState{
		TurnID:      turnID,
		SessionID:   session.SessionID,
		TurnIndex:   turnIndex,
		StartedAt:   req.OccurredAt,
		CompletedAt: &completedAt,
		Status:      "completed",
		Messages:    []SessionTranscriptMessageDurableState{message},
	}
	session.TranscriptTurns = append(session.TranscriptTurns, turn)
	session.TurnCount = len(session.TranscriptTurns)
	session.UpdatedAt = req.OccurredAt
	session.LastActivityAt = req.OccurredAt
	session.LastActivityKind = "chat_message"
	session.LastActivityPreview = req.ContentText
	session.LastInteractionSequence++
	session.HasIncompleteTurn = false
	session.Status = "active"
	session.WorkPosture = "running"
	session.WorkPostureReason = ""
	session.LinkedRunIDs = mergeSessionLinkedRunIDs(session.LinkedRunIDs, req.RelatedLinks.RunIDs)
	seq := session.LastInteractionSequence
	return turn, message, seq
}

func sessionReplayTurnAndMessage(session SessionDurableState, record SessionIdempotencyRecord) (SessionTranscriptTurnDurableState, SessionTranscriptMessageDurableState, bool) {
	for _, turn := range session.TranscriptTurns {
		if turn.TurnID != record.TurnID {
			continue
		}
		for _, message := range turn.Messages {
			if message.MessageID == record.MessageID {
				return copySessionTurnDurableState(turn), copySessionMessageDurableState(message), true
			}
		}
	}
	return SessionTranscriptTurnDurableState{}, SessionTranscriptMessageDurableState{}, false
}

func normalizeSessionLinksDurable(in SessionTranscriptLinksDurableState) SessionTranscriptLinksDurableState {
	return SessionTranscriptLinksDurableState{
		RunIDs:             uniqueSortedStrings(in.RunIDs),
		ApprovalIDs:        uniqueSortedStrings(in.ApprovalIDs),
		ArtifactDigests:    uniqueSortedStrings(in.ArtifactDigests),
		AuditRecordDigests: uniqueSortedStrings(in.AuditRecordDigests),
	}
}

func mergeSessionLinkedRunIDs(existing, added []string) []string {
	merged := append([]string{}, existing...)
	merged = append(merged, added...)
	return uniqueSortedStrings(merged)
}
