package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/policyengine"
)

func (s *Service) appendApprovedImplementationAuditEvent(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority, resolved approvedImplementationResolvedInput, approvalIDs, artifactDigests []string) error {
	return s.AppendTrustedAuditEvent("runecontext_approved_implementation_applied", "brokerapi", map[string]any{
		"run_id":                     strings.TrimSpace(authority.runID),
		"plan_id":                    strings.TrimSpace(authority.planID),
		"workflow_operation":         strings.TrimSpace(authority.workflowOperation),
		"input_set_artifact_digest":  strings.TrimSpace(resolved.inputSetArtifactDigest),
		"input_set_digest":           strings.TrimSpace(resolved.inputSetDigest),
		"approved_input_digests":     append([]string{}, resolved.approvedInputDigests...),
		"workspace_mutation_digests": append([]string{}, resolved.workspaceMutationDigests...),
		"metadata_mutation_digests":  append([]string{}, resolved.metadataMutationDigests...),
		"workspace_write_count":      len(resolved.resolvedWorkspaceWrites),
		"metadata_write_count":       len(resolved.resolvedMetadataWrites),
		"project_digest":             strings.TrimSpace(resolved.projectDigest),
		"project_snapshot_digest":    strings.TrimSpace(resolved.projectSnapshotDigest),
		"control_input_digest":       strings.TrimSpace(resolved.controlInputDigest),
		"repo_identity_digest":       strings.TrimSpace(resolved.repoIdentityDigest),
		"repo_state_identity_digest": strings.TrimSpace(resolved.repoStateIdentityDigest),
		"workflow_definition_hash":   strings.TrimSpace(resolved.workflowDefinitionHash),
		"process_definition_hash":    strings.TrimSpace(resolved.processDefinitionHash),
		"approval_ids":               append([]string{}, approvalIDs...),
		"mutation_artifact_digests":  append([]string{}, artifactDigests...),
		"trigger_id":                 strings.TrimSpace(result.Trigger.TriggerID),
		"turn_id":                    strings.TrimSpace(result.TurnExecution.TurnID),
	})
}

func (s *Service) recordApprovedImplementationMutationApprovals(_ artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority, resolved *approvedImplementationResolvedInput) ([]string, error) {
	allWrites := approvedImplementationAllWrites(resolved)
	approvalIDs := make([]string, 0, len(allWrites))
	for idx := range allWrites {
		write := &allWrites[idx]
		if err := s.recordAndMirrorApprovedImplementationApproval(authority, *resolved, write, resolved); err != nil {
			return nil, err
		}
		approvalIDs = append(approvalIDs, write.approvalID)
	}
	return uniqueSortedStrings(approvalIDs), nil
}

func approvedImplementationAllWrites(resolved *approvedImplementationResolvedInput) []approvedImplementationWorkspaceWrite {
	allWrites := append([]approvedImplementationWorkspaceWrite{}, resolved.resolvedWorkspaceWrites...)
	allWrites = append(allWrites, resolved.resolvedMetadataWrites...)
	return allWrites
}

func (s *Service) recordAndMirrorApprovedImplementationApproval(authority sessionExecutionPlanAuthority, resolved approvedImplementationResolvedInput, write *approvedImplementationWorkspaceWrite, fullResolved *approvedImplementationResolvedInput) error {
	decisionHash, err := s.recordApprovedImplementationPolicyDecision(authority, resolved, *write)
	if err != nil {
		return err
	}
	write.policyDecisionHash = decisionHash
	if err := s.recordApprovedImplementationApproval(authority, *write); err != nil {
		return err
	}
	reflectApprovedImplementationDecisionHash(fullResolved, *write, decisionHash)
	return nil
}

func reflectApprovedImplementationDecisionHash(resolved *approvedImplementationResolvedInput, write approvedImplementationWorkspaceWrite, decisionHash string) {
	target := &resolved.resolvedWorkspaceWrites
	if write.isLifecycleMetadata {
		target = &resolved.resolvedMetadataWrites
	}
	for i := range *target {
		if (*target)[i].sourceDigest == write.sourceDigest {
			(*target)[i].policyDecisionHash = decisionHash
			return
		}
	}
}

func (s *Service) recordApprovedImplementationPolicyDecision(authority sessionExecutionPlanAuthority, resolved approvedImplementationResolvedInput, write approvedImplementationWorkspaceWrite) (string, error) {
	decision := approvedImplementationPolicyDecision(authority, resolved, write)
	priorRefs := stringSetFromSlice(s.PolicyDecisionRefsForRun(strings.TrimSpace(authority.runID)))
	if err := s.RecordPolicyDecision(strings.TrimSpace(authority.runID), "", decision); err != nil {
		return "", fmt.Errorf("record approved implementation policy decision: %w", err)
	}
	decisionHash, err := recordedPolicyDecisionHashForRunAndAction(s, strings.TrimSpace(authority.runID), strings.TrimSpace(write.actionHash), priorRefs)
	if err != nil {
		return "", fmt.Errorf("locate approved implementation policy decision hash: %w", err)
	}
	return decisionHash, nil
}

func approvedImplementationPolicyDecision(authority sessionExecutionPlanAuthority, resolved approvedImplementationResolvedInput, write approvedImplementationWorkspaceWrite) policyengine.PolicyDecision {
	policyInputHashes := approvedImplementationPolicyInputHashes(resolved)
	relevantArtifactHashes := uniqueSortedStrings([]string{strings.TrimSpace(write.sourceDigest)})
	return policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        policyengine.DecisionRequireHumanApproval,
		PolicyReasonCode:       "approval_required",
		ManifestHash:           strings.TrimSpace(write.sourceDigest),
		ActionRequestHash:      strings.TrimSpace(write.actionHash),
		PolicyInputHashes:      policyInputHashes,
		RelevantArtifactHashes: relevantArtifactHashes,
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details: map[string]any{
			"precedence":         "approval_profile_moderate",
			"checkpoint_model":   "workspace_write",
			"workflow_operation": strings.TrimSpace(authority.workflowOperation),
		},
		RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.out_of_workspace_write.v0",
		RequiredApproval: map[string]any{
			"approval_trigger_code":    "out_of_workspace_write",
			"approval_assurance_level": approvalDefaultAssuranceLevel,
			"presence_mode":            approvalDefaultPresenceMode,
			"changes_if_approved":      approvedImplementationChangesIfApproved(write),
			"approval_ttl_seconds":     1800,
			"scope":                    approvedImplementationApprovalScope(authority, write),
			"related_hashes": map[string]any{
				"manifest_hash":            strings.TrimSpace(write.sourceDigest),
				"action_request_hash":      strings.TrimSpace(write.actionHash),
				"policy_input_hashes":      policyInputHashes,
				"relevant_artifact_hashes": relevantArtifactHashes,
			},
		},
	}
}

func approvedImplementationPolicyInputHashes(resolved approvedImplementationResolvedInput) []string {
	return uniqueSortedStrings([]string{
		strings.TrimSpace(resolved.projectDigest),
		strings.TrimSpace(resolved.repoStateIdentityDigest),
		strings.TrimSpace(resolved.controlInputDigest),
	})
}

func approvedImplementationApprovalScope(authority sessionExecutionPlanAuthority, write approvedImplementationWorkspaceWrite) map[string]any {
	return map[string]any{
		"schema_id":        "runecode.protocol.v0.ApprovalBoundScope",
		"schema_version":   "0.1.0",
		"workspace_id":     workspaceIDForRun(authority.runID),
		"run_id":           strings.TrimSpace(authority.runID),
		"stage_id":         strings.TrimSpace(authority.stageID),
		"step_id":          strings.TrimSpace(write.stepID),
		"role_instance_id": strings.TrimSpace(authority.roleInstanceID),
		"action_kind":      policyengine.ActionKindWorkspaceWrite,
	}
}

func (s *Service) recordApprovedImplementationApproval(authority sessionExecutionPlanAuthority, write approvedImplementationWorkspaceWrite) error {
	now := s.currentTimestamp()
	record := artifacts.ApprovalRecord{
		ApprovalID:             strings.TrimSpace(write.approvalID),
		Status:                 "consumed",
		WorkspaceID:            workspaceIDForRun(authority.runID),
		RunID:                  strings.TrimSpace(authority.runID),
		StageID:                strings.TrimSpace(authority.stageID),
		StepID:                 strings.TrimSpace(write.stepID),
		RoleInstanceID:         strings.TrimSpace(authority.roleInstanceID),
		ActionKind:             policyengine.ActionKindWorkspaceWrite,
		RequestedAt:            now,
		DecidedAt:              &now,
		ConsumedAt:             &now,
		ApprovalTriggerCode:    "out_of_workspace_write",
		ChangesIfApproved:      approvedImplementationChangesIfApproved(write),
		ApprovalAssuranceLevel: approvalDefaultAssuranceLevel,
		PresenceMode:           approvalDefaultPresenceMode,
		PolicyDecisionHash:     strings.TrimSpace(write.policyDecisionHash),
		ManifestHash:           strings.TrimSpace(write.sourceDigest),
		ActionRequestHash:      strings.TrimSpace(write.actionHash),
		RelevantArtifactHashes: []string{strings.TrimSpace(write.sourceDigest)},
		RequestDigest:          strings.TrimSpace(write.approvalID),
		DecisionDigest:         strings.TrimSpace(write.policyDecisionHash),
		SourceDigest:           strings.TrimSpace(write.sourceDigest),
	}
	if err := s.RecordApproval(record); err != nil {
		return fmt.Errorf("record approved implementation approval: %w", err)
	}
	return nil
}

func approvedImplementationChangesIfApproved(write approvedImplementationWorkspaceWrite) string {
	kind := "approved implementation file"
	if write.isLifecycleMetadata {
		kind = "approved RuneContext lifecycle metadata"
	}
	return fmt.Sprintf("Apply %s to %s.", kind, strings.TrimSpace(write.targetRelativePath))
}
