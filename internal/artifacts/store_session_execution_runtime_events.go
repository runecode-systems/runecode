package artifacts

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (s *Store) SyncSessionExecutionFromRunRuntime(runID string, facts launcherbackend.RuntimeFactsSnapshot, advisory RunnerAdvisoryState, occurredAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedRunID := strings.TrimSpace(runID)
	if normalizedRunID == "" {
		return fmt.Errorf("run id is required")
	}
	normalizedAt := normalizedOccurredAt(occurredAt)
	facts = normalizeRuntimeFactsForSessionExecution(facts)

	stateChanged := false
	for sessionID, session := range s.state.Sessions {
		updated, changed := syncSessionExecutionFromRunRuntime(session, normalizedRunID, facts, advisory, normalizedAt)
		if !changed {
			continue
		}
		s.state.Sessions[sessionID] = updated
		stateChanged = true
	}
	if !stateChanged {
		return nil
	}
	return s.saveStateLocked()
}

func syncSessionExecutionFromRunRuntime(session SessionDurableState, runID string, facts launcherbackend.RuntimeFactsSnapshot, advisory RunnerAdvisoryState, occurredAt time.Time) (SessionDurableState, bool) {
	if len(session.TurnExecutions) == 0 {
		return session, false
	}
	idx := latestSessionTurnExecutionForRun(session.TurnExecutions, runID)
	if idx < 0 {
		return session, false
	}
	prior := session.TurnExecutions[idx]
	updated := projectSessionExecutionRuntimeState(prior, runID, facts, advisory, occurredAt)
	if reflect.DeepEqual(sessionTurnExecutionComparable(prior), sessionTurnExecutionComparable(updated)) {
		return session, false
	}
	session.TurnExecutions[idx] = updated
	applySessionExecutionSummaryUpdate(&session, updated, occurredAt)
	return session, true
}

func latestSessionTurnExecutionForRun(executions []SessionTurnExecutionDurableState, runID string) int {
	for idx := len(executions) - 1; idx >= 0; idx-- {
		exec := executions[idx]
		if strings.TrimSpace(exec.PrimaryRunID) == strings.TrimSpace(runID) {
			return idx
		}
		for _, linked := range exec.LinkedRunIDs {
			if strings.TrimSpace(linked) == strings.TrimSpace(runID) {
				return idx
			}
		}
	}
	return -1
}

func projectSessionExecutionRuntimeState(exec SessionTurnExecutionDurableState, runID string, facts launcherbackend.RuntimeFactsSnapshot, advisory RunnerAdvisoryState, occurredAt time.Time) SessionTurnExecutionDurableState {
	projected := copySessionTurnExecutionDurableState(exec)
	if strings.TrimSpace(projected.PrimaryRunID) == "" {
		projected.PrimaryRunID = strings.TrimSpace(runID)
	}
	projected.LinkedRunIDs = mergeSessionLinkedRunIDs(projected.LinkedRunIDs, []string{runID})

	if updated, changed := projectSessionExecutionFromLifecycle(projected, advisory); changed {
		projected = updated
	}
	if updated, changed := projectSessionExecutionFromApprovalWait(projected, advisory); changed {
		projected = updated
	}
	if updated, changed := projectSessionExecutionFromRuntimeTerminal(projected, facts); changed {
		projected = updated
	}
	if !reflect.DeepEqual(sessionTurnExecutionComparable(projected), sessionTurnExecutionComparable(exec)) {
		projected.UpdatedAt = occurredAt
	}
	return projected
}

func projectSessionExecutionFromApprovalWait(exec SessionTurnExecutionDurableState, advisory RunnerAdvisoryState) (SessionTurnExecutionDurableState, bool) {
	for _, wait := range advisory.ApprovalWaits {
		if strings.TrimSpace(wait.RunID) != strings.TrimSpace(exec.PrimaryRunID) {
			continue
		}
		approvalID := strings.TrimSpace(wait.ApprovalID)
		if approvalID == "" {
			continue
		}
		exec.PendingApprovalID = approvalID
		exec.LinkedApprovalIDs = uniqueSortedStrings(append(exec.LinkedApprovalIDs, approvalID))
		switch strings.TrimSpace(wait.Status) {
		case "pending":
			exec.ExecutionState = "waiting"
			exec.WaitKind = "approval"
			exec.WaitState = "waiting_approval"
			exec.BlockedReasonCode = ""
			exec.TerminalOutcome = ""
			return exec, true
		case "approved", "consumed", "denied", "expired", "cancelled", "superseded":
			exec.PendingApprovalID = ""
			exec.WaitKind = ""
			exec.WaitState = ""
			if exec.ExecutionState == "waiting" {
				exec.ExecutionState = "running"
			}
			exec.BlockedReasonCode = ""
			return exec, true
		}
	}
	return exec, false
}

func projectSessionExecutionFromLifecycle(exec SessionTurnExecutionDurableState, advisory RunnerAdvisoryState) (SessionTurnExecutionDurableState, bool) {
	if strings.TrimSpace(exec.ExecutionState) == "blocked" && strings.TrimSpace(exec.WaitKind) == "project_blocked" {
		return exec, false
	}
	if advisory.Lifecycle == nil {
		return exec, false
	}
	state := strings.TrimSpace(advisory.Lifecycle.LifecycleState)
	checkpointCode := lifecycleCheckpointCode(advisory)
	switch state {
	case "pending", "starting":
		return projectExecutionLifecyclePlanning(exec), true
	case "active", "recovering":
		return projectExecutionLifecycleRunning(exec), true
	case "blocked":
		return projectExecutionLifecycleBlocked(exec, checkpointCode), true
	case "completed":
		return projectExecutionLifecycleTerminal(exec, "completed", "completed"), true
	case "failed":
		return projectExecutionLifecycleTerminal(exec, "failed", "failed"), true
	case "cancelled":
		return projectExecutionLifecycleTerminal(exec, "failed", "cancelled"), true
	default:
		return exec, false
	}
}

func lifecycleCheckpointCode(advisory RunnerAdvisoryState) string {
	if advisory.LastCheckpoint == nil {
		return ""
	}
	return strings.TrimSpace(advisory.LastCheckpoint.CheckpointCode)
}

func clearExecutionLifecycleWaitState(exec SessionTurnExecutionDurableState) SessionTurnExecutionDurableState {
	exec.WaitKind = ""
	exec.WaitState = ""
	exec.TerminalOutcome = ""
	exec.BlockedReasonCode = ""
	return exec
}

func projectExecutionLifecyclePlanning(exec SessionTurnExecutionDurableState) SessionTurnExecutionDurableState {
	exec.ExecutionState = "planning"
	return clearExecutionLifecycleWaitState(exec)
}

func projectExecutionLifecycleRunning(exec SessionTurnExecutionDurableState) SessionTurnExecutionDurableState {
	exec.ExecutionState = "running"
	return clearExecutionLifecycleWaitState(exec)
}

func projectExecutionLifecycleBlocked(exec SessionTurnExecutionDurableState, checkpointCode string) SessionTurnExecutionDurableState {
	exec.ExecutionState = "waiting"
	if checkpointCode == "approval_wait_entered" {
		exec.WaitKind = "approval"
		exec.WaitState = "waiting_approval"
	} else {
		if strings.TrimSpace(exec.WaitKind) == "" {
			exec.WaitKind = "external_dependency"
		}
		if strings.TrimSpace(exec.WaitState) == "" {
			exec.WaitState = "waiting_external_dependency"
		}
	}
	exec.TerminalOutcome = ""
	return exec
}

func projectExecutionLifecycleTerminal(exec SessionTurnExecutionDurableState, executionState, terminalOutcome string) SessionTurnExecutionDurableState {
	exec.ExecutionState = executionState
	exec.WaitKind = ""
	exec.WaitState = ""
	exec.PendingApprovalID = ""
	exec.TerminalOutcome = terminalOutcome
	exec.BlockedReasonCode = ""
	return exec
}

func projectSessionExecutionFromRuntimeTerminal(exec SessionTurnExecutionDurableState, facts launcherbackend.RuntimeFactsSnapshot) (SessionTurnExecutionDurableState, bool) {
	if facts.TerminalReport == nil {
		return exec, false
	}
	terminal := facts.TerminalReport.Normalized()
	switch terminal.TerminationKind {
	case launcherbackend.BackendTerminationKindCompleted:
		exec.ExecutionState = "completed"
		exec.TerminalOutcome = "completed"
		exec.WaitKind = ""
		exec.WaitState = ""
		exec.PendingApprovalID = ""
		exec.BlockedReasonCode = ""
		return exec, true
	case launcherbackend.BackendTerminationKindFailed:
		exec.ExecutionState = "failed"
		exec.TerminalOutcome = "failed"
		exec.WaitKind = ""
		exec.WaitState = ""
		exec.PendingApprovalID = ""
		if strings.TrimSpace(terminal.FailureReasonCode) != "" {
			exec.BlockedReasonCode = strings.TrimSpace(terminal.FailureReasonCode)
		} else {
			exec.BlockedReasonCode = ""
		}
		return exec, true
	default:
		return exec, false
	}
}

func normalizeRuntimeFactsForSessionExecution(facts launcherbackend.RuntimeFactsSnapshot) launcherbackend.RuntimeFactsSnapshot {
	facts.LaunchReceipt = facts.LaunchReceipt.Normalized()
	facts.TerminalReport = normalizeRuntimeTerminalReport(facts.TerminalReport)
	return facts
}
