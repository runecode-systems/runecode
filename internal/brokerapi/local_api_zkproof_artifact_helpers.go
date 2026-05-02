package brokerapi

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
)

func toTrustpolicyMerklePath(path zkproof.MerkleAuthenticationPath) []trustpolicy.AuditProofBindingMerkleAuthenticationStep {
	steps := make([]trustpolicy.AuditProofBindingMerkleAuthenticationStep, 0, len(path.Steps))
	for _, step := range path.Steps {
		steps = append(steps, trustpolicy.AuditProofBindingMerkleAuthenticationStep{SiblingDigest: step.SiblingDigest, SiblingPosition: step.SiblingPosition})
	}
	return steps
}

func buildZKProofArtifact(compiled zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract, bindingDigest trustpolicy.Digest, backend zkproof.ProofProver, _ zkproof.FrozenCircuitIdentity, trusted zkproof.TrustedVerifierPosture) (trustpolicy.ZKProofArtifactPayload, error) {
	publicInputs, publicInputCanonical, err := buildZKProofPublicInputs(compiled.PublicInputs)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, err
	}
	proofBytes, identity, err := backend.ProveDeterministic(compiled)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, err
	}
	if err := zkproof.VerifySetupIdentityMatchesTrustedPostureV0(identity, trusted); err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, err
	}
	artifact := trustpolicy.ZKProofArtifactPayload{
		SchemaID:               trustpolicy.ZKProofArtifactSchemaID,
		SchemaVersion:          trustpolicy.ZKProofArtifactSchemaVersion,
		StatementFamily:        compiled.PublicInputs.StatementFamily,
		StatementVersion:       compiled.PublicInputs.StatementVersion,
		SchemeID:               zkproof.ProofSchemeIDGroth16V0,
		CurveID:                zkproof.ProofCurveIDBN254V0,
		CircuitID:              zkProofCircuitIDV0,
		ConstraintSystemDigest: identity.ConstraintSystemDigest,
		VerifierKeyDigest:      identity.VerifierKeyDigest,
		SetupProvenanceDigest:  identity.SetupProvenanceDigest,
		NormalizationProfileID: compiled.PublicInputs.NormalizationProfileID,
		SchemeAdapterID:        compiled.PublicInputs.SchemeAdapterID,
		PublicInputs:           publicInputs,
		PublicInputsDigest:     publicInputCanonical,
		ProofBytes:             base64.StdEncoding.EncodeToString(proofBytes),
		SourceRefs:             []trustpolicy.ZKProofSourceRef{{SourceFamily: "audit_proof_binding", SourceDigest: bindingDigest, SourceRole: "binding"}},
	}
	return artifact, nil
}

func buildZKProofPublicInputs(publicInputs zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs) (map[string]any, trustpolicy.Digest, error) {
	encoded := map[string]any{
		"statement_family":                 publicInputs.StatementFamily,
		"statement_version":                publicInputs.StatementVersion,
		"normalization_profile_id":         publicInputs.NormalizationProfileID,
		"scheme_adapter_id":                publicInputs.SchemeAdapterID,
		"audit_segment_seal_digest":        mustDigestIdentityString(publicInputs.AuditSegmentSealDigest),
		"merkle_root":                      mustDigestIdentityString(publicInputs.MerkleRoot),
		"audit_record_digest":              mustDigestIdentityString(publicInputs.AuditRecordDigest),
		"protocol_bundle_manifest_hash":    mustDigestIdentityString(publicInputs.ProtocolBundleManifestHash),
		"runtime_image_descriptor_digest":  strings.TrimSpace(publicInputs.RuntimeImageDescriptorDigest),
		"attestation_evidence_digest":      strings.TrimSpace(publicInputs.AttestationEvidenceDigest),
		"applied_hardening_posture_digest": strings.TrimSpace(publicInputs.AppliedHardeningPostureDigest),
		"session_binding_digest":           strings.TrimSpace(publicInputs.SessionBindingDigest),
		"binding_commitment":               strings.TrimSpace(publicInputs.BindingCommitment),
	}
	if strings.TrimSpace(publicInputs.ProjectSubstrateSnapshotDigest) != "" {
		encoded["project_substrate_snapshot_digest"] = strings.TrimSpace(publicInputs.ProjectSubstrateSnapshotDigest)
	}
	digest, err := canonicalMapDigest(encoded)
	if err != nil {
		return nil, trustpolicy.Digest{}, err
	}
	return encoded, digest, nil
}

func decodeArtifactPublicInputs(raw map[string]any, publicInputsDigest trustpolicy.Digest) (zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs, error) {
	recomputedDigest, err := canonicalMapDigest(raw)
	if err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs{}, err
	}
	if recomputedDigest != publicInputsDigest {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs{}, &zkproof.FeasibilityError{Code: "invalid_public_inputs_digest", Message: "proof public_inputs_digest does not match canonical public_inputs content"}
	}
	parsed := zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs{}
	if err := assignArtifactStringFields(raw, &parsed); err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs{}, err
	}
	if err := assignArtifactDigestFields(raw, &parsed); err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs{}, err
	}
	return parsed, nil
}

func assignArtifactStringFields(raw map[string]any, parsed *zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs) error {
	if err := assignArtifactIdentityStrings(raw, parsed); err != nil {
		return err
	}
	return assignArtifactBindingStrings(raw, parsed)
}

func assignArtifactIdentityStrings(raw map[string]any, parsed *zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs) error {
	var err error
	parsed.StatementFamily, err = requiredStringField(raw, "statement_family")
	if err != nil {
		return err
	}
	parsed.StatementVersion, err = requiredStringField(raw, "statement_version")
	if err != nil {
		return err
	}
	parsed.NormalizationProfileID, err = requiredStringField(raw, "normalization_profile_id")
	if err != nil {
		return err
	}
	parsed.SchemeAdapterID, err = requiredStringField(raw, "scheme_adapter_id")
	if err != nil {
		return err
	}
	parsed.RuntimeImageDescriptorDigest, err = requiredStringField(raw, "runtime_image_descriptor_digest")
	return err
}

func assignArtifactBindingStrings(raw map[string]any, parsed *zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs) error {
	var err error
	parsed.AttestationEvidenceDigest, err = requiredStringField(raw, "attestation_evidence_digest")
	if err != nil {
		return err
	}
	parsed.AppliedHardeningPostureDigest, err = requiredStringField(raw, "applied_hardening_posture_digest")
	if err != nil {
		return err
	}
	parsed.SessionBindingDigest, err = requiredStringField(raw, "session_binding_digest")
	if err != nil {
		return err
	}
	parsed.BindingCommitment, err = requiredStringField(raw, "binding_commitment")
	if err != nil {
		return err
	}
	parsed.ProjectSubstrateSnapshotDigest, err = optionalStringField(raw, "project_substrate_snapshot_digest")
	return err
}

func assignArtifactDigestFields(raw map[string]any, parsed *zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs) error {
	var err error
	parsed.AuditSegmentSealDigest, err = requiredDigestField(raw, "audit_segment_seal_digest")
	if err != nil {
		return err
	}
	parsed.MerkleRoot, err = requiredDigestField(raw, "merkle_root")
	if err != nil {
		return err
	}
	parsed.AuditRecordDigest, err = requiredDigestField(raw, "audit_record_digest")
	if err != nil {
		return err
	}
	parsed.ProtocolBundleManifestHash, err = requiredDigestField(raw, "protocol_bundle_manifest_hash")
	return err
}

func requiredStringField(raw map[string]any, key string) (string, error) {
	value, ok := raw[key]
	if !ok {
		return "", fmt.Errorf("public_inputs.%s is required", key)
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("public_inputs.%s must be a non-empty string", key)
	}
	return strings.TrimSpace(text), nil
}

func optionalStringField(raw map[string]any, key string) (string, error) {
	value, ok := raw[key]
	if !ok || value == nil {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("public_inputs.%s must be a string", key)
	}
	return strings.TrimSpace(text), nil
}

func requiredDigestField(raw map[string]any, key string) (trustpolicy.Digest, error) {
	identity, err := requiredStringField(raw, key)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	parts := strings.SplitN(identity, ":", 2)
	if len(parts) != 2 {
		return trustpolicy.Digest{}, fmt.Errorf("public_inputs.%s must be sha256:<64 lowercase hex>", key)
	}
	d := trustpolicy.Digest{HashAlg: parts[0], Hash: parts[1]}
	if _, err := d.Identity(); err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("public_inputs.%s: %w", key, err)
	}
	return d, nil
}
