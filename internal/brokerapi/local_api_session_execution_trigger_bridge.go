package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) bridgeSessionExecutionTriggerToRun(requestID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) error {
	runID := strings.TrimSpace(authority.runID)
	if runID == "" {
		return nil
	}
	if err := s.reportSessionExecutionRunnerCheckpoint(requestID, runID, result, authority); err != nil {
		return err
	}
	if sessionExecutionCompletesWithImmediateRunnerResult(result, authority) {
		if err := s.reportSessionExecutionRunnerResult(requestID, runID, result, authority); err != nil {
			return err
		}
	}
	return nil
}

func sessionExecutionCompletesWithImmediateRunnerResult(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) bool {
	if strings.TrimSpace(authority.workflowOperation) == sessionWorkflowOperationDraftPromoteApply {
		return len(result.TurnExecution.WorkflowRouting.BoundInputArtifacts) > 0
	}
	if strings.TrimSpace(authority.workflowOperation) == sessionWorkflowOperationApprovedImplementation {
		return len(result.TurnExecution.WorkflowRouting.BoundInputArtifacts) > 0
	}
	return sessionExecutionCompletesAsDraftArtifact(authority.workflowOperation)
}

func sessionExecutionCompletesAsDraftArtifact(operation string) bool {
	switch strings.TrimSpace(operation) {
	case sessionWorkflowOperationChangeDraft, sessionWorkflowOperationSpecDraft:
		return true
	default:
		return false
	}
}

func (s *Service) reportSessionExecutionRunnerCheckpoint(requestID, runID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) error {
	report := RunnerCheckpointReport{
		SchemaID:               "runecode.protocol.v0.RunnerCheckpointReport",
		SchemaVersion:          "0.1.0",
		LifecycleState:         "active",
		CheckpointCode:         "gate_started",
		OccurredAt:             result.Trigger.CreatedAt.UTC().Format(time.RFC3339),
		IdempotencyKey:         "session-trigger-checkpoint-" + strings.TrimSpace(result.Trigger.TriggerID),
		PlanCheckpointCode:     authority.planCheckpointCode,
		PlanOrderIndex:         authority.planOrderIndex,
		GateID:                 authority.gateID,
		GateKind:               authority.gateKind,
		GateVersion:            authority.gateVersion,
		GateLifecycleState:     "running",
		StageID:                authority.stageID,
		StepID:                 authority.stepID,
		RoleInstanceID:         authority.roleInstanceID,
		StageAttemptID:         sessionExecutionAttemptID(result, "stage", 1),
		StepAttemptID:          sessionExecutionAttemptID(result, "step", 1),
		GateAttemptID:          sessionExecutionAttemptID(result, "gate", 1),
		NormalizedInputDigests: sessionExecutionNormalizedInputDigests(result, authority),
		Details:                sessionExecutionBridgeDetails(result, authority),
	}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerCheckpointReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID + "-runner-checkpoint",
		RunID:         runID,
		Report:        report,
	}, RequestContext{})
	if errResp != nil {
		return fmt.Errorf("runner checkpoint report rejected: %s", strings.TrimSpace(errResp.Error.Message))
	}
	return nil
}

func (s *Service) reportSessionExecutionRunnerResult(requestID, runID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) error {
	report := RunnerResultReport{
		SchemaID:               "runecode.protocol.v0.RunnerResultReport",
		SchemaVersion:          "0.1.0",
		LifecycleState:         "completed",
		ResultCode:             "gate_passed",
		OccurredAt:             result.Trigger.CreatedAt.UTC().Add(time.Second).Format(time.RFC3339),
		IdempotencyKey:         "session-trigger-result-" + strings.TrimSpace(result.Trigger.TriggerID),
		PlanCheckpointCode:     authority.planCheckpointCode,
		PlanOrderIndex:         authority.planOrderIndex,
		GateID:                 authority.gateID,
		GateKind:               authority.gateKind,
		GateVersion:            authority.gateVersion,
		GateLifecycleState:     "passed",
		StageID:                authority.stageID,
		StepID:                 authority.stepID,
		RoleInstanceID:         authority.roleInstanceID,
		StageAttemptID:         sessionExecutionAttemptID(result, "stage", 1),
		StepAttemptID:          sessionExecutionAttemptID(result, "step", 1),
		GateAttemptID:          sessionExecutionAttemptID(result, "gate", 1),
		NormalizedInputDigests: sessionExecutionNormalizedInputDigests(result, authority),
		GateEvidence:           sessionExecutionDraftGateEvidence(runID, result, authority),
		Details:                sessionExecutionBridgeDetails(result, authority),
	}
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID + "-runner-result",
		RunID:         runID,
		Report:        report,
	}, RequestContext{})
	if errResp != nil {
		return fmt.Errorf("runner result report rejected: %s", strings.TrimSpace(errResp.Error.Message))
	}
	return nil
}

func sessionExecutionDraftGateEvidence(runID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) *GateEvidence {
	outputDigests := sessionExecutionRunnerOutputArtifactDigests(result)
	return &GateEvidence{
		SchemaID:               "runecode.protocol.v0.GateEvidence",
		SchemaVersion:          "0.1.0",
		GateID:                 authority.gateID,
		GateKind:               authority.gateKind,
		GateVersion:            authority.gateVersion,
		ProjectContextID:       strings.TrimSpace(authority.projectContextIdentityDigest),
		PlanCheckpointCode:     authority.planCheckpointCode,
		PlanOrderIndex:         authority.planOrderIndex,
		RunID:                  strings.TrimSpace(runID),
		StageID:                authority.stageID,
		StepID:                 authority.stepID,
		RoleInstanceID:         authority.roleInstanceID,
		GateAttemptID:          sessionExecutionAttemptID(result, "gate", 1),
		StartedAt:              result.Trigger.CreatedAt.UTC().Format(time.RFC3339),
		FinishedAt:             result.Trigger.CreatedAt.UTC().Add(time.Second).Format(time.RFC3339),
		NormalizedInputDigests: sessionExecutionNormalizedInputDigests(result, authority),
		Runtime: map[string]any{
			"workflow_operation":              strings.TrimSpace(authority.workflowOperation),
			"workflow_definition_hash":        strings.TrimSpace(authority.workflowDefinitionHash),
			"process_definition_hash":         strings.TrimSpace(authority.processDefinitionHash),
			"project_context_identity_digest": strings.TrimSpace(authority.projectContextIdentityDigest),
			"turn_id":                         strings.TrimSpace(result.TurnExecution.TurnID),
		},
		Outcome: map[string]any{
			"result_code":           "gate_passed",
			"output_artifact_count": len(outputDigests),
		},
		OutputArtifactDigests: outputDigests,
	}
}

func sessionExecutionRunnerOutputArtifactDigests(result artifacts.SessionExecutionTriggerAppendResult) []string {
	return sessionExecutionDraftGateEvidenceDigests(result)
}

func sessionExecutionBridgeDetails(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) map[string]any {
	return map[string]any{
		"session_id":                      strings.TrimSpace(result.Trigger.SessionID),
		"trigger_id":                      strings.TrimSpace(result.Trigger.TriggerID),
		"turn_id":                         strings.TrimSpace(result.TurnExecution.TurnID),
		"workflow_operation":              strings.TrimSpace(authority.workflowOperation),
		"workflow_definition_hash":        strings.TrimSpace(authority.workflowDefinitionHash),
		"process_definition_hash":         strings.TrimSpace(authority.processDefinitionHash),
		"project_context_identity_digest": strings.TrimSpace(authority.projectContextIdentityDigest),
	}
}

func sessionExecutionAttemptID(result artifacts.SessionExecutionTriggerAppendResult, kind string, attempt int) string {
	if attempt < 1 {
		attempt = 1
	}
	return fmt.Sprintf("%s-%s-attempt-%d", strings.TrimSpace(result.Trigger.TriggerID), kind, attempt)
}

func sessionExecutionNormalizedInputDigest(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) string {
	payload := strings.TrimSpace(result.Trigger.TriggerID) + "\n" + strings.TrimSpace(result.TurnExecution.TurnID) + "\n" + strings.TrimSpace(authority.planID)
	return shaDigestIdentity(payload)
}

func sessionExecutionNormalizedInputDigests(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) []string {
	expected := strings.TrimSpace(authority.expectedInputDigest)
	if expected != "" {
		return []string{expected}
	}
	return []string{sessionExecutionNormalizedInputDigest(result, authority)}
}
