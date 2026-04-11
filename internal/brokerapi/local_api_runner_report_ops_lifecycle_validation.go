package brokerapi

import (
	"fmt"
	"strings"
)

func validateRunnerCheckpointTransition(current string, found bool, next string) error {
	if !isCheckpointLifecycle(next) {
		return fmt.Errorf("checkpoint lifecycle %q is invalid", next)
	}
	if !found {
		if next != "pending" {
			return fmt.Errorf("checkpoint transition for unknown run is invalid: %q", next)
		}
		return nil
	}
	if isTerminalLifecycle(current) {
		return fmt.Errorf("checkpoint transition %q -> %q is invalid", current, next)
	}
	if !isAllowedLifecycleTransition(current, next) {
		return fmt.Errorf("checkpoint transition %q -> %q is invalid", current, next)
	}
	return nil
}

func validateRunnerResultTransition(current string, found bool, next string) error {
	if !isTerminalLifecycle(next) {
		return fmt.Errorf("result lifecycle %q is invalid", next)
	}
	if !found {
		return fmt.Errorf("result transition for unknown run is invalid: %q", next)
	}
	if isTerminalLifecycle(current) && current != next {
		return fmt.Errorf("result transition %q -> %q is invalid", current, next)
	}
	if !isTerminalLifecycle(current) && !isAllowedLifecycleTransition(current, next) {
		return fmt.Errorf("result transition %q -> %q is invalid", current, next)
	}
	return nil
}

func isCheckpointLifecycle(state string) bool {
	switch state {
	case "pending", "starting", "active", "blocked", "recovering":
		return true
	default:
		return false
	}
}

func isTerminalLifecycle(state string) bool {
	switch state {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func isAllowedLifecycleTransition(current, next string) bool {
	if current == next {
		return true
	}
	switch current {
	case "pending":
		return next == "starting" || next == "active" || next == "blocked" || next == "recovering" || isTerminalLifecycle(next)
	case "starting":
		return next == "active" || next == "blocked" || next == "recovering" || isTerminalLifecycle(next)
	case "active":
		return next == "blocked" || next == "recovering" || isTerminalLifecycle(next)
	case "blocked":
		return next == "active" || next == "recovering" || isTerminalLifecycle(next)
	case "recovering":
		return next == "starting" || next == "active" || next == "blocked" || isTerminalLifecycle(next)
	default:
		return false
	}
}

func mapLifecycleToStoreStatus(state string) string {
	if state == "completed" {
		return "closed"
	}
	return state
}

func validateRunnerCheckpointCode(code string) error {
	switch strings.TrimSpace(code) {
	case "run_started", "stage_entered", "step_attempt_started", "action_request_issued",
		"step_validation_started", "step_validation_finished", "approval_wait_entered", "approval_wait_cleared",
		"gate_attempt_started", "gate_attempt_finished",
		"gate_planned", "gate_started", "gate_passed", "gate_failed", "gate_overridden", "gate_superseded",
		"step_execution_started", "step_execution_finished",
		"step_attest_started", "step_attest_finished", "step_attempt_finished", "run_terminal":
		return nil
	default:
		return fmt.Errorf("unsupported checkpoint code %q", strings.TrimSpace(code))
	}
}

func validateRunnerResultCode(code string) error {
	switch strings.TrimSpace(code) {
	case "run_completed", "run_failed", "run_cancelled", "step_failed", "gate_failed", "gate_passed", "gate_overridden", "gate_superseded":
		return nil
	default:
		return fmt.Errorf("unsupported result code %q", strings.TrimSpace(code))
	}
}
