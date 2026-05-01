package launcherbackend

import (
	"fmt"
	"strings"
)

func ValidateIsolateAttestationVerificationRecord(record IsolateAttestationVerificationRecord) error {
	if err := validateAttestationVerificationDigestFields(record); err != nil {
		return err
	}
	if err := validateAttestationVerificationEnums(record); err != nil {
		return err
	}
	if !looksLikeDigest(record.VerificationDigest) {
		return fmt.Errorf("verification_digest must be a sha256 digest")
	}
	normalized := record
	normalized.ReasonCodes = uniqueSortedStrings(normalized.ReasonCodes)
	normalized.DerivedMeasurementDigests = uniqueSortedStrings(normalized.DerivedMeasurementDigests)
	expectedDigest, err := canonicalSHA256Digest(isolateAttestationVerificationDigestInput(normalized), "isolate attestation verification")
	if err != nil {
		return err
	}
	if normalized.VerificationDigest != expectedDigest {
		return fmt.Errorf("verification_digest does not match attestation verification record")
	}
	return nil
}

func validateAttestationVerificationDigestFields(record IsolateAttestationVerificationRecord) error {
	if err := validateAttestationVerificationCoreDigests(record); err != nil {
		return err
	}
	if err := validateAttestationVerificationPolicyBinding(record); err != nil {
		return err
	}
	return validateAttestationDerivedMeasurementDigests(record.DerivedMeasurementDigests)
}

func validateAttestationVerificationCoreDigests(record IsolateAttestationVerificationRecord) error {
	if !looksLikeDigest(record.AttestationEvidenceDigest) {
		return fmt.Errorf("attestation_evidence_digest must be a sha256 digest")
	}
	if record.ReplayIdentityDigest != "" && !looksLikeDigest(record.ReplayIdentityDigest) {
		return fmt.Errorf("replay_identity_digest must be a sha256 digest")
	}
	return nil
}

func validateAttestationVerificationPolicyBinding(record IsolateAttestationVerificationRecord) error {
	policyID := strings.TrimSpace(record.VerifierPolicyID)
	policyDigest := strings.TrimSpace(record.VerifierPolicyDigest)
	if err := validateAttestationVerificationPolicyPairing(policyID, policyDigest); err != nil {
		return err
	}
	if err := validateTrustedAttestationPolicyRequirement(record, policyID, policyDigest); err != nil {
		return err
	}
	if policyDigest != "" && !looksLikeDigest(policyDigest) {
		return fmt.Errorf("verifier_policy_digest must be a sha256 digest")
	}
	return nil
}

func validateAttestationVerificationPolicyPairing(policyID, policyDigest string) error {
	if policyID == "" && policyDigest != "" {
		return fmt.Errorf("verifier_policy_id is required when verifier_policy_digest is set")
	}
	if policyID != "" && policyDigest == "" {
		return fmt.Errorf("verifier_policy_digest is required when verifier_policy_id is set")
	}
	return nil
}

func validateTrustedAttestationPolicyRequirement(record IsolateAttestationVerificationRecord, policyID, policyDigest string) error {
	if !requiresTrustedAttestationPolicyBinding(record) {
		return nil
	}
	if policyID == "" {
		return fmt.Errorf("verifier_policy_id is required for trusted attestation verification")
	}
	if policyDigest == "" {
		return fmt.Errorf("verifier_policy_digest is required for trusted attestation verification")
	}
	return nil
}

func validateAttestationDerivedMeasurementDigests(digests []string) error {
	for i, digest := range digests {
		if !looksLikeDigest(digest) {
			return fmt.Errorf("derived_measurement_digests[%d] must be a sha256 digest", i)
		}
	}
	return nil
}

func requiresTrustedAttestationPolicyBinding(record IsolateAttestationVerificationRecord) bool {
	verificationResult := strings.TrimSpace(record.VerificationResult)
	if verificationResult == "" || verificationResult == AttestationVerificationResultUnknown {
		return false
	}
	if verificationResult == AttestationVerificationResultInvalid && containsAnyReasonCode(record.ReasonCodes, attestationReasonCodeEvidenceRequired, attestationReasonCodeVerificationRequired) {
		return false
	}
	return true
}

func validateAttestationVerificationEnums(record IsolateAttestationVerificationRecord) error {
	switch record.VerificationResult {
	case AttestationVerificationResultValid, AttestationVerificationResultInvalid, AttestationVerificationResultUnknown:
	default:
		return fmt.Errorf("verification_result %q is invalid", record.VerificationResult)
	}
	switch record.ReplayVerdict {
	case "", AttestationReplayVerdictUnknown, AttestationReplayVerdictOriginal, AttestationReplayVerdictReplay:
	default:
		return fmt.Errorf("replay_verdict %q is invalid", record.ReplayVerdict)
	}
	return nil
}
