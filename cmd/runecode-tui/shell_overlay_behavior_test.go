package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLeaderOverlayConsumesMouseWithoutClickThrough(t *testing.T) {
	m := newShellModel()
	m.width = 120

	updated, _ := m.Update(paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeRuns}})
	shell := updated.(shellModel)
	if shell.currentRouteID() != routeRuns {
		t.Fatalf("expected baseline route %q, got %q", routeRuns, shell.currentRouteID())
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell = updated.(shellModel)
	if !shell.leader.Active() {
		t.Fatal("expected leader mode active")
	}

	startY, _ := shell.sidebarYRange()
	updated, _ = shell.Update(tea.MouseMsg{X: 2, Y: startY + 1, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell = updated.(shellModel)
	if shell.currentRouteID() != routeRuns {
		t.Fatalf("expected leader overlay to block sidebar click-through; route changed to %q", shell.currentRouteID())
	}
	if len(shell.history) != 1 {
		t.Fatalf("expected no extra navigation history while leader overlay active, got %+v", shell.history)
	}
	if !shell.leader.Active() {
		t.Fatal("expected leader mode to remain active after consumed mouse event")
	}
}
