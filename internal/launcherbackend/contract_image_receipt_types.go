package launcherbackend

const (
	BootProfileMicroVMLinuxKernelInitrdV1 = "microvm-linux-kernel-initrd-v1"
	BootProfileContainerOCIImageV1        = "container-oci-image-v1"

	RuntimeImageSignedPayloadSchemaID       = "runecode.protocol.v0.RuntimeImageSignedPayload"
	RuntimeImageSignedPayloadSchemaVersion  = "0.1.0"
	RuntimeToolchainDescriptorSchemaID      = "runecode.protocol.v0.RuntimeToolchainDescriptor"
	RuntimeToolchainDescriptorSchemaVersion = "0.1.0"
)

type RuntimeImageDescriptor struct {
	DescriptorDigest      string                       `json:"descriptor_digest"`
	BackendKind           string                       `json:"backend_kind"`
	PlatformCompatibility RuntimeImagePlatformCompat   `json:"platform_compatibility"`
	BootContractVersion   string                       `json:"boot_contract_version"`
	ComponentDigests      map[string]string            `json:"component_digests"`
	Signing               *RuntimeImageSigningHooks    `json:"signing,omitempty"`
	Attestation           *RuntimeImageAttestationHook `json:"attestation,omitempty"`
}

type RuntimeImagePlatformCompat struct {
	OS               string `json:"os"`
	Architecture     string `json:"architecture"`
	AccelerationKind string `json:"acceleration_kind,omitempty"`
}

type RuntimeImageSigningHooks struct {
	PayloadSchemaID      string                         `json:"payload_schema_id,omitempty"`
	PayloadSchemaVersion string                         `json:"payload_schema_version,omitempty"`
	PayloadDigest        string                         `json:"payload_digest,omitempty"`
	SignerRef            string                         `json:"signer_ref"`
	SignatureDigest      string                         `json:"signature_digest"`
	SignatureBundleRef   string                         `json:"signature_bundle_ref,omitempty"`
	VerifierSetRef       string                         `json:"verifier_set_ref,omitempty"`
	Publication          *RuntimeAssetPublicationBundle `json:"publication,omitempty"`
	Toolchain            *RuntimeToolchainSigningHooks  `json:"toolchain,omitempty"`
}

type RuntimeAssetPublicationBundle struct {
	DescriptorEnvelopeDigest  string `json:"descriptor_envelope_digest,omitempty"`
	ComponentBundleDigest     string `json:"component_bundle_digest,omitempty"`
	PublicationManifestDigest string `json:"publication_manifest_digest,omitempty"`
}

type RuntimeToolchainSigningHooks struct {
	DescriptorSchemaID      string `json:"descriptor_schema_id,omitempty"`
	DescriptorSchemaVersion string `json:"descriptor_schema_version,omitempty"`
	DescriptorDigest        string `json:"descriptor_digest"`
	SignerRef               string `json:"signer_ref"`
	SignatureDigest         string `json:"signature_digest"`
	SignatureBundleRef      string `json:"signature_bundle_ref,omitempty"`
	VerifierSetRef          string `json:"verifier_set_ref,omitempty"`
	BundleDigest            string `json:"bundle_digest,omitempty"`
}

type RuntimeImageAttestationHook struct {
	MeasurementProfile         string   `json:"measurement_profile,omitempty"`
	ExpectedMeasurementDigests []string `json:"expected_measurement_digests,omitempty"`
}

type BackendLaunchReceipt struct {
	RunID                            string                      `json:"run_id"`
	StageID                          string                      `json:"stage_id"`
	RoleInstanceID                   string                      `json:"role_instance_id"`
	RoleFamily                       string                      `json:"role_family,omitempty"`
	RoleKind                         string                      `json:"role_kind,omitempty"`
	BackendKind                      string                      `json:"backend_kind"`
	IsolationAssuranceLevel          string                      `json:"isolation_assurance_level"`
	ProvisioningPosture              string                      `json:"provisioning_posture,omitempty"`
	IsolateID                        string                      `json:"isolate_id,omitempty"`
	SessionID                        string                      `json:"session_id,omitempty"`
	SessionNonce                     string                      `json:"session_nonce,omitempty"`
	LaunchContextDigest              string                      `json:"launch_context_digest,omitempty"`
	HandshakeTranscriptHash          string                      `json:"handshake_transcript_hash,omitempty"`
	IsolateSessionKeyIDValue         string                      `json:"isolate_session_key_id_value,omitempty"`
	HostingNodeID                    string                      `json:"hosting_node_id,omitempty"`
	SessionSecurity                  *SessionSecurityPosture     `json:"session_security,omitempty"`
	ProvisioningPostureDegraded      bool                        `json:"provisioning_posture_degraded,omitempty"`
	ProvisioningDegradedReasons      []string                    `json:"provisioning_degraded_reasons,omitempty"`
	HypervisorImplementation         string                      `json:"hypervisor_implementation,omitempty"`
	AccelerationKind                 string                      `json:"acceleration_kind,omitempty"`
	TransportKind                    string                      `json:"transport_kind,omitempty"`
	QEMUProvenance                   *QEMUProvenance             `json:"qemu_provenance,omitempty"`
	RuntimeImageDescriptorDigest     string                      `json:"runtime_image_descriptor_digest,omitempty"`
	RuntimeImageBootProfile          string                      `json:"runtime_image_boot_profile,omitempty"`
	RuntimeImageSignerRef            string                      `json:"runtime_image_signer_ref,omitempty"`
	RuntimeImageVerifierRef          string                      `json:"runtime_image_verifier_ref,omitempty"`
	RuntimeImageSignatureDigest      string                      `json:"runtime_image_signature_digest,omitempty"`
	RuntimeToolchainDescriptorDigest string                      `json:"runtime_toolchain_descriptor_digest,omitempty"`
	RuntimeToolchainSignerRef        string                      `json:"runtime_toolchain_signer_ref,omitempty"`
	RuntimeToolchainVerifierRef      string                      `json:"runtime_toolchain_verifier_ref,omitempty"`
	RuntimeToolchainSignatureDigest  string                      `json:"runtime_toolchain_signature_digest,omitempty"`
	AuthorityStateDigest             string                      `json:"authority_state_digest,omitempty"`
	AuthorityStateRevision           uint64                      `json:"authority_state_revision,omitempty"`
	RuntimeImageDigest               string                      `json:"runtime_image_digest,omitempty"`
	BootComponentDigestByName        map[string]string           `json:"boot_component_digest_by_name,omitempty"`
	BootComponentDigests             []string                    `json:"boot_component_digests,omitempty"`
	ResourceLimits                   *BackendResourceLimits      `json:"resource_limits,omitempty"`
	WatchdogPolicy                   *BackendWatchdogPolicy      `json:"watchdog_policy,omitempty"`
	Lifecycle                        *BackendLifecycleSnapshot   `json:"backend_lifecycle,omitempty"`
	CachePosture                     *BackendCachePosture        `json:"cache_posture,omitempty"`
	CacheEvidence                    *BackendCacheEvidence       `json:"cache_evidence,omitempty"`
	AttachmentPlanSummary            *AttachmentPlanSummary      `json:"attachment_plan_summary,omitempty"`
	WorkspaceEncryptionPosture       *WorkspaceEncryptionPosture `json:"workspace_encryption_posture,omitempty"`
	LaunchFailureReasonCode          string                      `json:"launch_failure_reason_code,omitempty"`
}

type QEMUProvenance struct {
	Version       string `json:"version"`
	BuildIdentity string `json:"build_identity,omitempty"`
}

type AppliedHardeningPosture struct {
	Requested                 string   `json:"requested"`
	Effective                 string   `json:"effective"`
	DegradedReasons           []string `json:"degraded_reasons,omitempty"`
	ExecutionIdentityPosture  string   `json:"execution_identity_posture,omitempty"`
	RootlessPosture           string   `json:"rootless_posture,omitempty"`
	FilesystemExposurePosture string   `json:"filesystem_exposure_posture,omitempty"`
	WritableLayersPosture     string   `json:"writable_layers_posture,omitempty"`
	NetworkExposurePosture    string   `json:"network_exposure_posture,omitempty"`
	NetworkNamespacePosture   string   `json:"network_namespace_posture,omitempty"`
	NetworkDefaultPosture     string   `json:"network_default_posture,omitempty"`
	EgressEnforcementPosture  string   `json:"egress_enforcement_posture,omitempty"`
	SyscallFilteringPosture   string   `json:"syscall_filtering_posture,omitempty"`
	CapabilitiesPosture       string   `json:"capabilities_posture,omitempty"`
	DeviceSurfacePosture      string   `json:"device_surface_posture,omitempty"`
	ControlChannelKind        string   `json:"control_channel_kind,omitempty"`
	AccelerationKind          string   `json:"acceleration_kind,omitempty"`
	BackendEvidenceRefs       []string `json:"backend_evidence_refs,omitempty"`
}

type BackendTerminalReport struct {
	RunID             string `json:"run_id"`
	StageID           string `json:"stage_id"`
	RoleInstanceID    string `json:"role_instance_id"`
	IsolateID         string `json:"isolate_id,omitempty"`
	SessionID         string `json:"session_id,omitempty"`
	TerminationKind   string `json:"termination_kind"`
	FailureReasonCode string `json:"failure_reason_code,omitempty"`
	FailClosed        bool   `json:"fail_closed"`
	FallbackPosture   string `json:"fallback_posture,omitempty"`
	TerminatedAt      string `json:"terminated_at,omitempty"`
}

type RuntimeFactsSnapshot struct {
	LaunchReceipt    BackendLaunchReceipt    `json:"launch_receipt"`
	HardeningPosture AppliedHardeningPosture `json:"hardening_posture"`
	TerminalReport   *BackendTerminalReport  `json:"terminal_report,omitempty"`
}
