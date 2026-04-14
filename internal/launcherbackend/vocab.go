package launcherbackend

import "regexp"

const (
	BackendKindMicroVM   = "microvm"
	BackendKindContainer = "container"
	BackendKindUnknown   = "unknown"

	IsolationAssuranceIsolated      = "isolated"
	IsolationAssuranceDegraded      = "degraded"
	IsolationAssuranceUnknown       = "unknown"
	IsolationAssuranceNotApplicable = "not_applicable"

	ProvisioningPostureTOFU          = "tofu"
	ProvisioningPostureAttested      = "attested"
	ProvisioningPostureUnknown       = "unknown"
	ProvisioningPostureNotApplicable = "not_applicable"

	AttachmentRoleLaunchContext  = "launch_context"
	AttachmentRoleWorkspace      = "workspace"
	AttachmentRoleInputArtifacts = "input_artifacts"
	AttachmentRoleScratch        = "scratch"

	AttachmentChannelArtifactImage   = "artifact_image"
	AttachmentChannelReadOnlyVolume  = "read_only_volume"
	AttachmentChannelWritableVolume  = "writable_volume"
	AttachmentChannelEphemeralVolume = "ephemeral_volume"

	// Deprecated compatibility aliases.
	//
	// Keep these aliases while persisted runtime facts can still include legacy
	// values from the microVM-only vocabulary.
	AttachmentChannelVirtualDisk     = AttachmentChannelWritableVolume
	AttachmentChannelReadOnlyChannel = AttachmentChannelReadOnlyVolume
	AttachmentChannelEphemeralDisk   = AttachmentChannelEphemeralVolume

	WorkspaceAtRestProtectionHostManagedEncryption = "host_managed_encryption"
	WorkspaceAtRestProtectionUnknown               = "unknown"

	WorkspaceKeyProtectionHardwareBacked   = "hardware_backed"
	WorkspaceKeyProtectionOSKeystore       = "os_keystore"
	WorkspaceKeyProtectionExplicitDevOptIn = "explicit_dev_opt_in"
	WorkspaceKeyProtectionUnknown          = "unknown"

	HypervisorImplementationQEMU          = "qemu"
	HypervisorImplementationUnknown       = "unknown"
	HypervisorImplementationNotApplicable = "not_applicable"

	AccelerationKindKVM           = "kvm"
	AccelerationKindHVF           = "hvf"
	AccelerationKindWHPX          = "whpx"
	AccelerationKindNone          = "none"
	AccelerationKindUnknown       = "unknown"
	AccelerationKindNotApplicable = "not_applicable"

	TransportKindVSock         = "vsock"
	TransportKindVirtioSerial  = "virtio-serial"
	TransportKindUnknown       = "unknown"
	TransportKindNotApplicable = "not_applicable"

	SessionFramingLengthPrefixedV1 = "length_prefixed_v1"

	SessionChannelKeyModeDistinct = "distinct_from_isolate_identity"

	SessionKeyOriginIsolateBoundaryEphemeral = "isolate_boundary_ephemeral"

	SessionMaxFrameBytesHardLimit            = 1024 * 1024
	SessionMaxHandshakeMessageBytesHardLimit = 64 * 1024

	HardeningRequestedHardened = "hardened"
	HardeningRequestedUnknown  = "unknown"
	HardeningRequestedNone     = "none"

	HardeningEffectiveHardened = "hardened"
	HardeningEffectiveDegraded = "degraded"
	HardeningEffectiveUnknown  = "unknown"
	HardeningEffectiveNone     = "none"

	HardeningExecutionIdentityUnprivileged = "unprivileged"
	HardeningExecutionIdentityUnknown      = "unknown"
	HardeningExecutionIdentityNone         = "none"

	HardeningFilesystemExposureRestricted = "restricted"
	HardeningFilesystemExposureBroad      = "broad"
	HardeningFilesystemExposureUnknown    = "unknown"

	HardeningNetworkExposureNone       = "none"
	HardeningNetworkExposureRestricted = "restricted"
	HardeningNetworkExposureOpen       = "open"
	HardeningNetworkExposureUnknown    = "unknown"

	HardeningSyscallFilteringSeccomp = "seccomp"
	HardeningSyscallFilteringNone    = "none"
	HardeningSyscallFilteringUnknown = "unknown"

	HardeningDeviceSurfaceAllowlist = "allowlist"
	HardeningDeviceSurfaceBroad     = "broad"
	HardeningDeviceSurfaceUnknown   = "unknown"

	HardeningRootlessEnabled    = "enabled"
	HardeningRootlessDisabled   = "disabled"
	HardeningRootlessUnknown    = "unknown"
	HardeningRootlessBestEffort = "best_effort"

	HardeningCapabilitiesDropped = "dropped"
	HardeningCapabilitiesBroad   = "broad"
	HardeningCapabilitiesUnknown = "unknown"

	HardeningWritableLayersEphemeral  = "ephemeral_only"
	HardeningWritableLayersPersistent = "persistent"
	HardeningWritableLayersUnknown    = "unknown"

	HardeningNetworkNamespacePerRole = "per_role"
	HardeningNetworkNamespaceShared  = "shared"
	HardeningNetworkNamespaceUnknown = "unknown"

	HardeningNetworkDefaultNone         = "none"
	HardeningNetworkDefaultLoopbackOnly = "loopback_only"
	HardeningNetworkDefaultEgress       = "egress"
	HardeningNetworkDefaultUnknown      = "unknown"

	HardeningEgressEnforcementHostLevel   = "host_level_allowlist"
	HardeningEgressEnforcementInContainer = "in_container"
	HardeningEgressEnforcementNone        = "none"
	HardeningEgressEnforcementUnknown     = "unknown"

	BackendLifecycleStatePlanned     = "planned"
	BackendLifecycleStateLaunching   = "launching"
	BackendLifecycleStateStarted     = "started"
	BackendLifecycleStateBinding     = "binding"
	BackendLifecycleStateActive      = "active"
	BackendLifecycleStateTerminating = "terminating"
	BackendLifecycleStateTerminated  = "terminated"

	CacheResultHit    = "hit"
	CacheResultMiss   = "miss"
	CacheResultBypass = "bypass"

	WatchdogTerminationReasonCodeTimeout = "watchdog_timeout"

	BackendErrorCodeAccelerationUnavailable              = "acceleration_unavailable"
	BackendErrorCodeHypervisorLaunchFailed               = "hypervisor_launch_failed"
	BackendErrorCodeImageDescriptorSignatureMismatch     = "image_descriptor_signature_mismatch"
	BackendErrorCodeAttachmentPlanInvalid                = "attachment_plan_invalid"
	BackendErrorCodeHandshakeFailed                      = "handshake_failed"
	BackendErrorCodeReplayDetected                       = "replay_detected"
	BackendErrorCodeSessionBindingMismatch               = "session_binding_mismatch"
	BackendErrorCodeGuestUnresponsive                    = "guest_unresponsive"
	BackendErrorCodeWatchdogTimeout                      = "watchdog_timeout"
	BackendErrorCodeRequiredHardeningUnavailable         = "required_hardening_unavailable"
	BackendErrorCodeRequiredDiskEncryptionUnavailable    = "required_disk_encryption_unavailable"
	BackendErrorCodeContainerAutomaticFallbackDisallowed = "container_automatic_fallback_disallowed"
	BackendErrorCodeContainerOptInRequired               = "container_opt_in_required"
	BackendErrorCodeTerminalReportInvalid                = "terminal_report_invalid"

	BackendTerminationKindCompleted = "completed"
	BackendTerminationKindFailed    = "failed"
	BackendTerminationKindUnknown   = "unknown"

	BackendFallbackPostureNoAutomaticFallback = "no_automatic_fallback"
	BackendFallbackPostureContainerOptInOnly  = "container_opt_in_required"
)

var roleTokenPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var deviceNumberingPattern = regexp.MustCompile(`\b(vd[a-z][0-9]*|sd[a-z][0-9]*|xvd[a-z][0-9]*|nvme[0-9]+n[0-9]+(p[0-9]+)?)\b`)
