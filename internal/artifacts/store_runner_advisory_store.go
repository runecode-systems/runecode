package artifacts

import (
	"fmt"
	"strings"
)

const (
	runnerJournalFamily        = "runner_durable_journal"
	runnerSnapshotFamily       = "runner_durable_snapshot"
	runnerDurableSchemaVersion = 1
	runnerJournalFileName      = "runner_state.journal"
	runnerSnapshotFileName     = "runner_state.snapshot.json"
)

func (s *Store) reconcileRunnerAdvisoryDurableStateLocked() (bool, error) {
	runs, idem, seq, migrated, err := loadRunnerDurableState(s.rootDir)
	if err != nil {
		return false, err
	}
	if len(runs) == 0 && len(s.state.RunnerAdvisoryByRun) > 0 {
		runs = copyRunnerAdvisoryByRun(s.state.RunnerAdvisoryByRun)
		migrated = true
	}
	if len(runs) > 0 || len(idem) > 0 {
		s.state.RunnerAdvisoryByRun = runs
	}
	consumedChanged := reconcileConsumedGateOverrideApprovalsLocked(&s.state)
	if migrated {
		for runID := range runs {
			ensureRunnerStatusExists(&s.state, runID)
		}
		return true || consumedChanged, nil
	}
	return consumedChanged, ensureRunnerDurableFiles(s.rootDir, runs, idem, seq)
}

func (s *Store) RecordRunnerCheckpoint(runID string, checkpoint RunnerCheckpointAdvisory) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmedRunID, record, err := validatedRunnerCheckpointRecord(runID, checkpoint)
	if err != nil {
		return false, err
	}
	record.RunID = trimmedRunID
	return s.appendRunnerJournalRecordLocked(record)
}

func validatedRunnerCheckpointRecord(runID string, checkpoint RunnerCheckpointAdvisory) (string, RunnerDurableJournalRecord, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return "", RunnerDurableJournalRecord{}, fmt.Errorf("run id is required")
	}
	if strings.TrimSpace(checkpoint.IdempotencyKey) == "" {
		return "", RunnerDurableJournalRecord{}, fmt.Errorf("idempotency key is required")
	}
	if strings.TrimSpace(checkpoint.LifecycleState) == "" {
		return "", RunnerDurableJournalRecord{}, fmt.Errorf("lifecycle state is required")
	}
	if strings.TrimSpace(checkpoint.CheckpointCode) == "" {
		return "", RunnerDurableJournalRecord{}, fmt.Errorf("checkpoint code is required")
	}
	if err := validateRunnerLifecycleState(checkpoint.LifecycleState); err != nil {
		return "", RunnerDurableJournalRecord{}, err
	}
	if err := validateRunnerStepIdentity(checkpoint.StageID, checkpoint.StepID, checkpoint.RoleInstanceID); err != nil {
		return "", RunnerDurableJournalRecord{}, err
	}
	return trimmedRunID, RunnerDurableJournalRecord{
		Family:         runnerJournalFamily,
		SchemaVersion:  runnerDurableSchemaVersion,
		RecordType:     "checkpoint",
		IdempotencyKey: checkpoint.IdempotencyKey,
		OccurredAt:     checkpoint.OccurredAt.UTC(),
		Checkpoint:     cloneCheckpoint(&checkpoint),
	}, nil
}

func (s *Store) RecordRunnerResult(runID string, result RunnerResultAdvisory, overridePolicyRef string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmedRunID, record, err := validatedRunnerResultRecord(runID, result)
	if err != nil {
		return false, err
	}
	runs, idem, seq, _, err := loadRunnerDurableState(s.rootDir)
	if err != nil {
		return false, err
	}
	key := scopedRunnerIdempotencyKey(record)
	if prevSeq, ok := idem[key]; ok && prevSeq > 0 {
		s.state.RunnerAdvisoryByRun = runs
		return false, nil
	}
	rollback, err := s.consumeGateOverrideApprovalForResult(trimmedRunID, strings.TrimSpace(overridePolicyRef), result)
	if err != nil {
		return false, err
	}
	record.Sequence = seq + 1
	if err := appendRunnerJournalRecord(s.rootDir, record); err != nil {
		rollback()
		return false, err
	}
	if err := persistRunnerResultDurableStateLocked(&s.state, s.rootDir, runs, idem, record); err != nil {
		return false, err
	}
	return true, nil
}

func validatedRunnerResultRecord(runID string, result RunnerResultAdvisory) (string, RunnerDurableJournalRecord, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return "", RunnerDurableJournalRecord{}, fmt.Errorf("run id is required")
	}
	if strings.TrimSpace(result.IdempotencyKey) == "" {
		return "", RunnerDurableJournalRecord{}, fmt.Errorf("idempotency key is required")
	}
	if strings.TrimSpace(result.LifecycleState) == "" {
		return "", RunnerDurableJournalRecord{}, fmt.Errorf("lifecycle state is required")
	}
	if strings.TrimSpace(result.ResultCode) == "" {
		return "", RunnerDurableJournalRecord{}, fmt.Errorf("result code is required")
	}
	if err := validateRunnerTerminalLifecycleState(result.LifecycleState); err != nil {
		return "", RunnerDurableJournalRecord{}, err
	}
	if err := validateRunnerStepIdentity(result.StageID, result.StepID, result.RoleInstanceID); err != nil {
		return "", RunnerDurableJournalRecord{}, err
	}
	return trimmedRunID, RunnerDurableJournalRecord{
		Family:         runnerJournalFamily,
		SchemaVersion:  runnerDurableSchemaVersion,
		RecordType:     "result",
		RunID:          trimmedRunID,
		IdempotencyKey: result.IdempotencyKey,
		OccurredAt:     result.OccurredAt.UTC(),
		Result:         cloneResult(&result),
	}, nil
}

func (s *Store) consumeGateOverrideApprovalForResult(runID, overridePolicyRef string, result RunnerResultAdvisory) (func(), error) {
	approvalID, priorApproval, mutated, err := s.consumeGateOverrideApprovalLocked(runID, overridePolicyRef, result)
	if err != nil {
		return nil, err
	}
	return func() {
		if mutated {
			s.state.Approvals[approvalID] = priorApproval
			rebuildRunApprovalRefsLocked(&s.state)
		}
	}, nil
}

func persistRunnerResultDurableStateLocked(state *StoreState, rootDir string, runs map[string]RunnerAdvisoryState, idem map[string]int64, record RunnerDurableJournalRecord) error {
	if err := applyRunnerJournalRecord(runs, record); err != nil {
		return err
	}
	state.RunnerAdvisoryByRun = runs
	ensureRunnerStatusExists(state, record.RunID)
	idem[scopedRunnerIdempotencyKey(record)] = record.Sequence
	if err := writeRunnerSnapshot(rootDir, RunnerDurableSnapshot{
		Family:        runnerSnapshotFamily,
		SchemaVersion: runnerDurableSchemaVersion,
		LastSequence:  record.Sequence,
		Runs:          runs,
		Idempotency:   idem,
	}); err != nil {
		return err
	}
	return saveStoreStateLocked(state, rootDir)
}

func ensureRunnerStatusExists(state *StoreState, runID string) {
	if _, ok := state.Runs[runID]; !ok {
		state.Runs[runID] = "pending"
	}
}

func saveStoreStateLocked(state *StoreState, rootDir string) error {
	sio, err := newStoreIO(rootDir, defaultBlobDir(rootDir))
	if err != nil {
		return err
	}
	return sio.saveStateFile(*state)
}

func (s *Store) RecordRunnerApprovalWait(approval RunnerApproval) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(approval.ApprovalID) == "" {
		return fmt.Errorf("approval id is required")
	}
	if strings.TrimSpace(approval.RunID) == "" {
		return fmt.Errorf("run id is required")
	}
	if err := validateRunnerApprovalStatus(approval.Status); err != nil {
		return err
	}
	if err := validateRunnerApprovalTypeAndBinding(approval); err != nil {
		return err
	}
	record := RunnerDurableJournalRecord{
		Family:         runnerJournalFamily,
		SchemaVersion:  runnerDurableSchemaVersion,
		RecordType:     "approval_wait",
		RunID:          strings.TrimSpace(approval.RunID),
		IdempotencyKey: strings.TrimSpace(approval.ApprovalID) + "@" + strings.TrimSpace(approval.Status),
		OccurredAt:     approval.OccurredAt.UTC(),
		Approval:       cloneRunnerApproval(&approval),
	}
	_, err := s.appendRunnerJournalRecordLocked(record)
	return err
}

func (s *Store) RunnerAdvisory(runID string) (RunnerAdvisoryState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return RunnerAdvisoryState{}, false
	}
	state, ok := s.state.RunnerAdvisoryByRun[trimmedRunID]
	if !ok {
		return RunnerAdvisoryState{}, false
	}
	return copyRunnerAdvisoryState(state), true
}
