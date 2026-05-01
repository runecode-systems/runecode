package trustpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func validateExternalAnchorPayload(anchorKind string, external anchorExternalPayload) error {
	if external.TargetKind == "" {
		return fmt.Errorf("external_anchor.target_kind is required")
	}
	if external.RuntimeAdapter == "" {
		return fmt.Errorf("external_anchor.runtime_adapter is required")
	}
	if len(external.TargetDescriptor) == 0 {
		return fmt.Errorf("external_anchor.target_descriptor is required")
	}
	if !isJSONObject(external.TargetDescriptor) {
		return fmt.Errorf("external_anchor.target_descriptor must be a JSON object")
	}
	if _, err := external.TargetDescriptorDigest.Identity(); err != nil {
		return fmt.Errorf("external_anchor.target_descriptor_digest: %w", err)
	}
	if err := validateExternalAnchorProof(external.Proof); err != nil {
		return err
	}
	if err := validateDerivedExecutionForAnchorKind(anchorKind, external.DerivedExecution); err != nil {
		return err
	}
	if err := validateTargetKindAndProofBinding(anchorKind, external.TargetKind, external.RuntimeAdapter, external.Proof); err != nil {
		return err
	}
	return validateExternalTargetDescriptor(anchorKind, external.TargetDescriptor, external.TargetDescriptorDigest)
}

func validateExternalAnchorProof(proof anchorExternalProofRef) error {
	if proof.ProofKind == "" {
		return fmt.Errorf("external_anchor.proof.proof_kind is required")
	}
	if proof.ProofSchemaID == "" {
		return fmt.Errorf("external_anchor.proof.proof_schema_id is required")
	}
	if _, err := proof.ProofDigest.Identity(); err != nil {
		return fmt.Errorf("external_anchor.proof.proof_digest: %w", err)
	}
	return nil
}

func validateDerivedExecutionForAnchorKind(anchorKind string, derived json.RawMessage) error {
	if len(derived) == 0 {
		return nil
	}
	if !isJSONObject(derived) {
		return fmt.Errorf("external_anchor.derived_execution must be a JSON object")
	}
	validator, err := derivedExecutionValidator(anchorKind)
	if err != nil {
		return err
	}
	return validator(derived)
}

func derivedExecutionValidator(anchorKind string) (func(json.RawMessage) error, error) {
	switch anchorKind {
	case anchorKindExternalTransparencyLog:
		return validateTransparencyLogDerivedExecution, nil
	case anchorKindExternalTimestampAuth:
		return validateTimestampAuthorityDerivedExecution, nil
	case anchorKindExternalPublicChain:
		return validatePublicChainDerivedExecution, nil
	default:
		return nil, fmt.Errorf("unsupported anchor_kind %q", anchorKind)
	}
}

func validateTransparencyLogDerivedExecution(derived json.RawMessage) error {
	value := transparencyLogDerivedExecution{}
	if err := unmarshalJSONStrict(derived, &value); err != nil {
		return fmt.Errorf("decode external_anchor.derived_execution transparency_log: %w", err)
	}
	if value.SubmitEndpointURI == "" {
		return fmt.Errorf("external_anchor.derived_execution.%s is required when derived_execution is present", anchorDerivedFieldSubmitEndpointURI)
	}
	return nil
}

func validateTimestampAuthorityDerivedExecution(derived json.RawMessage) error {
	value := timestampAuthorityDerivedExecution{}
	if err := unmarshalJSONStrict(derived, &value); err != nil {
		return fmt.Errorf("decode external_anchor.derived_execution timestamp_authority: %w", err)
	}
	if value.TSAEndpointURI == "" {
		return fmt.Errorf("external_anchor.derived_execution.%s is required when derived_execution is present", anchorDerivedFieldTSAEndpointURI)
	}
	return nil
}

func validatePublicChainDerivedExecution(derived json.RawMessage) error {
	value := publicChainDerivedExecution{}
	if err := unmarshalJSONStrict(derived, &value); err != nil {
		return fmt.Errorf("decode external_anchor.derived_execution public_chain: %w", err)
	}
	if value.RPCEndpointURI == "" {
		return fmt.Errorf("external_anchor.derived_execution.%s is required when derived_execution is present", anchorDerivedFieldRPCEndpointURI)
	}
	return nil
}

func validateTargetKindAndProofBinding(anchorKind, targetKind, runtimeAdapter string, proof anchorExternalProofRef) error {
	wantTargetKind, wantRuntimeAdapter, wantProofKind, wantProofSchemaID, err := expectedTargetKindAndProofBinding(anchorKind)
	if err != nil {
		return err
	}
	if targetKind != wantTargetKind {
		return fmt.Errorf("external_anchor.target_kind %q does not match anchor_kind %q", targetKind, anchorKind)
	}
	if runtimeAdapter != wantRuntimeAdapter {
		return externalAnchorRuntimeAdapterBindingError(anchorKind, runtimeAdapter, wantRuntimeAdapter)
	}
	if proof.ProofKind != wantProofKind {
		return fmt.Errorf("external_anchor.proof.proof_kind %q does not match anchor_kind %q", proof.ProofKind, anchorKind)
	}
	if proof.ProofSchemaID != wantProofSchemaID {
		return fmt.Errorf("external_anchor.proof.proof_schema_id %q does not match anchor_kind %q", proof.ProofSchemaID, anchorKind)
	}
	return nil
}

func expectedTargetKindAndProofBinding(anchorKind string) (string, string, string, string, error) {
	switch anchorKind {
	case anchorKindExternalTransparencyLog:
		return anchorTargetKindTransparencyLog, anchorRuntimeAdapterTransparencyLogV0, anchorProofKindTransparencyLogReceiptV0, anchorProofSchemaTransparencyLogReceiptV0, nil
	case anchorKindExternalTimestampAuth:
		return anchorTargetKindTimestampAuth, anchorRuntimeAdapterTransparencyLogV0, anchorProofKindTimestampTokenV0, anchorProofSchemaTimestampTokenV0, nil
	case anchorKindExternalPublicChain:
		return anchorTargetKindPublicChain, anchorRuntimeAdapterTransparencyLogV0, anchorProofKindPublicChainTxReceiptV0, anchorProofSchemaPublicChainTxReceiptV0, nil
	default:
		return "", "", "", "", fmt.Errorf("unsupported anchor_kind %q", anchorKind)
	}
}

func externalAnchorRuntimeAdapterBindingError(anchorKind, runtimeAdapter, wantRuntimeAdapter string) error {
	if anchorKind == anchorKindExternalTransparencyLog {
		return fmt.Errorf("external_anchor.runtime_adapter %q is unsupported for anchor_kind %q", runtimeAdapter, anchorKind)
	}
	return fmt.Errorf("external_anchor.runtime_adapter %q is unsupported for anchor_kind %q; first concrete adapter is %q", runtimeAdapter, anchorKind, wantRuntimeAdapter)
}

func validateExternalTargetDescriptor(anchorKind string, descriptorRaw json.RawMessage, expectedDigest Digest) error {
	actualDigest, err := computeJSONCanonicalDigest(descriptorRaw)
	if err != nil {
		return fmt.Errorf("canonicalize external_anchor.target_descriptor: %w", err)
	}
	if mustDigestIdentity(actualDigest) != mustDigestIdentity(expectedDigest) {
		return fmt.Errorf("external_anchor.target_descriptor_digest does not match canonical target_descriptor digest")
	}
	switch anchorKind {
	case anchorKindExternalTransparencyLog:
		return validateTransparencyLogDescriptor(descriptorRaw)
	case anchorKindExternalTimestampAuth:
		return validateTimestampAuthorityDescriptor(descriptorRaw)
	case anchorKindExternalPublicChain:
		return validatePublicChainDescriptor(descriptorRaw)
	default:
		return fmt.Errorf("unsupported anchor_kind %q", anchorKind)
	}
}

func validateTransparencyLogDescriptor(raw json.RawMessage) error {
	value := transparencyLogTargetDescriptor{}
	if err := unmarshalJSONStrict(raw, &value); err != nil {
		return fmt.Errorf("decode external_anchor.target_descriptor transparency_log: %w", err)
	}
	if value.DescriptorSchemaID != anchorDescriptorSchemaTransparencyLogV0 {
		return fmt.Errorf("external_anchor.target_descriptor.%s must be %q", anchorDescriptorFieldDescriptorSchemaID, anchorDescriptorSchemaTransparencyLogV0)
	}
	if value.LogID == "" {
		return fmt.Errorf("external_anchor.target_descriptor.%s is required", anchorDescriptorFieldLogID)
	}
	if _, err := value.LogPublicKeyDigest.Identity(); err != nil {
		return fmt.Errorf("external_anchor.target_descriptor.%s: %w", anchorDescriptorFieldLogPublicKeyDigest, err)
	}
	if value.EntryEncoding == "" {
		return fmt.Errorf("external_anchor.target_descriptor.%s is required", anchorDescriptorFieldEntryEncodingProfile)
	}
	return nil
}

func validateTimestampAuthorityDescriptor(raw json.RawMessage) error {
	value := timestampAuthorityTargetDescriptor{}
	if err := unmarshalJSONStrict(raw, &value); err != nil {
		return fmt.Errorf("decode external_anchor.target_descriptor timestamp_authority: %w", err)
	}
	if value.DescriptorSchemaID != anchorDescriptorSchemaTimestampAuthorityV0 {
		return fmt.Errorf("external_anchor.target_descriptor.%s must be %q", anchorDescriptorFieldDescriptorSchemaID, anchorDescriptorSchemaTimestampAuthorityV0)
	}
	if value.AuthorityID == "" {
		return fmt.Errorf("external_anchor.target_descriptor.%s is required", anchorDescriptorFieldAuthorityID)
	}
	if _, err := value.CertificateChainDigest.Identity(); err != nil {
		return fmt.Errorf("external_anchor.target_descriptor.%s: %w", anchorDescriptorFieldCertificateDigest, err)
	}
	if value.TimestampProfile == "" {
		return fmt.Errorf("external_anchor.target_descriptor.%s is required", anchorDescriptorFieldTimestampProfile)
	}
	return nil
}

func validatePublicChainDescriptor(raw json.RawMessage) error {
	value := publicChainTargetDescriptor{}
	if err := unmarshalJSONStrict(raw, &value); err != nil {
		return fmt.Errorf("decode external_anchor.target_descriptor public_chain: %w", err)
	}
	if value.DescriptorSchemaID != anchorDescriptorSchemaPublicChainV0 {
		return fmt.Errorf("external_anchor.target_descriptor.%s must be %q", anchorDescriptorFieldDescriptorSchemaID, anchorDescriptorSchemaPublicChainV0)
	}
	if value.ChainNamespace == "" {
		return fmt.Errorf("external_anchor.target_descriptor.%s is required", anchorDescriptorFieldChainNamespace)
	}
	if value.NetworkID == "" {
		return fmt.Errorf("external_anchor.target_descriptor.%s is required", anchorDescriptorFieldNetworkID)
	}
	if _, err := value.SettlementContractDigest.Identity(); err != nil {
		return fmt.Errorf("external_anchor.target_descriptor.%s: %w", anchorDescriptorFieldSettlementDigest, err)
	}
	return nil
}

func computeJSONCanonicalDigest(raw json.RawMessage) (Digest, error) {
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return Digest{}, err
	}
	sum := sha256.Sum256(canonical)
	return Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}
