package zkproof

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

// CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0 compiles a narrow,
// bounded proof-input contract from already-verified trusted audit objects.
func CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0(input CompileAuditIsolateSessionBoundAttestedRuntimeInput) (AuditIsolateSessionBoundAttestedRuntimeProofInputContract, error) {
	if !input.DeterministicVerification {
		return AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, &FeasibilityError{Code: feasibilityCodeNonDeterministicVerification, Message: "deterministic trusted verification is required"}
	}
	normalizationProfileID, schemeAdapterID, err := resolveCompileProfiles(input)
	if err != nil {
		return AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, err
	}
	if err := validateCompileBoundedInputs(input); err != nil {
		return AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, err
	}
	payload, err := decodeEligibleIsolateSessionBoundPayload(input.VerifiedAuditEvent)
	if err != nil {
		return AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, err
	}
	normalizedPrivate, bindingCommitment, err := compileWitnessBinding(input, payload, schemeAdapterID)
	if err != nil {
		return AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, err
	}
	return buildCompileContract(input, payload, normalizedPrivate, normalizationProfileID, schemeAdapterID, bindingCommitment), nil
}

func resolveCompileProfiles(input CompileAuditIsolateSessionBoundAttestedRuntimeInput) (string, string, error) {
	normalizationProfileID := strings.TrimSpace(input.NormalizationProfileID)
	if normalizationProfileID == "" {
		normalizationProfileID = NormalizationProfileAuditIsolateSessionBoundAttestedRuntimeV0
	}
	if normalizationProfileID != NormalizationProfileAuditIsolateSessionBoundAttestedRuntimeV0 {
		return "", "", &FeasibilityError{Code: feasibilityCodeUnsupportedProfile, Message: fmt.Sprintf("unsupported normalization_profile_id %q", normalizationProfileID)}
	}
	schemeAdapterID := strings.TrimSpace(input.SchemeAdapterID)
	if schemeAdapterID == "" {
		schemeAdapterID = SchemeAdapterGnarkGroth16IsolateSessionBoundV0
	}
	if schemeAdapterID != SchemeAdapterGnarkGroth16IsolateSessionBoundV0 {
		return "", "", &FeasibilityError{Code: feasibilityCodeUnsupportedProfile, Message: fmt.Sprintf("unsupported scheme_adapter_id %q", schemeAdapterID)}
	}
	if !isSupportedLogicalNormalizationProfile(normalizationProfileID) || !isSupportedSchemeAdapterProfile(schemeAdapterID) {
		return "", "", &FeasibilityError{Code: feasibilityCodeUnsupportedProfile, Message: "unsupported normalization or scheme adapter contract"}
	}
	return normalizationProfileID, schemeAdapterID, nil
}

func validateCompileBoundedInputs(input CompileAuditIsolateSessionBoundAttestedRuntimeInput) error {
	if _, err := input.VerifiedAuditRecordDigest.Identity(); err != nil {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("audit_record_digest: %v", err)}
	}
	if _, err := input.VerifiedAuditSegmentSealDigest.Identity(); err != nil {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("audit_segment_seal_digest: %v", err)}
	}
	if err := validateSealEligibility(input.VerifiedAuditSegmentSeal); err != nil {
		return err
	}
	if err := validateMerkleAuthenticationPath(input.MerkleAuthenticationPath); err != nil {
		return err
	}
	return VerifyAuditSegmentMerkleAuthenticationPathAgainstSealV0(input.VerifiedAuditRecordDigest, input.MerkleAuthenticationPath, input.VerifiedAuditSegmentSeal)
}

func compileWitnessBinding(input CompileAuditIsolateSessionBoundAttestedRuntimeInput, payload trustpolicy.IsolateSessionBoundPayload, schemeAdapterID string) (IsolateSessionBoundPrivateRemainder, string, error) {
	normalizedPrivate, err := normalizePrivateRemainderV0(payload)
	if err != nil {
		return IsolateSessionBoundPrivateRemainder{}, "", err
	}
	if input.SessionBindingRelationshipVerify == nil {
		return IsolateSessionBoundPrivateRemainder{}, "", &FeasibilityError{Code: feasibilityCodeSessionBindingMismatch, Message: "trusted off-circuit session_binding_digest relationship verifier is required"}
	}
	if err := input.SessionBindingRelationshipVerify.VerifyNormalizedPrivateRemainderSessionBinding(normalizedPrivate, payload.SessionBindingDigest); err != nil {
		return IsolateSessionBoundPrivateRemainder{}, "", &FeasibilityError{Code: feasibilityCodeSessionBindingMismatch, Message: fmt.Sprintf("session_binding_digest relationship verification failed: %v", err)}
	}
	commitmentDeriver := input.BindingCommitmentDeriver
	if commitmentDeriver == nil {
		commitmentDeriver = unsupportedBindingCommitmentDeriver{}
	}
	bindingCommitment, err := commitmentDeriver.DeriveBindingCommitment(schemeAdapterID, normalizedPrivate)
	if err != nil {
		return IsolateSessionBoundPrivateRemainder{}, "", err
	}
	bindingCommitment = strings.TrimSpace(bindingCommitment)
	if err := requireDigestIdentity(bindingCommitment, "binding_commitment"); err != nil {
		return IsolateSessionBoundPrivateRemainder{}, "", err
	}
	return normalizedPrivate, bindingCommitment, nil
}

func buildCompileContract(input CompileAuditIsolateSessionBoundAttestedRuntimeInput, payload trustpolicy.IsolateSessionBoundPayload, normalizedPrivate IsolateSessionBoundPrivateRemainder, normalizationProfileID, schemeAdapterID, bindingCommitment string) AuditIsolateSessionBoundAttestedRuntimeProofInputContract {
	publicInputs := AuditIsolateSessionBoundAttestedRuntimePublicInputs{
		StatementFamily:                StatementFamilyAuditIsolateSessionBoundAttestedRuntimeMembershipV0,
		StatementVersion:               StatementVersionV0,
		NormalizationProfileID:         normalizationProfileID,
		SchemeAdapterID:                schemeAdapterID,
		AuditSegmentSealDigest:         input.VerifiedAuditSegmentSealDigest,
		MerkleRoot:                     input.VerifiedAuditSegmentSeal.MerkleRoot,
		AuditRecordDigest:              input.VerifiedAuditRecordDigest,
		ProtocolBundleManifestHash:     input.VerifiedAuditEvent.ProtocolBundleManifestHash,
		RuntimeImageDescriptorDigest:   payload.RuntimeImageDescriptorDigest,
		AttestationEvidenceDigest:      payload.AttestationEvidenceDigest,
		AppliedHardeningPostureDigest:  payload.AppliedHardeningPostureDigest,
		SessionBindingDigest:           payload.SessionBindingDigest,
		BindingCommitment:              bindingCommitment,
		ProjectSubstrateSnapshotDigest: strings.TrimSpace(input.ProjectSubstrateSnapshotDigest),
	}
	return AuditIsolateSessionBoundAttestedRuntimeProofInputContract{PublicInputs: publicInputs, WitnessInputs: AuditIsolateSessionBoundAttestedRuntimeWitnessInputs{PrivateRemainder: normalizedPrivate, MerkleAuthenticationPath: input.MerkleAuthenticationPath, MerkleAuthenticationDepth: len(input.MerkleAuthenticationPath.Steps)}}
}

func isSupportedLogicalNormalizationProfile(profileID string) bool {
	return profileID == NormalizationProfileAuditIsolateSessionBoundAttestedRuntimeV0 && proofDisclosureSemanticsV0 == "proof_disclosure_split_only" && len(logicalPublicFieldSetV0) == 5 && len(logicalPrivateFieldSetV0) == 8
}

func isSupportedSchemeAdapterProfile(adapterID string) bool {
	return adapterID == SchemeAdapterGnarkGroth16IsolateSessionBoundV0
}
