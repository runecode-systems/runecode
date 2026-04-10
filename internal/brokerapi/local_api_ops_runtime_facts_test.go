package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestRunSummaryAndDetailProjectRecordedLauncherRuntimeFacts(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-launcher-facts", "step-1")
	s.RecordRuntimeFacts("run-launcher-facts", launcherRuntimeFactsFixture())

	runList := mustRunListForRuntimeFactsTest(t, s)
	assertRuntimeFactsRunListProjection(t, runList.Runs)

	runGet := mustRunGetForRuntimeFactsTest(t, s)
	state := runGet.Run.AuthoritativeState
	assertRuntimeFactsIdentityProjection(t, state)
	assertRuntimeFactsImageProjection(t, state)
	assertRuntimeFactsBackendEvidenceProjection(t, state)
	assertRuntimeFactsRuntimePolicyProjection(t, state)
	assertRuntimeFactsSessionAndHardeningProjection(t, state)
	assertRuntimeFactsAttachmentProjection(t, state)
	assertRuntimeFactsTerminalProjection(t, state)
}

func launcherRuntimeFactsFixture() launcherbackend.RuntimeFactsSnapshot {
	return launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt:    launcherRuntimeFactsReceiptFixture(),
		HardeningPosture: launcherbackend.AppliedHardeningPosture{Requested: "hardened", Effective: "degraded", DegradedReasons: []string{"seccomp_unavailable"}, AccelerationKind: "kvm", BackendEvidenceRefs: []string{"qemu-provenance:sha256:" + strings.Repeat("9", 64)}},
		TerminalReport:   &launcherbackend.BackendTerminalReport{TerminationKind: launcherbackend.BackendTerminationKindFailed, FailureReasonCode: launcherbackend.BackendErrorCodeWatchdogTimeout, FailClosed: true, FallbackPosture: launcherbackend.BackendFallbackPostureNoAutomaticFallback, TerminatedAt: "2026-04-09T10:00:00Z"},
	}
}

func launcherRuntimeFactsReceiptFixture() launcherbackend.BackendLaunchReceipt {
	return launcherbackend.BackendLaunchReceipt{
		RunID:                        "run-launcher-facts",
		StageID:                      "artifact_flow",
		RoleInstanceID:               "workspace-1",
		RoleFamily:                   "workspace",
		RoleKind:                     "workspace-edit",
		BackendKind:                  launcherbackend.BackendKindMicroVM,
		IsolationAssuranceLevel:      launcherbackend.IsolationAssuranceIsolated,
		ProvisioningPosture:          launcherbackend.ProvisioningPostureTOFU,
		IsolateID:                    "isolate-1",
		SessionID:                    "session-1",
		SessionNonce:                 "nonce-0123456789abcdef",
		LaunchContextDigest:          "sha256:" + strings.Repeat("d", 64),
		HandshakeTranscriptHash:      "sha256:" + strings.Repeat("e", 64),
		IsolateSessionKeyIDValue:     strings.Repeat("f", 64),
		HostingNodeID:                "node-1",
		SessionSecurity:              &launcherbackend.SessionSecurityPosture{MutuallyAuthenticated: true, Encrypted: true, ProofOfPossessionVerified: true, ReplayProtected: true, FrameFormat: launcherbackend.SessionFramingLengthPrefixedV1, MaxFrameBytes: 4096, MaxHandshakeMessageBytes: 2048},
		HypervisorImplementation:     launcherbackend.HypervisorImplementationQEMU,
		AccelerationKind:             launcherbackend.AccelerationKindKVM,
		TransportKind:                launcherbackend.TransportKindVSock,
		QEMUProvenance:               &launcherbackend.QEMUProvenance{Version: "9.1.0", BuildIdentity: "qemu-system-x86_64 (runecode)"},
		RuntimeImageDescriptorDigest: "sha256:" + strings.Repeat("a", 64),
		RuntimeImageSignerRef:        "signer:trusted-ci",
		RuntimeImageSignatureDigest:  "sha256:" + strings.Repeat("9", 64),
		BootComponentDigestByName: map[string]string{
			"kernel": "sha256:" + strings.Repeat("b", 64),
			"rootfs": "sha256:" + strings.Repeat("c", 64),
		},
		BootComponentDigests:       []string{"sha256:" + strings.Repeat("b", 64), "sha256:" + strings.Repeat("c", 64)},
		ResourceLimits:             &launcherbackend.BackendResourceLimits{VCPUCount: 2, MemoryMiB: 512, DiskMiB: 4096, LaunchTimeoutSeconds: 60, BindTimeoutSeconds: 30, ActiveTimeoutSeconds: 600, TerminationGraceSeconds: 15},
		WatchdogPolicy:             &launcherbackend.BackendWatchdogPolicy{Enabled: true, TerminateOnMisbehavior: true, HeartbeatTimeoutSeconds: 30, NoProgressTimeoutSeconds: 120},
		Lifecycle:                  &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateActive, PreviousState: launcherbackend.BackendLifecycleStateBinding, TerminateBetweenSteps: true, TransitionCount: 4},
		CachePosture:               &launcherbackend.BackendCachePosture{WarmPoolEnabled: true, BootCacheEnabled: true, ResetOrDestroyBeforeReuse: true, ReusePriorSessionIdentityKeys: false, DigestPinned: true, SignaturePinned: true},
		CacheEvidence:              &launcherbackend.BackendCacheEvidence{ImageCacheResult: launcherbackend.CacheResultHit, BootArtifactCacheResult: launcherbackend.CacheResultMiss, ResolvedImageDescriptorDigest: "sha256:" + strings.Repeat("a", 64), ResolvedBootComponentDigests: []string{"sha256:" + strings.Repeat("b", 64), "sha256:" + strings.Repeat("c", 64)}},
		AttachmentPlanSummary:      runtimeFactsAttachmentPlanSummaryFixture(),
		WorkspaceEncryptionPosture: runtimeFactsWorkspaceEncryptionPostureFixture(),
		LaunchFailureReasonCode:    launcherbackend.BackendErrorCodeAccelerationUnavailable,
	}
}

func runtimeFactsAttachmentPlanSummaryFixture() *launcherbackend.AttachmentPlanSummary {
	return &launcherbackend.AttachmentPlanSummary{
		Roles: []launcherbackend.AttachmentRoleSummary{
			{Role: launcherbackend.AttachmentRoleLaunchContext, ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelReadOnlyChannel, DigestCount: 1},
			{Role: launcherbackend.AttachmentRoleWorkspace, ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelVirtualDisk},
			{Role: launcherbackend.AttachmentRoleInputArtifacts, ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelVirtualDisk, DigestCount: 2},
			{Role: launcherbackend.AttachmentRoleScratch, ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelEphemeralDisk},
		},
		Constraints: launcherbackend.AttachmentRealizationConstraints{NoHostFilesystemMounts: true, HostLocalPathsVisible: false, DeviceNumberingVisible: false, GuestMountAsContractIdentity: false},
	}
}

func runtimeFactsWorkspaceEncryptionPostureFixture() *launcherbackend.WorkspaceEncryptionPosture {
	return &launcherbackend.WorkspaceEncryptionPosture{Required: true, AtRestProtection: launcherbackend.WorkspaceAtRestProtectionHostManagedEncryption, KeyProtectionPosture: launcherbackend.WorkspaceKeyProtectionHardwareBacked, Effective: true, EvidenceRefs: []string{"workspace-encryption:host-managed"}}
}

func mustRunListForRuntimeFactsTest(t *testing.T, service *Service) RunListResponse {
	t.Helper()
	response, errResp := service.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "req-run-list-runtime-facts", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunList error response: %+v", errResp)
	}
	return response
}

func mustRunGetForRuntimeFactsTest(t *testing.T, service *Service) RunGetResponse {
	t.Helper()
	response, errResp := service.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-get-runtime-facts", RunID: "run-launcher-facts"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	return response
}

func assertRuntimeFactsRunListProjection(t *testing.T, runs []RunSummary) {
	t.Helper()
	if len(runs) != 1 {
		t.Fatalf("run count = %d, want 1", len(runs))
	}
	if runs[0].BackendKind != launcherbackend.BackendKindMicroVM {
		t.Fatalf("backend_kind = %q, want %q", runs[0].BackendKind, launcherbackend.BackendKindMicroVM)
	}
	if runs[0].IsolationAssuranceLevel != launcherbackend.IsolationAssuranceIsolated {
		t.Fatalf("isolation_assurance_level = %q, want %q", runs[0].IsolationAssuranceLevel, launcherbackend.IsolationAssuranceIsolated)
	}
	if runs[0].ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("provisioning_posture = %q, want %q", runs[0].ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
	}
	if runs[0].AssuranceLevel != runs[0].IsolationAssuranceLevel {
		t.Fatalf("assurance_level alias = %q, want %q", runs[0].AssuranceLevel, runs[0].IsolationAssuranceLevel)
	}
}

func assertRuntimeFactsIdentityProjection(t *testing.T, state map[string]any) {
	t.Helper()
	if state["backend_kind"] != launcherbackend.BackendKindMicroVM {
		t.Fatalf("authoritative_state.backend_kind = %v, want %q", state["backend_kind"], launcherbackend.BackendKindMicroVM)
	}
	if state["isolate_id"] != "isolate-1" {
		t.Fatalf("authoritative_state.isolate_id = %v, want isolate-1", state["isolate_id"])
	}
	if state["session_nonce"] != "nonce-0123456789abcdef" {
		t.Fatalf("authoritative_state.session_nonce = %v, want nonce", state["session_nonce"])
	}
	if state["launch_context_digest"] != "sha256:"+strings.Repeat("d", 64) {
		t.Fatalf("authoritative_state.launch_context_digest = %v, want launch context digest", state["launch_context_digest"])
	}
	if state["handshake_transcript_hash"] != "sha256:"+strings.Repeat("e", 64) {
		t.Fatalf("authoritative_state.handshake_transcript_hash = %v, want handshake transcript hash", state["handshake_transcript_hash"])
	}
	if state["isolate_session_key_id_value"] != strings.Repeat("f", 64) {
		t.Fatalf("authoritative_state.isolate_session_key_id_value = %v, want pinned isolate key id", state["isolate_session_key_id_value"])
	}
	if state["hosting_node_id"] != "node-1" {
		t.Fatalf("authoritative_state.hosting_node_id = %v, want node-1", state["hosting_node_id"])
	}
	if state["provisioning_posture_degraded"] != true {
		t.Fatalf("authoritative_state.provisioning_posture_degraded = %v, want true for tofu", state["provisioning_posture_degraded"])
	}
	if state["provisioning_degraded_reasons"] == nil {
		t.Fatal("authoritative_state.provisioning_degraded_reasons should be present for tofu")
	}
}

func assertRuntimeFactsImageProjection(t *testing.T, state map[string]any) {
	t.Helper()
	if state["runtime_image_descriptor_digest"] != "sha256:"+strings.Repeat("a", 64) {
		t.Fatalf("authoritative_state.runtime_image_descriptor_digest = %v, want image descriptor digest", state["runtime_image_descriptor_digest"])
	}
	if state["runtime_image_digest"] != "sha256:"+strings.Repeat("a", 64) {
		t.Fatalf("authoritative_state.runtime_image_digest = %v, want image descriptor digest alias", state["runtime_image_digest"])
	}
	if state["runtime_image_signer_ref"] != "signer:trusted-ci" {
		t.Fatalf("authoritative_state.runtime_image_signer_ref = %v, want signer reference", state["runtime_image_signer_ref"])
	}
	if state["runtime_image_signature_digest"] != "sha256:"+strings.Repeat("9", 64) {
		t.Fatalf("authoritative_state.runtime_image_signature_digest = %v, want signature digest", state["runtime_image_signature_digest"])
	}
	byName, ok := state["boot_component_digest_by_name"].(map[string]string)
	if !ok {
		t.Fatalf("authoritative_state.boot_component_digest_by_name = %T, want map[string]string", state["boot_component_digest_by_name"])
	}
	if byName["kernel"] == "" || byName["rootfs"] == "" {
		t.Fatalf("authoritative_state.boot_component_digest_by_name missing kernel/rootfs digests: %#v", byName)
	}
}

func assertRuntimeFactsBackendEvidenceProjection(t *testing.T, state map[string]any) {
	t.Helper()
	if state["hypervisor_implementation"] != launcherbackend.HypervisorImplementationQEMU {
		t.Fatalf("authoritative_state.hypervisor_implementation = %v, want %q", state["hypervisor_implementation"], launcherbackend.HypervisorImplementationQEMU)
	}
	if state["acceleration_kind"] != launcherbackend.AccelerationKindKVM {
		t.Fatalf("authoritative_state.acceleration_kind = %v, want %q", state["acceleration_kind"], launcherbackend.AccelerationKindKVM)
	}
	if state["transport_kind"] != launcherbackend.TransportKindVSock {
		t.Fatalf("authoritative_state.transport_kind = %v, want %q", state["transport_kind"], launcherbackend.TransportKindVSock)
	}
	qemuProvenance, ok := state["qemu_provenance"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.qemu_provenance = %T, want map", state["qemu_provenance"])
	}
	if qemuProvenance["version"] != "9.1.0" {
		t.Fatalf("authoritative_state.qemu_provenance.version = %v, want 9.1.0", qemuProvenance["version"])
	}
}

func assertRuntimeFactsRuntimePolicyProjection(t *testing.T, state map[string]any) {
	t.Helper()
	resourceLimits, ok := state["resource_limits"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.resource_limits = %T, want map", state["resource_limits"])
	}
	if resourceLimits["vcpu_count"] != 2 || resourceLimits["memory_mib"] != 512.0 && resourceLimits["memory_mib"] != 512 {
		t.Fatalf("authoritative_state.resource_limits = %#v, want vcpu_count=2 memory_mib=512", resourceLimits)
	}
	assertRuntimeFactsWatchdogLifecycleAndCache(t, state)
}

func assertRuntimeFactsWatchdogLifecycleAndCache(t *testing.T, state map[string]any) {
	t.Helper()
	assertRuntimeFactsWatchdogProjection(t, state)
	assertRuntimeFactsLifecycleProjection(t, state)
	assertRuntimeFactsCacheProjection(t, state)
}

func assertRuntimeFactsWatchdogProjection(t *testing.T, state map[string]any) {
	t.Helper()
	watchdogPolicy, ok := state["watchdog_policy"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.watchdog_policy = %T, want map", state["watchdog_policy"])
	}
	if watchdogPolicy["termination_reason_code"] != launcherbackend.WatchdogTerminationReasonCodeTimeout {
		t.Fatalf("authoritative_state.watchdog_policy.termination_reason_code = %v, want %q", watchdogPolicy["termination_reason_code"], launcherbackend.WatchdogTerminationReasonCodeTimeout)
	}
}

func assertRuntimeFactsLifecycleProjection(t *testing.T, state map[string]any) {
	t.Helper()
	lifecycle, ok := state["backend_lifecycle"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.backend_lifecycle = %T, want map", state["backend_lifecycle"])
	}
	if lifecycle["current_state"] != launcherbackend.BackendLifecycleStateActive || lifecycle["terminate_between_steps"] != true {
		t.Fatalf("authoritative_state.backend_lifecycle = %#v, want active + terminate_between_steps=true", lifecycle)
	}
}

func assertRuntimeFactsCacheProjection(t *testing.T, state map[string]any) {
	t.Helper()
	cachePosture, ok := state["cache_posture"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.cache_posture = %T, want map", state["cache_posture"])
	}
	if cachePosture["reset_or_destroy_before_reuse"] != true || cachePosture["reuse_prior_session_identity_keys"] != false {
		t.Fatalf("authoritative_state.cache_posture = %#v, want reset/destroy true and key reuse false", cachePosture)
	}
	cacheEvidence, ok := state["cache_evidence"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.cache_evidence = %T, want map", state["cache_evidence"])
	}
	if cacheEvidence["image_cache_result"] != launcherbackend.CacheResultHit || cacheEvidence["boot_artifact_cache_result"] != launcherbackend.CacheResultMiss {
		t.Fatalf("authoritative_state.cache_evidence = %#v, want hit/miss cache results", cacheEvidence)
	}
}

func assertRuntimeFactsSessionAndHardeningProjection(t *testing.T, state map[string]any) {
	t.Helper()
	sessionSecurity, ok := state["session_security"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.session_security = %T, want map", state["session_security"])
	}
	if sessionSecurity["encrypted"] != true || sessionSecurity["mutually_authenticated"] != true {
		t.Fatalf("authoritative_state.session_security = %#v, want encrypted+mutually_authenticated true", sessionSecurity)
	}
	hardening, ok := state["applied_hardening_posture"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.applied_hardening_posture = %T, want map", state["applied_hardening_posture"])
	}
	assertRuntimeFactsHardeningProjection(t, state, hardening)
}

func assertRuntimeFactsHardeningProjection(t *testing.T, state map[string]any, hardening map[string]any) {
	t.Helper()
	if hardening["degraded"] != true {
		t.Fatalf("applied_hardening_posture.degraded = %v, want true", hardening["degraded"])
	}
	if hardening["effective"] != "degraded" {
		t.Fatalf("applied_hardening_posture.effective = %v, want degraded", hardening["effective"])
	}
	if hardening["execution_identity_posture"] == nil || hardening["filesystem_exposure_posture"] == nil || hardening["network_exposure_posture"] == nil || hardening["syscall_filtering_posture"] == nil || hardening["device_surface_posture"] == nil {
		t.Fatalf("applied_hardening_posture missing one or more required common posture fields: %#v", hardening)
	}
	if hardening["control_channel_kind"] == nil || hardening["acceleration_kind"] == nil {
		t.Fatalf("applied_hardening_posture missing control_channel_kind/acceleration_kind: %#v", hardening)
	}
	if hardening["backend_evidence_refs"] == nil {
		t.Fatal("applied_hardening_posture.backend_evidence_refs should be present")
	}
	if state["hardening_degraded"] != true {
		t.Fatalf("authoritative_state.hardening_degraded = %v, want true", state["hardening_degraded"])
	}
}

func assertRuntimeFactsAttachmentProjection(t *testing.T, state map[string]any) {
	t.Helper()
	attachmentPlan, ok := state["attachment_plan"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.attachment_plan = %T, want map", state["attachment_plan"])
	}
	constraints, ok := attachmentPlan["constraints"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.attachment_plan.constraints = %T, want map", attachmentPlan["constraints"])
	}
	if constraints["no_host_filesystem_mounts"] != true {
		t.Fatalf("authoritative_state.attachment_plan.constraints.no_host_filesystem_mounts = %v, want true", constraints["no_host_filesystem_mounts"])
	}
	if constraints["host_local_paths_visible"] != false || constraints["device_numbering_visible"] != false || constraints["guest_mount_as_contract_identity"] != false {
		t.Fatalf("authoritative_state.attachment_plan.constraints = %#v, want all leakage toggles false", constraints)
	}
	if got := attachmentRolesLen(attachmentPlan["roles"]); got != 4 {
		t.Fatalf("authoritative_state.attachment_plan.roles len = %d, want 4", got)
	}
	assertRuntimeFactsWorkspaceEncryptionProjection(t, state)
}

func assertRuntimeFactsWorkspaceEncryptionProjection(t *testing.T, state map[string]any) {
	t.Helper()
	workspaceEncryption, ok := state["workspace_encryption_posture"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.workspace_encryption_posture = %T, want map", state["workspace_encryption_posture"])
	}
	if workspaceEncryption["required"] != true || workspaceEncryption["effective"] != true {
		t.Fatalf("authoritative_state.workspace_encryption_posture required/effective = %#v, want true/true", workspaceEncryption)
	}
	if workspaceEncryption["at_rest_protection"] != launcherbackend.WorkspaceAtRestProtectionHostManagedEncryption {
		t.Fatalf("authoritative_state.workspace_encryption_posture.at_rest_protection = %v, want %q", workspaceEncryption["at_rest_protection"], launcherbackend.WorkspaceAtRestProtectionHostManagedEncryption)
	}
	if workspaceEncryption["key_protection_posture"] != launcherbackend.WorkspaceKeyProtectionHardwareBacked {
		t.Fatalf("authoritative_state.workspace_encryption_posture.key_protection_posture = %v, want %q", workspaceEncryption["key_protection_posture"], launcherbackend.WorkspaceKeyProtectionHardwareBacked)
	}
}

func attachmentRolesLen(value any) int {
	if typed, ok := value.([]map[string]any); ok {
		return len(typed)
	}
	if typed, ok := value.([]any); ok {
		return len(typed)
	}
	return -1
}

func assertRuntimeFactsTerminalProjection(t *testing.T, state map[string]any) {
	t.Helper()
	terminal, ok := state["backend_terminal"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.backend_terminal = %T, want map", state["backend_terminal"])
	}
	if terminal["failure_reason_code"] != launcherbackend.BackendErrorCodeWatchdogTimeout {
		t.Fatalf("backend_terminal.failure_reason_code = %v, want %q", terminal["failure_reason_code"], launcherbackend.BackendErrorCodeWatchdogTimeout)
	}
	if terminal["fail_closed"] != true {
		t.Fatalf("backend_terminal.fail_closed = %v, want true", terminal["fail_closed"])
	}
	if terminal["fallback_posture"] != launcherbackend.BackendFallbackPostureNoAutomaticFallback {
		t.Fatalf("backend_terminal.fallback_posture = %v, want %q", terminal["fallback_posture"], launcherbackend.BackendFallbackPostureNoAutomaticFallback)
	}
}
