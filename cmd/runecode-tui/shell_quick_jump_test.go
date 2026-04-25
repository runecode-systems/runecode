package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestShellQuickJumpSupportsSingleStrokeGitRoutes(t *testing.T) {
	m := newShellModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	if cmd != nil {
		t.Fatal("did not expect retired quick-jump command for git setup")
	}
	shell := updated.(shellModel)
	if shell.currentRouteID() != routeChat {
		t.Fatalf("expected route to remain %q, got %q", routeChat, shell.currentRouteID())
	}

	updated, cmd = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	if cmd != nil {
		t.Fatal("did not expect retired quick-jump command for git remote")
	}
	shell = updated.(shellModel)
	if shell.currentRouteID() != routeChat {
		t.Fatalf("expected route to remain %q, got %q", routeChat, shell.currentRouteID())
	}
}
