package main

import "testing"

func TestStatusRouteRendersProjectSubstratePostureAndGuidance(t *testing.T) {
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"Project substrate posture:",
		"compatibility=supported_with_upgrade_available",
		"Project substrate remediation:",
		"Project substrate upgrade:",
	)
}
