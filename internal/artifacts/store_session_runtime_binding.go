package artifacts

import (
	"reflect"
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
			WorkPosture:       "running",
			LastActivityAt:    now,
			LastActivityKind:  "session_created",
			HasIncompleteTurn: false,
		}
	} else {
		session.UpdatedAt = now
		session.LastActivityAt = now
		session.LastActivityKind = "run_progress"
		if !hasExecutionDerivedSessionPosture(session) {
			session.WorkPosture = "running"
			session.WorkPostureReason = ""
		}
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

func hasExecutionDerivedSessionPosture(session SessionDurableState) bool {
	if len(session.TurnExecutions) == 0 {
		return false
	}
	latest := session.TurnExecutions[len(session.TurnExecutions)-1]
	switch strings.TrimSpace(latest.ExecutionState) {
	case "blocked", "failed", "waiting", "completed":
		return true
	default:
		return false
	}
}

func sessionDurableStateEqual(a, b SessionDurableState) bool {
	aNorm := normalizeSessionDurableState(copySessionDurableState(a))
	bNorm := normalizeSessionDurableState(copySessionDurableState(b))
	return reflect.DeepEqual(sessionDurableStateComparable(aNorm), sessionDurableStateComparable(bNorm))
}
