package main

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func chatRunIDFromExecution(exec brokerapi.SessionTurnExecution) string {
	if runID := strings.TrimSpace(exec.PrimaryRunID); runID != "" {
		return runID
	}
	if len(exec.LinkedRunIDs) > 0 {
		return strings.TrimSpace(exec.LinkedRunIDs[0])
	}
	return ""
}

func humanizeExecutionToken(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
	if value == "" {
		return "n/a"
	}
	return value
}

func maxInt(left, right int) int {
	if right > left {
		return right
	}
	return left
}

func countNoun(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", strings.TrimSpace(singular))
	}
	return fmt.Sprintf("%d %s", count, strings.TrimSpace(plural))
}

func executionWaiting(exec brokerapi.SessionTurnExecution, run *brokerapi.RunDetail) bool {
	if chatExecutionTerminal(exec) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "waiting") {
		return true
	}
	if strings.TrimSpace(exec.WaitState) != "" || strings.TrimSpace(exec.WaitKind) != "" {
		return true
	}
	return run != nil && run.Coordination.Blocked
}

func runPlanAuthorityAvailable(run *brokerapi.RunDetail) bool {
	if run == nil {
		return false
	}
	if strings.TrimSpace(run.Summary.WorkflowDefinitionHash) != "" {
		return true
	}
	if strings.TrimSpace(authoritativeString(run.AuthoritativeState, "workflow_projection_reason")) == "plan_authoritative" {
		return true
	}
	return false
}

func runRunnerActive(run *brokerapi.RunDetail) bool {
	if run == nil {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(authoritativeString(run.AdvisoryState, "runner")))
	return value == "active" || value == "running"
}

func runCheckpointCode(run *brokerapi.RunDetail) string {
	if run == nil {
		return ""
	}
	checkpoint, _ := run.AdvisoryState["last_checkpoint"].(map[string]any)
	return authoritativeString(checkpoint, "checkpoint_code")
}

func runArtifactCount(run *brokerapi.RunDetail) int {
	if run == nil {
		return 0
	}
	total := 0
	for _, count := range run.ArtifactCountsByClass {
		total += count
	}
	return total
}

func authoritativeString(state map[string]any, key string) string {
	if len(state) == 0 {
		return ""
	}
	value, _ := state[key].(string)
	return strings.TrimSpace(value)
}

func chatExecutionTerminal(exec brokerapi.SessionTurnExecution) bool {
	state := strings.ToLower(strings.TrimSpace(exec.ExecutionState))
	if state == "completed" || state == "failed" {
		return true
	}
	outcome := strings.ToLower(strings.TrimSpace(exec.TerminalOutcome))
	return outcome == "completed" || outcome == "failed" || outcome == "cancelled"
}

func chatExecutionStatusAndAction(exec brokerapi.SessionTurnExecution) (string, string) {
	status := chatExecutionHeadline(exec, nil)
	action := blockedExecutionAction(exec)
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "waiting") || chatExecutionApprovalRequired(exec, nil) {
		action = waitingExecutionAction(exec)
	}
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "blocked") && strings.TrimSpace(exec.WaitKind) == "project_blocked" {
		action = waitingExecutionAction(exec)
	}
	return status, strings.TrimSpace(action)
}

func blockedExecutionAction(exec brokerapi.SessionTurnExecution) string {
	if blocked := strings.TrimSpace(exec.BlockedReasonCode); blocked != "" {
		return "Review the blocking reason in Runs or the inspector: " + humanizeExecutionToken(blocked) + "."
	}
	return ""
}

func waitingExecutionAction(exec brokerapi.SessionTurnExecution) string {
	switch strings.TrimSpace(exec.WaitKind) {
	case "operator_input":
		return "Provide the missing operator input when you are ready to continue."
	case "approval":
		return "Open Approvals and review the exact gated action before deciding."
	case "external_dependency":
		return "Wait for the dependency to become ready, then check Runs for the resumed workflow."
	case "project_blocked":
		return "Project setup is blocking this workflow. Open Status and complete the recommended remediation before retrying."
	default:
		return blockedExecutionAction(exec)
	}
}

func chooseActionTextByPosture(existing string, posture brokerapi.ProjectSubstratePostureGetResponse, exec brokerapi.SessionTurnExecution) string {
	if strings.TrimSpace(existing) != "" && strings.TrimSpace(exec.WaitKind) != "project_blocked" {
		return existing
	}
	if !strings.EqualFold(strings.TrimSpace(exec.WaitKind), "project_blocked") {
		return existing
	}
	parts := []string{}
	if len(posture.RemediationGuidance) > 0 {
		parts = append(parts, "Remediation: "+joinCSVWithWrapHint(posture.RemediationGuidance))
	}
	if strings.TrimSpace(posture.BlockedExplanation) != "" {
		parts = append(parts, "Project posture: "+sanitizeUIText(posture.BlockedExplanation))
	}
	if len(parts) == 0 {
		return existing
	}
	if strings.TrimSpace(existing) != "" {
		parts = append([]string{strings.TrimSpace(existing)}, parts...)
	} else {
		parts = append([]string{"Project setup is blocking this workflow. Open Status for broker-owned remediation."}, parts...)
	}
	return strings.Join(parts, " | ")
}

func joinCSVWithWrapHint(values []string) string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	return strings.Join(clean, ", ")
}
