package main

import "testing"

func TestDashboardRouteShowsTypedLiveWatchFamilies(t *testing.T) {
	model := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeDashboard})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)

	mustContainAll(t, view,
		"Safety strip",
		"backend_kind=workspace",
		"runtime isolation=sandboxed",
		"audit posture=ok/degraded (unanchored/degraded)",
		"approval_profile=n/a",
		"Safety alerts:",
		"ALERT_AUDIT_UNANCHORED",
		"Live activity (typed watch families; logs are supplemental inspection only):",
		"run_watch events=2 snapshot=1 upsert=0 terminal=1 errors=0",
		"approval_watch events=2 snapshot=1 upsert=0 terminal=1 errors=0",
		"session_watch events=2 snapshot=1 upsert=0 terminal=1 errors=0",
	)
}
