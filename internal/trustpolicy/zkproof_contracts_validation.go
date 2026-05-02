package trustpolicy

import (
	"fmt"
	"strings"
	"time"
)

func ValidateZKProofArtifactPayload(payload ZKProofArtifactPayload) error {
	if err := validateSchemaIdentityPair(payload.SchemaID, ZKProofArtifactSchemaID, payload.SchemaVersion, ZKProofArtifactSchemaVersion, "proof artifact"); err != nil {
		return err
	}
	if err := requireZKProofArtifactStrings(payload); err != nil {
		return err
	}
	if len(payload.PublicInputs) == 0 {
		return fmt.Errorf("public_inputs is required")
	}
	if err := validateNamedDigests([]namedDigest{{"constraint_system_digest", payload.ConstraintSystemDigest}, {"verifier_key_digest", payload.VerifierKeyDigest}, {"setup_provenance_digest", payload.SetupProvenanceDigest}, {"public_inputs_digest", payload.PublicInputsDigest}}); err != nil {
		return err
	}
	if len(payload.SourceRefs) == 0 {
		return fmt.Errorf("source_refs is required")
	}
	return validateZKProofSourceRefs(payload.SourceRefs)
}

func ValidateAuditProofBindingPayload(payload AuditProofBindingPayload) error {
	if err := validateSchemaIdentityPair(payload.SchemaID, AuditProofBindingSchemaID, payload.SchemaVersion, AuditProofBindingSchemaVersion, "audit proof binding"); err != nil {
		return err
	}
	if err := requireAuditProofBindingStrings(payload); err != nil {
		return err
	}
	if err := validateAuditProofBindingDigests(payload); err != nil {
		return err
	}
	if err := requireDigestIdentityStringZK(payload.BindingCommitment, "binding_commitment"); err != nil {
		return err
	}
	if err := validateAuditProofBindingProjectedBindings(payload.ProjectedPublicBindings); err != nil {
		return err
	}
	if err := validateAuditProofBindingMerkleBounds(payload); err != nil {
		return err
	}
	if len(payload.SourceRefs) > 0 {
		return validateZKProofSourceRefs(payload.SourceRefs)
	}
	return nil
}

func ValidateZKProofVerificationRecordPayload(payload ZKProofVerificationRecordPayload) error {
	if err := validateSchemaIdentityPair(payload.SchemaID, ZKProofVerificationRecordSchemaID, payload.SchemaVersion, ZKProofVerificationRecordSchemaVersion, "proof verification record"); err != nil {
		return err
	}
	if err := requireZKProofVerificationStrings(payload); err != nil {
		return err
	}
	if err := validateZKProofVerificationDigests(payload); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.VerifiedAt)); err != nil {
		return fmt.Errorf("verified_at: %w", err)
	}
	if err := validateVerificationOutcome(payload.VerificationOutcome); err != nil {
		return err
	}
	if len(payload.ReasonCodes) == 0 {
		return fmt.Errorf("reason_codes is required")
	}
	seen, err := validateVerificationReasonCodes(payload.ReasonCodes)
	if err != nil {
		return err
	}
	if err := validateOutcomeReasonConsistency(strings.TrimSpace(payload.VerificationOutcome), seen); err != nil {
		return err
	}
	if payload.CacheProvenance != "" {
		return validateCacheProvenance(payload.CacheProvenance)
	}
	return nil
}

type namedDigest struct {
	name   string
	digest Digest
}

func requireZKProofArtifactStrings(payload ZKProofArtifactPayload) error {
	return requireNamedStrings([]struct {
		name  string
		value string
	}{{"statement_family", payload.StatementFamily}, {"statement_version", payload.StatementVersion}, {"scheme_id", payload.SchemeID}, {"curve_id", payload.CurveID}, {"circuit_id", payload.CircuitID}, {"normalization_profile_id", payload.NormalizationProfileID}, {"scheme_adapter_id", payload.SchemeAdapterID}, {"proof_bytes", payload.ProofBytes}})
}

func requireAuditProofBindingStrings(payload AuditProofBindingPayload) error {
	return requireNamedStrings([]struct {
		name  string
		value string
	}{{"statement_family", payload.StatementFamily}, {"statement_version", payload.StatementVersion}, {"normalization_profile_id", payload.NormalizationProfileID}, {"scheme_adapter_id", payload.SchemeAdapterID}, {"merkle_path_version", payload.MerklePathVersion}})
}

func validateAuditProofBindingDigests(payload AuditProofBindingPayload) error {
	return validateNamedDigests([]namedDigest{{"audit_record_digest", payload.AuditRecordDigest}, {"audit_segment_seal_digest", payload.AuditSegmentSealDigest}, {"merkle_root", payload.MerkleRoot}, {"protocol_bundle_manifest_hash", payload.ProtocolBundleManifest}})
}

func validateAuditProofBindingMerkleBounds(payload AuditProofBindingPayload) error {
	if payload.MerklePathDepth < 0 || payload.MerklePathDepth > 12 {
		return fmt.Errorf("merkle_path_depth must be between 0 and 12")
	}
	if payload.LeafIndex < 0 || payload.LeafIndex > 4095 {
		return fmt.Errorf("leaf_index must be between 0 and 4095")
	}
	if len(payload.MerkleAuthenticationPath) != payload.MerklePathDepth {
		return fmt.Errorf("merkle_authentication_path length must equal merkle_path_depth")
	}
	return validateMerkleAuthenticationPath(payload.MerkleAuthenticationPath)
}

func requireZKProofVerificationStrings(payload ZKProofVerificationRecordPayload) error {
	return requireNamedStrings([]struct {
		name  string
		value string
	}{{"statement_family", payload.StatementFamily}, {"statement_version", payload.StatementVersion}, {"scheme_id", payload.SchemeID}, {"curve_id", payload.CurveID}, {"circuit_id", payload.CircuitID}, {"normalization_profile_id", payload.NormalizationProfileID}, {"scheme_adapter_id", payload.SchemeAdapterID}, {"verifier_implementation_id", payload.VerifierImplementationID}})
}

func validateZKProofVerificationDigests(payload ZKProofVerificationRecordPayload) error {
	return validateNamedDigests([]namedDigest{{"proof_digest", payload.ProofDigest}, {"constraint_system_digest", payload.ConstraintSystemDigest}, {"verifier_key_digest", payload.VerifierKeyDigest}, {"setup_provenance_digest", payload.SetupProvenanceDigest}, {"public_inputs_digest", payload.PublicInputsDigest}})
}

func validateSchemaIdentityPair(schemaID, wantSchemaID, schemaVersion, wantSchemaVersion, prefix string) error {
	if strings.TrimSpace(schemaID) != wantSchemaID {
		return fmt.Errorf("%s schema_id must be %q", prefix, wantSchemaID)
	}
	if strings.TrimSpace(schemaVersion) != wantSchemaVersion {
		return fmt.Errorf("%s schema_version must be %q", prefix, wantSchemaVersion)
	}
	return nil
}

func requireNamedStrings(fields []struct {
	name  string
	value string
}) error {
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	return nil
}

func validateNamedDigests(fields []namedDigest) error {
	for _, field := range fields {
		if _, err := field.digest.Identity(); err != nil {
			return fmt.Errorf("%s: %w", field.name, err)
		}
	}
	return nil
}

func validateMerkleAuthenticationPath(path []AuditProofBindingMerkleAuthenticationStep) error {
	for i := range path {
		step := path[i]
		if _, err := step.SiblingDigest.Identity(); err != nil {
			return fmt.Errorf("merkle_authentication_path[%d].sibling_digest: %w", i, err)
		}
		if err := validateMerkleSiblingPosition(i, step.SiblingPosition); err != nil {
			return err
		}
	}
	return nil
}

func validateMerkleSiblingPosition(index int, position string) error {
	switch strings.TrimSpace(position) {
	case "left", "right", "duplicate":
		return nil
	default:
		return fmt.Errorf("merkle_authentication_path[%d].sibling_position %q is unsupported", index, position)
	}
}

func validateVerificationOutcome(outcome string) error {
	switch strings.TrimSpace(outcome) {
	case ProofVerificationOutcomeVerified, ProofVerificationOutcomeRejected:
		return nil
	default:
		return fmt.Errorf("verification_outcome %q is unsupported", outcome)
	}
}

func validateVerificationReasonCodes(reasonCodes []string) (map[string]struct{}, error) {
	seen := map[string]struct{}{}
	for i := range reasonCodes {
		code := strings.TrimSpace(reasonCodes[i])
		if _, ok := proofVerificationAllowedReasonCodes[code]; !ok {
			return nil, fmt.Errorf("reason_codes[%d] %q is unsupported", i, reasonCodes[i])
		}
		if _, ok := seen[code]; ok {
			return nil, fmt.Errorf("reason_codes[%d] duplicates code %q", i, code)
		}
		seen[code] = struct{}{}
	}
	return seen, nil
}

func validateOutcomeReasonConsistency(outcome string, seen map[string]struct{}) error {
	if outcome == ProofVerificationOutcomeVerified {
		if _, ok := seen[ProofVerificationReasonVerified]; !ok {
			return fmt.Errorf("verification_outcome=verified requires reason code %q", ProofVerificationReasonVerified)
		}
		return nil
	}
	if _, ok := seen[ProofVerificationReasonVerified]; ok {
		return fmt.Errorf("verification_outcome=rejected cannot include reason code %q", ProofVerificationReasonVerified)
	}
	return nil
}

func validateCacheProvenance(cacheProvenance string) error {
	switch strings.TrimSpace(cacheProvenance) {
	case "fresh", "cache_hit":
		return nil
	default:
		return fmt.Errorf("cache_provenance %q is unsupported", cacheProvenance)
	}
}
