package artifacts

import (
	"sort"
	"strings"
)

func SessionSummaryStatesByUpdateDesc(states map[string]SessionDurableState) []SessionDurableState {
	out := make([]SessionDurableState, 0, len(states))
	for _, state := range states {
		out = append(out, copySessionDurableState(state))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].SessionID < out[j].SessionID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func normalizeSessionDurableState(state SessionDurableState) SessionDurableState {
	state.SessionID = strings.TrimSpace(state.SessionID)
	state.WorkspaceID = strings.TrimSpace(state.WorkspaceID)
	if state.WorkspaceID == "" {
		state.WorkspaceID = "workspace-local"
	}
	state.CreatedByRunID = strings.TrimSpace(state.CreatedByRunID)
	normalizeSessionTimes(&state)
	if state.Status == "" {
		state.Status = "active"
	}
	if state.WorkPosture == "" {
		state.WorkPosture = "idle"
	}
	resetSessionWorkPostureReason(&state)
	if state.LastActivityKind == "" {
		state.LastActivityKind = "unknown"
	}
	if state.TurnCount < 0 {
		state.TurnCount = 0
	}
	if len(state.TranscriptTurns) > 0 {
		state.TurnCount = len(state.TranscriptTurns)
	}
	for idx := range state.ExecutionTriggers {
		state.ExecutionTriggers[idx] = copySessionExecutionTriggerDurableState(state.ExecutionTriggers[idx])
	}
	for idx := range state.TurnExecutions {
		state.TurnExecutions[idx] = normalizeSessionTurnExecutionDurableState(state.TurnExecutions[idx])
	}
	state.LinkedRunIDs = uniqueSortedStrings(state.LinkedRunIDs)
	state.LastActivityPreview = truncateSessionActivityPreview(strings.TrimSpace(state.LastActivityPreview))
	return state
}

func truncateSessionActivityPreview(value string) string {
	const maxPreviewRunes = 256
	runes := []rune(value)
	if len(runes) <= maxPreviewRunes {
		return value
	}
	return string(runes[:maxPreviewRunes])
}

func normalizeSessionTurnExecutionDurableState(in SessionTurnExecutionDurableState) SessionTurnExecutionDurableState {
	out := copySessionTurnExecutionDurableState(in)
	out.TurnID = strings.TrimSpace(out.TurnID)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.OrchestrationScopeID = strings.TrimSpace(out.OrchestrationScopeID)
	out.DependsOnScopeIDs = uniqueSortedStrings(out.DependsOnScopeIDs)
	out.TriggerID = strings.TrimSpace(out.TriggerID)
	out.TriggerSource = strings.TrimSpace(out.TriggerSource)
	out.RequestedOperation = strings.TrimSpace(out.RequestedOperation)
	out.ExecutionState = strings.TrimSpace(out.ExecutionState)
	out.WaitKind = strings.TrimSpace(out.WaitKind)
	out.WaitState = strings.TrimSpace(out.WaitState)
	out.ApprovalProfile = strings.TrimSpace(out.ApprovalProfile)
	out.AutonomyPosture = strings.TrimSpace(out.AutonomyPosture)
	out.PrimaryRunID = strings.TrimSpace(out.PrimaryRunID)
	out.PendingApprovalID = strings.TrimSpace(out.PendingApprovalID)
	out.BoundValidatedProjectSubstrateDigest = strings.TrimSpace(out.BoundValidatedProjectSubstrateDigest)
	out.BlockedReasonCode = strings.TrimSpace(out.BlockedReasonCode)
	out.TerminalOutcome = strings.TrimSpace(out.TerminalOutcome)
	out.LinkedRunIDs = uniqueSortedStrings(out.LinkedRunIDs)
	out.LinkedApprovalIDs = uniqueSortedStrings(out.LinkedApprovalIDs)
	out.LinkedArtifactDigests = uniqueSortedStrings(out.LinkedArtifactDigests)
	out.LinkedAuditRecordDigests = uniqueSortedStrings(out.LinkedAuditRecordDigests)
	if out.ExecutionState == "" {
		out.ExecutionState = "queued"
	}
	if out.ApprovalProfile == "" {
		out.ApprovalProfile = "moderate"
	}
	if out.AutonomyPosture == "" {
		out.AutonomyPosture = "operator_guided"
	}
	if out.CreatedAt.IsZero() {
		out.CreatedAt = out.UpdatedAt
	}
	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = out.CreatedAt
	}
	return out
}

func normalizeSessionTimes(state *SessionDurableState) {
	if state == nil {
		return
	}
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = state.CreatedAt
	}
	if state.LastActivityAt.IsZero() {
		state.LastActivityAt = state.UpdatedAt
	}
}

func resetSessionWorkPostureReason(state *SessionDurableState) {
	if state == nil {
		return
	}
	if state.WorkPosture != "blocked" && state.WorkPosture != "degraded" {
		state.WorkPostureReason = ""
	}
}
