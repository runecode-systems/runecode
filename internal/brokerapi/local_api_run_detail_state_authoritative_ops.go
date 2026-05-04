package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func buildAuthoritativeRunState(summary RunSummary, artifactsForRun []artifacts.ArtifactRecord, pendingIDs []string, manifestHashes []string, policyRefs []string, approvals []ApprovalSummary, runtimeFacts launcherbackend.RuntimeFactsSnapshot, runtimeEvidence launcherbackend.RuntimeEvidenceSnapshot, currentInstanceID string) map[string]any {
	receipt := runtimeFacts.LaunchReceipt.Normalized()
	evidence := runtimeEvidence
	if evidence.Launch.EvidenceDigest == "" {
		var err error
		evidence, _, err = launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(runtimeFacts)
		if err != nil {
			evidence = launcherbackend.RuntimeEvidenceSnapshot{}
		}
	}
	receipt = authoritativeReceiptFromLaunchEvidence(receipt, evidence)
	state := buildBaseAuthoritativeRunState(summary, len(artifactsForRun), len(pendingIDs), receipt)
	projectReceiptIdentityState(state, receipt, evidence)
	projectReceiptImageState(state, receipt)
	projectReceiptBackendEvidenceState(state, receipt)
	projectReceiptSessionAndAttachmentState(state, receipt)
	projectReceiptLifecycleAndCacheState(state, receipt)
	projectHardeningAndTerminalState(state, runtimeFacts)
	projectBackendPostureSelectionEvidenceState(state, currentInstanceID, summary.RunID, policyRefs, approvals)
	projectSupportedRuntimeRequirementsState(state)
	projectWorkflowDerivedState(state, summary, manifestHashes)
	return state
}

func authoritativeReceiptFromLaunchEvidence(receipt launcherbackend.BackendLaunchReceipt, evidence launcherbackend.RuntimeEvidenceSnapshot) launcherbackend.BackendLaunchReceipt {
	if strings.TrimSpace(evidence.Launch.EvidenceDigest) == "" {
		return receipt
	}
	launch := evidence.Launch
	receipt.RunID = launch.RunID
	receipt.StageID = launch.StageID
	receipt.RoleInstanceID = launch.RoleInstanceID
	receipt.RoleFamily = launch.RoleFamily
	receipt.RoleKind = launch.RoleKind
	receipt.BackendKind = launch.BackendKind
	receipt.IsolationAssuranceLevel = launch.IsolationAssuranceLevel
	receipt.ProvisioningPosture = launch.ProvisioningPosture
	receipt.IsolateID = launch.IsolateID
	receipt.SessionID = launch.SessionID
	receipt.LaunchContextDigest = launch.LaunchContextDigest
	receipt.HypervisorImplementation = launch.HypervisorImplementation
	receipt.AccelerationKind = launch.AccelerationKind
	receipt.TransportKind = launch.TransportKind
	receipt.QEMUProvenance = launch.QEMUProvenance
	receipt.RuntimeImageDescriptorDigest = launch.RuntimeImageDescriptorDigest
	receipt.RuntimeImageBootProfile = launch.RuntimeImageBootProfile
	receipt.RuntimeImageSignerRef = launch.RuntimeImageSignerRef
	receipt.RuntimeImageVerifierRef = launch.RuntimeImageVerifierRef
	receipt.RuntimeImageSignatureDigest = launch.RuntimeImageSignatureDigest
	receipt.RuntimeToolchainDescriptorDigest = launch.RuntimeToolchainDescriptorDigest
	receipt.RuntimeToolchainSignerRef = launch.RuntimeToolchainSignerRef
	receipt.RuntimeToolchainVerifierRef = launch.RuntimeToolchainVerifierRef
	receipt.RuntimeToolchainSignatureDigest = launch.RuntimeToolchainSignatureDigest
	receipt.AuthorityStateDigest = launch.AuthorityStateDigest
	receipt.AuthorityStateRevision = launch.AuthorityStateRevision
	receipt.BootComponentDigestByName = launch.BootComponentDigestByName
	receipt.BootComponentDigests = launch.BootComponentDigests
	receipt.AttachmentPlanSummary = launch.AttachmentPlanSummary
	receipt.WorkspaceEncryptionPosture = launch.WorkspaceEncryptionPosture
	receipt.CachePosture = launch.CachePosture
	receipt.CacheEvidence = launch.CacheEvidence
	return receipt
}

func buildBaseAuthoritativeRunState(summary RunSummary, artifactCount int, pendingCount int, receipt launcherbackend.BackendLaunchReceipt) map[string]any {
	return map[string]any{
		"source":                                   "broker_store",
		"provenance":                               "trusted_derived",
		"status":                                   summary.LifecycleState,
		"artifact_count":                           artifactCount,
		"pending_approval_count":                   pendingCount,
		"workspace_id":                             summary.WorkspaceID,
		"project_context_identity_digest":          strings.TrimSpace(summary.ProjectContextIdentity),
		"backend_kind":                             receipt.BackendKind,
		"isolation_assurance_level":                receipt.IsolationAssuranceLevel,
		"runtime_posture_degraded":                 runtimePostureDegraded(receipt.BackendKind, receipt.IsolationAssuranceLevel),
		"provisioning_posture":                     receipt.ProvisioningPosture,
		"attestation_posture":                      launcherbackend.AttestationPostureUnknown,
		"attestation_verifier_class":               launcherbackend.AttestationVerifierClassUnknown,
		"supported_runtime_requirements_satisfied": false,
		"runtime_facts_source":                     "launcher_backend_receipt",
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
	attestationVerifierClass := launcherbackend.DeriveAttestationVerifierClassFromEvidence(evidence)
	state["attestation_posture"] = attestationPosture
	state["attestation_verifier_class"] = attestationVerifierClass
	projectAttestationPresenceState(state, evidence)
	projectAttestationDigestState(state, evidence)
	projectAttestationVerificationMetadataState(state, evidence)
	if len(attestationReasons) > 0 {
		state["attestation_reason_codes"] = attestationReasons
	}
}

func projectAttestationPresenceState(state map[string]any, evidence launcherbackend.RuntimeEvidenceSnapshot) {
	sessionBindingPresent := evidence.Session != nil && strings.TrimSpace(evidence.Session.EvidenceDigest) != ""
	attestationEvidencePresent := evidence.Attestation != nil && strings.TrimSpace(evidence.Attestation.EvidenceDigest) != ""
	attestationVerificationPresent := evidence.AttestationVerification != nil
	attestationVerificationSucceeded := attestationVerificationPresent && evidence.AttestationVerification.VerificationResult == launcherbackend.AttestationVerificationResultValid && evidence.AttestationVerification.ReplayVerdict == launcherbackend.AttestationReplayVerdictOriginal
	state["session_binding_present"] = sessionBindingPresent
	state["attestation_evidence_present"] = attestationEvidencePresent
	state["attestation_verification_succeeded"] = attestationVerificationSucceeded
	state["attestation_verification_failed"] = attestationVerificationPresent && !attestationVerificationSucceeded
}

func projectAttestationDigestState(state map[string]any, evidence launcherbackend.RuntimeEvidenceSnapshot) {
	if evidence.Attestation != nil {
		if digest := strings.TrimSpace(evidence.Attestation.EvidenceDigest); digest != "" {
			state["attestation_evidence_digest"] = digest
		}
	}
	if evidence.AttestationVerification == nil {
		return
	}
	if digest := strings.TrimSpace(evidence.AttestationVerification.AttestationEvidenceDigest); digest != "" {
		state["attestation_verification_attestation_evidence_digest"] = digest
		if _, ok := state["attestation_evidence_digest"]; !ok {
			state["attestation_evidence_digest"] = digest
		}
	}
	if digest := strings.TrimSpace(evidence.AttestationVerification.VerificationDigest); digest != "" {
		state["attestation_verification_digest"] = digest
	}
	if digest := strings.TrimSpace(evidence.AttestationVerification.ReplayIdentityDigest); digest != "" {
		state["attestation_replay_identity_digest"] = digest
	}
}

func projectAttestationVerificationMetadataState(state map[string]any, evidence launcherbackend.RuntimeEvidenceSnapshot) {
	if evidence.AttestationVerification == nil {
		return
	}
	if policyID := strings.TrimSpace(evidence.AttestationVerification.VerifierPolicyID); policyID != "" {
		state["attestation_verifier_policy_id"] = policyID
	}
	if policyDigest := strings.TrimSpace(evidence.AttestationVerification.VerifierPolicyDigest); policyDigest != "" {
		state["attestation_verifier_policy_digest"] = policyDigest
	}
	if profile := strings.TrimSpace(evidence.AttestationVerification.VerificationRulesProfileVersion); profile != "" {
		state["attestation_verification_rules_profile_version"] = profile
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
