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
