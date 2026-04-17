package main

func (m shellModel) overlayOpen() bool {
	return m.palette.IsOpen() || m.sessions.IsOpen() || m.narrowSidebarOn || m.narrowInspectOn
}

func (m shellModel) commandOverlayOpen() bool {
	return m.palette.IsOpen() || m.sessions.IsOpen()
}

func (m *shellModel) beginOverlaySession() {
	if m.overlayOpen() {
		return
	}
	m.overlayReturn = m.focus
}

func (m *shellModel) setFocus(area focusArea) {
	m.focusManager.Set(area)
	m.focus = m.focusManager.Current()
}

func (m *shellModel) normalizeFocusForLayout() {
	layout := m.planShellLayout(m.activeShellSurface())
	m.focusManager.Normalize(layout, m.commandOverlayOpen())
	m.focus = m.focusManager.Current()
}

func (m *shellModel) restoreFocusAfterOverlayClose() {
	if m.overlayOpen() {
		return
	}
	layout := m.planShellLayout(m.activeShellSurface())
	m.focusManager.Set(m.overlayReturn)
	m.focusManager.Normalize(layout, false)
	m.focus = m.focusManager.Current()
	m.overlayReturn = focusContent
}
