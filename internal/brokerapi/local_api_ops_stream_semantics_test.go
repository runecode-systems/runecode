package brokerapi

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestArtifactStreamSemanticsRejectNonMonotonicSeq(t *testing.T) {
	err := validateArtifactStreamSemantics([]ArtifactStreamEvent{
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "artifact_stream_start", Digest: "sha256:" + strings.Repeat("a", 64), DataClass: "spec_text"},
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "artifact_stream_terminal", Digest: "sha256:" + strings.Repeat("a", 64), DataClass: "spec_text", Terminal: true, TerminalStatus: "completed"},
	})
	if err == nil {
		t.Fatal("validateArtifactStreamSemantics expected non-monotonic seq error")
	}
}

func TestLogStreamSemanticsRejectMultipleTerminalEvents(t *testing.T) {
	err := validateLogStreamSemantics([]LogStreamEvent{
		{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "log_stream_start"},
		{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 2, EventType: "log_stream_terminal", Terminal: true, TerminalStatus: "completed"},
		{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 3, EventType: "log_stream_terminal", Terminal: true, TerminalStatus: "completed"},
	})
	if err == nil {
		t.Fatal("validateLogStreamSemantics expected multiple terminal error")
	}
}

func TestLogStreamEventsCarryTypedErrorOnFailedTerminal(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ack, errResp := s.HandleLogStreamRequest(context.Background(), LogStreamRequest{SchemaID: "runecode.protocol.v0.LogStreamRequest", SchemaVersion: "0.1.0", RequestID: "req-log-fail", StreamID: "", StartCursor: "force_failure", Follow: false, IncludeBacklog: true}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleLogStreamRequest error response: %+v", errResp)
	}
	events, err := s.StreamLogEvents(ack)
	if err != nil {
		t.Fatalf("StreamLogEvents returned error: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("log stream events = %d, want at least start+terminal", len(events))
	}
	terminal := events[len(events)-1]
	if terminal.EventType != "log_stream_terminal" {
		t.Fatalf("last event_type = %q, want log_stream_terminal", terminal.EventType)
	}
	if terminal.TerminalStatus != "failed" {
		t.Fatalf("terminal_status = %q, want failed", terminal.TerminalStatus)
	}
	if terminal.Error == nil {
		t.Fatal("failed terminal event missing typed error envelope")
	}
}

func TestStreamSemanticsRejectCancelledTerminalWithError(t *testing.T) {
	err := validateLogStreamSemantics([]LogStreamEvent{
		{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "log_stream_start"},
		{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 2, EventType: "log_stream_terminal", Terminal: true, TerminalStatus: "cancelled", Error: &ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "request_cancelled", Category: "transport", Retryable: true, Message: "cancelled"}},
	})
	if err == nil {
		t.Fatal("validateLogStreamSemantics expected cancelled-with-error rejection")
	}
}

func TestRunWatchSemanticsRejectMultipleTerminalEvents(t *testing.T) {
	err := validateRunWatchSemantics([]RunWatchEvent{
		{SchemaID: "runecode.protocol.v0.RunWatchEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "run_watch_snapshot", Run: &RunSummary{SchemaID: "runecode.protocol.v0.RunSummary", SchemaVersion: "0.2.0", RunID: "run-1", WorkspaceID: "workspace-run-1", ProjectContextIdentity: "sha256:" + strings.Repeat("a", 64), CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z", LifecycleState: "active", PendingApprovalCount: 0, ApprovalProfile: "unknown", BackendKind: "unknown", IsolationAssuranceLevel: "unknown", ProvisioningPosture: "unknown", AssuranceLevel: "unknown", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok", AuditCurrentlyDegraded: false, RuntimePostureDegraded: false}},
		{SchemaID: "runecode.protocol.v0.RunWatchEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 2, EventType: "run_watch_terminal", Terminal: true, TerminalStatus: "completed"},
		{SchemaID: "runecode.protocol.v0.RunWatchEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 3, EventType: "run_watch_terminal", Terminal: true, TerminalStatus: "completed"},
	})
	if err == nil {
		t.Fatal("validateRunWatchSemantics expected multiple terminal error")
	}
}

func TestApprovalWatchSemanticsRejectNonMonotonicSeq(t *testing.T) {
	err := validateApprovalWatchSemantics([]ApprovalWatchEvent{
		{SchemaID: "runecode.protocol.v0.ApprovalWatchEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "approval_watch_snapshot", Approval: &ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: "sha256:" + strings.Repeat("a", 64), Status: "pending", RequestedAt: "2026-01-01T00:00:00Z", ApprovalTriggerCode: "manual_approval_required", ChangesIfApproved: "Promote reviewed file excerpts for downstream use.", ApprovalAssuranceLevel: "session_authenticated", PresenceMode: "os_confirmation", BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", ActionKind: "promotion"}}},
		{SchemaID: "runecode.protocol.v0.ApprovalWatchEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "approval_watch_terminal", Terminal: true, TerminalStatus: "completed"},
	})
	if err == nil {
		t.Fatal("validateApprovalWatchSemantics expected non-monotonic seq error")
	}
}

func TestSessionWatchSemanticsRejectCancelledTerminalWithError(t *testing.T) {
	err := validateSessionWatchSemantics([]SessionWatchEvent{
		{SchemaID: "runecode.protocol.v0.SessionWatchEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "session_watch_snapshot", Session: &SessionSummary{SchemaID: "runecode.protocol.v0.SessionSummary", SchemaVersion: "0.1.0", Identity: SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z"}, UpdatedAt: "2026-01-01T00:00:00Z", Status: "active", WorkPosture: "running", LastActivityKind: "chat_message", TurnCount: 1, LinkedRunCount: 1, LinkedApprovalCount: 0, LinkedArtifactCount: 0, LinkedAuditEventCount: 0, HasIncompleteTurn: false}},
		{SchemaID: "runecode.protocol.v0.SessionWatchEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 2, EventType: "session_watch_terminal", Terminal: true, TerminalStatus: "cancelled", Error: &ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "request_cancelled", Category: "transport", Retryable: true, Message: "cancelled"}},
	})
	if err == nil {
		t.Fatal("validateSessionWatchSemantics expected cancelled-with-error rejection")
	}
}

func TestSessionTurnExecutionWatchSemanticsRejectCancelledTerminalWithError(t *testing.T) {
	err := validateSessionTurnExecutionWatchSemantics([]SessionTurnExecutionWatchEvent{
		{SchemaID: "runecode.protocol.v0.SessionTurnExecutionWatchEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "session_turn_execution_watch_snapshot", TurnExecution: &SessionTurnExecution{SchemaID: "runecode.protocol.v0.SessionTurnExecution", SchemaVersion: "0.1.0", TurnID: "turn-1", SessionID: "sess-1", ExecutionIndex: 1, TriggerID: "trigger-1", TriggerSource: "interactive_user", RequestedOperation: "start", ExecutionState: "running", ApprovalProfile: "moderate", AutonomyPosture: "balanced", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}},
		{SchemaID: "runecode.protocol.v0.SessionTurnExecutionWatchEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 2, EventType: "session_turn_execution_watch_terminal", Terminal: true, TerminalStatus: "cancelled", Error: &ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "request_cancelled", Category: "transport", Retryable: true, Message: "cancelled"}},
	})
	if err == nil {
		t.Fatal("validateSessionTurnExecutionWatchSemantics expected cancelled-with-error rejection")
	}
}

func TestLogStreamHoldsInFlightSlotUntilStreamCompletes(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	meta := RequestContext{ClientID: "client-stream", LaneID: "lane-stream"}

	ack, errResp := openFollowLogStream(t, s, meta)
	if errResp != nil {
		t.Fatalf("HandleLogStreamRequest error response: %+v", errResp)
	}
	assertArtifactListBlockedWhileStreamOpen(t, s, meta)

	if _, err := s.StreamLogEvents(ack); err != nil {
		t.Fatalf("StreamLogEvents returned error: %v", err)
	}

	_, listErr := s.HandleArtifactListV0(context.Background(), LocalArtifactListRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-list-after-stream",
	}, meta)
	if listErr != nil {
		t.Fatalf("HandleArtifactListV0 after stream error response: %+v", listErr)
	}
}

func openFollowLogStream(t *testing.T, s *Service, meta RequestContext) (LogStreamRequest, *ErrorResponse) {
	t.Helper()
	return s.HandleLogStreamRequest(context.Background(), LogStreamRequest{SchemaID: "runecode.protocol.v0.LogStreamRequest", SchemaVersion: "0.1.0", RequestID: "req-log-open", Follow: true, IncludeBacklog: true}, meta)
}

func assertArtifactListBlockedWhileStreamOpen(t *testing.T, s *Service, meta RequestContext) {
	t.Helper()
	_, listErr := s.HandleArtifactListV0(context.Background(), LocalArtifactListRequest{SchemaID: "runecode.protocol.v0.ArtifactListRequest", SchemaVersion: "0.1.0", RequestID: "req-list-during-stream"}, meta)
	if listErr == nil {
		t.Fatal("expected in-flight saturation rejection while log stream is open")
	}
	if listErr.Error.Code != "broker_limit_in_flight_exceeded" {
		t.Fatalf("error code = %q, want broker_limit_in_flight_exceeded", listErr.Error.Code)
	}
	if !listErr.Error.Retryable {
		t.Fatal("in-flight saturation rejection should be retryable")
	}
}

type alwaysErrReader struct{}

func (r *alwaysErrReader) Read(_ []byte) (int, error) {
	return 0, errors.New("forced stream read failure")
}
