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
	return validateRuntimeImageAttestationHooks(d.Attestation)
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
	if strings.TrimSpace(bootContractVersion) == "" {
		return fmt.Errorf("boot_contract_version is required")
	}
	if len(componentDigests) == 0 {
		return fmt.Errorf("component_digests is required")
	}
	if err := validateBackendSpecificComponentRequirements(backendKind, componentDigests); err != nil {
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

func validateBackendSpecificComponentRequirements(backendKind string, componentDigests map[string]string) error {
	normalizedBackend := normalizeBackendKind(backendKind)
	if normalizedBackend == BackendKindMicroVM {
		if strings.TrimSpace(componentDigests["kernel"]) == "" || strings.TrimSpace(componentDigests["rootfs"]) == "" {
			return fmt.Errorf("component_digests must include kernel and rootfs for backend_kind microvm")
		}
		return nil
	}
	if normalizedBackend == BackendKindContainer && strings.TrimSpace(componentDigests["image"]) == "" {
		return fmt.Errorf("component_digests must include image for backend_kind container")
	}
	return nil
}

func validateRuntimeImageSigningHooks(signing *RuntimeImageSigningHooks) error {
	if signing == nil {
		return nil
	}
	if strings.TrimSpace(signing.SignerRef) == "" && strings.TrimSpace(signing.SignatureDigest) == "" && strings.TrimSpace(signing.SignatureBundleRef) == "" {
		return fmt.Errorf("signing must include at least one field")
	}
	if strings.TrimSpace(signing.SignerRef) != "" && looksLikeHostPath(signing.SignerRef) {
		return fmt.Errorf("signing.signer_ref must not include host-local path material")
	}
	if strings.TrimSpace(signing.SignatureDigest) != "" && !looksLikeDigest(signing.SignatureDigest) {
		return fmt.Errorf("signing.signature_digest must be sha256:<64 lowercase hex>")
	}
	if strings.TrimSpace(signing.SignatureBundleRef) != "" && looksLikeHostPath(signing.SignatureBundleRef) {
		return fmt.Errorf("signing.signature_bundle_ref must not include host-local path material")
	}
	return nil
}

func validateRuntimeImageAttestationHooks(attestation *RuntimeImageAttestationHook) error {
	if attestation == nil {
		return nil
	}
	if strings.TrimSpace(attestation.MeasurementProfile) == "" && len(attestation.ExpectedMeasurementDigests) == 0 {
		return fmt.Errorf("attestation must include at least one field")
	}
	for _, digest := range attestation.ExpectedMeasurementDigests {
		if !looksLikeDigest(digest) {
			return fmt.Errorf("attestation.expected_measurement_digests values must be sha256:<64 lowercase hex>")
		}
	}
	return nil
}
