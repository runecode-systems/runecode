package trustpolicy

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func base64DecodeFixture(t *testing.T, encoded string) string {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString returned error: %v", err)
	}
	return string(b)
}

func base64EncodeFixture(value []byte) string {
	return base64.StdEncoding.EncodeToString(value)
}

func mutateReceiptPayloadEnvelope(t *testing.T, fixture auditVerificationFixture, envelope SignedObjectEnvelope, mutate func(payload map[string]any)) SignedObjectEnvelope {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	mutate(payload)
	envelope.Payload = marshalJSONFixture(t, payload)
	return resignEnvelopeFixture(t, fixture.privateKey, envelope)
}

func (f auditVerificationFixture) metaAuditReceiptEnvelope(t *testing.T, subjectDigest Digest, kind string) SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        kind,
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:25:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.meta_audit_action.v0",
		"receipt_payload": map[string]any{
			"action_code":   kind,
			"action_family": "meta_audit",
			"scope_kind":    "verification_plane",
			"result":        "ok",
		},
	})
}

func (f auditVerificationFixture) resealForUpdatedFrame(t *testing.T, digest Digest) SignedObjectEnvelope {
	t.Helper()
	sealing := mustDecodeSealPayloadFixture(t, f.sealEnvelope.Payload)
	sealing.FirstRecordDigest = digest
	sealing.LastRecordDigest = digest
	root, err := ComputeOrderedAuditSegmentMerkleRoot([]Digest{digest})
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot returned error: %v", err)
	}
	sealing.MerkleRoot = root
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditSegmentSealSchemaID, AuditSegmentSealSchemaVersion, map[string]any{
		"schema_id":                     sealing.SchemaID,
		"schema_version":                sealing.SchemaVersion,
		"segment_id":                    sealing.SegmentID,
		"sealed_after_state":            sealing.SealedAfterState,
		"segment_state":                 sealing.SegmentState,
		"segment_cut":                   map[string]any{"ownership_scope": sealing.SegmentCut.OwnershipScope, "max_segment_bytes": sealing.SegmentCut.MaxSegmentBytes, "cut_trigger": sealing.SegmentCut.CutTrigger},
		"event_count":                   sealing.EventCount,
		"first_record_digest":           sealing.FirstRecordDigest,
		"last_record_digest":            sealing.LastRecordDigest,
		"merkle_profile":                sealing.MerkleProfile,
		"merkle_root":                   sealing.MerkleRoot,
		"segment_file_hash_scope":       sealing.SegmentFileHashScope,
		"segment_file_hash":             sealing.SegmentFileHash,
		"seal_chain_index":              sealing.SealChainIndex,
		"anchoring_subject":             sealing.AnchoringSubject,
		"sealed_at":                     sealing.SealedAt,
		"protocol_bundle_manifest_hash": sealing.ProtocolBundleManifestHash,
	})
}

func (f auditVerificationFixture) runtimeProviderInvocationReceiptEnvelope(t *testing.T, subjectDigest Digest, kind string) SignedObjectEnvelope {
	t.Helper()
	networkTarget, networkTargetDigest := runtimeProviderInvocationNetworkTargetFixture(t)
	outcome := runtimeProviderInvocationOutcome(kind)
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, runtimeProviderInvocationReceiptPayloadFixture(subjectDigest, kind, outcome, networkTarget, networkTargetDigest))
}

func runtimeProviderInvocationNetworkTargetFixture(t *testing.T) (map[string]any, Digest) {
	t.Helper()
	networkTarget := map[string]any{
		"descriptor_schema_id": "runecode.protocol.audit.network_target.gateway_destination.v0",
		"destination_kind":     "model_endpoint",
		"host":                 "model.example.com",
		"destination_ref":      "model.example.com/v1/chat/completions",
		"path_prefix":          "/v1/chat/completions",
	}
	networkTargetDigest, err := computeJSONCanonicalDigest(marshalJSONFixture(t, networkTarget))
	if err != nil {
		t.Fatalf("computeJSONCanonicalDigest returned error: %v", err)
	}
	return networkTarget, networkTargetDigest
}

func runtimeProviderInvocationOutcome(kind string) string {
	outcome := "authorized"
	if kind == "provider_invocation_denied" {
		outcome = "denied"
	}
	return outcome
}

func runtimeProviderInvocationReceiptPayloadFixture(subjectDigest Digest, kind string, outcome string, networkTarget map[string]any, networkTargetDigest Digest) map[string]any {
	return map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        kind,
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:25:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.provider_invocation.v0",
		"receipt_payload": map[string]any{
			"authorization_outcome":        outcome,
			"provider_kind":                "llm",
			"provider_profile_id":          "provider-profile-1",
			"model_id":                     "gpt-4.1-mini",
			"endpoint_identity":            "model.example.com/v1/chat/completions",
			"gateway_role_kind":            "model-gateway",
			"destination_kind":             "model_endpoint",
			"operation":                    "invoke_model",
			"request_digest":               map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)},
			"payload_digest":               map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)},
			"request_payload_digest_bound": true,
			"network_target":               networkTarget,
			"network_target_digest":        map[string]any{"hash_alg": networkTargetDigest.HashAlg, "hash": networkTargetDigest.Hash},
			"policy_decision_digest":       map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("2", 64)},
			"allowlist_ref_digest":         map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("3", 64)},
			"allowlist_entry_id":           "model_default",
			"lease_id_digest":              map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("4", 64)},
			"run_id_digest":                map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("5", 64)},
		},
	}
}
