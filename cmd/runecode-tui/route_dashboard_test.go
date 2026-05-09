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
		"Degraded",
		"Current work",
		"run-1 is active",
		"Approvals: 1 approval is waiting",
		"At a glance",
		"Work 1",
		"Approvals 1",
		"Review 1",
		"Next action",
		"Open Action Center",
		"Evidence: Runs, Audit, and Status keep proof details.",
	)
	for _, retired := range []string{"Runtime and evidence", "Supporting detail", "ALERT_AUDIT_UNANCHORED", "AUDIT_UNANCHORED_OR_DEGRADED", "Live activity"} {
		if strings.Contains(view, retired) {
			t.Fatalf("did not expect retired dashboard primary detail %q in view, got %q", retired, view)
		}
	}
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
		"Degraded",
		"Current work",
		"At a glance",
		"Review 1",
		"Next action",
		"Open Action Center",
	)
	if strings.Contains(view, "gateway_failure") || strings.Contains(view, "Evidence verification unavailable") {
		t.Fatalf("expected audit fallback detail to stay out of primary dashboard, got %q", view)
	}
}

func TestDashboardExecutiveHierarchyAndCalmPrimaryWording(t *testing.T) {
	model := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeDashboard})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)

	cardIndex := strings.Index(view, "Evidence or runtime posture needs review.")
	currentIndex := strings.Index(view, "Current work")
	countsIndex := strings.Index(view, "At a glance")
	nextIndex := strings.Index(view, "Next action")
	if cardIndex < 0 || currentIndex < 0 || countsIndex < 0 || nextIndex < 0 {
		t.Fatalf("expected executive hierarchy sections in view, got %q", view)
	}
	if !(cardIndex < currentIndex && currentIndex < countsIndex && countsIndex < nextIndex) {
		t.Fatalf("expected state card before current work before counts before next action, got %q", view)
	}
	if strings.Contains(view, "Protocol bundle") || strings.Contains(view, "watch_family") || strings.Contains(view, "Supporting detail") {
		t.Fatalf("expected no debug-heavy primary wording, got %q", view)
	}
	if !strings.Contains(view, "Route: Action Center") {
		t.Fatalf("expected dashboard to point operators to Action Center, got %q", view)
	}
	for _, unwanted := range []string{"DEGRADED Degraded", "! DEGRADED", "APPROVAL REQUIRED Needs attention", "BLOCKED Blocked", "EMPTY No work yet"} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("expected dashboard hero without duplicated state/title wording %q, got %q", unwanted, view)
		}
	}
}

func TestDashboardViewPreservesSectionGaps(t *testing.T) {
	model := newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeDashboard})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	for _, want := range []string{"Current work", "\n\n|  At a glance", "Review 1\n\n|  Next action"} {
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
	if strings.Contains(view, "\n\n\n") {
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
	if !strings.Contains(view, "Runs, Audit, and Status keep") {
		t.Fatalf("expected wrapped dashboard detail cue retained, got %q", view)
	}
	if strings.Contains(view, "AUDIT_UNANCHORED_OR_DEGRADED") {
		t.Fatalf("expected raw audit badge removed from primary dashboard, got %q", view)
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
	if !strings.Contains(view, "\n\n|  At a glance") {
		t.Fatalf("expected preserved single blank section gap before At a glance, got %q", view)
	}
	mustContainAll(t, view,
		"Dashboard",
		"Current work",
		"At a glance",
		"Next action",
	)
}
