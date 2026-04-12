package artifacts

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (s *Store) upsertSessionRuntimeBindingLocked(runID string, facts launcherbackend.RuntimeFactsSnapshot) bool {
	receipt := facts.LaunchReceipt.Normalized()
	sessionID := strings.TrimSpace(receipt.SessionID)
	if sessionID == "" {
		return false
	}
	normalized := normalizeSessionRuntimeBinding(s.state.Sessions[sessionID], sessionID, strings.TrimSpace(runID), s.nowFn().UTC())
	if existing, ok := s.state.Sessions[sessionID]; ok && sessionDurableStateEqual(existing, normalized) {
		return false
	}
	s.state.Sessions[sessionID] = normalized
	return true
}

func normalizeSessionRuntimeBinding(session SessionDurableState, sessionID, runID string, now time.Time) SessionDurableState {
	runID = strings.TrimSpace(runID)
	if session.SessionID == "" {
		session = SessionDurableState{
			SessionID:         sessionID,
			WorkspaceID:       "workspace-local",
			CreatedAt:         now,
			CreatedByRunID:    runID,
			UpdatedAt:         now,
			Status:            "active",
			LastActivityAt:    now,
			LastActivityKind:  "session_created",
			HasIncompleteTurn: false,
		}
	} else {
		session.UpdatedAt = now
		session.LastActivityAt = now
		session.LastActivityKind = "run_progress"
		if strings.TrimSpace(session.CreatedByRunID) == "" {
			session.CreatedByRunID = runID
		}
	}
	session.Status = "active"
	if runID != "" {
		session.LinkedRunIDs = uniqueSortedStrings(append(session.LinkedRunIDs, runID))
	}
	return normalizeSessionDurableState(session)
}

func sessionDurableStateEqual(a, b SessionDurableState) bool {
	aNorm := normalizeSessionDurableState(copySessionDurableState(a))
	bNorm := normalizeSessionDurableState(copySessionDurableState(b))
	return reflect.DeepEqual(sessionDurableStateComparable(aNorm), sessionDurableStateComparable(bNorm))
}

func sessionDurableStateComparable(state SessionDurableState) sessionDurableStateCompare {
	out := sessionDurableStateCompare{
		SessionID:            state.SessionID,
		WorkspaceID:          state.WorkspaceID,
		CreatedAtUnixNano:    state.CreatedAt.UnixNano(),
		CreatedByRunID:       state.CreatedByRunID,
		UpdatedAtUnixNano:    state.UpdatedAt.UnixNano(),
		Status:               state.Status,
		LastActivityUnixNano: state.LastActivityAt.UnixNano(),
		LastActivityKind:     state.LastActivityKind,
		LastActivityPreview:  state.LastActivityPreview,
		TurnCount:            state.TurnCount,
		HasIncompleteTurn:    state.HasIncompleteTurn,
		LinkedRunIDs:         append([]string{}, state.LinkedRunIDs...),
		TranscriptTurns:      make([]sessionTranscriptTurnDurableStateCompare, 0, len(state.TranscriptTurns)),
		IdempotencyByKey:     make(map[string]sessionIdempotencyRecordCompare, len(state.IdempotencyByKey)),
	}
	slices.Sort(out.LinkedRunIDs)
	for _, turn := range state.TranscriptTurns {
		out.TranscriptTurns = append(out.TranscriptTurns, sessionTranscriptTurnComparable(turn))
	}
	for key := range maps.Keys(state.IdempotencyByKey) {
		record := state.IdempotencyByKey[key]
		out.IdempotencyByKey[key] = sessionIdempotencyRecordCompare{
			RequestHash: record.RequestHash,
			TurnID:      record.TurnID,
			MessageID:   record.MessageID,
			Seq:         record.Seq,
		}
	}
	return out
}

func sessionTranscriptTurnComparable(turn SessionTranscriptTurnDurableState) sessionTranscriptTurnDurableStateCompare {
	completedAt := int64(0)
	if turn.CompletedAt != nil {
		completedAt = turn.CompletedAt.UnixNano()
	}
	out := sessionTranscriptTurnDurableStateCompare{
		TurnID:              turn.TurnID,
		SessionID:           turn.SessionID,
		TurnIndex:           turn.TurnIndex,
		StartedAtUnixNano:   turn.StartedAt.UnixNano(),
		CompletedAtUnixNano: completedAt,
		Status:              turn.Status,
		Messages:            make([]sessionTranscriptMessageDurableStateCompare, 0, len(turn.Messages)),
	}
	for _, message := range turn.Messages {
		out.Messages = append(out.Messages, sessionTranscriptMessageComparable(message))
	}
	return out
}

func sessionTranscriptMessageComparable(message SessionTranscriptMessageDurableState) sessionTranscriptMessageDurableStateCompare {
	out := sessionTranscriptMessageDurableStateCompare{
		MessageID:         message.MessageID,
		TurnID:            message.TurnID,
		SessionID:         message.SessionID,
		MessageIndex:      message.MessageIndex,
		Role:              message.Role,
		CreatedAtUnixNano: message.CreatedAt.UnixNano(),
		ContentText:       message.ContentText,
		RelatedLinks: sessionTranscriptLinksDurableStateCompare{
			RunIDs:             append([]string{}, message.RelatedLinks.RunIDs...),
			ApprovalIDs:        append([]string{}, message.RelatedLinks.ApprovalIDs...),
			ArtifactDigests:    append([]string{}, message.RelatedLinks.ArtifactDigests...),
			AuditRecordDigests: append([]string{}, message.RelatedLinks.AuditRecordDigests...),
		},
	}
	slices.Sort(out.RelatedLinks.RunIDs)
	slices.Sort(out.RelatedLinks.ApprovalIDs)
	slices.Sort(out.RelatedLinks.ArtifactDigests)
	slices.Sort(out.RelatedLinks.AuditRecordDigests)
	return out
}

type sessionDurableStateCompare struct {
	SessionID            string
	WorkspaceID          string
	CreatedAtUnixNano    int64
	CreatedByRunID       string
	UpdatedAtUnixNano    int64
	Status               string
	LastActivityUnixNano int64
	LastActivityKind     string
	LastActivityPreview  string
	TurnCount            int
	HasIncompleteTurn    bool
	TranscriptTurns      []sessionTranscriptTurnDurableStateCompare
	IdempotencyByKey     map[string]sessionIdempotencyRecordCompare
	LinkedRunIDs         []string
}

type sessionTranscriptTurnDurableStateCompare struct {
	TurnID              string
	SessionID           string
	TurnIndex           int
	StartedAtUnixNano   int64
	CompletedAtUnixNano int64
	Status              string
	Messages            []sessionTranscriptMessageDurableStateCompare
}

type sessionTranscriptMessageDurableStateCompare struct {
	MessageID         string
	TurnID            string
	SessionID         string
	MessageIndex      int
	Role              string
	CreatedAtUnixNano int64
	ContentText       string
	RelatedLinks      sessionTranscriptLinksDurableStateCompare
}

type sessionTranscriptLinksDurableStateCompare struct {
	RunIDs             []string
	ApprovalIDs        []string
	ArtifactDigests    []string
	AuditRecordDigests []string
}

type sessionIdempotencyRecordCompare struct {
	RequestHash string
	TurnID      string
	MessageID   string
	Seq         int64
}
