package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func validateRunnerCheckpointPhaseTransition(advisory artifacts.RunnerAdvisoryState, report RunnerCheckpointReport) error {
	if report.StepAttemptID == "" {
		return nil
	}
	nextPhase, ok := phaseForCheckpointCode(report.CheckpointCode)
	if !ok {
		return nil
	}
	current, hasCurrent := advisory.StepAttempts[report.StepAttemptID]
	if !hasCurrent || current.CurrentPhase == "" {
		if nextPhase != "propose" && nextPhase != "validate" && nextPhase != "authorize" {
			return fmt.Errorf("step_attempt %q phase transition <none> -> %s is invalid", report.StepAttemptID, nextPhase)
		}
		return nil
	}
	if !isAllowedExecutionPhaseTransition(current.CurrentPhase, nextPhase) {
		return fmt.Errorf("step_attempt %q phase transition %s -> %s is invalid", report.StepAttemptID, current.CurrentPhase, nextPhase)
	}
	return nil
}

func phaseForCheckpointCode(code string) (string, bool) {
	switch code {
	case "step_attempt_started", "action_request_issued":
		return "propose", true
	case "step_validation_started", "step_validation_finished", "gate_attempt_started", "gate_attempt_finished":
		return "validate", true
	case "approval_wait_entered", "approval_wait_cleared":
		return "authorize", true
	case "step_execution_started", "step_execution_finished":
		return "execute", true
	case "step_attest_started", "step_attest_finished", "step_attempt_finished":
		return "attest", true
	default:
		return "", false
	}
}

func isAllowedExecutionPhaseTransition(current, next string) bool {
	if current == next {
		return true
	}
	order := map[string]int{"propose": 0, "validate": 1, "authorize": 2, "execute": 3, "attest": 4}
	currentOrder, okCurrent := order[current]
	nextOrder, okNext := order[next]
	if !okCurrent || !okNext {
		return false
	}
	if nextOrder == currentOrder+1 {
		return true
	}
	return current == "validate" && next == "execute"
}

func sanitizeRunnerDetails(details map[string]any) (map[string]any, error) {
	if len(details) == 0 {
		return nil, nil
	}
	state := &runnerDetailsValidationState{}
	if err := validateRunnerDetailsObject(details, 0, state); err != nil {
		return nil, err
	}
	return cloneRunnerDetailsMap(details), nil
}

type runnerDetailsValidationState struct {
	seen int
}

func validateRunnerDetailsObject(details map[string]any, depth int, state *runnerDetailsValidationState) error {
	if depth > runnerDetailsMaxDepth {
		return fmt.Errorf("report.details exceeds max nesting depth")
	}
	if len(details) > runnerDetailsMaxEntries {
		return fmt.Errorf("report.details object exceeds max keys")
	}
	for key, value := range details {
		if err := validateRunnerDetailsKey(key, state); err != nil {
			return err
		}
		if err := validateRunnerDetailsValue(value, depth+1, state); err != nil {
			return err
		}
	}
	return nil
}

func validateRunnerDetailsValue(value any, depth int, state *runnerDetailsValidationState) error {
	if depth > runnerDetailsMaxDepth {
		return fmt.Errorf("report.details exceeds max nesting depth")
	}
	switch typed := value.(type) {
	case nil, bool, float64, int, int64:
		return nil
	case string:
		if len(typed) > runnerDetailsMaxStrLen {
			return fmt.Errorf("report.details string value exceeds max length")
		}
		return nil
	case map[string]any:
		return validateRunnerDetailsObject(typed, depth, state)
	case []any:
		return validateRunnerDetailsArray(typed, depth, state)
	default:
		return fmt.Errorf("report.details contains unsupported value type %T", value)
	}
}

func validateRunnerDetailsArray(items []any, depth int, state *runnerDetailsValidationState) error {
	if len(items) > runnerDetailsMaxArrayLen {
		return fmt.Errorf("report.details array exceeds max length")
	}
	for _, item := range items {
		if err := validateRunnerDetailsValue(item, depth+1, state); err != nil {
			return err
		}
	}
	return nil
}

func validateRunnerDetailsKey(key string, state *runnerDetailsValidationState) error {
	state.seen++
	if state.seen > runnerDetailsMaxEntries {
		return fmt.Errorf("report.details exceeds max total entries")
	}
	if key == "" {
		return fmt.Errorf("report.details contains empty key")
	}
	if len(key) > 128 {
		return fmt.Errorf("report.details key exceeds max length")
	}
	return nil
}

func cloneRunnerDetailsMap(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}
	out := make(map[string]any, len(details))
	for key, value := range details {
		out[key] = cloneRunnerDetailsValue(value)
	}
	return out
}

func cloneRunnerDetailsValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneRunnerDetailsMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneRunnerDetailsValue(typed[i])
		}
		return out
	default:
		return typed
	}
}
