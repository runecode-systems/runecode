package brokerapi

import (
	"strings"
	"time"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/policyengine"
)

func draftPromotePolicyDecision(authority sessionExecutionPlanAuthority, resolved sessionDraftPromoteResolvedInput, actionHash string) policyengine.PolicyDecision {
	policyInputHashes := uniqueSortedStrings([]string{strings.TrimSpace(resolved.projectDigest)})
	relevantArtifactHashes := uniqueSortedStrings([]string{
		strings.TrimSpace(resolved.draftArtifactDigest),
		strings.TrimSpace(resolved.draftTextDigest),
	})
	return policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        policyengine.DecisionRequireHumanApproval,
		PolicyReasonCode:       "approval_required",
		ManifestHash:           strings.TrimSpace(resolved.draftArtifactDigest),
		ActionRequestHash:      actionHash,
		PolicyInputHashes:      policyInputHashes,
		RelevantArtifactHashes: relevantArtifactHashes,
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details: map[string]any{
			"precedence":       "approval_profile_moderate",
			"checkpoint_model": "workspace_write",
		},
		RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.out_of_workspace_write.v0",
		RequiredApproval: map[string]any{
			"approval_trigger_code":    "out_of_workspace_write",
			"approval_assurance_level": approvalDefaultAssuranceLevel,
			"presence_mode":            approvalDefaultPresenceMode,
			"changes_if_approved":      "Apply reviewed RuneContext draft into canonical project files.",
			"approval_ttl_seconds":     1800,
			"scope":                    draftPromoteApprovalScope(authority),
			"related_hashes": map[string]any{
				"manifest_hash":            strings.TrimSpace(resolved.draftArtifactDigest),
				"action_request_hash":      actionHash,
				"policy_input_hashes":      policyInputHashes,
				"relevant_artifact_hashes": relevantArtifactHashes,
			},
		},
	}
}

func draftPromoteApprovalScope(authority sessionExecutionPlanAuthority) map[string]any {
	return map[string]any{
		"schema_id":        "runecode.protocol.v0.ApprovalBoundScope",
		"schema_version":   "0.1.0",
		"workspace_id":     workspaceIDForRun(authority.runID),
		"run_id":           strings.TrimSpace(authority.runID),
		"stage_id":         strings.TrimSpace(authority.stageID),
		"step_id":          strings.TrimSpace(authority.stepID),
		"role_instance_id": strings.TrimSpace(authority.roleInstanceID),
		"action_kind":      policyengine.ActionKindWorkspaceWrite,
	}
}

func draftPromoteApprovalRecord(authority sessionExecutionPlanAuthority, resolved sessionDraftPromoteResolvedInput, approvalID, actionHash, decisionHash string, now time.Time) artifacts.ApprovalRecord {
	return artifacts.ApprovalRecord{
		ApprovalID:             strings.TrimSpace(approvalID),
		Status:                 "consumed",
		WorkspaceID:            workspaceIDForRun(authority.runID),
		RunID:                  strings.TrimSpace(authority.runID),
		StageID:                strings.TrimSpace(authority.stageID),
		StepID:                 strings.TrimSpace(authority.stepID),
		RoleInstanceID:         strings.TrimSpace(authority.roleInstanceID),
		ActionKind:             policyengine.ActionKindWorkspaceWrite,
		RequestedAt:            now,
		DecidedAt:              &now,
		ConsumedAt:             &now,
		ApprovalTriggerCode:    "out_of_workspace_write",
		ChangesIfApproved:      "Apply reviewed RuneContext draft into canonical project files.",
		ApprovalAssuranceLevel: approvalDefaultAssuranceLevel,
		PresenceMode:           approvalDefaultPresenceMode,
		PolicyDecisionHash:     strings.TrimSpace(decisionHash),
		ManifestHash:           strings.TrimSpace(resolved.draftArtifactDigest),
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: uniqueSortedStrings([]string{resolved.draftArtifactDigest, resolved.draftTextDigest}),
		RequestDigest:          strings.TrimSpace(approvalID),
		DecisionDigest:         strings.TrimSpace(decisionHash),
		SourceDigest:           strings.TrimSpace(resolved.draftArtifactDigest),
	}
}
