package brokerapi

import (
	"crypto/sha256"
	"encoding/base64"
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

func buildDeterministicZKProofArtifact(compiled zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract, bindingDigest trustpolicy.Digest) (trustpolicy.ZKProofArtifactPayload, trustpolicy.Digest, error) {
	publicInputs, publicInputCanonical, err := buildDeterministicZKProofPublicInputs(compiled)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, trustpolicy.Digest{}, err
	}
	proofBytes := deterministicProofBytes(publicInputCanonical)
	trusted := trustedVerifierPostureFixtureV0()
	artifact := trustpolicy.ZKProofArtifactPayload{
		SchemaID:               trustpolicy.ZKProofArtifactSchemaID,
		SchemaVersion:          trustpolicy.ZKProofArtifactSchemaVersion,
		StatementFamily:        compiled.PublicInputs.StatementFamily,
		StatementVersion:       compiled.PublicInputs.StatementVersion,
		SchemeID:               zkproof.ProofSchemeIDGroth16V0,
		CurveID:                zkproof.ProofCurveIDBN254V0,
		CircuitID:              zkProofCircuitIDV0,
		ConstraintSystemDigest: trusted.ConstraintSystemDigest,
		VerifierKeyDigest:      trusted.VerifierKeyDigest,
		SetupProvenanceDigest:  trusted.SetupProvenanceDigest,
		NormalizationProfileID: compiled.PublicInputs.NormalizationProfileID,
		SchemeAdapterID:        compiled.PublicInputs.SchemeAdapterID,
		PublicInputs:           publicInputs,
		PublicInputsDigest:     publicInputCanonical,
		ProofBytes:             proofBytes,
		SourceRefs:             []trustpolicy.ZKProofSourceRef{{SourceFamily: "audit_proof_binding", SourceDigest: bindingDigest, SourceRole: "binding"}},
	}
	return artifact, publicInputCanonical, nil
}

func buildDeterministicZKProofPublicInputs(compiled zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract) (map[string]any, trustpolicy.Digest, error) {
	publicInputs := map[string]any{
		"statement_family":                 compiled.PublicInputs.StatementFamily,
		"statement_version":                compiled.PublicInputs.StatementVersion,
		"normalization_profile_id":         compiled.PublicInputs.NormalizationProfileID,
		"scheme_adapter_id":                compiled.PublicInputs.SchemeAdapterID,
		"audit_segment_seal_digest":        mustDigestIdentityString(compiled.PublicInputs.AuditSegmentSealDigest),
		"merkle_root":                      mustDigestIdentityString(compiled.PublicInputs.MerkleRoot),
		"audit_record_digest":              mustDigestIdentityString(compiled.PublicInputs.AuditRecordDigest),
		"protocol_bundle_manifest_hash":    mustDigestIdentityString(compiled.PublicInputs.ProtocolBundleManifestHash),
		"runtime_image_descriptor_digest":  compiled.PublicInputs.RuntimeImageDescriptorDigest,
		"attestation_evidence_digest":      compiled.PublicInputs.AttestationEvidenceDigest,
		"applied_hardening_posture_digest": compiled.PublicInputs.AppliedHardeningPostureDigest,
		"session_binding_digest":           compiled.PublicInputs.SessionBindingDigest,
		"binding_commitment":               compiled.PublicInputs.BindingCommitment,
	}
	if strings.TrimSpace(compiled.PublicInputs.ProjectSubstrateSnapshotDigest) != "" {
		publicInputs["project_substrate_snapshot_digest"] = strings.TrimSpace(compiled.PublicInputs.ProjectSubstrateSnapshotDigest)
	}
	publicInputCanonical, err := canonicalMapDigest(publicInputs)
	if err != nil {
		return nil, trustpolicy.Digest{}, err
	}
	return publicInputs, publicInputCanonical, nil
}

func deterministicProofBytes(publicInputCanonical trustpolicy.Digest) string {
	proofMaterial := sha256.Sum256(append([]byte("runecode.zkproof.fixture.proof.v0:"), []byte(mustDigestIdentityString(publicInputCanonical))...))
	return base64.StdEncoding.EncodeToString(proofMaterial[:])
}

func trustedVerifierPostureFixtureV0() zkproof.TrustedVerifierPosture {
	return zkproof.TrustedVerifierPosture{
		VerifierKeyDigest:      trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("8", 64)},
		ConstraintSystemDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("7", 64)},
		SetupProvenanceDigest:  trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("9", 64)},
	}
}
