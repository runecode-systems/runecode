package zkproof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (identity FrozenCircuitIdentity) ValidateV0() error {
	if strings.TrimSpace(identity.SchemeID) != ProofSchemeIDGroth16V0 {
		return &FeasibilityError{Code: feasibilityCodeUnsupportedProofBackend, Message: fmt.Sprintf("scheme_id must be %q", ProofSchemeIDGroth16V0)}
	}
	if strings.TrimSpace(identity.CurveID) != ProofCurveIDBN254V0 {
		return &FeasibilityError{Code: feasibilityCodeUnsupportedProofBackend, Message: fmt.Sprintf("curve_id must be %q", ProofCurveIDBN254V0)}
	}
	if strings.TrimSpace(identity.CircuitID) == "" {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: "circuit_id is required"}
	}
	if _, err := identity.ConstraintSystemDigest.Identity(); err != nil {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("constraint_system_digest: %v", err)}
	}
	return nil
}

func CanonicalSetupProvenanceDigestV0(lineage SetupLineageIdentity) (trustpolicy.Digest, error) {
	canonical, err := canonicalSetupLineageIdentity(lineage)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	blob, err := json.Marshal(canonical)
	if err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("marshal canonical setup lineage: %w", err)
	}
	sum := sha256.Sum256(blob)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

type canonicalSetupLineage struct {
	Phase1LineageID        string `json:"phase_1_lineage_id"`
	Phase1LineageDigest    string `json:"phase_1_lineage_digest"`
	Phase2TranscriptDigest string `json:"phase_2_transcript_digest"`
	FrozenCircuitSourceDig string `json:"frozen_circuit_source_digest"`
	ConstraintSystemDigest string `json:"constraint_system_digest"`
	GnarkModuleVersion     string `json:"gnark_module_version"`
}

func canonicalSetupLineageIdentity(lineage SetupLineageIdentity) (canonicalSetupLineage, error) {
	phase1ID := strings.TrimSpace(lineage.Phase1LineageID)
	if phase1ID == "" {
		return canonicalSetupLineage{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: "phase_1_lineage_id is required"}
	}
	phase1Digest, phase2Digest, frozenSourceDigest, constraintDigest, err := setupLineageDigestIdentities(lineage)
	if err != nil {
		return canonicalSetupLineage{}, err
	}
	gnarkModuleVersion := strings.TrimSpace(lineage.GnarkModuleVersion)
	if gnarkModuleVersion == "" {
		return canonicalSetupLineage{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: "gnark_module_version is required"}
	}
	return canonicalSetupLineage{Phase1LineageID: phase1ID, Phase1LineageDigest: phase1Digest, Phase2TranscriptDigest: phase2Digest, FrozenCircuitSourceDig: frozenSourceDigest, ConstraintSystemDigest: constraintDigest, GnarkModuleVersion: gnarkModuleVersion}, nil
}

func setupLineageDigestIdentities(lineage SetupLineageIdentity) (string, string, string, string, error) {
	phase1Digest, err := lineage.Phase1LineageDigest.Identity()
	if err != nil {
		return "", "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("phase_1_lineage_digest: %v", err)}
	}
	phase2Digest, err := lineage.Phase2TranscriptDigest.Identity()
	if err != nil {
		return "", "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("phase_2_transcript_digest: %v", err)}
	}
	frozenSourceDigest, err := lineage.FrozenCircuitSourceDig.Identity()
	if err != nil {
		return "", "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("frozen_circuit_source_digest: %v", err)}
	}
	constraintDigest, err := lineage.ConstraintSystemDigest.Identity()
	if err != nil {
		return "", "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("constraint_system_digest: %v", err)}
	}
	return phase1Digest, phase2Digest, frozenSourceDigest, constraintDigest, nil
}

func VerifySetupIdentityMatchesTrustedPostureV0(identity ProofVerificationIdentity, trusted TrustedVerifierPosture) error {
	identityVerifierKey, identityCS, identitySetup, err := proofVerificationDigestIdentities(identity)
	if err != nil {
		return err
	}
	trustedVerifierKey, trustedCS, trustedSetup, err := trustedVerifierDigestIdentities(trusted)
	if err != nil {
		return err
	}
	if identityVerifierKey != trustedVerifierKey {
		return &FeasibilityError{Code: feasibilityCodeSetupIdentityMismatch, Message: "verifier_key_digest mismatch"}
	}
	if identityCS != trustedCS {
		return &FeasibilityError{Code: feasibilityCodeSetupIdentityMismatch, Message: "constraint_system_digest mismatch"}
	}
	if identitySetup != trustedSetup {
		return &FeasibilityError{Code: feasibilityCodeSetupIdentityMismatch, Message: "setup_provenance_digest mismatch"}
	}
	return nil
}

func proofVerificationDigestIdentities(identity ProofVerificationIdentity) (string, string, string, error) {
	verifierKey, err := identity.VerifierKeyDigest.Identity()
	if err != nil {
		return "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("verifier_key_digest: %v", err)}
	}
	constraintSystem, err := identity.ConstraintSystemDigest.Identity()
	if err != nil {
		return "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("constraint_system_digest: %v", err)}
	}
	setup, err := identity.SetupProvenanceDigest.Identity()
	if err != nil {
		return "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("setup_provenance_digest: %v", err)}
	}
	return verifierKey, constraintSystem, setup, nil
}

func trustedVerifierDigestIdentities(trusted TrustedVerifierPosture) (string, string, string, error) {
	verifierKey, err := trusted.VerifierKeyDigest.Identity()
	if err != nil {
		return "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("trusted verifier_key_digest: %v", err)}
	}
	constraintSystem, err := trusted.ConstraintSystemDigest.Identity()
	if err != nil {
		return "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("trusted constraint_system_digest: %v", err)}
	}
	setup, err := trusted.SetupProvenanceDigest.Identity()
	if err != nil {
		return "", "", "", &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("trusted setup_provenance_digest: %v", err)}
	}
	return verifierKey, constraintSystem, setup, nil
}

func resolveProofBackend(backend ProofBackend) ProofBackend {
	if backend == nil {
		return unsupportedProofBackend{}
	}
	return backend
}

func VerifyProofWithTrustedPostureV0(backend ProofBackend, proof []byte, publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs, identity ProofVerificationIdentity, trusted TrustedVerifierPosture) error {
	if err := VerifySetupIdentityMatchesTrustedPostureV0(identity, trusted); err != nil {
		return err
	}
	if _, err := publicInputs.MerkleRoot.Identity(); err != nil {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("public_inputs.merkle_root: %v", err)}
	}
	if _, err := publicInputs.AuditRecordDigest.Identity(); err != nil {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("public_inputs.audit_record_digest: %v", err)}
	}
	if err := requireDigestIdentity(publicInputs.BindingCommitment, "public_inputs.binding_commitment"); err != nil {
		return err
	}
	return resolveProofBackend(backend).VerifyDeterministic(proof, publicInputs)
}
