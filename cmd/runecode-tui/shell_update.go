package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m shellModel) handleQuitMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}
	if key.String() == "ctrl+c" {
		if m.emergencyQuit.pending {
			m.quitConfirm = shellQuitConfirmState{}
			m.emergencyQuit.pending = false
			m.quitting = true
			return m, tea.Quit, true
		}
		updated, cmd := m.requestQuitActionWithReason("quit confirmation")
		shell := updated.(shellModel)
		if shell.quitConfirm.active {
			shell.emergencyQuit.pending = true
			shell.emergencyQuit.token++
			token := shell.emergencyQuit.token
			tick := tea.Tick(emergencyQuitArmWindow, func(time.Time) tea.Msg {
				return shellEmergencyQuitTimeoutMsg{token: token}
			})
			shell.toasts.Push(toastWarn, "Quit requested. Press ctrl+c again to quit immediately.")
			if cmd != nil {
				return shell, tea.Batch(cmd, tick), true
			}
			return shell, tick, true
		}
		return shell, cmd, true
	}
	if m.emergencyQuit.pending {
		m.emergencyQuit.pending = false
	}
	return m, nil, false
}

func (m shellModel) handleEmergencyQuitTimeoutMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	timed, ok := msg.(shellEmergencyQuitTimeoutMsg)
	if !ok {
		return m, nil, false
	}
	if !m.emergencyQuit.pending {
		return m, nil, true
	}
	if timed.token != m.emergencyQuit.token {
		return m, nil, true
	}
	m.emergencyQuit.pending = false
	return m, nil, true
}

func (m shellModel) handleQuitShortcutMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok || !m.keys.Quit.matches(key) {
		return m, nil, false
	}
	m.quitting = true
	return m, tea.Quit, true
}

func (m shellModel) handleKeyboardOwnershipMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}
	switch m.keyboardOwnership() {
	case routeKeyboardOwnershipExclusiveLocalCapture:
		updated, cmd := m.updateActiveRoute(key)
		return updated, cmd, true
	case routeKeyboardOwnershipTextEntry:
		if !routeOwnsTextEntryKey(key) {
			return m, nil, false
		}
		updated, cmd := m.updateActiveRoute(key)
		return updated, cmd, true
	default:
		return m, nil, false
	}
}

func routeOwnsTextEntryKey(key tea.KeyMsg) bool {
	if isTypingKey(key) {
		return true
	}
	switch key.Type {
	case tea.KeyBackspace, tea.KeyDelete, tea.KeyEnter, tea.KeyEsc, tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown, tea.KeyHome, tea.KeyEnd:
		return true
	default:
		return false
	}
}

func (m shellModel) handleOverlayMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if !m.palette.IsOpen() && !m.sessions.IsOpen() && !m.quitConfirm.active {
		return m, nil, false
	}
	switch msg.(type) {
	case tea.MouseMsg, tea.KeyMsg:
		if m.quitConfirm.active {
			return m.handleQuitConfirmMessage(msg)
		}
		return m.handlePaletteMessage(msg)
	default:
		return m, nil, false
	}
}

func (m shellModel) handleQuitConfirmMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}
	switch {
	case key.Type == tea.KeyEnter || key.String() == "y":
		m.quitConfirm = shellQuitConfirmState{}
		m.quitting = true
		m.syncOverlayStack()
		m.restoreFocusAfterOverlayClose()
		return m, tea.Quit, true
	case key.Type == tea.KeyEsc || key.String() == "n":
		m.quitConfirm = shellQuitConfirmState{}
		m.syncOverlayStack()
		m.restoreFocusAfterOverlayClose()
		return m, nil, true
	default:
		return m, nil, true
	}
}

func (m shellModel) handleShellMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch typed := msg.(type) {
	case shellEmergencyQuitTimeoutMsg:
		return m.handleEmergencyQuitTimeoutMessage(typed)
	case paletteActionMsg:
		updated, cmd := m.applyPaletteAction(typed)
		return updated, cmd, true
	case sessionWorkspaceLoadedMsg:
		m.applySessionWorkspaceLoaded(typed)
		return m, nil, true
	case shellObjectIndexLoadedMsg:
		m.applyObjectIndexLoaded(typed)
		return m, nil, true
	case shellWatchPollMsg:
		return m, m.loadWatchPollCmd(), true
	case shellWatchTransportLoadedMsg:
		return m.handleShellWatchLoaded(typed)
	case shellActivityTickMsg:
		return m.handleShellActivityTick()
	case tea.MouseMsg:
		updated, cmd := m.handleMouse(typed)
		return updated, cmd, true
	case tea.KeyMsg:
		updated, cmd := m.handleKey(typed)
		return updated, cmd, true
	default:
		return m, nil, false
	}
}

func (m shellModel) handleShellWatchLoaded(msg shellWatchTransportLoadedMsg) (tea.Model, tea.Cmd, bool) {
	m.applyWatchTransport(msg)
	activity := m.watch.projection.Activity
	m.toasts.SetActivity(activity.State == shellActivityStateRunning)
	if activity.State == shellActivityStateRunning {
		return m, tea.Batch(m.watchPollTickAfterCmd(m.watch.nextPollDelay()), m.activityTickCmd()), true
	}
	return m, m.watchPollTickAfterCmd(m.watch.nextPollDelay()), true
}

func (m shellModel) handleShellActivityTick() (tea.Model, tea.Cmd, bool) {
	if m.watch.projection.Activity.State != shellActivityStateRunning {
		return m, nil, true
	}
	m.activityFrame = (m.activityFrame + 1) % 8
	return m, m.activityTickCmd(), true
}

func (m shellModel) handleWindowSize(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	typed, ok := msg.(tea.WindowSizeMsg)
	if !ok {
		return m, nil, false
	}
	prev := m.breakpoint()
	m.width = typed.Width
	m.height = typed.Height
	if prev != shellBreakpointNarrow && m.breakpoint() == shellBreakpointNarrow {
		m.narrowSidebarOn = false
		m.narrowInspectOn = false
		m.restoreFocusAfterOverlayClose()
	}
	if prev == shellBreakpointNarrow && m.breakpoint() != shellBreakpointNarrow {
		m.narrowSidebarOn = false
		m.narrowInspectOn = false
		m.restoreFocusAfterOverlayClose()
	}
	m.syncOverlayStack()
	m.normalizeFocusForLayout()
	updated, cmd := m.updateActiveRoute(routeViewportResizeMsg{Width: m.width, Height: m.height})
	return updated, cmd, true
}

func (m shellModel) handlePaletteMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if m.sessions.IsOpen() {
		if updated, cmd, handled := m.handleSessionQuickSwitchMessage(msg); handled {
			return updated, cmd, true
		}
	}
	if !m.palette.IsOpen() {
		return m, nil, false
	}
	switch typed := msg.(type) {
	case tea.MouseMsg:
		return m.handlePaletteMouse(typed)
	case tea.KeyMsg:
		return m.handlePaletteKey(typed)
	default:
		return m, nil, false
	}
}

func (m shellModel) handleSessionQuickSwitchMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if _, ok := msg.(tea.MouseMsg); ok {
		return m, nil, true
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}
	switch {
	case m.keys.SessionQuickSwitchClose.matches(key):
		m.sessions = m.sessions.Close()
		m.syncOverlayStack()
		m.restoreFocusAfterOverlayClose()
		return m, nil, true
	case m.keys.SessionQuickSwitchPick.matches(key):
		sid := strings.TrimSpace(m.sessions.SelectedSessionID())
		m.sessions = m.sessions.Close()
		m.syncOverlayStack()
		m.restoreFocusAfterOverlayClose()
		if sid == "" {
			return m, nil, true
		}
		updated, cmd := m.activateSessionFromSidebarByID(sid)
		return updated, cmd, true
	case m.keys.SessionQuickSwitchNext.matches(key):
		m.sessions = m.sessions.Next()
		return m, nil, true
	case m.keys.SessionQuickSwitchPrev.matches(key):
		m.sessions = m.sessions.Prev()
		return m, nil, true
	case key.Type == tea.KeyBackspace || key.Type == tea.KeyDelete:
		m.sessions = m.sessions.DeleteQueryRune()
		return m, nil, true
	case isTypingKey(key):
		m.sessions = m.sessions.AppendQuery(key.String())
		return m, nil, true
	default:
		return m, nil, true
	}
}

func (m shellModel) handlePaletteMouse(mouse tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	updatedPalette, routeMsg, changed := m.palette.UpdateMouse(mouse, m.paletteStartY(), m.width)
	m.palette = updatedPalette
	m.syncOverlayStack()
	m.restoreFocusAfterOverlayClose()
	if changed {
		return m, func() tea.Msg { return routeMsg }, true
	}
	return m, nil, true
}

func (m shellModel) handlePaletteKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	updatedPalette, routeMsg, changed := m.palette.Update(key, m.keys)
	m.palette = updatedPalette
	m.syncOverlayStack()
	m.restoreFocusAfterOverlayClose()
	if changed {
		return m, func() tea.Msg { return routeMsg }, true
	}
	return m, nil, true
}

func (m *shellModel) syncOverlayStack() {
	ids := make([]shellOverlayID, 0, 4)
	if m.palette.IsOpen() {
		ids = append(ids, overlayIDQuickJump)
	}
	if m.sessions.IsOpen() {
		ids = append(ids, overlayIDSessions)
	}
	if m.leader.Active() {
		ids = append(ids, overlayIDLeader)
	}
	if m.quitConfirm.active {
		ids = append(ids, overlayIDQuitConfirm)
	}
	if m.breakpoint() == shellBreakpointNarrow {
		if m.narrowSidebarOn {
			ids = append(ids, overlayIDSidebar)
		}
		if m.narrowInspectOn {
			ids = append(ids, overlayIDInspector)
		}
	}
	m.overlayManager.Replace(ids...)
	if len(ids) == 0 {
		m.overlays = nil
		return
	}
	m.overlays = m.overlayManager.Stack()
}
