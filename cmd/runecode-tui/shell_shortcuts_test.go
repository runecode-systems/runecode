package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpRenderedFromRealKeyBindings(t *testing.T) {
	m := newShellModel()
	help := renderHelp(defaultShellKeyMap(), false, m.actions)
	for _, want := range []string{"ctrl+c", "space", "tab", "shift+tab", "ctrl+p", "ctrl+j"} {
		if !strings.Contains(help, want) {
			t.Fatalf("expected %q in help, got %q", want, help)
		}
	}
	for _, retired := range []string{"q/ctrl+c", "b/alt+left", "0-9", "pgup", "pgdown"} {
		if strings.Contains(help, retired) {
			t.Fatalf("did not expect retired shortcut %q in help, got %q", retired, help)
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

func TestShellProviderSecretEntryCapturesTypedRunes(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd != nil {
		t.Fatal("did not expect shell command while provider secret entry active")
	}
	shell := updated.(shellModel)
	provider := shell.routeModels[routeProviders].(providerSetupRouteModel)
	if string(provider.secretRunes) != "t" {
		t.Fatalf("expected provider secret input to capture 't', got %q", string(provider.secretRunes))
	}
	if shell.currentRouteID() != routeProviders {
		t.Fatalf("expected current route to remain %q, got %q", routeProviders, shell.currentRouteID())
	}
}

func TestShellProviderSecretEntryCapturesDigits(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if cmd != nil {
		t.Fatal("did not expect shell command while provider secret entry active")
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

type keyboardCaptureRouteModel struct {
	id        routeID
	keys      []string
	ownership routeKeyboardOwnership
}

func (m keyboardCaptureRouteModel) ID() routeID { return m.id }

func (m keyboardCaptureRouteModel) Title() string { return "Capture" }

func (m keyboardCaptureRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		m.keys = append(m.keys, key.String())
	}
	return m, nil
}

func (m keyboardCaptureRouteModel) View(width, height int, focus focusArea) string {
	_, _, _ = width, height, focus
	return "capture"
}

func (m keyboardCaptureRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	_ = ctx
	return routeSurface{}
}

func (m keyboardCaptureRouteModel) KeyboardOwnership() routeKeyboardOwnership {
	return m.ownership
}

func TestShellFormerPlainLetterGlobalsFlowToActiveRouteTyping(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeDashboard, Object: workbenchObjectRef{Kind: "route", ID: string(routeDashboard)}}
	m.routeModels[routeDashboard] = keyboardCaptureRouteModel{id: routeDashboard, ownership: routeKeyboardOwnershipNormal}

	for _, r := range []rune{'q', 'b', 'y', 't', 'j', 'k', '1'} {
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		if cmd != nil {
			t.Fatalf("did not expect shell command for retired key %q", string(r))
		}
		m = updated.(shellModel)
	}

	if m.currentRouteID() != routeDashboard {
		t.Fatalf("expected route unchanged after retired globals, got %q", m.currentRouteID())
	}
	captured := m.routeModels[routeDashboard].(keyboardCaptureRouteModel)
	if len(captured.keys) != 7 {
		t.Fatalf("expected route to capture 7 retired key presses, got %d (%v)", len(captured.keys), captured.keys)
	}
}

func TestShellCtrlPRemainsExplicitPaletteEntry(t *testing.T) {
	m := newShellModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if cmd != nil {
		t.Fatal("did not expect separate command while opening palette")
	}
	shell := updated.(shellModel)
	if !shell.palette.IsOpen() {
		t.Fatal("expected ctrl+p to open command palette")
	}
}

func TestShellTextEntryTypingDoesNotOpenLeaderOrPalette(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if cmd != nil {
		t.Fatal("did not expect shell command while composing")
	}
	shell := updated.(shellModel)
	if shell.leader.Active() {
		t.Fatal("expected leader mode to remain closed while composing")
	}
	if shell.palette.IsOpen() {
		t.Fatal("expected palette to remain closed while composing")
	}
	chat = shell.routeModels[routeChat].(chatRouteModel)
	if !strings.Contains(chat.composer.Value(), " ") {
		t.Fatalf("expected compose buffer to include typed space, got %q", chat.composer.Value())
	}

	updated, cmd = shell.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if cmd != nil {
		t.Fatal("did not expect shell command while composing")
	}
	shell = updated.(shellModel)
	if shell.palette.IsOpen() {
		t.Fatal("expected ctrl+p discovery to remain blocked during compose text entry")
	}
	if shell.leader.Active() {
		t.Fatal("expected ctrl+p not to enter leader mode")
	}

	updated, cmd = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("did not expect shell quit command while composing")
	}
	shell = updated.(shellModel)
	if shell.quitting {
		t.Fatal("expected shell to remain active while composing")
	}
	chat = shell.routeModels[routeChat].(chatRouteModel)
	if !strings.Contains(chat.composer.Value(), "q") {
		t.Fatalf("expected compose buffer to include typed key, got %q", chat.composer.Value())
	}
}

func TestShellChatComposeTextEntryBlocksCommandModeOpen(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	if cmd != nil {
		t.Fatal("did not expect shell command surface open command while composing")
	}
	shell := updated.(shellModel)
	if shell.palette.IsOpen() {
		t.Fatal("expected command surface to remain closed while compose text-entry owns typing")
	}
	if shell.commandMode.Active() {
		t.Fatal("expected shell command mode to remain closed while compose text-entry owns typing")
	}
	chat = shell.routeModels[routeChat].(chatRouteModel)
	if !strings.Contains(chat.composer.Value(), ":") {
		t.Fatalf("expected compose buffer to capture ':', got %q", chat.composer.Value())
	}
}

func TestShellColonOpensCommandModeNotPalette(t *testing.T) {
	m := newShellModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	if cmd != nil {
		t.Fatal("did not expect async command when entering command mode")
	}
	shell := updated.(shellModel)
	if !shell.commandMode.Active() {
		t.Fatal("expected command mode active after ':'")
	}
	if shell.palette.IsOpen() {
		t.Fatal("expected palette to remain closed when ':' enters command mode")
	}
}

func TestShellCommandModeRendersTypedInputInBottomStrip(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	for _, r := range "theme cycle" {
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		shell = updated.(shellModel)
	}
	surface := shell.activeShellSurface()
	bottom := shell.renderBottomStrip(surface)
	if !strings.Contains(bottom, ":theme cycle") {
		t.Fatalf("expected command draft in bottom strip, got %q", bottom)
	}
}

func TestShellCommandModeEscAbortsAndClearsDraft(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	shell = updated.(shellModel)

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEsc})
	shell = updated.(shellModel)
	if shell.commandMode.Active() {
		t.Fatal("expected command mode inactive after esc")
	}
	if shell.commandMode.draft != "" {
		t.Fatalf("expected cleared command draft after esc, got %q", shell.commandMode.draft)
	}
}

func TestShellCommandModeEnterExecutesShellCommand(t *testing.T) {
	m := newShellModel()
	m.width = 150
	if !m.sidebarVisible {
		t.Fatal("expected sidebar visible initially")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	for _, r := range "sidebar toggle" {
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		shell = updated.(shellModel)
	}
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = updated.(shellModel)
	if shell.sidebarVisible {
		t.Fatal("expected command-mode enter to execute shell sidebar toggle")
	}
	if shell.commandMode.Active() {
		t.Fatal("expected command mode closed after successful enter")
	}
}

func TestShellCommandModeAliasResolutionUsesActionDefinitions(t *testing.T) {
	m := newShellModel()
	m.width = 150
	cmd := m.commands.commands["shell.toggle_sidebar"]
	cmd.Aliases = []string{"sidebar flip"}
	m.commands.Register(cmd)
	m.actions = newShellActionGraph(m.routes, m.commands)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	for _, r := range "sidebar flip" {
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		shell = updated.(shellModel)
	}
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = updated.(shellModel)
	if shell.sidebarVisible {
		t.Fatal("expected action-graph alias to execute shell.toggle_sidebar")
	}
}

func TestShellCommandModeLongestAliasMatchHandlesMultiWordAliases(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	for _, r := range "open action center" {
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		shell = updated.(shellModel)
	}
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = updated.(shellModel)
	if shell.currentRouteID() != routeAction {
		t.Fatalf("expected longest multi-word alias to jump to %q, got %q", routeAction, shell.currentRouteID())
	}
}

func TestShellCommandModeSharedActionPreservesRegisteredAvailability(t *testing.T) {
	m := newShellModel()
	m.width = 150
	if _, ok := m.actions.resolveByID("shell.copy_route_action", m); !ok {
		t.Fatal("expected shared action to be available before exclusive capture starts")
	}
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}
	if _, ok := m.actions.resolveByID("shell.copy_route_action", m); ok {
		t.Fatal("expected shared action id to preserve registered availability gating during exclusive capture")
	}
}

func TestShellCommandModeParseAndExecutionErrorsRenderInBottomStrip(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = updated.(shellModel)
	parseBottom := shell.renderBottomStrip(shell.activeShellSurface())
	if !strings.Contains(parseBottom, "error:") || !strings.Contains(parseBottom, "empty command") {
		t.Fatalf("expected parse error in command-entry area, got %q", parseBottom)
	}

	for _, r := range "command shell.unknown" {
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		shell = updated.(shellModel)
	}
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = updated.(shellModel)
	execBottom := shell.renderBottomStrip(shell.activeShellSurface())
	if !strings.Contains(execBottom, "unknown command") || !strings.Contains(execBottom, "shell.unknown") {
		t.Fatalf("expected execution error in command-entry area, got %q", execBottom)
	}
}

func TestShellCommandModeUnknownCommandErrorDoesNotEchoRawDraft(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell := updated.(shellModel)
	for _, r := range "very-secret-token-value" {
		updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		shell = updated.(shellModel)
	}
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = updated.(shellModel)
	bottom := shell.renderBottomStrip(shell.activeShellSurface())
	if strings.Contains(bottom, "very-secret-token-value") {
		t.Fatalf("expected unknown command error not to echo raw draft, got %q", bottom)
	}
	if !strings.Contains(bottom, "unknown command") {
		t.Fatalf("expected generic unknown command error, got %q", bottom)
	}
}

func TestShellKeyboardOwnershipBlocksCommandModeOpenInExclusiveCapture(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	if cmd != nil {
		t.Fatal("did not expect shell command while provider secret entry owns keyboard")
	}
	shell := updated.(shellModel)
	if shell.commandMode.Active() {
		t.Fatal("expected command mode blocked by exclusive local capture")
	}
	provider := shell.routeModels[routeProviders].(providerSetupRouteModel)
	if string(provider.secretRunes) != ":" {
		t.Fatalf("expected route-local secret entry to capture ':', got %q", string(provider.secretRunes))
	}
}

func TestShellTextEntryAllowsFocusTraversalByDefault(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	m.setFocus(focusContent)
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatal("did not expect additional command from focus traversal")
	}
	shell := updated.(shellModel)
	if shell.focus != focusInspector {
		t.Fatalf("expected tab focus traversal in text_entry to reach inspector, got %v", shell.focus)
	}
}

func TestShellProviderExclusiveCaptureBlocksFocusTraversal(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.setFocus(focusContent)
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatal("did not expect shell focus traversal command during provider secret entry")
	}
	shell := updated.(shellModel)
	if shell.focus != focusContent {
		t.Fatalf("expected focus unchanged during exclusive local capture, got %v", shell.focus)
	}

	updated, cmd = shell.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if cmd != nil {
		t.Fatal("did not expect shell focus traversal command during provider secret entry")
	}
	shell = updated.(shellModel)
	if shell.focus != focusContent {
		t.Fatalf("expected shift+tab focus unchanged during exclusive local capture, got %v", shell.focus)
	}
}

func TestShellNormalTabAndShiftTabTraverseFocus(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}
	m.setFocus(focusContent)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if cmd != nil {
		t.Fatal("did not expect async command from shift+tab traversal")
	}
	shell := updated.(shellModel)
	if shell.focus != focusNav {
		t.Fatalf("expected shift+tab to traverse to nav in normal interaction, got %v", shell.focus)
	}

	updated, cmd = shell.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatal("did not expect async command from tab traversal")
	}
	shell = updated.(shellModel)
	if shell.focus != focusContent {
		t.Fatalf("expected tab to traverse back to content in normal interaction, got %v", shell.focus)
	}
}

func TestShellLeaderManagedFocusActionsExecuteThroughUnifiedActionGraph(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}
	m.setFocus(focusContent)

	cmd := m.commands.commands["shell.focus_next"]
	cmd.LeaderPath = []string{"z", "n"}
	cmd.LeaderGroup = "Custom"
	m.commands.Register(cmd)
	m.actions = newShellActionGraph(m.routes, m.commands)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	shell = updated.(shellModel)
	updated, leaderCmd := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if leaderCmd != nil {
		t.Fatal("did not expect async command from leader focus action")
	}
	shell = updated.(shellModel)
	if shell.focus != focusInspector {
		t.Fatalf("expected custom leader-bound focus action to run via action graph and move focus to inspector, got %v", shell.focus)
	}

	updated, tabCmd := shell.Update(tea.KeyMsg{Type: tea.KeyTab})
	if tabCmd != nil {
		t.Fatal("did not expect async command from default tab traversal")
	}
	shell = updated.(shellModel)
	if shell.focus != focusNav {
		t.Fatalf("expected default tab traversal unchanged after leader action, got %v", shell.focus)
	}
}

func TestShellLeaderActionRestoresFocusAfterSuccessfulSequence(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}
	m.setFocus(focusContent)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)
	if shell.focus != focusPalette {
		t.Fatalf("expected leader overlay focus, got %v", shell.focus)
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	shell = updated.(shellModel)
	updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd != nil {
		t.Fatal("did not expect async command from successful leader sequence")
	}
	shell = updated.(shellModel)
	if shell.focus == focusPalette {
		t.Fatalf("expected focus restored after successful leader sequence, got %v", shell.focus)
	}
	if shell.overlayManager.Contains(overlayIDLeader) {
		t.Fatalf("expected leader overlay removed after successful sequence, got %v", shell.overlays)
	}
}

func TestShellDefaultLeaderKeyStartsLeaderModeWithImmediateOverlay(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.height = 28

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if cmd != nil {
		t.Fatal("did not expect asynchronous command when entering leader mode")
	}
	shell := updated.(shellModel)
	if !shell.leader.Active() {
		t.Fatal("expected leader mode active after default leader key")
	}
	if shell.focus != focusPalette {
		t.Fatalf("expected leader mode to take overlay focus, got %v", shell.focus)
	}
	if !shell.overlayManager.Contains(overlayIDLeader) {
		t.Fatalf("expected leader overlay in stack, got %v", shell.overlays)
	}
	view := shell.View()
	if !strings.Contains(view, "Leader Mode") || !strings.Contains(view, "Valid next keys") {
		t.Fatalf("expected immediate which-key overlay, got %q", view)
	}
}

func TestShellLeaderOverlayNarrowsChoicesAfterValidStep(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.height = 28
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)
	if got := len(shell.leader.Choices()); got < 2 {
		t.Fatalf("expected multiple root leader choices, got %d", got)
	}
	root := shell.leader.Choices()
	workbenchFound := false
	for _, choice := range root {
		if choice.Key == "w" && choice.Label == "Workbench" {
			workbenchFound = true
			break
		}
	}
	if !workbenchFound {
		t.Fatalf("expected workbench leader group from action definitions, got %+v", root)
	}

	updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if cmd != nil {
		t.Fatal("did not expect command while progressing non-terminal leader sequence")
	}
	shell = updated.(shellModel)
	if !shell.leader.Active() {
		t.Fatal("expected leader mode to remain active after partial sequence")
	}
	choices := shell.leader.Choices()
	if len(choices) < 3 {
		t.Fatalf("expected workbench child choices from action definitions, got %+v", choices)
	}
	if !strings.Contains(shell.View(), "Sequence: w") {
		t.Fatalf("expected leader overlay to show sequence progression, got %q", shell.View())
	}
}

func TestShellLeaderRootShowsInitialFamilies(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)

	choices := shell.leader.Choices()
	keys := map[string]bool{}
	for _, choice := range choices {
		keys[choice.Key] = true
	}
	for _, want := range []string{"s", "o", "w", "c", "a", "q"} {
		if !keys[want] {
			t.Fatalf("expected leader root to include family %q, got %+v", want, choices)
		}
	}
}

func TestShellLeaderOverlayChoicesComeFromActionDefinitions(t *testing.T) {
	m := newShellModel()
	cmd := m.commands.commands["shell.toggle_sidebar"]
	cmd.LeaderPath = []string{"z", "s"}
	cmd.LeaderGroup = "Custom"
	m.commands.Register(cmd)
	m.actions = newShellActionGraph(m.routes, m.commands)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)
	rootChoices := shell.leader.Choices()
	found := false
	for _, choice := range rootChoices {
		if choice.Key == "z" && choice.Label == "Custom" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected leader root choices from action graph definitions, got %+v", rootChoices)
	}
}

func TestShellLeaderInvalidKeyAbortsWithFeedbackWithoutAction(t *testing.T) {
	m := newShellModel()
	initialRoute := m.currentRouteID()
	initialSidebar := m.sidebarVisible

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)
	updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Fatal("did not expect command execution on invalid leader key")
	}
	shell = updated.(shellModel)
	if shell.leader.Active() {
		t.Fatal("expected leader mode aborted on invalid key")
	}
	if shell.currentRouteID() != initialRoute {
		t.Fatalf("expected route unchanged after invalid leader key, got %q", shell.currentRouteID())
	}
	if shell.sidebarVisible != initialSidebar {
		t.Fatal("expected unrelated sidebar action not to run")
	}
	if shell.overlayManager.Contains(overlayIDLeader) {
		t.Fatalf("expected leader overlay closed after invalid key, got %v", shell.overlays)
	}
	if got := shell.toasts.Latest(); !strings.Contains(got, "Leader aborted:") {
		t.Fatalf("expected leader abort feedback toast, got %q", got)
	}
}

func TestShellLeaderEscAbortsMode(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEsc})
	shell = updated.(shellModel)
	if shell.leader.Active() {
		t.Fatal("expected leader mode inactive after esc")
	}
	if shell.overlayManager.Contains(overlayIDLeader) {
		t.Fatalf("expected leader overlay removed after esc, got %v", shell.overlays)
	}
}

func TestShellKeyboardOwnershipBlocksLeaderStart(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if cmd != nil {
		t.Fatal("did not expect shell command while provider secret entry owns keyboard")
	}
	shell := updated.(shellModel)
	if shell.leader.Active() {
		t.Fatal("expected leader mode blocked by exclusive local capture")
	}
	provider := shell.routeModels[routeProviders].(providerSetupRouteModel)
	if string(provider.secretRunes) != " " {
		t.Fatalf("expected route-local secret entry to capture space, got %q", string(provider.secretRunes))
	}
}

func TestShellKeyboardOwnershipBlocksPaletteAndSessionEntryInExclusiveCapture(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if cmd != nil {
		t.Fatal("did not expect shell palette command while provider secret entry owns keyboard")
	}
	shell := updated.(shellModel)
	if shell.palette.IsOpen() {
		t.Fatal("expected palette entry blocked by exclusive local capture")
	}

	updated, cmd = shell.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if cmd != nil {
		t.Fatal("did not expect shell session-switch command while provider secret entry owns keyboard")
	}
	shell = updated.(shellModel)
	if shell.sessions.IsOpen() {
		t.Fatal("expected session quick-switch blocked by exclusive local capture")
	}
}

func TestLeaderStartKeyValidationRejectsUnsafeOptions(t *testing.T) {
	for _, key := range []string{"enter", "esc", "ctrl+c"} {
		if _, err := shellLeaderStartKeyBinding(key); err == nil {
			t.Fatalf("expected unsafe leader key %q to be rejected", key)
		}
	}
}
