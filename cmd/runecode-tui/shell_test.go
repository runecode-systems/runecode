package main

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestShellQuickJumpSetsRouteAndFocusAndBackstack(t *testing.T) {
	m := newShellModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if cmd == nil {
		t.Fatal("expected route activation command")
	}
	shell := updated.(shellModel)
	if shell.currentRouteID() != routeRuns {
		t.Fatalf("expected route %q, got %q", routeRuns, shell.currentRouteID())
	}
	if shell.focus != focusContent {
		t.Fatalf("expected focusContent, got %v", shell.focus)
	}
	if len(shell.history) != 1 || shell.history[0].Primary.RouteID != routeChat {
		t.Fatalf("expected history [chat], got %+v", shell.history)
	}
}

func TestShellSidebarVisibleByDefaultAndToggle(t *testing.T) {
	m := newShellModel()
	m.width = 120
	if !m.effectiveSidebarVisible() {
		t.Fatal("expected sidebar visible by default")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
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

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
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
	if shell.currentRouteID() != routeChat {
		t.Fatalf("expected inspect to preserve primary route %q, got %q", routeChat, shell.currentRouteID())
	}
	if !shell.narrowInspectOn {
		t.Fatal("expected narrow inspector overlay on after inspect verb")
	}
	if !shell.overlayManager.Contains(overlayIDInspector) {
		t.Fatalf("expected inspector overlay in stack, got %v", shell.overlays)
	}
	if shell.focus != focusInspector {
		t.Fatalf("expected focusInspector, got %v", shell.focus)
	}
}

func TestShellOverlayCloseRestoresPreviousFocusTarget(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.setFocus(focusNav)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	if shell.focus != focusPalette {
		t.Fatalf("expected overlay focus while palette open, got %v", shell.focus)
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEsc})
	shell = updated.(shellModel)
	if shell.focus != focusNav {
		t.Fatalf("expected focus restored to nav, got %v", shell.focus)
	}
}

func TestShellFocusTraversalIncludesInspectorRegionOnWideLayout(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	shell := updated.(shellModel)
	if shell.focus != focusContent {
		t.Fatalf("expected first tab to focus content, got %v", shell.focus)
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyTab})
	shell = updated.(shellModel)
	if shell.focus != focusInspector {
		t.Fatalf("expected second tab to focus inspector, got %v", shell.focus)
	}
}

func TestShellViewCompositorPlacesPanesHorizontally(t *testing.T) {
	m := newShellModel()
	m.width = 150
	v := m.View()
	if !strings.Contains(v, "┌") || !strings.Contains(v, "┐") {
		t.Fatalf("expected lipgloss pane borders in compositor output, got %q", v)
	}
	if strings.Contains(v, "││") {
		t.Fatalf("expected single-width shared pane separators without doubled borders, got %q", v)
	}
	if !strings.Contains(v, "Main pane") || !strings.Contains(v, "Sidebar") {
		t.Fatalf("expected main+sidebar pane titles in compositor output, got %q", v)
	}
}

func TestRenderShellPanePreservesInternalBlankLines(t *testing.T) {
	pane := renderShellPane(shellPaneSpec{Title: "Test", Body: "line one\n\nline two", Width: 40, Height: 8, Focused: false, Border: shellPaneBorder{Top: true, Bottom: true, Left: true, Right: true}})
	if !strings.Contains(pane, "line one") || !strings.Contains(pane, "line two") {
		t.Fatalf("expected pane body content preserved, got %q", pane)
	}
	foundBlankInterior := false
	for _, line := range strings.Split(pane, "\n") {
		if strings.HasPrefix(line, "│") && strings.HasSuffix(line, "│") && strings.Trim(line, "│ ") == "" {
			foundBlankInterior = true
			break
		}
	}
	if !foundBlankInterior {
		t.Fatalf("expected preserved blank content row inside pane body, got %q", pane)
	}
}

func TestRenderShellPanesDoesNotDoubleConstrainRenderedRow(t *testing.T) {
	m := newShellModel()
	m.width = 150
	surface := m.activeShellSurface()
	layout := m.planShellLayout(surface)
	row := m.renderShellPanes(surface, layout)
	if strings.Contains(row, "││") {
		t.Fatalf("expected rendered pane row not to be re-split into double separators, got %q", row)
	}
	if got := lipgloss.Height(row); got < layout.Regions.Main.Height {
		t.Fatalf("expected pane row height at least %d, got %d", layout.Regions.Main.Height, got)
	}
	if strings.Contains(row, "┌") && strings.Count(row, "┌") < 2 {
		t.Fatalf("expected preserved multi-pane framing, got %q", row)
	}
}

func TestShellBackKeyPopsBackstack(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	shell := updated.(shellModel)
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	shell = updated.(shellModel)
	if shell.currentRouteID() != routeRuns {
		t.Fatalf("expected runs route, got %q", shell.currentRouteID())
	}
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	shell = updated.(shellModel)
	if shell.currentRouteID() != routeChat {
		t.Fatalf("expected back to chat, got %q", shell.currentRouteID())
	}
}

func TestShellMouseClickSidebarOpensRoute(t *testing.T) {
	m := newShellModel()
	m.width = 120
	startY, _ := m.sidebarYRange()
	updated, _ := m.Update(tea.MouseMsg{X: 2, Y: startY + 1, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell := updated.(shellModel)
	if shell.currentRouteID() != routeChat {
		t.Fatalf("expected route %q, got %q", routeChat, shell.currentRouteID())
	}
}

func TestShellMousePressDoesNotDuplicateSidebarNavigationHistory(t *testing.T) {
	m := newShellModel()
	m.width = 180
	m.sidebarRatio = 0.3
	startY, _ := m.sidebarYRange()

	updated, _ := m.Update(tea.MouseMsg{X: 2, Y: startY + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	shell := updated.(shellModel)
	if len(shell.history) != 0 {
		t.Fatalf("expected mouse press not to mutate history, got %+v", shell.history)
	}

	updated, _ = shell.Update(tea.MouseMsg{X: 2, Y: startY + 2, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell = updated.(shellModel)
	if len(shell.history) != 1 {
		t.Fatalf("expected single history entry after release, got %+v", shell.history)
	}
}

func TestShellMouseHitboxUsesPlannedSidebarWidth(t *testing.T) {
	m := newShellModel()
	m.width = 180
	m.sidebarRatio = 0.3
	startY, _ := m.sidebarYRange()

	updated, _ := m.Update(tea.MouseMsg{X: 30, Y: startY + 2, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell := updated.(shellModel)
	if len(shell.history) != 1 {
		t.Fatalf("expected sidebar click inside planned width to navigate once, got history=%+v", shell.history)
	}
}

func TestShellSidebarCursorMovesVerticallyAndEnterOpensSession(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: []brokerapi.SessionSummary{
		{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}},
		{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}},
	}})
	m.setFocus(focusNav)
	start := m.sidebarCursor

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	shell := updated.(shellModel)
	if shell.sidebarCursor != start+1 {
		t.Fatalf("expected sidebar cursor to move down, got %d from %d", shell.sidebarCursor, start)
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyDown})
	shell = updated.(shellModel)
	if shell.sidebarCursor != start+2 {
		t.Fatalf("expected sidebar cursor to move down with arrow, got %d", shell.sidebarCursor)
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyUp})
	shell = updated.(shellModel)
	if shell.sidebarCursor != start+1 {
		t.Fatalf("expected sidebar cursor to move up with arrow, got %d", shell.sidebarCursor)
	}

	for i := 0; i < len(shell.routes)-1; i++ {
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		shell = updated.(shellModel)
	}
	if entry, ok := shell.selectedSidebarEntry(); !ok || entry.Kind != sidebarEntrySession {
		t.Fatalf("expected cursor at a session entry, got ok=%t kind=%q", ok, entry.Kind)
	}

	updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected route/session activation command from enter")
	}
	shell = updated.(shellModel)
	if shell.activeSessionID != "session-1" {
		t.Fatalf("expected active session switched to session-1, got %q", shell.activeSessionID)
	}
	if shell.currentRouteID() != routeChat {
		t.Fatalf("expected session open to keep chat route, got %q", shell.currentRouteID())
	}
}

func TestShellSidebarRenderShowsSingleSelectedRouteAndActiveMarker(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}
	m.syncSidebarCursorToLocation()

	v := m.renderSidebar()
	if strings.Count(v, "> 3 Runs") != 1 {
		t.Fatalf("expected one selected runs row, got %q", v)
	}
	if strings.Count(v, "* 3 Runs") != 0 {
		t.Fatalf("did not expect active marker on selected row, got %q", v)
	}
	if strings.Count(v, "> 2 Chat") != 0 {
		t.Fatalf("did not expect non-cursor route selected, got %q", v)
	}
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, "> 3 Runs") && lipgloss.Width(line) < 12 {
			t.Fatalf("expected selected row to render as full-width line, got %q", line)
		}
	}
}

func TestShellSelectionModeDisablesMouseInteractions(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}
	m.selectionMode = true
	startY, _ := m.sidebarYRange()
	updated, _ := m.Update(tea.MouseMsg{X: 2, Y: startY + 1, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell := updated.(shellModel)
	if shell.currentRouteID() != routeRuns {
		t.Fatalf("expected route unchanged while selection mode on, got %q", shell.currentRouteID())
	}
}

func TestSessionQuickSwitchOverlayConsumesMouseWithoutClickThrough(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: []brokerapi.SessionSummary{{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}}}})
	m.sessions = m.sessions.Open(m.sessionItems)
	startY, _ := m.sidebarYRange()

	updated, _ := m.Update(tea.MouseMsg{X: 2, Y: startY + 1, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell := updated.(shellModel)
	if len(shell.history) != 0 {
		t.Fatalf("expected no click-through history mutation while session overlay open, got %+v", shell.history)
	}
	if !shell.sessions.IsOpen() {
		t.Fatal("expected session overlay to remain open after consumed mouse event")
	}
}

func TestShellViewRendersShellSurfaces(t *testing.T) {
	m := newShellModel()
	m.width = 150
	v := m.View()
	for _, want := range []string{"Runecode TUI α shell", "Path:", "History:", "Main pane", "Sidebar", "Bottom strip", "Status:"} {
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

func TestShellViewFillsViewportWithRootSurface(t *testing.T) {
	m := newShellModel()
	m.width = 110
	m.height = 32

	v := m.View()
	if got := lipgloss.Width(v); got != 110 {
		t.Fatalf("expected full-frame width=110, got %d", got)
	}
	if got := lipgloss.Height(v); got != 32 {
		t.Fatalf("expected full-frame height=32, got %d", got)
	}
}

func TestShellOverlayRemainsVisibleWithinViewport(t *testing.T) {
	m := newShellModel()
	m.width = 100
	m.height = 28

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	v := shell.View()
	if got := lipgloss.Height(v); got != 28 {
		t.Fatalf("expected full-frame height=28 with overlay open, got %d", got)
	}
	if !strings.Contains(v, "Workbench Command Surface") {
		t.Fatalf("expected palette overlay content in viewport, got %q", v)
	}
	for _, want := range []string{"Overlay", "Matches"} {
		if !strings.Contains(v, want) {
			t.Fatalf("expected styled overlay affordance %q in viewport, got %q", want, v)
		}
	}
}

func TestShellOverlayNarrowViewportKeepsFrameBounds(t *testing.T) {
	m := newShellModel()
	m.width = 42
	m.height = 16

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	v := shell.View()

	if got := lipgloss.Width(v); got != 42 {
		t.Fatalf("expected full-frame width=42 with overlay open, got %d", got)
	}
	if got := lipgloss.Height(v); got != 16 {
		t.Fatalf("expected full-frame height=16 with overlay open, got %d", got)
	}
	for _, line := range strings.Split(v, "\n") {
		if lipgloss.Width(line) > 42 {
			t.Fatalf("expected overlay/frame line width <= 42, got %d in %q", lipgloss.Width(line), line)
		}
	}
	if !strings.Contains(v, "Workbench Command Surface") {
		t.Fatalf("expected palette overlay content in narrow viewport, got %q", v)
	}
}

func TestShellOverlayBodyHeightClampsToViewportBudget(t *testing.T) {
	m := newShellModel()
	m.width = 52
	m.height = 12

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	surface := shell.activeShellSurface()
	layout := shell.planShellLayout(surface)
	overlay, overlayHeight := shell.overlayBodyWithHeight(surface, layout, shell.height)

	if overlayHeight != 4 {
		t.Fatalf("expected overlay height=4 from viewport budget (12-8), got %d", overlayHeight)
	}
	if got := lipgloss.Height(overlay); got != 4 {
		t.Fatalf("expected rendered overlay block height=4, got %d", got)
	}
	for _, line := range strings.Split(overlay, "\n") {
		if lipgloss.Width(line) > 52 {
			t.Fatalf("expected overlay line width <= 52, got %d in %q", lipgloss.Width(line), line)
		}
	}
}

func TestNarrowInspectorOverlayShowsEmptyStateWhenNoDetailSelected(t *testing.T) {
	m := newShellModel()
	m.width = 80
	m.height = 24
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	m.routeModels[routeChat] = &captureRouteModel{
		id:    routeChat,
		title: "Capture",
		surface: routeSurface{
			Regions:      routeSurfaceRegions{Inspector: routeSurfaceRegion{Title: "Empty Inspector", Body: ""}},
			Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: true}},
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	shell := updated.(shellModel)
	if !shell.narrowInspectOn {
		t.Fatal("expected narrow inspector overlay enabled")
	}
	v := shell.View()
	if !strings.Contains(v, "No inspector item selected.") {
		t.Fatalf("expected empty-state narrow inspector overlay, got %q", v)
	}
}

func TestShellChromeSanitizesBreadcrumbsHistoryAndActivityLabels(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.height = 32
	m.history = []shellWorkbenchLocation{{Primary: shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "run", ID: "run-1\nspoof"}}}}
	m.watch.projection.Activity = shellActivitySemantics{State: shellActivityStateRunning, Active: shellActivityFocus{Kind: "session\nkind", ID: "session-1\nspoof"}}
	surface := routeSurface{Chrome: routeSurfaceChrome{Breadcrumbs: []string{"Home", "Runs\nSpoof"}}}

	breadcrumbs := m.renderBreadcrumbs(surface)
	history := m.renderHistory()
	sync := m.renderSyncHealth()
	running := m.renderRunningIndicator()

	for _, got := range []string{breadcrumbs, history, sync, running} {
		if strings.Contains(got, "\n") || strings.Contains(got, "\r") {
			t.Fatalf("expected sanitized single-line shell chrome, got %q", got)
		}
	}
}

func TestShellToastRemainsVisibleWithinViewport(t *testing.T) {
	m := newShellModel()
	m.width = 100
	m.height = 28
	m.toasts.Push(toastInfo, "Sidebar visibility changed.")

	v := m.View()
	if got := lipgloss.Height(v); got != 28 {
		t.Fatalf("expected full-frame height=28 with toast visible, got %d", got)
	}
	if !strings.Contains(v, "Toast: INFO: Sidebar visibility changed.") {
		t.Fatalf("expected toast content in viewport, got %q", v)
	}
}

func TestHelpRenderedFromRealKeyBindings(t *testing.T) {
	help := renderHelp(defaultShellKeyMap(), false)
	for _, want := range []string{"q/ctrl+c", "tab", "S", "b/alt+left", "\\"} {
		if !strings.Contains(help, want) {
			t.Fatalf("expected %q in help, got %q", want, help)
		}
	}
}

func TestShellModelProvidersRouteReceivesLowercaseSForSetup(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	provider := newProviderSetupRouteModel(routeDefinition{ID: routeProviders, Label: "Model Providers"}, &fakeBrokerClient{})
	providerUpdated, providerCmd := provider.Update(routeActivatedMsg{RouteID: routeProviders})
	if providerCmd == nil {
		t.Fatal("expected provider activation load command")
	}
	providerUpdated, _ = providerUpdated.Update(providerCmd())
	m.routeModels[routeProviders] = providerUpdated

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("expected provider setup command from route-local lowercase s")
	}
	updated, _ = updated.Update(cmd())
	shell := updated.(shellModel)
	if !shell.sidebarVisible {
		t.Fatal("expected lowercase s on model providers route not to toggle sidebar")
	}
	providerModel, ok := shell.routeModels[routeProviders].(providerSetupRouteModel)
	if !ok {
		t.Fatalf("expected provider setup route model, got %T", shell.routeModels[routeProviders])
	}
	if !providerModel.entryActive {
		t.Fatal("expected provider route to enter masked secret setup mode after lowercase s")
	}
}

func TestShellProviderSecretEntrySuppressesGlobalThemeShortcut(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}
	beforeTheme := m.themePreset

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd != nil {
		t.Fatal("did not expect shell theme command while provider secret entry active")
	}
	shell := updated.(shellModel)
	if shell.themePreset != beforeTheme {
		t.Fatalf("expected theme preset unchanged during provider secret entry, got %q want %q", shell.themePreset, beforeTheme)
	}
	provider := shell.routeModels[routeProviders].(providerSetupRouteModel)
	if string(provider.secretRunes) != "t" {
		t.Fatalf("expected provider secret input to capture 't', got %q", string(provider.secretRunes))
	}
	if shell.currentRouteID() != routeProviders {
		t.Fatalf("expected current route to remain %q, got %q", routeProviders, shell.currentRouteID())
	}
}

func TestShellProviderSecretEntrySuppressesQuickJumpDigits(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if cmd != nil {
		t.Fatal("did not expect quick-jump command while provider secret entry active")
	}
	shell := updated.(shellModel)
	if shell.currentRouteID() != routeProviders {
		t.Fatalf("expected route to remain %q during provider secret entry, got %q", routeProviders, shell.currentRouteID())
	}
	provider := shell.routeModels[routeProviders].(providerSetupRouteModel)
	if string(provider.secretRunes) != "1" {
		t.Fatalf("expected provider secret input to capture '1', got %q", string(provider.secretRunes))
	}
}

func TestShellProviderSecretEntryAllowsEscapeToCancel(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true, secretRunes: []rune("secret")}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatal("did not expect shell overlay command while provider secret entry active")
	}
	shell := updated.(shellModel)
	provider := shell.routeModels[routeProviders].(providerSetupRouteModel)
	if provider.entryActive {
		t.Fatal("expected provider secret entry to cancel on escape")
	}
	if len(provider.secretRunes) != 0 {
		t.Fatalf("expected provider secret input cleared on escape, got %q", string(provider.secretRunes))
	}
	if !strings.Contains(provider.status, "Secret entry cancelled") {
		t.Fatalf("expected cancellation status, got %q", provider.status)
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
	m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}

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

func TestShellPaletteCommandToggleSelectionModeReturnsMouseCaptureCmd(t *testing.T) {
	m := newShellModel()
	updated, cmd := m.Update(paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: "shell.toggle_selection_mode"}})
	if cmd == nil {
		t.Fatal("expected mouse capture command for palette command toggle")
	}
	shell := updated.(shellModel)
	if !shell.selectionMode {
		t.Fatal("expected selection mode enabled by palette command")
	}
}

func TestShellEscapeCloseNarrowOverlaysResetsHiddenNavFocus(t *testing.T) {
	m := newShellModel()
	m.width = 80
	m.narrowSidebarOn = true
	m.focusManager.Set(focusNav)
	m.focus = m.focusManager.Current()
	m.syncOverlayStack()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	shell := updated.(shellModel)
	if shell.focus != focusContent {
		t.Fatalf("expected focus reset to content, got %v", shell.focus)
	}
}

func TestShellTextEntryGuardsGlobalQuitShortcut(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
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
	m.location.Primary = shellObjectLocation{RouteID: routeDashboard, Object: workbenchObjectRef{Kind: "route", ID: string(routeDashboard)}}
	opened, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := opened.(shellModel)
	if !shell.palette.IsOpen() {
		t.Fatal("expected palette open")
	}

	updated, _ := shell.Update(shellWatchTransportLoadedMsg{
		Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_terminal", Seq: 1, Terminal: true, TerminalStatus: "completed", Run: &brokerapi.RunSummary{RunID: "run-1"}}}},
	})
	shell = updated.(shellModel)
	if shell.watch.projection.Health.State != shellSyncStateHealthy {
		t.Fatalf("expected healthy sync after watch update with palette open, got %s", shell.watch.projection.Health.State)
	}
}

func TestShellBottomStripSelectionHintUsesCtrlT(t *testing.T) {
	m := newShellModel()
	m.width = 150
	v := m.View()
	if !strings.Contains(v, "Selection mode off") {
		t.Fatalf("expected updated selection hint in bottom strip, got %q", v)
	}
}

func TestShellScrollDispatchTargetsRouteViewportState(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}
	runs := newRunsRouteModel(routeDefinition{ID: routeRuns, Label: "Runs"}, &fakeBrokerClient{})
	runsUpdated, runsCmd := runs.Update(routeActivatedMsg{RouteID: routeRuns})
	if runsCmd == nil {
		t.Fatal("expected runs load command")
	}
	runsUpdated, _ = runsUpdated.Update(runsCmd())
	m.routeModels[routeRuns] = runsUpdated
	m.focusManager.Set(focusContent)
	m.focus = m.focusManager.Current()
	shell := m
	var updated tea.Model

	updated, _ = shell.Update(routeViewportResizeMsg{Width: 120, Height: 28})
	shell = updated.(shellModel)
	updated, _ = shell.Update(routeViewportScrollMsg{Region: routeRegionInspector, Delta: 2})
	shell = updated.(shellModel)
	shell.setFocus(focusInspector)

	before := shell.activeShellSurface().Regions.Inspector.Body
	if !strings.Contains(before, "offset=2") {
		t.Fatalf("expected baseline offset=2, got %q", before)
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	shell = updated.(shellModel)
	after := shell.activeShellSurface().Regions.Inspector.Body
	if !strings.Contains(after, "offset=3") {
		t.Fatalf("expected pgdown to dispatch route inspector scroll, got %q", after)
	}
	if strings.Contains(after, "scroll=") {
		t.Fatalf("expected shell-global scroll retired from status, got %q", after)
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
	m.client = &fakeBrokerClient{}
	loadedMsg, ok := m.loadObjectIndexCmd()().(shellObjectIndexLoadedMsg)
	if !ok {
		t.Fatalf("expected shellObjectIndexLoadedMsg, got %T", m.loadObjectIndexCmd()())
	}
	updated, _ := m.Update(loadedMsg)
	m = updated.(shellModel)

	entries := m.buildPaletteEntries()
	joined := ""
	for _, e := range entries {
		joined += e.Label + "\n"
	}
	for _, want := range []string{"open session session-1", "inspect run run-1", "inspect approval ap-1", "inspect artifact sha256:bbbb", "inspect audit sha256:"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in palette labels, got %q", want, joined)
		}
	}
}

func TestShellPaletteNavigationFromFreshLaunchUsesShellIndex(t *testing.T) {
	m := newShellModel()
	m.client = &fakeBrokerClient{}
	runs := m.routeModels[routeRuns].(runsRouteModel)
	if len(runs.runs) != 0 {
		t.Fatalf("expected runs model uninitialized at fresh launch, got %d items", len(runs.runs))
	}

	loadedMsg, ok := m.loadObjectIndexCmd()().(shellObjectIndexLoadedMsg)
	if !ok {
		t.Fatalf("expected shellObjectIndexLoadedMsg, got %T", m.loadObjectIndexCmd()())
	}
	updated, _ := m.Update(loadedMsg)
	shell := updated.(shellModel)

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell = updated.(shellModel)
	for _, r := range "run-1" {
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		shell = updated.(shellModel)
	}
	selected, ok := shell.palette.SelectedEntry()
	if !ok {
		t.Fatal("expected a selected palette entry")
	}
	if !strings.Contains(selected.Label, "inspect run run-1") {
		t.Fatalf("expected selected run entry after query, got %q", selected.Label)
	}

	updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected palette pick command")
	}
	shell = updated.(shellModel)
	paletteMsg := cmd()
	updated, cmd = shell.Update(paletteMsg)
	if cmd == nil {
		t.Fatal("expected route activation command for run navigation")
	}
	shell = updated.(shellModel)
	if shell.currentRouteID() != routeRuns {
		t.Fatalf("expected navigation to %q, got %q", routeRuns, shell.currentRouteID())
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
	if shell.currentRouteID() != routeAudit {
		t.Fatalf("expected route %q, got %q", routeAudit, shell.currentRouteID())
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
	if shell.currentRouteID() != routeAudit {
		t.Fatalf("expected route %q, got %q", routeAudit, shell.currentRouteID())
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
	if shell.currentRouteID() != routeStatus {
		t.Fatalf("expected route %q, got %q", routeStatus, shell.currentRouteID())
	}

	updated, _ = shell.Update(paletteActionMsg{Verb: verbBack})
	shell = updated.(shellModel)
	if shell.currentRouteID() != routeChat {
		t.Fatalf("expected back to %q, got %q", routeChat, shell.currentRouteID())
	}
}

func TestShellWatchManagerUpdatesRoutesAndSyncHealth(t *testing.T) {
	m := newShellModel()
	m.width = 160
	m.height = 90
	m.location.Primary = shellObjectLocation{RouteID: routeDashboard, Object: workbenchObjectRef{Kind: "route", ID: string(routeDashboard)}}
	updated, _ := m.Update(shellWatchTransportLoadedMsg{
		Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{
			{EventType: "run_watch_snapshot", Seq: 1, Run: &brokerapi.RunSummary{RunID: "run-1"}},
			{EventType: "run_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
		}},
		Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{
			{EventType: "approval_watch_snapshot", Seq: 1, Approval: &brokerapi.ApprovalSummary{ApprovalID: "ap-1"}},
			{EventType: "approval_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
		}},
		Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{
			{EventType: "session_watch_snapshot", Seq: 1, Session: &brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1"}}},
			{EventType: "session_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
		}},
	})

	shell := updated.(shellModel)
	if shell.watch.projection.Health.State != shellSyncStateHealthy {
		t.Fatalf("expected healthy sync, got %s", shell.watch.projection.Health.State)
	}
	view := shell.View()
	mustContainAll(t, view,
		"Sync health:",
		"sync=healthy",
		"last_event=run_watch_terminal subject=run-1 status=completed",
		"event=session_watch_terminal subject=session-1 status=completed",
	)
}

func TestShellWatchManagerRendersDisconnectedHealth(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(shellWatchTransportLoadedMsg{
		Run:      shellWatchRunTransportResult{Err: errors.New("local_ipc_dial_error")},
		Approval: shellWatchApprovalTransportResult{Err: errors.New("local_ipc_dial_error")},
		Session:  shellWatchSessionTransportResult{Err: errors.New("local_ipc_dial_error")},
	})
	shell := updated.(shellModel)
	if shell.watch.projection.Health.State != shellSyncStateDisconnected {
		t.Fatalf("expected disconnected sync, got %s", shell.watch.projection.Health.State)
	}
	if !strings.Contains(shell.View(), "sync=disconnected") {
		t.Fatalf("expected disconnected indicator in view, got %q", shell.View())
	}
}
