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

type alwaysErrReader struct{}

func (r *alwaysErrReader) Read(_ []byte) (int, error) {
	return 0, errors.New("forced stream read failure")
}
