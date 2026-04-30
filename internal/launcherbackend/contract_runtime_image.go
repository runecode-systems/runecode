package launcherbackend

import (
	"fmt"
	"strings"
)

func (d RuntimeImageDescriptor) Validate() error {
	if err := validateRuntimeImageDescriptorIdentityAndBackend(d); err != nil {
		return err
	}
	if err := validateRuntimeImageDescriptorPlatform(d.PlatformCompatibility); err != nil {
		return err
	}
	if err := validateRuntimeImageDescriptorComponents(d.BackendKind, d.BootContractVersion, d.ComponentDigests); err != nil {
		return err
	}
	if err := validateRuntimeImageSigningHooks(d.Signing); err != nil {
		return err
	}
	if err := validateRuntimeImageAttestationHooks(d.Attestation); err != nil {
		return err
	}
	return validateRuntimeImageSignedPayloadBinding(d)
}

func validateRuntimeImageDescriptorIdentityAndBackend(descriptor RuntimeImageDescriptor) error {
	if strings.TrimSpace(descriptor.DescriptorDigest) == "" {
		return fmt.Errorf("descriptor_digest is required")
	}
	if !looksLikeDigest(descriptor.DescriptorDigest) {
		return fmt.Errorf("descriptor_digest must be sha256:<64 lowercase hex>")
	}
	if normalizeBackendKind(descriptor.BackendKind) == BackendKindUnknown {
		return fmt.Errorf("backend_kind must be one of %q or %q", BackendKindMicroVM, BackendKindContainer)
	}
	return nil
}

func validateRuntimeImageDescriptorPlatform(platform RuntimeImagePlatformCompat) error {
	if strings.TrimSpace(platform.OS) == "" || strings.TrimSpace(platform.Architecture) == "" {
		return fmt.Errorf("platform_compatibility.os and platform_compatibility.architecture are required")
	}
	if !roleTokenPattern.MatchString(strings.TrimSpace(platform.OS)) || !roleTokenPattern.MatchString(strings.TrimSpace(platform.Architecture)) {
		return fmt.Errorf("platform_compatibility fields must match token pattern")
	}
	if strings.TrimSpace(platform.AccelerationKind) != "" && normalizeAccelerationKind(platform.AccelerationKind) == AccelerationKindUnknown {
		return fmt.Errorf("platform_compatibility.acceleration_kind must be one of %q, %q, %q, or %q", AccelerationKindKVM, AccelerationKindHVF, AccelerationKindWHPX, AccelerationKindNone)
	}
	return nil
}

func validateRuntimeImageDescriptorComponents(backendKind string, bootContractVersion string, componentDigests map[string]string) error {
	bootProfile := normalizeBootProfile(bootContractVersion)
	if bootProfile == "" {
		return fmt.Errorf("boot_contract_version must be one of %q or %q", BootProfileMicroVMLinuxKernelInitrdV1, BootProfileContainerOCIImageV1)
	}
	if len(componentDigests) == 0 {
		return fmt.Errorf("component_digests is required")
	}
	if err := validateBootProfileBackendCompatibility(backendKind, bootProfile); err != nil {
		return err
	}
	if err := validateBootProfileComponentRequirements(bootProfile, componentDigests); err != nil {
		return err
	}
	return validateComponentDigestEntries(componentDigests)
}

func validateComponentDigestEntries(componentDigests map[string]string) error {
	for name, digest := range componentDigests {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(digest) == "" {
			return fmt.Errorf("component_digests entries must be non-empty")
		}
		if !roleTokenPattern.MatchString(strings.TrimSpace(name)) {
			return fmt.Errorf("component_digests keys must match token pattern")
		}
		if !looksLikeDigest(digest) {
			return fmt.Errorf("component_digests values must be sha256:<64 lowercase hex>")
		}
	}
	return nil
}

func validateBootProfileBackendCompatibility(backendKind string, bootProfile string) error {
	switch normalizeBackendKind(backendKind) {
	case BackendKindMicroVM:
		if bootProfile != BootProfileMicroVMLinuxKernelInitrdV1 {
			return fmt.Errorf("backend_kind microvm requires boot_contract_version %q", BootProfileMicroVMLinuxKernelInitrdV1)
		}
	case BackendKindContainer:
		if bootProfile != BootProfileContainerOCIImageV1 {
			return fmt.Errorf("backend_kind container requires boot_contract_version %q", BootProfileContainerOCIImageV1)
		}
	}
	return nil
}

func validateBootProfileComponentRequirements(bootProfile string, componentDigests map[string]string) error {
	switch bootProfile {
	case BootProfileMicroVMLinuxKernelInitrdV1:
		if strings.TrimSpace(componentDigests["kernel"]) == "" || strings.TrimSpace(componentDigests["initrd"]) == "" {
			return fmt.Errorf("component_digests must include kernel and initrd for boot_contract_version %q", BootProfileMicroVMLinuxKernelInitrdV1)
		}
	case BootProfileContainerOCIImageV1:
		if strings.TrimSpace(componentDigests["image"]) == "" {
			return fmt.Errorf("component_digests must include image for boot_contract_version %q", BootProfileContainerOCIImageV1)
		}
	}
	return nil
}

func validateRuntimeImageAttestationHooks(attestation *RuntimeImageAttestationHook) error {
	if attestation == nil {
		return fmt.Errorf("attestation is required")
	}
	profile := strings.TrimSpace(attestation.MeasurementProfile)
	if profile == "" && len(attestation.ExpectedMeasurementDigests) == 0 {
		return fmt.Errorf("attestation must include at least one field")
	}
	if profile == "" {
		return fmt.Errorf("attestation.measurement_profile is required when expected_measurement_digests are declared")
	}
	if !measurementProfileKnown(profile) {
		return fmt.Errorf("attestation.measurement_profile %q is invalid", attestation.MeasurementProfile)
	}
	normalizedDigests := normalizeExpectedMeasurementDigests(profile, attestation.ExpectedMeasurementDigests)
	if err := validateMeasurementProfileExpectedDigests(profile, normalizedDigests); err != nil {
		return fmt.Errorf("attestation.%w", err)
	}
	attestation.MeasurementProfile = normalizeMeasurementProfile(profile)
	attestation.ExpectedMeasurementDigests = normalizedDigests
	return nil
}

func validateRuntimeImageSignedPayloadBinding(descriptor RuntimeImageDescriptor) error {
	expectedDigest, err := descriptor.ExpectedDescriptorDigest()
	if err != nil {
		return err
	}
	if descriptor.DescriptorDigest != expectedDigest {
		return fmt.Errorf("descriptor_digest must match canonical signed payload digest")
	}
	if descriptor.Signing == nil {
		return nil
	}
	if descriptor.Signing.PayloadSchemaID != RuntimeImageSignedPayloadSchemaID {
		return fmt.Errorf("signing.payload_schema_id must be %q", RuntimeImageSignedPayloadSchemaID)
	}
	if descriptor.Signing.PayloadSchemaVersion != RuntimeImageSignedPayloadSchemaVersion {
		return fmt.Errorf("signing.payload_schema_version must be %q", RuntimeImageSignedPayloadSchemaVersion)
	}
	if descriptor.Signing.PayloadDigest != expectedDigest {
		return fmt.Errorf("signing.payload_digest must match canonical signed payload digest")
	}
	return nil
}
