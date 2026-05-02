package trustpolicy

import (
	"fmt"
	"strings"
)

func EvaluateZKProofSetupIdentityMatch(proof ZKProofArtifactPayload, trusted ZKProofTrustedVerifierPosture) (bool, string, error) {
	if err := ValidateZKProofArtifactPayload(proof); err != nil {
		return false, "", err
	}
	trustedCS, trustedVK, trustedSetup, err := resolveTrustedVerifierPostureDigests(trusted)
	if err != nil {
		return false, "", err
	}
	proofCS, _ := proof.ConstraintSystemDigest.Identity()
	proofVK, _ := proof.VerifierKeyDigest.Identity()
	proofSetup, _ := proof.SetupProvenanceDigest.Identity()
	if proofCS != trustedCS || proofVK != trustedVK || proofSetup != trustedSetup {
		return false, ProofVerificationReasonSetupIdentityMismatch, nil
	}
	return true, ProofVerificationReasonVerified, nil
}

func resolveTrustedVerifierPostureDigests(trusted ZKProofTrustedVerifierPosture) (string, string, string, error) {
	trustedCS, err := trusted.ConstraintSystemDigest.Identity()
	if err != nil {
		return "", "", "", fmt.Errorf("trusted constraint_system_digest: %w", err)
	}
	trustedVK, err := trusted.VerifierKeyDigest.Identity()
	if err != nil {
		return "", "", "", fmt.Errorf("trusted verifier_key_digest: %w", err)
	}
	trustedSetup, err := trusted.SetupProvenanceDigest.Identity()
	if err != nil {
		return "", "", "", fmt.Errorf("trusted setup_provenance_digest: %w", err)
	}
	return trustedCS, trustedVK, trustedSetup, nil
}

func validateAuditProofBindingProjectedBindings(bindings AuditProofBindingProjectedPublicBindings) error {
	if err := validateProjectedBindingRequiredDigests(bindings); err != nil {
		return err
	}
	if strings.TrimSpace(bindings.ProjectSubstrateSnapshotDigest) != "" {
		if err := requireDigestIdentityStringZK(bindings.ProjectSubstrateSnapshotDigest, "project_substrate_snapshot_digest"); err != nil {
			return err
		}
	}
	if bindings.AttestationVerificationRecord != nil {
		if _, err := bindings.AttestationVerificationRecord.Identity(); err != nil {
			return fmt.Errorf("attestation_verification_record_digest: %w", err)
		}
	}
	return nil
}

func validateProjectedBindingRequiredDigests(bindings AuditProofBindingProjectedPublicBindings) error {
	for _, field := range []struct {
		name  string
		value string
	}{{"runtime_image_descriptor_digest", bindings.RuntimeImageDescriptorDigest}, {"attestation_evidence_digest", bindings.AttestationEvidenceDigest}, {"applied_hardening_posture_digest", bindings.AppliedHardeningPostureDigest}, {"session_binding_digest", bindings.SessionBindingDigest}} {
		if err := requireDigestIdentityStringZK(field.value, field.name); err != nil {
			return err
		}
	}
	return nil
}

func validateZKProofSourceRefs(sourceRefs []ZKProofSourceRef) error {
	seen := map[string]struct{}{}
	for i := range sourceRefs {
		if err := validateZKProofSourceRef(sourceRefs[i], i, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateZKProofSourceRef(ref ZKProofSourceRef, index int, seen map[string]struct{}) error {
	if strings.TrimSpace(ref.SourceFamily) == "" {
		return fmt.Errorf("source_refs[%d].source_family is required", index)
	}
	if strings.TrimSpace(ref.SourceRole) == "" {
		return fmt.Errorf("source_refs[%d].source_role is required", index)
	}
	identity, err := ref.SourceDigest.Identity()
	if err != nil {
		return fmt.Errorf("source_refs[%d].source_digest: %w", index, err)
	}
	key := strings.TrimSpace(ref.SourceFamily) + "|" + identity + "|" + strings.TrimSpace(ref.SourceRole)
	if _, ok := seen[key]; ok {
		return fmt.Errorf("source_refs[%d] duplicates source reference key", index)
	}
	seen[key] = struct{}{}
	return nil
}

func requireDigestIdentityStringZK(value string, field string) error {
	trimmed := strings.TrimSpace(value)
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("%s must be digest identity sha256:<64 lowercase hex>", field)
	}
	d := Digest{HashAlg: parts[0], Hash: parts[1]}
	if _, err := d.Identity(); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}
