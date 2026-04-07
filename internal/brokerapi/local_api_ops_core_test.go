package brokerapi

import (
	"context"
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestRunAndArtifactLocalTypedOperations(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	digest := putRunScopedArtifactForLocalOpsTest(t, s, "run-123", "step-1")
	assertRunListAndDetailForLocalOps(t, s)
	assertArtifactListAndHeadForLocalOps(t, s, digest)
	assertArtifactReadStreamCompletes(t, s, digest)
}

func assertRunListAndDetailForLocalOps(t *testing.T, s *Service) {
	t.Helper()
	runList, errResp := s.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "req-run-list", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunList error response: %+v", errResp)
	}
	if len(runList.Runs) != 1 || runList.Runs[0].RunID != "run-123" {
		t.Fatalf("run list = %+v, want run-123", runList.Runs)
	}
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-get", RunID: "run-123"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if runGet.Run.Summary.RunID != "run-123" {
		t.Fatalf("run detail run_id = %q, want run-123", runGet.Run.Summary.RunID)
	}
}

func assertArtifactListAndHeadForLocalOps(t *testing.T, s *Service, digest string) {
	t.Helper()
	artList, errResp := s.HandleArtifactListV0(context.Background(), LocalArtifactListRequest{SchemaID: "runecode.protocol.v0.ArtifactListRequest", SchemaVersion: "0.1.0", RequestID: "req-art-list", Order: "created_at_desc", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactListV0 error response: %+v", errResp)
	}
	if len(artList.Artifacts) != 1 || artList.Artifacts[0].RunID != "run-123" {
		t.Fatalf("artifact list = %+v, want run-scoped artifact", artList.Artifacts)
	}
	if artList.Artifacts[0].StageID != "artifact_flow" {
		t.Fatalf("artifact list stage_id = %q, want artifact_flow", artList.Artifacts[0].StageID)
	}
	headReq := LocalArtifactHeadRequest{SchemaID: "runecode.protocol.v0.ArtifactHeadRequest", SchemaVersion: "0.1.0", RequestID: "req-art-head", Digest: artList.Artifacts[0].Reference.Digest}
	headResp, errResp := s.HandleArtifactHeadV0(context.Background(), headReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactHeadV0 error response: %+v", errResp)
	}
	if headResp.Artifact.Reference.Digest != digest {
		t.Fatalf("artifact head digest = %q, want %q", headResp.Artifact.Reference.Digest, digest)
	}
}

func putRunScopedArtifactForLocalOpsTest(t *testing.T, s *Service, runID, stepID string) string {
	t.Helper()
	ref, putErr := s.Put(artifacts.PutRequest{Payload: []byte("artifact-a"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "workspace", RunID: runID, StepID: stepID})
	if putErr != nil {
		t.Fatalf("Put returned error: %v", putErr)
	}
	return ref.Digest
}

func assertArtifactReadStreamCompletes(t *testing.T, s *Service, digest string) {
	t.Helper()
	readReq := ArtifactReadRequest{SchemaID: "runecode.protocol.v0.ArtifactReadRequest", SchemaVersion: "0.1.0", RequestID: "req-art-read", Digest: digest, ProducerRole: "workspace", ConsumerRole: "model_gateway", ChunkBytes: 128}
	readResp, errResp := s.HandleArtifactRead(context.Background(), readReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactRead error response: %+v", errResp)
	}
	if readResp.StreamID == "" || readResp.Reader == nil {
		t.Fatalf("artifact read response invalid: %+v", readResp)
	}
	events, err := s.StreamArtifactReadEvents(readResp)
	if err != nil {
		t.Fatalf("StreamArtifactReadEvents error: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("artifact stream events = %d, want at least start+terminal", len(events))
	}
	if events[0].EventType != "artifact_stream_start" {
		t.Fatalf("first stream event_type = %q, want artifact_stream_start", events[0].EventType)
	}
	assertStreamSeqMonotonic(t, events)
	assertSingleArtifactTerminalEvent(t, events)
	if got := events[len(events)-1].TerminalStatus; got != "completed" {
		t.Fatalf("terminal_status = %q, want completed", got)
	}
}

func assertStreamSeqMonotonic(t *testing.T, events []ArtifactStreamEvent) {
	t.Helper()
	for i := 1; i < len(events); i++ {
		if events[i].Seq <= events[i-1].Seq {
			t.Fatalf("stream seq not monotonic: prev=%d curr=%d", events[i-1].Seq, events[i].Seq)
		}
	}
}

func assertSingleArtifactTerminalEvent(t *testing.T, events []ArtifactStreamEvent) {
	t.Helper()
	terminalCount := 0
	for _, event := range events {
		if event.EventType != "artifact_stream_terminal" {
			continue
		}
		terminalCount++
		if event.TerminalStatus == "" {
			t.Fatal("artifact terminal event missing in-band terminal_status")
		}
	}
	if terminalCount != 1 {
		t.Fatalf("terminal event count = %d, want 1", terminalCount)
	}
}

func TestArtifactReadRejectsRangeRequestsInMVP(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref, putErr := s.Put(artifacts.PutRequest{Payload: []byte("artifact-a"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "workspace"})
	if putErr != nil {
		t.Fatalf("Put returned error: %v", putErr)
	}
	rangeStart := int64(0)
	_, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{SchemaID: "runecode.protocol.v0.ArtifactReadRequest", SchemaVersion: "0.1.0", RequestID: "req-range", Digest: ref.Digest, ProducerRole: "workspace", ConsumerRole: "model_gateway", RangeStart: &rangeStart}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleArtifactRead expected range rejection error")
	}
	if errResp.Error.Code != "broker_validation_range_not_supported" {
		t.Fatalf("error code = %q, want broker_validation_range_not_supported", errResp.Error.Code)
	}
}

func TestRunGetNotFoundUsesRunSpecificCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleRunGet(context.Background(), RunGetRequest{
		SchemaID:      "runecode.protocol.v0.RunGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-run-missing",
		RunID:         "run-missing",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunGet expected not-found error")
	}
	if errResp.Error.Code != "broker_not_found_run" {
		t.Fatalf("error code = %q, want broker_not_found_run", errResp.Error.Code)
	}
}

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

func TestArtifactStreamEventsCloseWithSingleTerminalOnReadFailure(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	events, err := s.StreamArtifactReadEvents(ArtifactReadHandle{RequestID: "req-stream-fail", Digest: "sha256:" + strings.Repeat("a", 64), DataClass: artifacts.DataClassSpecText, StreamID: "stream-fail", ChunkBytes: 8, Reader: io.NopCloser(&alwaysErrReader{})})
	if err != nil {
		t.Fatalf("StreamArtifactReadEvents returned error: %v", err)
	}
	assertSingleFailedArtifactTerminal(t, events)
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
