package brokerapi

import (
	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func buildAuthoritativeRunState(summary RunSummary, artifactsForRun []artifacts.ArtifactRecord, pendingIDs []string, manifestHashes []string, runtimeFacts launcherbackend.RuntimeFactsSnapshot) map[string]any {
	receipt := runtimeFacts.LaunchReceipt.Normalized()
	state := buildBaseAuthoritativeRunState(summary, len(artifactsForRun), len(pendingIDs), receipt)
	projectReceiptIdentityState(state, receipt)
	projectReceiptImageState(state, receipt)
	projectReceiptBackendEvidenceState(state, receipt)
	projectReceiptSessionAndAttachmentState(state, receipt)
	projectReceiptLifecycleAndCacheState(state, receipt)
	projectHardeningAndTerminalState(state, runtimeFacts)
	projectWorkflowDerivedState(state, summary, manifestHashes)
	return state
}

func buildBaseAuthoritativeRunState(summary RunSummary, artifactCount int, pendingCount int, receipt launcherbackend.BackendLaunchReceipt) map[string]any {
	return map[string]any{
		"source":                    "broker_store",
		"provenance":                "trusted_derived",
		"status":                    summary.LifecycleState,
		"artifact_count":            artifactCount,
		"pending_approval_count":    pendingCount,
		"workspace_id":              summary.WorkspaceID,
		"backend_kind":              receipt.BackendKind,
		"isolation_assurance_level": receipt.IsolationAssuranceLevel,
		"provisioning_posture":      receipt.ProvisioningPosture,
		"runtime_facts_source":      "launcher_backend_receipt",
	}
}

func projectReceiptIdentityState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	if receipt.IsolateID != "" {
		state["isolate_id"] = receipt.IsolateID
	}
	if receipt.SessionID != "" {
		state["session_id"] = receipt.SessionID
	}
	if receipt.SessionNonce != "" {
		state["session_nonce"] = receipt.SessionNonce
	}
	if receipt.LaunchContextDigest != "" {
		state["launch_context_digest"] = receipt.LaunchContextDigest
	}
	if receipt.HandshakeTranscriptHash != "" {
		state["handshake_transcript_hash"] = receipt.HandshakeTranscriptHash
	}
	if receipt.IsolateSessionKeyIDValue != "" {
		state["isolate_session_key_id_value"] = receipt.IsolateSessionKeyIDValue
	}
	if receipt.HostingNodeID != "" {
		state["hosting_node_id"] = receipt.HostingNodeID
	}
	if receipt.ProvisioningPostureDegraded {
		state["provisioning_posture_degraded"] = true
	}
	if len(receipt.ProvisioningDegradedReasons) > 0 {
		state["provisioning_degraded_reasons"] = receipt.ProvisioningDegradedReasons
	}
}

func projectReceiptImageState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	if receipt.RuntimeImageDescriptorDigest != "" {
		state["runtime_image_descriptor_digest"] = receipt.RuntimeImageDescriptorDigest
		state["runtime_image_digest"] = receipt.RuntimeImageDescriptorDigest
	}
	if receipt.RuntimeImageSignerRef != "" {
		state["runtime_image_signer_ref"] = receipt.RuntimeImageSignerRef
	}
	if receipt.RuntimeImageSignatureDigest != "" {
		state["runtime_image_signature_digest"] = receipt.RuntimeImageSignatureDigest
	}
	if len(receipt.BootComponentDigestByName) > 0 {
		state["boot_component_digest_by_name"] = receipt.BootComponentDigestByName
	}
	if len(receipt.BootComponentDigests) > 0 {
		state["boot_component_digests"] = receipt.BootComponentDigests
	}
	if receipt.LaunchFailureReasonCode != "" {
		state["launch_failure_reason_code"] = receipt.LaunchFailureReasonCode
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

func buildAdvisoryRunState() map[string]any {
	return map[string]any{
		"source":       "runner_advisory",
		"provenance":   "none_reported",
		"available":    false,
		"bounded_keys": []string{},
	}
}
