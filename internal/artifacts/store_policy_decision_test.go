package artifacts

import "testing"

func TestRecordPolicyDecisionPersistsTypedRecordAndAuditBinding(t *testing.T) {
	store := newTestStore(t)
	rec := PolicyDecisionRecord{
		Digest:                 "",
		RunID:                  "run-p1",
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        "deny",
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           testDigest("1"),
		ActionRequestHash:      testDigest("2"),
		PolicyInputHashes:      []string{testDigest("3")},
		RelevantArtifactHashes: []string{testDigest("4")},
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details:                map[string]any{"precedence": "invariants_first"},
	}
	if err := store.RecordPolicyDecision(rec); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	if len(store.state.PolicyDecisions) != 1 {
		t.Fatalf("policy decision count = %d, want 1", len(store.state.PolicyDecisions))
	}
	stored, ok := firstPolicyDecisionRecord(store)
	if !ok {
		t.Fatal("policy decision missing from state")
	}
	assertPolicyDecisionAuditBinding(t, store, stored)
}

func firstPolicyDecisionRecord(store *Store) (PolicyDecisionRecord, bool) {
	for _, value := range store.state.PolicyDecisions {
		return value, true
	}
	return PolicyDecisionRecord{}, false
}

func assertPolicyDecisionAuditBinding(t *testing.T, store *Store, stored PolicyDecisionRecord) {
	t.Helper()
	if stored.AuditEventType != "policy_decision_recorded" {
		t.Fatalf("audit_event_type = %q, want policy_decision_recorded", stored.AuditEventType)
	}
	if stored.AuditEventSeq <= 0 {
		t.Fatalf("audit_event_seq = %d, want > 0", stored.AuditEventSeq)
	}
	refs := store.PolicyDecisionRefsForRun("run-p1")
	if len(refs) != 1 || refs[0] != stored.Digest {
		t.Fatalf("PolicyDecisionRefsForRun = %v, want [%s]", refs, stored.Digest)
	}
	events, err := store.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	if !containsAuditType(events, "policy_decision_recorded") {
		t.Fatal("expected policy_decision_recorded audit event")
	}
}

func TestRecordPolicyDecisionRejectsCrossRunDigestCollision(t *testing.T) {
	store := newTestStore(t)
	rec := basePolicyDecisionRecord("run-a", map[string]any{"precedence": "invariants_first"})
	if err := store.RecordPolicyDecision(rec); err != nil {
		t.Fatalf("RecordPolicyDecision first insert returned error: %v", err)
	}
	if len(store.state.PolicyDecisions) != 1 {
		t.Fatalf("policy decision count = %d, want 1", len(store.state.PolicyDecisions))
	}
	for digest := range store.state.PolicyDecisions {
		rec.Digest = digest
		break
	}
	rec.RunID = "run-b"
	rec.Details = map[string]any{"precedence": "different"}
	if err := store.RecordPolicyDecision(rec); err == nil {
		t.Fatal("RecordPolicyDecision returned nil error for cross-run digest collision")
	}
}

func TestRecordPolicyDecisionAllowsIdempotentReinsertSamePayload(t *testing.T) {
	store := newTestStore(t)
	rec := basePolicyDecisionRecord("run-a", map[string]any{"precedence": "invariants_first"})
	if err := store.RecordPolicyDecision(rec); err != nil {
		t.Fatalf("RecordPolicyDecision first insert returned error: %v", err)
	}
	if err := store.RecordPolicyDecision(rec); err != nil {
		t.Fatalf("RecordPolicyDecision idempotent insert returned error: %v", err)
	}
	if len(store.state.PolicyDecisions) != 1 {
		t.Fatalf("policy decision count = %d, want 1", len(store.state.PolicyDecisions))
	}
}

func basePolicyDecisionRecord(runID string, details map[string]any) PolicyDecisionRecord {
	return PolicyDecisionRecord{
		Digest:                   "",
		RunID:                    runID,
		SchemaID:                 "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:            "0.3.0",
		DecisionOutcome:          "deny",
		PolicyReasonCode:         "deny_by_default",
		ManifestHash:             testDigest("1"),
		ActionRequestHash:        testDigest("2"),
		PolicyInputHashes:        []string{testDigest("3")},
		RelevantArtifactHashes:   []string{testDigest("4")},
		DetailsSchemaID:          "runecode.protocol.details.policy.evaluation.v0",
		Details:                  details,
		RequiredApprovalSchemaID: "",
		RequiredApproval:         nil,
	}
}
