package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type approvedImplementationResolvedInput struct {
	inputSetArtifactDigest   string
	inputSetDigest           string
	approvedInputDigests     []string
	workspaceMutationDigests []string
	metadataMutationDigests  []string
	resolvedWorkspaceWrites  []approvedImplementationWorkspaceWrite
	resolvedMetadataWrites   []approvedImplementationWorkspaceWrite
	projectDigest            string
	projectSnapshotDigest    string
	controlInputDigest       string
	repoIdentityDigest       string
	repoStateIdentityDigest  string
	workflowDefinitionHash   string
	processDefinitionHash    string
}

type approvedImplementationWorkspaceWrite struct {
	sourceDigest        string
	targetRelativePath  string
	targetAbsolutePath  string
	writeMode           string
	content             []byte
	contentDigest       string
	isLifecycleMetadata bool
	actionHash          string
	approvalID          string
	stepID              string
	policyDecisionHash  string
}

func (s *Service) applySessionExecutionApprovedImplementation(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) ([]string, []string, error) {
	resolved, err := s.resolveApprovedImplementationInput(result, authority)
	if err != nil {
		return nil, nil, err
	}
	prepared, err := prepareApprovedImplementationResolvedWrites(resolved)
	if err != nil {
		return nil, nil, err
	}
	var approvalIDs []string
	artifactDigests := approvedImplementationArtifactDigests(resolved)
	if err := finalizeBrokerOwnedMutationWrites(prepared, func() error {
		var finalizeErr error
		approvalIDs, finalizeErr = s.recordApprovedImplementationMutationApprovals(result, authority, &resolved)
		if finalizeErr != nil {
			return finalizeErr
		}
		if err := s.appendApprovedImplementationAuditEvent(result, authority, resolved, approvalIDs, artifactDigests); err != nil {
			return fmt.Errorf("append approved implementation audit event: %w", err)
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}
	return approvalIDs, artifactDigests, nil
}

func (s *Service) resolveApprovedImplementationInput(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) (approvedImplementationResolvedInput, error) {
	binding, err := resolveSingleApprovedImplementationBinding(result.TurnExecution.WorkflowRouting.BoundInputArtifacts)
	if err != nil {
		return approvedImplementationResolvedInput{}, err
	}
	repoRoot, err := approvedImplementationRepositoryRoot(s)
	if err != nil {
		return approvedImplementationResolvedInput{}, err
	}
	inputSet, err := s.loadApprovedImplementationInputSet(result.Trigger.TriggerID, binding.ArtifactDigest)
	if err != nil {
		return approvedImplementationResolvedInput{}, err
	}
	digests, err := resolveApprovedImplementationDigests(inputSet.decoded)
	if err != nil {
		return approvedImplementationResolvedInput{}, err
	}
	writes, err := s.resolveApprovedImplementationWriteGroups(repoRoot, authority, digests.workspaceMutationDigests, digests.metadataMutationDigests)
	if err != nil {
		return approvedImplementationResolvedInput{}, err
	}
	if err := validateApprovedImplementationWriteAvailability(writes.workspaceWrites, writes.metadataWrites); err != nil {
		return approvedImplementationResolvedInput{}, err
	}
	if err := validateApprovedImplementationMutationMembership(digests.approvedInputDigests, digests.workspaceMutationDigests, digests.metadataMutationDigests); err != nil {
		return approvedImplementationResolvedInput{}, err
	}
	return buildApprovedImplementationResolvedInput(inputSet, digests, writes, inputSet.decoded), nil
}
