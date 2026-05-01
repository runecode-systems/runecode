package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunnerDurableStateReplaySnapshotAndIdempotency(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	checkpoint := RunnerCheckpointAdvisory{LifecycleState: "active", CheckpointCode: "step_attempt_started", OccurredAt: now, IdempotencyKey: "idem-checkpoint-1", StageID: "stage-1", StepID: "step-1", RoleInstanceID: "role-1", StepAttemptID: "step-attempt-1"}
	recordCheckpointAcceptance(t, store, "run-durable", checkpoint, true, "first")
	recordCheckpointAcceptance(t, store, "run-durable", checkpoint, false, "duplicate")
	result := RunnerResultAdvisory{LifecycleState: "completed", ResultCode: "run_completed", OccurredAt: now.Add(time.Minute), IdempotencyKey: "idem-result-1", StageID: "stage-1", StepID: "step-1", RoleInstanceID: "role-1", StepAttemptID: "step-attempt-1"}
	recordResultAcceptance(t, store, "run-durable", result, true, "result")

	journal := mustReadRunnerJournal(t, store.rootDir)
	if got := len(journal); got != 2 {
		t.Fatalf("runner journal records = %d, want 2", got)
	}

	assertReloadedRunnerDurableState(t, store.rootDir, "run-durable", "step-attempt-1")
}

func TestRunnerDurableStateIdempotencyScopedByRunID(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	checkpointA := RunnerCheckpointAdvisory{LifecycleState: "active", CheckpointCode: "run_started", OccurredAt: now, IdempotencyKey: "shared-idem"}
	checkpointB := RunnerCheckpointAdvisory{LifecycleState: "active", CheckpointCode: "run_started", OccurredAt: now.Add(time.Minute), IdempotencyKey: "shared-idem"}
	recordCheckpointAcceptance(t, store, "run-a", checkpointA, true, "run-a first")
	recordCheckpointAcceptance(t, store, "run-b", checkpointB, true, "run-b same key")

	if _, ok := store.RunnerAdvisory("run-a"); !ok {
		t.Fatal("RunnerAdvisory(run-a) missing")
	}
	if _, ok := store.RunnerAdvisory("run-b"); !ok {
		t.Fatal("RunnerAdvisory(run-b) missing")
	}
}

func TestRunnerDurableStateMigratesLegacyRunnerAdvisoryMap(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	seedLegacyRunnerAdvisoryState(t, store, now)
	removeRunnerDurableFiles(t, store.rootDir)

	reloaded, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore reload returned error: %v", err)
	}
	assertLegacyRunnerAdvisoryPresent(t, reloaded, "after migration")
	assertRunnerDurableFilesPresent(t, store.rootDir, "after migration")
	accepted, err := reloaded.RecordRunnerCheckpoint("run-fresh", RunnerCheckpointAdvisory{
		LifecycleState: "active",
		CheckpointCode: "run_started",
		OccurredAt:     now,
		IdempotencyKey: "post-migration-idem",
	})
	if err != nil {
		t.Fatalf("RecordRunnerCheckpoint after migration returned error: %v", err)
	}
	if !accepted {
		t.Fatal("RecordRunnerCheckpoint after migration accepted=false, want true")
	}
	reloadedAgain, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore second reload returned error: %v", err)
	}
	assertLegacyRunnerAdvisoryPresent(t, reloadedAgain, "after post-migration write")
}

func seedLegacyRunnerAdvisoryState(t *testing.T, store *Store, now time.Time) {
	t.Helper()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.state.RunnerAdvisoryByRun["run-legacy"] = RunnerAdvisoryState{
		LastCheckpoint: &RunnerCheckpointAdvisory{
			LifecycleState: "blocked",
			CheckpointCode: "approval_wait_entered",
			OccurredAt:     now.Add(-time.Hour),
			IdempotencyKey: "legacy-idem",
			StageID:        "stage-legacy",
		},
	}
	if err := store.saveStateLocked(); err != nil {
		t.Fatalf("saveStateLocked returned error: %v", err)
	}
}

func removeRunnerDurableFiles(t *testing.T, rootDir string) {
	t.Helper()
	if err := os.Remove(filepath.Join(rootDir, runnerSnapshotFileName)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove snapshot returned error: %v", err)
	}
	if err := os.Remove(filepath.Join(rootDir, runnerJournalFileName)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove journal returned error: %v", err)
	}
}

func assertLegacyRunnerAdvisoryPresent(t *testing.T, store *Store, label string) {
	t.Helper()
	state, ok := store.RunnerAdvisory("run-legacy")
	if !ok || state.LastCheckpoint == nil {
		t.Fatalf("legacy runner advisory missing %s: ok=%v state=%+v", label, ok, state)
	}
	if state.LastCheckpoint.CheckpointCode != "approval_wait_entered" {
		t.Fatalf("checkpoint_code %s = %q, want approval_wait_entered", label, state.LastCheckpoint.CheckpointCode)
	}
}

func assertRunnerDurableFilesPresent(t *testing.T, rootDir string, label string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(rootDir, runnerSnapshotFileName)); err != nil {
		t.Fatalf("runner snapshot missing %s: %v", label, err)
	}
	if _, err := os.Stat(filepath.Join(rootDir, runnerJournalFileName)); err != nil {
		t.Fatalf("runner journal missing %s: %v", label, err)
	}
}

func TestRunnerDurableStateRejectsUnknownSchemaVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, runnerSnapshotFileName), []byte(`{"family":"runner_durable_snapshot","schema_version":99,"last_sequence":0,"runs":{},"idempotency":{}}`), 0o600); err != nil {
		t.Fatalf("write snapshot returned error: %v", err)
	}
	if _, err := NewStore(root); err == nil {
		t.Fatal("NewStore expected unknown snapshot schema version error")
	}
}

func TestRunnerDurableStateRejectsUnknownJournalSchemaVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "state.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write state returned error: %v", err)
	}
	line := `{"family":"runner_durable_journal","schema_version":99,"sequence":1,"record_type":"checkpoint","run_id":"run-x","idempotency_key":"idem","occurred_at":"2026-04-10T10:00:00Z","checkpoint":{"lifecycle_state":"active","checkpoint_code":"run_started","occurred_at":"2026-04-10T10:00:00Z","idempotency_key":"idem"}}`
	if err := os.WriteFile(filepath.Join(root, runnerJournalFileName), []byte(line+"\n"), 0o600); err != nil {
		t.Fatalf("write runner journal returned error: %v", err)
	}
	if _, err := NewStore(root); err == nil {
		t.Fatal("NewStore expected unknown journal schema version error")
	}
}

func TestRunnerApprovalWaitDedupeSupersessionAndStatuses(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 4, 10, 11, 0, 0, 0, time.UTC)
	recordRunnerApprovalWait(t, store, buildRunnerApprovalForTest("run-approval", "1", "pending", now, nil))
	recordRunnerApprovalWait(t, store, buildRunnerApprovalForTest("run-approval", "2", "pending", now.Add(time.Minute), nil))
	recordRunnerApprovalWait(t, store, buildRunnerApprovalForTest("run-approval", "2", "pending", now.Add(time.Minute), nil))
	resolved := now.Add(2 * time.Minute)
	recordRunnerApprovalWait(t, store, buildRunnerApprovalForTest("run-approval", "2", "consumed", resolved, &resolved))

	state, ok := store.RunnerAdvisory("run-approval")
	if !ok {
		t.Fatal("RunnerAdvisory(run-approval) = missing, want present")
	}
	first := state.ApprovalWaits["sha256:"+strings.Repeat("1", 64)]
	if first.Status != "superseded" {
		t.Fatalf("first approval status = %q, want superseded", first.Status)
	}
	second := state.ApprovalWaits["sha256:"+strings.Repeat("2", 64)]
	if second.Status != "consumed" {
		t.Fatalf("second approval status = %q, want consumed", second.Status)
	}

	if err := store.RecordRunnerApprovalWait(RunnerApproval{ApprovalID: "sha256:" + strings.Repeat("3", 64), RunID: "run-approval", Status: "bogus", ApprovalType: "exact_action", BoundActionHash: "sha256:" + strings.Repeat("b", 64), OccurredAt: now}); err == nil {
		t.Fatal("RecordRunnerApprovalWait expected invalid status error")
	}
}

func TestRecordRunnerResultRollsBackConsumedApprovalOnJournalFailure(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 4, 10, 11, 30, 0, 0, time.UTC)
	approval := seedApprovedGateOverrideApproval(t, store, "run-rollback", testDigest("a"), now)
	snapshotPath := filepath.Join(store.rootDir, runnerSnapshotFileName)
	if err := os.Remove(snapshotPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove runner snapshot returned error: %v", err)
	}
	if err := os.Mkdir(snapshotPath, 0o700); err != nil {
		t.Fatalf("mkdir runner snapshot path returned error: %v", err)
	}
	_, err := store.RecordRunnerResult("run-rollback", RunnerResultAdvisory{LifecycleState: "failed", ResultCode: "gate_overridden", OccurredAt: now, IdempotencyKey: "idem-rollback"}, approval.PolicyDecisionHash)
	if err == nil {
		t.Fatal("RecordRunnerResult error = nil, want journal failure")
	}
	stored, ok := store.ApprovalGet(testDigest("a"))
	if !ok {
		t.Fatal("ApprovalGet missing approval after rollback")
	}
	if stored.Status != "approved" {
		t.Fatalf("approval status = %q, want approved after rollback", stored.Status)
	}
	if stored.ConsumedAt != nil {
		t.Fatal("approval consumed_at set after rollback")
	}
}

func TestRunnerDurableStateReconcilesConsumedGateOverrideApprovalFromResult(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	approval := seedApprovedGateOverrideApproval(t, store, "run-reconcile-consume", testDigest("b"), now)
	if _, err := store.RecordRunnerResult("run-reconcile-consume", RunnerResultAdvisory{LifecycleState: "failed", ResultCode: "gate_overridden", GateAttemptID: "gate-attempt-1", GateState: "overridden", OverridePolicyRef: approval.PolicyDecisionHash, OccurredAt: now, IdempotencyKey: "idem-reconcile"}, approval.PolicyDecisionHash); err != nil {
		t.Fatalf("RecordRunnerResult returned error: %v", err)
	}

	store.mu.Lock()
	approved := store.state.Approvals[approval.ApprovalID]
	approved.Status = "approved"
	approved.DecidedAt = nil
	approved.ConsumedAt = nil
	store.state.Approvals[approval.ApprovalID] = approved
	if err := store.saveStateLocked(); err != nil {
		store.mu.Unlock()
		t.Fatalf("saveStateLocked returned error: %v", err)
	}
	store.mu.Unlock()

	reloaded, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore reload returned error: %v", err)
	}
	restored, ok := reloaded.ApprovalGet(approval.ApprovalID)
	if !ok {
		t.Fatal("ApprovalGet missing after reload")
	}
	if restored.Status != "consumed" {
		t.Fatalf("approval status after reload = %q, want consumed", restored.Status)
	}
	if restored.ConsumedAt == nil {
		t.Fatal("approval consumed_at missing after reconciliation")
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func seedApprovedGateOverrideApproval(t *testing.T, store *Store, runID, approvalID string, now time.Time) ApprovalRecord {
	t.Helper()
	decision := basePolicyDecisionRecord(runID, map[string]any{"precedence": "approval"})
	decision.DecisionOutcome = "require_human_approval"
	decision.PolicyReasonCode = "approval_required"
	decision.RequiredApprovalSchemaID = "runecode.protocol.details.policy.required_approval.hard_floor.v0"
	decision.RequiredApproval = map[string]any{
		"approval_trigger_code": "gate_override",
		"scope": map[string]any{
			"workspace_id": "workspace-local",
			"run_id":       runID,
			"action_kind":  "action_gate_override",
		},
	}
	if err := store.RecordPolicyDecision(decision); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	decisionHash := firstPolicyDecisionDigest(store)
	approval := ApprovalRecord{
		ApprovalID:             approvalID,
		Status:                 "approved",
		WorkspaceID:            "workspace-local",
		RunID:                  runID,
		ActionKind:             "action_gate_override",
		RequestedAt:            now.Add(-time.Minute),
		ExpiresAt:              ptrTime(now.Add(time.Hour)),
		ApprovalTriggerCode:    "gate_override",
		ChangesIfApproved:      approvalChangesIfApprovedDefault,
		ApprovalAssuranceLevel: "reauthenticated",
		PresenceMode:           "hardware_touch",
		PolicyDecisionHash:     decisionHash,
		ManifestHash:           decision.ManifestHash,
		ActionRequestHash:      decision.ActionRequestHash,
	}
	if err := store.RecordApproval(approval); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	return approval
}

func firstPolicyDecisionDigest(store *Store) string {
	for _, rec := range store.state.PolicyDecisions {
		return rec.Digest
	}
	return ""
}

func recordCheckpointAcceptance(t *testing.T, store *Store, runID string, checkpoint RunnerCheckpointAdvisory, wantAccepted bool, label string) {
	t.Helper()
	accepted, err := store.RecordRunnerCheckpoint(runID, checkpoint)
	if err != nil {
		t.Fatalf("%s RecordRunnerCheckpoint returned error: %v", label, err)
	}
	if accepted != wantAccepted {
		t.Fatalf("%s RecordRunnerCheckpoint accepted=%v, want %v", label, accepted, wantAccepted)
	}
}

func recordResultAcceptance(t *testing.T, store *Store, runID string, result RunnerResultAdvisory, wantAccepted bool, label string) {
	t.Helper()
	accepted, err := store.RecordRunnerResult(runID, result, "")
	if err != nil {
		t.Fatalf("%s RecordRunnerResult returned error: %v", label, err)
	}
	if accepted != wantAccepted {
		t.Fatalf("%s RecordRunnerResult accepted=%v, want %v", label, accepted, wantAccepted)
	}
}

func assertReloadedRunnerDurableState(t *testing.T, rootDir, runID, stepAttemptID string) {
	t.Helper()
	reloaded, err := NewStore(rootDir)
	if err != nil {
		t.Fatalf("NewStore reload returned error: %v", err)
	}
	state, ok := reloaded.RunnerAdvisory(runID)
	if !ok {
		t.Fatalf("RunnerAdvisory(%s) = missing, want present", runID)
	}
	if state.LastResult == nil || state.LastResult.ResultCode != "run_completed" {
		t.Fatalf("last result = %+v, want run_completed", state.LastResult)
	}
	if state.Lifecycle == nil || state.Lifecycle.LifecycleState != "completed" {
		t.Fatalf("lifecycle hint = %+v, want completed", state.Lifecycle)
	}
	hint, ok := state.StepAttempts[stepAttemptID]
	if !ok {
		t.Fatalf("step_attempts missing %s: %+v", stepAttemptID, state.StepAttempts)
	}
	if hint.Status != "finished" {
		t.Fatalf("step attempt status = %q, want finished", hint.Status)
	}
}

func buildRunnerApprovalForTest(runID, idSuffix, status string, occurredAt time.Time, resolvedAt *time.Time) RunnerApproval {
	return RunnerApproval{
		ApprovalID:      "sha256:" + strings.Repeat(idSuffix, 64),
		RunID:           runID,
		StageID:         "stage-1",
		StepID:          "step-1",
		RoleInstanceID:  "role-1",
		Status:          status,
		ApprovalType:    "exact_action",
		BoundActionHash: "sha256:" + strings.Repeat("a", 64),
		OccurredAt:      occurredAt,
		ResolvedAt:      resolvedAt,
	}
}

func recordRunnerApprovalWait(t *testing.T, store *Store, approval RunnerApproval) {
	t.Helper()
	if err := store.RecordRunnerApprovalWait(approval); err != nil {
		t.Fatalf("RecordRunnerApprovalWait(%s) returned error: %v", approval.Status, err)
	}
}

func TestRunnerStepAttemptPhaseMappingAcrossExecutionLoop(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	sequence := []string{"step_attempt_started", "step_validation_started", "approval_wait_entered", "approval_wait_cleared", "step_execution_started", "step_attest_started", "step_attempt_finished"}
	for i, checkpointCode := range sequence {
		accepted, err := store.RecordRunnerCheckpoint("run-phase", RunnerCheckpointAdvisory{
			LifecycleState: "active",
			CheckpointCode: checkpointCode,
			OccurredAt:     now.Add(time.Duration(i) * time.Minute),
			IdempotencyKey: "idem-phase-" + checkpointCode,
			StageID:        "stage-1",
			StepID:         "step-1",
			StepAttemptID:  "attempt-1",
		})
		if err != nil {
			t.Fatalf("RecordRunnerCheckpoint(%s) returned error: %v", checkpointCode, err)
		}
		if !accepted {
			t.Fatalf("RecordRunnerCheckpoint(%s) accepted=false, want true", checkpointCode)
		}
	}
	state, ok := store.RunnerAdvisory("run-phase")
	if !ok {
		t.Fatal("RunnerAdvisory(run-phase) missing")
	}
	hint, ok := state.StepAttempts["attempt-1"]
	if !ok {
		t.Fatalf("step_attempts missing attempt-1: %+v", state.StepAttempts)
	}
	if hint.CurrentPhase != "attest" {
		t.Fatalf("current_phase = %q, want attest", hint.CurrentPhase)
	}
	if hint.PhaseStatus != "finished" {
		t.Fatalf("phase_status = %q, want finished", hint.PhaseStatus)
	}
}

func TestRecordRunnerCheckpointRejectsRoleInstanceWithoutStepIdentity(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 4, 10, 12, 30, 0, 0, time.UTC)
	_, err := store.RecordRunnerCheckpoint("run-role-missing-step", RunnerCheckpointAdvisory{
		LifecycleState: "active",
		CheckpointCode: "step_attempt_started",
		OccurredAt:     now,
		IdempotencyKey: "idem-role-missing-step",
		RoleInstanceID: "role-1",
	})
	if err == nil {
		t.Fatal("RecordRunnerCheckpoint error = nil, want role instance identity validation failure")
	}
}

func TestRecordRunnerResultRejectsRoleInstanceWithoutStepIdentity(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 4, 10, 12, 45, 0, 0, time.UTC)
	_, err := store.RecordRunnerResult("run-role-missing-step", RunnerResultAdvisory{
		LifecycleState: "failed",
		ResultCode:     "run_failed",
		OccurredAt:     now,
		IdempotencyKey: "idem-role-missing-step-result",
		RoleInstanceID: "role-1",
	}, "")
	if err == nil {
		t.Fatal("RecordRunnerResult error = nil, want role instance identity validation failure")
	}
}

func mustReadRunnerJournal(t *testing.T, root string) []RunnerDurableJournalRecord {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, runnerJournalFileName))
	if err != nil {
		t.Fatalf("read runner journal returned error: %v", err)
	}
	trimmed := strings.TrimSpace(string(b))
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	out := make([]RunnerDurableJournalRecord, 0, len(lines))
	for _, line := range lines {
		var rec RunnerDurableJournalRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("unmarshal runner journal line returned error: %v", err)
		}
		out = append(out, rec)
	}
	return out
}
