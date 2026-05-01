package launcherbackend

import (
	"strings"
	"testing"
)

func TestValidateIsolateAttestationVerificationRecordRequiresVerifierPolicyBindingForTrustedVerification(t *testing.T) {
	record := validTrustedAttestationVerificationRecordForTest(t)

	record.VerifierPolicyID = ""
	err := ValidateIsolateAttestationVerificationRecord(record)
	if err == nil {
		t.Fatal("ValidateIsolateAttestationVerificationRecord expected missing verifier policy id error")
	}
	if !strings.Contains(err.Error(), "verifier_policy_id is required when verifier_policy_digest is set") {
		t.Fatalf("ValidateIsolateAttestationVerificationRecord error = %v, want policy id/digest pairing requirement", err)
	}

	record = validTrustedAttestationVerificationRecordForTest(t)
	record.VerifierPolicyDigest = ""
	err = ValidateIsolateAttestationVerificationRecord(record)
	if err == nil {
		t.Fatal("ValidateIsolateAttestationVerificationRecord expected missing verifier policy digest error")
	}
	if !strings.Contains(err.Error(), "verifier_policy_digest is required when verifier_policy_id is set") {
		t.Fatalf("ValidateIsolateAttestationVerificationRecord error = %v, want policy id/digest pairing requirement", err)
	}

	record = validTrustedAttestationVerificationRecordForTest(t)
	record.VerifierPolicyID = ""
	record.VerifierPolicyDigest = ""
	err = ValidateIsolateAttestationVerificationRecord(record)
	if err == nil {
		t.Fatal("ValidateIsolateAttestationVerificationRecord expected trusted policy binding requirement")
	}
	if !strings.Contains(err.Error(), "verifier_policy_id is required for trusted attestation verification") {
		t.Fatalf("ValidateIsolateAttestationVerificationRecord error = %v, want trusted policy id requirement", err)
	}
}

func TestValidateIsolateAttestationVerificationRecordAllowsFailClosedVerificationWithoutPolicyBinding(t *testing.T) {
	record := IsolateAttestationVerificationRecord{
		AttestationEvidenceDigest: testDigest("e"),
		VerificationResult:        AttestationVerificationResultInvalid,
		ReasonCodes:               []string{attestationReasonCodeVerificationRequired},
		ReplayVerdict:             AttestationReplayVerdictUnknown,
	}
	if err := FinalizeIsolateAttestationVerificationRecord(&record); err != nil {
		t.Fatalf("FinalizeIsolateAttestationVerificationRecord returned error: %v", err)
	}
	if err := ValidateIsolateAttestationVerificationRecord(record); err != nil {
		t.Fatalf("ValidateIsolateAttestationVerificationRecord returned error: %v", err)
	}
}

func TestValidateIsolateAttestationVerificationRecordRequiresPolicyBindingForValidVerificationEvenWithFailClosedReason(t *testing.T) {
	record := validTrustedAttestationVerificationRecordForTest(t)
	record.VerifierPolicyID = ""
	record.VerifierPolicyDigest = ""
	record.ReasonCodes = []string{attestationReasonCodeVerificationRequired}
	if err := FinalizeIsolateAttestationVerificationRecord(&record); err != nil {
		t.Fatalf("FinalizeIsolateAttestationVerificationRecord returned error: %v", err)
	}
	err := ValidateIsolateAttestationVerificationRecord(record)
	if err == nil {
		t.Fatal("ValidateIsolateAttestationVerificationRecord expected policy binding requirement")
	}
	if !strings.Contains(err.Error(), "verifier_policy_id is required for trusted attestation verification") {
		t.Fatalf("ValidateIsolateAttestationVerificationRecord error = %v", err)
	}
}

func TestFinalizeIsolateAttestationVerificationRecordDigestBindsVerifierPolicyFields(t *testing.T) {
	record := validTrustedAttestationVerificationRecordForTest(t)
	baseline := record.VerificationDigest

	record.VerifierPolicyID = "runtime_asset_admission_identity_v2"
	if err := FinalizeIsolateAttestationVerificationRecord(&record); err != nil {
		t.Fatalf("FinalizeIsolateAttestationVerificationRecord returned error: %v", err)
	}
	if record.VerificationDigest == baseline {
		t.Fatalf("verification_digest = %q, want changed digest after verifier_policy_id mutation", record.VerificationDigest)
	}

	baseline = record.VerificationDigest
	record.VerifierPolicyDigest = testDigest("e")
	if err := FinalizeIsolateAttestationVerificationRecord(&record); err != nil {
		t.Fatalf("FinalizeIsolateAttestationVerificationRecord returned error: %v", err)
	}
	if record.VerificationDigest == baseline {
		t.Fatalf("verification_digest = %q, want changed digest after verifier_policy_digest mutation", record.VerificationDigest)
	}
}

func validTrustedAttestationVerificationRecordForTest(t *testing.T) IsolateAttestationVerificationRecord {
	t.Helper()
	record := IsolateAttestationVerificationRecord{
		AttestationEvidenceDigest:       testDigest("a"),
		ReplayIdentityDigest:            testDigest("b"),
		VerifierPolicyID:                "runtime_asset_admission_identity",
		VerifierPolicyDigest:            testDigest("c"),
		VerificationRulesProfileVersion: "v1",
		VerificationTimestamp:           "2026-04-29T12:00:00Z",
		VerificationResult:              AttestationVerificationResultValid,
		ReasonCodes:                     []string{"ok"},
		ReplayVerdict:                   AttestationReplayVerdictOriginal,
		DerivedMeasurementDigests:       []string{testDigest("d")},
	}
	if err := FinalizeIsolateAttestationVerificationRecord(&record); err != nil {
		t.Fatalf("FinalizeIsolateAttestationVerificationRecord returned error: %v", err)
	}
	return record
}
