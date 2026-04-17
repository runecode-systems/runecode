package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m shellModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	for _, handler := range []func(tea.KeyMsg) (tea.Model, tea.Cmd, bool){
		m.handleOpenPaletteKey,
		m.handleOpenSessionQuickSwitchKey,
		m.handleQuickJumpRouteKey,
		m.handleToggleSidebarKey,
		m.handleCopyIdentityKey,
		m.handleCopyRouteActionKey,
		m.handleToggleSelectionModeKey,
		m.handleRunCommandKey,
		m.handleCycleThemeKey,
		m.handleLayoutSidebarWiderKey,
		m.handleLayoutSidebarNarrowerKey,
		m.handleLayoutInspectorWiderKey,
		m.handleLayoutInspectorNarrowerKey,
		m.handleLayoutToggleSidebarCollapseKey,
		m.handleLayoutToggleInspectorCollapseKey,
		m.handleEscapeCloseNarrowOverlaysKey,
		m.handleNarrowInspectorHotkey,
		m.handleBackRouteKey,
		m.handleCycleFocusNextKey,
		m.handleCycleFocusPrevKey,
		m.handleNavFocusKeys,
		m.handleScrollKeys,
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

func (m shellModel) handleOpenPaletteKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.OpenPalette.matches(key) {
		return m, nil, false
	}
	m.narrowSidebarOn = false
	m.narrowInspectOn = false
	m.sessions = m.sessions.Close()
	m.palette = m.palette.UpdateEntries(m.buildPaletteEntries()).Open()
	m.focusManager.Set(focusPalette)
	m.focus = m.focusManager.Current()
	m.syncOverlayStack()
	return m, nil, true
}

func (m shellModel) handleOpenSessionQuickSwitchKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.OpenSessionQuickSwitch.matches(key) {
		return m, nil, false
	}
	m.narrowSidebarOn = false
	m.narrowInspectOn = false
	m.palette = m.palette.Close()
	m.sessions = m.sessions.Open(m.sessionItems)
	m.focusManager.Set(focusPalette)
	m.focus = m.focusManager.Current()
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
		m.narrowSidebarOn = !m.narrowSidebarOn
		if m.narrowSidebarOn {
			m.narrowInspectOn = false
			m.focusManager.Set(focusNav)
			m.focus = m.focusManager.Current()
		} else if m.focus == focusNav {
			m.focusManager.Set(focusContent)
			m.focus = m.focusManager.Current()
		}
		m.syncOverlayStack()
		m.toasts.Push(toastInfo, "Sidebar overlay toggled for narrow layout.")
		return m, nil, true
	}
	m.sidebarVisible = !m.sidebarVisible
	m.sidebarFolded = false
	if !m.effectiveSidebarVisible() && m.focus == focusNav {
		m.focusManager.Set(focusContent)
		m.focus = m.focusManager.Current()
	}
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
	if m.commands.Execute("shell.toggle_sidebar", &m) {
		m.persistWorkbenchState()
	}
	return m, nil, true
}

func (m shellModel) handleCycleThemeKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.CycleTheme.matches(key) {
		return m, nil, false
	}
	if m.commands.Execute("shell.cycle_theme", &m) {
		m.applyPreferredPresentationToRoutes()
		m.applyInspectorVisibilityToRoutes()
		m.persistWorkbenchState()
	}
	return m, nil, true
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
	if m.commands.Execute(commandID, &m) {
		m.persistWorkbenchState()
	}
	return m, nil, true
}

func (m shellModel) handleLayoutToggleInspectorCollapseKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.LayoutToggleInspectorCollapse.matches(key) {
		return m, nil, false
	}
	if m.breakpoint() == shellBreakpointNarrow {
		return m.toggleNarrowInspectorOverlay(), nil, true
	}
	if m.commands.Execute("shell.layout.toggle_inspector_collapse", &m) {
		m.persistWorkbenchState()
	}
	return m, nil, true
}

func (m shellModel) toggleNarrowInspectorOverlay() shellModel {
	surface := m.activeShellSurface()
	if strings.TrimSpace(surface.Inspector) == "" {
		m.toasts.Push(toastWarn, "Inspector unavailable for current route.")
		return m
	}
	m.narrowInspectOn = !m.narrowInspectOn
	if m.narrowInspectOn {
		m.narrowSidebarOn = false
	}
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
	if m.focus == focusPalette {
		m.focusManager.Set(focusContent)
		m.focus = m.focusManager.Current()
	}
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
	m.focusManager.Next(m.navigationSurfaceVisible(), m.palette.IsOpen() || m.sessions.IsOpen())
	m.focus = m.focusManager.Current()
	return m, nil, true
}

func (m shellModel) handleCycleFocusPrevKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.keys.CycleFocusPrev.matches(key) {
		return m, nil, false
	}
	m.focusManager.Prev(m.navigationSurfaceVisible(), m.palette.IsOpen() || m.sessions.IsOpen())
	m.focus = m.focusManager.Current()
	return m, nil, true
}

func (m shellModel) handleNavFocusKeys(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.focus != focusNav || !m.navigationSurfaceVisible() {
		return m, nil, false
	}
	if m.keys.SessionNext.matches(key) {
		m.moveSessionSelection(1)
		return m, nil, true
	}
	if m.keys.SessionPrev.matches(key) {
		m.moveSessionSelection(-1)
		return m, nil, true
	}
	if m.keys.SessionPin.matches(key) {
		m.toggleSelectedSessionPin()
		m.persistWorkbenchState()
		return m, nil, true
	}
	if m.keys.SessionOpen.matches(key) {
		updated, cmd := m.activateSelectedSessionFromSidebar()
		return updated, cmd, true
	}
	if m.keys.RouteNext.matches(key) {
		m.nav.MoveNext()
		return m, nil, true
	}
	if m.keys.RoutePrev.matches(key) {
		m.nav.MovePrev()
		return m, nil, true
	}
	if m.keys.RouteOpen.matches(key) {
		route := m.nav.Selected()
		updated, cmd := m.applyPaletteAction(paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "route", RouteID: route.ID}})
		return updated, cmd, true
	}
	return m, nil, false
}

func (m shellModel) handleScrollKeys(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.keys.ScrollDown.matches(key) {
		m.scroll++
		return m, nil, true
	}
	if m.keys.ScrollUp.matches(key) {
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil, true
	}
	return m, nil, false
}
