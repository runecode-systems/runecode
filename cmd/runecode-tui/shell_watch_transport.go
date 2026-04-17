package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const shellWatchPollInterval = 2 * time.Second
const shellWatchRetryMinDelay = 250 * time.Millisecond
const shellWatchRetryMaxDelay = 30 * time.Second

type shellActivityTickMsg struct{}

type shellWatchPollMsg struct{}

type shellWatchRunTransportResult struct {
	Events []brokerapi.RunWatchEvent
	Err    error
}

type shellWatchApprovalTransportResult struct {
	Events []brokerapi.ApprovalWatchEvent
	Err    error
}

type shellWatchSessionTransportResult struct {
	Events []brokerapi.SessionWatchEvent
	Err    error
}

type shellWatchTransportLoadedMsg struct {
	Run        shellWatchRunTransportResult
	Approval   shellWatchApprovalTransportResult
	Session    shellWatchSessionTransportResult
	ObservedAt time.Time
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
	return m.watchPollTickAfterCmd(shellWatchPollInterval)
}

func (m shellModel) watchPollTickAfterCmd(after time.Duration) tea.Cmd {
	if after <= 0 {
		after = shellWatchPollInterval
	}
	return tea.Tick(after, func(time.Time) tea.Msg {
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

		return shellWatchTransportLoadedMsg{
			Run:        shellWatchRunTransportResult{Events: runEvents, Err: runErr},
			Approval:   shellWatchApprovalTransportResult{Events: approvalEvents, Err: approvalErr},
			Session:    shellWatchSessionTransportResult{Events: sessionEvents, Err: sessionErr},
			ObservedAt: time.Now().UTC(),
		}
	}
}

func (m *shellModel) applyWatchTransport(msg shellWatchTransportLoadedMsg) {
	m.watch.applyTransport(msg)
	m.publishWatchStateToRoutes()
	m.refreshObjectIndexFromShellState()
}

func (m shellModel) activityTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return shellActivityTickMsg{}
	})
}

func (m shellWatchManager) nextPollDelay() time.Duration {
	now := m.now().UTC()
	minWait := time.Duration(0)
	hasRetryWindow := false
	for _, family := range shellWatchFamilies {
		state := m.families[family]
		if state.StreamState != shellWatchStreamDegraded && state.StreamState != shellWatchStreamReconnecting && state.StreamState != shellWatchStreamDisconnected {
			continue
		}
		if state.NextRetryAt.IsZero() {
			continue
		}
		wait := state.NextRetryAt.Sub(now)
		if wait < 0 {
			wait = 0
		}
		if !hasRetryWindow || wait < minWait {
			minWait = wait
			hasRetryWindow = true
		}
	}
	if !hasRetryWindow {
		return shellWatchPollInterval
	}
	if minWait < shellWatchRetryMinDelay {
		return shellWatchRetryMinDelay
	}
	return minWait
}

func shellWatchBackoffDelay(failureCount int) time.Duration {
	if failureCount <= 0 {
		return shellWatchRetryMinDelay
	}
	delay := 500 * time.Millisecond
	for i := 1; i < failureCount; i++ {
		delay *= 2
		if delay >= shellWatchRetryMaxDelay {
			return shellWatchRetryMaxDelay
		}
	}
	if delay < shellWatchRetryMinDelay {
		return shellWatchRetryMinDelay
	}
	if delay > shellWatchRetryMaxDelay {
		return shellWatchRetryMaxDelay
	}
	return delay
}
