package main

import "strings"

func (m *shellModel) capturePreferredPresentationFromActiveSurface() {
	surface := m.activeShellSurface()
	mode := normalizePresentationMode(contentPresentationMode(strings.TrimSpace(surface.Actions.ActiveTab)))
	if mode == m.preferredMode {
		return
	}
	m.preferredMode = mode
	m.persistWorkbenchState()
}

func (m *shellModel) captureInspectorVisibilityFromActiveRoute() {
	surface := m.activeShellSurface()
	if !surface.Capabilities.Inspector.Supported {
		return
	}
	visible := surface.Capabilities.Inspector.Enabled
	if visible == m.inspectorOn {
		return
	}
	m.inspectorOn = visible
	m.persistWorkbenchState()
}
