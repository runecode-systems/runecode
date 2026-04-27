package brokerapi

import (
	"fmt"
	"strings"
)

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

func validateOptionalReportedScopeAgainstPlanned(stageID, stepID, roleInstanceID string, planned runPlannedGateEntry) error {
	plannedStageID := strings.TrimSpace(planned.StageID)
	plannedStepID := strings.TrimSpace(planned.StepID)
	plannedRoleInstanceID := strings.TrimSpace(planned.RoleInstanceID)
	if trimmed := strings.TrimSpace(stageID); trimmed != "" && plannedStageID != "" && trimmed != plannedStageID {
		return fmt.Errorf("gate-scoped report stage_id %q does not match trusted run plan stage_id %q", trimmed, strings.TrimSpace(planned.StageID))
	}
	if trimmed := strings.TrimSpace(stepID); trimmed != "" && plannedStepID != "" && trimmed != plannedStepID {
		return fmt.Errorf("gate-scoped report step_id %q does not match trusted run plan step_id %q", trimmed, strings.TrimSpace(planned.StepID))
	}
	if trimmed := strings.TrimSpace(roleInstanceID); trimmed != "" && plannedRoleInstanceID != "" && trimmed != plannedRoleInstanceID {
		return fmt.Errorf("gate-scoped report role_instance_id %q does not match trusted run plan role_instance_id %q", trimmed, strings.TrimSpace(planned.RoleInstanceID))
	}
	return nil
}
