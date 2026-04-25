package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestShellMouseClickSidebarHeaderGapDoesNotTriggerAdjacentEntry(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: []brokerapi.SessionSummary{{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}}}})
	beforeSession := m.activeSessionID
	beforeRoute := m.currentRouteID()
	beforeCursor := m.sidebarCursor
	startY, _ := m.sidebarYRange()
	headerGapY := startY + len(m.routes)

	updated, _ := m.Update(tea.MouseMsg{X: 2, Y: headerGapY, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell := updated.(shellModel)
	if len(shell.history) != 0 {
		t.Fatalf("expected sidebar header gap click not to navigate, got history=%+v", shell.history)
	}
	if shell.activeSessionID != beforeSession {
		t.Fatalf("expected sidebar header gap click not to change active session, got %q want %q", shell.activeSessionID, beforeSession)
	}
	if shell.currentRouteID() != beforeRoute {
		t.Fatalf("expected sidebar header gap click not to change route, got %q want %q", shell.currentRouteID(), beforeRoute)
	}
	if shell.sidebarCursor != beforeCursor {
		t.Fatalf("expected sidebar header gap click not to move cursor, got %d want %d", shell.sidebarCursor, beforeCursor)
	}
	if shell.focus != focusContent {
		t.Fatalf("expected gap click to fall back to content focus, got %v", shell.focus)
	}
}
