package artifacts

import (
	"fmt"
	"strings"
)

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
		applyCheckpointRecord(&state, runID, *record.Checkpoint)
	case "result":
		if record.Result == nil {
			return fmt.Errorf("runner result journal record missing payload")
		}
		applyResultRecord(&state, runID, *record.Result)
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

func applyCheckpointRecord(state *RunnerAdvisoryState, runID string, checkpoint RunnerCheckpointAdvisory) {
	state.LastCheckpoint = cloneCheckpoint(&checkpoint)
	state.Lifecycle = checkpointLifecycleHint(checkpoint)
	applyStepHintFromCheckpoint(state, runID, checkpoint)
	applyGateHintFromCheckpoint(state, runID, checkpoint)
}

func applyResultRecord(state *RunnerAdvisoryState, runID string, result RunnerResultAdvisory) {
	state.LastResult = cloneResult(&result)
	state.Lifecycle = resultLifecycleHint(result)
	applyStepHintFromResult(state, runID, result)
	applyGateHintFromResult(state, runID, result)
}

func checkpointLifecycleHint(checkpoint RunnerCheckpointAdvisory) *RunnerLifecycleHint {
	return &RunnerLifecycleHint{
		LifecycleState: checkpoint.LifecycleState,
		OccurredAt:     checkpoint.OccurredAt.UTC(),
		StageID:        checkpoint.StageID,
		StepID:         checkpoint.StepID,
		RoleInstanceID: checkpoint.RoleInstanceID,
		StageAttemptID: checkpoint.StageAttemptID,
		StepAttemptID:  checkpoint.StepAttemptID,
		GateAttemptID:  checkpoint.GateAttemptID,
	}
}

func resultLifecycleHint(result RunnerResultAdvisory) *RunnerLifecycleHint {
	return &RunnerLifecycleHint{
		LifecycleState: result.LifecycleState,
		OccurredAt:     result.OccurredAt.UTC(),
		StageID:        result.StageID,
		StepID:         result.StepID,
		RoleInstanceID: result.RoleInstanceID,
		StageAttemptID: result.StageAttemptID,
		StepAttemptID:  result.StepAttemptID,
		GateAttemptID:  result.GateAttemptID,
	}
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
	applyStepHintCheckpointStatus(&hint, checkpoint)
	state.StepAttempts[stepAttemptID] = hint
}

func applyStepHintCheckpointStatus(hint *RunnerStepHint, checkpoint RunnerCheckpointAdvisory) {
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
