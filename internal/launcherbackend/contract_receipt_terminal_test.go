package launcherbackend

import "testing"

func TestBackendLaunchReceiptNormalizationDropsInvalidLifecycleWatchdogAndCacheEvidence(t *testing.T) {
	receipt := BackendLaunchReceipt{
		ResourceLimits: &BackendResourceLimits{VCPUCount: 2, MemoryMiB: 512, DiskMiB: 4096, LaunchTimeoutSeconds: 60, BindTimeoutSeconds: 30, ActiveTimeoutSeconds: 600, TerminationGraceSeconds: 15},
		WatchdogPolicy: &BackendWatchdogPolicy{Enabled: true, TerminateOnMisbehavior: false, HeartbeatTimeoutSeconds: 30, NoProgressTimeoutSeconds: 120},
		Lifecycle:      &BackendLifecycleSnapshot{CurrentState: BackendLifecycleStateStarted, PreviousState: BackendLifecycleStateLaunching, TerminateBetweenSteps: false},
		CachePosture:   &BackendCachePosture{WarmPoolEnabled: true, BootCacheEnabled: true, ResetOrDestroyBeforeReuse: false, ReusePriorSessionIdentityKeys: false, DigestPinned: true, SignaturePinned: true},
		CacheEvidence:  &BackendCacheEvidence{ImageCacheResult: "unknown", BootArtifactCacheResult: CacheResultHit, ResolvedImageDescriptorDigest: testDigest("a")},
	}

	normalized := receipt.Normalized()
	if normalized.ResourceLimits == nil {
		t.Fatal("resource_limits should be retained when valid")
	}
	if normalized.WatchdogPolicy != nil {
		t.Fatal("watchdog_policy should be dropped when invalid")
	}
	if normalized.Lifecycle != nil {
		t.Fatal("backend_lifecycle should be dropped when invalid")
	}
	if normalized.CachePosture != nil {
		t.Fatal("cache_posture should be dropped when invalid")
	}
	if normalized.CacheEvidence != nil {
		t.Fatal("cache_evidence should be dropped when invalid")
	}
}

func TestBackendLaunchReceiptNormalizationDropsInvalidAttachmentAndEncryptionProjection(t *testing.T) {
	receipt := BackendLaunchReceipt{
		AttachmentPlanSummary:      &AttachmentPlanSummary{Roles: []AttachmentRoleSummary{{Role: AttachmentRoleWorkspace, ChannelKind: "slot-0"}}, Constraints: AttachmentRealizationConstraints{NoHostFilesystemMounts: true}},
		WorkspaceEncryptionPosture: &WorkspaceEncryptionPosture{Required: true, AtRestProtection: WorkspaceAtRestProtectionHostManagedEncryption, KeyProtectionPosture: WorkspaceKeyProtectionHardwareBacked, Effective: false, DegradedReasons: []string{"missing_hardware_key_support"}},
	}
	normalized := receipt.Normalized()
	if normalized.AttachmentPlanSummary != nil {
		t.Fatal("attachment_plan_summary should be dropped when invalid")
	}
	if normalized.WorkspaceEncryptionPosture != nil {
		t.Fatal("workspace_encryption_posture should be dropped when invalid fail-closed")
	}
}

func TestBackendLaunchReceiptNormalizationKeepsClosedVocabularyAndAliases(t *testing.T) {
	receipt := BackendLaunchReceipt{
		BackendKind:                 "qemu",
		IsolationAssuranceLevel:     "verified",
		ProvisioningPosture:         "something_else",
		RuntimeImageDigest:          testDigest("a"),
		RuntimeImageSignerRef:       "signer:trusted-ci",
		RuntimeImageSignatureDigest: testDigest("f"),
		BootComponentDigestByName:   map[string]string{"kernel": testDigest("d"), "rootfs": testDigest("e")},
		BootComponentDigests:        []string{testDigest("c"), testDigest("b")},
	}
	normalized := receipt.Normalized()
	if normalized.BackendKind != BackendKindUnknown {
		t.Fatalf("backend_kind = %q, want %q", normalized.BackendKind, BackendKindUnknown)
	}
	if normalized.IsolationAssuranceLevel != IsolationAssuranceUnknown {
		t.Fatalf("isolation_assurance_level = %q, want %q", normalized.IsolationAssuranceLevel, IsolationAssuranceUnknown)
	}
	if normalized.ProvisioningPosture != ProvisioningPostureUnknown {
		t.Fatalf("provisioning_posture = %q, want %q", normalized.ProvisioningPosture, ProvisioningPostureUnknown)
	}
	assertRuntimeImageAliasAndEvidence(t, normalized)
}

func assertRuntimeImageAliasAndEvidence(t *testing.T, normalized BackendLaunchReceipt) {
	t.Helper()
	if normalized.RuntimeImageDescriptorDigest != normalized.RuntimeImageDigest {
		t.Fatalf("runtime image digests should be aliased, got descriptor=%q runtime=%q", normalized.RuntimeImageDescriptorDigest, normalized.RuntimeImageDigest)
	}
	if normalized.BootComponentDigests[0] > normalized.BootComponentDigests[1] {
		t.Fatalf("boot_component_digests not sorted: %v", normalized.BootComponentDigests)
	}
	if normalized.RuntimeImageSignerRef != "signer:trusted-ci" {
		t.Fatalf("runtime_image_signer_ref = %q, want signer:trusted-ci", normalized.RuntimeImageSignerRef)
	}
	if normalized.RuntimeImageSignatureDigest == "" {
		t.Fatal("runtime_image_signature_digest should be retained when valid digest")
	}
	if len(normalized.BootComponentDigestByName) == 0 {
		t.Fatal("boot_component_digest_by_name should be retained when valid")
	}
}

func TestBackendLaunchReceiptNormalizationIncludesImplementationEvidence(t *testing.T) {
	receipt := BackendLaunchReceipt{
		HypervisorImplementation: "QEMU",
		AccelerationKind:         "KVM",
		TransportKind:            "VSOCK",
		QEMUProvenance:           &QEMUProvenance{Version: " 9.1.0 ", BuildIdentity: " qemu-system-x86_64 (runecode) "},
	}
	normalized := receipt.Normalized()
	if normalized.HypervisorImplementation != HypervisorImplementationQEMU {
		t.Fatalf("hypervisor_implementation = %q, want %q", normalized.HypervisorImplementation, HypervisorImplementationQEMU)
	}
	if normalized.AccelerationKind != AccelerationKindKVM {
		t.Fatalf("acceleration_kind = %q, want %q", normalized.AccelerationKind, AccelerationKindKVM)
	}
	if normalized.TransportKind != TransportKindVSock {
		t.Fatalf("transport_kind = %q, want %q", normalized.TransportKind, TransportKindVSock)
	}
	if normalized.QEMUProvenance == nil || normalized.QEMUProvenance.Version != "9.1.0" {
		t.Fatalf("qemu_provenance = %#v, want trimmed version", normalized.QEMUProvenance)
	}
}

func TestValidateBackendErrorCodeClosedVocabulary(t *testing.T) {
	codes := []string{
		BackendErrorCodeAccelerationUnavailable,
		BackendErrorCodeHypervisorLaunchFailed,
		BackendErrorCodeImageDescriptorSignatureMismatch,
		BackendErrorCodeAttachmentPlanInvalid,
		BackendErrorCodeHandshakeFailed,
		BackendErrorCodeReplayDetected,
		BackendErrorCodeSessionBindingMismatch,
		BackendErrorCodeGuestUnresponsive,
		BackendErrorCodeWatchdogTimeout,
		BackendErrorCodeRequiredHardeningUnavailable,
		BackendErrorCodeRequiredDiskEncryptionUnavailable,
		BackendErrorCodeContainerAutomaticFallbackDisallowed,
		BackendErrorCodeContainerOptInRequired,
	}
	for _, code := range codes {
		if err := ValidateBackendErrorCode(code); err != nil {
			t.Fatalf("ValidateBackendErrorCode(%q) returned error: %v", code, err)
		}
	}
	if err := ValidateBackendErrorCode("qemu_stderr_contains_error"); err == nil {
		t.Fatal("ValidateBackendErrorCode expected unknown-code rejection")
	}
}

func TestBackendTerminalReportNormalizedAndValidatedFailClosed(t *testing.T) {
	report := BackendTerminalReport{TerminationKind: "FAILED", FailureReasonCode: "WATCHDOG_TIMEOUT", FallbackPosture: "", FailClosed: false}
	normalized := report.Normalized()
	if normalized.TerminationKind != BackendTerminationKindFailed {
		t.Fatalf("termination_kind = %q, want %q", normalized.TerminationKind, BackendTerminationKindFailed)
	}
	if normalized.FailureReasonCode != BackendErrorCodeWatchdogTimeout {
		t.Fatalf("failure_reason_code = %q, want %q", normalized.FailureReasonCode, BackendErrorCodeWatchdogTimeout)
	}
	if normalized.FallbackPosture != BackendFallbackPostureNoAutomaticFallback {
		t.Fatalf("fallback_posture = %q, want %q", normalized.FallbackPosture, BackendFallbackPostureNoAutomaticFallback)
	}
	if !normalized.FailClosed {
		t.Fatal("fail_closed = false, want true")
	}
	if err := normalized.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestBackendTerminalReportNormalizesMissingFailureReasonOnFailedTermination(t *testing.T) {
	report := BackendTerminalReport{TerminationKind: BackendTerminationKindFailed, FailureReasonCode: "", FailClosed: true, FallbackPosture: BackendFallbackPostureNoAutomaticFallback}
	normalized := report.Normalized()
	if normalized.FailureReasonCode != BackendErrorCodeTerminalReportInvalid {
		t.Fatalf("failure_reason_code = %q, want %q", normalized.FailureReasonCode, BackendErrorCodeTerminalReportInvalid)
	}
	if err := normalized.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestBackendTerminalReportCompletedClearsFailureReason(t *testing.T) {
	report := BackendTerminalReport{TerminationKind: BackendTerminationKindCompleted, FailureReasonCode: BackendErrorCodeWatchdogTimeout, FailClosed: true, FallbackPosture: BackendFallbackPostureNoAutomaticFallback}
	normalized := report.Normalized()
	if normalized.FailureReasonCode != "" {
		t.Fatalf("failure_reason_code = %q, want empty for completed termination", normalized.FailureReasonCode)
	}
	if err := normalized.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestBackendLaunchReceiptNormalizationSurfacesTOFUDegradedProvisioning(t *testing.T) {
	receipt := BackendLaunchReceipt{ProvisioningPosture: ProvisioningPostureTOFU, SessionSecurity: &SessionSecurityPosture{MutuallyAuthenticated: true, Encrypted: true, ProofOfPossessionVerified: true, ReplayProtected: true}}
	normalized := receipt.Normalized()
	if !normalized.ProvisioningPostureDegraded {
		t.Fatal("ProvisioningPostureDegraded = false, want true for tofu")
	}
	if len(normalized.ProvisioningDegradedReasons) == 0 {
		t.Fatal("ProvisioningDegradedReasons should include tofu degraded marker")
	}
	if normalized.SessionSecurity.FrameFormat != SessionFramingLengthPrefixedV1 {
		t.Fatalf("session_security.frame_format = %q, want %q", normalized.SessionSecurity.FrameFormat, SessionFramingLengthPrefixedV1)
	}
}

func TestBackendLaunchReceiptNormalizationUsesNotApplicableForContainerSpecificMechanics(t *testing.T) {
	receipt := BackendLaunchReceipt{
		BackendKind:              BackendKindContainer,
		IsolationAssuranceLevel:  IsolationAssuranceDegraded,
		ProvisioningPosture:      "",
		HypervisorImplementation: "",
		AccelerationKind:         "",
		TransportKind:            "",
	}
	normalized := receipt.Normalized()
	if normalized.ProvisioningPosture != ProvisioningPostureNotApplicable {
		t.Fatalf("provisioning_posture = %q, want %q", normalized.ProvisioningPosture, ProvisioningPostureNotApplicable)
	}
	if normalized.HypervisorImplementation != HypervisorImplementationNotApplicable {
		t.Fatalf("hypervisor_implementation = %q, want %q", normalized.HypervisorImplementation, HypervisorImplementationNotApplicable)
	}
	if normalized.AccelerationKind != AccelerationKindNotApplicable {
		t.Fatalf("acceleration_kind = %q, want %q", normalized.AccelerationKind, AccelerationKindNotApplicable)
	}
	if normalized.TransportKind != TransportKindNotApplicable {
		t.Fatalf("transport_kind = %q, want %q", normalized.TransportKind, TransportKindNotApplicable)
	}
	if normalized.ProvisioningPostureDegraded {
		t.Fatal("provisioning_posture_degraded = true, want false for not_applicable posture")
	}
}

func TestQEMUProvenanceRejectsHostPathLeakage(t *testing.T) {
	provenance := QEMUProvenance{Version: "9.1.0", BuildIdentity: "/usr/local/bin/qemu-system-x86_64"}
	if err := provenance.Validate(); err == nil {
		t.Fatal("Validate expected host path leakage rejection")
	}
	provenance = QEMUProvenance{Version: "9.1.0", BuildIdentity: "~/qemu-system-x86_64"}
	if err := provenance.Validate(); err == nil {
		t.Fatal("Validate expected home-relative path leakage rejection")
	}
}
