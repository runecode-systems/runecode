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
	default:
		return ProvisioningPostureUnknown
	}
}

func normalizeHypervisorImplementation(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HypervisorImplementationQEMU:
		return HypervisorImplementationQEMU
	default:
		return HypervisorImplementationUnknown
	}
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
	default:
		return AccelerationKindUnknown
	}
}

func normalizeTransportKind(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case TransportKindVSock:
		return TransportKindVSock
	case TransportKindVirtioSerial:
		return TransportKindVirtioSerial
	default:
		return TransportKindUnknown
	}
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
	case AttachmentChannelVirtualDisk:
		return AttachmentChannelVirtualDisk
	case AttachmentChannelReadOnlyChannel:
		return AttachmentChannelReadOnlyChannel
	case AttachmentChannelEphemeralDisk:
		return AttachmentChannelEphemeralDisk
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
