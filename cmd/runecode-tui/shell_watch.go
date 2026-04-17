package main

import (
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type shellWatchFamily string

const (
	shellWatchFamilyRun      shellWatchFamily = "run_watch"
	shellWatchFamilyApproval shellWatchFamily = "approval_watch"
	shellWatchFamilySession  shellWatchFamily = "session_watch"
)

var shellWatchFamilies = []shellWatchFamily{shellWatchFamilyRun, shellWatchFamilyApproval, shellWatchFamilySession}

type shellWatchStreamState string

const (
	shellWatchStreamLoading      shellWatchStreamState = "loading"
	shellWatchStreamHealthy      shellWatchStreamState = "healthy"
	shellWatchStreamDegraded     shellWatchStreamState = "degraded"
	shellWatchStreamReconnecting shellWatchStreamState = "reconnecting"
	shellWatchStreamDisconnected shellWatchStreamState = "disconnected"
)

type shellSyncState string

const (
	shellSyncStateLoading      shellSyncState = "loading"
	shellSyncStateHealthy      shellSyncState = "healthy"
	shellSyncStateDegraded     shellSyncState = "degraded"
	shellSyncStateReconnecting shellSyncState = "reconnecting"
	shellSyncStateDisconnected shellSyncState = "disconnected"
)

type shellSyncHealth struct {
	State                shellSyncState
	ErrorText            string
	DegradedFamilies     []string
	ReconnectingFamilies []string
	DisconnectedFamilies []string
}

type shellLiveActivityEntry struct {
	Family    string
	EventType string
	Subject   string
	Status    string
}

type shellWatchReductionState struct {
	Feed      []shellLiveActivityEntry
	runs      map[string]brokerapi.RunSummary
	approvals map[string]brokerapi.ApprovalSummary
	sessions  map[string]brokerapi.SessionSummary
}

type shellWatchFamilyState struct {
	Family              shellWatchFamily
	StreamState         shellWatchStreamState
	Summary             watchFamilySummary
	LastErrorText       string
	ConsecutiveFailures int
	LastSuccessAt       time.Time
	LastFailureAt       time.Time
	NextRetryAt         time.Time
}

type shellWatchProjectionState struct {
	Live     dashboardLiveActivity
	Feed     []shellLiveActivityEntry
	Health   shellSyncHealth
	Activity shellActivitySemantics
}

type shellWatchManager struct {
	now        func() time.Time
	families   map[shellWatchFamily]shellWatchFamilyState
	reduction  shellWatchReductionState
	projection shellWatchProjectionState
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
