package brokerapi

import (
	"context"
	"testing"

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
