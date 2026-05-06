package brokerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/perffixtures"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (r *phase5GatewayPerfRig) seedProviderAndLLMRequest() error {
	beginResp, err := r.beginProviderSetupSession("req-phase5-provider-begin", "model.example.com")
	if err != nil {
		return err
	}
	r.providerProfileID = beginResp.Profile.ProviderProfileID
	r.llmRequest = phase5GatewayLLMRequest(r.providerProfileID)
	if err := r.submitIngressForSession(beginResp.SetupSession.SetupSessionID, "seed"); err != nil {
		return err
	}
	if err := r.commitProviderValidation(); err != nil {
		return err
	}
	if err := r.putGatewayPromptArtifact(); err != nil {
		return err
	}
	return r.putLLMRequestArtifact()
}

func (r *phase5GatewayPerfRig) beginProviderSetupSession(requestID, canonicalHost string) (ProviderSetupSessionBeginResponse, error) {
	beginResp, errResp := r.service.HandleProviderSetupSessionBegin(context.Background(), ProviderSetupSessionBeginRequest{
		SchemaID:            "runecode.protocol.v0.ProviderSetupSessionBeginRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           requestID,
		DisplayLabel:        "Phase5 Gateway",
		ProviderFamily:      providerFamilyOpenAICompatible,
		AdapterKind:         providerAdapterKindOpenAIChatCompletionsV0,
		CanonicalHost:       canonicalHost,
		CanonicalPathPrefix: "/v1",
		AllowlistedModelIDs: []string{"gpt-4.1-mini"},
	}, RequestContext{})
	if err := phase5DependencyErr("provider setup begin", errResp); err != nil {
		return ProviderSetupSessionBeginResponse{}, err
	}
	return beginResp, nil
}

func (r *phase5GatewayPerfRig) commitProviderValidation() error {
	validationBegin, validationErr := r.service.HandleProviderValidationBegin(context.Background(), ProviderValidationBeginRequest{
		SchemaID:          "runecode.protocol.v0.ProviderValidationBeginRequest",
		SchemaVersion:     "0.1.0",
		RequestID:         "req-phase5-provider-validation-begin",
		ProviderProfileID: r.providerProfileID,
	}, RequestContext{})
	if err := phase5DependencyErr("provider validation begin", validationErr); err != nil {
		return err
	}
	_, validationCommitErr := r.service.HandleProviderValidationCommit(context.Background(), ProviderValidationCommitRequest{
		SchemaID:            "runecode.protocol.v0.ProviderValidationCommitRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-phase5-provider-validation-commit",
		ProviderProfileID:   r.providerProfileID,
		ValidationAttemptID: validationBegin.ValidationAttemptID,
		ConnectivityState:   "reachable",
		CompatibilityState:  "compatible",
	}, RequestContext{})
	return phase5DependencyErr("provider validation commit", validationCommitErr)
}

func (r *phase5GatewayPerfRig) putGatewayPromptArtifact() error {
	inputRef, err := r.service.Put(artifacts.PutRequest{
		Payload:               []byte("phase5 gateway prompt"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("c", 64),
		CreatedByRole:         "broker",
		TrustedSource:         true,
		RunID:                 r.runID,
		StepID:                "phase5-gateway-input",
	})
	if err != nil {
		return err
	}
	requestArtifacts, ok := r.llmRequest["input_artifacts"].([]any)
	if !ok || len(requestArtifacts) == 0 {
		return fmt.Errorf("phase5 gateway request missing input_artifacts")
	}
	artifact, ok := requestArtifacts[0].(map[string]any)
	if !ok {
		return fmt.Errorf("phase5 gateway request input_artifact malformed")
	}
	artifact["digest"] = phase5DigestObject(inputRef.Digest)
	artifact["size_bytes"] = len("phase5 gateway prompt")
	return nil
}

func phase5GatewayLLMRequest(providerProfileID string) map[string]any {
	return map[string]any{
		"schema_id":        "runecode.protocol.v0.LLMRequest",
		"schema_version":   "0.3.0",
		"selection_source": "signed_allowlist",
		"provider":         providerProfileID,
		"model":            "gpt-4.1-mini",
		"input_artifacts": []any{map[string]any{
			"schema_id":      "runecode.protocol.v0.ArtifactReference",
			"schema_version": "0.4.0",
			"digest":         map[string]any{},
			"size_bytes":     0,
			"content_type":   "text/plain",
			"data_class":     "spec_text",
			"provenance_receipt_hash": map[string]any{
				"hash_alg": "sha256",
				"hash":     strings.Repeat("d", 64),
			},
		}},
		"tool_allowlist": []any{map[string]any{
			"tool_name":                "noop",
			"arguments_schema_id":      "runecode.protocol.tools.noop.args",
			"arguments_schema_version": "0.1.0",
		}},
		"response_mode":  "text",
		"streaming_mode": "stream",
		"request_limits": map[string]any{
			"max_request_bytes":                  262144,
			"max_tool_calls":                     8,
			"max_total_tool_call_argument_bytes": 65536,
			"max_structured_output_bytes":        262144,
			"max_streamed_bytes":                 16777216,
			"max_stream_chunk_bytes":             65536,
			"stream_idle_timeout_ms":             15000,
		},
	}
}

func (r *phase5GatewayPerfRig) putLLMRequestArtifact() error {
	raw, err := json.Marshal(r.llmRequest)
	if err != nil {
		return err
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(canonical, &r.llmRequest); err != nil {
		return err
	}
	digest, err := canonicalDigestForValue(r.llmRequest)
	if err != nil {
		return err
	}
	r.requestDigest = digest
	_, err = r.service.Put(artifacts.PutRequest{
		Payload:               canonical,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64),
		CreatedByRole:         "broker",
		TrustedSource:         true,
		RunID:                 r.runID,
		StepID:                "phase5-gateway-request",
	})
	return err
}

func newPhase5GatewayPerfService(repoRoot string) (*Service, func(), error) {
	storeRoot, err := os.MkdirTemp("", "runecode-phase5-gateway-store-")
	if err != nil {
		return nil, nil, err
	}
	ledgerRoot, err := os.MkdirTemp("", "runecode-phase5-gateway-ledger-")
	if err != nil {
		_ = os.RemoveAll(storeRoot)
		return nil, nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(storeRoot)
		_ = os.RemoveAll(ledgerRoot)
	}
	service, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repoRoot})
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	service.gatewayRuntime.resolver = phase5GatewayStaticResolver{}
	originalClient := newLLMHTTPClient
	newLLMHTTPClient = func() llmHTTPClient {
		return &http.Client{Transport: phase5StubProviderTransport{backend: perffixtures.StubProviderBackend{}}}
	}
	return service, func() {
		newLLMHTTPClient = originalClient
		cleanup()
	}, nil
}

type phase5GatewayStaticResolver struct{}

func (phase5GatewayStaticResolver) LookupIP(_ context.Context, _ string, _ string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("93.184.216.34")}, nil
}

type phase5StubProviderTransport struct {
	backend perffixtures.StubProviderBackend
}

func (t phase5StubProviderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response := t.backend.Invoke(req.Context(), perffixtures.StubProviderRequest{Prompt: "phase5"})
	body, err := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": response.Text}}}})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body)), Request: req}, nil
}
