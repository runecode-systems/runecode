package brokerapi

import (
	"fmt"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/secretsd"
)

func (s *Service) issueProviderExecutionLease(runID string, profile ProviderProfile) (string, error) {
	if s == nil || s.secretsSvc == nil {
		return "", fmt.Errorf("provider credential lease issue unavailable")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return "", fmt.Errorf("run_id is required")
	}
	authMode := strings.TrimSpace(profile.CurrentAuthMode)
	material := profile.AuthMaterial
	if authMode != "direct_credential" || strings.TrimSpace(material.MaterialKind) != "direct_credential" {
		return "", fmt.Errorf("provider auth mode %q unavailable for direct-credential execution", authMode)
	}
	if strings.TrimSpace(material.MaterialState) != "present" || strings.TrimSpace(material.SecretRef) == "" {
		return "", fmt.Errorf("provider profile is missing direct credential material")
	}
	lease, err := s.secretsSvc.IssueLease(secretsd.IssueLeaseRequest{
		SecretRef:    material.SecretRef,
		ConsumerID:   "principal:gateway:model:" + runID,
		RoleKind:     "model-gateway",
		Scope:        "run:" + runID,
		DeliveryKind: "model_gateway",
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(lease.LeaseID), nil
}

func (s *Service) readArtifactPrompt(ref artifacts.ArtifactReference) (string, error) {
	if !isAdapterTextContentType(ref.ContentType) {
		return "", fmt.Errorf("llm_request input_artifacts must use text-compatible content types")
	}
	reader, err := s.Get(ref.Digest)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	bytes, err := io.ReadAll(io.LimitReader(reader, maxTranslatedPromptBytes))
	if err != nil {
		return "", err
	}
	prompt := strings.TrimSpace(string(bytes))
	if prompt == "" {
		return "", fmt.Errorf("llm_request input_artifacts must include non-empty prompt content")
	}
	return prompt, nil
}

func isAdapterTextContentType(contentType string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(trimmed, "text/") || trimmed == "application/json"
}

func (openAIChatCompletionsAdapter) translate(view canonicalLLMRequestView, prompt string) (map[string]any, error) {
	payload := map[string]any{
		"model":  strings.TrimSpace(view.Model),
		"stream": strings.TrimSpace(view.StreamingMode) == "stream",
		"messages": []any{
			map[string]any{"role": "user", "content": prompt},
		},
	}
	tools := openAIToolsFromAllowlist(view.ToolAllowlist)
	if len(tools) > 0 {
		payload["tools"] = tools
	}
	return payload, nil
}

func openAIToolsFromAllowlist(entries []canonicalLLMToolAllowed) []any {
	if len(entries) == 0 {
		return nil
	}
	out := make([]any, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.ToolName)
		if name == "" {
			continue
		}
		function := map[string]any{"name": name}
		if desc := strings.TrimSpace(entry.ToolDescription); desc != "" {
			function["description"] = desc
		}
		out = append(out, map[string]any{"type": "function", "function": function})
	}
	return out
}

func (anthropicMessagesAdapter) translate(view canonicalLLMRequestView, prompt string) (map[string]any, error) {
	payload := map[string]any{
		"model":      strings.TrimSpace(view.Model),
		"stream":     strings.TrimSpace(view.StreamingMode) == "stream",
		"max_tokens": 1024,
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": prompt}}},
		},
	}
	tools := anthropicToolsFromAllowlist(view.ToolAllowlist)
	if len(tools) > 0 {
		payload["tools"] = tools
	}
	return payload, nil
}

func anthropicToolsFromAllowlist(entries []canonicalLLMToolAllowed) []any {
	if len(entries) == 0 {
		return nil
	}
	out := make([]any, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.ToolName)
		if name == "" {
			continue
		}
		schemaID := strings.TrimSpace(entry.ArgumentsSchemaID)
		schemaVersion := strings.TrimSpace(entry.ArgumentsSchemaVersion)
		inputSchema := map[string]any{"type": "object", "additionalProperties": true}
		if schemaID != "" && schemaVersion != "" {
			inputSchema["x-runecode-arguments-schema"] = map[string]any{"schema_id": schemaID, "schema_version": schemaVersion}
		}
		tool := map[string]any{"name": name, "input_schema": inputSchema}
		if desc := strings.TrimSpace(entry.ToolDescription); desc != "" {
			tool["description"] = desc
		}
		out = append(out, tool)
	}
	return out
}
