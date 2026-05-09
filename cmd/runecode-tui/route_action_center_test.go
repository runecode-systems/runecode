package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestActionCenterViewKeepsFamiliesDistinctAndReservedQANotice(t *testing.T) {
	model := newActionCenterRouteModel(routeDefinition{ID: routeAction, Label: "Action Center"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeAction})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	updated, _ = updated.Update(shellLiveActivityUpdatedMsg{
		Live: dashboardLiveActivity{
			runWatch:      watchFamilySummary{family: "run_watch", errorCount: 1, lastStatus: "watch_error", lastSubject: "run-1"},
			approvalWatch: watchFamilySummary{family: "approval_watch", lastStatus: "ok", lastSubject: "ap-1"},
			sessionWatch:  watchFamilySummary{family: "session_watch", lastStatus: "ok", lastSubject: "session-1"},
		},
		Health: shellSyncHealth{State: shellSyncStateDegraded, ErrorText: "local_ipc_dial_error"},
	})

	view := updated.View(140, 40, focusContent)
	mustContainAll(t, view,
		"Action Center",
		"Blocked",
		"Queue families:",
		"approvals",
		"operational_attention",
		"blocked_work_impact",
		"Approvals queue",
		"Operational attention",
		"Blocked-work impact",
		"Action Center is the operator home for broker-known follow-up.",
		"owner/action:",
		"target:",
		"evidence:",
	)
	surface := updated.ShellSurface(routeShellContext{Width: 140, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	inspector := surface.Regions.Inspector.Body
	mustContainAll(t, inspector,
		"family=",
		"state=",
		"urgency=",
		"reason=",
		"impact=",
		"owner=",
		"required_action=",
		"evidence=",
	)
}

func TestActionCenterKeyboardTriageAndDrillDown(t *testing.T) {
	model := newActionCenterRouteModel(routeDefinition{ID: routeAction, Label: "Action Center"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeAction})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected drill-down command on enter")
	}
	msg := cmd()
	action, ok := msg.(paletteActionMsg)
	if !ok {
		t.Fatalf("expected paletteActionMsg drill-down, got %T", msg)
	}
	if action.Verb != verbJump {
		t.Fatalf("expected verbJump, got %q", action.Verb)
	}
	if strings.TrimSpace(action.Target.Kind) == "" {
		t.Fatal("expected non-empty drill-down target kind")
	}

	view := updated.View(140, 40, focusContent)
	if !strings.Contains(view, "Active triage family") {
		t.Fatalf("expected family indicator in view, got %q", view)
	}
	surface := updated.ShellSurface(routeShellContext{Width: 140, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	if !strings.Contains(surface.Regions.Inspector.Body, "drill_down_target=") {
		t.Fatalf("expected action center drill-down details in inspector, got %q", surface.Regions.Inspector.Body)
	}
}

func TestBuildApprovalActionItemsIncludesExpiryAndSupersededCues(t *testing.T) {
	now := timeNowUTCForTest()
	items := buildApprovalActionItems([]brokerapi.ApprovalSummary{
		{ApprovalID: "ap-expired", Status: "pending", ExpiresAt: now.Add(-1 * time.Minute).Format(time.RFC3339), BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-1", ActionKind: "promotion"}},
		{ApprovalID: "ap-soon", Status: "pending", ExpiresAt: now.Add(30 * time.Minute).Format(time.RFC3339), BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-2", ActionKind: "promotion"}},
		{ApprovalID: "ap-super", Status: "superseded", SupersededByApprovalID: "ap-new", BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-3", ActionKind: "promotion"}},
	}, now)
	joined := renderActionCenterItems(items)
	text := strings.Join(joined, "\n")
	mustContainAll(t, text,
		"approval ap-expired",
		"expired",
		"approval ap-soon",
		"expiring_soon",
		"approval ap-super",
		"superseded",
		"owner/action:",
		"owner: operator decision",
		"evidence:",
	)
}

func TestBuildOperationalAttentionItemsIncludesAuditAndWatchDisconnect(t *testing.T) {
	audit := &brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "degraded", CurrentlyDegraded: true, HardFailures: []string{"anchor_receipt_invalid"}}}
	watch := dashboardLiveActivity{
		runWatch:      watchFamilySummary{family: "run_watch", errorCount: 1, lastStatus: "watch_error"},
		approvalWatch: watchFamilySummary{family: "approval_watch", errorCount: 0, lastStatus: "ok"},
		sessionWatch:  watchFamilySummary{family: "session_watch", errorCount: 0, lastStatus: "ok"},
	}
	items := buildOperationalAttentionItems(audit, "", watch, shellSyncHealth{State: shellSyncStateDisconnected, ErrorText: "local_ipc_dial_error"}, []brokerapi.RunSummary{{RunID: "run-1", RuntimePostureDegraded: true}}, brokerapi.ProjectSubstratePostureGetResponse{})
	text := strings.Join(renderActionCenterItems(items), "\n")
	mustContainAll(t, text,
		"shell watch sync health",
		"state=disconnected",
		"audit verification posture",
		"anchoring=degraded",
		"run run-1 operational posture",
		"owner/action:",
	)
}

func TestActionCenterItemsStateReasonImpactOwnerTargetEvidence(t *testing.T) {
	model := newActionCenterRouteModel(routeDefinition{ID: routeAction, Label: "Action Center"}, &fakeBrokerClient{}).(actionCenterRouteModel)
	model.now = func() time.Time { return time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC) }
	model.runs = []brokerapi.RunSummary{{RunID: "run-1", LifecycleState: "waiting", PendingApprovalCount: 1}}
	model.approvals = []brokerapi.ApprovalSummary{{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-1", ActionKind: "promotion"}}}
	vm := model.snapshot()
	blockedItems := vm.Families[actionCenterFamilyBlocked]
	if len(blockedItems) == 0 {
		t.Fatal("expected blocked items")
	}
	item := blockedItems[0]
	if strings.TrimSpace(item.Reason) == "" || strings.TrimSpace(item.Impact) == "" || strings.TrimSpace(item.Owner) == "" || strings.TrimSpace(item.RequiredAction) == "" || strings.TrimSpace(item.TargetLabel) == "" || strings.TrimSpace(item.EvidenceCue) == "" {
		t.Fatalf("expected full item content, got %+v", item)
	}
	text := renderActionCenterItem(item, 0)
	mustContainAll(t, text,
		"reason:",
		"impact:",
		"owner/action:",
		"target:",
		"evidence:",
	)
}

func timeNowUTCForTest() time.Time {
	return time.Now().UTC()
}
