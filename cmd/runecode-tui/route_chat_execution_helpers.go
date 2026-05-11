package main

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/brokerapi"
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
			Reason:      chatSessionExecutionSummary(detail),
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
	if summary := chatSessionExecutionSummary(detail); summary != "" {
		parts = append(parts, summary)
	}
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

func chatSessionExecutionSummary(detail *brokerapi.SessionDetail) string {
	if detail == nil {
		return ""
	}
	parts := []string{}
	if current := detail.CurrentTurnExecution; current != nil {
		parts = append(parts, "Current broker state: "+chatExecutionSnapshotLabel(*current))
	}
	if latest := detail.LatestTurnExecution; latest != nil {
		parts = append(parts, "Latest broker state: "+chatExecutionSnapshotLabel(*latest))
	}
	if len(parts) == 0 {
		parts = append(parts, "No broker-owned execution snapshot is attached to this session.")
	}
	return strings.Join(parts, " • ")
}

func chatExecutionSnapshotLabel(exec brokerapi.SessionTurnExecution) string {
	state := humanizeExecutionToken(exec.ExecutionState)
	parts := []string{state}
	if strings.TrimSpace(exec.WaitState) != "" {
		parts = append(parts, "wait "+humanizeExecutionToken(exec.WaitState))
	}
	if strings.TrimSpace(exec.TerminalOutcome) != "" {
		parts = append(parts, "outcome "+humanizeExecutionToken(exec.TerminalOutcome))
	}
	return strings.Join(parts, " / ")
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
		countNoun(runs, "linked run", "linked runs"),
		countNoun(approvals, "approval", "approvals"),
		countNoun(maxInt(artifacts, runArtifactCount(run)), "artifact", "artifacts"),
		countNoun(audit, "audit record", "audit records"),
	}
	return "Evidence: " + strings.Join(parts, " • ")
}

func chatExecutionStageSummary(exec brokerapi.SessionTurnExecution, detail *brokerapi.SessionDetail, run *brokerapi.RunDetail) string {
	stages := make([]string, 0, 8)
	if executionWaiting(exec, run) {
		stages = append(stages, "waiting")
	}
	if run != nil && runPlanAuthorityAvailable(run) {
		stages = append(stages, "plan compiled")
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
	if len(exec.LinkedArtifactDigests) > 0 || runArtifactCount(run) > 0 {
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
	if chatExecutionApprovalRequired(exec, run) {
		return "Open Approvals and review the exact gated action before deciding."
	}
	action := strings.TrimSpace(blockedExecutionAction(exec))
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "waiting") {
		action = strings.TrimSpace(waitingExecutionAction(exec))
	}
	if strings.EqualFold(strings.TrimSpace(exec.ExecutionState), "blocked") && strings.EqualFold(strings.TrimSpace(exec.WaitKind), "project_blocked") {
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
	if strings.EqualFold(strings.TrimSpace(exec.WaitKind), "approval") || strings.TrimSpace(exec.PendingApprovalID) != "" {
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
	if chatExecutionTerminal(exec) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(exec.WaitKind), "approval") || strings.TrimSpace(exec.PendingApprovalID) != "" {
		return true
	}
	return run != nil && len(run.PendingApprovalIDs) > 0
}
