package trustpolicy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVerifyAuditEvidenceReconciliationReceiptFailsWhenSubjectDigestMismatchesSeal(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.reconciliationReceiptEnvelope(t, testDigestFromByte('9'))
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func (f auditVerificationFixture) reconciliationReceiptEnvelope(t *testing.T, subjectDigest Digest) SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":          AuditReceiptSchemaID,
		"schema_version":     AuditReceiptSchemaVersion,
		"subject_digest":     subjectDigest,
		"audit_receipt_kind": "reconciliation",
		"subject_family":     "audit_segment_seal",
		"recorder": map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "daemon",
			"principal_id":   "auditd",
			"instance_id":    "auditd-1",
		},
		"recorded_at":               "2026-03-13T12:25:00Z",
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.import_restore_provenance.v0",
		"receipt_payload": map[string]any{
			"provenance_action":       "reconciliation",
			"segment_file_hash_scope": AuditSegmentFileHashScopeRawFramedV1,
			"imported_segments": []any{
				map[string]any{
					"imported_segment_seal_digest": map[string]any{"hash_alg": subjectDigest.HashAlg, "hash": subjectDigest.Hash},
					"imported_segment_root":        map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("7", 64)},
					"source_segment_file_hash":     map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("6", 64)},
					"local_segment_file_hash":      map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("6", 64)},
					"byte_identity_verified":       true,
				},
			},
			"source_manifest_digests": []any{map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("4", 64)}},
		},
	})
}

func TestVerifyAuditEvidenceReconciliationReceiptPassesWhenSubjectDigestMatchesSeal(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.reconciliationReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal reconciliation receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	segments := receiptPayload["imported_segments"].([]any)
	matching := segments[0].(map[string]any)
	matching["imported_segment_root"] = map[string]any{"hash_alg": fixture.sealEnvelopeDigest.HashAlg, "hash": fixture.sealEnvelopeDigest.Hash}
	matching["source_segment_file_hash"] = map[string]any{"hash_alg": fixture.sealEnvelopeDigest.HashAlg, "hash": fixture.sealEnvelopeDigest.Hash}
	matching["local_segment_file_hash"] = map[string]any{"hash_alg": fixture.sealEnvelopeDigest.HashAlg, "hash": fixture.sealEnvelopeDigest.Hash}
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}
