package trustpolicy

import (
	"encoding/json"
	"testing"
	"time"
)

func TestVerifyAuditEvidenceMissingAnchorIsDegradedByDefault(t *testing.T) {
	report := mustVerifyAuditEvidenceReport(t, newAuditVerificationFixture(t, verifierStatusFixture{status: "active"}), nil)
	assertMissingAnchorDegradesReport(t, report)
	if report.AnchoringPosture != AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound {
		t.Fatalf("anchoring_posture = %q, want %q", report.AnchoringPosture, AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound)
	}
	assertDerivedSummaryDegraded(t, report)
}

func TestVerifyAuditEvidenceIgnoresHistoricalUnrelatedReceipts(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	validAnchor := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	unrelatedAnchor := fixture.anchorReceiptEnvelope(t, testDigestFromByte('9'))

	var payload map[string]any
	if err := json.Unmarshal(unrelatedAnchor.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	payload["recorded_at"] = "2026-03-13T12:15:00Z"
	unrelatedAnchor.Payload = marshalJSONFixture(t, payload)
	unrelatedAnchor = resignEnvelopeFixture(t, fixture.privateKey, unrelatedAnchor)

	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                 AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:               fixture.segment,
		RawFramedSegmentBytes: fixture.rawSegmentBytes,
		SegmentSealEnvelope:   fixture.sealEnvelope,
		KnownSealDigests:      []Digest{fixture.sealEnvelopeDigest, testDigestFromByte('9')},
		ReceiptEnvelopes:      []SignedObjectEnvelope{validAnchor, unrelatedAnchor},
		VerifierRecords:       fixture.verifierRecords,
		EventContractCatalog:  fixture.eventContractCatalog,
		SignerEvidence:        fixture.signerEvidence,
		Now:                   time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if report.AnchoringStatus != AuditVerificationStatusOK {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusOK)
	}
	if containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want no %q for unrelated historical receipt", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceFailsOnUnknownMismatchedHistoricalReceipt(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	unrelatedAnchor := fixture.anchorReceiptEnvelope(t, testDigestFromByte('9'))

	var payload map[string]any
	if err := json.Unmarshal(unrelatedAnchor.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	payload["recorded_at"] = "2026-03-13T12:15:00Z"
	unrelatedAnchor.Payload = marshalJSONFixture(t, payload)
	unrelatedAnchor = resignEnvelopeFixture(t, fixture.privateKey, unrelatedAnchor)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{unrelatedAnchor})
	if report.AnchoringStatus != AuditVerificationStatusFailed {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusFailed)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceDistinguishesHistoricalAdmissibilityFromCurrentDegradedPosture(t *testing.T) {
	statusChangedAt := "2026-03-13T12:30:00Z"
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "revoked", statusChangedAt: statusChangedAt})
	anchor := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                 AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:               fixture.segment,
		RawFramedSegmentBytes: fixture.rawSegmentBytes,
		SegmentSealEnvelope:   fixture.sealEnvelope,
		ReceiptEnvelopes:      []SignedObjectEnvelope{anchor},
		VerifierRecords:       fixture.verifierRecords,
		EventContractCatalog:  fixture.eventContractCatalog,
		SignerEvidence:        fixture.signerEvidence,
		Now:                   time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if !report.CryptographicallyValid {
		t.Fatal("cryptographically_valid = false, want true")
	}
	if !report.HistoricallyAdmissible {
		t.Fatal("historically_admissible = false, want true")
	}
	if !report.CurrentlyDegraded {
		t.Fatal("currently_degraded = false, want true")
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonSignerCurrentlyRevokedOrCompromised) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonSignerCurrentlyRevokedOrCompromised)
	}
	if report.AnchoringStatus != AuditVerificationStatusOK {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusOK)
	}
}

func TestVerifyAuditEvidenceFailsClosedOnInvalidReceiptRecorder(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	payload["recorder"] = map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity"}
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.AnchoringStatus != AuditVerificationStatusFailed {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusFailed)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceKeepsCryptographicValidityForAnchoringOnlyFailure(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	invalidAnchor := fixture.anchorReceiptEnvelope(t, testDigestFromByte('9'))
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{invalidAnchor})
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonAnchorReceiptMissing) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonAnchorReceiptMissing)
	}
	if !report.CryptographicallyValid {
		t.Fatal("cryptographically_valid = false, want true for anchoring degraded state")
	}
}

func TestVerifyAuditEvidenceMarksCryptographicValidityFalseForDigestFailure(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	broken := fixture
	broken.segment.Frames[0].RecordDigest = testDigestFromByte('9')
	report := mustVerifyAuditEvidenceReport(t, broken, nil)
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonSegmentFrameDigestMismatch) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonSegmentFrameDigestMismatch)
	}
	if report.CryptographicallyValid {
		t.Fatal("cryptographically_valid = true, want false for digest mismatch")
	}
}
