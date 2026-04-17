package main

import tea "github.com/charmbracelet/bubbletea"

func defaultShellCommandRegistry() shellCommandRegistry {
	r := newShellCommandRegistry()
	r.Register(shellCommand{ID: "shell.toggle_sidebar", Title: "Toggle Sidebar", Description: "Show or hide sidebar", Run: func(m *shellModel) {
		m.sidebarVisible = !m.sidebarVisible
		m.toasts.Push(toastInfo, "Sidebar visibility toggled from command registry.")
	}})
	r.Register(shellCommand{ID: "shell.focus_main", Title: "Focus Main Pane", Description: "Move focus to main content", Run: func(m *shellModel) {
		m.setFocus(focusContent)
	}})
	r.Register(shellCommand{ID: "shell.toggle_inspector", Title: "Toggle Inspector Pane", Description: "Toggle route inspector visibility", Run: func(m *shellModel) {
		m.toggleActiveInspector()
	}})
	r.Register(shellCommand{ID: "shell.cycle_theme", Title: "Cycle Theme Preset", Description: "Rotate dark/dusk/high-contrast themes", Run: func(m *shellModel) {
		m.themePreset = nextThemePreset(m.themePreset)
		appTheme = newTheme(m.themePreset)
		m.persistWorkbenchState()
		m.toasts.Push(toastInfo, "Theme preset set to "+string(m.themePreset)+".")
	}})
	r.Register(shellCommand{ID: "shell.layout.sidebar_wider", Title: "Layout Sidebar Wider", Description: "Increase sidebar pane ratio", Run: func(m *shellModel) {
		m.sidebarRatio = clampPaneRatio(m.sidebarRatio + 0.03)
		m.sidebarFolded = false
		m.persistWorkbenchState()
	}})
	r.Register(shellCommand{ID: "shell.layout.sidebar_narrower", Title: "Layout Sidebar Narrower", Description: "Decrease sidebar pane ratio", Run: func(m *shellModel) {
		m.sidebarRatio = clampPaneRatio(m.sidebarRatio - 0.03)
		m.persistWorkbenchState()
	}})
	r.Register(shellCommand{ID: "shell.layout.inspector_wider", Title: "Layout Inspector Wider", Description: "Increase inspector pane ratio", Run: func(m *shellModel) {
		m.inspectorRatio = clampPaneRatio(m.inspectorRatio + 0.03)
		m.inspectorFolded = false
		m.persistWorkbenchState()
	}})
	r.Register(shellCommand{ID: "shell.layout.inspector_narrower", Title: "Layout Inspector Narrower", Description: "Decrease inspector pane ratio", Run: func(m *shellModel) {
		m.inspectorRatio = clampPaneRatio(m.inspectorRatio - 0.03)
		m.persistWorkbenchState()
	}})
	r.Register(shellCommand{ID: "shell.layout.toggle_sidebar_collapse", Title: "Layout Toggle Sidebar Collapse", Description: "Collapse or expand sidebar pane", Run: func(m *shellModel) {
		m.sidebarFolded = !m.sidebarFolded
		m.persistWorkbenchState()
	}})
	r.Register(shellCommand{ID: "shell.layout.toggle_inspector_collapse", Title: "Layout Toggle Inspector Collapse", Description: "Collapse or expand inspector pane", Run: func(m *shellModel) {
		m.inspectorFolded = !m.inspectorFolded
		m.persistWorkbenchState()
	}})
	r.Register(shellCommand{ID: "shell.toggle_selection_mode", Title: "Toggle Selection Mode", Description: "Disable/enable mouse capture for drag-to-select", Run: func(m *shellModel) {
		m.selectionMode = !m.selectionMode
	}, PostRun: func(m *shellModel) tea.Cmd {
		_ = m
		return m.mouseCaptureCmd()
	}})
	return r
}
