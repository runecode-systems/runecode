package main

import "strings"

type stateCardSpec struct {
	State       routeLoadState
	Title       string
	Message     string
	Reason      string
	NextAction  string
	ShortcutCue string
	RouteCue    string
	EvidenceCue string
}

func renderStateCard(state routeLoadState, title, message string) string {
	return renderStateCardSpec(stateCardSpec{State: state, Title: title, Message: message})
}

func renderStateCardSpec(spec stateCardSpec) string {
	label := strings.TrimSpace(string(spec.State))
	if label == "" {
		label = string(routeLoadStateReady)
	}
	title := titleOrFallback(spec.Title, "Status")
	message := strings.TrimSpace(spec.Message)
	if message == "" {
		message = "n/a"
	}
	reason := strings.TrimSpace(spec.Reason)
	if reason == "" {
		reason = message
	}
	nextAction := strings.TrimSpace(spec.NextAction)
	if nextAction == "" {
		nextAction = stateCardNextStep(spec.State)
	}
	cues := stateCardCueLine(spec.ShortcutCue, spec.RouteCue)
	lines := []string{appTheme.SurfaceCard.Padding(0, 1).Render(compactLines("Message: "+message, "Reason: "+reason))}
	if nextAction != "" {
		lines = append(lines, "Next: "+nextAction)
	}
	if evidence := strings.TrimSpace(spec.EvidenceCue); evidence != "" {
		lines = append(lines, muted("Evidence: "+evidence))
	}
	if cues != "" {
		lines = append(lines, muted(cues))
	}
	return renderProductCard(productCardSpec{Tone: stateCardTone(spec.State), Title: strings.ToUpper(label) + " " + strings.TrimSpace(title), Badge: toneBadge(stateCardTone(spec.State), strings.ToUpper(label)), Lines: lines})
}

func stateCardTone(state routeLoadState) visualTone {
	switch state {
	case routeLoadStateReady, routeLoadStateCompleted:
		return visualToneSuccess
	case routeLoadStateWaiting, routeLoadStateApprovalRequired, routeLoadStateLoading, routeLoadStateEmpty:
		return visualToneAttention
	case routeLoadStateBlocked, routeLoadStateDegraded, routeLoadStateError:
		return visualToneDanger
	default:
		return visualToneInfo
	}
}

func stateCardCueLine(shortcutCue, routeCue string) string {
	parts := []string{}
	if shortcut := strings.TrimSpace(shortcutCue); shortcut != "" {
		parts = append(parts, "Shortcut: "+shortcut)
	}
	if route := strings.TrimSpace(routeCue); route != "" {
		parts = append(parts, "Route: "+route)
	}
	return strings.Join(parts, " • ")
}

func stateCardNextStep(state routeLoadState) string {
	switch state {
	case routeLoadStateLoading:
		return "Waiting for the broker response to settle."
	case routeLoadStateWaiting:
		return "Wait for the broker-owned state to advance or inspect the linked route for evidence."
	case routeLoadStateBlocked:
		return "Review the blocking reason and complete the required operator action before retrying."
	case routeLoadStateDegraded:
		return "Inspect the degraded details and use Status or Audit to confirm current evidence posture."
	case routeLoadStateError:
		return "Use the route reload shortcut to try again."
	case routeLoadStateCompleted:
		return "Review the resulting artifacts, approvals, or audit evidence if you need more detail."
	case routeLoadStateApprovalRequired:
		return "Open Approvals and review the exact gated action before deciding."
	case routeLoadStateEmpty:
		return "No matching records are available for the current route state."
	default:
		return ""
	}
}
