package brokerapi

import (
	"context"
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func assertArtifactStreamDecodedPayload(t *testing.T, events []ArtifactStreamEvent, want string) {
	t.Helper()
	decoded := ""
	for _, event := range events {
		if event.EventType != "artifact_stream_chunk" {
			continue
		}
		chunk, decodeErr := base64.StdEncoding.DecodeString(event.ChunkBase64)
		if decodeErr != nil {
			t.Fatalf("chunk decode error: %v", decodeErr)
		}
		decoded += string(chunk)
	}
	if decoded != want {
		t.Fatalf("decoded artifact stream payload = %q, want %q", decoded, want)
	}
}

func TestArtifactReadGatewayFlowDeniedForMismatchedProducerRole(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref, putErr := s.Put(artifacts.PutRequest{Payload: []byte("artifact-a"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "workspace"})
	if putErr != nil {
		t.Fatalf("Put returned error: %v", putErr)
	}
	_, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{SchemaID: "runecode.protocol.v0.ArtifactReadRequest", SchemaVersion: "0.1.0", RequestID: "req-role-mismatch", Digest: ref.Digest, ProducerRole: "auditd", ConsumerRole: "model_gateway", DataClass: string(artifacts.DataClassSpecText)}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleArtifactRead expected policy rejection for mismatched producer")
	}
	if errResp.Error.Code != "broker_limit_policy_rejected" {
		t.Fatalf("error code = %q, want broker_limit_policy_rejected", errResp.Error.Code)
	}
}

func TestArtifactReadFailsClosedWhenPolicyContextUnavailable(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-missing-policy-context"
	ref, putErr := s.Put(artifacts.PutRequest{
		Payload:               []byte("artifact-a"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64),
		CreatedByRole:         "workspace",
		RunID:                 runID,
		StepID:                "step-1",
	})
	if putErr != nil {
		t.Fatalf("Put returned error: %v", putErr)
	}
	_, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactReadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-policy-missing",
		Digest:        ref.Digest,
		ProducerRole:  "workspace",
		ConsumerRole:  "model_gateway",
		DataClass:     string(artifacts.DataClassSpecText),
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleArtifactRead expected policy rejection when trusted policy context is unavailable")
	}
	if errResp.Error.Code != "broker_limit_policy_rejected" {
		t.Fatalf("error code = %q, want broker_limit_policy_rejected", errResp.Error.Code)
	}
}

func TestArtifactStreamEventsCloseWithSingleTerminalOnReadFailure(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	events, err := s.StreamArtifactReadEvents(ArtifactReadHandle{RequestID: "req-stream-fail", Digest: "sha256:" + strings.Repeat("a", 64), DataClass: artifacts.DataClassSpecText, StreamID: "stream-fail", ChunkBytes: 8, Reader: io.NopCloser(&alwaysErrReader{})})
	if err != nil {
		t.Fatalf("StreamArtifactReadEvents returned error: %v", err)
	}
	assertSingleFailedArtifactTerminal(t, events)
}

func TestArtifactStreamOverflowUsesResponseStreamLimitCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxResponseStreamBytes: 16, MaxStreamChunkBytes: 8}})
	handle := ArtifactReadHandle{
		RequestID:  "req-stream-overflow",
		Digest:     "sha256:" + strings.Repeat("b", 64),
		DataClass:  artifacts.DataClassSpecText,
		StreamID:   "stream-overflow",
		ChunkBytes: 8,
		Reader:     io.NopCloser(strings.NewReader(strings.Repeat("x", 32))),
	}
	events, err := s.StreamArtifactReadEvents(handle)
	if err != nil {
		t.Fatalf("StreamArtifactReadEvents returned error: %v", err)
	}
	for _, event := range events {
		if event.EventType != "artifact_stream_terminal" || event.TerminalStatus != "failed" || event.Error == nil {
			continue
		}
		if event.Error.Code != "broker_limit_response_stream_size_exceeded" {
			t.Fatalf("error code = %q, want broker_limit_response_stream_size_exceeded", event.Error.Code)
		}
		return
	}
	t.Fatal("missing failed terminal event with typed stream overflow code")
}

func assertSingleFailedArtifactTerminal(t *testing.T, events []ArtifactStreamEvent) {
	t.Helper()
	terminal := 0
	for _, event := range events {
		if event.EventType != "artifact_stream_terminal" {
			continue
		}
		terminal++
		if event.TerminalStatus != "failed" {
			t.Fatalf("terminal_status = %q, want failed", event.TerminalStatus)
		}
		if event.Error == nil {
			t.Fatal("terminal failure event missing typed error envelope")
		}
	}
	if terminal != 1 {
		t.Fatalf("terminal event count = %d, want 1", terminal)
	}
}
