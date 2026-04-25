package main

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func BenchmarkShellViewEmpty(b *testing.B) {
	m := newShellModel()
	m.width = 120
	m.height = 40
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkShellViewWaitingSession(b *testing.B) {
	m := newShellModel()
	m.width = 120
	m.height = 40
	m.watch.reduction.sessions = map[string]brokerapi.SessionSummary{
		"session-wait": {
			Identity:          brokerapi.SessionIdentity{SessionID: "session-wait", WorkspaceID: "ws-1"},
			Status:            "waiting_external_dependency",
			HasIncompleteTurn: true,
		},
	}
	m.watch.projection.Activity = shellActivitySemantics{State: shellActivityStateWaiting, Active: shellActivityFocus{Kind: "session", ID: "session-wait"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkShellWatchApply(b *testing.B) {
	msg := shellWatchTransportLoadedMsg{
		Run:      shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &brokerapi.RunSummary{RunID: "run-1", LifecycleState: "active"}}}},
		Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &brokerapi.ApprovalSummary{ApprovalID: "ap-1", Status: "pending"}}}},
		Session:  shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active"}}}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := newShellModel()
		b.StartTimer()
		m.applyWatchTransport(msg)
	}
}

func BenchmarkBuildPaletteEntries(b *testing.B) {
	m := newShellModel()
	m.objectIndex.ingestSessions([]brokerapi.SessionSummary{{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", LastActivityKind: "chat_message"}})
	m.objectIndex.ingestRuns([]brokerapi.RunSummary{{RunID: "run-1", WorkspaceID: "ws-1", LifecycleState: "active"}})
	m.objectIndex.ingestApprovals([]brokerapi.ApprovalSummary{{ApprovalID: "ap-1", Status: "pending"}})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.buildPaletteEntries()
	}
}
