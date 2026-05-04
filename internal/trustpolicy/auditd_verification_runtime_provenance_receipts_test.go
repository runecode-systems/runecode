package trustpolicy

import (
	"strings"
	"testing"
)

func (f auditVerificationFixture) approvalEvidenceReceiptEnvelope(t *testing.T, subjectDigest Digest, kind string) SignedObjectEnvelope {
	t.Helper()
	receiptPayload := map[string]any{
		"approval_id":             "sha256:" + strings.Repeat("1", 64),
		"approval_status":         "approved",
		"request_digest":          map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("2", 64)},
		"decision_digest":         map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("3", 64)},
		"scope_digest":            map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("4", 64)},
		"artifact_set_digest":     map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("5", 64)},
		"diff_digest":             map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("6", 64)},
		"summary_preview_digest":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("7", 64)},
		"consumption_link_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("8", 64)},
		"policy_decision_digest":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("9", 64)},
		"run_id_digest":           map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("a", 64)},
		"action_kind":             "promotion",
		"recorded_from":           "approval_resolve",
		"approver":                map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": "human", "instance_id": "approval-session"},
	}
	if kind == auditReceiptKindApprovalConsumption {
		receiptPayload["approval_status"] = "consumed"
		receiptPayload["recorded_from"] = "approval_consumption"
	}
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        kind,
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:25:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"receipt_payload_schema_id": auditReceiptPayloadSchemaApprovalEvidenceV0,
		"receipt_payload":           receiptPayload,
	})
}

func (f auditVerificationFixture) publicationEvidenceReceiptEnvelope(t *testing.T, subjectDigest Digest) SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        auditReceiptKindArtifactPublished,
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:25:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"receipt_payload_schema_id": auditReceiptPayloadSchemaPublicationV0,
		"receipt_payload": map[string]any{
			"publication_kind":         "promotion",
			"artifact_digest":          map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
			"source_artifact_digest":   map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)},
			"approval_decision_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)},
			"approval_link_digest":     map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)},
			"run_id_digest":            map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)},
			"action_kind":              "promotion",
		},
	})
}

func (f auditVerificationFixture) overrideEvidenceReceiptEnvelope(t *testing.T, subjectDigest Digest) SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        auditReceiptKindOverrideOrBreakGlass,
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:25:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"receipt_payload_schema_id": auditReceiptPayloadSchemaOverrideV0,
		"receipt_payload": map[string]any{
			"override_kind":          "gate_override",
			"policy_decision_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)},
			"action_request_digest":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("2", 64)},
			"approval_link_digest":   map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("3", 64)},
			"run_id_digest":          map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("4", 64)},
			"approval_required":      true,
			"approval_consumed":      true,
		},
	})
}

func (f auditVerificationFixture) runtimeSecretLeaseReceiptEnvelope(t *testing.T, subjectDigest Digest, kind string) SignedObjectEnvelope {
	t.Helper()
	action := "issued"
	if kind == "secret_lease_revoked" {
		action = "revoked"
	}
	receiptPayload := map[string]any{
		"lease_action":       action,
		"lease_id_digest":    map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("6", 64)},
		"secret_ref_digest":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("7", 64)},
		"consumer_id_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("8", 64)},
		"role_kind":          "model-gateway",
		"scope_digest":       map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("9", 64)},
		"delivery_kind":      "model_gateway",
		"issued_at":          "2026-03-13T12:24:00Z",
		"run_id_digest":      map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("a", 64)},
	}
	if action == "revoked" {
		receiptPayload["revoked_at"] = "2026-03-13T12:25:00Z"
	}
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        kind,
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:25:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.secret_lease.v0",
		"receipt_payload":           receiptPayload,
	})
}

func (f auditVerificationFixture) runtimeSummaryReceiptEnvelope(t *testing.T, subjectDigest Digest, providerCount int64, leaseIssueCount int64) SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        "runtime_summary",
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:25:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.runtime_summary.v0",
		"receipt_payload": map[string]any{
			"summary_scope_kind":           "run",
			"provider_invocation_count":    providerCount,
			"secret_lease_issue_count":     leaseIssueCount,
			"secret_lease_revoke_count":    int64(0),
			"network_egress_count":         providerCount,
			"no_provider_invocation":       providerCount == 0,
			"no_secret_lease_issued":       leaseIssueCount == 0,
			"approval_consumption_count":   int64(0),
			"no_approval_consumed":         true,
			"boundary_crossing_count":      int64(0),
			"no_artifact_crossed_boundary": true,
			"boundary_route":               "artifact_io_promotion",
			"boundary_crossing_support":    "explicit",
		},
	})
}

func (f auditVerificationFixture) degradedPostureSummaryReceiptEnvelope(t *testing.T, subjectDigest Digest) SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        "degraded_posture_summary",
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:25:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.degraded_posture_summary.v0",
		"receipt_payload": map[string]any{
			"summary_scope_kind":           "run",
			"degraded":                     true,
			"degradation_cause_code":       "gate_override_applied",
			"degradation_reason_codes":     []string{"gate_override_applied"},
			"trust_claim_before":           "no_override_required",
			"trust_claim_after":            "override_required_or_applied",
			"changed_trust_claim":          true,
			"user_acknowledged":            true,
			"acknowledgment_evidence":      "approval_consumed",
			"approval_required":            true,
			"approval_consumed":            true,
			"override_required":            true,
			"override_applied":             true,
			"approval_policy_decision_ref": "sha256:" + strings.Repeat("1", 64),
			"approval_consumption_link":    "sha256:" + strings.Repeat("2", 64),
			"override_policy_decision_ref": "sha256:" + strings.Repeat("3", 64),
			"override_action_request_hash": "sha256:" + strings.Repeat("4", 64),
			"run_lifecycle_state":          "failed",
		},
	})
}

func (f auditVerificationFixture) negativeCapabilitySummaryReceiptEnvelope(t *testing.T, subjectDigest Digest) SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 AuditReceiptSchemaID,
		"schema_version":            AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        "negative_capability_summary",
		"subject_family":            "audit_segment_seal",
		"recorded_at":               "2026-03-13T12:25:00Z",
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "brokerapi", "instance_id": "brokerapi-1"},
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.negative_capability_summary.v0",
		"receipt_payload": map[string]any{
			"summary_scope_kind":                    "run",
			"no_secret_lease_issued":                true,
			"no_network_egress":                     true,
			"no_approval_consumed":                  true,
			"no_artifact_crossed_boundary":          true,
			"boundary_route":                        "artifact_io_promotion",
			"secret_lease_evidence_support":         "explicit",
			"network_egress_evidence_support":       "explicit",
			"approval_consumption_evidence_support": "limited",
			"boundary_crossing_evidence_support":    "explicit",
		},
	})
}
