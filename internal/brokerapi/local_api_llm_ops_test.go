package brokerapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestHandleLLMInvokeAndStreamFailClosedUntilAuthoritativeExecutionWiringExists(t *testing.T) {
	s, runID, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	_, errResp := s.HandleLLMInvoke(context.Background(), LLMInvokeRequest{SchemaID: "runecode.protocol.v0.LLMInvokeRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-invoke", RunID: runID, LLMRequest: requestObject, RequestDigest: &requestDigest}, RequestContext{})
	assertLLMExecutionUnavailable(t, errResp)

	_, _, _, errResp = s.HandleLLMStreamRequest(context.Background(), LLMStreamRequest{SchemaID: "runecode.protocol.v0.LLMStreamRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-stream", RunID: runID, StreamID: "llm-stream-1", LLMRequest: requestObject, RequestDigest: &requestDigest, Follow: false}, RequestContext{})
	assertLLMExecutionUnavailable(t, errResp)

	_, err := s.StreamLLMEvents(LLMStreamRequest{RequestID: "req-llm-stream"}, llmExecutionBinding{}, artifacts.ArtifactReference{})
	if err == nil || !strings.Contains(err.Error(), llmExecutionUnavailableMessage) {
		t.Fatalf("StreamLLMEvents error = %v, want unavailable message", err)
	}
}

func assertLLMExecutionUnavailable(t *testing.T, errResp *ErrorResponse) {
	t.Helper()
	if errResp == nil {
		t.Fatal("expected unavailable error response")
	}
	if errResp.Error.Code != "gateway_failure" {
		t.Fatalf("error code = %q, want gateway_failure", errResp.Error.Code)
	}
	if errResp.Error.Message != llmExecutionUnavailableMessage {
		t.Fatalf("error message = %q, want %q", errResp.Error.Message, llmExecutionUnavailableMessage)
	}
}

func prepareCanonicalLLMInvokeTest(t *testing.T) (*Service, string, map[string]any, trustpolicy.Digest, artifacts.ArtifactReference) {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-llm-local-api"
	putTrustedModelGatewayContextForRun(t, s, runID, []any{trustedModelGatewayAllowlistEntry()})
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"model.example.com": {"93.184.216.34"}}}
	input := putPayloadArtifactForLocalOpsTest(t, s, "llm input", runID, "step-input")
	llmRequestPayload := validLLMRequestPayload(input)
	llmRequestRaw := mustCanonicalJSONBytes(t, llmRequestPayload)
	requestRef, err := s.Put(artifacts.PutRequest{Payload: llmRequestRaw, ContentType: "application/json", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "broker", TrustedSource: true, RunID: runID, StepID: "step-llm"})
	if err != nil {
		t.Fatalf("Put canonical LLMRequest artifact returned error: %v", err)
	}
	requestObject := map[string]any{}
	if err := json.Unmarshal(llmRequestRaw, &requestObject); err != nil {
		t.Fatalf("Unmarshal canonical LLMRequest payload returned error: %v", err)
	}
	requestDigest := mustDigestFromIdentityForTest(t, requestRef.Digest)
	return s, runID, requestObject, requestDigest, requestRef
}

func validLLMRequestPayload(inputDigest string) map[string]any {
	return map[string]any{
		"schema_id":        "runecode.protocol.v0.LLMRequest",
		"schema_version":   "0.3.0",
		"selection_source": "signed_allowlist",
		"provider":         "provider-test",
		"model":            "model-test",
		"input_artifacts": []any{map[string]any{
			"schema_id":      "runecode.protocol.v0.ArtifactReference",
			"schema_version": "0.3.0",
			"digest":         digestPayload(inputDigest),
			"size_bytes":     8,
			"content_type":   "text/plain",
			"data_class":     "spec_text",
			"provenance_receipt_hash": map[string]any{
				"hash_alg": "sha256",
				"hash":     strings.Repeat("b", 64),
			},
		}},
		"tool_allowlist": []any{map[string]any{"tool_name": "noop", "arguments_schema_id": "runecode.protocol.tools.noop.args", "arguments_schema_version": "0.1.0"}},
		"response_mode":  "text",
		"streaming_mode": "stream",
		"request_limits": map[string]any{"max_request_bytes": 262144, "max_tool_calls": 8, "max_total_tool_call_argument_bytes": 65536, "max_structured_output_bytes": 262144, "max_streamed_bytes": 16777216, "max_stream_chunk_bytes": 65536, "stream_idle_timeout_ms": 15000},
	}
}

func digestPayload(identity string) map[string]any {
	trimmed := strings.TrimPrefix(strings.TrimSpace(identity), "sha256:")
	return map[string]any{"hash_alg": "sha256", "hash": trimmed}
}

func mustDigestFromIdentityForTest(t *testing.T, identity string) trustpolicy.Digest {
	t.Helper()
	digest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimPrefix(strings.TrimSpace(identity), "sha256:")}
	if _, err := digest.Identity(); err != nil {
		t.Fatalf("invalid digest identity %q: %v", identity, err)
	}
	return digest
}

func mustDigestIdentityForTest(t *testing.T, digest trustpolicy.Digest) string {
	t.Helper()
	identity, err := digest.Identity()
	if err != nil {
		t.Fatalf("digest identity returned error: %v", err)
	}
	return identity
}

func mustCanonicalJSONBytes(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		t.Fatalf("canonicalize returned error: %v", err)
	}
	return canonical
}

func TestBindLLMRequestToArtifactsSetsLeaseID(t *testing.T) {
	s, _, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	binding, _, errResp := s.bindLLMRequestToArtifacts("req-bind", "run-llm-local-api", &requestDigest, requestObject)
	if errResp != nil {
		t.Fatalf("bindLLMRequestToArtifacts returned error response: %+v", errResp)
	}
	if got := binding.LeaseID; got != llmLeaseIDUnavailableSentinel {
		t.Fatalf("lease_id = %q, want %q", got, llmLeaseIDUnavailableSentinel)
	}
}

func TestHandleLLMStreamRequestRejectsInFlightLimit(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	runID := "run-llm-local-api"
	input := putPayloadArtifactForLocalOpsTest(t, s, "llm input", runID, "step-input")
	requestObject := validLLMRequestPayload(input)
	requestDigest := mustDigestFromIdentityForTest(t, mustPutCanonicalLLMRequestArtifact(t, s, runID, requestObject))
	release, err := s.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	defer release()
	_, _, _, errResp := s.HandleLLMStreamRequest(
		context.Background(),
		LLMStreamRequest{SchemaID: "runecode.protocol.v0.LLMStreamRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-stream-limit", RunID: runID, StreamID: "llm-stream-limit", LLMRequest: requestObject, RequestDigest: &requestDigest, Follow: false},
		RequestContext{ClientID: "client-a", LaneID: "lane-a"},
	)
	if errResp == nil {
		t.Fatal("expected in-flight limit rejection")
	}
	if errResp.Error.Code != "broker_limit_in_flight_exceeded" {
		t.Fatalf("error code = %q, want broker_limit_in_flight_exceeded", errResp.Error.Code)
	}
}

func TestEnsureInputArtifactsExistRejectsEmptyRunBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref, err := s.Put(artifacts.PutRequest{Payload: []byte("input"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "workspace", TrustedSource: false})
	if err != nil {
		t.Fatalf("Put input artifact returned error: %v", err)
	}
	_, errResp := s.ensureInputArtifactsExist("req-llm-inputs", "run-llm-local-api", []artifacts.ArtifactReference{{Digest: ref.Digest}})
	if errResp == nil {
		t.Fatal("expected run binding mismatch error for empty run_id artifact")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
	if !strings.Contains(errResp.Error.Message, "run binding mismatch") {
		t.Fatalf("error message = %q, want run binding mismatch", errResp.Error.Message)
	}
}

func mustPutCanonicalLLMRequestArtifact(t *testing.T, s *Service, runID string, requestObject map[string]any) string {
	t.Helper()
	llmRequestRaw := mustCanonicalJSONBytes(t, requestObject)
	ref, err := s.Put(artifacts.PutRequest{Payload: llmRequestRaw, ContentType: "application/json", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "broker", TrustedSource: true, RunID: runID, StepID: "step-llm"})
	if err != nil {
		t.Fatalf("Put canonical LLMRequest artifact returned error: %v", err)
	}
	return ref.Digest
}
