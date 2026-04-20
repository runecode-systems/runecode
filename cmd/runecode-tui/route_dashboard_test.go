package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
		"Project substrate:",
		"compatibility=supported_with_upgrade_available",
		"Project substrate remediation:",
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
		"Project substrate:",
		"FAILED",
		"degraded=true",
		"AUDIT_VERIFICATION_UNAVAILABLE",
		"showing degraded fallback posture (gateway_failure)",
		"Control Plane",
		"Live Activity",
		"Live activity (typed watch families; logs are supplemental inspection only):",
	)
}

func TestDashboardViewPreservesSectionGaps(t *testing.T) {
	model := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeDashboard})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	for _, want := range []string{"PENDING_APPROVALS=1\n\nSafety Summary", "ALERT_AUDIT_UNANCHORED  audit posture unanchored/degraded\n\nControl Plane", "protocol bundle=0.9.0\n\nLive Activity"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected preserved blank section gap %q in view, got %q", want, view)
		}
	}
}

func TestDashboardAuditFallbackWithoutErrorDoesNotAddExtraBlankLine(t *testing.T) {
	model := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeDashboard})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	if strings.Contains(view, "Safety posture\n\nWorkflow posture") {
		t.Fatalf("did not expect extra blank line when audit fallback notice absent, got %q", view)
	}
}

func TestDashboardViewWrapsLongRowsToWidth(t *testing.T) {
	model := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeDashboard})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(64, 30, focusContent)
	for _, line := range strings.Split(view, "\n") {
		if lipgloss.Width(line) > 60 {
			t.Fatalf("expected wrapped dashboard line within content width, got width=%d line=%q", lipgloss.Width(line), line)
		}
	}
	if !strings.Contains(view, "runtime_posture_degraded=false") {
		t.Fatalf("expected wrapped safety strip content retained, got %q", view)
	}
	if !strings.Contains(view, "AUDIT_UNANCHORED_OR_DEGRADED") {
		t.Fatalf("expected wrapped long audit cue retained, got %q", view)
	}
}

func TestDashboardViewNarrowWidthKeepsBoundedLinesAndSectionSpacing(t *testing.T) {
	model := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeDashboard})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(44, 24, focusContent)
	if strings.Contains(view, "\n\n\n") {
		t.Fatalf("expected no triple blank section gaps in narrow view, got %q", view)
	}
	if !strings.Contains(view, "\n\nSafety Summary") {
		t.Fatalf("expected preserved single blank section gap before Safety Summary, got %q", view)
	}
	mustContainAll(t, view,
		"Dashboard",
		"Safety Summary",
		"Control Plane",
		"Live Activity",
	)
}
