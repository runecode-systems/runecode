package main

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func (m *shellWatchManager) recomputeProjection() {
	run := m.families[shellWatchFamilyRun].Summary
	approval := m.families[shellWatchFamilyApproval].Summary
	session := m.families[shellWatchFamilySession].Summary
	feed := append([]shellLiveActivityEntry(nil), m.reduction.Feed...)
	health := deriveShellSyncHealth(m.families)
	activity := deriveShellActivitySemantics(health, m.reduction)
	m.projection = shellWatchProjectionState{
		Live:     dashboardLiveActivity{runWatch: run, approvalWatch: approval, sessionWatch: session, feed: feed},
		Feed:     feed,
		Health:   health,
		Activity: activity,
	}
}

func deriveShellSyncHealth(families map[shellWatchFamily]shellWatchFamilyState) shellSyncHealth {
	health := shellSyncHealth{State: shellSyncStateLoading}
	if len(families) == 0 {
		return health
	}
	healthyCount, loadingCount, failingCount := deriveShellSyncHealthCounts(families, &health)
	return finalizeShellSyncHealth(health, healthyCount, loadingCount, failingCount)
}

func deriveShellSyncHealthCounts(families map[shellWatchFamily]shellWatchFamilyState, health *shellSyncHealth) (int, int, int) {
	healthyCount := 0
	loadingCount := 0
	failingCount := 0
	for _, family := range shellWatchFamilies {
		state, ok := families[family]
		if !ok {
			loadingCount++
			continue
		}
		if health.ErrorText == "" && strings.TrimSpace(state.LastErrorText) != "" {
			health.ErrorText = state.LastErrorText
		}
		healthyCount, loadingCount, failingCount = applyShellSyncFamilyState(health, family, state, healthyCount, loadingCount, failingCount)
	}
	return healthyCount, loadingCount, failingCount
}

func applyShellSyncFamilyState(health *shellSyncHealth, family shellWatchFamily, state shellWatchFamilyState, healthyCount, loadingCount, failingCount int) (int, int, int) {
	switch state.StreamState {
	case shellWatchStreamHealthy:
		healthyCount++
	case shellWatchStreamLoading:
		loadingCount++
	case shellWatchStreamDegraded:
		failingCount++
		health.DegradedFamilies = append(health.DegradedFamilies, string(family))
	case shellWatchStreamReconnecting:
		failingCount++
		health.ReconnectingFamilies = append(health.ReconnectingFamilies, string(family))
	case shellWatchStreamDisconnected:
		failingCount++
		health.DisconnectedFamilies = append(health.DisconnectedFamilies, string(family))
	default:
		loadingCount++
	}
	return healthyCount, loadingCount, failingCount
}

func finalizeShellSyncHealth(health shellSyncHealth, healthyCount, loadingCount, failingCount int) shellSyncHealth {
	if healthyCount == len(shellWatchFamilies) {
		health.State = shellSyncStateHealthy
		return health
	}
	if loadingCount == len(shellWatchFamilies) {
		health.State = shellSyncStateLoading
		return health
	}
	if len(health.DisconnectedFamilies) == len(shellWatchFamilies) {
		health.State = shellSyncStateDisconnected
		return health
	}
	if len(health.ReconnectingFamilies) == len(shellWatchFamilies) {
		health.State = shellSyncStateReconnecting
		return health
	}
	if failingCount > 0 {
		health.State = shellSyncStateDegraded
		return health
	}
	health.State = shellSyncStateLoading
	return health
}

func (m *shellModel) publishWatchStateToRoutes() {
	msg := shellLiveActivityUpdatedMsg{Live: m.watch.projection.Live, Feed: append([]shellLiveActivityEntry(nil), m.watch.projection.Feed...), Health: m.watch.projection.Health}
	for id, route := range m.routeModels {
		updated, _ := route.Update(msg)
		m.routeModels[id] = updated
	}
}

func renderShellSyncState(state shellSyncState) string {
	switch state {
	case shellSyncStateHealthy:
		return successBadge("sync=healthy")
	case shellSyncStateDegraded:
		return warnBadge("sync=degraded")
	case shellSyncStateReconnecting:
		return warnBadge("sync=reconnecting")
	case shellSyncStateDisconnected:
		return dangerBadge("sync=disconnected")
	default:
		return neutralBadge("sync=loading")
	}
}

func renderShellActivityState(state shellActivityState) string {
	switch state {
	case shellActivityStateLoading:
		return neutralBadge("activity=loading")
	case shellActivityStateRunning:
		return infoBadge("activity=running")
	case shellActivityStateDegradedSync:
		return warnBadge("activity=degraded_sync")
	default:
		return neutralBadge("activity=idle")
	}
}

func deriveShellActivitySemantics(health shellSyncHealth, cache shellWatchReductionState) shellActivitySemantics {
	if health.State == shellSyncStateLoading {
		return shellActivitySemantics{State: shellActivityStateLoading}
	}
	if health.State == shellSyncStateDegraded || health.State == shellSyncStateDisconnected || health.State == shellSyncStateReconnecting {
		return shellActivitySemantics{State: shellActivityStateDegradedSync}
	}
	if active := cache.activeFocus(); strings.TrimSpace(active.Kind) != "" {
		return shellActivitySemantics{State: shellActivityStateRunning, Active: active}
	}
	return shellActivitySemantics{State: shellActivityStateIdle}
}

func (c shellWatchReductionState) activeFocus() shellActivityFocus {
	if focus, ok := c.activeFocusFromFeed(); ok {
		return focus
	}
	if focus, ok := c.activeFocusFromSnapshots(); ok {
		return focus
	}
	return shellActivityFocus{}
}

func (c shellWatchReductionState) activeFocusFromFeed() (shellActivityFocus, bool) {
	for i := len(c.Feed) - 1; i >= 0; i-- {
		if focus, ok := c.focusFromFeedEntry(c.Feed[i]); ok {
			return focus, true
		}
	}
	return shellActivityFocus{}, false
}

func (c shellWatchReductionState) focusFromFeedEntry(e shellLiveActivityEntry) (shellActivityFocus, bool) {
	subject := strings.TrimSpace(e.Subject)
	switch e.Family {
	case "run_watch":
		if run, ok := c.runs[subject]; ok && runActivelyProgressing(run) {
			return shellActivityFocus{Kind: "run", ID: subject}, true
		}
	case "approval_watch":
		if approval, ok := c.approvals[subject]; ok && approvalActivelyProgressing(approval) {
			return shellActivityFocus{Kind: "approval", ID: subject}, true
		}
	case "session_watch":
		if session, ok := c.sessions[subject]; ok && sessionActivelyProgressing(session) {
			return shellActivityFocus{Kind: "session", ID: subject}, true
		}
	}
	return shellActivityFocus{}, false
}

func (c shellWatchReductionState) activeFocusFromSnapshots() (shellActivityFocus, bool) {
	if focus, ok := activeRunFocus(c.runs); ok {
		return focus, true
	}
	if focus, ok := activeApprovalFocus(c.approvals); ok {
		return focus, true
	}
	if focus, ok := activeSessionFocus(c.sessions); ok {
		return focus, true
	}
	return shellActivityFocus{}, false
}

func activeRunFocus(runs map[string]brokerapi.RunSummary) (shellActivityFocus, bool) {
	for id, run := range runs {
		if runActivelyProgressing(run) {
			return shellActivityFocus{Kind: "run", ID: id}, true
		}
	}
	return shellActivityFocus{}, false
}

func activeApprovalFocus(approvals map[string]brokerapi.ApprovalSummary) (shellActivityFocus, bool) {
	for id, approval := range approvals {
		if approvalActivelyProgressing(approval) {
			return shellActivityFocus{Kind: "approval", ID: id}, true
		}
	}
	return shellActivityFocus{}, false
}

func activeSessionFocus(sessions map[string]brokerapi.SessionSummary) (shellActivityFocus, bool) {
	for id, session := range sessions {
		if sessionActivelyProgressing(session) {
			return shellActivityFocus{Kind: "session", ID: id}, true
		}
	}
	return shellActivityFocus{}, false
}

func runActivelyProgressing(summary brokerapi.RunSummary) bool {
	state := strings.ToLower(strings.TrimSpace(summary.LifecycleState))
	if state == "" {
		return false
	}
	return strings.Contains(state, "active") || strings.Contains(state, "run") || strings.Contains(state, "progress") || strings.Contains(state, "queue") || strings.Contains(state, "wait") || strings.Contains(state, "pending")
}

func approvalActivelyProgressing(summary brokerapi.ApprovalSummary) bool {
	status := strings.ToLower(strings.TrimSpace(summary.Status))
	if status == "" {
		return false
	}
	return strings.Contains(status, "pending") || strings.Contains(status, "requested") || strings.Contains(status, "wait")
}

func sessionActivelyProgressing(summary brokerapi.SessionSummary) bool {
	status := strings.ToLower(strings.TrimSpace(summary.Status))
	if summary.HasIncompleteTurn {
		return true
	}
	return strings.Contains(status, "active") || strings.Contains(status, "run") || strings.Contains(status, "progress") || strings.Contains(status, "wait") || strings.Contains(status, "queued")
}
