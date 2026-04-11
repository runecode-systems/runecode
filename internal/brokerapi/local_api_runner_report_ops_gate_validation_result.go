package brokerapi

import "fmt"

func validateGateResultFields(report RunnerResultReport) error {
	if err := validatePlanOrderIndex(report.PlanCheckpointCode, report.PlanOrderIndex); err != nil {
		return err
	}
	if err := validateGateEvidenceBindingFields(report); err != nil {
		return err
	}
	if !shouldValidateGateScopedResult(report) {
		return nil
	}
	if err := validateGateScopedResultCode(report.ResultCode); err != nil {
		return err
	}
	if err := validateGateResultIdentity(report); err != nil {
		return err
	}
	if err := validateGateResultStateAndInputs(report); err != nil {
		return err
	}
	return validateGateResultOverrideFields(report)
}

func validateGateResultPlanBinding(report RunnerResultReport, planned compiledRunGatePlan) error {
	if !shouldValidateGateScopedResult(report) {
		return validateNoPlanBindingWithoutGate(report.PlanCheckpointCode, report.PlanOrderIndex)
	}
	if !planned.hasEntries() {
		if report.PlanCheckpointCode != "" || report.PlanOrderIndex != 0 {
			return fmt.Errorf("gate-scoped result requires trusted run plan gate entries")
		}
		return nil
	}
	if report.PlanCheckpointCode == "" {
		return fmt.Errorf("gate-scoped result requires plan_checkpoint_code")
	}
	entry, ok := planned.entryFor(report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex)
	if !ok {
		return fmt.Errorf("gate-scoped result does not match trusted run plan placement")
	}
	return validatePlannedInputDigestHooks(entry.ExpectedInputDigests, report.NormalizedInputDigests)
}

func validateGateScopedResultCode(code string) error {
	switch code {
	case "gate_failed", "gate_passed", "gate_overridden", "gate_superseded":
		return nil
	default:
		return fmt.Errorf("gate-scoped result requires gate_* result_code, got %q", code)
	}
}

func validateGateResultIdentity(report RunnerResultReport) error {
	if report.GateID == "" || report.GateKind == "" || report.GateVersion == "" || report.GateAttemptID == "" || report.GateLifecycleState == "" {
		return fmt.Errorf("gate-scoped result requires gate_id, gate_kind, gate_version, gate_attempt_id, and gate_lifecycle_state")
	}
	if !isGateKind(report.GateKind) {
		return fmt.Errorf("unsupported gate_kind %q", report.GateKind)
	}
	return nil
}

func validateGateResultStateAndInputs(report RunnerResultReport) error {
	if !isGateResultState(report.GateLifecycleState) {
		return fmt.Errorf("unsupported gate_lifecycle_state %q for result", report.GateLifecycleState)
	}
	if expected, ok := gateStateForResultCode(report.ResultCode); ok && expected != report.GateLifecycleState {
		return fmt.Errorf("result_code %q requires gate_lifecycle_state %q, got %q", report.ResultCode, expected, report.GateLifecycleState)
	}
	if err := validateNormalizedInputDigests(report.NormalizedInputDigests); err != nil {
		return err
	}
	if report.ResultCode == "gate_failed" && report.LifecycleState != "failed" {
		return fmt.Errorf("gate_failed result requires lifecycle_state failed")
	}
	return nil
}

func validateGateResultOverrideFields(report RunnerResultReport) error {
	if report.GateLifecycleState == "overridden" && report.OverriddenFailedResultRef == "" {
		return fmt.Errorf("overridden gate result requires overridden_failed_result_ref")
	}
	if report.OverriddenFailedResultRef != "" && !isValidDigestIdentity(report.OverriddenFailedResultRef) {
		return fmt.Errorf("overridden_failed_result_ref must be digest identity")
	}
	return nil
}
