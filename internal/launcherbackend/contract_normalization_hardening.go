package launcherbackend

import "strings"

func normalizeHardeningRequested(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningRequestedHardened:
		return HardeningRequestedHardened
	case HardeningRequestedNone:
		return HardeningRequestedNone
	default:
		return HardeningRequestedUnknown
	}
}

func normalizeHardeningEffective(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningEffectiveHardened:
		return HardeningEffectiveHardened
	case HardeningEffectiveDegraded:
		return HardeningEffectiveDegraded
	case HardeningEffectiveNone:
		return HardeningEffectiveNone
	default:
		return HardeningEffectiveUnknown
	}
}

func normalizeExecutionIdentityPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningExecutionIdentityUnprivileged:
		return HardeningExecutionIdentityUnprivileged
	case HardeningExecutionIdentityNone:
		return HardeningExecutionIdentityNone
	default:
		return HardeningExecutionIdentityUnknown
	}
}

func normalizeRootlessPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningRootlessEnabled:
		return HardeningRootlessEnabled
	case HardeningRootlessDisabled:
		return HardeningRootlessDisabled
	case HardeningRootlessBestEffort:
		return HardeningRootlessBestEffort
	default:
		return HardeningRootlessUnknown
	}
}

func normalizeFilesystemExposurePosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningFilesystemExposureRestricted:
		return HardeningFilesystemExposureRestricted
	case HardeningFilesystemExposureBroad:
		return HardeningFilesystemExposureBroad
	default:
		return HardeningFilesystemExposureUnknown
	}
}

func normalizeNetworkExposurePosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningNetworkExposureNone:
		return HardeningNetworkExposureNone
	case HardeningNetworkExposureRestricted:
		return HardeningNetworkExposureRestricted
	case HardeningNetworkExposureOpen:
		return HardeningNetworkExposureOpen
	default:
		return HardeningNetworkExposureUnknown
	}
}

func normalizeNetworkNamespacePosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningNetworkNamespacePerRole:
		return HardeningNetworkNamespacePerRole
	case HardeningNetworkNamespaceShared:
		return HardeningNetworkNamespaceShared
	default:
		return HardeningNetworkNamespaceUnknown
	}
}

func normalizeNetworkDefaultPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningNetworkDefaultNone:
		return HardeningNetworkDefaultNone
	case HardeningNetworkDefaultLoopbackOnly:
		return HardeningNetworkDefaultLoopbackOnly
	case HardeningNetworkDefaultEgress:
		return HardeningNetworkDefaultEgress
	default:
		return HardeningNetworkDefaultUnknown
	}
}

func normalizeEgressEnforcementPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningEgressEnforcementHostLevel:
		return HardeningEgressEnforcementHostLevel
	case HardeningEgressEnforcementInContainer:
		return HardeningEgressEnforcementInContainer
	case HardeningEgressEnforcementNone:
		return HardeningEgressEnforcementNone
	default:
		return HardeningEgressEnforcementUnknown
	}
}

func normalizeSyscallFilteringPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningSyscallFilteringSeccomp:
		return HardeningSyscallFilteringSeccomp
	case HardeningSyscallFilteringNone:
		return HardeningSyscallFilteringNone
	default:
		return HardeningSyscallFilteringUnknown
	}
}

func normalizeCapabilitiesPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningCapabilitiesDropped:
		return HardeningCapabilitiesDropped
	case HardeningCapabilitiesBroad:
		return HardeningCapabilitiesBroad
	default:
		return HardeningCapabilitiesUnknown
	}
}

func normalizeWritableLayersPosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningWritableLayersEphemeral:
		return HardeningWritableLayersEphemeral
	case HardeningWritableLayersPersistent:
		return HardeningWritableLayersPersistent
	default:
		return HardeningWritableLayersUnknown
	}
}

func normalizeDeviceSurfacePosture(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case HardeningDeviceSurfaceAllowlist:
		return HardeningDeviceSurfaceAllowlist
	case HardeningDeviceSurfaceBroad:
		return HardeningDeviceSurfaceBroad
	default:
		return HardeningDeviceSurfaceUnknown
	}
}
