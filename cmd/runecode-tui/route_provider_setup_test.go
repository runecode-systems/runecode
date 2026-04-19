package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestProviderSetupRouteActivationLoadsBrokerProjectedProfilePosture(t *testing.T) {
	recording := newRecordingBrokerClient(&fakeBrokerClient{})
	model := newProviderSetupRouteModel(routeDefinition{ID: routeProviders, Label: "Model Providers"}, recording)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeProviders})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Auth modes: supported=[direct_credential] current=direct_credential") {
		t.Fatalf("expected auth-mode posture in provider view, got %q", view)
	}
	if !strings.Contains(view, "Compatibility posture: unverified") {
		t.Fatalf("expected compatibility posture in provider view, got %q", view)
	}
	assertStringSliceEqual(t, recording.Calls(), []string{"ProviderProfileList"})
}

func TestProviderSetupRouteSetupFlowUsesBrokerOwnedContracts(t *testing.T) {
	recording := newRecordingBrokerClient(&fakeBrokerClient{})
	model := newProviderSetupRouteModel(routeDefinition{ID: routeProviders, Label: "Model Providers"}, recording)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeProviders})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("expected setup start command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s', 'k', '-', '1'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected setup submit command")
	}
	updated, cmd = updated.Update(cmd())
	if cmd == nil {
		t.Fatal("expected profile refresh command after submit")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Direct credential stored in secretsd") {
		t.Fatalf("expected success status in provider view, got %q", view)
	}
	assertStringSliceEqual(t, recording.Calls(), []string{"ProviderProfileList", "ProviderSetupSessionBegin", "ProviderSetupSecretIngressPrepare", "ProviderSetupSecretIngressSubmit", "ProviderProfileList"})
}
