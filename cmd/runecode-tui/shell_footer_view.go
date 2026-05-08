package main

import (
	"fmt"
	"strings"
)

func (m shellModel) renderBottomStrip(surface routeSurface) string {
	bottom := ""
	if m.emergencyQuit.pending {
		bottom = "Emergency quit armed — press ctrl+c once more to quit."
	} else {
		bottom = strings.TrimSpace(m.commandMode.RenderPrompt())
	}
	if bottom == "" {
		bottom = strings.TrimSpace(surface.Regions.Bottom.Body)
	}
	if bottom == "" {
		bottom = muted("No route composer or status actions for this screen.")
	}
	discovery := m.renderQuitDiscoverabilityHint()
	if routeCue := strings.TrimSpace(m.renderRouteActionHints(surface)); routeCue != "" {
		if discovery != "" {
			discovery += " | "
		}
		discovery += routeCue
	}
	diagnostic := m.renderBottomDiagnosticLine()
	return compactLines(
		tableHeader("Bottom strip"),
		bottom,
		discovery,
		muted(diagnostic),
	)
}

func (m shellModel) renderQuitDiscoverabilityHint() string {
	action, ok := m.actions.definitionByID("shell.quit")
	if !ok {
		return ""
	}
	label := "Quit RuneCode"
	if title := strings.TrimSpace(action.Title); title != "" {
		label = title
	}
	return "Quick action: " + label + " (:quit)"
}

func (m shellModel) renderRouteActionHints(surface routeSurface) string {
	parts := []string{}
	if len(surface.Actions.ReferenceActions) > 0 {
		parts = append(parts, fmt.Sprintf("%d linked refs", len(surface.Actions.ReferenceActions)))
	}
	if len(surface.Actions.LocalActions) > 0 {
		parts = append(parts, fmt.Sprintf("%d local actions", len(surface.Actions.LocalActions)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Next actions: " + strings.Join(parts, " | ")
}

func (m shellModel) renderStatusSurface(surface routeSurface) string {
	status := strings.TrimSpace(surface.Regions.Status.Body)
	if status == "" {
		status = fmt.Sprintf("route=%s", m.routeLabel(m.currentRouteID()))
	}
	return "Status: " + status
}

func (m shellModel) renderBottomDiagnosticLine() string {
	parts := []string{}
	if m.selectionMode {
		parts = append(parts, "Selection mode on")
	}
	if actions := len(m.activeShellSurface().Actions.CopyActions); actions > 0 {
		parts = append(parts, fmt.Sprintf("copy actions %d via action entry", actions))
	}
	if len(parts) == 0 {
		parts = append(parts, "Use Status, inspectors, or command discovery for detailed diagnostics")
	}
	return strings.Join(parts, " • ")
}
