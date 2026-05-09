package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func approvalCountsByRun(approvals []brokerapi.ApprovalSummary) map[string]int {
	byRun := map[string]int{}
	for _, ap := range approvals {
		runID := strings.TrimSpace(ap.BoundScope.RunID)
		if runID != "" {
			byRun[runID]++
		}
	}
	return byRun
}

func runHasBlockedImpact(run brokerapi.RunSummary) bool {
	state := strings.ToLower(strings.TrimSpace(run.LifecycleState))
	return strings.Contains(state, "block") || strings.Contains(state, "wait") || run.PendingApprovalCount > 0 || strings.TrimSpace(run.BlockingReasonCode) != ""
}

func linkedApprovalCount(run brokerapi.RunSummary, byRun map[string]int) int {
	if run.PendingApprovalCount > 0 {
		return run.PendingApprovalCount
	}
	return byRun[run.RunID]
}

func blockedImpactItem(run brokerapi.RunSummary, blockedCount int) actionCenterItem {
	urgency, stateCard, owner, requiredAction := blockedImpactDisposition(run)
	return actionCenterItem{
		Title:          fmt.Sprintf("run %s blocked impact", valueOrNA(run.RunID)),
		State:          stateCard,
		Urgency:        urgency,
		Reason:         blockedImpactReason(run, blockedCount),
		Impact:         fmt.Sprintf("Workflow progress is waiting for run %s; pending_approvals=%d linked_queue_items=%d.", valueOrNA(run.RunID), run.PendingApprovalCount, blockedCount),
		Owner:          owner,
		RequiredAction: requiredAction,
		TargetLabel:    fmt.Sprintf("Runs › %s", valueOrNA(run.RunID)),
		EvidenceCue:    fmt.Sprintf("source=run_summary run_id=%s", valueOrNA(run.RunID)),
		Target:         paletteTarget{Kind: "run", RouteID: routeRuns, RunID: run.RunID},
	}
}

func blockedImpactDisposition(run brokerapi.RunSummary) (string, routeLoadState, string, string) {
	if run.PendingApprovalCount > 0 {
		return "high", routeLoadStateApprovalRequired, "owner: operator makes approval decision", "Open Approvals or Runs, review the exact gate, then decide so workflow progress can resume."
	}
	return "medium", routeLoadStateBlocked, "owner: operator clears workflow blocker", "Open Runs to inspect the blocked reason and linked approvals, then continue from the target route."
}

func blockedImpactReason(run brokerapi.RunSummary, blockedCount int) string {
	reasonParts := []string{fmt.Sprintf("lifecycle=%s", valueOrNA(run.LifecycleState))}
	if strings.TrimSpace(run.BlockingReasonCode) != "" {
		reasonParts = append(reasonParts, fmt.Sprintf("reason=%s", valueOrNA(run.BlockingReasonCode)))
	}
	if blockedCount > 0 {
		reasonParts = append(reasonParts, fmt.Sprintf("approval_queue_items=%d", blockedCount))
	}
	return strings.Join(reasonParts, "; ")
}
