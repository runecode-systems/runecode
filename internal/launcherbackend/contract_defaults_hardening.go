package launcherbackend

import (
	"fmt"
	"strings"
)

func DefaultRuntimeFacts(runID string) RuntimeFactsSnapshot {
	return RuntimeFactsSnapshot{
		LaunchReceipt: BackendLaunchReceipt{
			RunID:                   runID,
			BackendKind:             BackendKindUnknown,
			IsolationAssuranceLevel: IsolationAssuranceUnknown,
			ProvisioningPosture:     ProvisioningPostureUnknown,
			Lifecycle: &BackendLifecycleSnapshot{
				CurrentState:          BackendLifecycleStatePlanned,
				TerminateBetweenSteps: true,
			},
			SessionSecurity: &SessionSecurityPosture{
				Degraded:        true,
				DegradedReasons: []string{"session_handshake_not_established"},
			},
		},
		HardeningPosture: DefaultAppliedHardeningPosture(),
	}
}

func DefaultAppliedHardeningPosture() AppliedHardeningPosture {
	return AppliedHardeningPosture{
		Requested:                 HardeningRequestedUnknown,
		Effective:                 HardeningEffectiveDegraded,
		DegradedReasons:           []string{"hardening_posture_unspecified"},
		ExecutionIdentityPosture:  HardeningExecutionIdentityUnknown,
		FilesystemExposurePosture: HardeningFilesystemExposureUnknown,
		NetworkExposurePosture:    HardeningNetworkExposureUnknown,
		SyscallFilteringPosture:   HardeningSyscallFilteringUnknown,
		DeviceSurfacePosture:      HardeningDeviceSurfaceUnknown,
		ControlChannelKind:        TransportKindUnknown,
		AccelerationKind:          AccelerationKindUnknown,
	}
}

func (p AppliedHardeningPosture) IsDegraded() bool {
	return normalizeHardeningEffective(p.Effective) == HardeningEffectiveDegraded || len(p.DegradedReasons) > 0
}

func (p AppliedHardeningPosture) Normalized() AppliedHardeningPosture {
	out := p
	out.Requested = normalizeHardeningRequested(out.Requested)
	out.Effective = normalizeHardeningEffective(out.Effective)
	out.ExecutionIdentityPosture = normalizeExecutionIdentityPosture(out.ExecutionIdentityPosture)
	out.FilesystemExposurePosture = normalizeFilesystemExposurePosture(out.FilesystemExposurePosture)
	out.NetworkExposurePosture = normalizeNetworkExposurePosture(out.NetworkExposurePosture)
	out.SyscallFilteringPosture = normalizeSyscallFilteringPosture(out.SyscallFilteringPosture)
	out.DeviceSurfacePosture = normalizeDeviceSurfacePosture(out.DeviceSurfacePosture)
	out.ControlChannelKind = normalizeTransportKind(out.ControlChannelKind)
	out.AccelerationKind = normalizeAccelerationKind(out.AccelerationKind)
	out.DegradedReasons = uniqueSortedStrings(out.DegradedReasons)
	out.BackendEvidenceRefs = uniqueSortedStrings(out.BackendEvidenceRefs)
	out.DegradedReasons = uniqueSortedStrings(append(out.DegradedReasons, hardeningDerivedDegradedReasons(out)...))
	if len(out.DegradedReasons) > 0 {
		out.Effective = HardeningEffectiveDegraded
	}
	return out
}

func (p AppliedHardeningPosture) Validate() error {
	if err := validateHardeningRequestedValue(p.Requested); err != nil {
		return err
	}
	if err := validateHardeningEffectiveValue(p.Effective); err != nil {
		return err
	}
	if err := validateHardeningEffectiveReasonConsistency(p.Effective, p.DegradedReasons); err != nil {
		return err
	}
	if err := validateHardeningReasonValues(p.DegradedReasons); err != nil {
		return err
	}
	return validateHardeningEvidenceRefs(p.BackendEvidenceRefs)
}

func hardeningDerivedDegradedReasons(posture AppliedHardeningPosture) []string {
	reasons := make([]string, 0, 16)
	checks := []struct {
		condition bool
		reason    string
	}{
		{posture.Requested == HardeningRequestedUnknown, "hardening_requested_unknown"},
		{posture.Requested == HardeningRequestedNone, "hardening_requested_none"},
		{posture.Effective == HardeningEffectiveUnknown, "hardening_effective_unknown"},
		{posture.Effective == HardeningEffectiveNone, "hardening_effective_none"},
		{posture.ExecutionIdentityPosture == HardeningExecutionIdentityUnknown, "execution_identity_posture_unknown"},
		{posture.ExecutionIdentityPosture == HardeningExecutionIdentityNone, "execution_identity_posture_none"},
		{posture.FilesystemExposurePosture == HardeningFilesystemExposureUnknown, "filesystem_exposure_posture_unknown"},
		{posture.FilesystemExposurePosture == HardeningFilesystemExposureBroad, "filesystem_exposure_posture_broad"},
		{posture.NetworkExposurePosture == HardeningNetworkExposureUnknown, "network_exposure_posture_unknown"},
		{posture.NetworkExposurePosture == HardeningNetworkExposureOpen, "network_exposure_posture_open"},
		{posture.SyscallFilteringPosture == HardeningSyscallFilteringUnknown, "syscall_filtering_posture_unknown"},
		{posture.SyscallFilteringPosture == HardeningSyscallFilteringNone, "syscall_filtering_posture_none"},
		{posture.DeviceSurfacePosture == HardeningDeviceSurfaceUnknown, "device_surface_posture_unknown"},
		{posture.DeviceSurfacePosture == HardeningDeviceSurfaceBroad, "device_surface_posture_broad"},
		{posture.ControlChannelKind == TransportKindUnknown, "control_channel_kind_unknown"},
		{posture.AccelerationKind == AccelerationKindUnknown, "acceleration_kind_unknown"},
		{posture.AccelerationKind == AccelerationKindNone, "acceleration_kind_none"},
	}
	for _, check := range checks {
		if check.condition {
			reasons = append(reasons, check.reason)
		}
	}
	return reasons
}

func validateHardeningRequestedValue(value string) error {
	if normalizeHardeningRequested(value) == HardeningRequestedUnknown && strings.TrimSpace(value) != "" && strings.TrimSpace(strings.ToLower(value)) != HardeningRequestedUnknown {
		return fmt.Errorf("requested must be one of %q, %q, or %q", HardeningRequestedHardened, HardeningRequestedNone, HardeningRequestedUnknown)
	}
	return nil
}

func validateHardeningEffectiveValue(value string) error {
	if normalizeHardeningEffective(value) == HardeningEffectiveUnknown && strings.TrimSpace(value) != "" && strings.TrimSpace(strings.ToLower(value)) != HardeningEffectiveUnknown {
		return fmt.Errorf("effective must be one of %q, %q, %q, or %q", HardeningEffectiveHardened, HardeningEffectiveDegraded, HardeningEffectiveNone, HardeningEffectiveUnknown)
	}
	return nil
}

func validateHardeningEffectiveReasonConsistency(effective string, degradedReasons []string) error {
	normalizedEffective := normalizeHardeningEffective(effective)
	if normalizedEffective == HardeningEffectiveDegraded && len(degradedReasons) == 0 {
		return fmt.Errorf("effective=degraded requires degraded_reasons")
	}
	if normalizedEffective == HardeningEffectiveHardened && len(degradedReasons) > 0 {
		return fmt.Errorf("effective=hardened cannot include degraded_reasons")
	}
	return nil
}

func validateHardeningReasonValues(reasons []string) error {
	for _, reason := range reasons {
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("degraded_reasons cannot contain empty values")
		}
		if looksLikeHostPath(reason) {
			return fmt.Errorf("degraded_reasons must not include host-local path material")
		}
	}
	return nil
}

func validateHardeningEvidenceRefs(refs []string) error {
	for _, ref := range refs {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("backend_evidence_refs cannot contain empty values")
		}
		if looksLikeHostPath(ref) {
			return fmt.Errorf("backend_evidence_refs must not include host-local path material")
		}
	}
	return nil
}
