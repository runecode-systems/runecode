package brokerapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func assertArtifactListAndHeadForLocalOps(t *testing.T, s *Service, digest string) {
	t.Helper()
	artList, errResp := s.HandleArtifactListV0(context.Background(), LocalArtifactListRequest{SchemaID: "runecode.protocol.v0.ArtifactListRequest", SchemaVersion: "0.1.0", RequestID: "req-art-list", Order: "created_at_desc", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactListV0 error response: %+v", errResp)
	}
	var listed *ArtifactSummary
	for i := range artList.Artifacts {
		if artList.Artifacts[i].Reference.Digest == digest {
			listed = &artList.Artifacts[i]
			break
		}
	}
	if listed == nil {
		t.Fatalf("artifact list missing digest %q: %+v", digest, artList.Artifacts)
	}
	if listed.RunID != "run-123" {
		t.Fatalf("artifact run_id = %q, want run-123", listed.RunID)
	}
	if listed.StageID != "artifact_flow" {
		t.Fatalf("artifact stage_id = %q, want artifact_flow", listed.StageID)
	}
	headReq := LocalArtifactHeadRequest{SchemaID: "runecode.protocol.v0.ArtifactHeadRequest", SchemaVersion: "0.1.0", RequestID: "req-art-head", Digest: digest}
	headResp, errResp := s.HandleArtifactHeadV0(context.Background(), headReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactHeadV0 error response: %+v", errResp)
	}
	if headResp.Artifact.Reference.Digest != digest {
		t.Fatalf("artifact head digest = %q, want %q", headResp.Artifact.Reference.Digest, digest)
	}
}

func TestArtifactListHonorsAscendingOrder(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	firstDigest := putPayloadArtifactForLocalOpsTest(t, s, "artifact-order-a", "run-order", "step-1")
	time.Sleep(1100 * time.Millisecond)
	secondDigest := putPayloadArtifactForLocalOpsTest(t, s, "artifact-order-b", "run-order", "step-2")

	resp, errResp := s.HandleArtifactListV0(context.Background(), LocalArtifactListRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-art-order-asc",
		RunID:         "run-order",
		Order:         "created_at_asc",
		Limit:         10,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactListV0 error response: %+v", errResp)
	}
	if len(resp.Artifacts) != 2 {
		t.Fatalf("artifact count = %d, want 2", len(resp.Artifacts))
	}
	if resp.Artifacts[0].Reference.Digest != firstDigest {
		t.Fatalf("first digest = %q, want %q", resp.Artifacts[0].Reference.Digest, firstDigest)
	}
	if resp.Artifacts[1].Reference.Digest != secondDigest {
		t.Fatalf("second digest = %q, want %q", resp.Artifacts[1].Reference.Digest, secondDigest)
	}
}

func putPayloadArtifactForLocalOpsTest(t *testing.T, s *Service, payload, runID, stepID string) string {
	t.Helper()
	ref, putErr := s.Put(artifacts.PutRequest{Payload: []byte(payload), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "workspace", RunID: runID, StepID: stepID})
	if putErr != nil {
		t.Fatalf("Put returned error: %v", putErr)
	}
	return ref.Digest
}

func putRunScopedArtifactForLocalOpsTest(t *testing.T, s *Service, runID, stepID string) string {
	t.Helper()
	ref, putErr := s.Put(artifacts.PutRequest{Payload: []byte("artifact-a"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "workspace", RunID: runID, StepID: stepID})
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

func TestArtifactLocalOpsRejectInFlightLimit(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	release, err := s.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	defer release()
	meta := RequestContext{ClientID: "client-a", LaneID: "lane-a"}
	assertArtifactLocalOpErrorCode(t, "artifact_list", "broker_limit_in_flight_exceeded", callArtifactListLocal(t, s, meta, "req-art-list-limit"))
	assertArtifactLocalOpErrorCode(t, "artifact_head", "broker_limit_in_flight_exceeded", callArtifactHeadLocal(t, s, meta, "req-art-head-limit"))
	assertArtifactLocalOpErrorCode(t, "artifact_read", "broker_limit_in_flight_exceeded", callArtifactReadLocal(t, s, meta, "req-art-read-limit"))
}

func TestArtifactLocalOpsRejectDeadlineExceeded(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	deadline := time.Now().Add(-time.Second)
	meta := RequestContext{Deadline: &deadline}
	assertArtifactLocalOpErrorCode(t, "artifact_list", "broker_timeout_request_deadline_exceeded", callArtifactListLocal(t, s, meta, "req-art-list-timeout"))
	assertArtifactLocalOpErrorCode(t, "artifact_head", "broker_timeout_request_deadline_exceeded", callArtifactHeadLocal(t, s, meta, "req-art-head-timeout"))
	assertArtifactLocalOpErrorCode(t, "artifact_read", "broker_timeout_request_deadline_exceeded", callArtifactReadLocal(t, s, meta, "req-art-read-timeout"))
}

func callArtifactListLocal(t *testing.T, s *Service, meta RequestContext, requestID string) *ErrorResponse {
	t.Helper()
	_, errResp := s.HandleArtifactListV0(context.Background(), LocalArtifactListRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
	}, meta)
	return errResp
}

func callArtifactHeadLocal(t *testing.T, s *Service, meta RequestContext, requestID string) *ErrorResponse {
	t.Helper()
	_, errResp := s.HandleArtifactHeadV0(context.Background(), LocalArtifactHeadRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactHeadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Digest:        "sha256:" + strings.Repeat("a", 64),
	}, meta)
	return errResp
}

func callArtifactReadLocal(t *testing.T, s *Service, meta RequestContext, requestID string) *ErrorResponse {
	t.Helper()
	_, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactReadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Digest:        "sha256:" + strings.Repeat("a", 64),
		ProducerRole:  "workspace",
		ConsumerRole:  "model_gateway",
	}, meta)
	return errResp
}

func assertArtifactLocalOpErrorCode(t *testing.T, opName, wantCode string, errResp *ErrorResponse) {
	t.Helper()
	if errResp == nil {
		t.Fatalf("%s expected typed error", opName)
	}
	if errResp.Error.Code != wantCode {
		t.Fatalf("%s error code = %q, want %s", opName, errResp.Error.Code, wantCode)
	}
}
