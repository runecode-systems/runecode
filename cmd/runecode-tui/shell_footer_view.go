package main

import (
	"fmt"
	"strings"
)

func (m shellModel) renderBottomStrip(surface routeSurface) string {
	if m.emergencyQuit.pending {
		return "Emergency quit armed: press ctrl+c once more to quit."
	}
	if prompt := strings.TrimSpace(m.commandMode.RenderPrompt()); prompt != "" {
		return prompt
	}
	parts := []string{"ctrl+p commands", ": command", "tab focus"}
	if quit := strings.TrimSpace(m.renderQuitDiscoverabilityHint()); quit != "" {
		parts = append(parts, quit)
	}
	if actions := strings.TrimSpace(m.renderRouteActionHints(surface)); actions != "" {
		parts = append(parts, actions)
	}
	if diagnostic := strings.TrimSpace(m.renderBottomDiagnosticLine(surface)); diagnostic != "" && !strings.Contains(diagnostic, "Broker-owned truth stays") {
		parts = append(parts, diagnostic)
	}
	if bottom := strings.TrimSpace(surface.Regions.Bottom.Body); bottom != "" {
		parts = append(parts, bottom)
	}
	return strings.Join(parts, "  •  ")
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

func (m shellModel) renderBottomActionLine(surface routeSurface) string {
	parts := make([]string, 0, 3)
	if quit := strings.TrimSpace(m.renderQuitDiscoverabilityHint()); quit != "" {
		parts = append(parts, quit)
	}
	if actions := strings.TrimSpace(m.renderRouteActionHints(surface)); actions != "" {
		parts = append(parts, actions)
	}
	if len(parts) == 0 {
		return muted("ctrl+p opens command discovery")
	}
	return strings.Join(parts, "  •  ")
}

func (m shellModel) renderRouteActionHints(surface routeSurface) string {
	parts := []string{}
	if len(surface.Actions.LocalActions) > 0 {
		parts = append(parts, fmt.Sprintf("%d route actions", len(surface.Actions.LocalActions)))
	}
	if len(surface.Actions.ReferenceActions) > 0 {
		parts = append(parts, fmt.Sprintf("%d linked refs", len(surface.Actions.ReferenceActions)))
	}
	if len(surface.Actions.CopyActions) > 0 {
		parts = append(parts, fmt.Sprintf("%d copy actions", len(surface.Actions.CopyActions)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Route actions: " + strings.Join(parts, " · ")
}

func (m shellModel) renderStatusSurface(surface routeSurface) string {
	status := strings.TrimSpace(surface.Regions.Status.Body)
	if status == "" {
		status = fmt.Sprintf("%s ready", m.routeLabel(m.currentRouteID()))
	}
	return "Status: " + status
}

func (m shellModel) renderBottomDiagnosticLine(surface routeSurface) string {
	parts := []string{}
	if m.selectionMode {
		parts = append(parts, "Selection mode on")
	}
	if actions := len(surface.Actions.CopyActions); actions > 0 {
		parts = append(parts, fmt.Sprintf("Copy via action entry (%d)", actions))
	}
	if m.commandMode.Active() {
		parts = append(parts, "Command mode")
	}
	if len(parts) == 0 {
		parts = append(parts, muted("Broker-owned truth stays in Status, inspectors, and command discovery"))
	}
	return strings.Join(parts, " • ")
}
