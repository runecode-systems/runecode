package brokerapi

import (
	"bytes"
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
	profile := mustCreateProviderProfileWithDirectCredential(t, s, providerFamilyOpenAICompatible, providerAdapterKindOpenAIChatCompletionsV0)
	putTrustedModelGatewayContextForRun(t, s, runID, []any{trustedModelGatewayAllowlistEntry()})
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"model.example.com": {"93.184.216.34"}}}
	input := putPayloadArtifactForLocalOpsTest(t, s, "llm input", runID, "step-input")
	llmRequestPayload := validLLMRequestPayload(input, profile.ProviderProfileID)
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

func validLLMRequestPayload(inputDigest, providerID string) map[string]any {
	return map[string]any{
		"schema_id":        "runecode.protocol.v0.LLMRequest",
		"schema_version":   "0.3.0",
		"selection_source": "signed_allowlist",
		"provider":         providerID,
		"model":            "gpt-4.1-mini",
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

func mustCreateProviderProfileWithDirectCredential(t *testing.T, s *Service, family, adapterKind string) ProviderProfile {
	t.Helper()
	profileInput := providerProfileFixture("Adapter", family, "model.example.com", "/v1")
	profileInput.AdapterKind = adapterKind
	profile, err := s.providerSubstrate.upsertProfile(profileInput)
	if err != nil {
		t.Fatalf("upsertProfile returned error: %v", err)
	}
	profile.AdapterKind = adapterKind
	profile, err = s.providerSubstrate.upsertProfile(profile)
	if err != nil {
		t.Fatalf("upsertProfile(adapter) returned error: %v", err)
	}
	secretRef := "secrets/model-providers/" + profile.ProviderProfileID + "/direct-credential"
	if _, err := s.secretsSvc.ImportSecret(secretRef, bytes.NewReader([]byte("sk-test"))); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	profile, err = s.providerSubstrate.setAuthMaterial(profile.ProviderProfileID, ProviderAuthMaterial{MaterialKind: "direct_credential", MaterialState: "present", SecretRef: secretRef})
	if err != nil {
		t.Fatalf("setAuthMaterial returned error: %v", err)
	}
	return profile
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
	if got := strings.TrimSpace(binding.LeaseID); got != llmLeaseIDUnavailableSentinel {
		t.Fatalf("lease_id = %q, want unavailable sentinel", got)
	}
}

func TestHandleLLMInvokeFailClosedDoesNotIssueProviderLease(t *testing.T) {
	s, runID, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	before := s.secretsSvc.RuntimeSnapshot()
	_, errResp := s.HandleLLMInvoke(context.Background(), LLMInvokeRequest{SchemaID: "runecode.protocol.v0.LLMInvokeRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-invoke-fail-closed", RunID: runID, LLMRequest: requestObject, RequestDigest: &requestDigest}, RequestContext{})
	assertLLMExecutionUnavailable(t, errResp)
	after := s.secretsSvc.RuntimeSnapshot()
	if after.LeaseIssueCount != before.LeaseIssueCount {
		t.Fatalf("lease_issue_count = %d, want %d", after.LeaseIssueCount, before.LeaseIssueCount)
	}
	if after.ActiveLeaseCount != before.ActiveLeaseCount {
		t.Fatalf("active_lease_count = %d, want %d", after.ActiveLeaseCount, before.ActiveLeaseCount)
	}
}

func TestHandleLLMStreamRequestRejectsInFlightLimit(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	runID := "run-llm-local-api"
	input := putPayloadArtifactForLocalOpsTest(t, s, "llm input", runID, "step-input")
	profile := mustCreateProviderProfileWithDirectCredential(t, s, providerFamilyOpenAICompatible, providerAdapterKindOpenAIChatCompletionsV0)
	requestObject := validLLMRequestPayload(input, profile.ProviderProfileID)
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

func TestHandleLLMInvokeRejectsUnknownProviderProfileBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-llm-local-api"
	putTrustedModelGatewayContextForRun(t, s, runID, []any{trustedModelGatewayAllowlistEntry()})
	input := putPayloadArtifactForLocalOpsTest(t, s, "llm input", runID, "step-input")
	requestObject := validLLMRequestPayload(input, "provider-profile-missing")
	requestDigest := mustDigestFromIdentityForTest(t, mustPutCanonicalLLMRequestArtifact(t, s, runID, requestObject))
	_, errResp := s.HandleLLMInvoke(context.Background(), LLMInvokeRequest{SchemaID: "runecode.protocol.v0.LLMInvokeRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-invoke-missing-provider", RunID: runID, LLMRequest: requestObject, RequestDigest: &requestDigest}, RequestContext{})
	if errResp == nil {
		t.Fatal("expected unknown provider profile rejection")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestBindLLMRequestToArtifactsRejectsModelOutsideManualAllowlist(t *testing.T) {
	s, runID, requestObject, _, _ := prepareCanonicalLLMInvokeTest(t)
	requestObject["model"] = "gpt-4.1"
	requestDigest := mustDigestFromIdentityForTest(t, mustPutCanonicalLLMRequestArtifact(t, s, runID, requestObject))
	_, _, errResp := s.bindLLMRequestToArtifacts("req-llm-bind-disallow-model", runID, &requestDigest, requestObject)
	if errResp == nil {
		t.Fatal("expected allowlist rejection for non-allowlisted model")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
	if !strings.Contains(errResp.Error.Message, "not allowlisted") {
		t.Fatalf("error message = %q, want not allowlisted", errResp.Error.Message)
	}
}

func TestBindLLMRequestToArtifactsAllowsAllowlistedModelWhenCompatibilityProbeIsAdvisory(t *testing.T) {
	s, runID, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	providerID, _ := requestObject["provider"].(string)
	profile, ok := s.providerProfileByID(providerID)
	if !ok {
		t.Fatalf("provider_profile_id %q missing", providerID)
	}
	profile.CompatibilityPosture = "incompatible"
	profile.ReadinessPosture = ProviderReadinessPosture{
		ConfigurationState: "configured",
		CredentialState:    "present",
		ConnectivityState:  "reachable",
		CompatibilityState: "incompatible",
		EffectiveReadiness: "not_ready",
		ReasonCodes:        []string{"probe_incompatible"},
	}
	if _, err := s.providerSubstrate.upsertProfile(profile); err != nil {
		t.Fatalf("upsertProfile(incompatible posture) error: %v", err)
	}
	if _, _, errResp := s.bindLLMRequestToArtifacts("req-llm-bind-advisory-probe", runID, &requestDigest, requestObject); errResp != nil {
		t.Fatalf("bindLLMRequestToArtifacts returned error response: %+v", errResp)
	}
}

func TestBindLLMRequestToArtifactsRejectsDiscoveredModelWhenManualAllowlistDoesNotIncludeIt(t *testing.T) {
	s, runID, requestObject, _, _ := prepareCanonicalLLMInvokeTest(t)
	providerID, _ := requestObject["provider"].(string)
	profile, ok := s.providerProfileByID(providerID)
	if !ok {
		t.Fatalf("provider_profile_id %q missing", providerID)
	}
	profile.ModelCatalogPosture.DiscoveredModelIDs = []string{"gpt-4.1", "gpt-4.1-mini"}
	profile.ModelCatalogPosture.ProbeCompatibleModelIDs = []string{"gpt-4.1"}
	if _, err := s.providerSubstrate.upsertProfile(profile); err != nil {
		t.Fatalf("upsertProfile(catalog advisory) error: %v", err)
	}
	requestObject["model"] = "gpt-4.1"
	requestDigest := mustDigestFromIdentityForTest(t, mustPutCanonicalLLMRequestArtifact(t, s, runID, requestObject))
	_, _, errResp := s.bindLLMRequestToArtifacts("req-llm-bind-advisory-discovery", runID, &requestDigest, requestObject)
	if errResp == nil {
		t.Fatal("expected allowlist rejection even when discovery/probe report model")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
	if !strings.Contains(errResp.Error.Message, "not allowlisted") {
		t.Fatalf("error message = %q, want not allowlisted", errResp.Error.Message)
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
