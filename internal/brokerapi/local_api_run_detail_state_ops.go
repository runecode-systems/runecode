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
	pendingByScope := pendingApprovalScopeCounts(advisory.ApprovalWaits)
	if len(pendingByScope) > 0 {
		state["pending_approval_scope_counts"] = pendingByScope
	}
	if checkpointState := buildAdvisoryLastCheckpointState(advisory.LastCheckpoint); checkpointState != nil {
		state["last_checkpoint"] = checkpointState
		bounded = append(bounded, "last_checkpoint")
	}
	if resultState := buildAdvisoryLastResultState(advisory.LastResult); resultState != nil {
		state["last_result"] = resultState
		bounded = append(bounded, "last_result")
	}
	markAdvisoryAvailability(state, bounded)
	if lifecycleHint := buildAdvisoryLifecycleHintState(advisory.Lifecycle); lifecycleHint != nil {
		state["lifecycle_hint"] = lifecycleHint
	}
	if stepAttempts := buildAdvisoryStepAttemptsState(advisory.StepAttempts, pendingByScope); len(stepAttempts) > 0 {
		state["step_attempts"] = stepAttempts
	}
	if len(advisory.ApprovalWaits) > 0 {
		state["approval_waits"] = redactedApprovalWaits(advisory.ApprovalWaits)
	}
	return state
}

func pendingApprovalScopeCounts(waits map[string]artifacts.RunnerApproval) map[string]int {
	counts := map[string]int{}
	for _, approval := range waits {
		if approval.Status != "pending" {
			continue
		}
		counts[approvalScopeKey(approval)]++
	}
	return counts
}

func buildAdvisoryLastCheckpointState(checkpoint *artifacts.RunnerCheckpointAdvisory) map[string]any {
	if checkpoint == nil {
		return nil
	}
	state := map[string]any{
		"lifecycle_state":        checkpoint.LifecycleState,
		"checkpoint_code":        checkpoint.CheckpointCode,
		"occurred_at":            checkpoint.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"idempotency_key":        checkpoint.IdempotencyKey,
		"stage_id":               checkpoint.StageID,
		"step_id":                checkpoint.StepID,
		"role_instance_id":       checkpoint.RoleInstanceID,
		"stage_attempt_id":       checkpoint.StageAttemptID,
		"step_attempt_id":        checkpoint.StepAttemptID,
		"gate_attempt_id":        checkpoint.GateAttemptID,
		"pending_approval_count": checkpoint.PendingApprovals,
	}
	if len(checkpoint.Details) > 0 {
		state["details"] = checkpoint.Details
	}
	return state
}

func buildAdvisoryLastResultState(result *artifacts.RunnerResultAdvisory) map[string]any {
	if result == nil {
		return nil
	}
	state := map[string]any{
		"lifecycle_state":     result.LifecycleState,
		"result_code":         result.ResultCode,
		"occurred_at":         result.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"idempotency_key":     result.IdempotencyKey,
		"stage_id":            result.StageID,
		"step_id":             result.StepID,
		"role_instance_id":    result.RoleInstanceID,
		"stage_attempt_id":    result.StageAttemptID,
		"step_attempt_id":     result.StepAttemptID,
		"gate_attempt_id":     result.GateAttemptID,
		"failure_reason_code": result.FailureReasonCode,
	}
	if len(result.Details) > 0 {
		state["details"] = result.Details
	}
	return state
}

func markAdvisoryAvailability(state map[string]any, bounded []string) {
	if len(bounded) == 0 {
		return
	}
	state["available"] = true
	state["provenance"] = "runner_reported"
	state["bounded_keys"] = bounded
}

func buildAdvisoryLifecycleHintState(lifecycle *artifacts.RunnerLifecycleHint) map[string]any {
	if lifecycle == nil {
		return nil
	}
	return map[string]any{
		"lifecycle_state":  lifecycle.LifecycleState,
		"occurred_at":      lifecycle.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"stage_id":         lifecycle.StageID,
		"step_id":          lifecycle.StepID,
		"role_instance_id": lifecycle.RoleInstanceID,
		"stage_attempt_id": lifecycle.StageAttemptID,
		"step_attempt_id":  lifecycle.StepAttemptID,
		"gate_attempt_id":  lifecycle.GateAttemptID,
	}
}

func buildAdvisoryStepAttemptsState(stepAttempts map[string]artifacts.RunnerStepHint, pendingByScope map[string]int) map[string]any {
	if len(stepAttempts) == 0 {
		return nil
	}
	out := map[string]any{}
	for attemptID, hint := range stepAttempts {
		out[attemptID] = buildAdvisoryStepAttemptEntry(hint, pendingByScope)
	}
	return out
}

func buildAdvisoryStepAttemptEntry(hint artifacts.RunnerStepHint, pendingByScope map[string]int) map[string]any {
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
		return entry
	}
	entry["blocked_on_scope_pending_approval"] = false
	return entry
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
