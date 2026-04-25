package main

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestLeaderPreferencePersistsAcrossRestore(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scope := "broker_local_api:leader-persist"

	m := newShellModel()
	m.workbench = store
	m.workbenchScope = scope
	if err := m.configureLeaderKey("comma"); err != nil {
		t.Fatalf("configureLeaderKey(comma) error = %v", err)
	}

	restored := newShellModel()
	restored.workbench = store
	restored.workbenchScope = scope
	restored.restoreWorkbenchState()

	if got := restored.leaderKeyConfig; got != "comma" {
		t.Fatalf("expected restored leader config comma, got %q", got)
	}
	if got := restored.keys.LeaderStart.label(); got != "," {
		t.Fatalf("expected restored leader binding ',' got %q", got)
	}
}

func TestLeaderPreferenceInvalidValueRejectedAndPersistedValueUnchanged(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scope := "broker_local_api:leader-invalid"
	m := newShellModel()
	m.workbench = store
	m.workbenchScope = scope

	if err := m.configureLeaderKey("comma"); err != nil {
		t.Fatalf("configureLeaderKey(comma) error = %v", err)
	}
	before := store.Read(scope)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	for _, r := range "set leader enter" {
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		shell = updated.(shellModel)
	}
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = updated.(shellModel)

	if got := shell.leaderKeyConfig; got != "comma" {
		t.Fatalf("expected invalid value to keep leader config comma, got %q", got)
	}
	if got := shell.keys.LeaderStart.label(); got != "," {
		t.Fatalf("expected invalid value to keep leader binding ',' got %q", got)
	}
	after := store.Read(scope)
	if got := strings.TrimSpace(after.LeaderKey); got != "comma" {
		t.Fatalf("expected persisted leader key to remain comma, got %q", got)
	}
	if before.LeaderKey != after.LeaderKey {
		t.Fatalf("expected persisted leader key unchanged; before=%q after=%q", before.LeaderKey, after.LeaderKey)
	}
}

func TestLeaderPreferenceDefaultResetsToSpace(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scope := "broker_local_api:leader-default"
	m := newShellModel()
	m.workbench = store
	m.workbenchScope = scope

	if err := m.configureLeaderKey("backslash"); err != nil {
		t.Fatalf("configureLeaderKey(backslash) error = %v", err)
	}
	if err := m.configureLeaderKey("default"); err != nil {
		t.Fatalf("configureLeaderKey(default) error = %v", err)
	}

	if got := m.leaderKeyConfig; got != "space" {
		t.Fatalf("expected default reset to space, got %q", got)
	}
	if got := m.keys.LeaderStart.label(); got != "space" {
		t.Fatalf("expected default binding to be space, got %q", got)
	}
	if got := strings.TrimSpace(store.Read(scope).LeaderKey); got != "space" {
		t.Fatalf("expected persisted default leader key space, got %q", got)
	}
}

func TestLeaderPreferenceScopedByLogicalTarget(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scopeA := "broker_local_api:leader-scope-a"
	scopeB := "broker_local_api:leader-scope-b"

	a := newShellModel()
	a.workbench = store
	a.workbenchScope = scopeA
	if err := a.configureLeaderKey("comma"); err != nil {
		t.Fatalf("configureLeaderKey(comma) error = %v", err)
	}

	b := newShellModel()
	b.workbench = store
	b.workbenchScope = scopeB
	b.restoreWorkbenchState()
	if got := b.leaderKeyConfig; got != "space" {
		t.Fatalf("expected separate scope default leader space, got %q", got)
	}

	a2 := newShellModel()
	a2.workbench = store
	a2.workbenchScope = scopeA
	a2.restoreWorkbenchState()
	if got := a2.leaderKeyConfig; got != "comma" {
		t.Fatalf("expected scope A leader comma after restore, got %q", got)
	}
}

func TestLeaderPreferenceInvalidPersistedValueFallsBackToDefaultWithWarning(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	scope := "broker_local_api:leader-invalid-persisted"
	store.Write(scope, workbenchLocalState{LeaderKey: "enter"})

	m := newShellModel()
	m.workbench = store
	m.workbenchScope = scope
	m.restoreWorkbenchState()

	if got := m.leaderKeyConfig; got != "space" {
		t.Fatalf("expected invalid persisted leader to fall back to space, got %q", got)
	}
	if got := m.leaderKeyInvalid; got != "enter" {
		t.Fatalf("expected invalid persisted leader marker enter, got %q", got)
	}
	if got := m.toasts.Latest(); !strings.Contains(got, "Persisted leader key invalid") {
		t.Fatalf("expected warning toast for invalid persisted leader, got %q", got)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		t.Fatal("expected space leader start to execute synchronously")
	}
	restored := updated.(shellModel)
	if !restored.leader.Active() {
		t.Fatal("expected default space leader to remain usable after invalid persisted value")
	}
}
