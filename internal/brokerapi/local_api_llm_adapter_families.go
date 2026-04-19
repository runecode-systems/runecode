package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

const (
	providerAdapterKindOpenAIChatCompletionsV0 = "chat_completions_v0"
	providerAdapterKindAnthropicMessagesV0     = "messages_v0"
	providerAdapterKindOpenAIResponsesV0       = "openai_responses_v0"

	providerFamilyOpenAICompatible    = "openai_compatible"
	providerFamilyAnthropicCompatible = "anthropic_compatible"

	maxTranslatedPromptBytes = 1 << 20
)

type canonicalLLMRequestView struct {
	Provider      string                    `json:"provider"`
	Model         string                    `json:"model"`
	ResponseMode  string                    `json:"response_mode"`
	StreamingMode string                    `json:"streaming_mode"`
	ToolAllowlist []canonicalLLMToolAllowed `json:"tool_allowlist"`
}

type canonicalLLMToolAllowed struct {
	ToolName               string `json:"tool_name"`
	ArgumentsSchemaID      string `json:"arguments_schema_id"`
	ArgumentsSchemaVersion string `json:"arguments_schema_version"`
	ToolDescription        string `json:"tool_description"`
}

type llmProviderAdapter interface {
	translate(canonicalLLMRequestView, string) (map[string]any, error)
}

type openAIChatCompletionsAdapter struct{}
type anthropicMessagesAdapter struct{}

func (s *Service) translateCanonicalLLMRequestForProfile(requestID string, binding llmExecutionBinding, llmReq any, inputRef artifacts.ArtifactReference) (map[string]any, *ErrorResponse) {
	profile, ok := s.providerProfileByID(binding.ProviderID)
	if !ok {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider profile unavailable for llm request")
		return nil, &errOut
	}
	adapter, err := adapterForProviderProfile(profile)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return nil, &errOut
	}
	prompt, err := s.readArtifactPrompt(inputRef)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return nil, &errOut
	}
	view, err := decodeCanonicalLLMRequestView(llmReq)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return nil, &errOut
	}
	translated, err := adapter.translate(view, prompt)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return nil, &errOut
	}
	return translated, nil
}

func (s *Service) providerProfileForLLMRequest(llmReq any) (ProviderProfile, error) {
	view, err := decodeCanonicalLLMRequestView(llmReq)
	if err != nil {
		return ProviderProfile{}, err
	}
	modelID := strings.TrimSpace(view.Model)
	if modelID == "" {
		return ProviderProfile{}, fmt.Errorf("llm_request model is required")
	}
	providerID := strings.TrimSpace(view.Provider)
	if providerID == "" {
		return ProviderProfile{}, fmt.Errorf("llm_request provider is required")
	}
	profile, ok := s.providerProfileByID(providerID)
	if !ok {
		return ProviderProfile{}, fmt.Errorf("llm_request provider must reference a configured provider_profile_id")
	}
	if profile.ProviderProfileID != providerID {
		return ProviderProfile{}, fmt.Errorf("llm_request provider binding mismatch")
	}
	if !isModelIDAllowlisted(profile.AllowlistedModelIDs, modelID) {
		return ProviderProfile{}, fmt.Errorf("llm_request model %q is not allowlisted for provider_profile_id %q", modelID, providerID)
	}
	return profile, nil
}

func isModelIDAllowlisted(allowlisted []string, modelID string) bool {
	trimmedModel := strings.TrimSpace(modelID)
	if trimmedModel == "" {
		return false
	}
	for _, allowlistedModel := range allowlisted {
		if strings.TrimSpace(allowlistedModel) == trimmedModel {
			return true
		}
	}
	return false
}

func adapterForProviderProfile(profile ProviderProfile) (llmProviderAdapter, error) {
	family := strings.TrimSpace(profile.ProviderFamily)
	kind := strings.TrimSpace(profile.AdapterKind)
	switch family {
	case providerFamilyOpenAICompatible:
		switch kind {
		case providerAdapterKindOpenAIChatCompletionsV0:
			return openAIChatCompletionsAdapter{}, nil
		case providerAdapterKindOpenAIResponsesV0:
			return nil, fmt.Errorf("adapter_kind %q reserved for additive future support", kind)
		default:
			return nil, fmt.Errorf("provider_family %q requires adapter_kind %q", family, providerAdapterKindOpenAIChatCompletionsV0)
		}
	case providerFamilyAnthropicCompatible:
		if kind != providerAdapterKindAnthropicMessagesV0 {
			return nil, fmt.Errorf("provider_family %q requires adapter_kind %q", family, providerAdapterKindAnthropicMessagesV0)
		}
		return anthropicMessagesAdapter{}, nil
	default:
		return nil, fmt.Errorf("provider_family %q not supported", family)
	}
}

func decodeCanonicalLLMRequestView(llmReq any) (canonicalLLMRequestView, error) {
	raw, err := json.Marshal(llmReq)
	if err != nil {
		return canonicalLLMRequestView{}, fmt.Errorf("decode llm_request failed: %w", err)
	}
	view := canonicalLLMRequestView{}
	if err := json.Unmarshal(raw, &view); err != nil {
		return canonicalLLMRequestView{}, fmt.Errorf("decode llm_request failed: %w", err)
	}
	if strings.TrimSpace(view.Model) == "" {
		return canonicalLLMRequestView{}, fmt.Errorf("llm_request model is required")
	}
	streamingMode := strings.TrimSpace(view.StreamingMode)
	if streamingMode != "stream" && streamingMode != "final_only" {
		return canonicalLLMRequestView{}, fmt.Errorf("llm_request streaming_mode is invalid")
	}
	responseMode := strings.TrimSpace(view.ResponseMode)
	if responseMode != "text" && responseMode != "structured_output" {
		return canonicalLLMRequestView{}, fmt.Errorf("llm_request response_mode is invalid")
	}
	return view, nil
}
