package brokerapi

import "github.com/runecode-ai/runecode/internal/launcherbackend"

func projectReceiptSessionAndAttachmentState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	if receipt.SessionSecurity != nil {
		state["session_security"] = buildSessionSecurityState(receipt.SessionSecurity)
	}
	if receipt.AttachmentPlanSummary != nil {
		state["attachment_plan"] = buildAttachmentPlanState(*receipt.AttachmentPlanSummary)
	}
	if receipt.WorkspaceEncryptionPosture != nil {
		state["workspace_encryption_posture"] = buildWorkspaceEncryptionState(*receipt.WorkspaceEncryptionPosture)
	}
}

func buildSessionSecurityState(posture *launcherbackend.SessionSecurityPosture) map[string]any {
	state := map[string]any{
		"mutually_authenticated":       posture.MutuallyAuthenticated,
		"encrypted":                    posture.Encrypted,
		"proof_of_possession_verified": posture.ProofOfPossessionVerified,
		"replay_protected":             posture.ReplayProtected,
		"degraded":                     posture.Degraded,
	}
	if posture.FrameFormat != "" {
		state["frame_format"] = posture.FrameFormat
	}
	if posture.MaxFrameBytes > 0 {
		state["max_frame_bytes"] = posture.MaxFrameBytes
	}
	if posture.MaxHandshakeMessageBytes > 0 {
		state["max_handshake_message_bytes"] = posture.MaxHandshakeMessageBytes
	}
	if len(posture.DegradedReasons) > 0 {
		state["degraded_reasons"] = posture.DegradedReasons
	}
	return state
}

func buildAttachmentPlanState(summary launcherbackend.AttachmentPlanSummary) map[string]any {
	state := map[string]any{
		"constraints": map[string]any{
			"no_host_filesystem_mounts":        summary.Constraints.NoHostFilesystemMounts,
			"host_local_paths_visible":         summary.Constraints.HostLocalPathsVisible,
			"device_numbering_visible":         summary.Constraints.DeviceNumberingVisible,
			"guest_mount_as_contract_identity": summary.Constraints.GuestMountAsContractIdentity,
		},
	}
	state["roles"] = buildAttachmentRoleStates(summary.Roles)
	return state
}

func buildAttachmentRoleStates(roles []launcherbackend.AttachmentRoleSummary) []map[string]any {
	entries := make([]map[string]any, 0, len(roles))
	for _, role := range roles {
		entry := map[string]any{
			"role":         role.Role,
			"read_only":    role.ReadOnly,
			"channel_kind": role.ChannelKind,
		}
		if role.DigestCount > 0 {
			entry["digest_count"] = role.DigestCount
		}
		entries = append(entries, entry)
	}
	return entries
}

func buildWorkspaceEncryptionState(posture launcherbackend.WorkspaceEncryptionPosture) map[string]any {
	state := map[string]any{
		"required":               posture.Required,
		"effective":              posture.Effective,
		"at_rest_protection":     posture.AtRestProtection,
		"key_protection_posture": posture.KeyProtectionPosture,
		"degraded":               !posture.Effective || len(posture.DegradedReasons) > 0,
	}
	if len(posture.DegradedReasons) > 0 {
		state["degraded_reasons"] = posture.DegradedReasons
	}
	if len(posture.EvidenceRefs) > 0 {
		state["evidence_refs"] = posture.EvidenceRefs
	}
	return state
}

func projectReceiptLifecycleAndCacheState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	if receipt.ResourceLimits != nil {
		state["resource_limits"] = map[string]any{
			"vcpu_count":                receipt.ResourceLimits.VCPUCount,
			"memory_mib":                receipt.ResourceLimits.MemoryMiB,
			"disk_mib":                  receipt.ResourceLimits.DiskMiB,
			"launch_timeout_seconds":    receipt.ResourceLimits.LaunchTimeoutSeconds,
			"bind_timeout_seconds":      receipt.ResourceLimits.BindTimeoutSeconds,
			"active_timeout_seconds":    receipt.ResourceLimits.ActiveTimeoutSeconds,
			"termination_grace_seconds": receipt.ResourceLimits.TerminationGraceSeconds,
		}
	}
	if receipt.WatchdogPolicy != nil {
		state["watchdog_policy"] = map[string]any{
			"enabled":                     receipt.WatchdogPolicy.Enabled,
			"terminate_on_misbehavior":    receipt.WatchdogPolicy.TerminateOnMisbehavior,
			"heartbeat_timeout_seconds":   receipt.WatchdogPolicy.HeartbeatTimeoutSeconds,
			"no_progress_timeout_seconds": receipt.WatchdogPolicy.NoProgressTimeoutSeconds,
			"termination_reason_code":     receipt.WatchdogPolicy.TerminationReasonCode,
		}
	}
	if receipt.Lifecycle != nil {
		state["backend_lifecycle"] = buildBackendLifecycleState(*receipt.Lifecycle)
	}
	if receipt.CachePosture != nil {
		state["cache_posture"] = map[string]any{
			"warm_pool_enabled":                 receipt.CachePosture.WarmPoolEnabled,
			"boot_cache_enabled":                receipt.CachePosture.BootCacheEnabled,
			"reset_or_destroy_before_reuse":     receipt.CachePosture.ResetOrDestroyBeforeReuse,
			"reuse_prior_session_identity_keys": receipt.CachePosture.ReusePriorSessionIdentityKeys,
			"digest_pinned":                     receipt.CachePosture.DigestPinned,
			"signature_pinned":                  receipt.CachePosture.SignaturePinned,
		}
	}
	if receipt.CacheEvidence != nil {
		state["cache_evidence"] = buildCacheEvidenceState(*receipt.CacheEvidence)
	}
}

func buildBackendLifecycleState(snapshot launcherbackend.BackendLifecycleSnapshot) map[string]any {
	state := map[string]any{
		"current_state":           snapshot.CurrentState,
		"terminate_between_steps": snapshot.TerminateBetweenSteps,
	}
	if snapshot.PreviousState != "" {
		state["previous_state"] = snapshot.PreviousState
	}
	if snapshot.TransitionCount > 0 {
		state["transition_count"] = snapshot.TransitionCount
	}
	return state
}

func buildCacheEvidenceState(evidence launcherbackend.BackendCacheEvidence) map[string]any {
	state := map[string]any{
		"image_cache_result":               evidence.ImageCacheResult,
		"boot_artifact_cache_result":       evidence.BootArtifactCacheResult,
		"resolved_image_descriptor_digest": evidence.ResolvedImageDescriptorDigest,
	}
	if len(evidence.ResolvedBootComponentDigests) > 0 {
		state["resolved_boot_component_digests"] = evidence.ResolvedBootComponentDigests
	}
	return state
}

func projectHardeningAndTerminalState(state map[string]any, runtimeFacts launcherbackend.RuntimeFactsSnapshot) {
	hardening := runtimeFacts.HardeningPosture.Normalized()
	projectHardeningState(state, hardening)
	projectTerminalState(state, runtimeFacts.TerminalReport)
}

func projectHardeningState(state map[string]any, hardening launcherbackend.AppliedHardeningPosture) {
	hardeningSummary := map[string]any{
		"requested": hardening.Requested,
		"effective": hardening.Effective,
		"degraded":  hardening.IsDegraded(),
	}
	projectOptionalHardeningSummaryFields(hardeningSummary, hardening)
	state["applied_hardening_posture"] = hardeningSummary
	state["hardening_requested"] = hardening.Requested
	state["hardening_effective"] = hardening.Effective
	state["hardening_degraded"] = hardening.IsDegraded()
	if len(hardening.DegradedReasons) > 0 {
		state["hardening_degraded_reasons"] = hardening.DegradedReasons
	}
}

func projectOptionalHardeningSummaryFields(summary map[string]any, hardening launcherbackend.AppliedHardeningPosture) {
	projectHardeningStringField(summary, "execution_identity_posture", hardening.ExecutionIdentityPosture)
	projectHardeningStringField(summary, "rootless_posture", hardening.RootlessPosture)
	projectHardeningStringField(summary, "filesystem_exposure_posture", hardening.FilesystemExposurePosture)
	projectHardeningStringField(summary, "writable_layers_posture", hardening.WritableLayersPosture)
	projectHardeningStringField(summary, "network_exposure_posture", hardening.NetworkExposurePosture)
	projectHardeningStringField(summary, "network_namespace_posture", hardening.NetworkNamespacePosture)
	projectHardeningStringField(summary, "network_default_posture", hardening.NetworkDefaultPosture)
	projectHardeningStringField(summary, "egress_enforcement_posture", hardening.EgressEnforcementPosture)
	projectHardeningStringField(summary, "syscall_filtering_posture", hardening.SyscallFilteringPosture)
	projectHardeningStringField(summary, "capabilities_posture", hardening.CapabilitiesPosture)
	projectHardeningStringField(summary, "device_surface_posture", hardening.DeviceSurfacePosture)
	projectHardeningStringField(summary, "control_channel_kind", hardening.ControlChannelKind)
	projectHardeningStringField(summary, "acceleration_kind", hardening.AccelerationKind)
	projectHardeningSliceField(summary, "degraded_reasons", hardening.DegradedReasons)
	projectHardeningSliceField(summary, "backend_evidence_refs", hardening.BackendEvidenceRefs)
}

func projectHardeningStringField(summary map[string]any, key, value string) {
	if value != "" {
		summary[key] = value
	}
}

func projectHardeningSliceField(summary map[string]any, key string, values []string) {
	if len(values) > 0 {
		summary[key] = values
	}
}

func projectTerminalState(state map[string]any, report *launcherbackend.BackendTerminalReport) {
	if report == nil {
		return
	}
	terminal := report.Normalized()
	state["backend_terminal"] = map[string]any{
		"termination_kind":    terminal.TerminationKind,
		"failure_reason_code": terminal.FailureReasonCode,
		"fail_closed":         terminal.FailClosed,
		"fallback_posture":    terminal.FallbackPosture,
		"terminated_at":       terminal.TerminatedAt,
	}
}
