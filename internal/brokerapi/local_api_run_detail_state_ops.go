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

func buildAdvisoryRunState(advisory artifacts.RunnerAdvisoryState) map[string]any {
	state := map[string]any{
		"source":       "runner_advisory",
		"provenance":   "none_reported",
		"available":    false,
		"bounded_keys": []string{},
	}
	bounded := make([]string, 0, 2)
	pendingByScope := map[string]int{}
	for _, approval := range advisory.ApprovalWaits {
		if approval.Status != "pending" {
			continue
		}
		scope := approvalScopeKey(approval)
		pendingByScope[scope]++
	}
	if len(pendingByScope) > 0 {
		state["pending_approval_scope_counts"] = pendingByScope
	}
	if advisory.LastCheckpoint != nil {
		state["last_checkpoint"] = map[string]any{
			"lifecycle_state":        advisory.LastCheckpoint.LifecycleState,
			"checkpoint_code":        advisory.LastCheckpoint.CheckpointCode,
			"occurred_at":            advisory.LastCheckpoint.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			"idempotency_key":        advisory.LastCheckpoint.IdempotencyKey,
			"stage_id":               advisory.LastCheckpoint.StageID,
			"step_id":                advisory.LastCheckpoint.StepID,
			"role_instance_id":       advisory.LastCheckpoint.RoleInstanceID,
			"stage_attempt_id":       advisory.LastCheckpoint.StageAttemptID,
			"step_attempt_id":        advisory.LastCheckpoint.StepAttemptID,
			"gate_attempt_id":        advisory.LastCheckpoint.GateAttemptID,
			"pending_approval_count": advisory.LastCheckpoint.PendingApprovals,
		}
		if len(advisory.LastCheckpoint.Details) > 0 {
			state["last_checkpoint"].(map[string]any)["details"] = advisory.LastCheckpoint.Details
		}
		bounded = append(bounded, "last_checkpoint")
	}
	if advisory.LastResult != nil {
		state["last_result"] = map[string]any{
			"lifecycle_state":     advisory.LastResult.LifecycleState,
			"result_code":         advisory.LastResult.ResultCode,
			"occurred_at":         advisory.LastResult.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			"idempotency_key":     advisory.LastResult.IdempotencyKey,
			"stage_id":            advisory.LastResult.StageID,
			"step_id":             advisory.LastResult.StepID,
			"role_instance_id":    advisory.LastResult.RoleInstanceID,
			"stage_attempt_id":    advisory.LastResult.StageAttemptID,
			"step_attempt_id":     advisory.LastResult.StepAttemptID,
			"gate_attempt_id":     advisory.LastResult.GateAttemptID,
			"failure_reason_code": advisory.LastResult.FailureReasonCode,
		}
		if len(advisory.LastResult.Details) > 0 {
			state["last_result"].(map[string]any)["details"] = advisory.LastResult.Details
		}
		bounded = append(bounded, "last_result")
	}
	if len(bounded) > 0 {
		state["available"] = true
		state["provenance"] = "runner_reported"
		state["bounded_keys"] = bounded
	}
	if advisory.Lifecycle != nil {
		state["lifecycle_hint"] = map[string]any{
			"lifecycle_state":  advisory.Lifecycle.LifecycleState,
			"occurred_at":      advisory.Lifecycle.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			"stage_id":         advisory.Lifecycle.StageID,
			"step_id":          advisory.Lifecycle.StepID,
			"role_instance_id": advisory.Lifecycle.RoleInstanceID,
			"stage_attempt_id": advisory.Lifecycle.StageAttemptID,
			"step_attempt_id":  advisory.Lifecycle.StepAttemptID,
			"gate_attempt_id":  advisory.Lifecycle.GateAttemptID,
		}
	}
	if len(advisory.StepAttempts) > 0 {
		stepAttempts := map[string]any{}
		for attemptID, hint := range advisory.StepAttempts {
			entry := map[string]any{
				"step_attempt_id":  hint.StepAttemptID,
				"run_id":           hint.RunID,
				"stage_id":         hint.StageID,
				"step_id":          hint.StepID,
				"role_instance_id": hint.RoleInstanceID,
				"stage_attempt_id": hint.StageAttemptID,
				"gate_attempt_id":  hint.GateAttemptID,
				"status":           hint.Status,
				"last_updated_at":  hint.LastUpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			}
			if !hint.StartedAt.IsZero() {
				entry["started_at"] = hint.StartedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			}
			if !hint.FinishedAt.IsZero() {
				entry["finished_at"] = hint.FinishedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			}
			if hint.CurrentPhase != "" {
				entry["current_phase"] = hint.CurrentPhase
			}
			if hint.PhaseStatus != "" {
				entry["phase_status"] = hint.PhaseStatus
			}
			scopeKey := approvalScopeKey(artifacts.RunnerApproval{RunID: hint.RunID, StageID: hint.StageID, StepID: hint.StepID, RoleInstanceID: hint.RoleInstanceID})
			if pending := pendingByScope[scopeKey]; pending > 0 {
				entry["blocked_on_scope_pending_approval"] = true
				entry["pending_approval_scope_count"] = pending
			} else {
				entry["blocked_on_scope_pending_approval"] = false
			}
			stepAttempts[attemptID] = entry
		}
		state["step_attempts"] = stepAttempts
	}
	if len(advisory.ApprovalWaits) > 0 {
		state["approval_waits"] = redactedApprovalWaits(advisory.ApprovalWaits)
	}
	return state
}

func redactedApprovalWaits(waits map[string]artifacts.RunnerApproval) map[string]artifacts.RunnerApproval {
	out := make(map[string]artifacts.RunnerApproval, len(waits))
	for approvalID, wait := range waits {
		copyWait := wait
		copyWait.BoundActionHash = ""
		copyWait.BoundStageSummaryHash = ""
		out[approvalID] = copyWait
	}
	return out
}

func approvalScopeKey(approval artifacts.RunnerApproval) string {
	return approval.RunID + "|" + approval.StageID + "|" + approval.StepID + "|" + approval.RoleInstanceID
}
