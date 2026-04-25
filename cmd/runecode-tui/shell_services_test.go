package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestShellFocusManagerCyclesBySidebarVisibility(t *testing.T) {
	mgr := newShellFocusManager(focusNav)
	mgr.Next(shellLayoutPlan{NavigationVisible: true}, false)
	if mgr.Current() != focusContent {
		t.Fatalf("expected focusContent, got %v", mgr.Current())
	}
	mgr.Next(shellLayoutPlan{NavigationVisible: true, InspectorVisible: true}, false)
	if mgr.Current() != focusInspector {
		t.Fatalf("expected focusInspector when inspector visible, got %v", mgr.Current())
	}
	mgr.Next(shellLayoutPlan{NavigationVisible: false, InspectorVisible: false}, false)
	if mgr.Current() != focusContent {
		t.Fatalf("expected focusContent when only main available, got %v", mgr.Current())
	}
	mgr.Next(shellLayoutPlan{NavigationVisible: true}, true)
	if mgr.Current() != focusPalette {
		t.Fatalf("expected focusPalette when palette open, got %v", mgr.Current())
	}
}

func TestShellOverlayManagerStackOperations(t *testing.T) {
	var mgr shellOverlayManager
	mgr.Open("quick-jump")
	mgr.Open("help")
	mgr.Open("quick-jump")
	if len(mgr.Stack()) != 2 {
		t.Fatalf("expected deduplicated overlays, got %v", mgr.Stack())
	}
	mgr.Close("quick-jump")
	if mgr.Contains("quick-jump") {
		t.Fatal("expected quick-jump to be closed")
	}
}

func TestShellCommandRegistryRegisterAndExecute(t *testing.T) {
	r := newShellCommandRegistry()
	called := false
	r.Register(shellCommand{ID: "test.cmd", Title: "Test", Run: func(_ *shellModel) { called = true }})
	model := newShellModel()
	cmd := r.Execute("test.cmd", &model)
	if cmd != nil {
		t.Fatal("expected no follow-up command")
	}
	if !called {
		t.Fatal("expected registered command callback")
	}
}

func TestWorkbenchStateStoreRoundTrip(t *testing.T) {
	store := &memoryWorkbenchStateStore{}
	state := workbenchLocalState{
		SidebarVisible:     true,
		InspectorVisible:   true,
		InspectorMode:      presentationStructured,
		ThemePreset:        themePresetDusk,
		LastRouteID:        routeRuns,
		LastSessionID:      "session-1",
		LastSessionByWS:    map[string]string{"ws-1": "session-1"},
		PinnedSessions:     []workbenchSessionRef{{WorkspaceID: "ws-1", SessionID: "session-1"}},
		RecentSessions:     []workbenchSessionRef{{WorkspaceID: "ws-2", SessionID: "session-2"}, {WorkspaceID: "ws-1", SessionID: "session-1"}},
		RecentObjects:      []workbenchObjectRef{{Kind: "session", ID: "session-1", WorkspaceID: "ws-1", SessionID: "session-1"}},
		ViewedActivity:     map[string]string{"session-1": "2026-01-01T00:00:00Z"},
		SidebarPaneRatio:   0.24,
		InspectorPaneRatio: 0.31,
		SidebarCollapsed:   false,
		InspectorCollapsed: true,
	}
	store.Write("broker_local_api:test", state)
	got := store.Read("broker_local_api:test")
	if !reflect.DeepEqual(got, state) {
		t.Fatalf("expected %+v, got %+v", state, got)
	}
}

func TestShellToastServiceLatestMessage(t *testing.T) {
	svc := newShellToastService()
	svc.Push(toastInfo, "loaded")
	svc.Push(toastWarn, "warned")
	if got := svc.Latest(); got != "WARN: warned" {
		t.Fatalf("expected WARN latest toast, got %q", got)
	}
}

func TestMemoryClipboardServiceSupportsOSC52Integration(t *testing.T) {
	buf := &bytes.Buffer{}
	clip := &memoryClipboardService{osc52: true, osc52Writer: buf}
	clip.Copy("hello")
	if got := clip.Last(); got != "hello" {
		t.Fatalf("expected last copied text, got %q", got)
	}
	if got := clip.IntegrationHint(); got != "shell clipboard + OSC52" {
		t.Fatalf("expected osc52 hint, got %q", got)
	}
	if buf.Len() == 0 {
		t.Fatal("expected osc52 escape sequence output")
	}
}

func TestOSC52EnabledByEnv(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "1", want: true},
		{value: "true", want: true},
		{value: "on", want: true},
		{value: "yes", want: true},
		{value: "", want: false},
		{value: "0", want: false},
		{value: "false", want: false},
	}
	for _, tc := range tests {
		if got := osc52EnabledByEnv(tc.value); got != tc.want {
			t.Fatalf("osc52EnabledByEnv(%q)=%t want %t", tc.value, got, tc.want)
		}
	}
}

func TestNewShellClipboardServiceRequiresExplicitOptIn(t *testing.T) {
	old := os.Getenv("RUNECODE_TUI_OSC52")
	t.Cleanup(func() {
		_ = os.Setenv("RUNECODE_TUI_OSC52", old)
	})

	_ = os.Unsetenv("RUNECODE_TUI_OSC52")
	clip := newShellClipboardService()
	mem, ok := clip.(*memoryClipboardService)
	if !ok {
		t.Fatalf("expected memory clipboard service, got %T", clip)
	}
	if mem.osc52 {
		t.Fatal("expected osc52 disabled by default")
	}
}

func TestNormalizeBrokerTargetAlias(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "workspace/prod", want: "workspace-prod"},
		{in: "WS_01", want: "ws_01"},
		{in: "!!!", want: "local-default"},
	}
	for _, tc := range tests {
		if got := normalizeBrokerTargetAlias(tc.in); got != tc.want {
			t.Fatalf("normalizeBrokerTargetAlias(%q)=%q want %q", tc.in, got, tc.want)
		}
	}

	veryLong := strings.Repeat("a", 200)
	if got := len(normalizeBrokerTargetAlias(veryLong)); got > 128 {
		t.Fatalf("expected alias length <= 128, got %d", got)
	}
}

func TestCenteredOverlayBlockBoundedClipsToHeight(t *testing.T) {
	body := strings.Repeat("line\n", 40)
	rendered := centeredOverlayBlockBounded(overlayIDQuickJump, body, 80, 6)
	if got := lipgloss.Height(rendered); got != 6 {
		t.Fatalf("expected bounded overlay height=6, got %d", got)
	}
	if got := lipgloss.Width(rendered); got != 80 {
		t.Fatalf("expected bounded overlay width=80, got %d", got)
	}
}

func TestCenteredOverlayContentBoundsUseInnerContentArea(t *testing.T) {
	start, end := centeredOverlayContentBounds(80)
	if start != 2 || end != 69 {
		t.Fatalf("expected content bounds [2,69], got [%d,%d]", start, end)
	}
}
