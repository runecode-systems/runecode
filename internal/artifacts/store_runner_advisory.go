package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
			if _, ok := s.state.Runs[runID]; !ok {
				s.state.Runs[runID] = "pending"
			}
		}
		return true || consumedChanged, nil
	}
	return consumedChanged, ensureRunnerDurableFiles(s.rootDir, runs, idem, seq)
}

func (s *Store) RecordRunnerCheckpoint(runID string, checkpoint RunnerCheckpointAdvisory) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return false, fmt.Errorf("run id is required")
	}
	if strings.TrimSpace(checkpoint.IdempotencyKey) == "" {
		return false, fmt.Errorf("idempotency key is required")
	}
	if strings.TrimSpace(checkpoint.LifecycleState) == "" {
		return false, fmt.Errorf("lifecycle state is required")
	}
	if strings.TrimSpace(checkpoint.CheckpointCode) == "" {
		return false, fmt.Errorf("checkpoint code is required")
	}
	if err := validateRunnerLifecycleState(checkpoint.LifecycleState); err != nil {
		return false, err
	}
	if err := validateRunnerStepIdentity(checkpoint.StageID, checkpoint.StepID, checkpoint.RoleInstanceID); err != nil {
		return false, err
	}
	record := RunnerDurableJournalRecord{
		Family:         runnerJournalFamily,
		SchemaVersion:  runnerDurableSchemaVersion,
		RecordType:     "checkpoint",
		RunID:          trimmedRunID,
		IdempotencyKey: checkpoint.IdempotencyKey,
		OccurredAt:     checkpoint.OccurredAt.UTC(),
		Checkpoint:     cloneCheckpoint(&checkpoint),
	}
	return s.appendRunnerJournalRecordLocked(record)
}

func (s *Store) RecordRunnerResult(runID string, result RunnerResultAdvisory, overridePolicyRef string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return false, fmt.Errorf("run id is required")
	}
	if strings.TrimSpace(result.IdempotencyKey) == "" {
		return false, fmt.Errorf("idempotency key is required")
	}
	if strings.TrimSpace(result.LifecycleState) == "" {
		return false, fmt.Errorf("lifecycle state is required")
	}
	if strings.TrimSpace(result.ResultCode) == "" {
		return false, fmt.Errorf("result code is required")
	}
	if err := validateRunnerTerminalLifecycleState(result.LifecycleState); err != nil {
		return false, err
	}
	if err := validateRunnerStepIdentity(result.StageID, result.StepID, result.RoleInstanceID); err != nil {
		return false, err
	}
	record := RunnerDurableJournalRecord{
		Family:         runnerJournalFamily,
		SchemaVersion:  runnerDurableSchemaVersion,
		RecordType:     "result",
		RunID:          trimmedRunID,
		IdempotencyKey: result.IdempotencyKey,
		OccurredAt:     result.OccurredAt.UTC(),
		Result:         cloneResult(&result),
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
	approvalID, priorApproval, mutated, err := s.consumeGateOverrideApprovalLocked(trimmedRunID, strings.TrimSpace(overridePolicyRef), result)
	if err != nil {
		return false, err
	}
	record.Sequence = seq + 1
	if err := appendRunnerJournalRecord(s.rootDir, record); err != nil {
		if mutated {
			s.state.Approvals[approvalID] = priorApproval
			rebuildRunApprovalRefsLocked(&s.state)
		}
		return false, err
	}
	if err := applyRunnerJournalRecord(runs, record); err != nil {
		return false, err
	}
	s.state.RunnerAdvisoryByRun = runs
	if _, ok := s.state.Runs[record.RunID]; !ok {
		s.state.Runs[record.RunID] = "pending"
	}
	idem[key] = record.Sequence
	if err := writeRunnerSnapshot(s.rootDir, RunnerDurableSnapshot{
		Family:        runnerSnapshotFamily,
		SchemaVersion: runnerDurableSchemaVersion,
		LastSequence:  record.Sequence,
		Runs:          runs,
		Idempotency:   idem,
	}); err != nil {
		return false, err
	}
	if err := s.saveStateLocked(); err != nil {
		return false, err
	}
	return true, nil
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

func (s *Store) appendRunnerJournalRecordLocked(record RunnerDurableJournalRecord) (bool, error) {
	runs, idem, seq, _, err := loadRunnerDurableState(s.rootDir)
	if err != nil {
		return false, err
	}
	key := scopedRunnerIdempotencyKey(record)
	if prevSeq, ok := idem[key]; ok {
		if prevSeq > 0 {
			s.state.RunnerAdvisoryByRun = runs
			return false, nil
		}
	}
	record.Sequence = seq + 1
	if err := appendRunnerJournalRecord(s.rootDir, record); err != nil {
		return false, err
	}
	if err := applyRunnerJournalRecord(runs, record); err != nil {
		return false, err
	}
	idem[key] = record.Sequence
	if err := writeRunnerSnapshot(s.rootDir, RunnerDurableSnapshot{
		Family:        runnerSnapshotFamily,
		SchemaVersion: runnerDurableSchemaVersion,
		LastSequence:  record.Sequence,
		Runs:          runs,
		Idempotency:   idem,
	}); err != nil {
		return false, err
	}
	s.state.RunnerAdvisoryByRun = runs
	if _, ok := s.state.Runs[record.RunID]; !ok {
		s.state.Runs[record.RunID] = "pending"
	}
	if err := s.saveStateLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func loadRunnerDurableState(rootDir string) (map[string]RunnerAdvisoryState, map[string]int64, int64, bool, error) {
	runs, idem, seq, err := loadRunnerSnapshot(rootDir)
	if err != nil {
		return nil, nil, 0, false, err
	}
	records, err := readRunnerJournalRecords(rootDir)
	if err != nil {
		return nil, nil, 0, false, err
	}
	for _, record := range records {
		if record.Sequence <= seq {
			continue
		}
		key := scopedRunnerIdempotencyKey(record)
		if prev, ok := idem[key]; ok && prev >= record.Sequence {
			continue
		}
		if err := applyRunnerJournalRecord(runs, record); err != nil {
			return nil, nil, 0, false, err
		}
		idem[key] = record.Sequence
		seq = record.Sequence
	}
	needsSnapshot := false
	if len(records) > 0 {
		last := records[len(records)-1].Sequence
		if last != seq {
			seq = last
			needsSnapshot = true
		}
	}
	return runs, idem, seq, needsSnapshot, nil
}

func loadRunnerSnapshot(rootDir string) (map[string]RunnerAdvisoryState, map[string]int64, int64, error) {
	path := filepath.Join(rootDir, runnerSnapshotFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]RunnerAdvisoryState{}, map[string]int64{}, 0, nil
		}
		return nil, nil, 0, err
	}
	if len(b) == 0 {
		return map[string]RunnerAdvisoryState{}, map[string]int64{}, 0, nil
	}
	var snap RunnerDurableSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, nil, 0, err
	}
	if snap.SchemaVersion == 0 {
		snap.SchemaVersion = 1
	}
	if snap.SchemaVersion != runnerDurableSchemaVersion {
		return nil, nil, 0, fmt.Errorf("unsupported runner snapshot schema version %d", snap.SchemaVersion)
	}
	if snap.Family != "" && snap.Family != runnerSnapshotFamily {
		return nil, nil, 0, fmt.Errorf("unsupported runner snapshot family %q", snap.Family)
	}
	if snap.Runs == nil {
		snap.Runs = map[string]RunnerAdvisoryState{}
	}
	if snap.Idempotency == nil {
		snap.Idempotency = map[string]int64{}
	}
	return snap.Runs, snap.Idempotency, snap.LastSequence, nil
}

func readRunnerJournalRecords(rootDir string) ([]RunnerDurableJournalRecord, error) {
	lines, err := loadRunnerJournalLines(rootDir)
	if err != nil {
		return nil, err
	}
	records := make([]RunnerDurableJournalRecord, 0, len(lines))
	for _, line := range lines {
		rec, ok, err := parseRunnerJournalRecord(line)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		records = append(records, rec)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Sequence < records[j].Sequence })
	return records, nil
}

func loadRunnerJournalLines(rootDir string) ([]string, error) {
	path := filepath.Join(rootDir, runnerJournalFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	trimmed := strings.TrimSpace(string(b))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func parseRunnerJournalRecord(line string) (RunnerDurableJournalRecord, bool, error) {
	if strings.TrimSpace(line) == "" {
		return RunnerDurableJournalRecord{}, false, nil
	}
	var rec RunnerDurableJournalRecord
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		return RunnerDurableJournalRecord{}, false, err
	}
	if err := validateRunnerJournalRecordShape(&rec); err != nil {
		return RunnerDurableJournalRecord{}, false, err
	}
	return rec, true, nil
}

func validateRunnerJournalRecordShape(rec *RunnerDurableJournalRecord) error {
	if rec.SchemaVersion == 0 {
		rec.SchemaVersion = 1
	}
	if rec.SchemaVersion != runnerDurableSchemaVersion {
		return fmt.Errorf("unsupported runner journal schema version %d", rec.SchemaVersion)
	}
	if rec.Family != "" && rec.Family != runnerJournalFamily {
		return fmt.Errorf("unsupported runner journal family %q", rec.Family)
	}
	return nil
}

func (s *Store) consumeGateOverrideApprovalLocked(runID, policyDecisionRef string, result RunnerResultAdvisory) (string, ApprovalRecord, bool, error) {
	if strings.TrimSpace(policyDecisionRef) == "" {
		return "", ApprovalRecord{}, false, nil
	}
	for approvalID, approval := range s.state.Approvals {
		if !matchesApprovedGateOverrideApprovalRecord(approval, runID, policyDecisionRef) {
			continue
		}
		prior := approval
		now := result.OccurredAt.UTC()
		approval.Status = "consumed"
		approval.DecidedAt = &now
		approval.ConsumedAt = &now
		s.state.Approvals[approvalID] = approval
		rebuildRunApprovalRefsLocked(&s.state)
		return approvalID, prior, true, nil
	}
	return "", ApprovalRecord{}, false, fmt.Errorf("gate override requires explicit approved approval")
}

func matchesApprovedGateOverrideApprovalRecord(approval ApprovalRecord, runID, policyDecisionRef string) bool {
	if strings.TrimSpace(approval.RunID) != strings.TrimSpace(runID) {
		return false
	}
	if strings.TrimSpace(approval.ActionKind) != "action_gate_override" {
		return false
	}
	if strings.TrimSpace(approval.PolicyDecisionHash) != strings.TrimSpace(policyDecisionRef) {
		return false
	}
	return strings.TrimSpace(approval.Status) == "approved"
}

func reconcileConsumedGateOverrideApprovalsLocked(state *StoreState) bool {
	if state == nil || len(state.RunnerAdvisoryByRun) == 0 || len(state.Approvals) == 0 {
		return false
	}
	changed := false
	for runID, advisory := range state.RunnerAdvisoryByRun {
		for _, gateAttempt := range advisory.GateAttempts {
			policyRef := strings.TrimSpace(gateAttempt.OverridePolicyRef)
			if policyRef == "" || strings.TrimSpace(gateAttempt.GateState) != "overridden" {
				continue
			}
			for approvalID, approval := range state.Approvals {
				if !matchesApprovedGateOverrideApprovalRecord(approval, runID, policyRef) {
					continue
				}
				when := gateAttempt.FinishedAt.UTC()
				if when.IsZero() {
					when = gateAttempt.LastUpdatedAt.UTC()
				}
				approval.Status = "consumed"
				approval.DecidedAt = &when
				approval.ConsumedAt = &when
				state.Approvals[approvalID] = approval
				changed = true
				break
			}
		}
	}
	if changed {
		rebuildRunApprovalRefsLocked(state)
	}
	return changed
}

func appendRunnerJournalRecord(rootDir string, record RunnerDurableJournalRecord) error {
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(rootDir, runnerJournalFileName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func writeRunnerSnapshot(rootDir string, snapshot RunnerDurableSnapshot) error {
	b, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(rootDir, runnerSnapshotFileName), b, 0o600)
}

func ensureRunnerDurableFiles(rootDir string, runs map[string]RunnerAdvisoryState, idem map[string]int64, seq int64) error {
	if err := writeRunnerSnapshot(rootDir, RunnerDurableSnapshot{
		Family:        runnerSnapshotFamily,
		SchemaVersion: runnerDurableSchemaVersion,
		LastSequence:  seq,
		Runs:          runs,
		Idempotency:   idem,
	}); err != nil {
		return err
	}
	path := filepath.Join(rootDir, runnerJournalFileName)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte{}, 0o600)
}

func scopedRunnerIdempotencyKey(record RunnerDurableJournalRecord) string {
	runID := strings.TrimSpace(record.RunID)
	key := strings.TrimSpace(record.IdempotencyKey)
	if runID == "" {
		return key
	}
	if key == "" {
		return runID + "@"
	}
	return runID + "@" + key
}

func applyRunnerJournalRecord(runs map[string]RunnerAdvisoryState, record RunnerDurableJournalRecord) error {
	runID := strings.TrimSpace(record.RunID)
	if runID == "" {
		return fmt.Errorf("runner journal record run_id is required")
	}
	if strings.TrimSpace(record.IdempotencyKey) == "" {
		return fmt.Errorf("runner journal idempotency_key is required")
	}
	state := runs[runID]
	switch record.RecordType {
	case "checkpoint":
		if record.Checkpoint == nil {
			return fmt.Errorf("runner checkpoint journal record missing payload")
		}
		state.LastCheckpoint = cloneCheckpoint(record.Checkpoint)
		state.Lifecycle = &RunnerLifecycleHint{
			LifecycleState: record.Checkpoint.LifecycleState,
			OccurredAt:     record.Checkpoint.OccurredAt.UTC(),
			StageID:        record.Checkpoint.StageID,
			StepID:         record.Checkpoint.StepID,
			RoleInstanceID: record.Checkpoint.RoleInstanceID,
			StageAttemptID: record.Checkpoint.StageAttemptID,
			StepAttemptID:  record.Checkpoint.StepAttemptID,
			GateAttemptID:  record.Checkpoint.GateAttemptID,
		}
		applyStepHintFromCheckpoint(&state, runID, *record.Checkpoint)
		applyGateHintFromCheckpoint(&state, runID, *record.Checkpoint)
	case "result":
		if record.Result == nil {
			return fmt.Errorf("runner result journal record missing payload")
		}
		state.LastResult = cloneResult(record.Result)
		state.Lifecycle = &RunnerLifecycleHint{
			LifecycleState: record.Result.LifecycleState,
			OccurredAt:     record.Result.OccurredAt.UTC(),
			StageID:        record.Result.StageID,
			StepID:         record.Result.StepID,
			RoleInstanceID: record.Result.RoleInstanceID,
			StageAttemptID: record.Result.StageAttemptID,
			StepAttemptID:  record.Result.StepAttemptID,
			GateAttemptID:  record.Result.GateAttemptID,
		}
		applyStepHintFromResult(&state, runID, *record.Result)
		applyGateHintFromResult(&state, runID, *record.Result)
	case "approval_wait":
		if record.Approval == nil {
			return fmt.Errorf("runner approval journal record missing payload")
		}
		applyApprovalWait(&state, *record.Approval)
	default:
		return fmt.Errorf("unsupported runner journal record_type %q", record.RecordType)
	}
	runs[runID] = state
	return nil
}

func applyStepHintFromCheckpoint(state *RunnerAdvisoryState, runID string, checkpoint RunnerCheckpointAdvisory) {
	stepAttemptID := strings.TrimSpace(checkpoint.StepAttemptID)
	if stepAttemptID == "" {
		return
	}
	if state.StepAttempts == nil {
		state.StepAttempts = map[string]RunnerStepHint{}
	}
	hint := state.StepAttempts[stepAttemptID]
	hint.StepAttemptID = stepAttemptID
	hint.RunID = runID
	hint.GateID = checkpoint.GateID
	hint.GateKind = checkpoint.GateKind
	hint.GateVersion = checkpoint.GateVersion
	hint.GateState = checkpoint.GateState
	hint.StageID = checkpoint.StageID
	hint.StepID = checkpoint.StepID
	hint.RoleInstanceID = checkpoint.RoleInstanceID
	hint.StageAttemptID = checkpoint.StageAttemptID
	hint.GateAttemptID = checkpoint.GateAttemptID
	hint.GateEvidenceRef = checkpoint.GateEvidenceRef
	hint.LastUpdatedAt = checkpoint.OccurredAt.UTC()
	if phase, ok := runnerExecutionPhaseForCheckpointCode(checkpoint.CheckpointCode); ok {
		hint.CurrentPhase = phase
		hint.PhaseStatus = runnerPhaseStatusForCheckpointCode(checkpoint.CheckpointCode)
	}
	switch checkpoint.CheckpointCode {
	case "step_attempt_started":
		hint.Status = "started"
		hint.StartedAt = checkpoint.OccurredAt.UTC()
	case "step_attempt_finished":
		hint.Status = "finished"
		t := checkpoint.OccurredAt.UTC()
		hint.FinishedAt = t
	default:
		if strings.TrimSpace(hint.Status) == "" {
			hint.Status = "active"
		}
	}
	state.StepAttempts[stepAttemptID] = hint
}

func applyStepHintFromResult(state *RunnerAdvisoryState, runID string, result RunnerResultAdvisory) {
	stepAttemptID := strings.TrimSpace(result.StepAttemptID)
	if stepAttemptID == "" {
		return
	}
	if state.StepAttempts == nil {
		state.StepAttempts = map[string]RunnerStepHint{}
	}
	hint := state.StepAttempts[stepAttemptID]
	hint.StepAttemptID = stepAttemptID
	hint.RunID = runID
	hint.GateID = result.GateID
	hint.GateKind = result.GateKind
	hint.GateVersion = result.GateVersion
	hint.GateState = result.GateState
	hint.StageID = result.StageID
	hint.StepID = result.StepID
	hint.RoleInstanceID = result.RoleInstanceID
	hint.StageAttemptID = result.StageAttemptID
	hint.GateAttemptID = result.GateAttemptID
	hint.GateEvidenceRef = result.GateEvidenceRef
	hint.Status = "finished"
	hint.CurrentPhase = "attest"
	hint.PhaseStatus = "finished"
	hint.LastUpdatedAt = result.OccurredAt.UTC()
	t := result.OccurredAt.UTC()
	hint.FinishedAt = t
	state.StepAttempts[stepAttemptID] = hint
}

func applyGateHintFromCheckpoint(state *RunnerAdvisoryState, runID string, checkpoint RunnerCheckpointAdvisory) {
	gateAttemptID := strings.TrimSpace(checkpoint.GateAttemptID)
	if gateAttemptID == "" {
		return
	}
	if state.GateAttempts == nil {
		state.GateAttempts = map[string]RunnerGateHint{}
	}
	hint := state.GateAttempts[gateAttemptID]
	hint.GateAttemptID = gateAttemptID
	hint.RunID = runID
	hint.PlanCheckpoint = checkpoint.PlanCheckpoint
	hint.PlanOrderIndex = checkpoint.PlanOrderIndex
	hint.GateID = checkpoint.GateID
	hint.GateKind = checkpoint.GateKind
	hint.GateVersion = checkpoint.GateVersion
	hint.GateState = checkpoint.GateState
	hint.StageID = checkpoint.StageID
	hint.StepID = checkpoint.StepID
	hint.RoleInstanceID = checkpoint.RoleInstanceID
	hint.StageAttemptID = checkpoint.StageAttemptID
	hint.StepAttemptID = checkpoint.StepAttemptID
	hint.GateEvidenceRef = checkpoint.GateEvidenceRef
	hint.LastUpdatedAt = checkpoint.OccurredAt.UTC()
	if strings.TrimSpace(hint.ResultCode) == "" {
		hint.ResultCode = checkpoint.CheckpointCode
	}
	if !hint.Terminal && checkpoint.CheckpointCode == "gate_started" {
		hint.StartedAt = checkpoint.OccurredAt.UTC()
	}
	state.GateAttempts[gateAttemptID] = hint
}

func applyGateHintFromResult(state *RunnerAdvisoryState, runID string, result RunnerResultAdvisory) {
	gateAttemptID := strings.TrimSpace(result.GateAttemptID)
	if gateAttemptID == "" {
		return
	}
	if state.GateAttempts == nil {
		state.GateAttempts = map[string]RunnerGateHint{}
	}
	hint := state.GateAttempts[gateAttemptID]
	hint.GateAttemptID = gateAttemptID
	hint.RunID = runID
	hint.PlanCheckpoint = result.PlanCheckpoint
	hint.PlanOrderIndex = result.PlanOrderIndex
	hint.GateID = result.GateID
	hint.GateKind = result.GateKind
	hint.GateVersion = result.GateVersion
	hint.GateState = result.GateState
	hint.StageID = result.StageID
	hint.StepID = result.StepID
	hint.RoleInstanceID = result.RoleInstanceID
	hint.StageAttemptID = result.StageAttemptID
	hint.StepAttemptID = result.StepAttemptID
	hint.GateEvidenceRef = result.GateEvidenceRef
	hint.FailureReasonCode = result.FailureReasonCode
	hint.OverrideFailedRef = result.OverrideFailedRef
	hint.OverrideActionHash = result.OverrideActionHash
	hint.OverridePolicyRef = result.OverridePolicyRef
	hint.ResultRef = result.ResultRef
	hint.ResultCode = result.ResultCode
	hint.Terminal = true
	hint.LastUpdatedAt = result.OccurredAt.UTC()
	t := result.OccurredAt.UTC()
	hint.FinishedAt = t
	if hint.StartedAt.IsZero() {
		hint.StartedAt = t
	}
	state.GateAttempts[gateAttemptID] = hint
}

func runnerExecutionPhaseForCheckpointCode(code string) (string, bool) {
	switch strings.TrimSpace(code) {
	case "step_attempt_started", "action_request_issued":
		return "propose", true
	case "gate_attempt_started", "gate_attempt_finished", "step_validation_started", "step_validation_finished":
		return "validate", true
	case "approval_wait_entered", "approval_wait_cleared":
		return "authorize", true
	case "step_execution_started", "step_execution_finished":
		return "execute", true
	case "step_attest_started", "step_attest_finished", "step_attempt_finished":
		return "attest", true
	default:
		return "", false
	}
}

func runnerPhaseStatusForCheckpointCode(code string) string {
	switch strings.TrimSpace(code) {
	case "gate_attempt_finished", "step_validation_finished", "approval_wait_cleared", "step_execution_finished", "step_attest_finished", "step_attempt_finished":
		return "finished"
	case "approval_wait_entered":
		return "waiting"
	default:
		return "started"
	}
}

func applyApprovalWait(state *RunnerAdvisoryState, approval RunnerApproval) {
	if state.ApprovalWaits == nil {
		state.ApprovalWaits = map[string]RunnerApproval{}
	}
	approvalID := strings.TrimSpace(approval.ApprovalID)
	approval.Status = strings.TrimSpace(approval.Status)
	if approval.Status == "pending" {
		for existingID, existing := range state.ApprovalWaits {
			if existingID == approvalID || existing.Status != "pending" {
				continue
			}
			if !runnerApprovalSupersedesByIdentity(existing, approval) {
				continue
			}
			now := approval.OccurredAt.UTC()
			existing.Status = "superseded"
			existing.SupersededByApproval = approvalID
			existing.ResolvedAt = &now
			state.ApprovalWaits[existingID] = existing
		}
	}
	state.ApprovalWaits[approvalID] = approval
}

func runnerApprovalSupersedesByIdentity(current, incoming RunnerApproval) bool {
	if current.ApprovalType != incoming.ApprovalType {
		return false
	}
	if current.RunID != incoming.RunID || current.StageID != incoming.StageID || current.StepID != incoming.StepID || current.RoleInstanceID != incoming.RoleInstanceID {
		return false
	}
	if incoming.ApprovalType == "exact_action" {
		return strings.TrimSpace(current.BoundActionHash) != "" && current.BoundActionHash == incoming.BoundActionHash
	}
	if incoming.ApprovalType == "stage_sign_off" {
		return strings.TrimSpace(current.BoundStageSummaryHash) != "" && current.BoundStageSummaryHash == incoming.BoundStageSummaryHash
	}
	return false
}

func validateRunnerStepIdentity(stageID, stepID, roleInstanceID string) error {
	if strings.TrimSpace(stageID) == "" && strings.TrimSpace(stepID) != "" {
		return fmt.Errorf("stage id is required when step id is set")
	}
	if strings.TrimSpace(roleInstanceID) == "" {
		return nil
	}
	return nil
}

func validateRunnerLifecycleState(state string) error {
	switch strings.TrimSpace(state) {
	case "pending", "starting", "active", "blocked", "recovering":
		return nil
	default:
		return fmt.Errorf("unsupported runner lifecycle state %q", state)
	}
}

func validateRunnerTerminalLifecycleState(state string) error {
	switch strings.TrimSpace(state) {
	case "completed", "failed", "cancelled":
		return nil
	default:
		return fmt.Errorf("unsupported runner terminal lifecycle state %q", state)
	}
}

func validateRunnerApprovalStatus(status string) error {
	switch strings.TrimSpace(status) {
	case "pending", "approved", "denied", "expired", "superseded", "cancelled", "consumed":
		return nil
	default:
		return fmt.Errorf("unsupported runner approval status %q", status)
	}
}

func validateRunnerApprovalTypeAndBinding(approval RunnerApproval) error {
	switch strings.TrimSpace(approval.ApprovalType) {
	case "exact_action":
		if !isValidDigest(strings.TrimSpace(approval.BoundActionHash)) {
			return fmt.Errorf("bound action hash is required for exact_action approval")
		}
	case "stage_sign_off":
		if !isValidDigest(strings.TrimSpace(approval.BoundStageSummaryHash)) {
			return fmt.Errorf("bound stage summary hash is required for stage_sign_off approval")
		}
	default:
		return fmt.Errorf("unsupported approval type %q", approval.ApprovalType)
	}
	return nil
}

func copyRunnerAdvisoryState(in RunnerAdvisoryState) RunnerAdvisoryState {
	out := RunnerAdvisoryState{}
	if in.LastCheckpoint != nil {
		out.LastCheckpoint = cloneCheckpoint(in.LastCheckpoint)
	}
	if in.LastResult != nil {
		out.LastResult = cloneResult(in.LastResult)
	}
	if in.Lifecycle != nil {
		lifecycle := *in.Lifecycle
		out.Lifecycle = &lifecycle
	}
	if len(in.StepAttempts) > 0 {
		out.StepAttempts = make(map[string]RunnerStepHint, len(in.StepAttempts))
		for k, v := range in.StepAttempts {
			out.StepAttempts[k] = v
		}
	}
	if len(in.GateAttempts) > 0 {
		out.GateAttempts = make(map[string]RunnerGateHint, len(in.GateAttempts))
		for k, v := range in.GateAttempts {
			out.GateAttempts[k] = v
		}
	}
	if len(in.ApprovalWaits) > 0 {
		out.ApprovalWaits = make(map[string]RunnerApproval, len(in.ApprovalWaits))
		for k, v := range in.ApprovalWaits {
			copyApproval := v
			if v.ResolvedAt != nil {
				t := *v.ResolvedAt
				copyApproval.ResolvedAt = &t
			}
			out.ApprovalWaits[k] = copyApproval
		}
	}
	return out
}

func cloneCheckpoint(in *RunnerCheckpointAdvisory) *RunnerCheckpointAdvisory {
	if in == nil {
		return nil
	}
	out := *in
	if in.Details != nil {
		out.Details = copyMap(in.Details)
	}
	return &out
}

func cloneResult(in *RunnerResultAdvisory) *RunnerResultAdvisory {
	if in == nil {
		return nil
	}
	out := *in
	if in.Details != nil {
		out.Details = copyMap(in.Details)
	}
	return &out
}

func cloneRunnerApproval(in *RunnerApproval) *RunnerApproval {
	if in == nil {
		return nil
	}
	out := *in
	if in.ResolvedAt != nil {
		t := *in.ResolvedAt
		out.ResolvedAt = &t
	}
	return &out
}

func copyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = copyAny(value)
	}
	return out
}

func copyAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return copyMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = copyAny(typed[i])
		}
		return out
	default:
		return typed
	}
}

func copyRunnerAdvisoryByRun(in map[string]RunnerAdvisoryState) map[string]RunnerAdvisoryState {
	out := make(map[string]RunnerAdvisoryState, len(in))
	for key, value := range in {
		out[key] = copyRunnerAdvisoryState(value)
	}
	return out
}
