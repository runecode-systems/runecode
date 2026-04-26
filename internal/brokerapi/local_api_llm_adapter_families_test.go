package brokerapi

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestTranslateCanonicalLLMRequestForProfileOpenAIChatCompletions(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-llm-adapter-openai"
	profile := mustCreateProviderProfileWithDirectCredential(t, s, providerFamilyOpenAICompatible, providerAdapterKindOpenAIChatCompletionsV0)
	inputDigest := putPayloadArtifactForLocalOpsTest(t, s, "hello openai", runID, "step-input")
	requestObject := validLLMRequestPayload(inputDigest, profile.ProviderProfileID)
	requestDigest := mustDigestFromIdentityForTest(t, mustPutCanonicalLLMRequestArtifact(t, s, runID, requestObject))
	binding, inputRef, errResp := s.bindLLMRequestToArtifacts("req-adapter-openai", runID, &requestDigest, requestObject)
	if errResp != nil {
		t.Fatalf("bindLLMRequestToArtifacts returned error response: %+v", errResp)
	}
	payload, errResp := s.translateCanonicalLLMRequestForProfile("req-adapter-openai", binding, requestObject, inputRef)
	if errResp != nil {
		t.Fatalf("translateCanonicalLLMRequestForProfile returned error response: %+v", errResp)
	}
	if got := payload["model"]; got != "gpt-4.1-mini" {
		t.Fatalf("payload.model = %v, want gpt-4.1-mini", got)
	}
	if got, _ := payload["stream"].(bool); !got {
		t.Fatalf("payload.stream = %v, want true", payload["stream"])
	}
	messages, _ := payload["messages"].([]any)
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	first, _ := messages[0].(map[string]any)
	if got := first["content"]; got != "hello openai" {
		t.Fatalf("messages[0].content = %v, want hello openai", got)
	}
}

func TestTranslateCanonicalLLMRequestForProfileAnthropicMessages(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-llm-adapter-anthropic"
	profile := mustCreateProviderProfileWithDirectCredential(t, s, providerFamilyAnthropicCompatible, providerAdapterKindAnthropicMessagesV0)
	inputDigest := putPayloadArtifactForLocalOpsTest(t, s, "hello anthropic", runID, "step-input")
	requestObject := validLLMRequestPayload(inputDigest, profile.ProviderProfileID)
	requestObject["streaming_mode"] = "final_only"
	requestDigest := mustDigestFromIdentityForTest(t, mustPutCanonicalLLMRequestArtifact(t, s, runID, requestObject))
	binding, inputRef, errResp := s.bindLLMRequestToArtifacts("req-adapter-anthropic", runID, &requestDigest, requestObject)
	if errResp != nil {
		t.Fatalf("bindLLMRequestToArtifacts returned error response: %+v", errResp)
	}
	payload, errResp := s.translateCanonicalLLMRequestForProfile("req-adapter-anthropic", binding, requestObject, inputRef)
	if errResp != nil {
		t.Fatalf("translateCanonicalLLMRequestForProfile returned error response: %+v", errResp)
	}
	if got := payload["model"]; got != "gpt-4.1-mini" {
		t.Fatalf("payload.model = %v, want gpt-4.1-mini", got)
	}
	if got, _ := payload["stream"].(bool); got {
		t.Fatalf("payload.stream = %v, want false", payload["stream"])
	}
	messages, _ := payload["messages"].([]any)
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	first, _ := messages[0].(map[string]any)
	content, _ := first["content"].([]any)
	part, _ := content[0].(map[string]any)
	if got := part["text"]; got != "hello anthropic" {
		t.Fatalf("messages[0].content[0].text = %v, want hello anthropic", got)
	}
}

func TestTranslateCanonicalLLMRequestForProfileRejectsNonTextInputArtifact(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-llm-adapter-nontext"
	profile := mustCreateProviderProfileWithDirectCredential(t, s, providerFamilyOpenAICompatible, providerAdapterKindOpenAIChatCompletionsV0)
	inputRef, err := s.Put(artifacts.PutRequest{Payload: []byte("\x00\x01\x02"), ContentType: "application/octet-stream", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "workspace", TrustedSource: false, RunID: runID, StepID: "step-input"})
	if err != nil {
		t.Fatalf("Put binary input artifact returned error: %v", err)
	}
	requestObject := validLLMRequestPayload(inputRef.Digest, profile.ProviderProfileID)
	requestObject["input_artifacts"] = []any{map[string]any{
		"schema_id":               "runecode.protocol.v0.ArtifactReference",
		"schema_version":          "0.4.0",
		"digest":                  digestPayload(inputRef.Digest),
		"size_bytes":              3,
		"content_type":            "application/octet-stream",
		"data_class":              "spec_text",
		"provenance_receipt_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
	}}
	requestDigest := mustDigestFromIdentityForTest(t, mustPutCanonicalLLMRequestArtifact(t, s, runID, requestObject))
	binding, primaryInput, errResp := s.bindLLMRequestToArtifacts("req-adapter-nontext", runID, &requestDigest, requestObject)
	if errResp != nil {
		t.Fatalf("bindLLMRequestToArtifacts returned error response: %+v", errResp)
	}
	_, errResp = s.translateCanonicalLLMRequestForProfile("req-adapter-nontext", binding, requestObject, primaryInput)
	if errResp == nil {
		t.Fatal("expected non-text input artifact rejection")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}
