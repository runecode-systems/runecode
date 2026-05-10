package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
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

func TestShellSessionOverlayViewReusesCachedFrameWhileTyping(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.height = 32
	m.sessionItems = []brokerapi.SessionSummary{
		{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}},
		{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-2"}},
	}
	route := &countingRouteModel{id: routeDashboard}
	m.location.Primary = shellObjectLocation{RouteID: routeDashboard, Object: workbenchObjectRef{Kind: "route", ID: string(routeDashboard)}}
	m.routeModels[routeDashboard] = route

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	shell := updated.(shellModel)
	_ = shell.View()
	initialCalls := route.shellSurfaceCalls
	if initialCalls == 0 {
		t.Fatal("expected initial session overlay render to compute background frame")
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	shell = updated.(shellModel)
	beforeView := route.shellSurfaceCalls
	view := shell.View()
	if !strings.Contains(view, "session-2") {
		t.Fatalf("expected updated quick-switch overlay after typing, got %q", view)
	}
	if route.shellSurfaceCalls != beforeView {
		t.Fatalf("expected cached background frame during session typing, calls before=%d after=%d", beforeView, route.shellSurfaceCalls)
	}
}
