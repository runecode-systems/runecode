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
	updated, _ = updated.Update(shellLiveActivityUpdatedMsg{
		Live: dashboardLiveActivity{
			runWatch:      summarizeRunWatchEvents([]brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &brokerapi.RunSummary{RunID: "run-1"}}, {EventType: "run_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"}}),
			approvalWatch: summarizeApprovalWatchEvents([]brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &brokerapi.ApprovalSummary{ApprovalID: "ap-1"}}, {EventType: "approval_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"}}),
			sessionWatch:  summarizeSessionWatchEvents([]brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1"}}}, {EventType: "session_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"}}),
		},
		Feed: []shellLiveActivityEntry{{Family: "session_watch", EventType: "session_watch_terminal", Subject: "session-1", Status: "completed"}},
	})
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
		"feed:",
		"event=session_watch_terminal subject=session-1 status=completed",
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

func (c *dashboardAuditUnavailableClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error) {
	return c.fakeBrokerClient.AuditAnchorSegment(ctx, req)
}

func (c *dashboardAuditUnavailableClient) AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, error) {
	return c.fakeBrokerClient.AuditAnchorPresenceGet(ctx, req)
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
