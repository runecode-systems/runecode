package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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
