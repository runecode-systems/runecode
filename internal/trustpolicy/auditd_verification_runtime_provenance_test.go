package trustpolicy

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestVerifyAuditEvidenceProviderInvocationReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeProviderInvocationReceiptEnvelope(t, fixture.sealEnvelopeDigest, "provider_invocation_authorized")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceProviderInvocationDeniedReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeProviderInvocationReceiptEnvelope(t, fixture.sealEnvelopeDigest, "provider_invocation_denied")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceSecretLeaseReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeSecretLeaseReceiptEnvelope(t, fixture.sealEnvelopeDigest, "secret_lease_issued")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceRuntimeSummaryReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeSummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest, 0, 0)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceDegradedPostureSummaryReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.degradedPostureSummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceDegradedPostureSummaryMissingApprovalConsumptionLinkFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.degradedPostureSummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	receipt = mutateReceiptPayloadEnvelope(t, fixture, receipt, func(payload map[string]any) {
		receiptPayload, _ := payload["receipt_payload"].(map[string]any)
		delete(receiptPayload, "approval_consumption_link")
		payload["receipt_payload"] = receiptPayload
	})
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceNegativeCapabilitySummaryReceiptPasses(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.negativeCapabilitySummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, unexpected %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceNegativeCapabilitySummaryInvalidSupportFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.negativeCapabilitySummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	receipt = mutateReceiptPayloadEnvelope(t, fixture, receipt, func(payload map[string]any) {
		receiptPayload, _ := payload["receipt_payload"].(map[string]any)
		receiptPayload["approval_consumption_evidence_support"] = "unknown"
		payload["receipt_payload"] = receiptPayload
	})
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceFlagsMissingNegativeCapabilitySummaryWhenRuntimeSummaryPresent(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeSummaryReceiptEnvelope(t, fixture.sealEnvelopeDigest, 0, 0)
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonNegativeCapabilitySummaryMissing) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonNegativeCapabilitySummaryMissing)
	}
}

func TestVerifyAuditEvidenceFlagsMissingApprovalEvidenceForBoundaryAuthorization(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeProviderInvocationReceiptEnvelope(t, fixture.sealEnvelopeDigest, "provider_invocation_authorized")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonMissingRequiredApprovalEvidence) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonMissingRequiredApprovalEvidence)
	}
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if report.CryptographicallyValid {
		t.Fatal("cryptographically_valid=true, want false when required approval evidence is missing as a hard failure")
	}
}

func TestVerifyAuditEvidenceFlagsEvidenceExportIncompleteWhenMetaAuditPresentWithoutExport(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.metaAuditReceiptEnvelope(t, fixture.sealEnvelopeDigest, "trust_root_updated")
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonEvidenceExportIncomplete) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonEvidenceExportIncomplete)
	}
}

func TestVerifyAuditEvidenceMetaAuditActionKindMismatchFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.metaAuditReceiptEnvelope(t, fixture.sealEnvelopeDigest, auditReceiptKindTrustRootUpdated)
	receipt = mutateReceiptPayloadEnvelope(t, fixture, receipt, func(payload map[string]any) {
		receiptPayload, _ := payload["receipt_payload"].(map[string]any)
		receiptPayload["action_code"] = auditReceiptKindVerifierConfigurationChanged
		payload["receipt_payload"] = receiptPayload
	})
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceFlagsMissingRuntimeAttestationForAttestedPayload(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	frameEnvelope := SignedObjectEnvelope{}
	if err := json.Unmarshal([]byte(base64DecodeFixture(t, fixture.segment.Frames[0].CanonicalSignedEnvelopeBytes)), &frameEnvelope); err != nil {
		t.Fatalf("Unmarshal frame envelope returned error: %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(frameEnvelope.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal event payload returned error: %v", err)
	}
	eventPayload, _ := payload["event_payload"].(map[string]any)
	eventPayload["provisioning_posture"] = "attested"
	delete(eventPayload, "attestation_evidence_digest")
	updatedEventPayloadHash := hashCanonicalJSONFixture(t, eventPayload)
	payload["event_payload"] = eventPayload
	payload["event_payload_hash"] = map[string]any{"hash_alg": "sha256", "hash": updatedEventPayloadHash}
	frameEnvelope.Payload = marshalJSONFixture(t, payload)
	frameEnvelope = resignEnvelopeFixture(t, fixture.privateKey, frameEnvelope)
	canonicalEnvelope := canonicalEnvelopeBytesFixture(t, frameEnvelope)
	updatedDigest := digestForBytesFixture(canonicalEnvelope)
	fixture.segment.Frames[0].CanonicalSignedEnvelopeBytes = base64EncodeFixture(canonicalEnvelope)
	fixture.segment.Frames[0].ByteLength = int64(len(canonicalEnvelope))
	fixture.segment.Frames[0].RecordDigest = updatedDigest
	fixture.sealEnvelope = fixture.resealForUpdatedFrame(t, updatedDigest)
	sealDigest, err := ComputeSignedEnvelopeAuditRecordDigest(fixture.sealEnvelope)
	if err != nil {
		t.Fatalf("ComputeSignedEnvelopeAuditRecordDigest returned error: %v", err)
	}
	fixture.sealEnvelopeDigest = sealDigest
	report := mustVerifyAuditEvidenceReport(t, fixture, nil)
	if !report.CurrentlyDegraded {
		t.Fatal("currently_degraded=false, want true when attestation evidence is missing")
	}
	if report.IntegrityStatus != AuditVerificationStatusDegraded {
		t.Fatalf("integrity_status=%q, want degraded", report.IntegrityStatus)
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonMissingRuntimeAttestationEvidence) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonMissingRuntimeAttestationEvidence)
	}
}

func TestVerifyAuditEvidenceProviderInvocationNetworkDigestMismatchFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.runtimeProviderInvocationReceiptEnvelope(t, fixture.sealEnvelopeDigest, "provider_invocation_authorized")
	receipt = mutateReceiptPayloadEnvelope(t, fixture, receipt, func(payload map[string]any) {
		receiptPayload, _ := payload["receipt_payload"].(map[string]any)
		receiptPayload["network_target_digest"] = map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}
		payload["receipt_payload"] = receiptPayload
	})
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.IntegrityStatus != AuditVerificationStatusFailed {
		t.Fatalf("integrity_status=%q, want failed", report.IntegrityStatus)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonReceiptInvalid)
	}
}

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
