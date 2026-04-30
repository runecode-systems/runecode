package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func buildAuthoritativeRunState(summary RunSummary, artifactsForRun []artifacts.ArtifactRecord, pendingIDs []string, manifestHashes []string, policyRefs []string, approvals []ApprovalSummary, runtimeFacts launcherbackend.RuntimeFactsSnapshot, currentInstanceID string) map[string]any {
	receipt := runtimeFacts.LaunchReceipt.Normalized()
	evidence, _, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(runtimeFacts)
	if err != nil {
		evidence = launcherbackend.RuntimeEvidenceSnapshot{}
	}
	state := buildBaseAuthoritativeRunState(summary, len(artifactsForRun), len(pendingIDs), receipt)
	projectReceiptIdentityState(state, receipt, evidence)
	projectReceiptImageState(state, receipt)
	projectReceiptBackendEvidenceState(state, receipt)
	projectReceiptSessionAndAttachmentState(state, receipt)
	projectReceiptLifecycleAndCacheState(state, receipt)
	projectHardeningAndTerminalState(state, runtimeFacts)
	projectBackendPostureSelectionEvidenceState(state, currentInstanceID, summary.RunID, policyRefs, approvals)
	projectWorkflowDerivedState(state, summary, manifestHashes)
	return state
}

func projectBackendPostureSelectionEvidenceState(state map[string]any, instanceID string, runID string, policyRefs []string, approvals []ApprovalSummary) {
	reducedAssurance := state["backend_kind"] == launcherbackend.BackendKindContainer || state["runtime_posture_degraded"] == true
	if !reducedAssurance {
		return
	}
	evidence := map[string]any{}
	approvalEvidence := backendPostureApprovalEvidence(instanceID, runID, approvals)
	if len(policyRefs) == 0 {
		if approvalPolicyHash, ok := approvalEvidence["policy_decision_hash"].(string); ok && approvalPolicyHash != "" {
			policyRefs = []string{approvalPolicyHash}
		}
	}
	if len(policyRefs) > 0 {
		evidence["policy_decision_refs"] = append([]string{}, policyRefs...)
	}
	if len(approvalEvidence) > 0 {
		evidence["approval"] = approvalEvidence
	}
	if len(evidence) > 0 {
		state["backend_posture_selection_evidence"] = evidence
	}
}

func backendPostureApprovalEvidence(instanceID, runID string, approvals []ApprovalSummary) map[string]any {
	if instanceID == "" {
		return nil
	}
	best, ok := bestBackendPostureApproval(instanceID, runID, approvals)
	if !ok {
		return nil
	}
	approval := best
	evidence := map[string]any{"approval_id": approval.ApprovalID}
	if approval.RequestDigest != "" {
		evidence["approval_request_digest"] = approval.RequestDigest
	}
	if approval.DecisionDigest != "" {
		evidence["approval_decision_digest"] = approval.DecisionDigest
	}
	if approval.PolicyDecisionHash != "" {
		evidence["policy_decision_hash"] = approval.PolicyDecisionHash
	}
	if approval.Status != "" {
		evidence["status"] = approval.Status
	}
	return evidence
}

func bestBackendPostureApproval(instanceID, runID string, approvals []ApprovalSummary) (ApprovalSummary, bool) {
	var best ApprovalSummary
	found := false
	for _, approval := range approvals {
		if !isBackendPostureApproval(instanceID, runID, approval) {
			continue
		}
		if !found || approvalEvidencePrecedes(approval, best) {
			best = approval
			found = true
		}
	}
	return best, found
}

func isBackendPostureApproval(instanceID, runID string, approval ApprovalSummary) bool {
	if approval.BoundScope.ActionKind != policyengine.ActionKindBackendPosture {
		return false
	}
	if instanceID == "" {
		return false
	}
	if approval.BoundScope.InstanceID != instanceID {
		return false
	}
	expectedSelectorRunID := instanceControlRunIDForInstanceID(instanceID)
	if expectedSelectorRunID == "" {
		return false
	}
	boundRunID := approval.BoundScope.RunID
	if boundRunID == "" {
		return false
	}
	if strings.HasPrefix(boundRunID, "instance-control:") {
		return boundRunID == expectedSelectorRunID
	}
	if runID == "" {
		return false
	}
	return boundRunID == runID
}

func approvalEvidencePrecedes(candidate ApprovalSummary, existing ApprovalSummary) bool {
	if approvalEvidenceStatusRank(candidate.Status) != approvalEvidenceStatusRank(existing.Status) {
		return approvalEvidenceStatusRank(candidate.Status) < approvalEvidenceStatusRank(existing.Status)
	}
	if candidate.RequestedAt != existing.RequestedAt {
		return candidate.RequestedAt > existing.RequestedAt
	}
	return candidate.ApprovalID > existing.ApprovalID
}

func approvalEvidenceStatusRank(status string) int {
	switch status {
	case "consumed":
		return 0
	case "approved":
		return 1
	case "pending":
		return 2
	case "superseded":
		return 3
	case "denied":
		return 4
	case "expired":
		return 5
	case "cancelled":
		return 6
	default:
		return 7
	}
}

func buildBaseAuthoritativeRunState(summary RunSummary, artifactCount int, pendingCount int, receipt launcherbackend.BackendLaunchReceipt) map[string]any {
	return map[string]any{
		"source":                          "broker_store",
		"provenance":                      "trusted_derived",
		"status":                          summary.LifecycleState,
		"artifact_count":                  artifactCount,
		"pending_approval_count":          pendingCount,
		"workspace_id":                    summary.WorkspaceID,
		"project_context_identity_digest": strings.TrimSpace(summary.ProjectContextIdentity),
		"backend_kind":                    receipt.BackendKind,
		"isolation_assurance_level":       receipt.IsolationAssuranceLevel,
		"runtime_posture_degraded":        runtimePostureDegraded(receipt.BackendKind, receipt.IsolationAssuranceLevel),
		"provisioning_posture":            receipt.ProvisioningPosture,
		"attestation_posture":             launcherbackend.AttestationPostureUnknown,
		"runtime_facts_source":            "launcher_backend_receipt",
	}
}

func projectReceiptIdentityState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt, evidence launcherbackend.RuntimeEvidenceSnapshot) {
	projectAttestationIdentityState(state, evidence)
	if projectIdentity, ok := state["project_context_identity_digest"].(string); ok && strings.TrimSpace(projectIdentity) != "" {
		state["validated_project_substrate_identity_digest"] = strings.TrimSpace(projectIdentity)
	}
	projectRuntimeBindingIdentityState(state, receipt)
	projectProvisioningDegradedState(state, receipt)
}

func projectAttestationIdentityState(state map[string]any, evidence launcherbackend.RuntimeEvidenceSnapshot) {
	attestationPosture, attestationReasons := launcherbackend.DeriveAttestationPostureFromEvidence(evidence)
	state["attestation_posture"] = attestationPosture
	state["session_binding_present"] = evidence.Session != nil && strings.TrimSpace(evidence.Session.EvidenceDigest) != ""
	state["attestation_evidence_present"] = evidence.Attestation != nil && strings.TrimSpace(evidence.Attestation.EvidenceDigest) != ""
	state["attestation_verification_succeeded"] = evidence.AttestationVerification != nil && evidence.AttestationVerification.VerificationResult == launcherbackend.AttestationVerificationResultValid && evidence.AttestationVerification.ReplayVerdict == launcherbackend.AttestationReplayVerdictOriginal
	if len(attestationReasons) > 0 {
		state["attestation_reason_codes"] = attestationReasons
	}
}

func projectRuntimeBindingIdentityState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	if receipt.IsolateID != "" {
		state["isolate_id"] = receipt.IsolateID
	}
	if receipt.SessionID != "" {
		state["session_id"] = receipt.SessionID
	}
	if receipt.LaunchContextDigest != "" {
		state["launch_context_digest"] = receipt.LaunchContextDigest
	}
}

func projectProvisioningDegradedState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	if receipt.ProvisioningPostureDegraded {
		state["provisioning_posture_degraded"] = true
	}
	if len(receipt.ProvisioningDegradedReasons) > 0 {
		state["provisioning_degraded_reasons"] = receipt.ProvisioningDegradedReasons
	}
}

func projectReceiptBackendEvidenceState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	if receipt.HypervisorImplementation != launcherbackend.HypervisorImplementationUnknown {
		state["hypervisor_implementation"] = receipt.HypervisorImplementation
	}
	if receipt.AccelerationKind != launcherbackend.AccelerationKindUnknown {
		state["acceleration_kind"] = receipt.AccelerationKind
	}
	if receipt.TransportKind != launcherbackend.TransportKindUnknown {
		state["transport_kind"] = receipt.TransportKind
	}
	if receipt.QEMUProvenance != nil {
		provenance := map[string]any{"version": receipt.QEMUProvenance.Version}
		if receipt.QEMUProvenance.BuildIdentity != "" {
			provenance["build_identity"] = receipt.QEMUProvenance.BuildIdentity
		}
		state["qemu_provenance"] = provenance
	}
}

func projectWorkflowDerivedState(state map[string]any, summary RunSummary, manifestHashes []string) {
	if len(manifestHashes) > 0 {
		state["active_manifest_hashes_count"] = len(manifestHashes)
	}
	if summary.WorkflowDefinitionHash != "" {
		state["workflow_definition_hash"] = summary.WorkflowDefinitionHash
	}
	if summary.CurrentStageID != "" {
		state["current_stage_id"] = summary.CurrentStageID
	}
	if summary.ApprovalProfile != "" {
		state["approval_profile"] = summary.ApprovalProfile
	}
	if summary.WorkflowKind != "" {
		state["workflow_kind"] = summary.WorkflowKind
	}
}
