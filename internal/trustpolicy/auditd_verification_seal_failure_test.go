package trustpolicy

import (
	"encoding/json"
	"testing"
	"time"
)

func TestVerifyAuditEvidenceFailsClosedForAnchorReceiptWhenSealValidationFails(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	invalidSeal := fixture.sealEnvelope
	var sealPayload map[string]any
	if err := json.Unmarshal(invalidSeal.Payload, &sealPayload); err != nil {
		t.Fatalf("Unmarshal seal payload returned error: %v", err)
	}
	sealPayload["event_count"] = 99
	invalidSeal.Payload = marshalJSONFixture(t, sealPayload)
	invalidSeal = resignEnvelopeFixture(t, fixture.privateKey, invalidSeal)

	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                 AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:               fixture.segment,
		RawFramedSegmentBytes: fixture.rawSegmentBytes,
		SegmentSealEnvelope:   invalidSeal,
		ReceiptEnvelopes:      []SignedObjectEnvelope{fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)},
		VerifierRecords:       fixture.verifierRecords,
		EventContractCatalog:  fixture.eventContractCatalog,
		SignerEvidence:        fixture.signerEvidence,
		Now:                   time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonSegmentSealInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonSegmentSealInvalid)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q when seal validation fails", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}
