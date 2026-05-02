package zkproof

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func CanonicalPublicInputsDigestV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) (trustpolicy.Digest, error) {
	preimage, err := canonicalPublicInputsPreimageV0(publicInputs)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	sum := sha256.Sum256(preimage)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func ValidatePublicInputsDigestBindingV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) error {
	if _, err := publicInputs.PublicInputsDigest.Identity(); err != nil {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("public_inputs_digest: %v", err)}
	}
	want, err := CanonicalPublicInputsDigestV0(publicInputs)
	if err != nil {
		return err
	}
	if publicInputs.PublicInputsDigest != want {
		return &FeasibilityError{Code: "invalid_public_inputs_digest", Message: "public_inputs_digest mismatch against canonical typed public inputs"}
	}
	return nil
}

func canonicalPublicInputsPreimageV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) ([]byte, error) {
	critical, err := canonicalPublicInputsCriticalDigestsV0(publicInputs)
	if err != nil {
		return nil, err
	}
	preimage := make([]byte, 0, 32*9)
	preimage = appendLabeledConstantBytesV0(preimage, "statement_family", []byte(StatementFamilyAuditIsolateSessionBoundAttestedRuntimeMembershipV0))
	preimage = appendLabeledConstantBytesV0(preimage, "statement_version", []byte(StatementVersionV0))
	preimage = appendLabeledConstantBytesV0(preimage, "normalization_profile_id", []byte(NormalizationProfileAuditIsolateSessionBoundAttestedRuntimeV0))
	preimage = appendLabeledConstantBytesV0(preimage, "scheme_adapter_id", []byte(SchemeAdapterGnarkGroth16IsolateSessionBoundV0))
	for _, field := range []struct {
		name  string
		value trustpolicy.Digest
	}{
		{name: "audit_segment_seal_digest", value: critical.AuditSegmentSealDigest},
		{name: "merkle_root", value: critical.MerkleRoot},
		{name: "audit_record_digest", value: critical.AuditRecordDigest},
		{name: "protocol_bundle_manifest_hash", value: critical.ProtocolBundleManifestHash},
		{name: "runtime_image_descriptor_digest", value: critical.RuntimeImageDescriptorDigest},
		{name: "attestation_evidence_digest", value: critical.AttestationEvidenceDigest},
		{name: "applied_hardening_posture_digest", value: critical.AppliedHardeningPostureDigest},
		{name: "session_binding_digest", value: critical.SessionBindingDigest},
		{name: "binding_commitment", value: critical.BindingCommitment},
	} {
		preimage = appendLabeledDigestBytesV0(preimage, field.name, field.value)
	}
	if strings.TrimSpace(publicInputs.ProjectSubstrateSnapshotDigest) != "" {
		return nil, &FeasibilityError{Code: feasibilityCodeUnsupportedProfile, Message: "project_substrate_snapshot_digest is not part of the v0 statement family"}
	}
	return preimage, nil
}

type canonicalPublicInputsCriticalDigests struct {
	AuditSegmentSealDigest        trustpolicy.Digest
	MerkleRoot                    trustpolicy.Digest
	AuditRecordDigest             trustpolicy.Digest
	ProtocolBundleManifestHash    trustpolicy.Digest
	RuntimeImageDescriptorDigest  trustpolicy.Digest
	AttestationEvidenceDigest     trustpolicy.Digest
	AppliedHardeningPostureDigest trustpolicy.Digest
	SessionBindingDigest          trustpolicy.Digest
	BindingCommitment             trustpolicy.Digest
}

type canonicalPublicInputsParsedStrings struct {
	RuntimeImageDescriptorDigest  trustpolicy.Digest
	AttestationEvidenceDigest     trustpolicy.Digest
	AppliedHardeningPostureDigest trustpolicy.Digest
	SessionBindingDigest          trustpolicy.Digest
	BindingCommitment             trustpolicy.Digest
}

func canonicalPublicInputsCriticalDigestsV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) (canonicalPublicInputsCriticalDigests, error) {
	if err := validateCanonicalPublicInputsProfileV0(publicInputs); err != nil {
		return canonicalPublicInputsCriticalDigests{}, err
	}
	parsed, err := parseCanonicalPublicInputsStringDigestsV0(publicInputs)
	if err != nil {
		return canonicalPublicInputsCriticalDigests{}, err
	}
	if err := validateCanonicalPublicInputsDigestIdentitiesV0(publicInputs, parsed); err != nil {
		return canonicalPublicInputsCriticalDigests{}, err
	}
	return canonicalPublicInputsCriticalDigests{
		AuditSegmentSealDigest:        publicInputs.AuditSegmentSealDigest,
		MerkleRoot:                    publicInputs.MerkleRoot,
		AuditRecordDigest:             publicInputs.AuditRecordDigest,
		ProtocolBundleManifestHash:    publicInputs.ProtocolBundleManifestHash,
		RuntimeImageDescriptorDigest:  parsed.RuntimeImageDescriptorDigest,
		AttestationEvidenceDigest:     parsed.AttestationEvidenceDigest,
		AppliedHardeningPostureDigest: parsed.AppliedHardeningPostureDigest,
		SessionBindingDigest:          parsed.SessionBindingDigest,
		BindingCommitment:             parsed.BindingCommitment,
	}, nil
}

func validateCanonicalPublicInputsProfileV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) error {
	for _, field := range []struct {
		name string
		got  string
		want string
	}{
		{name: "statement_family", got: strings.TrimSpace(publicInputs.StatementFamily), want: StatementFamilyAuditIsolateSessionBoundAttestedRuntimeMembershipV0},
		{name: "statement_version", got: strings.TrimSpace(publicInputs.StatementVersion), want: StatementVersionV0},
		{name: "normalization_profile_id", got: strings.TrimSpace(publicInputs.NormalizationProfileID), want: NormalizationProfileAuditIsolateSessionBoundAttestedRuntimeV0},
		{name: "scheme_adapter_id", got: strings.TrimSpace(publicInputs.SchemeAdapterID), want: SchemeAdapterGnarkGroth16IsolateSessionBoundV0},
	} {
		if field.got != field.want {
			return &FeasibilityError{Code: feasibilityCodeUnsupportedProfile, Message: field.name + " mismatch"}
		}
	}
	return nil
}

func parseCanonicalPublicInputsStringDigestsV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) (canonicalPublicInputsParsedStrings, error) {
	runtimeImageDescriptorDigest, err := parseDigestIdentity(publicInputs.RuntimeImageDescriptorDigest, "runtime_image_descriptor_digest")
	if err != nil {
		return canonicalPublicInputsParsedStrings{}, err
	}
	attestationEvidenceDigest, err := parseDigestIdentity(publicInputs.AttestationEvidenceDigest, "attestation_evidence_digest")
	if err != nil {
		return canonicalPublicInputsParsedStrings{}, err
	}
	appliedHardeningPostureDigest, err := parseDigestIdentity(publicInputs.AppliedHardeningPostureDigest, "applied_hardening_posture_digest")
	if err != nil {
		return canonicalPublicInputsParsedStrings{}, err
	}
	sessionBindingDigest, err := parseDigestIdentity(publicInputs.SessionBindingDigest, "session_binding_digest")
	if err != nil {
		return canonicalPublicInputsParsedStrings{}, err
	}
	bindingCommitment, err := parseDigestIdentity(publicInputs.BindingCommitment, "binding_commitment")
	if err != nil {
		return canonicalPublicInputsParsedStrings{}, err
	}
	return canonicalPublicInputsParsedStrings{
		RuntimeImageDescriptorDigest:  runtimeImageDescriptorDigest,
		AttestationEvidenceDigest:     attestationEvidenceDigest,
		AppliedHardeningPostureDigest: appliedHardeningPostureDigest,
		SessionBindingDigest:          sessionBindingDigest,
		BindingCommitment:             bindingCommitment,
	}, nil
}

func validateCanonicalPublicInputsDigestIdentitiesV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs, parsed canonicalPublicInputsParsedStrings) error {
	for _, field := range []struct {
		name  string
		value trustpolicy.Digest
	}{
		{name: "audit_segment_seal_digest", value: publicInputs.AuditSegmentSealDigest},
		{name: "merkle_root", value: publicInputs.MerkleRoot},
		{name: "audit_record_digest", value: publicInputs.AuditRecordDigest},
		{name: "protocol_bundle_manifest_hash", value: publicInputs.ProtocolBundleManifestHash},
		{name: "runtime_image_descriptor_digest", value: parsed.RuntimeImageDescriptorDigest},
		{name: "attestation_evidence_digest", value: parsed.AttestationEvidenceDigest},
		{name: "applied_hardening_posture_digest", value: parsed.AppliedHardeningPostureDigest},
		{name: "session_binding_digest", value: parsed.SessionBindingDigest},
		{name: "binding_commitment", value: parsed.BindingCommitment},
	} {
		if _, err := field.value.Identity(); err != nil {
			return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s: %v", field.name, err)}
		}
	}
	return nil
}

func appendLabeledConstantBytesV0(dst []byte, label string, value []byte) []byte {
	dst = append(dst, []byte(label)...)
	dst = append(dst, '=')
	dst = append(dst, value...)
	return append(dst, '|')
}

func appendLabeledDigestBytesV0(dst []byte, label string, value trustpolicy.Digest) []byte {
	dst = append(dst, []byte(label)...)
	dst = append(dst, '=')
	raw, _ := hex.DecodeString(value.Hash)
	dst = append(dst, raw...)
	return append(dst, '|')
}
