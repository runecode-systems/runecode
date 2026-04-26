package brokerapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestHandleLLMInvokeExecutesOpenAICompatible(t *testing.T) {
	var outboundCalls int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&outboundCalls, 1)
		if got := r.URL.Path; got != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", got)
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
			t.Fatalf("authorization header missing bearer token: %q", got)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello from openai"}}]}`))
	}))
	defer server.Close()

	s, runID, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	configureProviderEndpointForServer(t, s, requestObject, server.URL, server.Client(), providerFamilyOpenAICompatible, providerAdapterKindOpenAIChatCompletionsV0)

	resp, errResp := s.HandleLLMInvoke(context.Background(), LLMInvokeRequest{SchemaID: "runecode.protocol.v0.LLMInvokeRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-invoke", RunID: runID, LLMRequest: requestObject, RequestDigest: &requestDigest}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleLLMInvoke returned error response: %+v", errResp)
	}
	if atomic.LoadInt32(&outboundCalls) != 1 {
		t.Fatalf("outbound calls = %d, want 1", outboundCalls)
	}
	if resp.RequestID != "req-llm-invoke" {
		t.Fatalf("request_id = %q, want req-llm-invoke", resp.RequestID)
	}
}

func TestHandleLLMInvokeExecutesAnthropicCompatible(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/messages" {
			t.Fatalf("path = %q, want /v1/messages", got)
		}
		if got := r.Header.Get("x-api-key"); got == "" {
			t.Fatal("x-api-key missing")
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Fatalf("anthropic-version = %q, want 2023-06-01", got)
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"hello from anthropic"}]}`))
	}))
	defer server.Close()

	s, runID, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	configureProviderEndpointForServer(t, s, requestObject, server.URL, server.Client(), providerFamilyAnthropicCompatible, providerAdapterKindAnthropicMessagesV0)

	resp, errResp := s.HandleLLMInvoke(context.Background(), LLMInvokeRequest{SchemaID: "runecode.protocol.v0.LLMInvokeRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-invoke-anthropic", RunID: runID, LLMRequest: requestObject, RequestDigest: &requestDigest}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleLLMInvoke returned error response: %+v", errResp)
	}
	if resp.RequestID != "req-llm-invoke-anthropic" {
		t.Fatalf("request_id = %q, want req-llm-invoke-anthropic", resp.RequestID)
	}
}

func TestHandleLLMInvokeRejectsProviderNotReady(t *testing.T) {
	s, runID, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	providerID, _ := requestObject["provider"].(string)
	profile, ok := s.providerProfileByID(providerID)
	if !ok {
		t.Fatalf("provider %q not found", providerID)
	}
	profile.ReadinessPosture.EffectiveReadiness = "not_ready"
	if _, _, err := s.providerSubstrate.upsertProfile(profile); err != nil {
		t.Fatalf("upsertProfile returned error: %v", err)
	}

	_, errResp := s.HandleLLMInvoke(context.Background(), LLMInvokeRequest{SchemaID: "runecode.protocol.v0.LLMInvokeRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-not-ready", RunID: runID, LLMRequest: requestObject, RequestDigest: &requestDigest}, RequestContext{})
	if errResp == nil {
		t.Fatal("expected not-ready rejection")
	}
	if errResp.Error.Code != "broker_validation_operation_invalid" {
		t.Fatalf("error code = %q, want broker_validation_operation_invalid", errResp.Error.Code)
	}
}

func TestHandleLLMStreamRequestAndStreamLLMEventsLifecycle(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"stream final text"}}]}`))
	}))
	defer server.Close()

	s, runID, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	configureProviderEndpointForServer(t, s, requestObject, server.URL, server.Client(), providerFamilyOpenAICompatible, providerAdapterKindOpenAIChatCompletionsV0)

	var released int32
	ack, binding, opaque, errResp := s.HandleLLMStreamRequest(context.Background(), LLMStreamRequest{SchemaID: "runecode.protocol.v0.LLMStreamRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-stream", RunID: runID, StreamID: "stream-1", LLMRequest: requestObject, RequestDigest: &requestDigest}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleLLMStreamRequest returned error response: %+v", errResp)
	}
	origRelease := ack.Release
	ack.Release = func() {
		atomic.AddInt32(&released, 1)
		if origRelease != nil {
			origRelease()
		}
	}
	envelope, err := s.StreamLLMEvents(ack, binding, opaque)
	if err != nil {
		t.Fatalf("StreamLLMEvents returned error: %v", err)
	}
	if len(envelope.Events) < 2 {
		t.Fatalf("events len = %d, want >=2", len(envelope.Events))
	}
	if atomic.LoadInt32(&released) != 1 {
		t.Fatalf("release count = %d, want 1", released)
	}
}

func TestHandleLLMInvokeAdmissionUsesResolvedDestinationRef(t *testing.T) {
	var outboundCalls int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&outboundCalls, 1)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"should-not-run"}}]}`))
	}))
	defer server.Close()

	s, runID, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	configureProviderEndpointForServer(t, s, requestObject, server.URL, server.Client(), providerFamilyOpenAICompatible, providerAdapterKindOpenAIChatCompletionsV0)

	providerID, _ := requestObject["provider"].(string)
	profile, ok := s.providerProfileByID(providerID)
	if !ok {
		t.Fatalf("provider %q not found", providerID)
	}
	profile.DestinationIdentity.CanonicalHost = "forbidden.example.com"
	if _, _, err := s.providerSubstrate.upsertProfile(profile); err != nil {
		t.Fatalf("upsert profile returned error: %v", err)
	}

	_, errResp := s.HandleLLMInvoke(context.Background(), LLMInvokeRequest{SchemaID: "runecode.protocol.v0.LLMInvokeRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-invoke-destination-mismatch", RunID: runID, LLMRequest: requestObject, RequestDigest: &requestDigest}, RequestContext{})
	if errResp == nil {
		t.Fatal("expected policy rejection")
	}
	if errResp.Error.Code != "broker_limit_policy_rejected" {
		t.Fatalf("error code = %q, want broker_limit_policy_rejected", errResp.Error.Code)
	}
	if atomic.LoadInt32(&outboundCalls) != 0 {
		t.Fatalf("outbound calls = %d, want 0", outboundCalls)
	}
}

func TestHandleLLMStreamRequestOpaqueStateDoesNotSerializeHeaders(t *testing.T) {
	s, runID, requestObject, requestDigest, _ := prepareCanonicalLLMInvokeTest(t)
	ack, _, opaque, errResp := s.HandleLLMStreamRequest(context.Background(), LLMStreamRequest{SchemaID: "runecode.protocol.v0.LLMStreamRequest", SchemaVersion: "0.1.0", RequestID: "req-llm-stream-opaque", RunID: runID, StreamID: "stream-opaque", LLMRequest: requestObject, RequestDigest: &requestDigest}, RequestContext{})
	if ack.Release != nil {
		ack.Release()
	}
	if ack.Cancel != nil {
		ack.Cancel()
	}
	if errResp != nil {
		t.Fatalf("HandleLLMStreamRequest returned error response: %+v", errResp)
	}
	if strings.Contains(opaque.Digest, "sk-test") {
		t.Fatal("opaque stream state leaked provider credential")
	}
	state := map[string]any{}
	if err := json.Unmarshal([]byte(opaque.Digest), &state); err != nil {
		t.Fatalf("Unmarshal opaque stream state returned error: %v", err)
	}
	if _, ok := state["headers"]; ok {
		t.Fatal("opaque stream state must not include headers")
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

func configureProviderEndpointForServer(t *testing.T, s *Service, requestObject map[string]any, serverURL string, _ llmHTTPClient, family, adapterKind string) {
	t.Helper()
	u := mustParseURLForLLMTest(t, serverURL)
	providerID, _ := requestObject["provider"].(string)
	profile, ok := s.providerProfileByID(providerID)
	if !ok {
		t.Fatalf("provider %q not found", providerID)
	}
	profile.ProviderFamily = family
	profile.AdapterKind = adapterKind
	profile.DestinationIdentity.CanonicalHost = "model.example.com"
	profile.DestinationIdentity.CanonicalPathPrefix = "/v1"
	profile.DestinationIdentity.TLSRequired = true
	profile.DestinationIdentity.PrivateRangeBlocking = "enforced"
	profile.DestinationIdentity.DNSRebindingProtection = "enforced"
	profile.ReadinessPosture.EffectiveReadiness = "ready"
	if _, _, err := s.providerSubstrate.upsertProfile(profile); err != nil {
		t.Fatalf("upsert profile returned error: %v", err)
	}
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"model.example.com": {"93.184.216.34"}}}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			dialer := &net.Dialer{}
			return dialer.DialContext(ctx, network, u.Host)
		},
	}
	newLLMHTTPClient = func() llmHTTPClient { return &http.Client{Transport: transport} }
	t.Cleanup(func() {
		newLLMHTTPClient = func() llmHTTPClient { return &http.Client{Timeout: llmHTTPTimeout} }
	})
}

func mustParseURLForLLMTest(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse server url returned error: %v", err)
	}
	return u
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
			"schema_version": "0.4.0",
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
	profile, _, err := s.providerSubstrate.upsertProfile(profileInput)
	if err != nil {
		t.Fatalf("upsertProfile returned error: %v", err)
	}
	profile.AdapterKind = adapterKind
	profile.ReadinessPosture.EffectiveReadiness = "ready"
	profile.ReadinessPosture.ConnectivityState = "reachable"
	profile.ReadinessPosture.CompatibilityState = "compatible"
	profile, _, err = s.providerSubstrate.upsertProfile(profile)
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

func mustPutCanonicalLLMRequestArtifact(t *testing.T, s *Service, runID string, requestObject map[string]any) string {
	t.Helper()
	llmRequestRaw := mustCanonicalJSONBytes(t, requestObject)
	ref, err := s.Put(artifacts.PutRequest{Payload: llmRequestRaw, ContentType: "application/json", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "broker", TrustedSource: true, RunID: runID, StepID: "step-llm"})
	if err != nil {
		t.Fatalf("Put canonical LLMRequest artifact returned error: %v", err)
	}
	return ref.Digest
}
