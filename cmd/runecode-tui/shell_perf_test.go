package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func BenchmarkShellViewEmpty(b *testing.B) {
	m := newShellModel()
	m.width = 120
	m.height = 40
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkShellViewWaitingSession(b *testing.B) {
	m := newShellModel()
	m.width = 120
	m.height = 40
	m.watch.reduction.sessions = map[string]brokerapi.SessionSummary{
		"session-wait": {
			Identity:          brokerapi.SessionIdentity{SessionID: "session-wait", WorkspaceID: "ws-1"},
			Status:            "active",
			WorkPosture:       "waiting",
			HasIncompleteTurn: true,
		},
	}
	m.watch.projection.Activity = shellActivitySemantics{State: shellActivityStateWaiting, Active: shellActivityFocus{Kind: "session", ID: "session-wait"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkShellWatchApply(b *testing.B) {
	msg := shellWatchTransportLoadedMsg{
		Run:      shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &brokerapi.RunSummary{RunID: "run-1", LifecycleState: "active"}}}},
		Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &brokerapi.ApprovalSummary{ApprovalID: "ap-1", Status: "pending"}}}},
		Session:  shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active"}}}},
	}
	m := newShellModel()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.applyWatchTransport(msg)
	}
}

func BenchmarkBuildPaletteEntries(b *testing.B) {
	m := newShellModel()
	m.objectIndex.ingestSessions([]brokerapi.SessionSummary{{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", LastActivityKind: "chat_message"}})
	m.objectIndex.ingestRuns([]brokerapi.RunSummary{{RunID: "run-1", WorkspaceID: "ws-1", LifecycleState: "active"}})
	m.objectIndex.ingestApprovals([]brokerapi.ApprovalSummary{{ApprovalID: "ap-1", Status: "pending"}})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.buildPaletteEntries()
	}
}

type blockingWorkbenchStateStore struct {
	memoryWorkbenchStateStore
	writeDelay time.Duration
	writes     int
}

func (s *blockingWorkbenchStateStore) Write(targetKey string, next workbenchLocalState) {
	time.Sleep(s.writeDelay)
	s.writes++
	s.memoryWorkbenchStateStore.Write(targetKey, next)
}

func TestShellSessionQuickSwitchTypingDoesNotPersistWorkbenchState(t *testing.T) {
	store := &blockingWorkbenchStateStore{writeDelay: 20 * time.Millisecond}
	m := newShellModelWithWorkbenchStore(store)
	m.sessionItems = []brokerapi.SessionSummary{
		{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}},
		{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-2"}},
	}
	m.sessions = m.sessions.Open(m.sessionItems)
	m.syncOverlayStack()
	beforeWrites := store.writes
	start := time.Now()
	for _, r := range "session-1" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(shellModel)
	}
	if elapsed := time.Since(start); elapsed >= store.writeDelay {
		t.Fatalf("expected typing path to avoid synchronous persistence delay, elapsed=%v writes before=%d after=%d", elapsed, beforeWrites, store.writes)
	}
	if store.writes != beforeWrites {
		t.Fatalf("expected no workbench persistence during quick-switch typing, writes before=%d after=%d", beforeWrites, store.writes)
	}
	if got := m.sessions.query; got != "session-1" {
		t.Fatalf("expected full quick-switch query retained, got %q", got)
	}
}

func TestFileWorkbenchStateStoreWritesAsynchronously(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "workbench-state.json")
	store := newFileWorkbenchStateStore(path)
	store.writeDebounce = 0
	store.writeDelay = func() { time.Sleep(25 * time.Millisecond) }
	state := workbenchLocalState{ThemePreset: themePresetHigh, LastRouteID: routeRuns}
	start := time.Now()
	store.Write("broker_local_api:test", state)
	if elapsed := time.Since(start); elapsed >= 25*time.Millisecond {
		t.Fatalf("expected asynchronous write to return before simulated write delay, took %v", elapsed)
	}
	store.Flush()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted workbench state: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected persisted workbench state file content")
	}
	got := store.Read("broker_local_api:test")
	if got.LastRouteID != routeRuns || got.ThemePreset != themePresetHigh {
		t.Fatalf("unexpected persisted state: %+v", got)
	}
}

func TestFileWorkbenchStateStoreRetainsDirtyStateOnWriteFailure(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "workbench-state.json")
	store := newFileWorkbenchStateStore(tmpDir)
	store.writeDebounce = 0
	store.Write("broker_local_api:test", workbenchLocalState{LastRouteID: routeRuns})
	store.Flush()
	if store.LastError() == nil {
		t.Fatal("expected write failure to be reported")
	}
	store.path = path
	store.Flush()
	if err := store.LastError(); err != nil {
		t.Fatalf("expected successful retry to clear write error, got %v", err)
	}
	if got := store.Read("broker_local_api:test"); got.LastRouteID != routeRuns {
		t.Fatalf("expected retry to persist state, got %+v", got)
	}
}

func BenchmarkShellSessionQuickSwitchTyping(b *testing.B) {
	m := newShellModel()
	m.sessionItems = []brokerapi.SessionSummary{
		{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, LastActivityPreview: "draft review"},
		{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-2"}, LastActivityPreview: "plan sync"},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		shell := m
		shell.sessions = shell.sessions.Open(shell.sessionItems)
		shell.syncOverlayStack()
		for _, r := range "session-1" {
			updated, _ := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			shell = updated.(shellModel)
		}
	}
}

func BenchmarkPaletteFilterTyping(b *testing.B) {
	base := newPaletteModel(makePaletteBenchmarkEntries(2000)).Open()
	query := "session-1999"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := base
		state.query = query
		state.appliedNeedle = ""
		state.filterRequest = 1
		_ = buildPaletteFilterResult(state.filterRequest, state.entriesVersion, state.query, state.normalizedEntries, state.appliedNeedle, state.matchIndexes)
	}
}

func BenchmarkPaletteOpenAndFastCopyTyping(b *testing.B) {
	base := newShellModel()
	base.width = 120
	base.height = 32
	base.paletteCache = makePaletteBenchmarkEntries(4000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shell := base
		updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
		shell = updated.(shellModel)
		if cmd != nil {
			updated, follow := shell.Update(cmd())
			shell = updated.(shellModel)
			if follow != nil {
				updated, _ = shell.Update(follow())
				shell = updated.(shellModel)
			}
		}
		for _, r := range "copy" {
			updated, cmd = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			shell = updated.(shellModel)
			if cmd != nil {
				updated, _ = shell.Update(cmd())
				shell = updated.(shellModel)
			}
		}
	}
}

func BenchmarkSessionSwitcherFilterTyping(b *testing.B) {
	sessions := makeSessionSwitcherBenchmarkSessions(2000)
	base := newSessionSwitcherModel().UpdateSessions(sessions)
	query := []rune("session-1999")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := base
		state.query = ""
		state.selectedIndex = 0
		state.rebuildMatches()
		for _, r := range query {
			state = state.AppendQuery(string(r))
		}
	}
}

func BenchmarkRenderPaletteLargeMatchWindow(b *testing.B) {
	m := newShellModel()
	m.width = 120
	m.palette = newPaletteModel(makePaletteBenchmarkEntries(2000)).Open()
	m.palette.selectedIndex = 1000
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderPalette()
	}
}

func TestShellPaletteFastTypingRetainsCopyQuery(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.height = 32
	m.paletteCache = []paletteEntry{
		{Index: 1, Label: "copy current identity", Description: "copy the current route identity", Search: "copy identity route"},
		{Index: 2, Label: "copy next route action", Description: "copy the next route action", Search: "copy next route action"},
		{Index: 3, Label: "open runs", Description: "jump to runs", Search: "open runs"},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	shell := updated.(shellModel)
	if cmd != nil {
		updated, follow := shell.Update(cmd())
		shell = updated.(shellModel)
		if follow != nil {
			updated, _ = shell.Update(follow())
			shell = updated.(shellModel)
		}
	}
	for _, r := range "copy" {
		updated, cmd = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		shell = updated.(shellModel)
		if cmd != nil {
			updated, _ = shell.Update(cmd())
			shell = updated.(shellModel)
		}
	}
	if got := shell.palette.query; got != "copy" {
		t.Fatalf("expected full palette query retained, got %q", got)
	}
	if shell.palette.MatchCount() == 0 {
		t.Fatal("expected copy query to retain matching palette entries")
	}
	selected, ok := shell.palette.SelectedEntry()
	if !ok || !strings.Contains(selected.Label, "copy") {
		t.Fatalf("expected selected copy entry after fast typing, got %+v ok=%v", selected, ok)
	}
}

func BenchmarkRenderSessionQuickSwitcherLargeWindow(b *testing.B) {
	m := newShellModel()
	m.width = 120
	m.sessions = newSessionSwitcherModel().Open(makeSessionSwitcherBenchmarkSessions(2000))
	m.sessions.selectedIndex = 1000
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderSessionQuickSwitcher()
	}
}

func makePaletteBenchmarkEntries(count int) []paletteEntry {
	entries := make([]paletteEntry, 0, count)
	for i := 0; i < count; i++ {
		entries = append(entries, paletteEntry{
			Index:       i + 1,
			Label:       fmt.Sprintf("command %04d", i),
			Description: fmt.Sprintf("open workspace ws-%d", i%250),
			Search:      fmt.Sprintf("session-%04d route-chat evidence-%04d", i, i),
		})
	}
	return entries
}

func makeSessionSwitcherBenchmarkSessions(count int) []brokerapi.SessionSummary {
	sessions := make([]brokerapi.SessionSummary, 0, count)
	for i := 0; i < count; i++ {
		sessions = append(sessions, brokerapi.SessionSummary{
			Identity:            brokerapi.SessionIdentity{SessionID: fmt.Sprintf("session-%04d", i), WorkspaceID: fmt.Sprintf("ws-%03d", i%250)},
			LastActivityKind:    "chat_message",
			LastActivityPreview: fmt.Sprintf("preview for session-%04d", i),
			Status:              "active",
		})
	}
	return sessions
}

type countingRouteModel struct {
	id                routeID
	shellSurfaceCalls int
}

func (m *countingRouteModel) ID() routeID { return m.id }

func (m *countingRouteModel) Title() string { return "Counting" }

func (m *countingRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	_ = msg
	return m, nil
}

func (m *countingRouteModel) View(width, height int, focus focusArea) string {
	_, _, _ = width, height, focus
	return "counting"
}

func (m *countingRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	_ = ctx
	m.shellSurfaceCalls++
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:      routeSurfaceRegion{Title: "Counting", Body: "main"},
			Inspector: routeSurfaceRegion{Title: "Inspector", Body: "detail"},
		},
		Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: true}},
	}
}

func TestShellLeaderOverlayViewReusesCachedFrameOnStep(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.height = 32
	route := &countingRouteModel{id: routeDashboard}
	m.location.Primary = shellObjectLocation{RouteID: routeDashboard, Object: workbenchObjectRef{Kind: "route", ID: string(routeDashboard)}}
	m.routeModels[routeDashboard] = route

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)
	_ = shell.View()
	firstCalls := route.shellSurfaceCalls
	if firstCalls == 0 {
		t.Fatal("expected initial overlay view to render route surface")
	}
	if shell.overlayFrameCache == nil || shell.overlayFrameCache.frame == "" {
		t.Fatal("expected overlay frame cache populated after initial leader view")
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	shell = updated.(shellModel)
	beforeSecondView := route.shellSurfaceCalls
	view := shell.View()
	if !strings.Contains(view, "Sequence: w") {
		t.Fatalf("expected updated leader sequence in cached view, got %q", view)
	}
	if route.shellSurfaceCalls != beforeSecondView {
		t.Fatalf("expected cached overlay view to avoid shell surface rerender, calls before=%d after=%d", beforeSecondView, route.shellSurfaceCalls)
	}
}

func BenchmarkShellLeaderOpenAndStep(b *testing.B) {
	base := newShellModel()
	base.width = 120
	base.height = 32
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shell := base
		updated, _ := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
		shell = updated.(shellModel)
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
		_ = updated.(shellModel)
	}
}

func BenchmarkShellViewLeaderOverlayCached(b *testing.B) {
	m := newShellModel()
	m.width = 120
	m.height = 32
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)
	_ = shell.View()
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	shell = updated.(shellModel)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = shell.View()
	}
}

func BenchmarkShellFocusTraversalHotPath(b *testing.B) {
	base := newShellModel()
	base.width = 150
	base.height = 40
	base.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}
	base.setFocus(focusContent)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shell := base
		updated, _ := shell.Update(tea.KeyMsg{Type: tea.KeyTab})
		shell = updated.(shellModel)
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
		_ = updated.(shellModel)
	}
}
