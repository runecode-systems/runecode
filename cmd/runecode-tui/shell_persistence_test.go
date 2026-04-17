package main

import (
	"os"
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

	if m.currentID != routeRuns {
		t.Fatalf("expected restored route runs, got %q", m.currentID)
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
