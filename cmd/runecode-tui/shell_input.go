package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m shellModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	for _, handler := range []func(tea.KeyMsg) (tea.Model, tea.Cmd, bool){
		m.handleCommandModeActiveKey,
		m.handleLeaderActiveKey,
		m.handleOpenCommandModeKey,
		m.handleOpenLeaderKey,
		m.handleOpenPaletteKey,
		m.handleOpenSessionQuickSwitchKey,
		m.handleEscapeCloseNarrowOverlaysKey,
		m.handleCycleFocusNextKey,
		m.handleCycleFocusPrevKey,
		m.handleNavFocusKeys,
	} {
		if updated, cmd, handled := handler(key); handled {
			return updated, cmd
		}
	}

	updated, cmd := m.updateActiveRoute(key)
	shell := updated.(shellModel)
	shell.captureInspectorVisibilityFromActiveRoute()
	shell.capturePreferredPresentationFromActiveSurface()
	return shell, cmd
}

func (m shellModel) handleOpenLeaderKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.leader.Active() || !m.keys.LeaderStart.matches(key) {
		return m, nil, false
	}
	if !m.shellPowerKeysAllowed() {
		return m, nil, false
	}
	m.leader.Rebind(m.actions.leaderBindings(m))
	m.beginOverlaySession()
	m.leader.Start()
	m.setFocus(focusPalette)
	m.syncOverlayStack()
	return m, nil, true
}

func (m shellModel) handleLeaderActiveKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.leader.Active() {
		return m, nil, false
	}
	if key.Type == tea.KeyEsc {
		m.leader.Abort()
		m.syncOverlayStack()
		m.restoreFocusAfterOverlayClose()
		m.toasts.Push(toastInfo, "Leader mode aborted.")
		return m, nil, true
	}
	token, ok := leaderTokenFromKey(key)
	if !ok {
		prior := append([]string(nil), m.leader.prefix...)
		m.leader.Abort()
		m.syncOverlayStack()
		m.restoreFocusAfterOverlayClose()
		m.toasts.Push(toastWarn, formatLeaderInvalidKeyMessage(prior, key.String()))
		return m, nil, true
	}
	prior := append([]string(nil), m.leader.prefix...)
	action, complete := m.leader.Step(token)
	if !complete {
		if !m.leader.Active() {
			m.syncOverlayStack()
			m.restoreFocusAfterOverlayClose()
			m.toasts.Push(toastWarn, formatLeaderInvalidKeyMessage(prior, token))
			return m, nil, true
		}
		m.syncOverlayStack()
		return m, nil, true
	}
	m.syncOverlayStack()
	m.restoreFocusAfterOverlayClose()
	updated, cmd := m.applyPaletteAction(action)
	return updated, cmd, true
}

func (m shellModel) handleOpenPaletteKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.OpenPalette.matches(key) {
		return m, nil, false
	}
	if !m.shellPowerKeysAllowed() {
		return m, nil, false
	}
	m.narrowSidebarOn = false
	m.narrowInspectOn = false
	m.beginOverlaySession()
	m.sessions = m.sessions.Close()
	m.palette = m.palette.UpdateEntries(m.buildPaletteEntries()).Open()
	m.setFocus(focusPalette)
	m.syncOverlayStack()
	return m, nil, true
}

func (m shellModel) handleOpenSessionQuickSwitchKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.OpenSessionQuickSwitch.matches(key) {
		return m, nil, false
	}
	if !m.shellPowerKeysAllowed() {
		return m, nil, false
	}
	m.narrowSidebarOn = false
	m.narrowInspectOn = false
	m.beginOverlaySession()
	m.palette = m.palette.Close()
	m.sessions = m.sessions.Open(m.sessionItems)
	m.setFocus(focusPalette)
	m.syncOverlayStack()
	return m, nil, true
}

func (m shellModel) handleQuickJumpRouteKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	route, ok := routeByQuickJumpKey(key.String(), m.routes)
	if !ok {
		return m, nil, false
	}
	updated, cmd := m.applyPaletteAction(paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: route.ID}})
	return updated, cmd, true
}

func (m shellModel) handleToggleSidebarKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.ToggleSidebar.matches(key) {
		return m, nil, false
	}
	if m.breakpoint() == shellBreakpointNarrow {
		m.beginOverlaySession()
		m.narrowSidebarOn = !m.narrowSidebarOn
		if m.narrowSidebarOn {
			m.narrowInspectOn = false
			m.setFocus(focusNav)
		} else if m.focus == focusNav {
			m.restoreFocusAfterOverlayClose()
		}
		m.syncOverlayStack()
		m.toasts.Push(toastInfo, "Sidebar overlay toggled for narrow layout.")
		return m, nil, true
	}
	m.sidebarVisible = !m.sidebarVisible
	m.sidebarFolded = false
	m.normalizeFocusForLayout()
	m.persistWorkbenchState()
	m.toasts.Push(toastInfo, "Sidebar visibility changed.")
	return m, nil, true
}

func (m shellModel) handleCopyIdentityKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.CopyIdentity.matches(key) {
		return m, nil, false
	}
	m.copyCurrentIdentity()
	return m, nil, true
}

func (m shellModel) handleCopyRouteActionKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.CopyRouteAction.matches(key) {
		return m, nil, false
	}
	m.copyNextRouteAction()
	return m, nil, true
}

func (m shellModel) handleToggleSelectionModeKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.ToggleSelectionMode.matches(key) {
		return m, nil, false
	}
	m.selectionMode = !m.selectionMode
	state := "off"
	if m.selectionMode {
		state = "on"
	}
	m.toasts.Push(toastInfo, "Selection mode "+state+" (mouse capture toggled).")
	return m, m.mouseCaptureCmd(), true
}

func (m shellModel) handleRunCommandKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.RunCommand.matches(key) {
		return m, nil, false
	}
	cmd := m.commands.Execute("shell.toggle_sidebar", &m)
	m.persistWorkbenchState()
	return m, cmd, true
}

func (m shellModel) handleCycleThemeKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.CycleTheme.matches(key) {
		return m, nil, false
	}
	cmd := m.commands.Execute("shell.cycle_theme", &m)
	m.publishShellPreferencesToCurrentRoute()
	m.persistWorkbenchState()
	return m, cmd, true
}

func (m shellModel) handleLayoutSidebarWiderKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	return m.handleLayoutCommandKey(key, m.keys.LayoutSidebarWider, "shell.layout.sidebar_wider")
}

func (m shellModel) handleLayoutSidebarNarrowerKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	return m.handleLayoutCommandKey(key, m.keys.LayoutSidebarNarrower, "shell.layout.sidebar_narrower")
}

func (m shellModel) handleLayoutInspectorWiderKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	return m.handleLayoutCommandKey(key, m.keys.LayoutInspectorWider, "shell.layout.inspector_wider")
}

func (m shellModel) handleLayoutInspectorNarrowerKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	return m.handleLayoutCommandKey(key, m.keys.LayoutInspectorNarrower, "shell.layout.inspector_narrower")
}

func (m shellModel) handleLayoutToggleSidebarCollapseKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	return m.handleLayoutCommandKey(key, m.keys.LayoutToggleSidebarCollapse, "shell.layout.toggle_sidebar_collapse")
}

func (m shellModel) handleLayoutCommandKey(key tea.KeyMsg, binding keyBinding, commandID string) (tea.Model, tea.Cmd, bool) {
	if !binding.matches(key) {
		return m, nil, false
	}
	cmd := m.commands.Execute(commandID, &m)
	m.publishShellPreferencesToCurrentRoute()
	m.persistWorkbenchState()
	return m, cmd, true
}

func (m shellModel) handleLayoutToggleInspectorCollapseKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.LayoutToggleInspectorCollapse.matches(key) {
		return m, nil, false
	}
	if m.breakpoint() == shellBreakpointNarrow {
		m.beginOverlaySession()
		return m.toggleNarrowInspectorOverlay(), nil, true
	}
	cmd := m.commands.Execute("shell.layout.toggle_inspector_collapse", &m)
	m.publishShellPreferencesToCurrentRoute()
	m.normalizeFocusForLayout()
	m.persistWorkbenchState()
	return m, cmd, true
}

func (m shellModel) toggleNarrowInspectorOverlay() shellModel {
	surface := m.activeShellSurface()
	if !routeInspectorAvailable(surface) {
		m.toasts.Push(toastWarn, "Inspector unavailable for current route.")
		return m
	}
	m.narrowInspectOn = !m.narrowInspectOn
	if m.narrowInspectOn {
		m.narrowSidebarOn = false
		if strings.TrimSpace(surface.Regions.Inspector.Body) == "" {
			m.toasts.Push(toastInfo, "Inspector opened with no selected detail; showing empty-state inspector.")
		}
		m.setFocus(focusInspector)
	} else {
		m.restoreFocusAfterOverlayClose()
	}
	m.publishShellPreferencesToCurrentRoute()
	m.syncOverlayStack()
	return m
}

func (m shellModel) handleEscapeCloseNarrowOverlaysKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if key.Type != tea.KeyEsc || (!m.narrowSidebarOn && !m.narrowInspectOn) {
		return m, nil, false
	}
	m.narrowSidebarOn = false
	m.narrowInspectOn = false
	m.syncOverlayStack()
	m.restoreFocusAfterOverlayClose()
	return m, nil, true
}

func (m shellModel) handleNarrowInspectorHotkey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.breakpoint() != shellBreakpointNarrow || key.String() != "i" {
		return m, nil, false
	}
	m = m.toggleNarrowInspectorOverlay()
	return m, nil, true
}

func (m shellModel) handleBackRouteKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.BackRoute.matches(key) {
		return m, nil, false
	}
	updated, cmd := m.applyPaletteAction(paletteActionMsg{Verb: verbBack})
	return updated, cmd, true
}

func (m shellModel) handleCycleFocusNextKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.CycleFocusNext.matches(key) {
		return m, nil, false
	}
	if !m.shellFocusTraversalAllowed() {
		return m, nil, false
	}
	m.focusManager.Next(m.planShellLayout(m.activeShellSurface()), m.commandOverlayOpen())
	m.focus = m.focusManager.Current()
	return m, nil, true
}

func (m shellModel) handleCycleFocusPrevKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.CycleFocusPrev.matches(key) {
		return m, nil, false
	}
	if !m.shellFocusTraversalAllowed() {
		return m, nil, false
	}
	m.focusManager.Prev(m.planShellLayout(m.activeShellSurface()), m.commandOverlayOpen())
	m.focus = m.focusManager.Current()
	return m, nil, true
}

func (m shellModel) handleNavFocusKeys(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.focus != focusNav || !m.navigationSurfaceVisible() {
		return m, nil, false
	}
	if m.keys.RouteNext.matches(key) || m.keys.SessionNext.matches(key) {
		m.moveSidebarCursor(1)
		return m, nil, true
	}
	if m.keys.RoutePrev.matches(key) || m.keys.SessionPrev.matches(key) {
		m.moveSidebarCursor(-1)
		return m, nil, true
	}
	if m.keys.SessionPin.matches(key) {
		m.toggleSelectedSessionPin()
		m.persistWorkbenchState()
		return m, nil, true
	}
	if m.keys.RouteOpen.matches(key) || m.keys.SessionOpen.matches(key) {
		entry, ok := m.selectedSidebarEntry()
		if !ok {
			return m, nil, true
		}
		switch entry.Kind {
		case sidebarEntryRoute:
			updated, cmd := m.applyPaletteAction(paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: entry.Route.ID}})
			return updated, cmd, true
		case sidebarEntrySession:
			updated, cmd := m.activateSessionFromSidebarByID(entry.Session.Identity.SessionID)
			return updated, cmd, true
		case sidebarEntryAction:
			updated, cmd := m.activateSidebarAction(entry.ActionID)
			return updated, cmd, true
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m shellModel) handleScrollKeys(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.keys.ScrollDown.matches(key) {
		updated, cmd := m.updateActiveRoute(routeViewportScrollMsg{Region: m.focusedRouteRegion(), Delta: 1})
		return updated, cmd, true
	}
	if m.keys.ScrollUp.matches(key) {
		updated, cmd := m.updateActiveRoute(routeViewportScrollMsg{Region: m.focusedRouteRegion(), Delta: -1})
		return updated, cmd, true
	}
	return m, nil, false
}
