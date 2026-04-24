package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) bridgeSessionExecutionTriggerToRun(runID string, result artifacts.SessionExecutionTriggerAppendResult) error {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return nil
	}
	if _, err := s.RecordRunnerCheckpoint(trimmedRunID, sessionExecutionBridgeCheckpoint(result)); err != nil {
		return err
	}
	return s.SetRunStatus(trimmedRunID, "active")
}

func sessionExecutionBridgeCheckpoint(result artifacts.SessionExecutionTriggerAppendResult) artifacts.RunnerCheckpointAdvisory {
	return artifacts.RunnerCheckpointAdvisory{
		LifecycleState: "active",
		CheckpointCode: "run_started",
		OccurredAt:     result.Trigger.CreatedAt.UTC(),
		IdempotencyKey: "session-trigger-" + result.Trigger.TriggerID,
		Details: map[string]any{
			"session_id":          result.Trigger.SessionID,
			"trigger_id":          result.Trigger.TriggerID,
			"turn_id":             result.TurnExecution.TurnID,
			"trigger_source":      result.Trigger.TriggerSource,
			"requested_operation": result.Trigger.RequestedOperation,
			"approval_profile":    result.TurnExecution.ApprovalProfile,
			"autonomy_posture":    result.TurnExecution.AutonomyPosture,
		},
	}
}
