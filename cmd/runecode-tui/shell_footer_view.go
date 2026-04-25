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
	selectionHint := "Selection mode off; mouse capture remains enabled."
	if m.selectionMode {
		selectionHint = "Selection mode on; drag-to-select is enabled until you exit it."
	}
	selectionHint = m.renderQuitDiscoverabilityHint() + " | " + selectionHint
	return compactLines(
		tableHeader("Bottom strip"),
		bottom,
		m.renderRouteActionHints(surface),
		m.renderRouteCopyActions(),
		selectionHint,
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
		parts = append(parts, fmt.Sprintf("Linked refs actionable=%d", len(surface.Actions.ReferenceActions)))
	}
	if len(surface.Actions.LocalActions) > 0 {
		parts = append(parts, fmt.Sprintf("Local actions executable=%d", len(surface.Actions.LocalActions)))
	}
	if len(parts) == 0 {
		return muted("Actionable refs/actions: none for the current view")
	}
	return "Actionable refs/actions: " + strings.Join(parts, " | ")
}

func (m shellModel) renderStatusSurface(surface routeSurface) string {
	status := strings.TrimSpace(surface.Regions.Status.Body)
	if status == "" {
		status = fmt.Sprintf("route=%s", m.routeLabel(m.currentRouteID()))
	}
	selection := "selection=off"
	if m.selectionMode {
		selection = "selection=on"
	}
	return "Status: " + status + " | " + selection + " | clipboard=" + sanitizeUIText(m.clipboard.IntegrationHint())
}

func (m shellModel) renderRouteCopyActions() string {
	actions := m.activeShellSurface().Actions.CopyActions
	if len(actions) == 0 {
		return muted("Copy actions: none (use terminal selection for long-form text).")
	}
	items := make([]string, 0, len(actions))
	for i, action := range actions {
		label := strings.TrimSpace(action.Label)
		if label == "" {
			label = strings.TrimSpace(action.ID)
		}
		if label == "" {
			label = fmt.Sprintf("copy-%d", i+1)
		}
		if i == m.copyActionIndex {
			label = "[next:" + label + "]"
		}
		items = append(items, label)
	}
	return "Copy actions (use action entry): " + strings.Join(items, " | ")
}
