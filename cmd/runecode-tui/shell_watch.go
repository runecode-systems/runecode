package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const shellWatchPollInterval = 2 * time.Second
const shellLiveActivityFeedMax = 18

type shellSyncState string

const (
	shellSyncStateLoading      shellSyncState = "loading"
	shellSyncStateHealthy      shellSyncState = "healthy"
	shellSyncStateDegraded     shellSyncState = "degraded"
	shellSyncStateDisconnected shellSyncState = "disconnected"
)

type shellSyncHealth struct {
	State     shellSyncState
	ErrorText string
}

type shellLiveActivityEntry struct {
	Family    string
	EventType string
	Subject   string
	Status    string
}

type shellLiveActivityCache struct {
	Live      dashboardLiveActivity
	Feed      []shellLiveActivityEntry
	runs      map[string]brokerapi.RunSummary
	approvals map[string]brokerapi.ApprovalSummary
	sessions  map[string]brokerapi.SessionSummary
}

type shellActivityState string

const (
	shellActivityStateIdle         shellActivityState = "idle"
	shellActivityStateLoading      shellActivityState = "loading"
	shellActivityStateRunning      shellActivityState = "running"
	shellActivityStateDegradedSync shellActivityState = "degraded sync"
)

type shellActivityFocus struct {
	Kind string
	ID   string
}

type shellActivitySemantics struct {
	State  shellActivityState
	Active shellActivityFocus
}

type shellActivityTickMsg struct{}

type shellWatchPollMsg struct{}

type shellWatchLoadedMsg struct {
	runEvents      []brokerapi.RunWatchEvent
	runErr         error
	approvalEvents []brokerapi.ApprovalWatchEvent
	approvalErr    error
	sessionEvents  []brokerapi.SessionWatchEvent
	sessionErr     error
}

type shellLiveActivityUpdatedMsg struct {
	Live   dashboardLiveActivity
	Feed   []shellLiveActivityEntry
	Health shellSyncHealth
}

func (m shellModel) startWatchPollCmd() tea.Cmd {
	return func() tea.Msg {
		return shellWatchPollMsg{}
	}
}

func (m shellModel) watchPollTickCmd() tea.Cmd {
	return tea.Tick(shellWatchPollInterval, func(time.Time) tea.Msg {
		return shellWatchPollMsg{}
	})
}

func (m shellModel) loadWatchPollCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()

		runEvents, runErr := m.client.RunWatch(ctx, brokerapi.RunWatchRequest{StreamID: newRequestID("shell-run-watch-stream"), IncludeSnapshot: true, Follow: true})
		approvalEvents, approvalErr := m.client.ApprovalWatch(ctx, brokerapi.ApprovalWatchRequest{StreamID: newRequestID("shell-approval-watch-stream"), IncludeSnapshot: true, Follow: true})
		sessionEvents, sessionErr := m.client.SessionWatch(ctx, brokerapi.SessionWatchRequest{StreamID: newRequestID("shell-session-watch-stream"), IncludeSnapshot: true, Follow: true})

		return shellWatchLoadedMsg{
			runEvents:      runEvents,
			runErr:         runErr,
			approvalEvents: approvalEvents,
			approvalErr:    approvalErr,
			sessionEvents:  sessionEvents,
			sessionErr:     sessionErr,
		}
	}
}

func (m *shellModel) applyWatchPoll(msg shellWatchLoadedMsg) {
	m.watchCache.Live.runWatch = shellWatchSummaryWithFallback(summarizeRunWatchEvents(msg.runEvents), msg.runErr, "run_watch")
	m.watchCache.Live.approvalWatch = shellWatchSummaryWithFallback(summarizeApprovalWatchEvents(msg.approvalEvents), msg.approvalErr, "approval_watch")
	m.watchCache.Live.sessionWatch = shellWatchSummaryWithFallback(summarizeSessionWatchEvents(msg.sessionEvents), msg.sessionErr, "session_watch")

	m.watchCache.appendRunEvents(msg.runEvents)
	m.watchCache.appendApprovalEvents(msg.approvalEvents)
	m.watchCache.appendSessionEvents(msg.sessionEvents)

	m.watchHealth = deriveShellSyncHealth(msg)
	m.activity = deriveShellActivitySemantics(m.watchHealth, m.watchCache)
	m.publishWatchStateToRoutes()
}

func (m shellModel) activityTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return shellActivityTickMsg{}
	})
}

func shellWatchSummaryWithFallback(summary watchFamilySummary, err error, family string) watchFamilySummary {
	summary.family = family
	if err == nil {
		if strings.TrimSpace(summary.lastStatus) == "" {
			summary.lastStatus = "ok"
		}
		return summary
	}
	summary.lastStatus = "watch_error"
	summary.lastSubject = "ipc_watch_error"
	if strings.TrimSpace(summary.lastEventType) == "" {
		summary.lastEventType = "watch_error"
	}
	return summary
}

func (c *shellLiveActivityCache) appendRunEvents(events []brokerapi.RunWatchEvent) {
	if c.runs == nil {
		c.runs = map[string]brokerapi.RunSummary{}
	}
	for _, event := range events {
		subject := ""
		if event.Run != nil {
			subject = strings.TrimSpace(event.Run.RunID)
			if subject != "" {
				c.runs[subject] = *event.Run
			}
		}
		status := strings.TrimSpace(event.TerminalStatus)
		if status == "" {
			status = "ok"
		}
		c.append(shellLiveActivityEntry{Family: "run_watch", EventType: event.EventType, Subject: subject, Status: status})
	}
}

func (c *shellLiveActivityCache) appendApprovalEvents(events []brokerapi.ApprovalWatchEvent) {
	if c.approvals == nil {
		c.approvals = map[string]brokerapi.ApprovalSummary{}
	}
	for _, event := range events {
		subject := ""
		if event.Approval != nil {
			subject = strings.TrimSpace(event.Approval.ApprovalID)
			if subject != "" {
				c.approvals[subject] = *event.Approval
			}
		}
		status := strings.TrimSpace(event.TerminalStatus)
		if status == "" {
			status = "ok"
		}
		c.append(shellLiveActivityEntry{Family: "approval_watch", EventType: event.EventType, Subject: subject, Status: status})
	}
}

func (c *shellLiveActivityCache) appendSessionEvents(events []brokerapi.SessionWatchEvent) {
	if c.sessions == nil {
		c.sessions = map[string]brokerapi.SessionSummary{}
	}
	for _, event := range events {
		subject := ""
		if event.Session != nil {
			subject = strings.TrimSpace(event.Session.Identity.SessionID)
			if subject != "" {
				c.sessions[subject] = *event.Session
			}
		}
		status := strings.TrimSpace(event.TerminalStatus)
		if status == "" {
			status = "ok"
		}
		c.append(shellLiveActivityEntry{Family: "session_watch", EventType: event.EventType, Subject: subject, Status: status})
	}
}

func (c *shellLiveActivityCache) append(entry shellLiveActivityEntry) {
	if strings.TrimSpace(entry.EventType) == "" {
		return
	}
	c.Feed = append(c.Feed, entry)
	if len(c.Feed) <= shellLiveActivityFeedMax {
		return
	}
	c.Feed = c.Feed[len(c.Feed)-shellLiveActivityFeedMax:]
}

func deriveShellSyncHealth(msg shellWatchLoadedMsg) shellSyncHealth {
	failureCount := 0
	dialLikeCount := 0
	firstErrText := ""
	for _, err := range []error{msg.runErr, msg.approvalErr, msg.sessionErr} {
		if err == nil {
			continue
		}
		failureCount++
		if firstErrText == "" {
			firstErrText = safeUIErrorText(err)
		}
		errText := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.HasPrefix(errText, "local_ipc_dial_error") || strings.HasPrefix(errText, "local_ipc_config_error") {
			dialLikeCount++
		}
	}

	switch {
	case failureCount == 0:
		return shellSyncHealth{State: shellSyncStateHealthy}
	case failureCount == 3 || dialLikeCount == 3:
		return shellSyncHealth{State: shellSyncStateDisconnected, ErrorText: firstErrText}
	default:
		return shellSyncHealth{State: shellSyncStateDegraded, ErrorText: firstErrText}
	}
}

func (m *shellModel) publishWatchStateToRoutes() {
	msg := shellLiveActivityUpdatedMsg{Live: m.watchCache.Live, Feed: append([]shellLiveActivityEntry(nil), m.watchCache.Feed...), Health: m.watchHealth}
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

func deriveShellActivitySemantics(health shellSyncHealth, cache shellLiveActivityCache) shellActivitySemantics {
	if health.State == shellSyncStateLoading {
		return shellActivitySemantics{State: shellActivityStateLoading}
	}
	if health.State == shellSyncStateDegraded || health.State == shellSyncStateDisconnected {
		return shellActivitySemantics{State: shellActivityStateDegradedSync}
	}
	if active := cache.activeFocus(); strings.TrimSpace(active.Kind) != "" {
		return shellActivitySemantics{State: shellActivityStateRunning, Active: active}
	}
	return shellActivitySemantics{State: shellActivityStateIdle}
}

func (c shellLiveActivityCache) activeFocus() shellActivityFocus {
	if focus, ok := c.activeFocusFromFeed(); ok {
		return focus
	}
	if focus, ok := c.activeFocusFromSnapshots(); ok {
		return focus
	}
	return shellActivityFocus{}
}

func (c shellLiveActivityCache) activeFocusFromFeed() (shellActivityFocus, bool) {
	for i := len(c.Feed) - 1; i >= 0; i-- {
		if focus, ok := c.focusFromFeedEntry(c.Feed[i]); ok {
			return focus, true
		}
	}
	return shellActivityFocus{}, false
}

func (c shellLiveActivityCache) focusFromFeedEntry(e shellLiveActivityEntry) (shellActivityFocus, bool) {
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

func (c shellLiveActivityCache) activeFocusFromSnapshots() (shellActivityFocus, bool) {
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
	if strings.Contains(state, "active") || strings.Contains(state, "run") || strings.Contains(state, "progress") || strings.Contains(state, "queue") || strings.Contains(state, "wait") || strings.Contains(state, "pending") {
		return true
	}
	return false
}

func approvalActivelyProgressing(summary brokerapi.ApprovalSummary) bool {
	status := strings.ToLower(strings.TrimSpace(summary.Status))
	if status == "" {
		return false
	}
	if strings.Contains(status, "pending") || strings.Contains(status, "requested") || strings.Contains(status, "wait") {
		return true
	}
	return false
}

func sessionActivelyProgressing(summary brokerapi.SessionSummary) bool {
	status := strings.ToLower(strings.TrimSpace(summary.Status))
	if summary.HasIncompleteTurn {
		return true
	}
	if strings.Contains(status, "active") || strings.Contains(status, "run") || strings.Contains(status, "progress") || strings.Contains(status, "wait") || strings.Contains(status, "queued") {
		return true
	}
	return false
}
