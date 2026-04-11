package brokerapi

import "github.com/runecode-ai/runecode/internal/artifacts"

func (s *Service) validateRunnerCheckpointReport(current string, found bool, runID string, report RunnerCheckpointReport) (map[string]any, error) {
	planned, err := s.compileRunGatePlan(runID)
	if err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	runnerAdvisory, _ := s.RunnerAdvisory(runID)
	if err := validateRunnerCheckpointTransition(current, found, report.LifecycleState); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateRunnerCheckpointCode(report.CheckpointCode); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateGateCheckpointFields(report); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateGateCheckpointPlanBinding(report, planned); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateGateAttemptRetryPosture(runnerAdvisory, report.GateAttemptID, report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex, planned); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateCheckpointGateAttemptMutation(runnerAdvisory, report); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	if err := validateRunnerCheckpointPhaseTransition(runnerAdvisory, report); err != nil {
		return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	details, err := sanitizeRunnerDetails(report.Details)
	if err != nil {
		return nil, runnerValidationError{code: "broker_validation_schema_invalid", msg: err.Error()}
	}
	return details, nil
}

func (s *Service) prepareRunnerResultBindings(current string, found bool, runID string, report RunnerResultReport) (runnerResultBindings, error) {
	planned, err := s.compileRunGatePlan(runID)
	if err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
	}
	runnerAdvisory, _ := s.RunnerAdvisory(runID)
	if err := validateRunnerResultReportCore(current, found, report, planned, runnerAdvisory); err != nil {
		return runnerResultBindings{}, validationTransitionErr(err)
	}
	details, err := sanitizeRunnerDetails(report.Details)
	if err != nil {
		return runnerResultBindings{}, runnerValidationError{code: "broker_validation_schema_invalid", msg: err.Error()}
	}
	overrideActionHash, overridePolicyRef, gateEvidenceRef, gateResultRef, err := s.resolveRunnerResultRefs(runID, report, details, planned)
	if err != nil {
		return runnerResultBindings{}, validationTransitionErr(err)
	}
	return runnerResultBindings{
		details:            details,
		overrideActionHash: overrideActionHash,
		overridePolicyRef:  overridePolicyRef,
		gateEvidenceRef:    gateEvidenceRef,
		gateResultRef:      gateResultRef,
	}, nil
}

func validateRunnerResultReportCore(current string, found bool, report RunnerResultReport, planned compiledRunGatePlan, advisory artifacts.RunnerAdvisoryState) error {
	if err := validateRunnerResultTransition(current, found, report.LifecycleState); err != nil {
		return err
	}
	if err := validateRunnerResultCode(report.ResultCode); err != nil {
		return err
	}
	if err := validateGateResultFields(report); err != nil {
		return err
	}
	if err := validateGateResultPlanBinding(report, planned); err != nil {
		return err
	}
	if err := validateGateAttemptRetryPosture(advisory, report.GateAttemptID, report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex, planned); err != nil {
		return err
	}
	if err := validateResultGateAttemptMutation(advisory, report); err != nil {
		return err
	}
	return validateOverrideReferenceAgainstHistory(advisory, report)
}

func (s *Service) resolveRunnerResultRefs(runID string, report RunnerResultReport, details map[string]any, planned compiledRunGatePlan) (string, string, string, string, error) {
	overrideActionHash, overridePolicyRef, err := s.resolveOverrideApprovalBindings(runID, report, details)
	if err != nil {
		return "", "", "", "", err
	}
	plannedEntry, _ := planned.entryFor(report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex)
	gateEvidenceRef, err := s.resolveGateEvidenceRef(runID, report, plannedEntry)
	if err != nil {
		return "", "", "", "", err
	}
	gateResultRef, err := canonicalGateResultRef(runID, report, gateEvidenceRef)
	if err != nil {
		return "", "", "", "", err
	}
	return overrideActionHash, overridePolicyRef, gateEvidenceRef, gateResultRef, nil
}

func validationTransitionErr(err error) error {
	return runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: err.Error()}
}
