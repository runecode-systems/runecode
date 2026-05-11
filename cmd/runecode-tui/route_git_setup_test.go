package main

import (
	"strings"
	"testing"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func TestGitSetupRouteViewUsesGuidedSummaries(t *testing.T) {
	model := newGitSetupRouteModel(routeDefinition{ID: routeGitSetup, Label: "Git Setup"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeGitSetup})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(120, 40, focusContent)
	for _, want := range []string{
		"Provider link:",
		"Commit identity:",
		"Review posture:",
		"Next safe action:",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q in %q", want, view)
		}
	}
	for _, unwanted := range []string{"bootstrap_mode=", "headless=", "token_fallback=", "Route keys:"} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("view unexpectedly contains %q in %q", unwanted, view)
		}
	}
}

func TestGitSetupRouteViewGuidesIdentitySetupWhenProviderLinked(t *testing.T) {
	model := newGitSetupRouteModel(routeDefinition{ID: routeGitSetup, Label: "Git Setup"}, &fakeBrokerClient{})
	loaded := gitSetupLoadedMsg{resp: brokerapi.GitSetupGetResponse{
		ProviderAccount: brokerapi.GitProviderAccountState{Provider: "github", AccountUsername: "octocat", Linked: true},
		AuthPosture:     brokerapi.GitAuthPostureState{Provider: "github", AuthStatus: "linked"},
		ControlPlaneState: brokerapi.GitControlPlaneState{
			DefaultIdentityProfileID: "",
		},
		PolicySurface: brokerapi.GitPolicySurfaceState{ArtifactManagedOnly: true, DirectMutationSupport: false},
	}, seq: 0}

	updated, _ := model.Update(loaded)
	view := updated.View(120, 40, focusContent)
	for _, want := range []string{
		"The provider account is linked, but commit identity still needs setup.",
		"Next safe action: Upsert the default broker-managed commit identity.",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q in %q", want, view)
		}
	}
}
