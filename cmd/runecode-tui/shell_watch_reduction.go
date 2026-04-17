package main

import (
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const shellLiveActivityFeedMax = 18

func newShellWatchManager() shellWatchManager {
	families := map[shellWatchFamily]shellWatchFamilyState{}
	for _, family := range shellWatchFamilies {
		summary := watchFamilySummary{family: string(family), lastStatus: string(shellWatchStreamLoading)}
		families[family] = shellWatchFamilyState{Family: family, StreamState: shellWatchStreamLoading, Summary: summary}
	}
	manager := shellWatchManager{now: time.Now, families: families}
	manager.recomputeProjection()
	return manager
}

func (m *shellWatchManager) applyTransport(msg shellWatchTransportLoadedMsg) {
	now := msg.ObservedAt
	if now.IsZero() {
		now = m.now().UTC()
	}
	m.applyRunTransport(msg.Run, now)
	m.applyApprovalTransport(msg.Approval, now)
	m.applySessionTransport(msg.Session, now)
	m.recomputeProjection()
}

func (m *shellWatchManager) applyRunTransport(result shellWatchRunTransportResult, now time.Time) {
	m.reduction.appendRunEvents(result.Events)
	state := m.families[shellWatchFamilyRun]
	state.Summary = shellWatchSummaryWithFallback(summarizeRunWatchEvents(result.Events), result.Err, string(shellWatchFamilyRun))
	m.applyFamilyPosture(&state, result.Err, now)
	m.families[shellWatchFamilyRun] = state
}

func (m *shellWatchManager) applyApprovalTransport(result shellWatchApprovalTransportResult, now time.Time) {
	m.reduction.appendApprovalEvents(result.Events)
	state := m.families[shellWatchFamilyApproval]
	state.Summary = shellWatchSummaryWithFallback(summarizeApprovalWatchEvents(result.Events), result.Err, string(shellWatchFamilyApproval))
	m.applyFamilyPosture(&state, result.Err, now)
	m.families[shellWatchFamilyApproval] = state
}

func (m *shellWatchManager) applySessionTransport(result shellWatchSessionTransportResult, now time.Time) {
	m.reduction.appendSessionEvents(result.Events)
	state := m.families[shellWatchFamilySession]
	state.Summary = shellWatchSummaryWithFallback(summarizeSessionWatchEvents(result.Events), result.Err, string(shellWatchFamilySession))
	m.applyFamilyPosture(&state, result.Err, now)
	m.families[shellWatchFamilySession] = state
}

func (m *shellWatchManager) applyFamilyPosture(state *shellWatchFamilyState, err error, now time.Time) {
	if state == nil {
		return
	}
	if err == nil {
		state.StreamState = shellWatchStreamHealthy
		state.LastErrorText = ""
		state.ConsecutiveFailures = 0
		state.LastSuccessAt = now
		state.NextRetryAt = time.Time{}
		return
	}
	state.ConsecutiveFailures++
	state.LastFailureAt = now
	state.LastErrorText = safeUIErrorText(err)
	state.NextRetryAt = now.Add(shellWatchBackoffDelay(state.ConsecutiveFailures))
	state.StreamState = deriveFamilyStreamState(err, state.ConsecutiveFailures)
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

func (c *shellWatchReductionState) appendRunEvents(events []brokerapi.RunWatchEvent) {
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

func (c *shellWatchReductionState) appendApprovalEvents(events []brokerapi.ApprovalWatchEvent) {
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

func (c *shellWatchReductionState) appendSessionEvents(events []brokerapi.SessionWatchEvent) {
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

func (c *shellWatchReductionState) append(entry shellLiveActivityEntry) {
	if strings.TrimSpace(entry.EventType) == "" {
		return
	}
	c.Feed = append(c.Feed, entry)
	if len(c.Feed) <= shellLiveActivityFeedMax {
		return
	}
	c.Feed = c.Feed[len(c.Feed)-shellLiveActivityFeedMax:]
}

func deriveFamilyStreamState(err error, failures int) shellWatchStreamState {
	if err == nil {
		return shellWatchStreamHealthy
	}
	errText := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.HasPrefix(errText, "local_ipc_dial_error") || strings.HasPrefix(errText, "local_ipc_config_error") {
		return shellWatchStreamDisconnected
	}
	if failures <= 1 {
		return shellWatchStreamDegraded
	}
	return shellWatchStreamReconnecting
}
