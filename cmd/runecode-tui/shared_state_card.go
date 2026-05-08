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
	lines := []string{
		tableHeader(strings.ToUpper(label)) + " " + strings.TrimSpace(title),
		appTheme.SurfaceCard.Padding(0, 1).Render(compactLines("Message: "+message, "Reason: "+reason)),
	}
	if nextAction != "" {
		lines = append(lines, muted("Next: "+nextAction))
	}
	if cues != "" {
		lines = append(lines, muted(cues))
	}
	return compactLines(lines...)
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
