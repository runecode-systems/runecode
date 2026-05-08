package brokerapi

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func (s *Service) resolveSessionDraftPromoteApplyInput(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) (sessionDraftPromoteResolvedInput, error) {
	binding, err := resolveSingleDraftPromoteBinding(result.TurnExecution.WorkflowRouting.BoundInputArtifacts)
	if err != nil {
		return sessionDraftPromoteResolvedInput{}, err
	}
	repoRoot, err := draftPromoteRepositoryRoot(s)
	if err != nil {
		return sessionDraftPromoteResolvedInput{}, err
	}
	decoded, err := s.loadDraftPromoteDecodedArtifact(binding)
	if err != nil {
		return sessionDraftPromoteResolvedInput{}, err
	}
	draftTextDigest, draftText, err := s.loadDraftPromoteText(decoded, binding.ArtifactDigest)
	if err != nil {
		return sessionDraftPromoteResolvedInput{}, err
	}
	draftIdentity, targetRelativePath, err := sessionDraftPromoteIdentityAndPath(binding.ArtifactRef, decoded)
	if err != nil {
		return sessionDraftPromoteResolvedInput{}, err
	}
	targetRelativePath, projectDigest, targetAbsolutePath, err := s.resolveDraftPromoteTarget(binding.ArtifactRef, authority, decoded, result.TurnExecution.BoundValidatedProjectSubstrateDigest, repoRoot)
	if err != nil {
		return sessionDraftPromoteResolvedInput{}, err
	}
	return sessionDraftPromoteResolvedInput{
		draftArtifactDigest: strings.TrimSpace(binding.ArtifactDigest),
		draftTextDigest:     strings.TrimSpace(draftTextDigest),
		draftSchemaID:       strings.TrimSpace(draftPromoteStringValueFromMap(decoded, "schema_id")),
		draftIdentity:       strings.TrimSpace(draftIdentity),
		draftText:           append([]byte(nil), draftText...),
		targetRelativePath:  targetRelativePath,
		targetAbsolutePath:  targetAbsolutePath,
		appliedFileDigest:   artifacts.DigestBytes(draftText),
		sourcePromptDigest:  draftPromoteDigestObjectValueFromMap(decoded, "source_prompt_identity_digest"),
		projectDigest:       projectDigest,
	}, nil
}

func (s *Service) resolveDraftPromoteTarget(artifactRef string, authority sessionExecutionPlanAuthority, decoded map[string]any, boundDigest, repoRoot string) (string, string, string, error) {
	_, targetRelativePath, err := sessionDraftPromoteIdentityAndPath(artifactRef, decoded)
	if err != nil {
		return "", "", "", err
	}
	if err := validateDraftPromoteTargetPath(authority, targetRelativePath); err != nil {
		return "", "", "", err
	}
	projectDigest := draftPromoteDigestObjectValueFromMap(decoded, "validated_project_substrate_digest")
	if err := validateDraftPromoteProjectDigest(projectDigest, boundDigest); err != nil {
		return "", "", "", err
	}
	targetAbsolutePath, err := brokerOwnedDraftPromoteTargetPath(repoRoot, targetRelativePath)
	if err != nil {
		return "", "", "", err
	}
	return targetRelativePath, projectDigest, targetAbsolutePath, nil
}

func draftPromoteRepositoryRoot(s *Service) (string, error) {
	repoRoot := strings.TrimSpace(s.projectSubstrate.RepositoryRoot)
	if repoRoot == "" {
		repoRoot = strings.TrimSpace(s.apiConfig.RepositoryRoot)
	}
	if repoRoot == "" {
		return "", fmt.Errorf("repository root is required for draft promote/apply")
	}
	return repoRoot, nil
}

func (s *Service) loadDraftPromoteDecodedArtifact(binding artifacts.SessionWorkflowPackBoundInputArtifactDurableState) (map[string]any, error) {
	payload, err := s.readArtifactPayloadVerified(binding.ArtifactDigest)
	if err != nil {
		return nil, fmt.Errorf("read bound draft artifact %q: %w", binding.ArtifactDigest, err)
	}
	if err := validateDraftPromoteArtifactPayload(binding.ArtifactRef, payload); err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode bound draft artifact %q: %w", binding.ArtifactDigest, err)
	}
	return decoded, nil
}

func (s *Service) loadDraftPromoteText(decoded map[string]any, artifactDigest string) (string, []byte, error) {
	draftTextDigest := draftPromoteDigestObjectValueFromMap(decoded, "artifact_digest")
	if strings.TrimSpace(draftTextDigest) == "" {
		return "", nil, fmt.Errorf("bound draft artifact %q missing artifact_digest", artifactDigest)
	}
	draftText, err := s.readArtifactPayloadVerified(draftTextDigest)
	if err != nil {
		return "", nil, fmt.Errorf("read draft text artifact %q: %w", draftTextDigest, err)
	}
	return draftTextDigest, draftText, nil
}

func validateDraftPromoteProjectDigest(projectDigest, boundDigest string) error {
	if strings.TrimSpace(boundDigest) != "" && strings.TrimSpace(projectDigest) == "" {
		return fmt.Errorf("draft promote/apply artifact missing validated_project_substrate_digest")
	}
	if strings.TrimSpace(projectDigest) == "" {
		return nil
	}
	if strings.TrimSpace(projectDigest) != strings.TrimSpace(boundDigest) {
		return fmt.Errorf("draft promote/apply validated_project_substrate_digest drift detected")
	}
	return nil
}

func validateDraftPromoteArtifactPayload(artifactRef string, payload []byte) error {
	schemaPath, err := draftPromoteSchemaPathForArtifactRef(artifactRef)
	if err != nil {
		return err
	}
	if err := artifacts.ValidateObjectPayloadAgainstSchema(payload, schemaPath); err != nil {
		return fmt.Errorf("validate draft promote/apply input artifact: %w", err)
	}
	return nil
}

func draftPromoteSchemaPathForArtifactRef(artifactRef string) (string, error) {
	switch strings.TrimSpace(artifactRef) {
	case "change_draft_artifact":
		return "objects/RuneContextChangeDraftArtifact.schema.json", nil
	case "spec_draft_artifact":
		return "objects/RuneContextSpecDraftArtifact.schema.json", nil
	default:
		return "", fmt.Errorf("draft promote/apply bound artifact ref %q is unsupported", strings.TrimSpace(artifactRef))
	}
}

func resolveSingleDraftPromoteBinding(bindings []artifacts.SessionWorkflowPackBoundInputArtifactDurableState) (artifacts.SessionWorkflowPackBoundInputArtifactDurableState, error) {
	if len(bindings) != 1 {
		return artifacts.SessionWorkflowPackBoundInputArtifactDurableState{}, fmt.Errorf("draft promote/apply requires exactly one bound draft artifact")
	}
	binding := bindings[0]
	if strings.TrimSpace(binding.ArtifactDigest) == "" {
		return artifacts.SessionWorkflowPackBoundInputArtifactDurableState{}, fmt.Errorf("draft promote/apply bound draft artifact digest is required")
	}
	if _, err := draftPromoteSchemaPathForArtifactRef(binding.ArtifactRef); err != nil {
		return artifacts.SessionWorkflowPackBoundInputArtifactDurableState{}, err
	}
	return binding, nil
}

func sessionDraftPromoteIdentityAndPath(artifactRef string, decoded map[string]any) (string, string, error) {
	switch strings.TrimSpace(artifactRef) {
	case "change_draft_artifact":
		return sessionDraftPromoteChangeIdentityAndPath(decoded)
	case "spec_draft_artifact":
		return sessionDraftPromoteSpecIdentityAndPath(decoded)
	default:
		return "", "", fmt.Errorf("draft promote/apply bound artifact ref %q is unsupported", strings.TrimSpace(artifactRef))
	}
}

func sessionDraftPromoteChangeIdentityAndPath(decoded map[string]any) (string, string, error) {
	changeID := strings.TrimSpace(draftPromoteStringValueFromMap(decoded, "change_id"))
	if changeID == "" {
		return "", "", fmt.Errorf("change draft artifact missing change_id")
	}
	return changeID, filepath.ToSlash(filepath.Join(projectsubstrate.CanonicalChangesPath, changeID, "proposal.md")), nil
}

func sessionDraftPromoteSpecIdentityAndPath(decoded map[string]any) (string, string, error) {
	specID := strings.TrimSpace(draftPromoteStringValueFromMap(decoded, "spec_id"))
	if specID == "" {
		return "", "", fmt.Errorf("spec draft artifact missing spec_id")
	}
	return specID, filepath.ToSlash(filepath.Join(projectsubstrate.CanonicalSpecsPath, specID+".md")), nil
}

func validateDraftPromoteTargetPath(authority sessionExecutionPlanAuthority, targetRelativePath string) error {
	entry, err := builtInCatalogEntryForWorkflowOperation(authority.workflowOperation)
	if err != nil {
		return err
	}
	target, err := normalizeBrokerOwnedRelativeTargetPath(targetRelativePath)
	if err != nil {
		return err
	}
	if target == "" {
		return fmt.Errorf("draft promote/apply target path is required")
	}
	for _, allowed := range entry.WritableRuneContextPath {
		prefix, err := normalizeBrokerOwnedRelativeTargetPath(allowed)
		if err != nil {
			return err
		}
		if pathWithinAllowedPrefix(target, prefix) {
			return nil
		}
	}
	return fmt.Errorf("draft promote/apply target path %q is outside writable RuneContext scope", target)
}

func draftPromoteDigestObjectValueFromMap(in map[string]any, key string) string {
	raw, ok := in[key]
	if !ok {
		return ""
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	hash, _ := value["hash"].(string)
	if strings.TrimSpace(hash) == "" {
		return ""
	}
	return "sha256:" + strings.TrimSpace(hash)
}

func draftPromoteStringValueFromMap(in map[string]any, key string) string {
	raw, ok := in[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return value
}
