package main

import "strings"

func humanFocusLabel(focus focusArea) string {
	label := sanitizeUIText(focus.Label())
	if label == "" {
		return "Unknown"
	}
	return strings.ToUpper(label[:1]) + label[1:]
}

func humanShellActivitySummary(activity shellActivitySemantics) string {
	switch activity.State {
	case shellActivityStateLoading:
		return "Loading broker state"
	case shellActivityStateWaiting:
		if target := strings.TrimSpace(humanShellActivityTarget(activity.Active)); target != "" {
			return "Waiting on " + target
		}
		return "Waiting on broker follow-up"
	case shellActivityStateRunning:
		if target := strings.TrimSpace(humanShellActivityTarget(activity.Active)); target != "" {
			return "Working on " + target
		}
		return "Work is in progress"
	case shellActivityStateDegradedSync:
		return "Sync needs attention"
	default:
		return ""
	}
}

func humanShellActivityTarget(active shellActivityFocus) string {
	kind := strings.TrimSpace(active.Kind)
	id := strings.TrimSpace(active.ID)
	if kind == "" && id == "" {
		return ""
	}
	if kind == "" {
		return sanitizeUIText(id)
	}
	if id == "" {
		return sanitizeUIText(kind)
	}
	return sanitizeUIText(kind) + " " + sanitizeUIText(id)
}
