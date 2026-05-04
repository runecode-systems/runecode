package launcherbackend

import "testing"

func TestDeriveAttestationPostureAttestedReceiptWithoutEvidenceDigestIsUnavailable(t *testing.T) {
	receipt := BackendLaunchReceipt{
		ProvisioningPosture:           ProvisioningPostureAttested,
		AttestationEvidenceSourceKind: AttestationSourceKindTPMQuote,
		AttestationMeasurementProfile: MeasurementProfileMicroVMBootV1,
		AttestationVerificationResult: AttestationVerificationResultValid,
		AttestationReplayVerdict:      AttestationReplayVerdictOriginal,
	}

	posture, reasons := DeriveAttestationPosture(receipt)
	if posture != AttestationPostureUnavailable {
		t.Fatalf("posture = %q, want %q", posture, AttestationPostureUnavailable)
	}
	if !containsAnyReasonCode(reasons, "attestation_evidence_unavailable") {
		t.Fatalf("reasons = %#v, want attestation_evidence_unavailable", reasons)
	}
}

func TestDeriveAttestationPostureAttestedReceiptWithoutVerificationDigestIsUnavailable(t *testing.T) {
	receipt := BackendLaunchReceipt{
		ProvisioningPosture:           ProvisioningPostureAttested,
		AttestationEvidenceSourceKind: AttestationSourceKindTPMQuote,
		AttestationMeasurementProfile: MeasurementProfileMicroVMBootV1,
		AttestationEvidenceDigest:     testDigest("1"),
		AttestationVerificationResult: AttestationVerificationResultValid,
		AttestationReplayVerdict:      AttestationReplayVerdictOriginal,
	}

	posture, reasons := DeriveAttestationPosture(receipt)
	if posture != AttestationPostureUnavailable {
		t.Fatalf("posture = %q, want %q", posture, AttestationPostureUnavailable)
	}
	if !containsAnyReasonCode(reasons, "attestation_verification_unavailable") {
		t.Fatalf("reasons = %#v, want attestation_verification_unavailable", reasons)
	}
}

func TestDeriveAttestationPostureAttestedReceiptWithEvidenceAndVerificationDigestIsValid(t *testing.T) {
	receipt := BackendLaunchReceipt{
		ProvisioningPosture:           ProvisioningPostureAttested,
		AttestationEvidenceSourceKind: AttestationSourceKindTPMQuote,
		AttestationMeasurementProfile: MeasurementProfileMicroVMBootV1,
		AttestationEvidenceDigest:     testDigest("1"),
		AttestationVerificationResult: AttestationVerificationResultValid,
		AttestationReplayVerdict:      AttestationReplayVerdictOriginal,
		AttestationVerificationDigest: testDigest("2"),
	}

	posture, reasons := DeriveAttestationPosture(receipt)
	if posture != AttestationPostureValid {
		t.Fatalf("posture = %q, want %q", posture, AttestationPostureValid)
	}
	if len(reasons) != 0 {
		t.Fatalf("reasons = %#v, want empty", reasons)
	}
}

func TestDeriveAttestationPostureFromEvidenceRequiresVerificationDigestForValid(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	evidence.AttestationVerification.VerificationDigest = ""

	posture, reasons := DeriveAttestationPostureFromEvidence(evidence)
	if posture != AttestationPostureUnavailable {
		t.Fatalf("posture = %q, want %q", posture, AttestationPostureUnavailable)
	}
	if !containsAnyReasonCode(reasons, "attestation_verification_unavailable") {
		t.Fatalf("reasons = %#v, want attestation_verification_unavailable", reasons)
	}
}
