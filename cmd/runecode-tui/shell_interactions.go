package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m shellModel) handleMouse(mouse tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.selectionMode {
		return m, nil
	}
	if updated, cmd, handled := m.handleMouseLeftClick(mouse); handled {
		return updated, cmd
	}
	if updated, cmd, handled := m.handleMouseWheel(mouse); handled {
		return updated, cmd
	}
	return m, nil
}

func (m shellModel) handleMouseLeftClick(mouse tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	if mouse.Button != tea.MouseButtonLeft || mouse.Action != tea.MouseActionRelease {
		return m, nil, false
	}
	if m.navigationSurfaceVisible() {
		if idx, ok := m.sidebarIndexAtMouse(mouse.X, mouse.Y); ok {
			entries := m.sidebarEntries()
			if idx >= 0 && idx < len(entries) {
				m.sidebarCursor = idx
				entry := entries[idx]
				switch entry.Kind {
				case sidebarEntryRoute:
					updated, cmd := m.applyPaletteAction(paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: entry.Route.ID}})
					return updated, cmd, true
				case sidebarEntrySession:
					updated, cmd := m.activateSessionFromSidebarByID(entry.Session.Identity.SessionID)
					return updated, cmd, true
				}
			}
			return m, nil, true
		}
	}
	m.setFocus(focusContent)
	return m, nil, true
}

func (m shellModel) handleMouseWheel(mouse tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	if mouse.Button != tea.MouseButtonWheelUp && mouse.Button != tea.MouseButtonWheelDown {
		return m, nil, false
	}
	delta := 0
	switch mouse.Button {
	case tea.MouseButtonWheelUp:
		delta = -1
	case tea.MouseButtonWheelDown:
		delta = 1
	}
	m.setFocus(focusContent)
	updated, cmd := m.updateActiveRoute(routeViewportScrollMsg{Region: m.focusedRouteRegion(), Delta: delta})
	return updated, cmd, true
}

func (m *shellModel) copyCurrentIdentity() {
	breadcrumbs := m.activeShellSurface().Chrome.Breadcrumbs
	identity := m.routeLabel(m.currentRouteID())
	if len(breadcrumbs) > 0 {
		identity = breadcrumbs[len(breadcrumbs)-1]
	}
	m.copyText(identity, "Copied identity")
}

func (m *shellModel) copyNextRouteAction() {
	actions := m.activeShellSurface().Actions.CopyActions
	if len(actions) == 0 {
		m.toasts.Push(toastWarn, "No route copy actions available; use terminal selection mode for long-form content.")
		return
	}
	if m.copyActionIndex >= len(actions) {
		m.copyActionIndex = 0
	}
	action := actions[m.copyActionIndex]
	m.copyActionIndex = (m.copyActionIndex + 1) % len(actions)
	text := strings.TrimSpace(action.Text)
	if text == "" {
		m.toasts.Push(toastWarn, "Route copy action has empty content: "+defaultPlaceholder(action.Label, action.ID))
		return
	}
	label := strings.TrimSpace(action.Label)
	if label == "" {
		label = strings.TrimSpace(action.ID)
	}
	if label == "" {
		label = "route action"
	}
	m.copyText(text, "Copied "+label)
}

func (m *shellModel) copyText(text string, label string) {
	text = strings.TrimSpace(redactSecrets(text))
	if text == "" {
		return
	}
	m.clipboard.Copy(text)
	m.toasts.Push(toastInfo, fmt.Sprintf("%s via %s", label, m.clipboard.IntegrationHint()))
}
