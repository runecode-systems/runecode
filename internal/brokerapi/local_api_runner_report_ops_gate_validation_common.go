package brokerapi

import "fmt"

func shouldValidateGateScopedResult(report RunnerResultReport) bool {
	return hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) || report.OverriddenFailedResultRef != ""
}

func validatePlanOrderIndex(planCheckpointCode string, planOrderIndex int) error {
	if planCheckpointCode != "" && planOrderIndex < 0 {
		return fmt.Errorf("plan_order_index must be >= 0 when plan_checkpoint_code is set")
	}
	return nil
}

func validateNoPlanBindingWithoutGate(planCheckpointCode string, planOrderIndex int) error {
	if planCheckpointCode != "" {
		return fmt.Errorf("plan_checkpoint_code requires gate identity binding")
	}
	if planOrderIndex != 0 {
		return fmt.Errorf("plan_order_index requires gate identity binding")
	}
	return nil
}

func validateOptionalGateEvidenceRef(gateEvidenceRef string, hasBinding bool) error {
	if gateEvidenceRef == "" {
		return nil
	}
	if !isValidDigestIdentity(gateEvidenceRef) {
		return fmt.Errorf("gate_evidence_ref must be digest identity")
	}
	if !hasBinding {
		return fmt.Errorf("gate_evidence_ref requires gate identity binding")
	}
	return nil
}
