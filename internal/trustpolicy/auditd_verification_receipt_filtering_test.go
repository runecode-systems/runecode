package trustpolicy

import (
	"testing"
)

func TestVerifyAuditEvidenceInvalidAnchorFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	invalidAnchor := fixture.anchorReceiptEnvelope(t, testDigestFromByte('9'))
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{invalidAnchor})
	if report.AnchoringStatus != AuditVerificationStatusFailed {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusFailed)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}
