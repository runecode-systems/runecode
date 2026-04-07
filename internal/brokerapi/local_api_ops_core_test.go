package brokerapi

import (
	"context"
	"encoding/base64"
	"io"
	"strings"
	"testing"
	"time"

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
	if runList.Runs[0].WorkspaceID != "workspace-run-123" {
		t.Fatalf("workspace_id = %q, want workspace-run-123", runList.Runs[0].WorkspaceID)
	}
	if runList.Runs[0].WorkflowKind == "" {
		t.Fatal("workflow_kind should be populated")
	}
	if runList.Runs[0].WorkflowDefinitionHash == "" {
		t.Fatal("workflow_definition_hash should be populated")
	}
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-get", RunID: "run-123"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if runGet.Run.Summary.RunID != "run-123" {
		t.Fatalf("run detail run_id = %q, want run-123", runGet.Run.Summary.RunID)
	}
	if len(runGet.Run.ActiveManifestHashes) == 0 {
		t.Fatal("run detail active_manifest_hashes should not be empty")
	}
	if runGet.Run.AuthoritativeState["source"] != "broker_store" {
		t.Fatalf("authoritative_state.source = %v, want broker_store", runGet.Run.AuthoritativeState["source"])
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

func TestRunGetFallsBackWhenAuditVerificationUnavailable(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, service, "run-fallback", "step-1")
	service.auditLedger = nil

	resp, errResp := service.HandleRunGet(context.Background(), RunGetRequest{
		SchemaID:      "runecode.protocol.v0.RunGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-run-fallback",
		RunID:         "run-fallback",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if !resp.Run.AuditSummary.CurrentlyDegraded {
		t.Fatal("audit summary should be degraded when verification surface is unavailable")
	}
	if resp.Run.AuditSummary.IntegrityStatus != "failed" {
		t.Fatalf("integrity_status = %q, want failed", resp.Run.AuditSummary.IntegrityStatus)
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
