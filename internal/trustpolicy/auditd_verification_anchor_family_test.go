package trustpolicy

import (
	"encoding/json"
	"testing"
)

func TestVerifyAuditEvidenceFailsClosedOnAnchorReceiptWithUnexpectedSubjectFamily(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	payload["subject_family"] = "unexpected_family"
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}
