package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestShellSidebarRenderShowsSingleSelectedRouteAndActiveMarker(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}
	m.syncSidebarCursorToLocation()

	v := m.renderSidebar()
	if strings.Count(v, "> o 3 Runs") != 1 {
		t.Fatalf("expected one selected runs row, got %q", v)
	}
	if strings.Count(v, "* o 3 Runs") != 0 {
		t.Fatalf("did not expect active marker on selected row, got %q", v)
	}
	if strings.Count(v, "> o 2 Chat") != 0 {
		t.Fatalf("did not expect non-cursor route selected, got %q", v)
	}
	if strings.Contains(v, "> 3 Runs") || strings.Contains(v, "* 3 Runs") {
		t.Fatalf("expected sidebar to avoid retired bare quick-jump hints, got %q", v)
	}
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, "> o 3 Runs") && lipgloss.Width(line) < 12 {
			t.Fatalf("expected selected row to render as full-width line, got %q", line)
		}
	}
}

func TestShellSidebarRouteHintsMatchRouteJumpActionLeaderBindings(t *testing.T) {
	m := newShellModel()
	m.width = 320
	v := m.renderSidebar()

	for _, route := range m.routes {
		action, ok := m.actions.definitionByID("route.jump." + string(route.ID))
		if !ok {
			t.Fatalf("expected route jump action for %q", route.ID)
		}
		path := normalizeActionPath(action.LeaderPath)
		if len(path) == 0 {
			t.Fatalf("expected leader path for route %q", route.ID)
		}
		want := strings.Join(path, " ") + " " + route.Label
		if !strings.Contains(v, want) {
			t.Fatalf("expected sidebar discoverability to render action-graph route hint %q, got %q", want, v)
		}
	}
	if strings.Contains(v, "\n  0 Git Setup") || strings.Contains(v, "\n  - Git Remote") {
		t.Fatalf("expected sidebar discoverability not to imply retired single-stroke route jumps, got %q", v)
	}
}

func TestShellMoveSidebarCursorHonorsDeltaMagnitude(t *testing.T) {
	m := newShellModel()
	entries := m.sidebarEntries()
	if len(entries) < 3 {
		t.Fatalf("expected enough sidebar entries for cursor movement test, got %d", len(entries))
	}

	m.sidebarCursor = 0
	m.moveSidebarCursor(2)
	if m.sidebarCursor != 2 {
		t.Fatalf("expected cursor to move forward by 2, got %d", m.sidebarCursor)
	}

	m.moveSidebarCursor(-3)
	want := len(entries) - 1
	if m.sidebarCursor != want {
		t.Fatalf("expected cursor to wrap backward to %d, got %d", want, m.sidebarCursor)
	}
}
