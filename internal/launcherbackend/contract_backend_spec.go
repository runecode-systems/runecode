package launcherbackend

import (
	"fmt"
	"runtime"
	"strings"
)

func (s BackendLaunchSpec) Validate() error {
	if err := validateBackendLaunchSpecIdentity(s); err != nil {
		return err
	}
	if err := validateBackendLaunchSpecBackendSelection(s); err != nil {
		return err
	}
	if err := validateBackendLaunchSpecImageAndSigning(s); err != nil {
		return err
	}
	if err := validateBackendLaunchSpecRuntimeContracts(s); err != nil {
		return err
	}
	return nil
}

func validateBackendLaunchSpecIdentity(spec BackendLaunchSpec) error {
	if strings.TrimSpace(spec.RunID) == "" || strings.TrimSpace(spec.StageID) == "" {
		return fmt.Errorf("run_id and stage_id are required")
	}
	if err := validateSingleRoleToken("role_instance_id", spec.RoleInstanceID); err != nil {
		return err
	}
	if err := validateSingleRoleToken("role_family", spec.RoleFamily); err != nil {
		return err
	}
	return validateSingleRoleToken("role_kind", spec.RoleKind)
}

func validateBackendLaunchSpecBackendSelection(spec BackendLaunchSpec) error {
	if normalizeBackendKind(spec.RequestedBackend) == BackendKindUnknown {
		return fmt.Errorf("requested_backend must be one of %q or %q", BackendKindMicroVM, BackendKindContainer)
	}
	if strings.TrimSpace(spec.RequestedBackend) != "" && normalizeBackendKind(spec.RequestedBackend) != normalizeBackendKind(spec.Image.BackendKind) {
		return fmt.Errorf("requested_backend %q does not match runtime_image.backend_kind %q", spec.RequestedBackend, spec.Image.BackendKind)
	}
	if err := validateAccelerationForPlatform(spec.RequestedAccelerationKind, spec.RequestedBackend); err != nil {
		return err
	}
	return validateControlTransportKind(spec.ControlTransportKind, spec.RequestedBackend)
}

func validateBackendLaunchSpecImageAndSigning(spec BackendLaunchSpec) error {
	if err := spec.Image.Validate(); err != nil {
		return fmt.Errorf("runtime_image: %w", err)
	}
	if spec.Image.Signing == nil {
		return fmt.Errorf("runtime_image.signing is required for descriptor-pinned launch contract")
	}
	if strings.TrimSpace(spec.Image.Signing.SignerRef) == "" || strings.TrimSpace(spec.Image.Signing.SignatureDigest) == "" {
		return fmt.Errorf("runtime_image.signing.signer_ref and runtime_image.signing.signature_digest are required")
	}
	return nil
}

func validateBackendLaunchSpecRuntimeContracts(spec BackendLaunchSpec) error {
	if err := spec.Attachments.Validate(); err != nil {
		return fmt.Errorf("attachments: %w", err)
	}
	if err := spec.ResourceLimits.Validate(); err != nil {
		return fmt.Errorf("resource_limits: %w", err)
	}
	if err := spec.WatchdogPolicy.Validate(); err != nil {
		return fmt.Errorf("watchdog_policy: %w", err)
	}
	if err := spec.LifecyclePolicy.Validate(); err != nil {
		return fmt.Errorf("lifecycle_policy: %w", err)
	}
	if err := spec.CachePosture.Validate(); err != nil {
		return fmt.Errorf("cache_posture: %w", err)
	}
	return nil
}

func (l BackendResourceLimits) Validate() error {
	if err := validateResourceCapacity(l); err != nil {
		return err
	}
	if err := validateResourceTimeouts(l); err != nil {
		return err
	}
	return validateTerminationGrace(l.TerminationGraceSeconds)
}

func validateResourceCapacity(l BackendResourceLimits) error {
	if l.VCPUCount < 1 || l.VCPUCount > 64 {
		return fmt.Errorf("vcpu_count must be between 1 and 64")
	}
	if l.MemoryMiB < 128 || l.MemoryMiB > 1048576 {
		return fmt.Errorf("memory_mib must be between 128 and 1048576")
	}
	if l.DiskMiB < 64 || l.DiskMiB > 10485760 {
		return fmt.Errorf("disk_mib must be between 64 and 10485760")
	}
	return nil
}

func validateResourceTimeouts(l BackendResourceLimits) error {
	if l.LaunchTimeoutSeconds < 1 || l.LaunchTimeoutSeconds > 3600 {
		return fmt.Errorf("launch_timeout_seconds must be between 1 and 3600")
	}
	if l.BindTimeoutSeconds < 1 || l.BindTimeoutSeconds > 3600 {
		return fmt.Errorf("bind_timeout_seconds must be between 1 and 3600")
	}
	if l.ActiveTimeoutSeconds < 1 || l.ActiveTimeoutSeconds > 86400 {
		return fmt.Errorf("active_timeout_seconds must be between 1 and 86400")
	}
	return nil
}

func validateTerminationGrace(seconds int) error {
	if seconds < 0 || seconds > 600 {
		return fmt.Errorf("termination_grace_seconds must be between 0 and 600")
	}
	return nil
}

func (p BackendWatchdogPolicy) Normalized() BackendWatchdogPolicy {
	out := p
	out.TerminationReasonCode = strings.TrimSpace(strings.ToLower(out.TerminationReasonCode))
	if out.TerminationReasonCode == "" {
		out.TerminationReasonCode = WatchdogTerminationReasonCodeTimeout
	}
	return out
}

func (p BackendWatchdogPolicy) Validate() error {
	normalized := p.Normalized()
	if !normalized.Enabled {
		return fmt.Errorf("enabled must be true for fail-closed watchdog")
	}
	if !normalized.TerminateOnMisbehavior {
		return fmt.Errorf("terminate_on_misbehavior must be true")
	}
	if normalized.HeartbeatTimeoutSeconds < 1 || normalized.HeartbeatTimeoutSeconds > 3600 {
		return fmt.Errorf("heartbeat_timeout_seconds must be between 1 and 3600")
	}
	if normalized.NoProgressTimeoutSeconds < 1 || normalized.NoProgressTimeoutSeconds > 86400 {
		return fmt.Errorf("no_progress_timeout_seconds must be between 1 and 86400")
	}
	if normalized.TerminationReasonCode != WatchdogTerminationReasonCodeTimeout {
		return fmt.Errorf("termination_reason_code must be %q", WatchdogTerminationReasonCodeTimeout)
	}
	return nil
}

func (p BackendLifecyclePolicy) Validate() error {
	if !p.TerminateBetweenSteps {
		return fmt.Errorf("terminate_between_steps must be true")
	}
	return nil
}

func (p BackendCachePosture) Validate() error {
	if !p.ResetOrDestroyBeforeReuse {
		return fmt.Errorf("reset_or_destroy_before_reuse must be true")
	}
	if p.ReusePriorSessionIdentityKeys {
		return fmt.Errorf("reuse_prior_session_identity_keys must be false")
	}
	if !p.DigestPinned {
		return fmt.Errorf("digest_pinned must be true")
	}
	if !p.SignaturePinned {
		return fmt.Errorf("signature_pinned must be true")
	}
	return nil
}

func validateAccelerationForPlatform(kind string, requestedBackend string) error {
	normalizedBackend := normalizeBackendKind(requestedBackend)
	if normalizedBackend != BackendKindMicroVM {
		return nil
	}
	normalized := normalizeAccelerationKind(kind)
	if normalized == AccelerationKindUnknown {
		return fmt.Errorf("requested_acceleration_kind must be one of %q, %q, %q, or %q", AccelerationKindKVM, AccelerationKindHVF, AccelerationKindWHPX, AccelerationKindNone)
	}
	if runtime.GOOS != "linux" {
		return fmt.Errorf("microvm acceleration unsupported on %s; MVP supports only linux/%s", runtime.GOOS, AccelerationKindKVM)
	}
	if normalized != AccelerationKindKVM {
		return fmt.Errorf("requested_acceleration_kind %q unsupported on %s; MVP requires %q", normalized, runtime.GOOS, AccelerationKindKVM)
	}
	return nil
}

func validateControlTransportKind(kind string, requestedBackend string) error {
	normalizedBackend := normalizeBackendKind(requestedBackend)
	if normalizedBackend != BackendKindMicroVM {
		return nil
	}
	normalized := normalizeTransportKind(kind)
	if normalized == TransportKindUnknown {
		return fmt.Errorf("control_transport_kind must be one of %q or %q", TransportKindVSock, TransportKindVirtioSerial)
	}
	return nil
}
