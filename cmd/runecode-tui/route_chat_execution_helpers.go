package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func chatVisibleExecution(detail *brokerapi.SessionDetail) *brokerapi.SessionTurnExecution {
	if detail == nil {
		return nil
	}
	if detail.CurrentTurnExecution != nil {
		return detail.CurrentTurnExecution
	}
	return detail.LatestTurnExecution
}

func chatExecutionStateCard(detail *brokerapi.SessionDetail, posture *brokerapi.ProjectSubstratePostureGetResponse, run *brokerapi.RunDetail) stateCardSpec {
	exec := chatVisibleExecution(detail)
	if detail == nil {
		return stateCardSpec{
			State:       routeLoadStateEmpty,
			Title:       "Workflow execution",
			Message:     "Select a canonical session to review workflow progress.",
			Reason:      "Chat follows the broker-owned session selected in this workspace.",
			NextAction:  "Pick a session from the directory or reload if none are listed.",
			ShortcutCue: "j/k move • enter review",
			RouteCue:    "Chat",
		}
	}
	if exec == nil {
		return stateCardSpec{
			State:       routeLoadStateEmpty,
			Title:       "Workflow execution",
			Message:     "No workflow execution is attached to this session yet.",
			Reason:      chatExecutionEvidenceSummary(detail, nil),
			NextAction:  "Open the composer when you are ready to start the next broker-owned workflow.",
			ShortcutCue: "c compose",
			RouteCue:    "Runs after start",
		}
	}
	state := chatExecutionCardState(*exec, run)
	message := chatExecutionHeadline(*exec, run)
	reason := chatExecutionReason(*exec, detail, posture, run)
	nextAction := chatExecutionNextAction(*exec, posture, run)
	return stateCardSpec{
		State:       state,
		Title:       "Workflow execution",
		Message:     message,
		Reason:      reason,
		NextAction:  nextAction,
		ShortcutCue: chatExecutionShortcutCue(*exec),
		RouteCue:    chatExecutionRouteCue(*exec, detail, run),
	}
}

func chatExecutionCardState(exec brokerapi.SessionTurnExecution, run *brokerapi.RunDetail) routeLoadState {
	if chatExecutionApprovalRequired(exec, run) {
		return routeLoadStateApprovalRequired
	}
	state := strings.ToLower(strings.TrimSpace(exec.ExecutionState))
	switch {
	case state == "waiting":
		return routeLoadStateWaiting
	case state == "blocked":
		return routeLoadStateBlocked
	case state == "completed" || strings.EqualFold(strings.TrimSpace(exec.TerminalOutcome), "completed"):
		return routeLoadStateCompleted
	case state == "failed" || strings.EqualFold(strings.TrimSpace(exec.TerminalOutcome), "failed"):
		return routeLoadStateDegraded
	case run != nil && run.Coordination.Blocked:
		return routeLoadStateBlocked
	default:
		return routeLoadStateReady
	}
}

func chatExecutionHeadline(exec brokerapi.SessionTurnExecution, run *brokerapi.RunDetail) string {
	state := strings.ToLower(strings.TrimSpace(exec.ExecutionState))
	switch {
	case chatExecutionApprovalRequired(exec, run):
		return "Workflow is waiting for approval."
	case state == "waiting":
		return "Workflow is waiting for the next broker-owned signal."
	case state == "blocked":
		return "Workflow cannot continue yet."
	case state == "completed" || strings.EqualFold(strings.TrimSpace(exec.TerminalOutcome), "completed"):
		return "Workflow completed."
	case state == "failed" || strings.EqualFold(strings.TrimSpace(exec.TerminalOutcome), "failed"):
		return "Workflow failed."
	default:
		if run != nil && runCheckpointCode(run) != "" {
			return "Workflow is active and reporting progress."
		}
		if chatRunIDFromExecution(exec) != "" {
			return "Workflow is active in a linked run."
		}
		return "Workflow is active."
	}
}

func chatExecutionReason(exec brokerapi.SessionTurnExecution, detail *brokerapi.SessionDetail, posture *brokerapi.ProjectSubstratePostureGetResponse, run *brokerapi.RunDetail) string {
	parts := []string{}
	if reason := chatExecutionWaitReason(exec, posture, run); reason != "" {
		parts = append(parts, reason)
	}
	if stages := chatExecutionStageSummary(exec, detail, run); stages != "" {
		parts = append(parts, stages)
	}
	if evidence := chatExecutionEvidenceSummary(detail, run); evidence != "" {
		parts = append(parts, evidence)
	}
	return strings.Join(parts, " • ")
}

func chatExecutionWaitReason(exec brokerapi.SessionTurnExecution, posture *brokerapi.ProjectSubstratePostureGetResponse, run *brokerapi.RunDetail) string {
	if chatExecutionApprovalRequired(exec, run) {
		return "Approval is still required before the broker can continue this workflow."
	}
	if wait := strings.TrimSpace(exec.WaitState); wait != "" {
		switch strings.TrimSpace(exec.WaitKind) {
		case "operator_input":
			return "Waiting for more operator input."
		case "external_dependency":
			return "Waiting for an external dependency to become ready."
		case "project_blocked":
			if posture != nil && strings.TrimSpace(posture.BlockedExplanation) != "" {
				return sanitizeUIText(posture.BlockedExplanation)
			}
			return "Project setup is currently blocking normal workflow execution."
		default:
			return fmt.Sprintf("Waiting reason: %s.", humanizeExecutionToken(wait))
		}
	}
	if blocked := strings.TrimSpace(exec.BlockedReasonCode); blocked != "" {
		return fmt.Sprintf("Blocked reason: %s.", humanizeExecutionToken(blocked))
	}
	if run != nil && run.Coordination.Blocked {
		if reason := strings.TrimSpace(run.Coordination.WaitReasonCode); reason != "" {
			return fmt.Sprintf("Run is blocked by %s.", humanizeExecutionToken(reason))
		}
		return "Run coordination is currently blocked."
	}
	return ""
}

func chatExecutionEvidenceSummary(detail *brokerapi.SessionDetail, run *brokerapi.RunDetail) string {
	if detail == nil {
		return ""
	}
	runs := len(detail.LinkedRunIDs)
	approvals := len(detail.LinkedApprovalIDs)
	artifacts := len(detail.LinkedArtifactDigests)
	audit := len(detail.LinkedAuditRecordDigests)
	if run != nil && runs == 0 && strings.TrimSpace(run.Summary.RunID) != "" {
		runs = 1
	}
	parts := []string{
		fmt.Sprintf("linked runs %d", runs),
		fmt.Sprintf("approvals %d", approvals),
		fmt.Sprintf("artifacts %d", maxInt(artifacts, runArtifactCount(run))),
		fmt.Sprintf("audit %d", audit),
	}
	return "Evidence: " + strings.Join(parts, " • ")
}

func chatExecutionStageSummary(exec brokerapi.SessionTurnExecution, detail *brokerapi.SessionDetail, run *brokerapi.RunDetail) string {
	stages := make([]string, 0, 6)
	if run != nil && runPlanAuthorityAvailable(run) {
		stages = append(stages, "plan compiled")
	}
	if chatRunIDFromExecution(exec) != "" {
		stages = append(stages, "run linked")
	}
	if runRunnerActive(run) {
		stages = append(stages, "runner active")
	}
	if runCheckpointCode(run) != "" {
		stages = append(stages, "checkpoint received")
	}
	if chatExecutionApprovalRequired(exec, run) {
		stages = append(stages, "approval required")
	}
	if len(exec.LinkedArtifactDigests) > 0 || len(detail.LinkedArtifactDigests) > 0 || runArtifactCount(run) > 0 {
		stages = append(stages, "artifact ready")
	}
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "failed") || strings.EqualFold(strings.TrimSpace(exec.TerminalOutcome), "failed") {
		stages = append(stages, "failed")
	}
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "completed") || strings.EqualFold(strings.TrimSpace(exec.TerminalOutcome), "completed") {
		stages = append(stages, "completed")
	}
	stages = uniqueStageLabels(stages)
	if len(stages) == 0 {
		return ""
	}
	return "Broker-known stages: " + strings.Join(stages, " • ")
}

func uniqueStageLabels(labels []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	return out
}

func chatExecutionNextAction(exec brokerapi.SessionTurnExecution, posture *brokerapi.ProjectSubstratePostureGetResponse, run *brokerapi.RunDetail) string {
	action := strings.TrimSpace(blockedExecutionAction(exec))
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "waiting") || chatExecutionApprovalRequired(exec, run) {
		action = strings.TrimSpace(waitingExecutionAction(exec))
	}
	if posture != nil {
		action = strings.TrimSpace(chooseActionTextByPosture(action, *posture, exec))
	}
	if action != "" {
		return action
	}
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "completed") {
		return "Review the linked run, artifacts, or audit evidence if you want the full result trail."
	}
	return "Stay in Chat for follow-up, or open Runs for fuller workflow evidence."
}

func chatExecutionShortcutCue(exec brokerapi.SessionTurnExecution) string {
	if strings.EqualFold(strings.TrimSpace(exec.WaitKind), "approval") || strings.TrimSpace(exec.PendingApprovalID) != "" || len(exec.LinkedApprovalIDs) > 0 {
		return "open inspector or palette references"
	}
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "completed") {
		return "enter review • c follow-up"
	}
	return "c compose • enter review"
}

func chatExecutionRouteCue(exec brokerapi.SessionTurnExecution, detail *brokerapi.SessionDetail, run *brokerapi.RunDetail) string {
	routes := []string{"Runs"}
	if chatExecutionApprovalRequired(exec, run) || len(detail.LinkedApprovalIDs) > 0 {
		routes = append(routes, "Approvals")
	}
	if len(detail.LinkedArtifactDigests) > 0 || runArtifactCount(run) > 0 {
		routes = append(routes, "Artifacts")
	}
	if len(detail.LinkedAuditRecordDigests) > 0 {
		routes = append(routes, "Audit")
	}
	return strings.Join(uniqueStageLabels(routes), " / ")
}

func chatExecutionApprovalRequired(exec brokerapi.SessionTurnExecution, run *brokerapi.RunDetail) bool {
	if strings.EqualFold(strings.TrimSpace(exec.WaitKind), "approval") || strings.TrimSpace(exec.PendingApprovalID) != "" || len(exec.LinkedApprovalIDs) > 0 {
		return true
	}
	return run != nil && len(run.PendingApprovalIDs) > 0
}

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
		action = "Open Status and complete the recommended project setup remediation before retrying."
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
		return "Use Status to fix project setup before retrying this workflow."
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
