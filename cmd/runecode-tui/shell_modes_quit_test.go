package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestShellCommandModeOpenChatResolvesThroughActionGraph(t *testing.T) {
	testShellCommandModeCommand(t, ":open chat", func(m *shellModel) {
		m.location.Primary = shellObjectLocation{RouteID: routeRuns, Object: workbenchObjectRef{Kind: "route", ID: string(routeRuns)}}
	}, "open chat", func(t *testing.T, shell shellModel, cmd tea.Cmd) {
		if cmd == nil {
			t.Fatal("expected :open chat to return route activation command")
		}
		updated, followCmd := shell.Update(cmd())
		shell = updated.(shellModel)
		if followCmd != nil {
			updated, _ = shell.Update(followCmd())
			shell = updated.(shellModel)
		}
		if shell.currentRouteID() != routeChat {
			t.Fatalf("expected :open chat to jump to %q, got %q", routeChat, shell.currentRouteID())
		}
	})
}

func TestShellCommandModeSidebarToggleResolvesThroughActionGraph(t *testing.T) {
	testShellCommandModeCommand(t, ":sidebar toggle", func(m *shellModel) {
		m.width = 150
		if !m.sidebarVisible {
			t.Fatal("expected sidebar visible initially")
		}
	}, "sidebar toggle", func(t *testing.T, shell shellModel, cmd tea.Cmd) {
		if cmd != nil {
			t.Fatal("expected :sidebar toggle to execute synchronously")
		}
		if shell.sidebarVisible {
			t.Fatal("expected :sidebar toggle to execute")
		}
	})
}

func TestShellCommandModeThemeCycleResolvesThroughActionGraph(t *testing.T) {
	testShellCommandModeCommand(t, ":theme cycle", nil, "theme cycle", func(t *testing.T, shell shellModel, cmd tea.Cmd) {
		if cmd != nil {
			t.Fatal("expected :theme cycle to execute synchronously")
		}
		if shell.themePreset == themePresetDark {
			t.Fatalf("expected :theme cycle to change theme preset from %q", themePresetDark)
		}
	})
}

func TestShellCommandModeSetLeaderVariantsResolveThroughActionGraph(t *testing.T) {
	testShellCommandModeCommand(t, ":set leader comma", nil, "set leader comma", func(t *testing.T, shell shellModel, cmd tea.Cmd) {
		if cmd != nil {
			t.Fatal("expected :set leader comma to execute synchronously")
		}
		if shell.keys.LeaderStart.label() != "," {
			t.Fatalf("expected leader key to be comma, got %q", shell.keys.LeaderStart.label())
		}
	})
	testShellCommandModeCommand(t, ":set leader backslash", nil, "set leader backslash", func(t *testing.T, shell shellModel, cmd tea.Cmd) {
		if cmd != nil {
			t.Fatal("expected :set leader backslash to execute synchronously")
		}
		if shell.keys.LeaderStart.label() != "\\" {
			t.Fatalf("expected leader key to be backslash, got %q", shell.keys.LeaderStart.label())
		}
	})
	testShellCommandModeCommand(t, ":set leader default", func(m *shellModel) {
		if err := m.configureLeaderKey("comma"); err != nil {
			t.Fatalf("configureLeaderKey(comma) error = %v", err)
		}
	}, "set leader default", func(t *testing.T, shell shellModel, cmd tea.Cmd) {
		if cmd != nil {
			t.Fatal("expected :set leader default to execute synchronously")
		}
		if shell.keys.LeaderStart.label() != "space" {
			t.Fatalf("expected leader key reset to space, got %q", shell.keys.LeaderStart.label())
		}
	})
}

func TestShellCommandModeQuitAliasesResolveThroughActionGraph(t *testing.T) {
	for _, command := range []string{"q", "quit"} {
		testShellCommandModeCommand(t, ":"+command, nil, command, func(t *testing.T, shell shellModel, cmd tea.Cmd) {
			if cmd != nil {
				t.Fatalf("expected %q to request quit confirmation first", command)
			}
			if shell.quitting {
				t.Fatalf("expected %q not to quit immediately while command entry is active", command)
			}
			if !shell.quitConfirm.active {
				t.Fatalf("expected %q to activate quit confirmation", command)
			}
		})
	}
}

func testShellCommandModeCommand(t *testing.T, name string, setup func(*shellModel), draft string, assert func(*testing.T, shellModel, tea.Cmd)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		m := newShellModel()
		if setup != nil {
			setup(&m)
		}
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		shell := updated.(shellModel)
		for _, r := range draft {
			updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			shell = updated.(shellModel)
		}
		updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
		shell = updated.(shellModel)
		assert(t, shell, cmd)
	})
}

func TestShellVisibleQuitPromptsWhenComposeEntryWouldBeDiscarded(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, cmd := m.activateSidebarAction("shell.quit")
	shell := updated.(shellModel)
	if cmd != nil {
		t.Fatal("expected visible quit to prompt before quitting during compose entry")
	}
	if shell.quitting {
		t.Fatal("expected visible quit not to quit immediately during compose entry")
	}
	if !shell.quitConfirm.active {
		t.Fatal("expected visible quit to open quit confirmation during compose entry")
	}
	if !strings.Contains(shell.renderQuitConfirmDialog(), "chat compose") {
		t.Fatalf("expected quit confirmation to explain compose discard risk, got %q", shell.renderQuitConfirmDialog())
	}
}

func TestShellVisibleQuitPromptsWhenProviderSecretEntryWouldBeDiscarded(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}

	updated, cmd := m.activateSidebarAction("shell.quit")
	shell := updated.(shellModel)
	if cmd != nil {
		t.Fatal("expected visible quit to prompt before quitting during provider secret entry")
	}
	if shell.quitting {
		t.Fatal("expected visible quit not to quit immediately during provider secret entry")
	}
	if !shell.quitConfirm.active {
		t.Fatal("expected visible quit to open quit confirmation during provider secret entry")
	}
	if !strings.Contains(shell.renderQuitConfirmDialog(), "provider secret entry") {
		t.Fatalf("expected quit confirmation to explain provider secret discard risk, got %q", shell.renderQuitConfirmDialog())
	}
}

func TestShellLeaderQuitPromptsWhenDispatchedDuringCommandEntry(t *testing.T) {
	m := newShellModel()
	m.commandMode = m.commandMode.Open().Append("q")
	m.leader.Rebind(m.actions.leaderBindings(m))
	m.leader.Start()
	_, _ = m.leader.Step("q")
	action, complete := m.leader.Step("q")
	if !complete {
		t.Fatal("expected leader q q to resolve quit action")
	}

	updated, cmd := m.applyPaletteAction(action)
	shell := updated.(shellModel)
	if cmd != nil {
		t.Fatal("expected leader-dispatched quit to prompt while command entry active")
	}
	if shell.quitting {
		t.Fatal("expected leader-dispatched quit not to quit immediately during command entry")
	}
	if !shell.quitConfirm.active {
		t.Fatal("expected leader-dispatched quit to activate quit confirmation")
	}
	if !strings.Contains(shell.renderQuitConfirmDialog(), "command entry") {
		t.Fatalf("expected quit confirmation reason to mention command entry, got %q", shell.renderQuitConfirmDialog())
	}
}

func TestShellNormalQuitStillImmediateWhenNoLocalEntryState(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.setFocus(focusNav)

	entries := m.sidebarEntries()
	quitIndex := -1
	for i, entry := range entries {
		if entry.Kind == sidebarEntryAction && entry.ActionID == "shell.quit" {
			quitIndex = i
			break
		}
	}
	if quitIndex < 0 {
		t.Fatal("expected sidebar quit action entry")
	}
	m.sidebarCursor = quitIndex

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell := updated.(shellModel)
	if cmd == nil {
		t.Fatal("expected normal visible quit to return quit command")
	}
	if !shell.quitting {
		t.Fatal("expected normal visible quit to set quitting state")
	}
	if shell.quitConfirm.active {
		t.Fatal("did not expect quit confirmation when no local entry state exists")
	}
}

func TestShellQuitConfirmationNotShownForNonEntryState(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.selectionMode = true
	m.setFocus(focusNav)

	entries := m.sidebarEntries()
	quitIndex := -1
	for i, entry := range entries {
		if entry.Kind == sidebarEntryAction && entry.ActionID == "shell.quit" {
			quitIndex = i
			break
		}
	}
	if quitIndex < 0 {
		t.Fatal("expected sidebar quit action entry")
	}
	m.sidebarCursor = quitIndex

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell := updated.(shellModel)
	if cmd == nil {
		t.Fatal("expected quit command without confirmation for non-entry state")
	}
	if !shell.quitting {
		t.Fatal("expected shell to quit for non-entry state")
	}
	if shell.quitConfirm.active {
		t.Fatal("did not expect quit confirmation for non-entry state")
	}
}

func TestShellEmergencyCtrlCStillExitsWhenQuitConfirmationShown(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, _ := m.activateSidebarAction("shell.quit")
	shell := updated.(shellModel)
	if !shell.quitConfirm.active {
		t.Fatal("expected quit confirmation active before emergency ctrl+c test")
	}

	updated, firstCmd := shell.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if firstCmd == nil {
		t.Fatal("expected first ctrl+c to arm emergency path while confirmation is shown")
	}
	shell = updated.(shellModel)
	if shell.quitting {
		t.Fatal("expected first ctrl+c not to quit")
	}
	if !shell.emergencyQuit.pending {
		t.Fatal("expected first ctrl+c to arm emergency pending state")
	}
	if !shell.quitConfirm.active {
		t.Fatal("expected first ctrl+c to keep normal quit confirmation visible")
	}

	updated, secondCmd := shell.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	shell = updated.(shellModel)
	if secondCmd == nil {
		t.Fatal("expected second ctrl+c to quit while emergency pending")
	}
	if !shell.quitting {
		t.Fatal("expected second ctrl+c to set quitting state")
	}
}

func TestShellCtrlPDiscoveryUsesUnifiedActionGraphEntries(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	shell := updated.(shellModel)
	entries := shell.palette.entries

	have := map[string]bool{}
	for _, entry := range entries {
		have[strings.ToLower(strings.TrimSpace(entry.Label))] = true
	}
	for _, want := range []string{"toggle sidebar", "cycle theme preset", "copy current identity", "open approvals", "open action center", "quit runecode"} {
		if !have[want] {
			t.Fatalf("expected ctrl+p discovery to include action-graph entry %q", want)
		}
	}
}

func TestShellSidebarShowsVisibleQuitActionOutsideLeaderAndCommandMode(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40

	entries := m.sidebarEntries()
	foundQuit := false
	for _, entry := range entries {
		if entry.Kind == sidebarEntryAction && entry.ActionID == "shell.quit" {
			foundQuit = true
			break
		}
	}
	if !foundQuit {
		t.Fatal("expected sidebar entries to include visible quit shell action")
	}

	sidebar := m.renderSidebar()
	if !strings.Contains(sidebar, "Actions") || !strings.Contains(sidebar, "Quit RuneCode") {
		t.Fatalf("expected rendered sidebar to expose quit action, got %q", sidebar)
	}
}

func TestShellBottomStripShowsQuitDiscoverabilityWhenSidebarHiddenNarrow(t *testing.T) {
	m := newShellModel()
	m.width = shellMediumMinWidth - 1
	m.height = 40
	if m.effectiveSidebarVisible() {
		t.Fatal("expected narrow layout to hide sidebar surface")
	}

	bottom := m.renderBottomStrip(m.activeShellSurface())
	if !strings.Contains(bottom, "Quick action: Quit RuneCode") || !strings.Contains(bottom, "(:quit)") {
		t.Fatalf("expected bottom strip to expose beginner quit affordance when sidebar hidden, got %q", bottom)
	}
}

func TestShellBottomStripShowsQuitDiscoverabilityWhenSidebarToggledOff(t *testing.T) {
	m := newShellModel()
	m.width = shellMediumMinWidth
	m.height = 40
	m.sidebarVisible = false
	if m.effectiveSidebarVisible() {
		t.Fatal("expected toggled-off sidebar to be hidden")
	}

	bottom := m.renderBottomStrip(m.activeShellSurface())
	if !strings.Contains(bottom, "Quick action: Quit RuneCode") || !strings.Contains(bottom, "(:quit)") {
		t.Fatalf("expected bottom strip to expose beginner quit affordance when sidebar is off, got %q", bottom)
	}
}

func TestShellBottomStripHidesQuitDiscoverabilityWhenQuitActionMissing(t *testing.T) {
	m := newShellModel()
	delete(m.actions.byID, "shell.quit")

	bottom := m.renderBottomStrip(m.activeShellSurface())
	if strings.Contains(bottom, "Quick action:") || strings.Contains(bottom, "(:quit)") {
		t.Fatalf("expected bottom strip not to advertise quit when quit action is missing, got %q", bottom)
	}
}

func TestShellSidebarQuitActionExecutesViaUnifiedActionGraph(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.setFocus(focusNav)

	entries := m.sidebarEntries()
	quitIndex := -1
	for i, entry := range entries {
		if entry.Kind == sidebarEntryAction && entry.ActionID == "shell.quit" {
			quitIndex = i
			break
		}
	}
	if quitIndex < 0 {
		t.Fatal("expected sidebar quit action entry")
	}
	m.sidebarCursor = quitIndex

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell := updated.(shellModel)
	if cmd == nil {
		t.Fatal("expected sidebar quit action to return quit command")
	}
	if !shell.quitting {
		t.Fatal("expected sidebar quit action to set shell quitting state")
	}
}

func TestShellLeaderQuitExecutesThroughUnifiedActionGraph(t *testing.T) {
	m := newShellModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	shell = updated.(shellModel)
	updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	shell = updated.(shellModel)

	if cmd == nil {
		t.Fatal("expected leader quit action to return quit command")
	}
	if !shell.quitting {
		t.Fatal("expected leader quit sequence to set shell quitting state")
	}
}

func TestShellCtrlCFirstPressArmsEmergencyWhenNoEntryState(t *testing.T) {
	m := newShellModel()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected first ctrl+c to return emergency arm timeout command")
	}
	shell := updated.(shellModel)
	if shell.quitting {
		t.Fatal("expected first ctrl+c not to quit immediately")
	}
	if !shell.emergencyQuit.pending {
		t.Fatal("expected emergency quit pending after first ctrl+c")
	}
	if shell.quitConfirm.active {
		t.Fatal("did not expect quit confirmation after first ctrl+c without explicit quit action")
	}
}

func TestShellCtrlCFirstPressArmsEmergencyWhenEntryStateActive(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected first ctrl+c to return emergency arm timeout command while entry state active")
	}
	shell := updated.(shellModel)
	if shell.quitting {
		t.Fatal("expected first ctrl+c not to quit while entry state is active")
	}
	if shell.quitConfirm.active {
		t.Fatal("did not expect first ctrl+c to open quit confirmation while entry state is active")
	}
	if !shell.emergencyQuit.pending {
		t.Fatal("expected first ctrl+c to arm pending emergency quit state")
	}
	bottom := shell.renderBottomStrip(shell.activeShellSurface())
	if !strings.Contains(bottom, "press ctrl+c once more to quit") {
		t.Fatalf("expected emergency follow-up warning in command-entry area, got %q", bottom)
	}
}

func TestShellEmergencyCtrlCSecondPressQuitsWhilePending(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	shell := updated.(shellModel)
	updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	shell = updated.(shellModel)
	if cmd == nil {
		t.Fatal("expected second ctrl+c while pending to return quit command")
	}
	if !shell.quitting {
		t.Fatal("expected second ctrl+c while pending to set quitting state")
	}
}

func TestShellEmergencyCtrlCPendingClearsOnNormalInteraction(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	shell := updated.(shellModel)
	if !shell.emergencyQuit.pending {
		t.Fatal("expected emergency quit pending after first ctrl+c")
	}

	updated, cmd := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Fatal("did not expect quit command on normal follow-up interaction")
	}
	shell = updated.(shellModel)
	if shell.quitConfirm.active {
		t.Fatal("did not expect normal interaction to open quit confirmation")
	}
	if shell.emergencyQuit.pending {
		t.Fatal("expected normal interaction to clear emergency quit pending state")
	}
	if shell.quitting {
		t.Fatal("did not expect normal interaction to quit")
	}
}

func TestShellEmergencyCtrlCPendingClearsOnTimeout(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	shell := updated.(shellModel)
	if !shell.emergencyQuit.pending {
		t.Fatal("expected emergency quit pending after first ctrl+c")
	}
	token := shell.emergencyQuit.token

	updated, cmd := shell.Update(shellEmergencyQuitTimeoutMsg{token: token})
	if cmd != nil {
		t.Fatal("did not expect follow-up command when timeout clears emergency state")
	}
	shell = updated.(shellModel)
	if shell.emergencyQuit.pending {
		t.Fatal("expected timeout to clear emergency quit pending state")
	}
	if shell.quitConfirm.active {
		t.Fatal("did not expect emergency timeout to leave quit confirmation active")
	}
	if shell.quitting {
		t.Fatal("did not expect timeout to quit")
	}
}

func TestShellEmergencyCtrlCBypassesExclusiveLocalCapture(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeProviders, Object: workbenchObjectRef{Kind: "route", ID: string(routeProviders)}}
	m.routeModels[routeProviders] = providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, entryActive: true}

	updated, firstCmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if firstCmd == nil {
		t.Fatal("expected first ctrl+c to arm emergency quit even during exclusive local capture")
	}
	shell := updated.(shellModel)
	if shell.quitting {
		t.Fatal("expected first ctrl+c not to quit")
	}
	if shell.quitConfirm.active {
		t.Fatal("did not expect first ctrl+c to open quit confirmation during exclusive local capture")
	}
	if !shell.emergencyQuit.pending {
		t.Fatal("expected first ctrl+c to arm emergency pending state")
	}
	provider := shell.routeModels[routeProviders].(providerSetupRouteModel)
	if got := string(provider.secretRunes); got != "" {
		t.Fatalf("expected route-local secret entry not to capture emergency ctrl+c, got %q", got)
	}

	updated, secondCmd := shell.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	shell = updated.(shellModel)
	if secondCmd == nil {
		t.Fatal("expected second ctrl+c to quit during exclusive local capture")
	}
	if !shell.quitting {
		t.Fatal("expected emergency second ctrl+c to set quitting state")
	}
}

func TestQuitModeledAsActionNotRoute(t *testing.T) {
	m := newShellModel()
	for _, route := range m.routes {
		if string(route.ID) == "shell.quit" || strings.EqualFold(strings.TrimSpace(route.Label), "Quit RuneCode") {
			t.Fatalf("expected quit to be modeled as action, found route id=%q label=%q", route.ID, route.Label)
		}
	}
	if _, ok := m.actions.definitionByID("shell.quit"); !ok {
		t.Fatal("expected quit action definition in unified action graph")
	}
}

func TestShellCommandCoverageDoesNotPreemptRouteTextEntry(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}
	chat := m.routeModels[routeChat].(chatRouteModel)
	chat.composeOn = true
	chat.composer.Focus()
	m.routeModels[routeChat] = chat

	for _, r := range "open chat" {
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		if cmd != nil {
			t.Fatalf("did not expect shell command while composing for rune %q", string(r))
		}
		m = updated.(shellModel)
	}

	if m.currentRouteID() != routeChat {
		t.Fatalf("expected route unchanged during compose typing, got %q", m.currentRouteID())
	}
	chat = m.routeModels[routeChat].(chatRouteModel)
	if !strings.Contains(chat.composer.Value(), "open chat") {
		t.Fatalf("expected compose buffer to retain typed text, got %q", chat.composer.Value())
	}
}
