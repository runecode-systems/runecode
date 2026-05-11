//go:build runecode_tui_snapshot

package main

import (
	"time"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func watchProjection(msg shellWatchTransportLoadedMsg) shellWatchProjectionState {
	manager := newShellWatchManager()
	manager.applyTransport(msg)
	return manager.projection
}

func snapshotWatchHealthyEmpty() shellWatchTransportLoadedMsg {
	return shellWatchTransportLoadedMsg{ObservedAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)}
}

func snapshotWatchApprovalWaiting() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 5, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", PendingApprovalCount: 1}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-1", Status: "pending"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", LastActivityPreview: "Awaiting approval"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}}}}
}

func snapshotWatchBlocked() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 10, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-blocked", LifecycleState: "blocked", BlockingReasonCode: "approval_wait", PendingApprovalCount: 1}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-blocked", Status: "pending"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}, Status: "blocked", LastActivityPreview: "Blocked on approval"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}}}}
}

func snapshotWatchDegraded() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 15, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-degraded", LifecycleState: "active", RuntimePostureDegraded: true, AuditCurrentlyDegraded: true}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-quiet", Status: "consumed"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "degraded", LastActivityPreview: "Runtime posture degraded"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}, {EventType: "run_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}, {EventType: "session_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"}}}}
}

func snapshotWatchActionCenter() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 20, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-degraded", LifecycleState: "active", RuntimePostureDegraded: true}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-urgent", Status: "pending"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}, Status: "active", LastActivityPreview: "Triage action center"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}}}}
}
