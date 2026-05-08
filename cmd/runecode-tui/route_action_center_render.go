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
		fmt.Sprintf("%s %s urgency=%s", tableHeader(strings.ToUpper(string(item.State))), item.Title, valueOrNA(item.Urgency)),
		"  reason: " + valueOrNA(item.Reason),
		"  impact: " + valueOrNA(item.Impact),
		"  action: " + valueOrNA(item.RequiredAction),
		"  target: " + valueOrNA(item.TargetLabel),
		"  evidence: " + valueOrNA(item.EvidenceCue),
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

func renderActionCenterInspector(family actionCenterFamily, items []actionCenterItem, selected int) string {
	if len(items) == 0 || selected < 0 || selected >= len(items) {
		return renderInspectorHeader("Action Center inspector", appTheme.InspectorHint.Render("triage detail")) + "\n  No item selected."
	}
	item := items[selected]
	return compactLines(
		renderInspectorHeader("Action Center inspector", appTheme.InspectorHint.Render("triage detail")),
		fmt.Sprintf("  family=%s", family),
		fmt.Sprintf("  title=%s", item.Title),
		fmt.Sprintf("  state=%s", valueOrNA(string(item.State))),
		fmt.Sprintf("  urgency=%s", valueOrNA(item.Urgency)),
		fmt.Sprintf("  reason=%s", valueOrNA(item.Reason)),
		fmt.Sprintf("  impact=%s", valueOrNA(item.Impact)),
		fmt.Sprintf("  required_action=%s", valueOrNA(item.RequiredAction)),
		fmt.Sprintf("  target=%s", valueOrNA(item.TargetLabel)),
		fmt.Sprintf("  evidence=%s", valueOrNA(item.EvidenceCue)),
		fmt.Sprintf("  drill_down_target=%s/%s", valueOrNA(item.Target.Kind), valueOrNA(string(item.Target.RouteID))),
	)
}

func buildActionCenterSummary(families map[actionCenterFamily][]actionCenterItem) actionCenterSummary {
	approvals := countActiveActionCenterItems(families[actionCenterFamilyApprovals])
	ops := countActiveActionCenterItems(families[actionCenterFamilyOps])
	blocked := countActiveActionCenterItems(families[actionCenterFamilyBlocked])
	if blocked > 0 {
		return actionCenterSummary{State: routeLoadStateBlocked, Title: "Blocked follow-up", Message: "Action Center has broker-known blockers that are holding workflow progress.", Reason: fmt.Sprintf("blocked_work_impact=%d operational_attention=%d approvals=%d", blocked, ops, approvals), NextAction: "Start with blocked-work impact items, then review linked approvals or setup guidance."}
	}
	if ops > 0 {
		return actionCenterSummary{State: routeLoadStateDegraded, Title: "Needs operational attention", Message: "Action Center has degraded runtime, evidence, setup, or sync follow-up to review.", Reason: fmt.Sprintf("operational_attention=%d approvals=%d", ops, approvals), NextAction: "Review degraded items first, then confirm detail in Audit, Runs, or Status."}
	}
	if approvals > 0 {
		return actionCenterSummary{State: routeLoadStateApprovalRequired, Title: "Approval follow-up waiting", Message: "Action Center has broker-known approval decisions waiting on the operator.", Reason: fmt.Sprintf("approvals=%d", approvals), NextAction: "Open the approvals queue and review the exact gated action before deciding."}
	}
	return actionCenterSummary{State: routeLoadStateReady, Title: "No immediate follow-up", Message: "No broker-known attention items are currently waiting in Action Center.", Reason: "Approvals, blocked-work impact, and operational attention queues are clear on the current broker surfaces.", NextAction: "Return to Dashboard or Chat and continue work until new follow-up appears."}
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
