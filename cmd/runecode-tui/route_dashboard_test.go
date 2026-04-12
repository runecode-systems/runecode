package main

import (
	"context"
	"errors"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestDashboardRouteShowsTypedLiveWatchFamilies(t *testing.T) {
	model := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeDashboard})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)

	mustContainAll(t, view,
		"Now",
		"CONTENT_READY",
		"Safety Summary",
		"Safety strip",
		"backend_kind=workspace",
		"runtime isolation=sandboxed",
		"audit posture=ok/degraded (unanchored/degraded)",
		"approval_profile=n/a",
		"Safety alerts:",
		"ALERT_AUDIT_UNANCHORED",
		"Control Plane",
		"Live Activity",
		"Live activity (typed watch families; logs are supplemental inspection only):",
		"totals events=2 snapshot=1 upsert=0 terminal=1 errors=0",
		"last_event=run_watch_terminal subject=run-1 status=completed",
		"last_event=approval_watch_terminal subject=ap-1 status=completed",
		"last_event=session_watch_terminal subject=session-1 status=completed",
		"Actions",
		"tab moves focus",
	)
}

type dashboardAuditUnavailableClient struct{ fakeBrokerClient }

func (c *dashboardAuditUnavailableClient) AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error) {
	_ = ctx
	_ = viewLimit
	return brokerapi.AuditVerificationGetResponse{}, errors.New("gateway_failure")
}

func TestDashboardRouteFallsBackWhenAuditVerificationUnavailable(t *testing.T) {
	model := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, &dashboardAuditUnavailableClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeDashboard})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)

	mustContainAll(t, view,
		"Dashboard",
		"Now",
		"Safety posture",
		"FAILED",
		"degraded=true",
		"AUDIT_VERIFICATION_UNAVAILABLE",
		"showing degraded fallback posture (gateway_failure)",
		"Control Plane",
		"Live Activity",
		"Live activity (typed watch families; logs are supplemental inspection only):",
	)
}
