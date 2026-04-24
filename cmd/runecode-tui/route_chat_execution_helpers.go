package main

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func chatExecutionTerminal(exec brokerapi.SessionTurnExecution) bool {
	state := strings.ToLower(strings.TrimSpace(exec.ExecutionState))
	if state == "completed" || state == "failed" {
		return true
	}
	outcome := strings.ToLower(strings.TrimSpace(exec.TerminalOutcome))
	return outcome == "completed" || outcome == "failed" || outcome == "cancelled"
}

func chatExecutionStatusAndAction(exec brokerapi.SessionTurnExecution) (string, string) {
	state := strings.TrimSpace(exec.ExecutionState)
	if state == "" {
		state = "unknown"
	}
	status := "Execution progress: " + state
	if wait := strings.TrimSpace(exec.WaitState); wait != "" {
		status += " (" + wait + ")"
	}
	action := blockedExecutionAction(exec)
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "waiting") {
		action = waitingExecutionAction(exec)
	}
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "blocked") && strings.TrimSpace(exec.WaitKind) == "project_blocked" {
		action = "Execution is blocked by project posture; diagnostics/remediation attach remains inspect-only."
	}
	return status, strings.TrimSpace(action)
}

func blockedExecutionAction(exec brokerapi.SessionTurnExecution) string {
	if blocked := strings.TrimSpace(exec.BlockedReasonCode); blocked != "" {
		return "Blocked reason: " + blocked
	}
	return ""
}

func waitingExecutionAction(exec brokerapi.SessionTurnExecution) string {
	switch strings.TrimSpace(exec.WaitKind) {
	case "operator_input":
		return "Follow-up: provide operator input to continue this dependent scope."
	case "approval":
		return "Follow-up: open Approvals and resolve the pending formal approval."
	case "external_dependency":
		return "Follow-up: wait for external dependency readiness, then continue."
	case "project_blocked":
		return "Follow-up: project substrate posture blocks execution; apply remediation posture before continue."
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
	return strings.Join(parts, " | ")
}
