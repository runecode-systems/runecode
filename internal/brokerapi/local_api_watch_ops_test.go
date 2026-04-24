package brokerapi

import (
	"context"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestRunWatchStreamIncludesSnapshotUpsertAndTerminal(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-watch", "step-1")
	if err := s.RecordRuntimeFacts("run-watch", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-watch", SessionID: "sess-watch"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}

	ack, errResp := s.HandleRunWatchRequest(context.Background(), RunWatchRequest{
		SchemaID:        "runecode.protocol.v0.RunWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-run-watch",
		StreamID:        "",
		RunID:           "run-watch",
		Follow:          true,
		IncludeSnapshot: true,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunWatchRequest error response: %+v", errResp)
	}
	events, err := s.StreamRunWatchEvents(ack)
	if err != nil {
		t.Fatalf("StreamRunWatchEvents returned error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("run watch events len = %d, want 3", len(events))
	}
	if events[0].EventType != "run_watch_snapshot" || events[1].EventType != "run_watch_upsert" {
		t.Fatalf("run watch event types = [%q,%q], want snapshot/upsert", events[0].EventType, events[1].EventType)
	}
	terminal := events[2]
	if terminal.EventType != "run_watch_terminal" || !terminal.Terminal || terminal.TerminalStatus != "completed" {
		t.Fatalf("run watch terminal = %+v, want completed terminal", terminal)
	}
}

func TestApprovalWatchStreamIncludesSnapshotAndTerminal(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref := s.mustPutApprovalFixtureArtifact(t)
	approvalID := createPendingApprovalFromPolicyDecision(t, s, "run-approval-watch", "step-1", ref)

	ack, errResp := s.HandleApprovalWatchRequest(context.Background(), ApprovalWatchRequest{
		SchemaID:        "runecode.protocol.v0.ApprovalWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-approval-watch",
		StreamID:        "",
		ApprovalID:      approvalID,
		Follow:          false,
		IncludeSnapshot: true,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalWatchRequest error response: %+v", errResp)
	}
	events, err := s.StreamApprovalWatchEvents(ack)
	if err != nil {
		t.Fatalf("StreamApprovalWatchEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("approval watch events len = %d, want 2", len(events))
	}
	if events[0].EventType != "approval_watch_snapshot" || events[0].Approval == nil || events[0].Approval.ApprovalID != approvalID {
		t.Fatalf("approval watch snapshot = %+v, want approval %q", events[0], approvalID)
	}
	if events[1].EventType != "approval_watch_terminal" || events[1].TerminalStatus != "completed" {
		t.Fatalf("approval watch terminal = %+v, want completed terminal", events[1])
	}
}

func TestSessionWatchStreamIncludesSnapshotAndTerminal(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-session-watch", "step-1")
	if err := s.RecordRuntimeFacts("run-session-watch", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-session-watch", SessionID: "sess-watch"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}

	ack, errResp := s.HandleSessionWatchRequest(context.Background(), SessionWatchRequest{
		SchemaID:        "runecode.protocol.v0.SessionWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-session-watch",
		StreamID:        "",
		SessionID:       "sess-watch",
		Follow:          false,
		IncludeSnapshot: true,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionWatchRequest error response: %+v", errResp)
	}
	events, err := s.StreamSessionWatchEvents(ack)
	if err != nil {
		t.Fatalf("StreamSessionWatchEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("session watch events len = %d, want 2", len(events))
	}
	if events[0].EventType != "session_watch_snapshot" || events[0].Session == nil || events[0].Session.Identity.SessionID != "sess-watch" {
		t.Fatalf("session watch snapshot = %+v, want sess-watch", events[0])
	}
	if events[0].Session.WorkPosture != "running" {
		t.Fatalf("session watch snapshot work_posture = %q, want running", events[0].Session.WorkPosture)
	}
	if events[1].EventType != "session_watch_terminal" || events[1].TerminalStatus != "completed" {
		t.Fatalf("session watch terminal = %+v, want completed terminal", events[1])
	}
}

func TestRunWatchTerminalCancelledOnContextCancel(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-watch-cancel", "step-1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ack, errResp := s.HandleRunWatchRequest(ctx, RunWatchRequest{
		SchemaID:        "runecode.protocol.v0.RunWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-run-watch-cancel",
		StreamID:        "run-watch-cancel",
		Follow:          true,
		IncludeSnapshot: true,
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunWatchRequest expected cancelled request error")
	}
	if ack.RequestID != "" {
		t.Fatalf("ack request id = %q, want empty on error", ack.RequestID)
	}
}

func TestSessionTurnExecutionWatchStreamIncludesSnapshotAndTerminal(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionTurnExecutionWatchFixture(t, s, "run-session-turn-watch", "sess-turn-watch", "req-session-turn-watch-trigger", "hello")
	ack := mustHandleSessionTurnExecutionWatchRequest(t, s, SessionTurnExecutionWatchRequest{
		SchemaID:        "runecode.protocol.v0.SessionTurnExecutionWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-session-turn-watch",
		StreamID:        "",
		SessionID:       "sess-turn-watch",
		Follow:          false,
		IncludeSnapshot: true,
	})
	events, err := s.StreamSessionTurnExecutionWatchEvents(ack)
	if err != nil {
		t.Fatalf("StreamSessionTurnExecutionWatchEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("session turn execution watch events len = %d, want 2", len(events))
	}
	if events[0].EventType != "session_turn_execution_watch_snapshot" || events[0].TurnExecution == nil || events[0].TurnExecution.SessionID != "sess-turn-watch" {
		t.Fatalf("session turn execution watch snapshot = %+v, want sess-turn-watch", events[0])
	}
	if events[0].TurnExecution.ExecutionState != "running" {
		t.Fatalf("session turn execution watch snapshot execution_state = %q, want running", events[0].TurnExecution.ExecutionState)
	}
	if events[1].EventType != "session_turn_execution_watch_terminal" || events[1].TerminalStatus != "completed" {
		t.Fatalf("session turn execution watch terminal = %+v, want completed terminal", events[1])
	}
}

func TestSessionWatchFollowWithoutSnapshotUsesLatestUpsert(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return now })
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-watch-a", "sess-watch-a")
	now = now.Add(time.Second)
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-watch-b", "sess-watch-b")

	ack, errResp := s.HandleSessionWatchRequest(context.Background(), SessionWatchRequest{
		SchemaID:        "runecode.protocol.v0.SessionWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-session-watch-follow-no-snapshot",
		Follow:          true,
		IncludeSnapshot: false,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionWatchRequest error response: %+v", errResp)
	}
	events, err := s.StreamSessionWatchEvents(ack)
	if err != nil {
		t.Fatalf("StreamSessionWatchEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("session watch events len = %d, want 2", len(events))
	}
	if events[0].EventType != "session_watch_upsert" {
		t.Fatalf("event_type = %q, want session_watch_upsert", events[0].EventType)
	}
	if events[0].Session == nil {
		t.Fatal("upsert session missing")
	}
	if events[0].Session.Identity.SessionID != "sess-watch-b" {
		t.Fatalf("upsert session_id = %q, want latest sess-watch-b", events[0].Session.Identity.SessionID)
	}
	if events[1].EventType != "session_watch_terminal" {
		t.Fatalf("terminal event_type = %q, want session_watch_terminal", events[1].EventType)
	}
}

func TestSessionTurnExecutionWatchFollowWithoutSnapshotUsesLatestUpsert(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := seedFollowWithoutSnapshotWatchFixture(t, s)
	ack := newSessionTurnExecutionWatchAck(t, s, SessionTurnExecutionWatchRequest{
		SchemaID:        "runecode.protocol.v0.SessionTurnExecutionWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-session-turn-watch-follow-no-snapshot",
		Follow:          true,
		IncludeSnapshot: false,
	})
	_ = now
	events, err := s.StreamSessionTurnExecutionWatchEvents(ack)
	if err != nil {
		t.Fatalf("StreamSessionTurnExecutionWatchEvents returned error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("session turn execution watch events len = %d, want 3", len(events))
	}
	if events[0].EventType != "session_turn_execution_watch_upsert" {
		t.Fatalf("event_type = %q, want session_turn_execution_watch_upsert", events[0].EventType)
	}
	if events[0].TurnExecution == nil {
		t.Fatal("upsert turn_execution missing")
	}
	if events[0].TurnExecution.SessionID != "sess-turn-watch-b" {
		t.Fatalf("upsert session_id = %q, want latest sess-turn-watch-b", events[0].TurnExecution.SessionID)
	}
	if events[1].EventType != "session_turn_execution_watch_upsert" {
		t.Fatalf("second event_type = %q, want session_turn_execution_watch_upsert", events[1].EventType)
	}
	if events[1].TurnExecution == nil || events[1].TurnExecution.SessionID != "sess-turn-watch-a" {
		t.Fatalf("second upsert session_id = %q, want sess-turn-watch-a", valueSessionID(events[1].TurnExecution))
	}
	if events[2].EventType != "session_turn_execution_watch_terminal" {
		t.Fatalf("terminal event_type = %q, want session_turn_execution_watch_terminal", events[2].EventType)
	}
}

func TestSessionTurnExecutionWatchFollowIncludesSnapshotThenProgressUpdates(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedFollowWithSnapshotWatchFixture(t, s)
	ack := newSessionTurnExecutionWatchAck(t, s, SessionTurnExecutionWatchRequest{
		SchemaID:        "runecode.protocol.v0.SessionTurnExecutionWatchRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-session-turn-watch-follow-snapshot",
		SessionID:       "sess-turn-watch-main",
		Follow:          true,
		IncludeSnapshot: true,
	})
	events, err := s.StreamSessionTurnExecutionWatchEvents(ack)
	if err != nil {
		t.Fatalf("StreamSessionTurnExecutionWatchEvents returned error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("session turn execution watch events len = %d, want 3", len(events))
	}
	if events[0].EventType != "session_turn_execution_watch_snapshot" {
		t.Fatalf("snapshot event_type = %q, want session_turn_execution_watch_snapshot", events[0].EventType)
	}
	if events[0].TurnExecution == nil || events[0].TurnExecution.ExecutionIndex != 2 {
		t.Fatalf("snapshot execution_index = %d, want 2", valueExecutionIndex(events[0].TurnExecution))
	}
	if events[1].EventType != "session_turn_execution_watch_upsert" {
		t.Fatalf("upsert event_type = %q, want session_turn_execution_watch_upsert", events[1].EventType)
	}
	if events[1].TurnExecution == nil || events[1].TurnExecution.ExecutionIndex != 1 {
		t.Fatalf("upsert execution_index = %d, want 1", valueExecutionIndex(events[1].TurnExecution))
	}
	if events[2].EventType != "session_turn_execution_watch_terminal" {
		t.Fatalf("terminal event_type = %q, want session_turn_execution_watch_terminal", events[2].EventType)
	}
}

func TestSessionTurnExecutionWatchIncludesMultiplePendingWaitScopes(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-turn-watch-multi", "sess-turn-watch-multi")
	first := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-turn-watch-multi-1", SessionID: "sess-turn-watch-multi", TriggerSource: "autonomous_background", RequestedOperation: "start", AutonomyPosture: "operator_guided", UserMessageContentText: "first"})
	updateWatchExecutionWait(t, s, first.TurnID, "operator_input", "waiting_operator_input")
	second := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-turn-watch-multi-2", SessionID: "sess-turn-watch-multi", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "second"})
	updateWatchExecutionWait(t, s, second.TurnID, "external_dependency", "waiting_external_dependency")
	ack := mustHandleSessionTurnExecutionWatchRequest(t, s, SessionTurnExecutionWatchRequest{SchemaID: "runecode.protocol.v0.SessionTurnExecutionWatchRequest", SchemaVersion: "0.1.0", RequestID: "req-session-turn-watch-multi-watch", SessionID: "sess-turn-watch-multi", Follow: true, IncludeSnapshot: false})
	events, err := s.StreamSessionTurnExecutionWatchEvents(ack)
	if err != nil {
		t.Fatalf("StreamSessionTurnExecutionWatchEvents returned error: %v", err)
	}
	assertWatchIncludesPendingWaitScopes(t, events, "operator_input", "external_dependency")
}

func updateWatchExecutionWait(t *testing.T, s *Service, turnID, waitKind, waitState string) {
	t.Helper()
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: "sess-turn-watch-multi", TurnID: turnID, ExecutionState: "waiting", WaitKind: waitKind, WaitState: waitState, OccurredAt: s.currentTimestamp()}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
}

func assertWatchIncludesPendingWaitScopes(t *testing.T, events []SessionTurnExecutionWatchEvent, expectedWaitKinds ...string) {
	t.Helper()
	if len(events) < 3 {
		t.Fatalf("watch events len = %d, want at least 3", len(events))
	}
	waitKinds := map[string]struct{}{}
	upserts := 0
	for _, event := range events {
		if event.EventType != "session_turn_execution_watch_upsert" || event.TurnExecution == nil {
			continue
		}
		upserts++
		waitKinds[event.TurnExecution.WaitKind] = struct{}{}
	}
	if upserts < 2 {
		t.Fatalf("upsert count = %d, want at least 2", upserts)
	}
	for _, waitKind := range expectedWaitKinds {
		if _, ok := waitKinds[waitKind]; !ok {
			t.Fatalf("watch upserts missing %s wait: %+v", waitKind, waitKinds)
		}
	}
}

func seedFollowWithoutSnapshotWatchFixture(t *testing.T, s *Service) time.Time {
	t.Helper()
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return now })
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-turn-watch-a", "sess-turn-watch-a")
	now = now.Add(time.Second)
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-turn-watch-b", "sess-turn-watch-b")
	triggerSessionExecutionWatch(t, s, "req-session-turn-watch-a", "sess-turn-watch-a", "first")
	now = now.Add(time.Second)
	triggerSessionExecutionWatch(t, s, "req-session-turn-watch-b", "sess-turn-watch-b", "second")
	return now
}

func seedFollowWithSnapshotWatchFixture(t *testing.T, s *Service) {
	t.Helper()
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return now })
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-turn-watch-main", "sess-turn-watch-main")
	triggerSessionExecutionWatch(t, s, "req-session-turn-watch-main", "sess-turn-watch-main", "first")
	completeSessionTurnExecutionForWatch(t, s, now)
	now = now.Add(time.Second)
	triggerSessionExecutionWatch(t, s, "req-session-turn-watch-main-2", "sess-turn-watch-main", "second")
}

func completeSessionTurnExecutionForWatch(t *testing.T, s *Service, now time.Time) {
	t.Helper()
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: "sess-turn-watch-main", TurnID: "sess-turn-watch-main.exec.000001", ExecutionState: "completed", TerminalOutcome: "completed", OccurredAt: now.Add(500 * time.Millisecond)}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
}

func newSessionTurnExecutionWatchAck(t *testing.T, s *Service, req SessionTurnExecutionWatchRequest) SessionTurnExecutionWatchRequest {
	t.Helper()
	return mustHandleSessionTurnExecutionWatchRequest(t, s, req)
}

func valueSessionID(execution *SessionTurnExecution) string {
	if execution == nil {
		return ""
	}
	return execution.SessionID
}

func valueExecutionIndex(execution *SessionTurnExecution) int {
	if execution == nil {
		return 0
	}
	return execution.ExecutionIndex
}

func seedSessionTurnExecutionWatchFixture(t *testing.T, s *Service, runID, sessionID, requestID, content string) {
	t.Helper()
	putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	if err := s.RecordRuntimeFacts(runID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: runID, SessionID: sessionID}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	triggerSessionExecutionWatch(t, s, requestID, sessionID, content)
}

func triggerSessionExecutionWatch(t *testing.T, s *Service, requestID, sessionID, content string) {
	t.Helper()
	if _, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: requestID, SessionID: sessionID, TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: content}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleSessionExecutionTrigger error response: %+v", errResp)
	}
}

func mustHandleSessionTurnExecutionWatchRequest(t *testing.T, s *Service, req SessionTurnExecutionWatchRequest) SessionTurnExecutionWatchRequest {
	t.Helper()
	ack, errResp := s.HandleSessionTurnExecutionWatchRequest(context.Background(), req, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionTurnExecutionWatchRequest error response: %+v", errResp)
	}
	return ack
}

func (s *Service) mustPutApprovalFixtureArtifact(t *testing.T) string {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               []byte("private excerpt"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		CreatedByRole:         "workspace",
		RunID:                 "run-approval-watch",
		StepID:                "step-1",
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	return ref.Digest
}
