package brokerapi

import "github.com/runecode-ai/runecode/internal/launcherbackend"

func appendContainerRoleScopeReason(reasons []string, roleFamily string) []string {
	if roleFamily != "workspace" {
		return append(reasons, "container_role_family_not_supported_v0")
	}
	return reasons
}

func appendContainerRootlessReasons(reasons []string, posture string) []string {
	switch posture {
	case launcherbackend.HardeningRootlessEnabled:
		return reasons
	case launcherbackend.HardeningRootlessBestEffort:
		return append(reasons, "rootless_best_effort_only")
	case launcherbackend.HardeningRootlessDisabled:
		return append(reasons, "rootless_not_enabled")
	default:
		return append(reasons, "rootless_posture_unknown")
	}
}

func appendContainerKernelAndFsReasons(reasons []string, posture launcherbackend.AppliedHardeningPosture) []string {
	if posture.SyscallFilteringPosture != launcherbackend.HardeningSyscallFilteringSeccomp {
		reasons = append(reasons, "seccomp_required")
	}
	if posture.CapabilitiesPosture != launcherbackend.HardeningCapabilitiesDropped {
		reasons = append(reasons, "capabilities_drop_required")
	}
	if posture.FilesystemExposurePosture != launcherbackend.HardeningFilesystemExposureRestricted {
		reasons = append(reasons, "filesystem_exposure_restricted_required")
	}
	if posture.WritableLayersPosture != launcherbackend.HardeningWritableLayersEphemeral {
		reasons = append(reasons, "writable_layers_ephemeral_required")
	}
	return reasons
}

func appendContainerNetworkReasons(reasons []string, posture launcherbackend.AppliedHardeningPosture) []string {
	reasons = appendContainerNetworkNamespaceReason(reasons, posture.NetworkNamespacePosture)
	reasons = appendContainerNetworkDefaultReason(reasons, posture.NetworkDefaultPosture)
	reasons = appendContainerNetworkExposureReason(reasons, posture.NetworkExposurePosture)
	if posture.NetworkExposurePosture == launcherbackend.HardeningNetworkExposureRestricted {
		reasons = appendContainerEgressReason(reasons, posture.EgressEnforcementPosture)
	}
	return reasons
}

func appendContainerNetworkNamespaceReason(reasons []string, posture string) []string {
	switch posture {
	case launcherbackend.HardeningNetworkNamespacePerRole:
		return reasons
	case launcherbackend.HardeningNetworkNamespaceShared:
		return append(reasons, "network_namespace_shared")
	default:
		return append(reasons, "network_namespace_posture_unknown")
	}
}

func appendContainerNetworkDefaultReason(reasons []string, posture string) []string {
	if posture == launcherbackend.HardeningNetworkDefaultNone || posture == launcherbackend.HardeningNetworkDefaultLoopbackOnly {
		return reasons
	}
	return append(reasons, "workspace_network_default_must_be_none_or_loopback")
}

func appendContainerNetworkExposureReason(reasons []string, posture string) []string {
	if posture == launcherbackend.HardeningNetworkExposureNone {
		return reasons
	}
	return append(reasons, "workspace_network_exposure_must_be_none")
}

func appendContainerEgressReason(reasons []string, posture string) []string {
	switch posture {
	case launcherbackend.HardeningEgressEnforcementHostLevel:
		return reasons
	case launcherbackend.HardeningEgressEnforcementInContainer:
		return append(reasons, "egress_enforcement_in_container")
	default:
		return append(reasons, "egress_host_level_enforcement_required")
	}
}
