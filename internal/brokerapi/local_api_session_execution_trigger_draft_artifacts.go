package brokerapi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

var nonDraftIdentityRunePattern = regexp.MustCompile(`[^a-z0-9._-]+`)

func (s *Service) materializeSessionExecutionDraftArtifacts(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) ([]string, error) {
	schemaID := strings.TrimSpace(authority.draftArtifactSchemaID)
	if schemaID == "" {
		return nil, nil
	}
	runID := strings.TrimSpace(authority.runID)
	if runID == "" {
		return nil, fmt.Errorf("draft artifact materialization requires run id")
	}
	promptPayload := buildSessionExecutionPromptArtifact(result, authority)
	promptRef, err := s.persistSessionExecutionPromptArtifact(runID, authority, promptPayload)
	if err != nil {
		return nil, err
	}
	draftTextPayload := buildSessionExecutionDraftTextArtifact(result, authority)
	draftTextRef, err := s.persistSessionExecutionDraftTextArtifact(runID, authority, draftTextPayload)
	if err != nil {
		return nil, err
	}
	draftPayload, draftRef, err := s.persistSessionExecutionTypedDraftArtifact(runID, result, authority, promptRef.Digest, draftTextRef.Digest)
	if err != nil {
		return nil, err
	}
	if err := s.validateSessionExecutionTypedDraftArtifactPayload(schemaID, draftPayload); err != nil {
		return nil, err
	}
	return uniqueSortedStrings([]string{promptRef.Digest, draftTextRef.Digest, draftRef.Digest}), nil
}

func buildSessionExecutionPromptArtifact(result artifacts.SessionExecutionTriggerAppendResult, _ sessionExecutionPlanAuthority) []byte {
	text := strings.TrimSpace(result.Trigger.UserMessageContentText)
	if text == "" {
		text = strings.TrimSpace(result.Trigger.TriggerID)
	}
	return []byte(text)
}

func buildSessionExecutionDraftTextArtifact(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) []byte {
	identity := sessionExecutionDraftIdentity(result, "")
	prompt := strings.TrimSpace(result.Trigger.UserMessageContentText)
	if prompt == "" {
		prompt = "No prompt content provided."
	}
	var b strings.Builder
	switch strings.TrimSpace(authority.workflowOperation) {
	case sessionWorkflowOperationChangeDraft:
		b.WriteString("# ")
		b.WriteString(identity)
		b.WriteString("\n\n## Summary\n")
		b.WriteString(prompt)
		b.WriteString("\n\n## Scope\n- Drafted through the broker-owned trusted execution path.\n")
	case sessionWorkflowOperationSpecDraft:
		b.WriteString("# ")
		b.WriteString(identity)
		b.WriteString("\n\n## Goal\n")
		b.WriteString(prompt)
		b.WriteString("\n\n## Notes\n- Drafted through the broker-owned trusted execution path.\n")
	default:
		b.WriteString(prompt)
	}
	return []byte(b.String())
}

func (s *Service) persistSessionExecutionTypedDraftArtifact(runID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority, sourcePromptDigest, draftArtifactDigest string) ([]byte, artifacts.ArtifactReference, error) {
	payload, err := sessionExecutionTypedDraftArtifactPayload(result, authority, sourcePromptDigest, draftArtifactDigest)
	if err != nil {
		return nil, artifacts.ArtifactReference{}, err
	}
	stepID, artifactRef, err := sessionExecutionDraftArtifactBinding(authority)
	if err != nil {
		return nil, artifacts.ArtifactReference{}, err
	}
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: artifacts.DigestBytes(payload),
		CreatedByRole:         "brokerapi",
		TrustedSource:         true,
		RunID:                 runID,
		StepID:                stepID,
	})
	if err != nil {
		return nil, artifacts.ArtifactReference{}, fmt.Errorf("persist %s artifact: %w", artifactRef, err)
	}
	return payload, ref, nil
}

func sessionExecutionTypedDraftArtifactPayload(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority, sourcePromptDigest, draftArtifactDigest string) ([]byte, error) {
	if strings.TrimSpace(sourcePromptDigest) == "" {
		return nil, fmt.Errorf("typed draft artifact requires source prompt digest")
	}
	if strings.TrimSpace(draftArtifactDigest) == "" {
		return nil, fmt.Errorf("typed draft artifact requires artifact digest")
	}
	payload := map[string]any{
		"schema_id":                     strings.TrimSpace(authority.draftArtifactSchemaID),
		"schema_version":                "0.1.0",
		"data_class":                    string(artifacts.DataClassSpecText),
		"artifact_digest":               digestIdentityObject(strings.TrimSpace(draftArtifactDigest)),
		"source_prompt_identity_digest": digestIdentityObject(strings.TrimSpace(sourcePromptDigest)),
	}
	if digest := strings.TrimSpace(result.TurnExecution.BoundValidatedProjectSubstrateDigest); digest != "" {
		payload["validated_project_substrate_digest"] = digestIdentityObject(digest)
	}
	switch strings.TrimSpace(authority.workflowOperation) {
	case sessionWorkflowOperationChangeDraft:
		payload["change_id"] = sessionExecutionDraftIdentity(result, "CHG-")
	case sessionWorkflowOperationSpecDraft:
		payload["spec_id"] = sessionExecutionDraftIdentity(result, "spec-")
	default:
		return nil, fmt.Errorf("typed draft artifact unsupported for workflow operation %q", strings.TrimSpace(authority.workflowOperation))
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal typed draft artifact payload: %w", err)
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalize typed draft artifact payload: %w", err)
	}
	return canonical, nil
}

func digestIdentityObject(digest string) map[string]any {
	trimmed := strings.TrimSpace(strings.TrimPrefix(digest, "sha256:"))
	return map[string]any{
		"hash_alg": "sha256",
		"hash":     trimmed,
	}
}

func sessionExecutionDraftTextArtifactBinding(authority sessionExecutionPlanAuthority) (string, string, error) {
	switch strings.TrimSpace(authority.workflowOperation) {
	case sessionWorkflowOperationChangeDraft:
		return "session_execution/change_draft_text", "change_draft_text", nil
	case sessionWorkflowOperationSpecDraft:
		return "session_execution/spec_draft_text", "spec_draft_text", nil
	default:
		return "", "", fmt.Errorf("draft text binding unsupported for workflow operation %q", strings.TrimSpace(authority.workflowOperation))
	}
}

func sessionExecutionPromptArtifactBinding(authority sessionExecutionPlanAuthority) (string, string, error) {
	switch strings.TrimSpace(authority.workflowOperation) {
	case sessionWorkflowOperationChangeDraft:
		return "session_execution/change_draft_prompt", "change_draft_prompt", nil
	case sessionWorkflowOperationSpecDraft:
		return "session_execution/spec_draft_prompt", "spec_draft_prompt", nil
	default:
		return "", "", fmt.Errorf("draft prompt binding unsupported for workflow operation %q", strings.TrimSpace(authority.workflowOperation))
	}
}

func sessionExecutionDraftArtifactBinding(authority sessionExecutionPlanAuthority) (string, string, error) {
	switch strings.TrimSpace(authority.workflowOperation) {
	case sessionWorkflowOperationChangeDraft:
		return "session_execution/change_draft_artifact", "change_draft_artifact", nil
	case sessionWorkflowOperationSpecDraft:
		return "session_execution/spec_draft_artifact", "spec_draft_artifact", nil
	default:
		return "", "", fmt.Errorf("draft artifact binding unsupported for workflow operation %q", strings.TrimSpace(authority.workflowOperation))
	}
}

func (s *Service) validateSessionExecutionTypedDraftArtifactPayload(schemaID string, payload []byte) error {
	var schemaPath string
	switch strings.TrimSpace(schemaID) {
	case "runecode.protocol.v0.RuneContextChangeDraftArtifact":
		schemaPath = "objects/RuneContextChangeDraftArtifact.schema.json"
	case "runecode.protocol.v0.RuneContextSpecDraftArtifact":
		schemaPath = "objects/RuneContextSpecDraftArtifact.schema.json"
	default:
		return fmt.Errorf("unsupported typed draft artifact schema %q", strings.TrimSpace(schemaID))
	}
	if err := artifacts.ValidateObjectPayloadAgainstSchema(payload, schemaPath); err != nil {
		return fmt.Errorf("validate typed draft artifact: %w", err)
	}
	return nil
}

func sessionExecutionDraftIdentity(result artifacts.SessionExecutionTriggerAppendResult, prefix string) string {
	seed := strings.TrimSpace(result.Trigger.UserMessageContentText)
	if seed == "" {
		seed = strings.TrimSpace(result.TurnExecution.TurnID)
	}
	normalized := nonDraftIdentityRunePattern.ReplaceAllString(strings.ToLower(seed), "-")
	normalized = strings.Trim(normalized, "-._")
	normalized = strings.ReplaceAll(normalized, "--", "-")
	if normalized == "" {
		normalized = sessionExecutionIdentifierToken(result.TurnExecution.TurnID)
	}
	if strings.TrimSpace(prefix) == "CHG-" {
		if len(normalized) > 96 {
			normalized = normalized[:96]
		}
		return "CHG-" + normalized
	}
	if len(normalized) > 123 {
		normalized = normalized[:123]
	}
	return prefix + normalized
}

func sessionExecutionDraftGateEvidenceDigests(result artifacts.SessionExecutionTriggerAppendResult) []string {
	digests := make([]string, 0, len(result.TurnExecution.LinkedArtifactDigests))
	for _, digest := range result.TurnExecution.LinkedArtifactDigests {
		trimmed := strings.TrimSpace(digest)
		if trimmed == "" {
			continue
		}
		digests = append(digests, trimmed)
	}
	return uniqueSortedStrings(digests)
}
