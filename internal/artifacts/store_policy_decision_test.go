package artifacts

import (
	"testing"
	"time"
)

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

func TestRecordPolicyDecisionCreatesPendingApprovalLinkedToPolicyDigest(t *testing.T) {
	store := newTestStore(t)
	rec := basePolicyDecisionRecord("run-link", map[string]any{"precedence": "approval"})
	rec.DecisionOutcome = "require_human_approval"
	rec.PolicyReasonCode = "approval_required"
	rec.RequiredApprovalSchemaID = "runecode.protocol.details.policy.required_approval.moderate.workspace_write.v0"
	rec.RequiredApproval = map[string]any{
		"approval_trigger_code": "excerpt_promotion",
		"scope": map[string]any{
			"workspace_id": "workspace-local",
			"run_id":       "run-link",
			"stage_id":     "artifact_flow",
			"step_id":      "step-1",
			"action_kind":  "promotion",
		},
	}
	if err := store.RecordPolicyDecision(rec); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	storedDecision, ok := firstPolicyDecisionRecord(store)
	if !ok {
		t.Fatal("policy decision missing from state")
	}
	if len(store.state.Approvals) != 1 {
		t.Fatalf("approval count = %d, want 1", len(store.state.Approvals))
	}
	for _, approval := range store.state.Approvals {
		if approval.PolicyDecisionHash != storedDecision.Digest {
			t.Fatalf("approval policy_decision_hash = %q, want %q", approval.PolicyDecisionHash, storedDecision.Digest)
		}
	}
}

func TestRecordApprovalBackfillsPolicyDecisionHashFromPersistedDecision(t *testing.T) {
	store := newTestStore(t)
	rec := basePolicyDecisionRecord("run-backfill", map[string]any{"precedence": "deny"})
	if err := store.RecordPolicyDecision(rec); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	storedDecision, ok := firstPolicyDecisionRecord(store)
	if !ok {
		t.Fatal("policy decision missing from state")
	}

	requestedAt := time.Now().UTC()
	approval := ApprovalRecord{
		ApprovalID:             testDigest("a"),
		Status:                 "pending",
		WorkspaceID:            "workspace-local",
		RunID:                  "run-backfill",
		StageID:                "artifact_flow",
		StepID:                 "step-1",
		ActionKind:             "promotion",
		RequestedAt:            requestedAt,
		ApprovalTriggerCode:    "excerpt_promotion",
		ChangesIfApproved:      approvalChangesIfApprovedDefault,
		ApprovalAssuranceLevel: "session_authenticated",
		PresenceMode:           "os_confirmation",
		ManifestHash:           storedDecision.ManifestHash,
		ActionRequestHash:      storedDecision.ActionRequestHash,
	}
	if err := store.RecordApproval(approval); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	storedApproval, ok := store.ApprovalGet(testDigest("a"))
	if !ok {
		t.Fatal("stored approval missing")
	}
	if storedApproval.PolicyDecisionHash != storedDecision.Digest {
		t.Fatalf("stored approval policy_decision_hash = %q, want %q", storedApproval.PolicyDecisionHash, storedDecision.Digest)
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
