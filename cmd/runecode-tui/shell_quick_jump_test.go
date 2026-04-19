package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestShellQuickJumpSupportsSingleStrokeGitRoutes(t *testing.T) {
	m := newShellModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	if cmd == nil {
		t.Fatal("expected route activation command for git setup")
	}
	shell := updated.(shellModel)
	if shell.currentRouteID() != routeGitSetup {
		t.Fatalf("expected route %q, got %q", routeGitSetup, shell.currentRouteID())
	}

	updated, cmd = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	if cmd == nil {
		t.Fatal("expected route activation command for git remote")
	}
	shell = updated.(shellModel)
	if shell.currentRouteID() != routeGitRemote {
		t.Fatalf("expected route %q, got %q", routeGitRemote, shell.currentRouteID())
	}
}
