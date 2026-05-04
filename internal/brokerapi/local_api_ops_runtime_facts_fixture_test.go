package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func launcherRuntimeFactsFixture() launcherbackend.RuntimeFactsSnapshot {
	receipt := launcherRuntimeFactsReceiptFixture()
	return launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt:                 receipt,
		PostHandshakeAttestationInput: runtimeFactsPostHandshakeAttestationInput(receipt),
		HardeningPosture: launcherbackend.AppliedHardeningPosture{
			Requested:           "hardened",
			Effective:           "degraded",
			DegradedReasons:     []string{"seccomp_unavailable"},
			AccelerationKind:    "kvm",
			BackendEvidenceRefs: []string{"qemu-provenance:sha256:" + strings.Repeat("9", 64)},
		},
		TerminalReport: &launcherbackend.BackendTerminalReport{
			TerminationKind:   launcherbackend.BackendTerminationKindFailed,
			FailureReasonCode: launcherbackend.BackendErrorCodeWatchdogTimeout,
			FailClosed:        true,
			FallbackPosture:   launcherbackend.BackendFallbackPostureNoAutomaticFallback,
			TerminatedAt:      "2026-04-09T10:00:00Z",
		},
	}
}

func runtimeFactsPostHandshakeAttestationInput(receipt launcherbackend.BackendLaunchReceipt) *launcherbackend.PostHandshakeRuntimeAttestationInput {
	return &launcherbackend.PostHandshakeRuntimeAttestationInput{
		RunID:                        receipt.RunID,
		IsolateID:                    receipt.IsolateID,
		SessionID:                    receipt.SessionID,
		SessionNonce:                 receipt.SessionNonce,
		RuntimeEvidenceCollected:     true,
		LaunchContextDigest:          receipt.LaunchContextDigest,
		HandshakeTranscriptHash:      receipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     receipt.IsolateSessionKeyIDValue,
		RuntimeImageDescriptorDigest: receipt.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      receipt.RuntimeImageBootProfile,
		RuntimeImageVerifierRef:      receipt.RuntimeImageVerifierRef,
		AuthorityStateDigest:         receipt.AuthorityStateDigest,
		BootComponentDigestByName: map[string]string{
			"kernel": receipt.BootComponentDigestByName["kernel"],
			"initrd": receipt.BootComponentDigestByName["initrd"],
		},
		BootComponentDigests:   append([]string{}, receipt.BootComponentDigests...),
		AttestationSourceKind:  receipt.AttestationEvidenceSourceKind,
		MeasurementProfile:     receipt.AttestationMeasurementProfile,
		FreshnessMaterial:      append([]string{}, receipt.AttestationFreshnessMaterial...),
		FreshnessBindingClaims: append([]string{}, receipt.AttestationFreshnessBindingClaims...),
		EvidenceClaimsDigest:   receipt.AttestationEvidenceClaimsDigest,
	}
}

func launcherRuntimeFactsReceiptFixture() launcherbackend.BackendLaunchReceipt {
	receipt := runtimeFactsMicroVMReceiptIdentity()
	receipt.ResourceLimits = &launcherbackend.BackendResourceLimits{VCPUCount: 2, MemoryMiB: 512, DiskMiB: 4096, LaunchTimeoutSeconds: 60, BindTimeoutSeconds: 30, ActiveTimeoutSeconds: 600, TerminationGraceSeconds: 15}
	receipt.WatchdogPolicy = &launcherbackend.BackendWatchdogPolicy{Enabled: true, TerminateOnMisbehavior: true, HeartbeatTimeoutSeconds: 30, NoProgressTimeoutSeconds: 120}
	receipt.Lifecycle = &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateActive, PreviousState: launcherbackend.BackendLifecycleStateBinding, TerminateBetweenSteps: true, TransitionCount: 4}
	receipt.CachePosture = &launcherbackend.BackendCachePosture{WarmPoolEnabled: true, BootCacheEnabled: true, ResetOrDestroyBeforeReuse: true, ReusePriorSessionIdentityKeys: false, DigestPinned: true, SignaturePinned: true}
	receipt.CacheEvidence = &launcherbackend.BackendCacheEvidence{ImageCacheResult: launcherbackend.CacheResultHit, BootArtifactCacheResult: launcherbackend.CacheResultMiss, ResolvedImageDescriptorDigest: "sha256:" + strings.Repeat("a", 64), ResolvedBootComponentDigests: []string{"sha256:" + strings.Repeat("b", 64), "sha256:" + strings.Repeat("c", 64)}}
	receipt.AttachmentPlanSummary = runtimeFactsAttachmentPlanSummaryFixture()
	receipt.WorkspaceEncryptionPosture = runtimeFactsWorkspaceEncryptionPostureFixture()
	receipt.LaunchFailureReasonCode = launcherbackend.BackendErrorCodeAccelerationUnavailable
	applyRuntimeFactsAttestation(&receipt)
	return receipt
}

func runtimeFactsMicroVMReceiptIdentity() launcherbackend.BackendLaunchReceipt {
	return launcherbackend.BackendLaunchReceipt{
		RunID:                        "run-launcher-facts",
		StageID:                      "artifact_flow",
		RoleInstanceID:               "workspace-1",
		RoleFamily:                   "workspace",
		RoleKind:                     "workspace-edit",
		BackendKind:                  launcherbackend.BackendKindMicroVM,
		IsolationAssuranceLevel:      launcherbackend.IsolationAssuranceIsolated,
		ProvisioningPosture:          launcherbackend.ProvisioningPostureAttested,
		IsolateID:                    "isolate-1",
		SessionID:                    "session-1",
		SessionNonce:                 "nonce-0123456789abcdef",
		LaunchContextDigest:          "sha256:" + strings.Repeat("d", 64),
		HandshakeTranscriptHash:      "sha256:" + strings.Repeat("e", 64),
		IsolateSessionKeyIDValue:     strings.Repeat("f", 64),
		HostingNodeID:                "node-1",
		SessionSecurity:              runtimeFactsSessionSecurityFixture(),
		HypervisorImplementation:     launcherbackend.HypervisorImplementationQEMU,
		AccelerationKind:             launcherbackend.AccelerationKindKVM,
		TransportKind:                launcherbackend.TransportKindVSock,
		QEMUProvenance:               &launcherbackend.QEMUProvenance{Version: "9.1.0", BuildIdentity: "qemu-system-x86_64 (runecode)"},
		RuntimeImageDescriptorDigest: "sha256:" + strings.Repeat("a", 64),
		RuntimeImageBootProfile:      launcherbackend.BootProfileMicroVMLinuxKernelInitrdV1,
		RuntimeImageSignerRef:        "signer:trusted-ci",
		RuntimeImageSignatureDigest:  "sha256:" + strings.Repeat("9", 64),
		AuthorityStateDigest:         "sha256:" + strings.Repeat("8", 64),
		AuthorityStateRevision:       1,
		BootComponentDigestByName:    runtimeFactsBootComponentDigestsByNameFixture(),
		BootComponentDigests:         []string{"sha256:" + strings.Repeat("b", 64), "sha256:" + strings.Repeat("c", 64)},
	}
}

func applyRuntimeFactsAttestation(receipt *launcherbackend.BackendLaunchReceipt) {
	if receipt == nil {
		return
	}
	receipt.AttestationEvidenceSourceKind = launcherbackend.AttestationSourceKindTrustedRuntime
	receipt.AttestationMeasurementProfile = launcherbackend.MeasurementProfileMicroVMBootV1
	receipt.AttestationFreshnessMaterial = []string{"session_nonce"}
	receipt.AttestationFreshnessBindingClaims = []string{"session_nonce", "handshake_transcript_hash", "launch_context_digest"}
	receipt.AttestationEvidenceClaimsDigest = runtimeFactsMeasurementDigests(*receipt)[0]
}

func runtimeFactsMeasurementDigests(receipt launcherbackend.BackendLaunchReceipt) []string {
	digests, err := launcherbackend.DeriveExpectedMeasurementDigests(receipt.AttestationMeasurementProfile, receipt.RuntimeImageBootProfile, receipt.BootComponentDigestByName)
	if err != nil {
		panic(err)
	}
	return digests
}

func runtimeFactsSessionSecurityFixture() *launcherbackend.SessionSecurityPosture {
	return &launcherbackend.SessionSecurityPosture{MutuallyAuthenticated: true, Encrypted: true, ProofOfPossessionVerified: true, ReplayProtected: true, FrameFormat: launcherbackend.SessionFramingLengthPrefixedV1, MaxFrameBytes: 4096, MaxHandshakeMessageBytes: 2048}
}

func runtimeFactsBootComponentDigestsByNameFixture() map[string]string {
	return map[string]string{
		"kernel": "sha256:" + strings.Repeat("b", 64),
		"initrd": "sha256:" + strings.Repeat("c", 64),
	}
}

func runtimeFactsAttachmentPlanSummaryFixture() *launcherbackend.AttachmentPlanSummary {
	return &launcherbackend.AttachmentPlanSummary{
		Roles: []launcherbackend.AttachmentRoleSummary{
			{Role: launcherbackend.AttachmentRoleLaunchContext, ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelReadOnlyVolume, DigestCount: 1},
			{Role: launcherbackend.AttachmentRoleWorkspace, ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelWritableVolume},
			{Role: launcherbackend.AttachmentRoleInputArtifacts, ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelArtifactImage, DigestCount: 2},
			{Role: launcherbackend.AttachmentRoleScratch, ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelEphemeralVolume},
		},
		Constraints: launcherbackend.AttachmentRealizationConstraints{NoHostFilesystemMounts: true, HostLocalPathsVisible: false, DeviceNumberingVisible: false, GuestMountAsContractIdentity: false},
	}
}

func runtimeFactsWorkspaceEncryptionPostureFixture() *launcherbackend.WorkspaceEncryptionPosture {
	return &launcherbackend.WorkspaceEncryptionPosture{Required: true, AtRestProtection: launcherbackend.WorkspaceAtRestProtectionHostManagedEncryption, KeyProtectionPosture: launcherbackend.WorkspaceKeyProtectionHardwareBacked, Effective: true, EvidenceRefs: []string{"workspace-encryption:host-managed"}}
}
