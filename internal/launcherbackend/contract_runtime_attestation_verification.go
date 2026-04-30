package launcherbackend

import "fmt"

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
	if !looksLikeDigest(record.AttestationEvidenceDigest) {
		return fmt.Errorf("attestation_evidence_digest must be a sha256 digest")
	}
	if record.ReplayIdentityDigest != "" && !looksLikeDigest(record.ReplayIdentityDigest) {
		return fmt.Errorf("replay_identity_digest must be a sha256 digest")
	}
	if record.VerifierPolicyDigest != "" && !looksLikeDigest(record.VerifierPolicyDigest) {
		return fmt.Errorf("verifier_policy_digest must be a sha256 digest")
	}
	for i, digest := range record.DerivedMeasurementDigests {
		if !looksLikeDigest(digest) {
			return fmt.Errorf("derived_measurement_digests[%d] must be a sha256 digest", i)
		}
	}
	return nil
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
