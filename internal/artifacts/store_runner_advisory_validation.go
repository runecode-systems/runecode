package artifacts

import (
	"fmt"
	"strings"
)

func validateRunnerStepIdentity(stageID, stepID, roleInstanceID string) error {
	trimmedStageID := strings.TrimSpace(stageID)
	trimmedStepID := strings.TrimSpace(stepID)
	trimmedRoleInstanceID := strings.TrimSpace(roleInstanceID)
	if trimmedStageID == "" && trimmedStepID != "" {
		return fmt.Errorf("stage id is required when step id is set")
	}
	if trimmedRoleInstanceID == "" {
		return nil
	}
	if trimmedStageID == "" || trimmedStepID == "" {
		return fmt.Errorf("stage id and step id are required when role instance id is set")
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
