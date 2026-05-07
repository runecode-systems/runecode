package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type approvedImplementationDigestSet struct {
	approvedInputDigests     []string
	workspaceMutationDigests []string
	metadataMutationDigests  []string
}

type approvedImplementationWriteSet struct {
	workspaceWrites []approvedImplementationWorkspaceWrite
	metadataWrites  []approvedImplementationWorkspaceWrite
}

func approvedImplementationRepositoryRoot(s *Service) (string, error) {
	repoRoot := strings.TrimSpace(s.projectSubstrate.RepositoryRoot)
	if repoRoot == "" {
		repoRoot = strings.TrimSpace(s.apiConfig.RepositoryRoot)
	}
	if repoRoot == "" {
		return "", fmt.Errorf("repository root is required for approved implementation")
	}
	return repoRoot, nil
}

func (s *Service) loadApprovedImplementationInputSet(triggerID, inputSetArtifactDigest string) (approvedImplementationInputSetState, error) {
	inputSet, errResp := s.decodeApprovedImplementationInputSet(triggerID, inputSetArtifactDigest)
	if errResp != nil {
		return approvedImplementationInputSetState{}, fmt.Errorf("%s", strings.TrimSpace(errResp.Error.Message))
	}
	return inputSet, nil
}

func resolveApprovedImplementationDigests(decoded map[string]any) (approvedImplementationDigestSet, error) {
	approvedDigests, err := approvedImplementationDigestList(decoded, "approved_input_digests")
	if err != nil {
		return approvedImplementationDigestSet{}, err
	}
	workspaceDigests, err := approvedImplementationDigestList(decoded, "workspace_mutation_digests")
	if err != nil {
		return approvedImplementationDigestSet{}, err
	}
	metadataDigests, err := approvedImplementationDigestList(decoded, "lifecycle_metadata_mutation_digests")
	if err != nil {
		return approvedImplementationDigestSet{}, err
	}
	return approvedImplementationDigestSet{
		approvedInputDigests:     approvedDigests,
		workspaceMutationDigests: workspaceDigests,
		metadataMutationDigests:  metadataDigests,
	}, nil
}

func (s *Service) resolveApprovedImplementationWriteGroups(repoRoot string, authority sessionExecutionPlanAuthority, workspaceDigests, metadataDigests []string) (approvedImplementationWriteSet, error) {
	workspaceWrites, err := s.resolveApprovedImplementationWrites(repoRoot, authority, workspaceDigests, false)
	if err != nil {
		return approvedImplementationWriteSet{}, err
	}
	metadataWrites, err := s.resolveApprovedImplementationWrites(repoRoot, authority, metadataDigests, true)
	if err != nil {
		return approvedImplementationWriteSet{}, err
	}
	return approvedImplementationWriteSet{workspaceWrites: workspaceWrites, metadataWrites: metadataWrites}, nil
}

func validateApprovedImplementationWriteAvailability(workspaceWrites, metadataWrites []approvedImplementationWorkspaceWrite) error {
	if len(workspaceWrites) == 0 && len(metadataWrites) == 0 {
		return fmt.Errorf("implementation_input_set must bind at least one approved workspace or lifecycle metadata mutation")
	}
	return nil
}

func validateApprovedImplementationMutationMembership(approvedDigests, workspaceDigests, metadataDigests []string) error {
	allowed := map[string]struct{}{}
	for _, digest := range approvedDigests {
		allowed[strings.TrimSpace(digest)] = struct{}{}
	}
	for _, digest := range append(append([]string{}, workspaceDigests...), metadataDigests...) {
		trimmed := strings.TrimSpace(digest)
		if trimmed == "" {
			continue
		}
		if _, ok := allowed[trimmed]; !ok {
			return fmt.Errorf("implementation_input_set mutation digest %q is not included in approved_input_digests", trimmed)
		}
	}
	return nil
}

func buildApprovedImplementationResolvedInput(inputSet approvedImplementationInputSetState, digests approvedImplementationDigestSet, writes approvedImplementationWriteSet, decoded map[string]any) approvedImplementationResolvedInput {
	projectDigest, projectSnapshotDigest, controlInputDigest, repoIdentityDigest, repoStateDigest, workflowHash, processHash := approvedImplementationContextDigests(decoded)
	return approvedImplementationResolvedInput{
		inputSetArtifactDigest:   strings.TrimSpace(inputSet.inputSetArtifactDigest),
		inputSetDigest:           strings.TrimSpace(inputSet.inputSetDigest),
		approvedInputDigests:     digests.approvedInputDigests,
		workspaceMutationDigests: digests.workspaceMutationDigests,
		metadataMutationDigests:  digests.metadataMutationDigests,
		resolvedWorkspaceWrites:  writes.workspaceWrites,
		resolvedMetadataWrites:   writes.metadataWrites,
		projectDigest:            projectDigest,
		projectSnapshotDigest:    projectSnapshotDigest,
		controlInputDigest:       controlInputDigest,
		repoIdentityDigest:       repoIdentityDigest,
		repoStateIdentityDigest:  repoStateDigest,
		workflowDefinitionHash:   workflowHash,
		processDefinitionHash:    processHash,
	}
}

func approvedImplementationContextDigests(decoded map[string]any) (string, string, string, string, string, string, string) {
	projectDigest, _ := digestIdentityFromApprovedImplementationField(decoded, "validated_project_substrate_digest")
	projectSnapshotDigest, _ := digestIdentityFromApprovedImplementationField(decoded, "project_substrate_snapshot_digest")
	controlInputDigest, _ := digestIdentityFromApprovedImplementationField(decoded, "control_input_digest")
	repoIdentityDigest, _ := digestIdentityFromApprovedImplementationField(decoded, "repo_identity_digest")
	repoStateDigest, _ := digestIdentityFromApprovedImplementationField(decoded, "repo_state_identity_digest")
	workflowHash, _ := digestIdentityFromApprovedImplementationField(decoded, "workflow_definition_hash")
	processHash, _ := digestIdentityFromApprovedImplementationField(decoded, "process_definition_hash")
	return strings.TrimSpace(projectDigest), strings.TrimSpace(projectSnapshotDigest), strings.TrimSpace(controlInputDigest), strings.TrimSpace(repoIdentityDigest), strings.TrimSpace(repoStateDigest), strings.TrimSpace(workflowHash), strings.TrimSpace(processHash)
}

func resolveSingleApprovedImplementationBinding(bindings []artifacts.SessionWorkflowPackBoundInputArtifactDurableState) (artifacts.SessionWorkflowPackBoundInputArtifactDurableState, error) {
	if len(bindings) != 1 {
		return artifacts.SessionWorkflowPackBoundInputArtifactDurableState{}, fmt.Errorf("approved implementation requires exactly one bound implementation input set")
	}
	binding := bindings[0]
	if strings.TrimSpace(binding.ArtifactRef) != "implementation_input_set" {
		return artifacts.SessionWorkflowPackBoundInputArtifactDurableState{}, fmt.Errorf("approved implementation bound artifact ref %q is unsupported", strings.TrimSpace(binding.ArtifactRef))
	}
	if strings.TrimSpace(binding.ArtifactDigest) == "" {
		return artifacts.SessionWorkflowPackBoundInputArtifactDurableState{}, fmt.Errorf("approved implementation bound implementation input set digest is required")
	}
	return binding, nil
}

func approvedImplementationDigestList(decoded map[string]any, field string) ([]string, error) {
	raw, ok := decoded[field]
	if !ok {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("implementation_input_set %s must be an array", field)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		typed, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("implementation_input_set %s contains malformed digest object", field)
		}
		identity, err := digestIdentityFromApprovedImplementationValue(typed)
		if err != nil {
			return nil, fmt.Errorf("implementation_input_set %s contains invalid digest: %w", field, err)
		}
		out = append(out, identity)
	}
	return uniqueSortedStrings(out), nil
}

func (s *Service) resolveApprovedImplementationWrites(repoRoot string, authority sessionExecutionPlanAuthority, digests []string, lifecycleMetadata bool) ([]approvedImplementationWorkspaceWrite, error) {
	if len(digests) == 0 {
		return nil, nil
	}
	out := make([]approvedImplementationWorkspaceWrite, 0, len(digests))
	for _, digest := range digests {
		write, err := s.resolveApprovedImplementationWrite(repoRoot, authority, digest, lifecycleMetadata)
		if err != nil {
			return nil, err
		}
		out = append(out, write)
	}
	return out, nil
}

func (s *Service) resolveApprovedImplementationWrite(repoRoot string, authority sessionExecutionPlanAuthority, digest string, lifecycleMetadata bool) (approvedImplementationWorkspaceWrite, error) {
	payload, err := s.readArtifactPayloadVerified(digest)
	if err != nil {
		return approvedImplementationWorkspaceWrite{}, fmt.Errorf("read approved implementation artifact %q: %w", digest, err)
	}
	targetRelativePath, writeMode, content, err := approvedImplementationWriteIntent(payload)
	if err != nil {
		return approvedImplementationWorkspaceWrite{}, err
	}
	if err := validateApprovedImplementationTargetPath(authority, targetRelativePath, lifecycleMetadata); err != nil {
		return approvedImplementationWorkspaceWrite{}, err
	}
	targetAbsolutePath, err := brokerOwnedDraftPromoteTargetPath(repoRoot, targetRelativePath)
	if err != nil {
		return approvedImplementationWorkspaceWrite{}, err
	}
	contentDigest := artifacts.DigestBytes(content)
	stepID := approvedImplementationMutationStepID(targetRelativePath, lifecycleMetadata)
	return approvedImplementationWorkspaceWrite{
		sourceDigest:        strings.TrimSpace(digest),
		targetRelativePath:  strings.TrimSpace(targetRelativePath),
		targetAbsolutePath:  targetAbsolutePath,
		writeMode:           writeMode,
		content:             append([]byte(nil), content...),
		contentDigest:       contentDigest,
		isLifecycleMetadata: lifecycleMetadata,
		actionHash:          approvedImplementationActionHash(targetRelativePath, contentDigest, digest),
		approvalID:          approvedImplementationApprovalID(targetRelativePath, digest),
		stepID:              stepID,
	}, nil
}

func writeApprovedImplementationResolvedWrites(resolved approvedImplementationResolvedInput) (func() error, error) {
	writes := append([]approvedImplementationWorkspaceWrite{}, resolved.resolvedWorkspaceWrites...)
	writes = append(writes, resolved.resolvedMetadataWrites...)
	snapshots, err := captureApprovedImplementationWriteSnapshots(writes)
	if err != nil {
		return nil, err
	}
	if err := writeApprovedImplementationWrites(writes); err != nil {
		return nil, joinBrokerOwnedRollbackError(err, rollbackBrokerOwnedFileSnapshots(snapshots))
	}
	return func() error { return rollbackBrokerOwnedFileSnapshots(snapshots) }, nil
}

func captureApprovedImplementationWriteSnapshots(writes []approvedImplementationWorkspaceWrite) ([]brokerOwnedFileSnapshot, error) {
	snapshots := make([]brokerOwnedFileSnapshot, 0, len(writes))
	for _, write := range writes {
		snapshot, err := captureBrokerOwnedFileSnapshot(write.targetAbsolutePath)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func writeApprovedImplementationWrites(writes []approvedImplementationWorkspaceWrite) error {
	for idx := range writes {
		write := &writes[idx]
		if err := writeApprovedImplementationFile(*write); err != nil {
			return err
		}
	}
	return nil
}

func approvedImplementationArtifactDigests(resolved approvedImplementationResolvedInput) []string {
	return uniqueSortedStrings(append([]string{strings.TrimSpace(resolved.inputSetArtifactDigest)}, append(append([]string{}, resolved.workspaceMutationDigests...), resolved.metadataMutationDigests...)...))
}
