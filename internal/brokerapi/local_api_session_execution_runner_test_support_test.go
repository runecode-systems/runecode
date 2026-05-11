package brokerapi

import (
	"context"
	"time"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

func launchSessionExecutionRunnerInProcessForTests(ctx context.Context, s *Service, spec sessionExecutionRunnerLaunchSpec) error {
	return launchSessionExecutionRunnerCheckpointOnlyInProcessForTests(ctx, s, spec)
}

func launchSessionExecutionRunnerCheckpointOnlyInProcessForTests(ctx context.Context, s *Service, spec sessionExecutionRunnerLaunchSpec) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	entry, err := activeSessionExecutionPlanEntryForTests(s, spec.runID)
	if err != nil {
		return err
	}
	return reportSessionExecutionCheckpointForTests(ctx, s, spec, entry)
}

func launchSessionExecutionRunnerCompleteInProcessForTests(ctx context.Context, s *Service, spec sessionExecutionRunnerLaunchSpec) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	entry, err := activeSessionExecutionPlanEntryForTests(s, spec.runID)
	if err != nil {
		return err
	}
	if err := reportSessionExecutionCheckpointForTests(ctx, s, spec, entry); err != nil {
		return err
	}
	return reportSessionExecutionResultForTests(ctx, s, spec, entry)
}

func activeSessionExecutionPlanEntryForTests(s *Service, runID string) (artifacts.RunPlanGateEntryRecord, error) {
	authority, ok, err := s.ActiveRunPlanAuthority(runID)
	if err != nil {
		return artifacts.RunPlanGateEntryRecord{}, err
	}
	if !ok {
		return artifacts.RunPlanGateEntryRecord{}, runnerBridgeError("checkpoint", "trusted run plan authority missing")
	}
	entry, err := selectSessionExecutionPlanEntry(authority.Entries)
	if err != nil {
		return artifacts.RunPlanGateEntryRecord{}, err
	}
	return entry, nil
}

func reportSessionExecutionCheckpointForTests(ctx context.Context, s *Service, spec sessionExecutionRunnerLaunchSpec, entry artifacts.RunPlanGateEntryRecord) error {
	now := time.Now().UTC()
	checkpoint := RunnerCheckpointReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerCheckpointReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     spec.requestID + ":test-checkpoint",
		RunID:         spec.runID,
		Report: RunnerCheckpointReport{
			SchemaID:               "runecode.protocol.v0.RunnerCheckpointReport",
			SchemaVersion:          "0.1.0",
			LifecycleState:         "active",
			CheckpointCode:         "gate_started",
			OccurredAt:             now.Format(time.RFC3339),
			IdempotencyKey:         spec.planID + ":test-checkpoint",
			PlanCheckpointCode:     entry.PlanCheckpointCode,
			PlanOrderIndex:         entry.PlanOrderIndex,
			GateID:                 entry.GateID,
			GateKind:               entry.GateKind,
			GateVersion:            entry.GateVersion,
			GateLifecycleState:     "running",
			NormalizedInputDigests: append([]string{}, entry.ExpectedInputDigests...),
			StageID:                entry.StageID,
			StepID:                 entry.StepID,
			RoleInstanceID:         entry.RoleInstanceID,
			StageAttemptID:         sessionExecutionDerivedAttemptID("stage_attempt", spec.planID, 1),
			StepAttemptID:          sessionExecutionDerivedAttemptID("step_attempt", spec.planID, 1),
			GateAttemptID:          sessionExecutionDerivedAttemptID("gate_attempt", spec.planID, 1),
		},
	}
	if _, errResp := s.HandleRunnerCheckpointReport(ctx, checkpoint, RequestContext{}); errResp != nil {
		return runnerBridgeError("checkpoint", errResp.Error.Message)
	}
	return nil
}

func reportSessionExecutionResultForTests(ctx context.Context, s *Service, spec sessionExecutionRunnerLaunchSpec, entry artifacts.RunPlanGateEntryRecord) error {
	now := time.Now().UTC()
	result := RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     spec.requestID + ":test-result",
		RunID:         spec.runID,
		Report: RunnerResultReport{
			SchemaID:               "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:          "0.1.0",
			LifecycleState:         "completed",
			ResultCode:             "gate_passed",
			OccurredAt:             now.Add(time.Second).Format(time.RFC3339),
			IdempotencyKey:         spec.planID + ":test-result",
			PlanCheckpointCode:     entry.PlanCheckpointCode,
			PlanOrderIndex:         entry.PlanOrderIndex,
			GateID:                 entry.GateID,
			GateKind:               entry.GateKind,
			GateVersion:            entry.GateVersion,
			GateLifecycleState:     "passed",
			NormalizedInputDigests: append([]string{}, entry.ExpectedInputDigests...),
			StageID:                entry.StageID,
			StepID:                 entry.StepID,
			RoleInstanceID:         entry.RoleInstanceID,
			StageAttemptID:         sessionExecutionDerivedAttemptID("stage_attempt", spec.planID, 1),
			StepAttemptID:          sessionExecutionDerivedAttemptID("step_attempt", spec.planID, 1),
			GateAttemptID:          sessionExecutionDerivedAttemptID("gate_attempt", spec.planID, 1),
		},
	}
	if _, errResp := s.HandleRunnerResultReport(ctx, result, RequestContext{}); errResp != nil {
		return runnerBridgeError("result", errResp.Error.Message)
	}
	return nil
}

func runnerBridgeError(kind, message string) error {
	return &bridgeFailure{kind: kind, message: message}
}

type bridgeFailure struct {
	kind    string
	message string
}

func (e *bridgeFailure) Error() string {
	return "runner " + e.kind + " report rejected: " + e.message
}
