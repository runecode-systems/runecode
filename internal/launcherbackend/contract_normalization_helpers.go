package launcherbackend

import "strings"

func normalizeBackendKind(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case BackendKindMicroVM:
		return BackendKindMicroVM
	case BackendKindContainer:
		return BackendKindContainer
	default:
		return BackendKindUnknown
	}
}

func normalizeIsolationAssuranceLevel(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case IsolationAssuranceIsolated:
		return IsolationAssuranceIsolated
	case IsolationAssuranceDegraded:
		return IsolationAssuranceDegraded
	default:
		return IsolationAssuranceUnknown
	}
}

func normalizeProvisioningPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case ProvisioningPostureTOFU:
		return ProvisioningPostureTOFU
	case ProvisioningPostureAttested:
		return ProvisioningPostureAttested
	case ProvisioningPostureNotApplicable:
		return ProvisioningPostureNotApplicable
	default:
		return ProvisioningPostureUnknown
	}
}

func normalizeProvisioningPostureForBackend(value string, backendKind string) string {
	normalized := normalizeProvisioningPosture(value)
	if normalized != ProvisioningPostureUnknown {
		return normalized
	}
	if normalizeBackendKind(backendKind) == BackendKindContainer {
		return ProvisioningPostureNotApplicable
	}
	return ProvisioningPostureUnknown
}

func normalizeHypervisorImplementation(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HypervisorImplementationQEMU:
		return HypervisorImplementationQEMU
	case HypervisorImplementationNotApplicable:
		return HypervisorImplementationNotApplicable
	default:
		return HypervisorImplementationUnknown
	}
}

func normalizeHypervisorImplementationForBackend(value string, backendKind string) string {
	normalized := normalizeHypervisorImplementation(value)
	if normalized != HypervisorImplementationUnknown {
		return normalized
	}
	if normalizeBackendKind(backendKind) == BackendKindContainer {
		return HypervisorImplementationNotApplicable
	}
	return HypervisorImplementationUnknown
}

func normalizeAccelerationKind(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case AccelerationKindKVM:
		return AccelerationKindKVM
	case AccelerationKindHVF:
		return AccelerationKindHVF
	case AccelerationKindWHPX:
		return AccelerationKindWHPX
	case AccelerationKindNone:
		return AccelerationKindNone
	case AccelerationKindNotApplicable:
		return AccelerationKindNotApplicable
	default:
		return AccelerationKindUnknown
	}
}

func normalizeAccelerationKindForBackend(value string, backendKind string) string {
	normalized := normalizeAccelerationKind(value)
	if normalized != AccelerationKindUnknown {
		return normalized
	}
	if normalizeBackendKind(backendKind) == BackendKindContainer {
		return AccelerationKindNotApplicable
	}
	return AccelerationKindUnknown
}

func normalizeTransportKind(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case TransportKindVSock:
		return TransportKindVSock
	case TransportKindVirtioSerial:
		return TransportKindVirtioSerial
	case TransportKindNotApplicable:
		return TransportKindNotApplicable
	default:
		return TransportKindUnknown
	}
}

func normalizeTransportKindForBackend(value string, backendKind string) string {
	normalized := normalizeTransportKind(value)
	if normalized != TransportKindUnknown {
		return normalized
	}
	if normalizeBackendKind(backendKind) == BackendKindContainer {
		return TransportKindNotApplicable
	}
	return TransportKindUnknown
}

func normalizeBackendLifecycleState(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case BackendLifecycleStatePlanned:
		return BackendLifecycleStatePlanned
	case BackendLifecycleStateLaunching:
		return BackendLifecycleStateLaunching
	case BackendLifecycleStateStarted:
		return BackendLifecycleStateStarted
	case BackendLifecycleStateBinding:
		return BackendLifecycleStateBinding
	case BackendLifecycleStateActive:
		return BackendLifecycleStateActive
	case BackendLifecycleStateTerminating:
		return BackendLifecycleStateTerminating
	case BackendLifecycleStateTerminated:
		return BackendLifecycleStateTerminated
	default:
		return ""
	}
}

func normalizeBackendTerminationKind(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case BackendTerminationKindCompleted:
		return BackendTerminationKindCompleted
	case BackendTerminationKindFailed:
		return BackendTerminationKindFailed
	default:
		return BackendTerminationKindUnknown
	}
}

func normalizeBackendFallbackPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case BackendFallbackPostureNoAutomaticFallback:
		return BackendFallbackPostureNoAutomaticFallback
	case BackendFallbackPostureContainerOptInOnly:
		return BackendFallbackPostureContainerOptInOnly
	default:
		return ""
	}
}

func normalizeBackendErrorCode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case BackendErrorCodeAccelerationUnavailable:
		return BackendErrorCodeAccelerationUnavailable
	case BackendErrorCodeHypervisorLaunchFailed:
		return BackendErrorCodeHypervisorLaunchFailed
	case BackendErrorCodeImageDescriptorSignatureMismatch:
		return BackendErrorCodeImageDescriptorSignatureMismatch
	case BackendErrorCodeAttachmentPlanInvalid:
		return BackendErrorCodeAttachmentPlanInvalid
	case BackendErrorCodeHandshakeFailed:
		return BackendErrorCodeHandshakeFailed
	case BackendErrorCodeReplayDetected:
		return BackendErrorCodeReplayDetected
	case BackendErrorCodeSessionBindingMismatch:
		return BackendErrorCodeSessionBindingMismatch
	case BackendErrorCodeGuestUnresponsive:
		return BackendErrorCodeGuestUnresponsive
	case BackendErrorCodeWatchdogTimeout:
		return BackendErrorCodeWatchdogTimeout
	case BackendErrorCodeRequiredHardeningUnavailable:
		return BackendErrorCodeRequiredHardeningUnavailable
	case BackendErrorCodeRequiredDiskEncryptionUnavailable:
		return BackendErrorCodeRequiredDiskEncryptionUnavailable
	case BackendErrorCodeContainerAutomaticFallbackDisallowed:
		return BackendErrorCodeContainerAutomaticFallbackDisallowed
	case BackendErrorCodeContainerOptInRequired:
		return BackendErrorCodeContainerOptInRequired
	case BackendErrorCodeTerminalReportInvalid:
		return BackendErrorCodeTerminalReportInvalid
	default:
		return ""
	}
}

func normalizeCacheResult(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case CacheResultHit:
		return CacheResultHit
	case CacheResultMiss:
		return CacheResultMiss
	case CacheResultBypass:
		return CacheResultBypass
	default:
		return ""
	}
}

func normalizeAttachmentChannelKind(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case AttachmentChannelArtifactImage:
		return AttachmentChannelArtifactImage
	case AttachmentChannelReadOnlyVolume, "read_only_channel":
		return AttachmentChannelReadOnlyVolume
	case AttachmentChannelWritableVolume, "virtual_disk":
		return AttachmentChannelWritableVolume
	case AttachmentChannelEphemeralVolume, "ephemeral_disk":
		return AttachmentChannelEphemeralVolume
	default:
		return ""
	}
}

func normalizeWorkspaceAtRestProtection(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case WorkspaceAtRestProtectionHostManagedEncryption:
		return WorkspaceAtRestProtectionHostManagedEncryption
	default:
		return WorkspaceAtRestProtectionUnknown
	}
}

func normalizeWorkspaceKeyProtectionPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case WorkspaceKeyProtectionHardwareBacked:
		return WorkspaceKeyProtectionHardwareBacked
	case WorkspaceKeyProtectionOSKeystore:
		return WorkspaceKeyProtectionOSKeystore
	case WorkspaceKeyProtectionExplicitDevOptIn:
		return WorkspaceKeyProtectionExplicitDevOptIn
	default:
		return WorkspaceKeyProtectionUnknown
	}
}
