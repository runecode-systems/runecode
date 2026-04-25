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
	case shellActivityStateWaiting:
		return warnBadge("activity=waiting")
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
	if active, state := cache.activeFocus(); strings.TrimSpace(active.Kind) != "" {
		return shellActivitySemantics{State: state, Active: active}
	}
	return shellActivitySemantics{State: shellActivityStateIdle}
}

func (c shellWatchReductionState) activeFocus() (shellActivityFocus, shellActivityState) {
	if focus, ok := c.activeFocusFromFeed(); ok {
		return focus, activityStateForFocus(c, focus)
	}
	if focus, ok := c.activeFocusFromSnapshots(); ok {
		return focus, activityStateForFocus(c, focus)
	}
	return shellActivityFocus{}, shellActivityStateIdle
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

func activityStateForFocus(c shellWatchReductionState, focus shellActivityFocus) shellActivityState {
	switch strings.TrimSpace(focus.Kind) {
	case "run":
		if run, ok := c.runs[strings.TrimSpace(focus.ID)]; ok {
			return runActivityState(run)
		}
	case "approval":
		if approval, ok := c.approvals[strings.TrimSpace(focus.ID)]; ok {
			return approvalActivityState(approval)
		}
	case "session":
		if session, ok := c.sessions[strings.TrimSpace(focus.ID)]; ok {
			return sessionActivityState(session)
		}
	}
	return shellActivityStateIdle
}

func activeRunFocus(runs map[string]brokerapi.RunSummary) (shellActivityFocus, bool) {
	bestID := ""
	for id, run := range runs {
		if runActivelyProgressing(run) {
			if bestID == "" || id < bestID {
				bestID = id
			}
		}
	}
	if bestID == "" {
		return shellActivityFocus{}, false
	}
	return shellActivityFocus{Kind: "run", ID: bestID}, true
}

func activeApprovalFocus(approvals map[string]brokerapi.ApprovalSummary) (shellActivityFocus, bool) {
	bestID := ""
	for id, approval := range approvals {
		if approvalActivelyProgressing(approval) {
			if bestID == "" || id < bestID {
				bestID = id
			}
		}
	}
	if bestID == "" {
		return shellActivityFocus{}, false
	}
	return shellActivityFocus{Kind: "approval", ID: bestID}, true
}

func activeSessionFocus(sessions map[string]brokerapi.SessionSummary) (shellActivityFocus, bool) {
	bestID := ""
	for id, session := range sessions {
		if sessionActivelyProgressing(session) {
			if bestID == "" || id < bestID {
				bestID = id
			}
		}
	}
	if bestID == "" {
		return shellActivityFocus{}, false
	}
	return shellActivityFocus{Kind: "session", ID: bestID}, true
}

func runActivelyProgressing(summary brokerapi.RunSummary) bool {
	return runActivityState(summary) != shellActivityStateIdle
}

func runActivityState(summary brokerapi.RunSummary) shellActivityState {
	state := normalizeActivityValue(summary.LifecycleState)
	if state == "" {
		return shellActivityStateIdle
	}
	if isWaitingActivityValue(state) {
		return shellActivityStateWaiting
	}
	if isRunningActivityValue(state) {
		return shellActivityStateRunning
	}
	return shellActivityStateIdle
}

func approvalActivelyProgressing(summary brokerapi.ApprovalSummary) bool {
	return approvalActivityState(summary) != shellActivityStateIdle
}

func approvalActivityState(summary brokerapi.ApprovalSummary) shellActivityState {
	status := normalizeActivityValue(summary.Status)
	if status == "" {
		return shellActivityStateIdle
	}
	if isApprovalWaitingValue(status) {
		return shellActivityStateWaiting
	}
	return shellActivityStateIdle
}

func sessionActivelyProgressing(summary brokerapi.SessionSummary) bool {
	return sessionActivityState(summary) != shellActivityStateIdle
}

func sessionActivityState(summary brokerapi.SessionSummary) shellActivityState {
	if posture := normalizeActivityValue(summary.WorkPosture); posture != "" {
		if isIdleActivityValue(posture) {
			return shellActivityStateIdle
		}
		if isWaitingActivityValue(posture) {
			return shellActivityStateWaiting
		}
		if isRunningActivityValue(posture) {
			return shellActivityStateRunning
		}
	}
	status := normalizeActivityValue(summary.Status)
	if isWaitingActivityValue(status) {
		return shellActivityStateWaiting
	}
	if isRunningActivityValue(status) {
		return shellActivityStateRunning
	}
	if summary.HasIncompleteTurn {
		return shellActivityStateWaiting
	}
	return shellActivityStateIdle
}

func normalizeActivityValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func isWaitingActivityValue(value string) bool {
	switch value {
	case "waiting", "queued", "pending", "blocked", "requested":
		return true
	}
	return strings.HasPrefix(value, "waiting_") || strings.HasPrefix(value, "queued_") || strings.HasPrefix(value, "pending_") || strings.HasPrefix(value, "blocked_")
}

func isRunningActivityValue(value string) bool {
	switch value {
	case "active", "running", "planning", "in_progress", "progressing", "starting", "resuming":
		return true
	}
	return strings.HasPrefix(value, "running_") || strings.HasPrefix(value, "active_")
}

func isIdleActivityValue(value string) bool {
	switch value {
	case "idle", "completed", "failed", "cancelled", "canceled", "degraded", "inactive", "not_running", "stopped", "finished", "approved", "consumed", "denied", "expired", "superseded":
		return true
	}
	return strings.HasPrefix(value, "completed_") || strings.HasPrefix(value, "failed_") || strings.HasPrefix(value, "idle_")
}

func isApprovalWaitingValue(value string) bool {
	switch value {
	case "pending", "requested", "waiting":
		return true
	}
	return strings.HasPrefix(value, "waiting_")
}
