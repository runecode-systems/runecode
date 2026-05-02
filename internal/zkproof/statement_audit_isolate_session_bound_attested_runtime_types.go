package zkproof

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	StatementFamilyAuditIsolateSessionBoundAttestedRuntimeMembershipV0 = "audit.isolate_session_bound.attested_runtime_membership.v0"
	StatementVersionV0                                                 = "v0"

	ProofSchemeIDGroth16V0 = "groth16"
	ProofCurveIDBN254V0    = "bn254"

	NormalizationProfileAuditIsolateSessionBoundAttestedRuntimeV0 = "runecode.zk.normalize.audit.isolate_session_bound.attested_runtime.v0"
	SchemeAdapterGnarkGroth16IsolateSessionBoundV0                = "runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v0"

	MaxMerklePathDepthV0             = 12
	MerkleAuthenticationPathFormatV1 = "runecode.zkproof.merkle_authentication_path.ordered_sha256_dse_v1"

	VerifierTargetWarmMaxV0Milliseconds       = 100
	VerifierTargetColdMaxV0Milliseconds       = 250
	VerifierRejectInvalidMaxV0Milliseconds    = 50
	VerifierCacheLookupMaxV0Milliseconds      = 10
	ProofArtifactSizeTargetV0MaxBytes         = 16 * 1024
	PublicInputEnvelopeTargetV0MaxBytes       = 4 * 1024
	ProofGenerationWorkerConcurrencyDefaultV0 = 1
)

const (
	feasibilityCodeMissingBoundedInput          = "missing_bounded_input"
	feasibilityCodeIneligibleAuditEvent         = "ineligible_audit_event"
	feasibilityCodeNonDeterministicVerification = "non_deterministic_verification"
	feasibilityCodeUnsupportedProfile           = "unsupported_profile"
	feasibilityCodeInvalidMerklePath            = "invalid_merkle_authentication_path"
	feasibilityCodeUnsupportedCommitmentDeriver = "unsupported_commitment_deriver"
	feasibilityCodeSessionBindingMismatch       = "session_binding_relation_mismatch"
	feasibilityCodeSetupIdentityMismatch        = "setup_identity_mismatch"
	feasibilityCodeUnsupportedProofBackend      = "unsupported_proof_backend"
	feasibilityCodeUnconfiguredProofBackend     = "unconfigured_proof_backend"
)

const (
	merkleSiblingPositionLeft      = "left"
	merkleSiblingPositionRight     = "right"
	merkleSiblingPositionDuplicate = "duplicate"
)

type FeasibilityError struct {
	Code    string
	Message string
}

func (e *FeasibilityError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type MerkleAuthenticationStep struct {
	SiblingDigest   trustpolicy.Digest `json:"sibling_digest"`
	SiblingPosition string             `json:"sibling_position"`
}

type MerkleAuthenticationPath struct {
	PathVersion string                     `json:"path_version"`
	LeafIndex   uint64                     `json:"leaf_index"`
	Steps       []MerkleAuthenticationStep `json:"steps"`
}

type IsolateSessionBoundPrivateRemainder struct {
	RunIDDigest                   trustpolicy.Digest `json:"run_id_digest"`
	IsolateIDDigest               trustpolicy.Digest `json:"isolate_id_digest"`
	SessionIDDigest               trustpolicy.Digest `json:"session_id_digest"`
	BackendKindCode               uint16             `json:"backend_kind_code"`
	IsolationAssuranceLevelCode   uint16             `json:"isolation_assurance_level_code"`
	ProvisioningPostureCode       uint16             `json:"provisioning_posture_code"`
	LaunchContextDigest           trustpolicy.Digest `json:"launch_context_digest"`
	HandshakeTranscriptHashDigest trustpolicy.Digest `json:"handshake_transcript_hash_digest"`
}

const proofDisclosureSemanticsV0 = "proof_disclosure_split_only"

var (
	logicalPublicFieldSetV0  = []string{"runtime_image_descriptor_digest", "attestation_evidence_digest", "applied_hardening_posture_digest", "session_binding_digest", "protocol_bundle_manifest_hash"}
	logicalPrivateFieldSetV0 = []string{"run_id", "isolate_id", "session_id", "backend_kind", "isolation_assurance_level", "provisioning_posture", "launch_context_digest", "handshake_transcript_hash"}
)

type BindingCommitmentDeriver interface {
	DeriveBindingCommitment(adapterProfileID string, normalized IsolateSessionBoundPrivateRemainder) (string, error)
}

type SessionBindingRelationshipVerifier interface {
	VerifyNormalizedPrivateRemainderSessionBinding(normalized IsolateSessionBoundPrivateRemainder, sourceSessionBindingDigest string) error
}

type ProofBackend interface {
	BackendIdentity() string
	VerifyDeterministic(proof []byte, publicInputsDigest trustpolicy.Digest) error
}

type unsupportedProofBackend struct{}

func (unsupportedProofBackend) BackendIdentity() string { return "unsupported" }

func (unsupportedProofBackend) VerifyDeterministic(proof []byte, publicInputsDigest trustpolicy.Digest) error {
	_ = proof
	_ = publicInputsDigest
	return &FeasibilityError{Code: feasibilityCodeUnconfiguredProofBackend, Message: "proof backend is not configured; fail-closed for verification"}
}

type unsupportedBindingCommitmentDeriver struct{}

func (unsupportedBindingCommitmentDeriver) DeriveBindingCommitment(adapterProfileID string, normalized IsolateSessionBoundPrivateRemainder) (string, error) {
	_ = adapterProfileID
	_ = normalized
	return "", &FeasibilityError{Code: feasibilityCodeUnsupportedCommitmentDeriver, Message: "poseidon-family binding commitment deriver is not configured; fail-closed for production use"}
}

type AuditIsolateSessionBoundAttestedRuntimePublicInputs struct {
	StatementFamily                string             `json:"statement_family"`
	StatementVersion               string             `json:"statement_version"`
	NormalizationProfileID         string             `json:"normalization_profile_id"`
	SchemeAdapterID                string             `json:"scheme_adapter_id"`
	AuditSegmentSealDigest         trustpolicy.Digest `json:"audit_segment_seal_digest"`
	MerkleRoot                     trustpolicy.Digest `json:"merkle_root"`
	AuditRecordDigest              trustpolicy.Digest `json:"audit_record_digest"`
	ProtocolBundleManifestHash     trustpolicy.Digest `json:"protocol_bundle_manifest_hash"`
	RuntimeImageDescriptorDigest   string             `json:"runtime_image_descriptor_digest"`
	AttestationEvidenceDigest      string             `json:"attestation_evidence_digest"`
	AppliedHardeningPostureDigest  string             `json:"applied_hardening_posture_digest"`
	SessionBindingDigest           string             `json:"session_binding_digest"`
	BindingCommitment              string             `json:"binding_commitment"`
	ProjectSubstrateSnapshotDigest string             `json:"project_substrate_snapshot_digest,omitempty"`
}

type AuditIsolateSessionBoundAttestedRuntimeWitnessInputs struct {
	PrivateRemainder          IsolateSessionBoundPrivateRemainder `json:"private_remainder"`
	MerkleAuthenticationPath  MerkleAuthenticationPath            `json:"merkle_authentication_path"`
	MerkleAuthenticationDepth int                                 `json:"merkle_authentication_depth"`
}

type AuditIsolateSessionBoundAttestedRuntimeProofInputContract struct {
	PublicInputs  AuditIsolateSessionBoundAttestedRuntimePublicInputs  `json:"public_inputs"`
	WitnessInputs AuditIsolateSessionBoundAttestedRuntimeWitnessInputs `json:"witness_inputs"`
}

type CompileAuditIsolateSessionBoundAttestedRuntimeInput struct {
	DeterministicVerification        bool
	VerifiedAuditEvent               trustpolicy.AuditEventPayload
	VerifiedAuditRecordDigest        trustpolicy.Digest
	VerifiedAuditSegmentSeal         trustpolicy.AuditSegmentSealPayload
	VerifiedAuditSegmentSealDigest   trustpolicy.Digest
	MerkleAuthenticationPath         MerkleAuthenticationPath
	BindingCommitmentDeriver         BindingCommitmentDeriver
	SessionBindingRelationshipVerify SessionBindingRelationshipVerifier
	NormalizationProfileID           string
	SchemeAdapterID                  string
	ProjectSubstrateSnapshotDigest   string
}

type FrozenCircuitIdentity struct {
	SchemeID               string             `json:"scheme_id"`
	CurveID                string             `json:"curve_id"`
	CircuitID              string             `json:"circuit_id"`
	ConstraintSystemDigest trustpolicy.Digest `json:"constraint_system_digest"`
}

type SetupLineageIdentity struct {
	Phase1LineageID        string             `json:"phase_1_lineage_id"`
	Phase1LineageDigest    trustpolicy.Digest `json:"phase_1_lineage_digest"`
	Phase2TranscriptDigest trustpolicy.Digest `json:"phase_2_transcript_digest"`
	FrozenCircuitSourceDig trustpolicy.Digest `json:"frozen_circuit_source_digest"`
	ConstraintSystemDigest trustpolicy.Digest `json:"constraint_system_digest"`
	GnarkModuleVersion     string             `json:"gnark_module_version"`
}

type ProofVerificationIdentity struct {
	VerifierKeyDigest      trustpolicy.Digest `json:"verifier_key_digest"`
	ConstraintSystemDigest trustpolicy.Digest `json:"constraint_system_digest"`
	SetupProvenanceDigest  trustpolicy.Digest `json:"setup_provenance_digest"`
}

type TrustedVerifierPosture struct {
	VerifierKeyDigest      trustpolicy.Digest `json:"verifier_key_digest"`
	ConstraintSystemDigest trustpolicy.Digest `json:"constraint_system_digest"`
	SetupProvenanceDigest  trustpolicy.Digest `json:"setup_provenance_digest"`
}
