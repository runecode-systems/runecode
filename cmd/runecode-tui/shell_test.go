package main

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestShellQuickJumpSetsRouteAndFocusAndBackstack(t *testing.T) {
	m := newShellModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if cmd == nil {
		t.Fatal("expected route activation command")
	}
	shell := updated.(shellModel)
	if shell.currentID != routeRuns {
		t.Fatalf("expected route %q, got %q", routeRuns, shell.currentID)
	}
	if shell.focus != focusContent {
		t.Fatalf("expected focusContent, got %v", shell.focus)
	}
	if len(shell.backstack) != 1 || shell.backstack[0] != routeChat {
		t.Fatalf("expected backstack [chat], got %v", shell.backstack)
	}
}

func TestShellSidebarVisibleByDefaultAndToggle(t *testing.T) {
	m := newShellModel()
	m.width = 120
	if !m.effectiveSidebarVisible() {
		t.Fatal("expected sidebar visible by default")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	shell := updated.(shellModel)
	if shell.effectiveSidebarVisible() {
		t.Fatal("expected sidebar hidden after toggle")
	}
}

func TestShellSidebarForcedHiddenOnNarrowBreakpoint(t *testing.T) {
	m := newShellModel()
	m.width = 80
	if got := m.breakpoint(); got != shellBreakpointNarrow {
		t.Fatalf("expected narrow breakpoint, got %s", got)
	}
	if m.effectiveSidebarVisible() {
		t.Fatal("expected sidebar hidden on narrow breakpoint")
	}
}

func TestShellBreakpointsStandardized(t *testing.T) {
	cases := []struct {
		width int
		want  shellBreakpoint
	}{
		{width: 70, want: shellBreakpointNarrow},
		{width: 100, want: shellBreakpointMedium},
		{width: 150, want: shellBreakpointWide},
	}
	for _, tc := range cases {
		if got := shellBreakpointForWidth(tc.width); got != tc.want {
			t.Fatalf("width=%d got=%s want=%s", tc.width, got, tc.want)
		}
	}
}

func TestShellOverlayStackTracksPalette(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	if len(shell.overlays) != 1 || shell.overlays[0] != overlayIDQuickJump {
		t.Fatalf("expected quick-jump overlay, got %v", shell.overlays)
	}
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEsc})
	shell = updated.(shellModel)
	if len(shell.overlays) != 0 {
		t.Fatalf("expected cleared overlays, got %v", shell.overlays)
	}
}

func TestShellNarrowSidebarToggleUsesOverlayNavigation(t *testing.T) {
	m := newShellModel()
	m.width = 80
	if m.navigationSurfaceVisible() {
		t.Fatal("expected nav surface hidden on narrow without overlay")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	shell := updated.(shellModel)
	if !shell.narrowSidebarOn {
		t.Fatal("expected narrow sidebar overlay on")
	}
	if !shell.navigationSurfaceVisible() {
		t.Fatal("expected nav surface visible via narrow overlay")
	}
	if !shell.overlayManager.Contains(overlayIDSidebar) {
		t.Fatalf("expected sidebar overlay in stack, got %v", shell.overlays)
	}
}

func TestShellNarrowInspectVerbOpensInspectorOverlay(t *testing.T) {
	m := newShellModel()
	m.width = 80

	updated, _ := m.Update(paletteActionMsg{Verb: verbInspect, Target: paletteTarget{Kind: "run", RunID: "run-1"}})
	shell := updated.(shellModel)
	if shell.currentID != routeRuns {
		t.Fatalf("expected route %q, got %q", routeRuns, shell.currentID)
	}
	if !shell.narrowInspectOn {
		t.Fatal("expected narrow inspector overlay on after inspect verb")
	}
	if !shell.overlayManager.Contains(overlayIDInspector) {
		t.Fatalf("expected inspector overlay in stack, got %v", shell.overlays)
	}
}

func TestShellBackKeyPopsBackstack(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	shell := updated.(shellModel)
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	shell = updated.(shellModel)
	if shell.currentID != routeRuns {
		t.Fatalf("expected runs route, got %q", shell.currentID)
	}
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	shell = updated.(shellModel)
	if shell.currentID != routeChat {
		t.Fatalf("expected back to chat, got %q", shell.currentID)
	}
}

func TestShellMouseClickSidebarOpensRoute(t *testing.T) {
	m := newShellModel()
	m.width = 120
	startY, _ := m.sidebarYRange()
	updated, _ := m.Update(tea.MouseMsg{X: 2, Y: startY + 1, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell := updated.(shellModel)
	if shell.currentID != routeChat {
		t.Fatalf("expected route %q, got %q", routeChat, shell.currentID)
	}
}

func TestShellSelectionModeDisablesMouseInteractions(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.currentID = routeRuns
	m.selectionMode = true
	startY, _ := m.sidebarYRange()
	updated, _ := m.Update(tea.MouseMsg{X: 2, Y: startY + 1, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell := updated.(shellModel)
	if shell.currentID != routeRuns {
		t.Fatalf("expected route unchanged while selection mode on, got %q", shell.currentID)
	}
}

func TestShellViewRendersShellSurfaces(t *testing.T) {
	m := newShellModel()
	m.width = 150
	v := m.View()
	for _, want := range []string{"Top status", "Breadcrumbs:", "Backstack:", "Main pane", "Sidebar", "Bottom strip", "Status:"} {
		if !strings.Contains(v, want) {
			t.Fatalf("expected %q in view, got %q", want, v)
		}
	}
	for _, want := range []string{"┌────────────────", "FOCUS"} {
		if !strings.Contains(v, want) {
			t.Fatalf("expected pane framing affordance %q in view, got %q", want, v)
		}
	}
}

func TestHelpRenderedFromRealKeyBindings(t *testing.T) {
	help := renderHelp(defaultShellKeyMap(), false)
	for _, want := range []string{"q/ctrl+c", "tab", "s", "b/alt+left"} {
		if !strings.Contains(help, want) {
			t.Fatalf("expected %q in help, got %q", want, help)
		}
	}
}

func TestShellClipboardCopiesCurrentBreadcrumbIdentity(t *testing.T) {
	m := newShellModel()
	m.width = 150
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	shell := updated.(shellModel)
	clip, ok := shell.clipboard.(*memoryClipboardService)
	if !ok {
		t.Fatalf("expected memory clipboard service, got %T", shell.clipboard)
	}
	if strings.TrimSpace(clip.Last()) == "" {
		t.Fatal("expected copied identity to be non-empty")
	}
}

func TestShellCopyRouteActionCopiesInspectorAction(t *testing.T) {
	m := newShellModel()
	m.width = 150
	runs := m.routeModels[routeRuns].(runsRouteModel)
	runs.active = &brokerapi.RunDetail{Summary: brokerapi.RunSummary{RunID: "run-1", WorkspaceID: "ws-1", LifecycleState: "active", BackendKind: "workspace"}}
	m.routeModels[routeRuns] = runs
	m.currentID = routeRuns

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	shell := updated.(shellModel)
	clip, ok := shell.clipboard.(*memoryClipboardService)
	if !ok {
		t.Fatalf("expected memory clipboard service, got %T", shell.clipboard)
	}
	if strings.TrimSpace(clip.Last()) == "" {
		t.Fatal("expected copied route action text to be non-empty")
	}
}

func TestShellSelectionModeToggleReflectsInView(t *testing.T) {
	m := newShellModel()
	m.width = 150
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	shell := updated.(shellModel)
	if !shell.selectionMode {
		t.Fatal("expected selection mode enabled")
	}
	v := shell.View()
	if !strings.Contains(v, "selection=on") {
		t.Fatalf("expected selection mode state in view, got %q", v)
	}
}

func TestShellTextEntryGuardsGlobalQuitShortcut(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.currentID = routeChat
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("did not expect shell quit command while composing")
	}
	shell := updated.(shellModel)
	if shell.quitting {
		t.Fatal("expected shell to remain active while composing")
	}
	chat = shell.routeModels[routeChat].(chatRouteModel)
	if !strings.Contains(chat.composer.Value(), "q") {
		t.Fatalf("expected compose buffer to include typed key, got %q", chat.composer.Value())
	}
}

func TestShellOverlayDoesNotBlockWatchUpdates(t *testing.T) {
	m := newShellModel()
	m.currentID = routeDashboard
	opened, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := opened.(shellModel)
	if !shell.palette.IsOpen() {
		t.Fatal("expected palette open")
	}

	updated, _ := shell.Update(shellWatchLoadedMsg{
		runEvents: []brokerapi.RunWatchEvent{{EventType: "run_watch_terminal", Seq: 1, Terminal: true, TerminalStatus: "completed", Run: &brokerapi.RunSummary{RunID: "run-1"}}},
	})
	shell = updated.(shellModel)
	if shell.watchHealth.State != shellSyncStateHealthy {
		t.Fatalf("expected healthy sync after watch update with palette open, got %s", shell.watchHealth.State)
	}
}

func TestShellBottomStripSelectionHintUsesCtrlT(t *testing.T) {
	m := newShellModel()
	m.width = 150
	v := m.View()
	if !strings.Contains(v, "ctrl+t") {
		t.Fatalf("expected ctrl+t in selection hint, got %q", v)
	}
}

func TestShellCommandRegistryExecutesToggleSidebar(t *testing.T) {
	m := newShellModel()
	m.width = 150
	if !m.sidebarVisible {
		t.Fatal("expected sidebar visible initially")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	shell := updated.(shellModel)
	if shell.sidebarVisible {
		t.Fatal("expected sidebar hidden after command execution")
	}
}

func TestShellPaletteEntriesAreObjectAware(t *testing.T) {
	m := newShellModel()
	m.sessionItems = []brokerapi.SessionSummary{{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, LastActivityKind: "chat_message", LastActivityPreview: "hello"}}

	runs := m.routeModels[routeRuns].(runsRouteModel)
	runs.runs = []brokerapi.RunSummary{{RunID: "run-1", LifecycleState: "active", PendingApprovalCount: 1}}
	m.routeModels[routeRuns] = runs

	approvals := m.routeModels[routeApprovals].(approvalsRouteModel)
	approvals.items = []brokerapi.ApprovalSummary{{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate"}}
	m.routeModels[routeApprovals] = approvals

	artifacts := m.routeModels[routeArtifacts].(artifactsRouteModel)
	artifacts.items = []brokerapi.ArtifactSummary{{Reference: brokerapi.ArtifactSummary{}.Reference}}
	artifacts.items[0].Reference.Digest = "sha256:cccc"
	artifacts.items[0].Reference.DataClass = "diffs"
	m.routeModels[routeArtifacts] = artifacts

	audit := m.routeModels[routeAudit].(auditRouteModel)
	audit.timeline = []brokerapi.AuditTimelineViewEntry{{RecordDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)}, EventType: "run_state", Summary: "changed"}}
	m.routeModels[routeAudit] = audit

	entries := m.buildPaletteEntries()
	joined := ""
	for _, e := range entries {
		joined += e.Label + "\n"
	}
	for _, want := range []string{"open session session-1", "inspect run run-1", "inspect approval ap-1", "inspect artifact sha256:cccc", "inspect audit sha256:"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in palette labels, got %q", want, joined)
		}
	}
}

func TestShellPaletteNavigationWorksWhenSidebarHidden(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.sidebarVisible = false

	updated, cmd := m.Update(paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}})
	if cmd == nil {
		t.Fatal("expected command for route activation")
	}
	shell := updated.(shellModel)
	if shell.currentID != routeAudit {
		t.Fatalf("expected route %q, got %q", routeAudit, shell.currentID)
	}
	if shell.effectiveSidebarVisible() {
		t.Fatal("expected sidebar to remain hidden")
	}
}

func TestShellPaletteNavigationWorksWhenSidebarAutoCollapsedNarrow(t *testing.T) {
	m := newShellModel()
	m.width = 80

	updated, cmd := m.Update(paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}})
	if cmd == nil {
		t.Fatal("expected command for route activation")
	}
	shell := updated.(shellModel)
	if shell.currentID != routeAudit {
		t.Fatalf("expected route %q, got %q", routeAudit, shell.currentID)
	}
	if shell.effectiveSidebarVisible() {
		t.Fatal("expected sidebar to remain auto-hidden on narrow breakpoint")
	}
	if shell.navigationSurfaceVisible() {
		t.Fatal("expected nav surface hidden until narrow sidebar overlay is opened")
	}
}

func TestShellStandardizedBackVerb(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeStatus}})
	shell := updated.(shellModel)
	if shell.currentID != routeStatus {
		t.Fatalf("expected route %q, got %q", routeStatus, shell.currentID)
	}

	updated, _ = shell.Update(paletteActionMsg{Verb: verbBack})
	shell = updated.(shellModel)
	if shell.currentID != routeChat {
		t.Fatalf("expected back to %q, got %q", routeChat, shell.currentID)
	}
}

func TestShellWatchManagerUpdatesRoutesAndSyncHealth(t *testing.T) {
	m := newShellModel()
	m.currentID = routeDashboard
	updated, _ := m.Update(shellWatchLoadedMsg{
		runEvents: []brokerapi.RunWatchEvent{
			{EventType: "run_watch_snapshot", Seq: 1, Run: &brokerapi.RunSummary{RunID: "run-1"}},
			{EventType: "run_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
		},
		approvalEvents: []brokerapi.ApprovalWatchEvent{
			{EventType: "approval_watch_snapshot", Seq: 1, Approval: &brokerapi.ApprovalSummary{ApprovalID: "ap-1"}},
			{EventType: "approval_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
		},
		sessionEvents: []brokerapi.SessionWatchEvent{
			{EventType: "session_watch_snapshot", Seq: 1, Session: &brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1"}}},
			{EventType: "session_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
		},
	})

	shell := updated.(shellModel)
	if shell.watchHealth.State != shellSyncStateHealthy {
		t.Fatalf("expected healthy sync, got %s", shell.watchHealth.State)
	}
	view := shell.View()
	mustContainAll(t, view,
		"Shell sync health:",
		"sync=healthy",
		"last_event=run_watch_terminal subject=run-1 status=completed",
		"event=session_watch_terminal subject=session-1 status=completed",
	)
}

func TestShellWatchManagerRendersDisconnectedHealth(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(shellWatchLoadedMsg{
		runErr:      errors.New("local_ipc_dial_error"),
		approvalErr: errors.New("local_ipc_dial_error"),
		sessionErr:  errors.New("local_ipc_dial_error"),
	})
	shell := updated.(shellModel)
	if shell.watchHealth.State != shellSyncStateDisconnected {
		t.Fatalf("expected disconnected sync, got %s", shell.watchHealth.State)
	}
	if !strings.Contains(shell.View(), "sync=disconnected") {
		t.Fatalf("expected disconnected indicator in view, got %q", shell.View())
	}
}
