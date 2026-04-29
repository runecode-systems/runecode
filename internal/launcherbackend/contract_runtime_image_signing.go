package launcherbackend

import (
	"fmt"
	"strings"
)

type RuntimeImageSignedPayload struct {
	SchemaID              string                       `json:"schema_id"`
	SchemaVersion         string                       `json:"schema_version"`
	BackendKind           string                       `json:"backend_kind"`
	PlatformCompatibility RuntimeImagePlatformCompat   `json:"platform_compatibility"`
	BootContractVersion   string                       `json:"boot_contract_version"`
	ComponentDigests      map[string]string            `json:"component_digests"`
	Attestation           *RuntimeImageAttestationHook `json:"attestation,omitempty"`
}

func (d RuntimeImageDescriptor) SignedPayload() RuntimeImageSignedPayload {
	return RuntimeImageSignedPayload{
		SchemaID:              RuntimeImageSignedPayloadSchemaID,
		SchemaVersion:         RuntimeImageSignedPayloadSchemaVersion,
		BackendKind:           normalizeBackendKind(d.BackendKind),
		PlatformCompatibility: d.PlatformCompatibility,
		BootContractVersion:   strings.TrimSpace(d.BootContractVersion),
		ComponentDigests:      cloneStringMap(d.ComponentDigests),
		Attestation:           cloneAttestationHook(d.Attestation),
	}
}

func (d RuntimeImageDescriptor) SignedPayloadCanonicalBytes() ([]byte, error) {
	return canonicalJSONBytes(d.SignedPayload(), "runtime image signed payload")
}

func (d RuntimeImageDescriptor) ExpectedDescriptorDigest() (string, error) {
	return canonicalSHA256Digest(d.SignedPayload(), "runtime image signed payload")
}

func validateRuntimeImageSigningHooks(signing *RuntimeImageSigningHooks) error {
	if signing == nil {
		return nil
	}
	if err := validateRuntimeImageSigningCore(signing); err != nil {
		return err
	}
	if err := validateRuntimeImageSigningReferences(signing); err != nil {
		return err
	}
	if err := validateRuntimeAssetPublicationBundle(signing.Publication); err != nil {
		return err
	}
	return validateRuntimeToolchainSigningHooks(signing.Toolchain)
}

func validateRuntimeImageSigningCore(signing *RuntimeImageSigningHooks) error {
	if strings.TrimSpace(signing.PayloadSchemaID) == "" || strings.TrimSpace(signing.PayloadSchemaVersion) == "" {
		return fmt.Errorf("signing.payload_schema_id and signing.payload_schema_version are required")
	}
	if strings.TrimSpace(signing.PayloadDigest) == "" {
		return fmt.Errorf("signing.payload_digest is required")
	}
	if !looksLikeDigest(signing.PayloadDigest) {
		return fmt.Errorf("signing.payload_digest must be sha256:<64 lowercase hex>")
	}
	if strings.TrimSpace(signing.SignerRef) == "" || strings.TrimSpace(signing.SignatureDigest) == "" {
		return fmt.Errorf("signing.signer_ref and signing.signature_digest are required")
	}
	if strings.TrimSpace(signing.VerifierSetRef) == "" {
		return fmt.Errorf("signing.verifier_set_ref is required")
	}
	if !looksLikeDigest(signing.VerifierSetRef) {
		return fmt.Errorf("signing.verifier_set_ref must be sha256:<64 lowercase hex>")
	}
	return nil
}

func validateRuntimeImageSigningReferences(signing *RuntimeImageSigningHooks) error {
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

func validateRuntimeAssetPublicationBundle(bundle *RuntimeAssetPublicationBundle) error {
	if bundle == nil {
		return nil
	}
	if strings.TrimSpace(bundle.DescriptorEnvelopeDigest) == "" && strings.TrimSpace(bundle.ComponentBundleDigest) == "" && strings.TrimSpace(bundle.PublicationManifestDigest) == "" {
		return fmt.Errorf("signing.publication must include at least one digest")
	}
	for fieldName, digest := range map[string]string{
		"signing.publication.descriptor_envelope_digest":  bundle.DescriptorEnvelopeDigest,
		"signing.publication.component_bundle_digest":     bundle.ComponentBundleDigest,
		"signing.publication.publication_manifest_digest": bundle.PublicationManifestDigest,
	} {
		if strings.TrimSpace(digest) != "" && !looksLikeDigest(digest) {
			return fmt.Errorf("%s must be sha256:<64 lowercase hex>", fieldName)
		}
	}
	return nil
}

func validateRuntimeToolchainSigningHooks(toolchain *RuntimeToolchainSigningHooks) error {
	if toolchain == nil {
		return nil
	}
	applyRuntimeToolchainDescriptorDefaults(toolchain)
	if err := validateRuntimeToolchainSigningCore(toolchain); err != nil {
		return err
	}
	return validateRuntimeToolchainSigningReferences(toolchain)
}

func applyRuntimeToolchainDescriptorDefaults(toolchain *RuntimeToolchainSigningHooks) {
	if strings.TrimSpace(toolchain.DescriptorSchemaID) == "" {
		toolchain.DescriptorSchemaID = RuntimeToolchainDescriptorSchemaID
	}
	if strings.TrimSpace(toolchain.DescriptorSchemaVersion) == "" {
		toolchain.DescriptorSchemaVersion = RuntimeToolchainDescriptorSchemaVersion
	}
}

func validateRuntimeToolchainSigningCore(toolchain *RuntimeToolchainSigningHooks) error {
	if strings.TrimSpace(toolchain.DescriptorDigest) == "" || strings.TrimSpace(toolchain.SignerRef) == "" || strings.TrimSpace(toolchain.SignatureDigest) == "" {
		return fmt.Errorf("signing.toolchain.descriptor_digest, signer_ref, and signature_digest are required")
	}
	if !looksLikeDigest(toolchain.DescriptorDigest) {
		return fmt.Errorf("signing.toolchain.descriptor_digest must be sha256:<64 lowercase hex>")
	}
	if !looksLikeDigest(toolchain.SignatureDigest) {
		return fmt.Errorf("signing.toolchain.signature_digest must be sha256:<64 lowercase hex>")
	}
	if strings.TrimSpace(toolchain.VerifierSetRef) == "" {
		return fmt.Errorf("signing.toolchain.verifier_set_ref is required")
	}
	if !looksLikeDigest(toolchain.VerifierSetRef) {
		return fmt.Errorf("signing.toolchain.verifier_set_ref must be sha256:<64 lowercase hex>")
	}
	return nil
}

func validateRuntimeToolchainSigningReferences(toolchain *RuntimeToolchainSigningHooks) error {
	if looksLikeHostPath(toolchain.SignerRef) {
		return fmt.Errorf("signing.toolchain.signer_ref must not include host-local path material")
	}
	if strings.TrimSpace(toolchain.SignatureBundleRef) != "" && looksLikeHostPath(toolchain.SignatureBundleRef) {
		return fmt.Errorf("signing.toolchain.signature_bundle_ref must not include host-local path material")
	}
	if strings.TrimSpace(toolchain.BundleDigest) != "" && !looksLikeDigest(toolchain.BundleDigest) {
		return fmt.Errorf("signing.toolchain.bundle_digest must be sha256:<64 lowercase hex>")
	}
	return nil
}

func normalizeBootProfile(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case BootProfileMicroVMLinuxKernelInitrdV1:
		return BootProfileMicroVMLinuxKernelInitrdV1
	case BootProfileContainerOCIImageV1:
		return BootProfileContainerOCIImageV1
	default:
		return ""
	}
}

func cloneAttestationHook(value *RuntimeImageAttestationHook) *RuntimeImageAttestationHook {
	if value == nil {
		return nil
	}
	clone := *value
	clone.ExpectedMeasurementDigests = append([]string{}, value.ExpectedMeasurementDigests...)
	return &clone
}
