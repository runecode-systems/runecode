package main

import (
	"strings"
	"testing"
)

func TestNewRouteModelsHybridMVPRoutesLoad(t *testing.T) {
	originalFactory := localBrokerClientFactory
	localBrokerClientFactory = func() localBrokerClient { return &fakeBrokerClient{} }
	t.Cleanup(func() {
		localBrokerClientFactory = originalFactory
	})

	models := newRouteModels(shellRoutes())
	ids := []routeID{routeDashboard, routeChat, routeRuns, routeApprovals, routeArtifacts, routeAudit, routeStatus}
	for _, id := range ids {
		model, ok := models[id]
		if !ok {
			t.Fatalf("missing model for route %q", id)
		}
		updated, cmd := model.Update(routeActivatedMsg{RouteID: id})
		if cmd == nil {
			t.Fatalf("expected load command for route %q", id)
		}
		loaded := cmd()
		updated, _ = updated.Update(loaded)
		view := updated.View(120, 40, focusContent)
		if strings.Contains(view, "Route initialization failed") {
			t.Fatalf("expected functional route view for %q, got %q", id, view)
		}
		if strings.Contains(strings.ToLower(view), "load failed") {
			t.Fatalf("expected successful route load for %q, got %q", id, view)
		}
	}
}

func TestSafeUIErrorTextAddsRemediationForLocalIPCDialFallbackCode(t *testing.T) {
	got := safeUIErrorText(assertError("local_ipc_dial_error"))
	if !strings.Contains(got, "runecode-broker serve-local") {
		t.Fatalf("expected broker remediation in %q", got)
	}
	if !strings.Contains(got, "press r to retry") {
		t.Fatalf("expected retry hint in %q", got)
	}
}

func TestSafeUIErrorTextAddsRemediationForLocalIPCConfigFallbackCode(t *testing.T) {
	got := safeUIErrorText(assertError("local_ipc_config_error"))
	if !strings.Contains(got, "Linux") {
		t.Fatalf("expected platform hint in %q", got)
	}
	if !strings.Contains(got, "--runtime-dir/--socket-name") {
		t.Fatalf("expected ipc override hint in %q", got)
	}
	if strings.Contains(got, "local_ipc_config_error") {
		t.Fatalf("expected user-facing remediation text, got %q", got)
	}
}

type fixedError string

func (e fixedError) Error() string { return string(e) }

func assertError(text string) error { return fixedError(text) }
