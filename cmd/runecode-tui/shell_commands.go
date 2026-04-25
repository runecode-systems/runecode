package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func defaultShellCommandRegistry() shellCommandRegistry {
	r := newShellCommandRegistry()
	registerWorkbenchCommands(&r)
	registerLayoutCommands(&r)
	registerClipboardAndOverlayCommands(&r)
	registerCommandModeCommands(&r)
	return r
}

func registerWorkbenchCommands(r *shellCommandRegistry) {
	for _, cmd := range []shellCommand{
		{ID: "shell.toggle_sidebar", Title: "Toggle Sidebar", Description: "Show or hide sidebar", Aliases: []string{"sidebar toggle"}, LeaderPath: []string{"w", "s"}, LeaderGroup: "Workbench", PaletteShow: true, PaletteText: "sidebar panel visibility", HelpText: "sidebar toggle — show or hide sidebar", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { m.toggleSidebar() }},
		{ID: "shell.focus_main", Title: "Focus Main Pane", Description: "Move focus to main content", PaletteShow: false, HelpText: "focus main pane", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { m.setFocus(focusContent) }},
		{ID: "shell.focus_next", Title: "Focus Next Pane", Description: "Move focus to next shell pane", Aliases: []string{"focus next"}, LeaderPath: []string{"w", "n"}, LeaderGroup: "Workbench", PaletteShow: true, PaletteText: "focus next pane", HelpText: "focus next — move focus to next pane", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { moveShellFocus(m, true) }},
		{ID: "shell.focus_prev", Title: "Focus Previous Pane", Description: "Move focus to previous shell pane", Aliases: []string{"focus prev", "focus previous"}, LeaderPath: []string{"w", "p"}, LeaderGroup: "Workbench", PaletteShow: true, PaletteText: "focus previous pane", HelpText: "focus prev — move focus to previous pane", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { moveShellFocus(m, false) }},
		{ID: "shell.toggle_inspector", Title: "Toggle Inspector Pane", Description: "Toggle route inspector visibility", Aliases: []string{"inspector toggle"}, LeaderPath: []string{"w", "i"}, LeaderGroup: "Workbench", PaletteShow: true, PaletteText: "inspector pane visibility", HelpText: "inspector toggle — show or hide inspector", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { m.toggleActiveInspector() }},
		{ID: "shell.cycle_theme", Title: "Cycle Theme Preset", Description: "Rotate dark/dusk/high-contrast themes", Aliases: []string{"theme cycle"}, LeaderPath: []string{"w", "t"}, LeaderGroup: "Workbench", PaletteShow: true, PaletteText: "theme dark dusk high-contrast", HelpText: "theme cycle — rotate theme preset", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { cycleShellTheme(m) }},
	} {
		r.Register(cmd)
	}
}

func registerLayoutCommands(r *shellCommandRegistry) {
	for _, cmd := range []shellCommand{
		{ID: "shell.layout.sidebar_wider", Title: "Layout Sidebar Wider", Description: "Increase sidebar pane ratio", PaletteShow: true, PaletteText: "layout sidebar wider", HelpText: "layout sidebar wider", Scope: shellActionScopeGlobal, Run: func(m *shellModel) {
			m.sidebarRatio = clampPaneRatio(m.sidebarRatio + 0.03)
			m.sidebarFolded = false
			m.persistWorkbenchState()
		}},
		{ID: "shell.layout.sidebar_narrower", Title: "Layout Sidebar Narrower", Description: "Decrease sidebar pane ratio", PaletteShow: true, PaletteText: "layout sidebar narrower", HelpText: "layout sidebar narrower", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { m.sidebarRatio = clampPaneRatio(m.sidebarRatio - 0.03); m.persistWorkbenchState() }},
		{ID: "shell.layout.inspector_wider", Title: "Layout Inspector Wider", Description: "Increase inspector pane ratio", PaletteShow: true, PaletteText: "layout inspector wider", HelpText: "layout inspector wider", Scope: shellActionScopeGlobal, Run: func(m *shellModel) {
			m.inspectorRatio = clampPaneRatio(m.inspectorRatio + 0.03)
			m.inspectorFolded = false
			m.persistWorkbenchState()
		}},
		{ID: "shell.layout.inspector_narrower", Title: "Layout Inspector Narrower", Description: "Decrease inspector pane ratio", PaletteShow: true, PaletteText: "layout inspector narrower", HelpText: "layout inspector narrower", Scope: shellActionScopeGlobal, Run: func(m *shellModel) {
			m.inspectorRatio = clampPaneRatio(m.inspectorRatio - 0.03)
			m.persistWorkbenchState()
		}},
		{ID: "shell.layout.toggle_sidebar_collapse", Title: "Layout Toggle Sidebar Collapse", Description: "Collapse or expand sidebar pane", PaletteShow: true, PaletteText: "layout sidebar collapse", HelpText: "layout toggle sidebar collapse", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { m.sidebarFolded = !m.sidebarFolded; m.persistWorkbenchState() }},
		{ID: "shell.layout.toggle_inspector_collapse", Title: "Layout Toggle Inspector Collapse", Description: "Collapse or expand inspector pane", PaletteShow: true, PaletteText: "layout inspector collapse", HelpText: "layout toggle inspector collapse", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { m.inspectorFolded = !m.inspectorFolded; m.persistWorkbenchState() }},
	} {
		r.Register(cmd)
	}
}

func registerClipboardAndOverlayCommands(r *shellCommandRegistry) {
	for _, cmd := range []shellCommand{
		{ID: "shell.toggle_selection_mode", Title: "Toggle Selection Mode", Description: "Disable/enable mouse capture for drag-to-select", Aliases: []string{"selection toggle"}, PaletteShow: true, PaletteText: "selection mouse capture toggle", HelpText: "selection toggle — toggle mouse capture", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { toggleSelectionMode(m) }, PostRun: func(m *shellModel) tea.Cmd { return m.mouseCaptureCmd() }},
		{ID: "shell.copy_identity", Title: "Copy Current Identity", Description: "Copy the current route/object identity", PaletteShow: true, PaletteText: "copy identity route object", HelpText: "copy identity — copy current route/object identity", Scope: shellActionScopeRouteSensitive, Available: routeSensitivePowerAvailable, Run: func(m *shellModel) { m.copyCurrentIdentity() }},
		{ID: "shell.copy_route_action", Title: "Copy Next Route Action", Description: "Cycle through and execute route copy actions", PaletteShow: true, PaletteText: "copy next route action", HelpText: "copy next — cycle route copy actions", Scope: shellActionScopeRouteSensitive, Available: routeSensitivePowerAvailable, Run: func(m *shellModel) { m.copyNextRouteAction() }},
		{ID: "shell.open_palette", Title: "Open Command Discovery", Description: "Open fuzzy discovery palette", PaletteShow: false, HelpText: "search — open fuzzy command discovery", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { openPaletteOverlay(m) }},
		{ID: "shell.open_sessions", Title: "Open Session Switcher", Description: "Open fuzzy session quick-switch", PaletteShow: true, PaletteText: "sessions quick switch discover", HelpText: "sessions — open fuzzy session quick switcher", Scope: shellActionScopeGlobal, Run: func(m *shellModel) { openSessionSwitcherOverlay(m) }},
	} {
		r.Register(cmd)
	}
}

func registerCommandModeCommands(r *shellCommandRegistry) {
	r.Register(shellCommand{ID: "shell.open_route", Title: "Open Route", Description: "Open route by id/alias", PaletteShow: false, HelpText: "open <route> — jump to route", Scope: shellActionScopeGlobal, ResolveArgs: resolveOpenRouteArgs, Run: func(m *shellModel) {
		// ResolveArgs already applied route changes and history.
	}, PostRun: func(m *shellModel) tea.Cmd {
		m.publishShellPreferencesToCurrentRoute()
		m.persistWorkbenchState()
		return m.activateCurrentRouteCmd()
	}})
	r.Register(shellCommand{ID: "shell.set_leader", Title: "Set Leader Key", Description: "Set leader key in local workbench preferences", PaletteShow: true, PaletteText: "set leader key", HelpText: "set leader <space|comma|backslash|default> — configure persisted leader key", Scope: shellActionScopeGlobal, ResolveArgs: resolveSetLeaderArgs, Run: func(m *shellModel) {
		if m.leaderKeyConfig == "space" {
			m.toasts.Push(toastInfo, "Leader key reset to default (space).")
			return
		}
		m.toasts.Push(toastInfo, fmt.Sprintf("Leader key set to %q.", m.leaderKeyConfig))
	}})
}

func moveShellFocus(m *shellModel, next bool) {
	if !m.shellFocusTraversalAllowed() {
		return
	}
	if m.focus == focusPalette {
		m.setFocus(m.overlayReturn)
	}
	if next {
		m.focusManager.Next(m.planShellLayout(m.activeShellSurface()), m.commandOverlayOpen())
	} else {
		m.focusManager.Prev(m.planShellLayout(m.activeShellSurface()), m.commandOverlayOpen())
	}
	m.focus = m.focusManager.Current()
}

func cycleShellTheme(m *shellModel) {
	m.themePreset = nextThemePreset(m.themePreset)
	appTheme = newTheme(m.themePreset)
	m.persistWorkbenchState()
	m.toasts.Push(toastInfo, "Theme preset set to "+string(m.themePreset)+".")
}

func toggleSelectionMode(m *shellModel) {
	m.selectionMode = !m.selectionMode
	state := "off"
	if m.selectionMode {
		state = "on"
	}
	m.toasts.Push(toastInfo, "Selection mode "+state+" (mouse capture toggled).")
}

func routeSensitivePowerAvailable(m shellModel) bool {
	return m.keyboardOwnership() == routeKeyboardOwnershipNormal
}

func openPaletteOverlay(m *shellModel) {
	if !m.shellPowerKeysAllowed() {
		return
	}
	m.narrowSidebarOn = false
	m.narrowInspectOn = false
	m.beginOverlaySession()
	m.sessions = m.sessions.Close()
	m.palette = m.palette.UpdateEntries(m.buildPaletteEntries()).Open()
	m.setFocus(focusPalette)
	m.syncOverlayStack()
}

func openSessionSwitcherOverlay(m *shellModel) {
	if !m.shellPowerKeysAllowed() {
		return
	}
	m.narrowSidebarOn = false
	m.narrowInspectOn = false
	m.beginOverlaySession()
	m.palette = m.palette.Close()
	m.sessions = m.sessions.Open(m.sessionItems)
	m.setFocus(focusPalette)
	m.syncOverlayStack()
}

func resolveOpenRouteArgs(m *shellModel, args []string) bool {
	if len(args) == 0 {
		m.commandMode = m.commandMode.SetError("usage: open <route>")
		return false
	}
	target := strings.Join(args, " ")
	rid, ok := resolveRouteToken(target, m.routes)
	if !ok {
		m.commandMode = m.commandMode.SetError("unknown route \"" + strings.TrimSpace(target) + "\"")
		return false
	}
	prior := m.currentLocation()
	m.location.Primary = shellObjectLocation{RouteID: rid, Object: workbenchObjectRef{Kind: "route", ID: string(rid)}}
	m.location.Inspector = nil
	m.nav.SelectByRouteID(rid)
	m.copyActionIndex = 0
	m.syncSidebarCursorToLocation()
	m.setFocus(focusContent)
	m.history = append(m.history, prior)
	return true
}

func resolveSetLeaderArgs(m *shellModel, args []string) bool {
	if len(args) == 0 {
		m.commandMode = m.commandMode.SetError("usage: set leader <space|comma|backslash|default>")
		return false
	}
	if err := m.configureLeaderKey(args[0]); err != nil {
		m.commandMode = m.commandMode.SetError(err.Error())
		return false
	}
	return true
}

func resolveRouteToken(token string, routes []routeDefinition) (routeID, bool) {
	token = strings.ToLower(strings.TrimSpace(token))
	token = strings.TrimPrefix(token, "/")
	token = strings.ReplaceAll(token, "_", "-")
	if token == "" {
		return "", false
	}
	if token == "action" || token == "actioncenter" {
		token = "action-center"
	}
	for _, route := range routes {
		id := strings.ToLower(strings.TrimSpace(string(route.ID)))
		label := strings.ToLower(strings.TrimSpace(route.Label))
		label = strings.ReplaceAll(label, " ", "-")
		if token == id || token == label {
			return route.ID, true
		}
	}
	return "", false
}
