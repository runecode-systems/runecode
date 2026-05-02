package trustpolicy

const (
	AuditProofBindingSchemaID      = "runecode.protocol.v0.AuditProofBinding"
	AuditProofBindingSchemaVersion = "0.1.0"

	ZKProofArtifactSchemaID      = "runecode.protocol.v0.ZKProofArtifact"
	ZKProofArtifactSchemaVersion = "0.1.0"

	ZKProofVerificationRecordSchemaID      = "runecode.protocol.v0.ZKProofVerificationRecord"
	ZKProofVerificationRecordSchemaVersion = "0.1.0"

	ProofVerificationOutcomeVerified = "verified"
	ProofVerificationOutcomeRejected = "rejected"

	ProofVerificationReasonVerified                = "verified"
	ProofVerificationReasonProofInvalid            = "proof_invalid"
	ProofVerificationReasonSetupIdentityMismatch   = "setup_identity_mismatch"
	ProofVerificationReasonUnsupportedProfile      = "unsupported_profile"
	ProofVerificationReasonInvalidPublicInputsHash = "invalid_public_inputs_digest"
	ProofVerificationReasonUnconfiguredBackend     = "unconfigured_backend"
)

var proofVerificationAllowedReasonCodes = map[string]struct{}{
	ProofVerificationReasonVerified:                {},
	ProofVerificationReasonProofInvalid:            {},
	ProofVerificationReasonSetupIdentityMismatch:   {},
	ProofVerificationReasonUnsupportedProfile:      {},
	ProofVerificationReasonInvalidPublicInputsHash: {},
	ProofVerificationReasonUnconfiguredBackend:     {},
}

type ZKProofSourceRef struct {
	SourceFamily string `json:"source_family"`
	SourceDigest Digest `json:"source_digest"`
	SourceRole   string `json:"source_role"`
}

type ZKProofArtifactPayload struct {
	SchemaID               string             `json:"schema_id"`
	SchemaVersion          string             `json:"schema_version"`
	StatementFamily        string             `json:"statement_family"`
	StatementVersion       string             `json:"statement_version"`
	SchemeID               string             `json:"scheme_id"`
	CurveID                string             `json:"curve_id"`
	CircuitID              string             `json:"circuit_id"`
	ConstraintSystemDigest Digest             `json:"constraint_system_digest"`
	VerifierKeyDigest      Digest             `json:"verifier_key_digest"`
	SetupProvenanceDigest  Digest             `json:"setup_provenance_digest"`
	NormalizationProfileID string             `json:"normalization_profile_id"`
	SchemeAdapterID        string             `json:"scheme_adapter_id"`
	PublicInputs           map[string]any     `json:"public_inputs"`
	PublicInputsDigest     Digest             `json:"public_inputs_digest"`
	ProofBytes             string             `json:"proof_bytes"`
	SourceRefs             []ZKProofSourceRef `json:"source_refs"`
}

type AuditProofBindingProjectedPublicBindings struct {
	RuntimeImageDescriptorDigest   string  `json:"runtime_image_descriptor_digest"`
	AttestationEvidenceDigest      string  `json:"attestation_evidence_digest"`
	AppliedHardeningPostureDigest  string  `json:"applied_hardening_posture_digest"`
	SessionBindingDigest           string  `json:"session_binding_digest"`
	ProjectSubstrateSnapshotDigest string  `json:"project_substrate_snapshot_digest,omitempty"`
	AttestationVerificationRecord  *Digest `json:"attestation_verification_record_digest,omitempty"`
}

type AuditProofBindingMerkleAuthenticationStep struct {
	SiblingDigest   Digest `json:"sibling_digest"`
	SiblingPosition string `json:"sibling_position"`
}

type AuditProofBindingPayload struct {
	SchemaID                 string                                      `json:"schema_id"`
	SchemaVersion            string                                      `json:"schema_version"`
	StatementFamily          string                                      `json:"statement_family"`
	StatementVersion         string                                      `json:"statement_version"`
	NormalizationProfileID   string                                      `json:"normalization_profile_id"`
	SchemeAdapterID          string                                      `json:"scheme_adapter_id"`
	AuditRecordDigest        Digest                                      `json:"audit_record_digest"`
	AuditSegmentSealDigest   Digest                                      `json:"audit_segment_seal_digest"`
	MerkleRoot               Digest                                      `json:"merkle_root"`
	ProtocolBundleManifest   Digest                                      `json:"protocol_bundle_manifest_hash"`
	BindingCommitment        string                                      `json:"binding_commitment"`
	ProjectedPublicBindings  AuditProofBindingProjectedPublicBindings    `json:"projected_public_bindings"`
	MerklePathVersion        string                                      `json:"merkle_path_version"`
	MerkleAuthenticationPath []AuditProofBindingMerkleAuthenticationStep `json:"merkle_authentication_path"`
	MerklePathDepth          int                                         `json:"merkle_path_depth"`
	LeafIndex                int                                         `json:"leaf_index"`
	SourceRefs               []ZKProofSourceRef                          `json:"source_refs,omitempty"`
}

type ZKProofVerificationRecordPayload struct {
	SchemaID                 string   `json:"schema_id"`
	SchemaVersion            string   `json:"schema_version"`
	ProofDigest              Digest   `json:"proof_digest"`
	StatementFamily          string   `json:"statement_family"`
	StatementVersion         string   `json:"statement_version"`
	SchemeID                 string   `json:"scheme_id"`
	CurveID                  string   `json:"curve_id"`
	CircuitID                string   `json:"circuit_id"`
	ConstraintSystemDigest   Digest   `json:"constraint_system_digest"`
	VerifierKeyDigest        Digest   `json:"verifier_key_digest"`
	SetupProvenanceDigest    Digest   `json:"setup_provenance_digest"`
	NormalizationProfileID   string   `json:"normalization_profile_id"`
	SchemeAdapterID          string   `json:"scheme_adapter_id"`
	PublicInputsDigest       Digest   `json:"public_inputs_digest"`
	VerifierImplementationID string   `json:"verifier_implementation_id"`
	VerifiedAt               string   `json:"verified_at"`
	VerificationOutcome      string   `json:"verification_outcome"`
	ReasonCodes              []string `json:"reason_codes"`
	CacheProvenance          string   `json:"cache_provenance,omitempty"`
}

type ZKProofTrustedVerifierPosture struct {
	ConstraintSystemDigest Digest
	VerifierKeyDigest      Digest
	SetupProvenanceDigest  Digest
}
