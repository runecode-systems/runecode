package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) validateRunnerCheckpointReport(current string, found bool, runID string, report RunnerCheckpointReport) (map[string]any, error) {
	planned, entry, err := s.resolveCheckpointPlanBinding(runID, report)
	if err != nil {
		return nil, err
	}
	runnerAdvisory, _ := s.RunnerAdvisory(runID)
	if err := validateRunnerCheckpointReportCore(current, found, report, planned, runnerAdvisory); err != nil {
		return nil, validationTransitionErr(err)
	}
	return runnerCheckpointDetailsWithProjectContext(report.Details, entry, planned)
}

func (s *Service) resolveCheckpointPlanBinding(runID string, report RunnerCheckpointReport) (compiledRunGatePlan, runPlannedGateEntry, error) {
	planned, err := s.compileRunGatePlan(runID)
	if err != nil {
		return compiledRunGatePlan{}, runPlannedGateEntry{}, validationTransitionErr(err)
	}
	entry, err := validateGateCheckpointPlanBinding(report, planned)
	if err != nil {
		return compiledRunGatePlan{}, runPlannedGateEntry{}, validationTransitionErr(err)
	}
	return planned, entry, nil
}

func validateRunnerCheckpointReportCore(current string, found bool, report RunnerCheckpointReport, planned compiledRunGatePlan, advisory artifacts.RunnerAdvisoryState) error {
	if err := validateRunnerCheckpointTransition(current, found, report.LifecycleState); err != nil {
		return err
	}
	if err := validateRunnerCheckpointCode(report.CheckpointCode); err != nil {
		return err
	}
	if err := validateGateCheckpointFields(report); err != nil {
		return err
	}
	if err := validateGateAttemptRetryPosture(advisory, report.GateAttemptID, report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex, planned); err != nil {
		return err
	}
	if err := validateCheckpointGateAttemptMutation(advisory, report); err != nil {
		return err
	}
	return validateRunnerCheckpointPhaseTransition(advisory, report)
}

func runPlanDetailsMergeContext(rawDetails map[string]any, entry runPlannedGateEntry, planned compiledRunGatePlan) (map[string]any, error) {
	details, err := sanitizeRunnerDetails(rawDetails)
	if err != nil {
		return nil, runnerValidationError{code: "broker_validation_schema_invalid", msg: err.Error()}
	}
	if details == nil {
		details = map[string]any{}
	}
	if entry.ProjectContextID == "" {
		entry.ProjectContextID = planned.projectContextID
	}
	if entry.ProjectContextID != "" {
		details["project_context_identity_digest"] = entry.ProjectContextID
	}
	if planned.planID != "" {
		details["run_plan_id"] = planned.planID
	}
	if planned.runPlanRef != "" {
		details["run_plan_ref"] = planned.runPlanRef
	}
	details, err = runPlanDetailsMergeHashes(details, planned)
	if err != nil {
		return nil, err
	}
	if len(entry.DependencyCacheHandoffs) > 0 {
		details["dependency_cache_handoffs"] = dependencyCacheHandoffDetails(entry.DependencyCacheHandoffs)
	}
	return details, nil
}

func runPlanDetailsMergeHashes(details map[string]any, planned compiledRunGatePlan) (map[string]any, error) {
	var err error
	details, err = mergeWorkflowDefinitionHash(details, planned)
	if err != nil {
		return nil, err
	}
	details, err = mergeProcessDefinitionHash(details, planned)
	if err != nil {
		return nil, err
	}
	details, err = mergePolicyContextHash(details, planned)
	if err != nil {
		return nil, err
	}
	return details, nil
}

func mergeWorkflowDefinitionHash(details map[string]any, planned compiledRunGatePlan) (map[string]any, error) {
	if planned.workflowDefinitionHash != "" {
		if existing, _ := details["workflow_definition_hash"].(string); existing != "" && existing != planned.workflowDefinitionHash {
			return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: "details.workflow_definition_hash must match trusted run plan binding"}
		}
		details["workflow_definition_hash"] = planned.workflowDefinitionHash
	}
	return details, nil
}

func mergeProcessDefinitionHash(details map[string]any, planned compiledRunGatePlan) (map[string]any, error) {
	if planned.processDefinitionHash != "" {
		if existing, _ := details["process_definition_hash"].(string); existing != "" && existing != planned.processDefinitionHash {
			return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: "details.process_definition_hash must match trusted run plan binding"}
		}
		details["process_definition_hash"] = planned.processDefinitionHash
	}
	return details, nil
}

func mergePolicyContextHash(details map[string]any, planned compiledRunGatePlan) (map[string]any, error) {
	if planned.policyContextHash != "" {
		if existing, _ := details["policy_context_hash"].(string); existing != "" && existing != planned.policyContextHash {
			return nil, runnerValidationError{code: "broker_validation_runner_transition_invalid", msg: "details.policy_context_hash must match trusted run plan binding"}
		}
		details["policy_context_hash"] = planned.policyContextHash
	}
	return details, nil
}

func runnerCheckpointDetailsWithProjectContext(rawDetails map[string]any, entry runPlannedGateEntry, planned compiledRunGatePlan) (map[string]any, error) {
	details, err := runPlanDetailsMergeContext(rawDetails, entry, planned)
	if err != nil {
		return nil, err
	}
	if details == nil {
		details = map[string]any{}
	}
	return details, nil
}

func runnerResultDetailsWithProjectContext(rawDetails map[string]any, entry runPlannedGateEntry, planned compiledRunGatePlan, gateEvidence *GateEvidence) (map[string]any, error) {
	details, err := runPlanDetailsMergeContext(rawDetails, entry, planned)
	if err != nil {
		return nil, err
	}
	if details == nil {
		details = map[string]any{}
	}
	if gateEvidence != nil && entry.ProjectContextID != "" && gateEvidence.ProjectContextID == "" {
		gateEvidence.ProjectContextID = entry.ProjectContextID
	}
	return details, nil
}

func dependencyCacheHandoffDetails(handoffs []runPlannedDependencyCacheHandoff) []map[string]any {
	if len(handoffs) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(handoffs))
	for _, handoff := range handoffs {
		out = append(out, map[string]any{
			"request_digest": digestObjectForIdentity(handoff.RequestDigest),
			"consumer_role":  handoff.ConsumerRole,
			"required":       handoff.Required,
		})
	}
	return out
}

func (s *Service) prepareRunnerResultBindings(current string, found bool, runID string, report RunnerResultReport) (runnerResultBindings, error) {
	planned, entry, err := s.resolveResultPlanBinding(runID, report)
	if err != nil {
		return runnerResultBindings{}, err
	}
	runnerAdvisory, _ := s.RunnerAdvisory(runID)
	if err := validateRunnerResultReportCore(current, found, report, planned, runnerAdvisory, entry); err != nil {
		return runnerResultBindings{}, validationTransitionErr(err)
	}
	details, err := runnerResultDetailsWithProjectContext(report.Details, entry, planned, report.GateEvidence)
	if err != nil {
		return runnerResultBindings{}, err
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

func (s *Service) resolveResultPlanBinding(runID string, report RunnerResultReport) (compiledRunGatePlan, runPlannedGateEntry, error) {
	planned, err := s.compileRunGatePlan(runID)
	if err != nil {
		return compiledRunGatePlan{}, runPlannedGateEntry{}, validationTransitionErr(err)
	}
	entry, err := validateGateResultPlanBinding(report, planned)
	if err != nil {
		return compiledRunGatePlan{}, runPlannedGateEntry{}, validationTransitionErr(err)
	}
	return planned, entry, nil
}

func validateRunnerResultReportCore(current string, found bool, report RunnerResultReport, planned compiledRunGatePlan, advisory artifacts.RunnerAdvisoryState, planEntry runPlannedGateEntry) error {
	if err := validateRunnerResultTransition(current, found, report.LifecycleState); err != nil {
		return err
	}
	if err := validateRunnerResultCode(report.ResultCode); err != nil {
		return err
	}
	if err := validateGateResultFields(report); err != nil {
		return err
	}
	_ = planned
	_ = planEntry
	if err := validateGateAttemptRetryPosture(advisory, report.GateAttemptID, report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex, planned); err != nil {
		return err
	}
	if err := validateResultGateAttemptMutation(advisory, report); err != nil {
		return err
	}
	return validateOverrideReferenceAgainstHistory(advisory, report)
}

func (s *Service) resolveRunnerResultRefs(runID string, report RunnerResultReport, details map[string]any, planned compiledRunGatePlan) (string, string, string, string, error) {
	if report.GateLifecycleState == "overridden" && planned.policyContextHash != "" {
		boundPolicyContext, _ := details["policy_context_hash"].(string)
		if boundPolicyContext != planned.policyContextHash {
			return "", "", "", "", fmt.Errorf("gate override details.policy_context_hash must match trusted run plan policy_context_hash")
		}
	}
	overrideActionHash, overridePolicyRef, err := s.resolveOverrideApprovalBindings(runID, report, details)
	if err != nil {
		return "", "", "", "", err
	}
	plannedEntry, _ := planned.entryFor(report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex)
	if plannedEntry.ProjectContextID == "" {
		plannedEntry.ProjectContextID = planned.projectContextID
	}
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
