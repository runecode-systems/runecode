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
