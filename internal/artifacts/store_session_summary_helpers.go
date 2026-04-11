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
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = state.CreatedAt
	}
	if state.LastActivityAt.IsZero() {
		state.LastActivityAt = state.UpdatedAt
	}
	if state.Status == "" {
		state.Status = "active"
	}
	if state.LastActivityKind == "" {
		state.LastActivityKind = "unknown"
	}
	if state.TurnCount < 0 {
		state.TurnCount = 0
	}
	if len(state.TranscriptTurns) > 0 {
		state.TurnCount = len(state.TranscriptTurns)
	}
	state.LinkedRunIDs = uniqueSortedStrings(state.LinkedRunIDs)
	state.LastActivityPreview = strings.TrimSpace(state.LastActivityPreview)
	return state
}
