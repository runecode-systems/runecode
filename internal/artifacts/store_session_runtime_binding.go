package artifacts

import (
	"encoding/json"
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
	session.LinkedRunIDs = uniqueSortedStrings(append(session.LinkedRunIDs, runID))
	return normalizeSessionDurableState(session)
}

func sessionDurableStateEqual(a, b SessionDurableState) bool {
	aNorm := normalizeSessionDurableState(copySessionDurableState(a))
	bNorm := normalizeSessionDurableState(copySessionDurableState(b))
	aJSON, aErr := json.Marshal(aNorm)
	bJSON, bErr := json.Marshal(bNorm)
	if aErr != nil || bErr != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}
