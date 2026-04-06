package trustpolicy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestComputeSignedEnvelopeAuditRecordDigestIsDeterministic(t *testing.T) {
	request := validAuditAdmissionRequestFixture(t)
	digestA, err := ComputeSignedEnvelopeAuditRecordDigest(request.Envelope)
	if err != nil {
		t.Fatalf("ComputeSignedEnvelopeAuditRecordDigest returned error: %v", err)
	}
	digestB, err := ComputeSignedEnvelopeAuditRecordDigest(request.Envelope)
	if err != nil {
		t.Fatalf("ComputeSignedEnvelopeAuditRecordDigest returned error: %v", err)
	}
	if digestA != digestB {
		t.Fatalf("digest mismatch for same envelope: %v vs %v", digestA, digestB)
	}

	mutated := request.Envelope
	mutated.Payload = json.RawMessage(strings.ReplaceAll(string(mutated.Payload), "session-1", "session-2"))
	digestC, err := ComputeSignedEnvelopeAuditRecordDigest(mutated)
	if err != nil {
		t.Fatalf("ComputeSignedEnvelopeAuditRecordDigest returned error for mutated envelope: %v", err)
	}
	if digestA == digestC {
		t.Fatalf("digest should change for mutated envelope, got %v", digestA)
	}
}

func TestBuildDefaultOperationalAuditViewRedactsSensitiveEventFields(t *testing.T) {
	request := validAuditAdmissionRequestFixture(t)
	view, err := BuildDefaultOperationalAuditView(request.Envelope)
	if err != nil {
		t.Fatalf("BuildDefaultOperationalAuditView returned error: %v", err)
	}
	assertOperationalEventViewMetadata(t, view)
	event := decodeOperationalEventView(t, view)
	assertOperationalEventRedactionSemantics(t, event)
}

func assertOperationalEventViewMetadata(t *testing.T, view AuditOperationalView) {
	t.Helper()
	if view.ViewPolicyID != AuditOperationalViewPolicyID {
		t.Fatalf("view_policy_id = %q, want %q", view.ViewPolicyID, AuditOperationalViewPolicyID)
	}
	if view.Event == nil {
		t.Fatal("expected event operational payload in view")
	}
	if len(view.Redaction.ExcludedDataClasses) != 2 || view.Redaction.ExcludedDataClasses[0] != "secret" || view.Redaction.ExcludedDataClasses[1] != "sensitive" {
		t.Fatalf("excluded_data_classes = %v, want [secret sensitive]", view.Redaction.ExcludedDataClasses)
	}
	if len(view.Redaction.RedactedFields) != 2 || view.Redaction.RedactedFields[0] != "event_payload" || view.Redaction.RedactedFields[1] != "principal" {
		t.Fatalf("redacted_fields = %v, want [event_payload principal]", view.Redaction.RedactedFields)
	}
}

func decodeOperationalEventView(t *testing.T, view AuditOperationalView) map[string]any {
	t.Helper()
	decoded := decodeOperationalViewJSON(t, view)
	event, ok := decoded["event"].(map[string]any)
	if !ok {
		t.Fatalf("event view payload has unexpected type %T", decoded["event"])
	}
	return event
}

func assertOperationalEventRedactionSemantics(t *testing.T, event map[string]any) {
	t.Helper()
	if _, found := event["event_payload"]; found {
		t.Fatalf("operational event view must not include event_payload: %+v", event)
	}
	if _, found := event["principal"]; found {
		t.Fatalf("operational event view must not include principal: %+v", event)
	}
	if _, found := event["event_payload_hash"]; !found {
		t.Fatalf("operational event view must retain event_payload_hash evidence: %+v", event)
	}
}

func decodeOperationalViewJSON(t *testing.T, view AuditOperationalView) map[string]any {
	t.Helper()
	viewJSON, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("Marshal view returned error: %v", err)
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(viewJSON, &decoded); err != nil {
		t.Fatalf("Unmarshal view returned error: %v", err)
	}
	return decoded
}

func TestBuildDefaultOperationalAuditViewRedactsSensitiveReceiptFields(t *testing.T) {
	envelope := receiptEnvelopeFixture(t)

	view, err := BuildDefaultOperationalAuditView(envelope)
	if err != nil {
		t.Fatalf("BuildDefaultOperationalAuditView returned error: %v", err)
	}
	if view.Receipt == nil {
		t.Fatal("expected receipt operational payload in view")
	}
	if len(view.Redaction.RedactedFields) != 2 || view.Redaction.RedactedFields[0] != "receipt_payload" || view.Redaction.RedactedFields[1] != "recorder" {
		t.Fatalf("redacted_fields = %v, want [receipt_payload recorder]", view.Redaction.RedactedFields)
	}
	viewJSON, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("Marshal view returned error: %v", err)
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(viewJSON, &decoded); err != nil {
		t.Fatalf("Unmarshal view returned error: %v", err)
	}
	receipt, ok := decoded["receipt"].(map[string]any)
	if !ok {
		t.Fatalf("receipt view payload has unexpected type %T", decoded["receipt"])
	}
	if _, found := receipt["receipt_payload"]; found {
		t.Fatalf("operational receipt view must not include receipt_payload: %+v", receipt)
	}
	if _, found := receipt["recorder"]; found {
		t.Fatalf("operational receipt view must not include recorder: %+v", receipt)
	}
	if _, found := receipt["subject_digest"]; !found {
		t.Fatalf("operational receipt view must retain subject_digest evidence: %+v", receipt)
	}
}

func receiptEnvelopeFixture(t *testing.T) SignedObjectEnvelope {
	t.Helper()
	receiptPayload := receiptPayloadFixture()
	payloadBytes, err := json.Marshal(receiptPayload)
	if err != nil {
		t.Fatalf("Marshal receipt payload returned error: %v", err)
	}
	return SignedObjectEnvelope{
		SchemaID:             EnvelopeSchemaID,
		SchemaVersion:        EnvelopeSchemaVersion,
		PayloadSchemaID:      AuditReceiptSchemaID,
		PayloadSchemaVersion: AuditReceiptSchemaVersion,
		Payload:              payloadBytes,
		SignatureInput:       SignatureInputProfile,
		Signature: SignatureBlock{
			Alg:        "ed25519",
			KeyID:      KeyIDProfile,
			KeyIDValue: strings.Repeat("f", 64),
			Signature:  "c2ln",
		},
	}
}

func receiptPayloadFixture() map[string]any {
	return map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("a", 64)},
		"audit_receipt_kind":        "import",
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:16:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.import_restore_provenance.v0",
		"receipt_payload": map[string]any{
			"provenance_action":       "import",
			"segment_file_hash_scope": "raw_framed_segment_bytes_v1",
			"imported_segments":       []any{map[string]any{"imported_segment_seal_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)}, "imported_segment_root": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)}, "source_segment_file_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "local_segment_file_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "byte_identity_verified": true}},
			"source_manifest_digests": []any{map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)}},
		},
		"authority_context": map[string]any{"authority_kind": "operator", "authority_id": "operator-1"},
	}
}

func TestBuildDefaultOperationalAuditViewFailsClosedOnUnsupportedPayloadFamily(t *testing.T) {
	envelope := SignedObjectEnvelope{
		SchemaID:             EnvelopeSchemaID,
		SchemaVersion:        EnvelopeSchemaVersion,
		PayloadSchemaID:      "runecode.protocol.v0.UnknownPayload",
		PayloadSchemaVersion: "0.1.0",
		Payload:              json.RawMessage(`{"schema_id":"runecode.protocol.v0.UnknownPayload","schema_version":"0.1.0"}`),
		SignatureInput:       SignatureInputProfile,
		Signature: SignatureBlock{
			Alg:        "ed25519",
			KeyID:      KeyIDProfile,
			KeyIDValue: strings.Repeat("a", 64),
			Signature:  "c2ln",
		},
	}
	if _, err := BuildDefaultOperationalAuditView(envelope); err == nil {
		t.Fatal("BuildDefaultOperationalAuditView expected failure for unsupported payload schema")
	}
}
