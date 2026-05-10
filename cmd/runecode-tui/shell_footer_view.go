package main

import "strings"

func (m shellModel) renderBottomStrip(surface routeSurface) string {
	routeKeys := strings.TrimSpace(surface.Regions.Bottom.Body)
	commandLine := m.renderBottomCommandLine()
	width := m.width
	if width <= 0 {
		width = 120
	}
	return constrainShellBlock(routeKeys, width, 1) + "\n" + commandLine
}

func (m shellModel) renderStatusSurface(surface routeSurface) string {
	parts := []string{}
	if toast := strings.TrimSpace(m.toasts.Latest()); toast != "" {
		parts = append(parts, "Toast: "+sanitizeUIText(toast))
	}
	if status := strings.TrimSpace(surface.Regions.Status.Body); status != "" {
		parts = append(parts, status)
	}
	return strings.Join(parts, "  •  ")
}

func (m shellModel) renderBottomCommandLine() string {
	if m.emergencyQuit.pending {
		return "Emergency quit armed: press ctrl+c once more to quit."
	}
	if prompt := strings.TrimSpace(m.commandMode.RenderPrompt()); prompt != "" {
		return prompt
	}
	parts := []string{"ctrl+p commands", ": command", m.keys.LeaderStart.label() + " leader", "tab focus"}
	if quit := strings.TrimSpace(m.renderQuitDiscoverabilityHint()); quit != "" {
		parts = append(parts, quit)
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
