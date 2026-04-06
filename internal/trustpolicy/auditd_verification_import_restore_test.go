package trustpolicy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVerifyAuditEvidenceImportRestoreReceiptMatchesRelevantImportedSegmentEntry(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	matchingImportReceipt := fixture.importRestoreReceiptEnvelope(t, fixture.sealEnvelopeDigest, fixture.sealEnvelope)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{matchingImportReceipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent)
	}
}

func TestVerifyAuditEvidenceImportRestoreReceiptFailsWhenNoImportedEntryMatchesSegment(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	brokenImportReceipt := fixture.importRestoreReceiptEnvelope(t, fixture.sealEnvelopeDigest, fixture.sealEnvelope)

	var payload map[string]any
	if err := json.Unmarshal(brokenImportReceipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal import receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	segments := receiptPayload["imported_segments"].([]any)
	for i := range segments {
		segment := segments[i].(map[string]any)
		nonMatching := map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("9", 64)}
		segment["source_segment_file_hash"] = nonMatching
		segment["local_segment_file_hash"] = nonMatching
	}
	brokenImportReceipt.Payload = marshalJSONFixture(t, payload)
	brokenImportReceipt = resignEnvelopeFixture(t, fixture.privateKey, brokenImportReceipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{brokenImportReceipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent)
	}
}

func TestVerifyAuditEvidenceImportRestoreReceiptFailsOnDuplicateMatchingSegmentEntries(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	brokeDuplicate := fixture.importRestoreReceiptEnvelope(t, fixture.sealEnvelopeDigest, fixture.sealEnvelope)

	var payload map[string]any
	if err := json.Unmarshal(brokeDuplicate.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal import receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	segments := receiptPayload["imported_segments"].([]any)
	matchingEntry := segments[1].(map[string]any)
	duplicateMatching := map[string]any{}
	for key, value := range matchingEntry {
		duplicateMatching[key] = value
	}
	receiptPayload["imported_segments"] = append(segments, duplicateMatching)
	brokeDuplicate.Payload = marshalJSONFixture(t, payload)
	brokeDuplicate = resignEnvelopeFixture(t, fixture.privateKey, brokeDuplicate)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{brokeDuplicate})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent)
	}
}

func TestVerifyAuditEvidenceImportRestoreReceiptAllowsAdditionalUnrelatedSegmentEntries(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.importRestoreReceiptEnvelope(t, fixture.sealEnvelopeDigest, fixture.sealEnvelope)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal import receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	segments := receiptPayload["imported_segments"].([]any)
	unrelated := map[string]any{
		"imported_segment_seal_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("a", 64)},
		"imported_segment_root":        map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
		"source_segment_file_hash":     map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)},
		"local_segment_file_hash":      map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)},
		"byte_identity_verified":       true,
	}
	receiptPayload["imported_segments"] = append(segments, unrelated)
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonImportRestoreProvenanceInconsistent)
	}
}
