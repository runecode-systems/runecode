package main

import (
	"fmt"
	"strings"
)

func renderActionCenterDirectory(title string, items []actionCenterItem, selected int, width int) string {
	if len(items) == 0 {
		return renderStateCardSpec(stateCardSpec{State: routeLoadStateEmpty, Title: title, Message: "No broker-known follow-up items are in this queue.", Reason: "The current canonical broker surfaces did not return triage items for this family."})
	}
	rows := make([]boundedListRow, 0, len(items))
	for i, item := range items {
		marker := " "
		if i == selected {
			marker = ">"
		}
		rows = append(rows, boundedListRow{Text: fmt.Sprintf(" %s %s", marker, renderActionCenterItem(item, width-4)), Selectable: true})
	}
	return compactLines(tableHeader(title), renderBoundedList(boundedListSpec{Rows: rows, Selected: selected, Width: 0, ApplySelected: true, PreserveGaps: true}))
}

func renderActionCenterItems(items []actionCenterItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, renderActionCenterItem(item, 0))
	}
	return out
}

func renderActionCenterItem(item actionCenterItem, width int) string {
	parts := []string{
		fmt.Sprintf("%s %s", tableHeader(actionCenterStateLabel(item.State)), item.Title),
		"  urgency: " + humanizeExecutionToken(item.Urgency),
		"  reason: " + valueOrNA(item.Reason),
		"  impact: " + valueOrNA(item.Impact),
		"  owner/action: " + valueOrNA(strings.TrimSpace(strings.Join([]string{item.Owner, item.RequiredAction}, " • "))),
		"  target: " + valueOrNA(item.TargetLabel),
	}
	if width <= 0 {
		return strings.Join(parts, "\n")
	}
	wrapped := make([]string, 0, len(parts))
	for _, part := range parts {
		wrapped = append(wrapped, wrapDashboardLine(part, width))
	}
	return strings.Join(wrapped, "\n")
}

func actionCenterStateLabel(state routeLoadState) string {
	switch state {
	case routeLoadStateApprovalRequired:
		return "Approval required"
	case routeLoadStateBlocked:
		return "Blocked"
	case routeLoadStateDegraded:
		return "Needs review"
	case routeLoadStateReady:
		return "Ready"
	case routeLoadStateEmpty:
		return "Clear"
	default:
		return strings.ToUpper(string(state))
	}
}

func actionCenterFamilyLabel(family actionCenterFamily) string {
	switch family {
	case actionCenterFamilyApprovals:
		return "Approvals"
	case actionCenterFamilyOps:
		return "Operational Attention"
	case actionCenterFamilyBlocked:
		return "Blocked Work"
	default:
		return strings.TrimSpace(string(family))
	}
}

func renderActionCenterInspector(family actionCenterFamily, items []actionCenterItem, selected int) string {
	if len(items) == 0 || selected < 0 || selected >= len(items) {
		return renderInspectorHeader("Action Center inspector", appTheme.InspectorHint.Render("triage detail")) + "\n  No item selected."
	}
	item := items[selected]
	return compactLines(
		renderInspectorHeader("Action Center inspector", appTheme.InspectorHint.Render("triage detail")),
		fmt.Sprintf("  queue=%s", actionCenterFamilyLabel(family)),
		fmt.Sprintf("  title=%s", item.Title),
		fmt.Sprintf("  state=%s", valueOrNA(string(item.State))),
		fmt.Sprintf("  urgency=%s", valueOrNA(item.Urgency)),
		fmt.Sprintf("  reason=%s", valueOrNA(item.Reason)),
		fmt.Sprintf("  impact=%s", valueOrNA(item.Impact)),
		fmt.Sprintf("  owner=%s", valueOrNA(item.Owner)),
		fmt.Sprintf("  required_action=%s", valueOrNA(item.RequiredAction)),
		fmt.Sprintf("  target=%s", valueOrNA(item.TargetLabel)),
		fmt.Sprintf("  evidence=%s", valueOrNA(item.EvidenceCue)),
		fmt.Sprintf("  drill_down_target=%s/%s", valueOrNA(item.Target.Kind), valueOrNA(string(item.Target.RouteID))),
	)
}

func buildActionCenterSummary(families map[actionCenterFamily][]actionCenterItem) actionCenterSummary {
	approvals := countActiveActionCenterItems(families[actionCenterFamilyApprovals])
	ops := countActiveActionCenterItems(families[actionCenterFamilyOps])
	blocked := countActionCenterItemsByState(families[actionCenterFamilyBlocked], routeLoadStateBlocked)
	approvalBlocked := countActionCenterItemsByState(families[actionCenterFamilyBlocked], routeLoadStateApprovalRequired)
	if blocked > 0 {
		return actionCenterSummary{State: routeLoadStateBlocked, Title: "Blocked", Message: "Workflow progress is blocked and needs operator follow-up.", Reason: fmt.Sprintf("%d blocked item(s), %d approval-gated item(s), %d operational item(s), and %d approval decision(s) are visible.", blocked, approvalBlocked, ops, approvals), NextAction: "Start with blocked-work impact items, then review linked approvals or setup guidance."}
	}
	if approvals > 0 || approvalBlocked > 0 {
		return actionCenterSummary{State: routeLoadStateApprovalRequired, Title: "Needs attention", Message: "Approval decisions are waiting on the operator.", Reason: fmt.Sprintf("%d approval decision(s), %d approval-gated workflow item(s), and %d operational item(s) are visible.", approvals, approvalBlocked, ops), NextAction: "Open the approvals queue and review the exact gated action before deciding."}
	}
	if ops > 0 {
		return actionCenterSummary{State: routeLoadStateDegraded, Title: "Degraded", Message: "Runtime, evidence, setup, or sync posture needs review.", Reason: fmt.Sprintf("%d operational attention item(s) are visible.", ops), NextAction: "Review degraded items first, then confirm detail in Audit, Runs, or Status."}
	}
	return actionCenterSummary{State: routeLoadStateReady, Title: "No immediate follow-up", Message: "No broker-known attention items are currently waiting in Action Center.", Reason: "Approvals, blocked-work impact, and operational attention queues are clear on the current broker surfaces.", NextAction: "Return to Dashboard or Chat and continue work until new follow-up appears."}
}

func countActionCenterItemsByState(items []actionCenterItem, state routeLoadState) int {
	total := 0
	for _, item := range items {
		if item.State == state {
			total++
		}
	}
	return total
}

func countActiveActionCenterItems(items []actionCenterItem) int {
	total := 0
	for _, item := range items {
		if item.State != routeLoadStateReady && item.State != routeLoadStateEmpty {
			total++
		}
	}
	return total
}
