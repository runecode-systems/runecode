package trustpolicy

import "testing"

func TestVerifyAuditEvidenceRetainsAnchorMissingDegradedWhenOtherHardFailureExists(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	invalidAnchor := fixture.anchorReceiptEnvelope(t, testDigestFromByte('9'))
	invalidReconciliation := fixture.reconciliationReceiptEnvelope(t, testDigestFromByte('8'))

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{invalidAnchor, invalidReconciliation})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent)
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonAnchorReceiptMissing) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonAnchorReceiptMissing)
	}
}
