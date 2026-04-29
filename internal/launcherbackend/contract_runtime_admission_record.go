package launcherbackend

import (
	"fmt"
	"strings"
)

// RuntimeAdmissionRecord is launcher-private trusted admission state.
// It captures verified runtime identity only (no host-local paths).
type RuntimeAdmissionRecord struct {
	DescriptorDigest       string                     `json:"descriptor_digest"`
	BackendKind            string                     `json:"backend_kind"`
	PlatformCompatibility  RuntimeImagePlatformCompat `json:"platform_compatibility"`
	BootContractVersion    string                     `json:"boot_contract_version"`
	ComponentDigests       map[string]string          `json:"component_digests"`
	AuthorityStateDigest   string                     `json:"authority_state_digest"`
	AuthorityStateRevision uint64                     `json:"authority_state_revision"`

	RuntimeImageSignerRef       string `json:"runtime_image_signer_ref"`
	RuntimeImageVerifierSetRef  string `json:"runtime_image_verifier_set_ref"`
	RuntimeImageSignatureDigest string `json:"runtime_image_signature_digest"`

	RuntimeToolchainDescriptorDigest string `json:"runtime_toolchain_descriptor_digest,omitempty"`
	RuntimeToolchainSignerRef        string `json:"runtime_toolchain_signer_ref,omitempty"`
	RuntimeToolchainVerifierSetRef   string `json:"runtime_toolchain_verifier_set_ref,omitempty"`
	RuntimeToolchainSignatureDigest  string `json:"runtime_toolchain_signature_digest,omitempty"`
	RuntimeToolchainBundleDigest     string `json:"runtime_toolchain_bundle_digest,omitempty"`
}

func NewRuntimeAdmissionRecord(image RuntimeImageDescriptor) (RuntimeAdmissionRecord, error) {
	if err := image.Validate(); err != nil {
		return RuntimeAdmissionRecord{}, err
	}
	if image.Signing == nil {
		return RuntimeAdmissionRecord{}, fmt.Errorf("signing metadata is required for admitted runtime identity")
	}
	record := RuntimeAdmissionRecord{
		DescriptorDigest:                image.DescriptorDigest,
		BackendKind:                     normalizeBackendKind(image.BackendKind),
		PlatformCompatibility:           image.PlatformCompatibility,
		BootContractVersion:             normalizeBootProfile(image.BootContractVersion),
		ComponentDigests:                cloneStringMap(image.ComponentDigests),
		RuntimeImageSignerRef:           strings.TrimSpace(image.Signing.SignerRef),
		RuntimeImageVerifierSetRef:      strings.TrimSpace(image.Signing.VerifierSetRef),
		RuntimeImageSignatureDigest:     strings.TrimSpace(image.Signing.SignatureDigest),
		RuntimeToolchainSignerRef:       trimToolchainSignerRef(image.Signing.Toolchain),
		RuntimeToolchainVerifierSetRef:  trimToolchainVerifierRef(image.Signing.Toolchain),
		RuntimeToolchainSignatureDigest: trimToolchainSignatureDigest(image.Signing.Toolchain),
		RuntimeToolchainBundleDigest:    trimToolchainBundleDigest(image.Signing.Toolchain),
	}
	if image.Signing.Toolchain != nil {
		record.RuntimeToolchainDescriptorDigest = strings.TrimSpace(image.Signing.Toolchain.DescriptorDigest)
	}
	if err := record.Validate(); err != nil {
		return RuntimeAdmissionRecord{}, err
	}
	return record, nil
}

func (r RuntimeAdmissionRecord) Validate() error {
	if err := validateAdmissionRecordDescriptor(r); err != nil {
		return err
	}
	if err := validateAdmissionRecordRuntimeImageSigning(r); err != nil {
		return err
	}
	return validateAdmissionRecordToolchainSigning(r)
}

func validateAdmissionRecordDescriptor(record RuntimeAdmissionRecord) error {
	descriptor := RuntimeImageDescriptor{
		DescriptorDigest:      strings.TrimSpace(record.DescriptorDigest),
		BackendKind:           strings.TrimSpace(record.BackendKind),
		PlatformCompatibility: record.PlatformCompatibility,
		BootContractVersion:   strings.TrimSpace(record.BootContractVersion),
		ComponentDigests:      cloneStringMap(record.ComponentDigests),
	}
	if err := validateRuntimeImageDescriptorIdentityAndBackend(descriptor); err != nil {
		return fmt.Errorf("admission record %w", err)
	}
	if err := validateRuntimeImageDescriptorPlatform(descriptor.PlatformCompatibility); err != nil {
		return fmt.Errorf("admission record %w", err)
	}
	if err := validateRuntimeImageDescriptorComponents(descriptor.BackendKind, descriptor.BootContractVersion, descriptor.ComponentDigests); err != nil {
		return fmt.Errorf("admission record %w", err)
	}
	hasAuthorityDigest := strings.TrimSpace(record.AuthorityStateDigest) != ""
	hasAuthorityRevision := record.AuthorityStateRevision > 0
	if hasAuthorityDigest != hasAuthorityRevision {
		return fmt.Errorf("authority_state_digest and authority_state_revision must be both set or both empty")
	}
	if hasAuthorityDigest && !looksLikeDigest(strings.TrimSpace(record.AuthorityStateDigest)) {
		return fmt.Errorf("authority_state_digest must be sha256:<64 lowercase hex>")
	}
	return nil
}

func validateAdmissionRecordRuntimeImageSigning(record RuntimeAdmissionRecord) error {
	if err := requireRuntimeSignerIdentity(record.RuntimeImageSignerRef, "runtime_image_signer_ref", true); err != nil {
		return err
	}
	if err := requireRuntimeSignerIdentity(record.RuntimeToolchainSignerRef, "runtime_toolchain_signer_ref", false); err != nil {
		return err
	}
	if !looksLikeDigest(strings.TrimSpace(record.RuntimeImageVerifierSetRef)) {
		return fmt.Errorf("runtime_image_verifier_set_ref must be sha256:<64 lowercase hex>")
	}
	if strings.TrimSpace(record.RuntimeImageSignatureDigest) == "" || !looksLikeDigest(record.RuntimeImageSignatureDigest) {
		return fmt.Errorf("runtime_image_signature_digest must be sha256:<64 lowercase hex>")
	}
	return nil
}

func validateAdmissionRecordToolchainSigning(record RuntimeAdmissionRecord) error {
	if !runtimeToolchainFieldsPresent(record) {
		return nil
	}
	if strings.TrimSpace(record.RuntimeToolchainDescriptorDigest) == "" || strings.TrimSpace(record.RuntimeToolchainSignerRef) == "" || strings.TrimSpace(record.RuntimeToolchainVerifierSetRef) == "" || strings.TrimSpace(record.RuntimeToolchainSignatureDigest) == "" {
		return fmt.Errorf("runtime toolchain admission fields must be all-or-none")
	}
	if !looksLikeDigest(record.RuntimeToolchainDescriptorDigest) {
		return fmt.Errorf("runtime_toolchain_descriptor_digest must be sha256:<64 lowercase hex>")
	}
	if !looksLikeDigest(strings.TrimSpace(record.RuntimeToolchainVerifierSetRef)) {
		return fmt.Errorf("runtime_toolchain_verifier_set_ref must be sha256:<64 lowercase hex>")
	}
	if !looksLikeDigest(strings.TrimSpace(record.RuntimeToolchainSignatureDigest)) {
		return fmt.Errorf("runtime_toolchain_signature_digest must be sha256:<64 lowercase hex>")
	}
	if strings.TrimSpace(record.RuntimeToolchainBundleDigest) != "" && !looksLikeDigest(strings.TrimSpace(record.RuntimeToolchainBundleDigest)) {
		return fmt.Errorf("runtime_toolchain_bundle_digest must be sha256:<64 lowercase hex>")
	}
	return nil
}

func runtimeToolchainFieldsPresent(record RuntimeAdmissionRecord) bool {
	return strings.TrimSpace(record.RuntimeToolchainDescriptorDigest) != "" ||
		strings.TrimSpace(record.RuntimeToolchainSignerRef) != "" ||
		strings.TrimSpace(record.RuntimeToolchainVerifierSetRef) != "" ||
		strings.TrimSpace(record.RuntimeToolchainSignatureDigest) != "" ||
		strings.TrimSpace(record.RuntimeToolchainBundleDigest) != ""
}

func requireRuntimeSignerIdentity(value string, field string, required bool) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if !required {
			return nil
		}
		return fmt.Errorf("%s is required", field)
	}
	if looksLikeHostPath(trimmed) {
		return fmt.Errorf("%s must not include host-local path material", field)
	}
	return nil
}

func trimToolchainSignerRef(toolchain *RuntimeToolchainSigningHooks) string {
	if toolchain == nil {
		return ""
	}
	return strings.TrimSpace(toolchain.SignerRef)
}

func trimToolchainVerifierRef(toolchain *RuntimeToolchainSigningHooks) string {
	if toolchain == nil {
		return ""
	}
	return strings.TrimSpace(toolchain.VerifierSetRef)
}

func trimToolchainSignatureDigest(toolchain *RuntimeToolchainSigningHooks) string {
	if toolchain == nil {
		return ""
	}
	return strings.TrimSpace(toolchain.SignatureDigest)
}

func trimToolchainBundleDigest(toolchain *RuntimeToolchainSigningHooks) string {
	if toolchain == nil {
		return ""
	}
	return strings.TrimSpace(toolchain.BundleDigest)
}
