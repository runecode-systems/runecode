package launcherbackend

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
)

func testDigest(char string) string {
	return "sha256:" + strings.Repeat(char, 64)
}

func validMicroVMSpecForContractTests() BackendLaunchSpec {
	return BackendLaunchSpec{
		RunID:                     "run-1",
		StageID:                   "stage-1",
		RoleInstanceID:            "workspace-1",
		RoleFamily:                "workspace",
		RoleKind:                  "workspace-edit",
		RequestedBackend:          BackendKindMicroVM,
		RequestedAccelerationKind: AccelerationKindKVM,
		ControlTransportKind:      TransportKindVSock,
		Image:                     validRuntimeImageDescriptorForContractTests(),
		Attachments:               validAttachmentPlanForContractTests(),
		ResourceLimits:            validResourceLimitsForContractTests(),
		WatchdogPolicy:            validWatchdogPolicyForContractTests(),
		LifecyclePolicy:           BackendLifecyclePolicy{TerminateBetweenSteps: true},
		CachePosture:              validCachePostureForContractTests(),
	}
}

func validRuntimeImageDescriptorForContractTests() RuntimeImageDescriptor {
	return RuntimeImageDescriptor{
		DescriptorDigest:      testDigest("1"),
		BackendKind:           BackendKindMicroVM,
		PlatformCompatibility: RuntimeImagePlatformCompat{OS: "linux", Architecture: "amd64", AccelerationKind: AccelerationKindKVM},
		BootContractVersion:   "v1",
		ComponentDigests: map[string]string{
			"kernel": testDigest("2"),
			"rootfs": testDigest("4"),
		},
		Signing: &RuntimeImageSigningHooks{SignerRef: "signer:trusted-ci", SignatureDigest: testDigest("5")},
	}
}

func validResourceLimitsForContractTests() BackendResourceLimits {
	return BackendResourceLimits{VCPUCount: 2, MemoryMiB: 512, DiskMiB: 4096, LaunchTimeoutSeconds: 60, BindTimeoutSeconds: 30, ActiveTimeoutSeconds: 600, TerminationGraceSeconds: 15}
}

func validWatchdogPolicyForContractTests() BackendWatchdogPolicy {
	return BackendWatchdogPolicy{Enabled: true, TerminateOnMisbehavior: true, HeartbeatTimeoutSeconds: 30, NoProgressTimeoutSeconds: 120}
}

func validCachePostureForContractTests() BackendCachePosture {
	return BackendCachePosture{WarmPoolEnabled: true, BootCacheEnabled: true, ResetOrDestroyBeforeReuse: true, ReusePriorSessionIdentityKeys: false, DigestPinned: true, SignaturePinned: true}
}

func validAttachmentPlanForContractTests() AttachmentPlan {
	return AttachmentPlan{
		ByRole: map[string]AttachmentBinding{
			AttachmentRoleLaunchContext:  {ReadOnly: true, ChannelKind: AttachmentChannelReadOnlyChannel, RequiredDigests: []string{testDigest("3")}},
			AttachmentRoleWorkspace:      {ReadOnly: false, ChannelKind: AttachmentChannelVirtualDisk},
			AttachmentRoleInputArtifacts: {ReadOnly: true, ChannelKind: AttachmentChannelVirtualDisk, RequiredDigests: []string{testDigest("6")}},
			AttachmentRoleScratch:        {ReadOnly: false, ChannelKind: AttachmentChannelEphemeralDisk},
		},
		Constraints: AttachmentRealizationConstraints{NoHostFilesystemMounts: true},
		WorkspaceEncryption: &WorkspaceEncryptionPosture{
			Required:             true,
			AtRestProtection:     WorkspaceAtRestProtectionHostManagedEncryption,
			KeyProtectionPosture: WorkspaceKeyProtectionHardwareBacked,
			Effective:            true,
			EvidenceRefs:         []string{"workspace-encryption:host-managed"},
		},
	}
}

func unsupportedAccelerationForPlatform(goos string) string {
	switch goos {
	case "linux":
		return AccelerationKindHVF
	case "darwin":
		return AccelerationKindWHPX
	case "windows":
		return AccelerationKindKVM
	default:
		return AccelerationKindNone
	}
}

func validLaunchContextForContractTests() LaunchContext {
	return LaunchContext{
		RunID:                "run-1",
		StageID:              "artifact_flow",
		RoleInstanceID:       "workspace-1",
		SessionID:            "session-1",
		SessionNonce:         "nonce-0123456789abcdef",
		ActiveManifestHashes: []string{testDigest("1")},
		PolicyDecisionRefs:   []string{testDigest("2")},
		ApprovedArtifactRefs: []string{testDigest("3")},
	}
}

func validHostHelloForContractTests(ctx LaunchContext) HostHello {
	return HostHello{
		RunID:               ctx.RunID,
		StageID:             ctx.StageID,
		RoleInstanceID:      ctx.RoleInstanceID,
		IsolateID:           "isolate-1",
		SessionID:           ctx.SessionID,
		SessionNonce:        ctx.SessionNonce,
		LaunchContextDigest: ctx.LaunchContextDigest,
		TransportKind:       TransportKindVSock,
		TransportRequirements: SessionTransportRequirements{
			MutualAuthenticationRequired: true,
			EncryptionRequired:           true,
			ReplayProtectionRequired:     true,
		},
		Framing:       SessionFramingContract{FrameFormat: SessionFramingLengthPrefixedV1, MaxFrameBytes: 4096, MaxHandshakeMessageBytes: 2048},
		HostingNodeID: "node-1",
	}
}

func validIsolateHelloForContractTests(host HostHello) IsolateHello {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	keyIDValue := sha256Hex(publicKey)
	hello := IsolateHello{
		RunID:               host.RunID,
		IsolateID:           host.IsolateID,
		SessionID:           host.SessionID,
		SessionNonce:        host.SessionNonce,
		LaunchContextDigest: host.LaunchContextDigest,
		IsolateSessionKey: IsolateSessionKey{
			Alg:               "ed25519",
			KeyID:             "key_sha256",
			KeyIDValue:        keyIDValue,
			PublicKeyEncoding: "base64",
			PublicKey:         base64.StdEncoding.EncodeToString(publicKey),
			KeyOrigin:         SessionKeyOriginIsolateBoundaryEphemeral,
		},
		ProofOfPossession: SessionKeyProof{Alg: "ed25519", KeyID: "key_sha256", KeyIDValue: keyIDValue},
	}
	transcript, err := HandshakeTranscriptHash(host, hello)
	if err != nil {
		panic(err)
	}
	hello.HandshakeTranscriptHash = transcript
	payload, err := sessionProofPayload(host, hello, transcript)
	if err != nil {
		panic(err)
	}
	hello.ProofOfPossession.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	return hello
}

func validSessionReadyForContractTests(host HostHello, isolate IsolateHello) SessionReady {
	return SessionReady{
		RunID: host.RunID, IsolateID: host.IsolateID, SessionID: host.SessionID, SessionNonce: host.SessionNonce,
		ProvisioningMode: ProvisioningPostureTOFU, IdentityBindingPosture: ProvisioningPostureTOFU,
		IsolateKeyIDValue: isolate.IsolateSessionKey.KeyIDValue, ChannelKeyMode: SessionChannelKeyModeDistinct,
		MutuallyAuthenticated: true, Encrypted: true, ProofOfPossessionVerified: true,
	}
}
