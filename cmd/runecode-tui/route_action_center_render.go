package main

import (
	"fmt"
	"strings"
)

func renderActionCenterQueueStrip(vm actionCenterViewModel, active actionCenterFamily) string {
	parts := []string{
		fmt.Sprintf("Approvals %d", len(vm.Families[actionCenterFamilyApprovals])),
		fmt.Sprintf("Operational Attention %d", len(vm.Families[actionCenterFamilyOps])),
		fmt.Sprintf("Blocked Work %d", len(vm.Families[actionCenterFamilyBlocked])),
		fmt.Sprintf("Focus %s", actionCenterFamilyLabel(active)),
	}
	return strings.Join(parts, "  •  ")
}

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
	title := fmt.Sprintf("%s %s", tableHeader(actionCenterStateLabel(item.State)), item.Title)
	summary := "  " + actionCenterMainListSummary(item)
	if width <= 0 {
		return strings.Join([]string{title, summary}, "\n")
	}
	return strings.Join([]string{clipDisplayText(title, width), clipDisplayText(summary, width)}, "\n")
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

func actionCenterMainListSummary(item actionCenterItem) string {
	reason := actionCenterShortReason(item.Reason)
	continueIn := actionCenterContinueCue(item.TargetLabel)
	if continueIn == "" {
		return valueOrNA(reason)
	}
	if reason == "" {
		return continueIn
	}
	return reason + " • " + continueIn
}

func actionCenterShortReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" || strings.EqualFold(reason, "n/a") {
		return ""
	}
	for _, sep := range []string{";", "."} {
		if idx := strings.Index(reason, sep); idx > 0 {
			reason = reason[:idx]
			break
		}
	}
	return strings.TrimSpace(reason)
}

func actionCenterContinueCue(target string) string {
	target = strings.TrimSpace(target)
	if target == "" || strings.EqualFold(target, "n/a") {
		return ""
	}
	return "Continue in " + target
}

func renderActionCenterInspector(family actionCenterFamily, items []actionCenterItem, selected int) string {
	if len(items) == 0 || selected < 0 || selected >= len(items) {
		return renderInspectorHeader("Action Center inspector", appTheme.InspectorHint.Render("triage detail")) + "\n  No item selected."
	}
	item := items[selected]
	drillDown := "Unavailable"
	if strings.TrimSpace(item.Target.Kind) != "" || strings.TrimSpace(string(item.Target.RouteID)) != "" {
		drillDown = strings.TrimSpace(strings.Join([]string{valueOrNA(item.TargetLabel), fmt.Sprintf("(%s/%s)", valueOrNA(item.Target.Kind), valueOrNA(string(item.Target.RouteID)))}, " "))
	}
	return compactLines(
		renderInspectorHeader("Action Center inspector", appTheme.InspectorHint.Render("triage detail")),
		fmt.Sprintf("  %s • %s • %s", actionCenterFamilyLabel(family), actionCenterStateLabel(item.State), humanizeExecutionToken(item.Urgency)),
		"  "+item.Title,
		"",
		"  Why this needs attention",
		"    "+valueOrNA(item.Reason),
		"  What it affects",
		"    "+valueOrNA(item.Impact),
		"  What to do",
		"    "+valueOrNA(strings.TrimSpace(strings.Join([]string{item.Owner, item.RequiredAction}, " • "))),
		"  Continue in",
		"    "+drillDown,
		"  Evidence",
		"    "+valueOrNA(item.EvidenceCue),
	)
}

func buildActionCenterSummary(families map[actionCenterFamily][]actionCenterItem) actionCenterSummary {
	approvals := countActiveActionCenterItems(families[actionCenterFamilyApprovals])
	ops := countActiveActionCenterItems(families[actionCenterFamilyOps])
	blocked := countActionCenterItemsByState(families[actionCenterFamilyBlocked], routeLoadStateBlocked)
	approvalBlocked := countActionCenterItemsByState(families[actionCenterFamilyBlocked], routeLoadStateApprovalRequired)
	if blocked > 0 {
		return actionCenterSummary{State: routeLoadStateBlocked, Title: "Action Center", Message: "Workflow progress is blocked and needs operator follow-up.", Reason: fmt.Sprintf("%d blocked item(s), %d approval-gated item(s), %d operational item(s), and %d approval decision(s) are visible.", blocked, approvalBlocked, ops, approvals), NextAction: "Start with Blocked Work, then review linked approvals or setup guidance."}
	}
	if approvals > 0 || approvalBlocked > 0 {
		return actionCenterSummary{State: routeLoadStateApprovalRequired, Title: "Action Center", Message: "Approval decisions are waiting on the operator.", Reason: fmt.Sprintf("%d approval decision(s), %d approval-gated workflow item(s), and %d operational item(s) are visible.", approvals, approvalBlocked, ops), NextAction: "Open Approvals and review the exact gated action before deciding."}
	}
	if ops > 0 {
		return actionCenterSummary{State: routeLoadStateDegraded, Title: "Action Center", Message: "Runtime, evidence, setup, or sync posture needs review.", Reason: fmt.Sprintf("%d operational attention item(s) are visible.", ops), NextAction: "Review Operational Attention first, then confirm detail in Audit, Runs, or Status."}
	}
	return actionCenterSummary{State: routeLoadStateReady, Title: "Action Center", Message: "No broker-known attention items are currently waiting in Action Center.", Reason: "Approvals, Blocked Work, and Operational Attention are clear on the current broker surfaces.", NextAction: "Return to Dashboard or Chat and continue work until new follow-up appears."}
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
