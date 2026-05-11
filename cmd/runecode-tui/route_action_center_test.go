package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-systems/runecode/internal/brokerapi"
	"github.com/runecode-systems/runecode/internal/trustpolicy"
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
		"Approvals",
		"Operational Attention",
		"Blocked Work",
		"Focus Approvals",
		"Continue in",
	)
	mustNotContainAny(t, view,
		"Blocked-work impact",
		"owner/action:",
		"Focused queue:",
	)
	surface := updated.ShellSurface(routeShellContext{Width: 140, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	inspector := surface.Regions.Inspector.Body
	mustContainAll(t, inspector,
		"Why this needs attention",
		"What it affects",
		"What to do",
		"Continue in",
		"Evidence",
	)
	mustNotContainAny(t, inspector,
		"queue=",
		"state=",
		"required_action=",
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
	if !strings.Contains(view, "Focus Blocked Work") {
		t.Fatalf("expected family indicator in view, got %q", view)
	}
	surface := updated.ShellSurface(routeShellContext{Width: 140, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	if !strings.Contains(surface.Regions.Inspector.Body, "Continue in") {
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
		"expiring soon",
		"approval ap-super",
		"superseded",
		"Continue in Approvals",
	)
	mustNotContainAny(t, text, "owner/action:", "Next:")
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
		"Watch sync is disconnected",
		"audit verification posture",
		"receipts are degraded",
		"run run-1 operational posture",
		"Continue in",
	)
	mustNotContainAny(t, text, "owner/action:", "Next:")
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
		item.Title,
		actionCenterShortReason(item.Reason),
		"Continue in",
	)
	mustNotContainAny(t, text,
		"owner/action:",
		"target:",
		"impact:",
		"Next:",
	)
}

func timeNowUTCForTest() time.Time {
	return time.Now().UTC()
}

func mustNotContainAny(t *testing.T, body string, values ...string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(body, value) {
			t.Fatalf("expected %q to be absent from %q", value, body)
		}
	}
}
