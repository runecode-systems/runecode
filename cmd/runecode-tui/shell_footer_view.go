package main

import "strings"

func (m shellModel) renderBottomStrip(surface routeSurface) string {
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
