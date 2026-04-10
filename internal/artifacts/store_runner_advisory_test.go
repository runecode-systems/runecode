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
	accepted, err := store.RecordRunnerCheckpoint("run-durable", RunnerCheckpointAdvisory{
		LifecycleState: "active",
		CheckpointCode: "step_attempt_started",
		OccurredAt:     now,
		IdempotencyKey: "idem-checkpoint-1",
		StageID:        "stage-1",
		StepID:         "step-1",
		RoleInstanceID: "role-1",
		StepAttemptID:  "step-attempt-1",
	})
	if err != nil {
		t.Fatalf("RecordRunnerCheckpoint returned error: %v", err)
	}
	if !accepted {
		t.Fatal("first RecordRunnerCheckpoint accepted=false, want true")
	}
	accepted, err = store.RecordRunnerCheckpoint("run-durable", RunnerCheckpointAdvisory{
		LifecycleState: "active",
		CheckpointCode: "step_attempt_started",
		OccurredAt:     now,
		IdempotencyKey: "idem-checkpoint-1",
		StageID:        "stage-1",
		StepID:         "step-1",
		RoleInstanceID: "role-1",
		StepAttemptID:  "step-attempt-1",
	})
	if err != nil {
		t.Fatalf("duplicate RecordRunnerCheckpoint returned error: %v", err)
	}
	if accepted {
		t.Fatal("duplicate RecordRunnerCheckpoint accepted=true, want false")
	}
	accepted, err = store.RecordRunnerResult("run-durable", RunnerResultAdvisory{
		LifecycleState: "completed",
		ResultCode:     "run_completed",
		OccurredAt:     now.Add(time.Minute),
		IdempotencyKey: "idem-result-1",
		StageID:        "stage-1",
		StepID:         "step-1",
		RoleInstanceID: "role-1",
		StepAttemptID:  "step-attempt-1",
	})
	if err != nil {
		t.Fatalf("RecordRunnerResult returned error: %v", err)
	}
	if !accepted {
		t.Fatal("RecordRunnerResult accepted=false, want true")
	}

	journal := mustReadRunnerJournal(t, store.rootDir)
	if got := len(journal); got != 2 {
		t.Fatalf("runner journal records = %d, want 2", got)
	}

	reloaded, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore reload returned error: %v", err)
	}
	state, ok := reloaded.RunnerAdvisory("run-durable")
	if !ok {
		t.Fatal("RunnerAdvisory(run-durable) = missing, want present")
	}
	if state.LastResult == nil || state.LastResult.ResultCode != "run_completed" {
		t.Fatalf("last result = %+v, want run_completed", state.LastResult)
	}
	if state.Lifecycle == nil || state.Lifecycle.LifecycleState != "completed" {
		t.Fatalf("lifecycle hint = %+v, want completed", state.Lifecycle)
	}
	hint, ok := state.StepAttempts["step-attempt-1"]
	if !ok {
		t.Fatalf("step_attempts missing step-attempt-1: %+v", state.StepAttempts)
	}
	if hint.Status != "finished" {
		t.Fatalf("step attempt status = %q, want finished", hint.Status)
	}
}

func TestRunnerDurableStateMigratesLegacyRunnerAdvisoryMap(t *testing.T) {
	store := newTestStore(t)
	store.mu.Lock()
	store.state.RunnerAdvisoryByRun["run-legacy"] = RunnerAdvisoryState{
		LastCheckpoint: &RunnerCheckpointAdvisory{
			LifecycleState: "blocked",
			CheckpointCode: "approval_wait_entered",
			OccurredAt:     time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC),
			IdempotencyKey: "legacy-idem",
			StageID:        "stage-legacy",
		},
	}
	if err := store.saveStateLocked(); err != nil {
		store.mu.Unlock()
		t.Fatalf("saveStateLocked returned error: %v", err)
	}
	store.mu.Unlock()

	if err := os.Remove(filepath.Join(store.rootDir, runnerSnapshotFileName)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove snapshot returned error: %v", err)
	}
	if err := os.Remove(filepath.Join(store.rootDir, runnerJournalFileName)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove journal returned error: %v", err)
	}

	reloaded, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore reload returned error: %v", err)
	}
	state, ok := reloaded.RunnerAdvisory("run-legacy")
	if !ok || state.LastCheckpoint == nil {
		t.Fatalf("legacy runner advisory missing after migration: ok=%v state=%+v", ok, state)
	}
	if state.LastCheckpoint.CheckpointCode != "approval_wait_entered" {
		t.Fatalf("checkpoint_code = %q, want approval_wait_entered", state.LastCheckpoint.CheckpointCode)
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
	if err := store.RecordRunnerApprovalWait(RunnerApproval{
		ApprovalID:      "sha256:" + strings.Repeat("1", 64),
		RunID:           "run-approval",
		StageID:         "stage-1",
		StepID:          "step-1",
		RoleInstanceID:  "role-1",
		Status:          "pending",
		ApprovalType:    "exact_action",
		BoundActionHash: "sha256:" + strings.Repeat("a", 64),
		OccurredAt:      now,
	}); err != nil {
		t.Fatalf("RecordRunnerApprovalWait pending #1 returned error: %v", err)
	}
	if err := store.RecordRunnerApprovalWait(RunnerApproval{
		ApprovalID:      "sha256:" + strings.Repeat("2", 64),
		RunID:           "run-approval",
		StageID:         "stage-1",
		StepID:          "step-1",
		RoleInstanceID:  "role-1",
		Status:          "pending",
		ApprovalType:    "exact_action",
		BoundActionHash: "sha256:" + strings.Repeat("a", 64),
		OccurredAt:      now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("RecordRunnerApprovalWait pending #2 returned error: %v", err)
	}
	if err := store.RecordRunnerApprovalWait(RunnerApproval{
		ApprovalID:      "sha256:" + strings.Repeat("2", 64),
		RunID:           "run-approval",
		StageID:         "stage-1",
		StepID:          "step-1",
		RoleInstanceID:  "role-1",
		Status:          "pending",
		ApprovalType:    "exact_action",
		BoundActionHash: "sha256:" + strings.Repeat("a", 64),
		OccurredAt:      now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("duplicate RecordRunnerApprovalWait pending #2 returned error: %v", err)
	}
	resolved := now.Add(2 * time.Minute)
	if err := store.RecordRunnerApprovalWait(RunnerApproval{
		ApprovalID:      "sha256:" + strings.Repeat("2", 64),
		RunID:           "run-approval",
		StageID:         "stage-1",
		StepID:          "step-1",
		RoleInstanceID:  "role-1",
		Status:          "consumed",
		ApprovalType:    "exact_action",
		BoundActionHash: "sha256:" + strings.Repeat("a", 64),
		OccurredAt:      resolved,
		ResolvedAt:      &resolved,
	}); err != nil {
		t.Fatalf("RecordRunnerApprovalWait consumed returned error: %v", err)
	}

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
