package main

import (
	"os"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestLogicalBrokerTargetKeyUsesStableAliasNotHostPath(t *testing.T) {
	t.Setenv("RUNECODE_TUI_BROKER_TARGET", "prod / west\\socket")
	got := logicalBrokerTargetKey()
	if got != "broker_local_api:prod-west-socket" {
		t.Fatalf("logicalBrokerTargetKey() = %q", got)
	}

	os.Unsetenv("RUNECODE_TUI_BROKER_TARGET")
	if got := logicalBrokerTargetKey(); got != "broker_local_api:local-default" {
		t.Fatalf("default logical target = %q", got)
	}
}

func TestShellRestoresPersistedLocalWorkbenchState(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scope := "broker_local_api:test"
	store.Write(scope, workbenchLocalState{
		SidebarVisible:     false,
		InspectorVisible:   false,
		InspectorMode:      presentationStructured,
		ThemePreset:        themePresetHigh,
		LastRouteID:        routeRuns,
		LastSessionID:      "session-2",
		LastSessionByWS:    map[string]string{"ws-2": "session-2"},
		PinnedSessions:     []workbenchSessionRef{{WorkspaceID: "ws-2", SessionID: "session-2"}},
		RecentSessions:     []workbenchSessionRef{{WorkspaceID: "ws-2", SessionID: "session-2"}},
		RecentObjects:      []workbenchObjectRef{{Kind: "run", ID: "run-9", WorkspaceID: "ws-2", SessionID: "session-2"}},
		ViewedActivity:     map[string]string{"session-2": "2026-03-01T00:00:00Z"},
		SidebarPaneRatio:   0.26,
		InspectorPaneRatio: 0.34,
		SidebarCollapsed:   true,
		InspectorCollapsed: true,
	})

	m := newShellModel()
	m.workbench = store
	m.workbenchScope = scope
	m.restoreWorkbenchState()

	if m.currentRouteID() != routeRuns {
		t.Fatalf("expected restored route runs, got %q", m.currentRouteID())
	}
	if m.sidebarVisible {
		t.Fatal("expected restored hidden sidebar")
	}
	if m.inspectorOn {
		t.Fatal("expected restored hidden inspector")
	}
	if m.preferredMode != presentationStructured {
		t.Fatalf("expected structured presentation, got %q", m.preferredMode)
	}
	if m.themePreset != themePresetHigh {
		t.Fatalf("expected high-contrast theme, got %q", m.themePreset)
	}
	if m.sidebarRatio != 0.26 || m.inspectorRatio != 0.34 {
		t.Fatalf("expected restored pane ratios, got sidebar=%v inspector=%v", m.sidebarRatio, m.inspectorRatio)
	}
	if !m.sidebarFolded || !m.inspectorFolded {
		t.Fatalf("expected restored collapsed panes, got sidebar=%t inspector=%t", m.sidebarFolded, m.inspectorFolded)
	}
}

func TestShellPersistStateUsesWorkspaceScopedSessionRefs(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scope := "broker_local_api:test"
	m := newShellModel()
	m.workbench = store
	m.workbenchScope = scope
	m.sessionItems = []brokerapi.SessionSummary{
		{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}},
		{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-2"}},
	}
	m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: m.sessionItems})
	m.activeSessionID = "session-2"
	m.trackRecentSession("session-2")
	m.trackRecentSession("session-1")
	m.pinnedSessions["session-2"] = struct{}{}
	m.lastSessionByWS["ws-2"] = "session-2"
	m.persistWorkbenchState()

	saved := store.Read(scope)
	if len(saved.PinnedSessions) != 1 || saved.PinnedSessions[0].WorkspaceID != "ws-2" || saved.PinnedSessions[0].SessionID != "session-2" {
		t.Fatalf("expected workspace/session pin ref, got %+v", saved.PinnedSessions)
	}
	if len(saved.RecentSessions) < 2 || saved.RecentSessions[0].WorkspaceID == "" {
		t.Fatalf("expected recent session refs with workspace IDs, got %+v", saved.RecentSessions)
	}
	if got := saved.LastSessionByWS["ws-2"]; got != "session-2" {
		t.Fatalf("expected last session by ws-2, got %q", got)
	}
}

func TestShellRestoreInvalidPersistedRouteFallsBackToChat(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scope := "broker_local_api:test-invalid-route"
	store.Write(scope, workbenchLocalState{LastRouteID: routeID("missing-route")})

	m := newShellModel()
	m.workbench = store
	m.workbenchScope = scope
	m.restoreWorkbenchState()

	if m.currentRouteID() != routeChat {
		t.Fatalf("expected invalid restored route to fall back to chat, got %q", m.currentRouteID())
	}
}

func TestShellRestoredPaneRatiosDriveCompositorGeometry(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scope := "broker_local_api:test-layout-geometry"
	store.Write(scope, workbenchLocalState{
		SidebarVisible:     true,
		InspectorVisible:   true,
		InspectorMode:      presentationStructured,
		LastRouteID:        routeRuns,
		SidebarPaneRatio:   0.35,
		InspectorPaneRatio: 0.25,
		SidebarCollapsed:   false,
		InspectorCollapsed: false,
	})

	m := newShellModel()
	m.workbench = store
	m.workbenchScope = scope
	m.restoreWorkbenchState()
	m.width = 200
	m.height = 50

	surface := m.activeShellSurface()
	layout := m.planShellLayout(surface)
	if !layout.NavigationVisible || !layout.InspectorVisible {
		t.Fatalf("expected sidebar+inspector visible, got sidebar=%t inspector=%t", layout.NavigationVisible, layout.InspectorVisible)
	}
	secondaryBudget := 200 - minimumMainPaneWidth(layout.Breakpoint)
	wantSidebar := paneWidthForRatio(secondaryBudget, 0.35, 20, secondaryBudget)
	remainingBudget := secondaryBudget - wantSidebar
	if remainingBudget < 0 {
		remainingBudget = 0
	}
	wantInspector := paneWidthForRatio(remainingBudget, 0.25, 24, remainingBudget)
	if got := layout.Regions.Sidebar.Width; got != wantSidebar {
		t.Fatalf("expected sidebar width %d from persisted ratio, got %d", wantSidebar, got)
	}
	if got := layout.Regions.Inspector.Width; got != wantInspector {
		t.Fatalf("expected inspector width %d from persisted ratio, got %d", wantInspector, got)
	}
	wantMain := 200 - wantSidebar - wantInspector
	if got := layout.Regions.Main.Width; got != wantMain {
		t.Fatalf("expected main width %d after pane allocation, got %d", wantMain, got)
	}
}

func TestShellRestoredCollapsedPanesAffectRenderedLayout(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scope := "broker_local_api:test-layout-collapse"
	store.Write(scope, workbenchLocalState{
		SidebarVisible:     true,
		InspectorVisible:   true,
		LastRouteID:        routeRuns,
		SidebarPaneRatio:   0.30,
		InspectorPaneRatio: 0.30,
		SidebarCollapsed:   true,
		InspectorCollapsed: true,
	})

	m := newShellModel()
	m.workbench = store
	m.workbenchScope = scope
	m.restoreWorkbenchState()
	m.width = 180
	m.height = 44

	surface := m.activeShellSurface()
	layout := m.planShellLayout(surface)
	if layout.NavigationVisible {
		t.Fatal("expected sidebar hidden by restored collapsed state")
	}
	if layout.InspectorVisible {
		t.Fatal("expected inspector hidden by restored collapsed state")
	}
	v := m.View()
	if contains := "Sidebar ("; hasSubstring(v, contains) {
		t.Fatalf("expected collapsed sidebar not rendered, still found %q", contains)
	}
	if contains := "Inspector pane"; hasSubstring(v, contains) {
		t.Fatalf("expected collapsed inspector not rendered, still found %q", contains)
	}
}

func hasSubstring(text string, want string) bool {
	return strings.Contains(text, want)
}

func TestShellCaptureInspectorVisibilityIgnoresRoutesWithoutInspectorSupport(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.inspectorOn = true
	m.location.Primary = shellObjectLocation{RouteID: routeDashboard, Object: workbenchObjectRef{Kind: "route", ID: string(routeDashboard)}}

	m.captureInspectorVisibilityFromActiveRoute()
	if !m.inspectorOn {
		t.Fatal("expected global inspector preference unchanged for route without inspector support")
	}
}
