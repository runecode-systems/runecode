package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRunsRouteExplainsBrokerPostureAndStateTaxonomy(t *testing.T) {
	model := newRunsRouteModel(routeDefinition{ID: routeRuns, Label: "Runs"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeRuns})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)

	mustContainAll(t, view,
		"backend_kind=workspace",
		"Runtime isolation assurance (authoritative): runtime isolation=sandboxed",
		"Provisioning/binding posture (authoritative): provisioning posture=ok",
		"Audit posture (authoritative): audit posture=ok/degraded (unanchored/degraded)",
		"Approval profile (authoritative): approval_profile=n/a",
		"Authoritative broker state (control-plane truth):",
		"Advisory state (non-authoritative runner hints):",
		"Coordination summary: blocked=true wait_reason=approval_wait",
		"Blocking cue:",
		"APPROVAL_REQUIRED",
		"Stage summaries: 2 total, 1 with pending approvals",
		"Role summaries: 2 total, 1 reporting coordination waits",
	)
}

func TestApprovalsRouteDistinguishesCodesLifecycleAndBinding(t *testing.T) {
	model := newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeApprovals})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)

	mustContainAll(t, view,
		"Approval type: exact-action approval (binding_kind=exact_action)",
		"Approval safety strip",
		"Lifecycle state: pending (stale)",
		"Lifecycle reason code: awaiting_decision",
		"Policy reason code: requires_human_review",
		"Approval trigger code: policy_gate",
		"Distinct blocking semantics: trigger=policy_gate",
		"Execution/system errors: shown as load failures above",
		"What changes if approved: effect=unblock_next_stage summary=Promotion continues",
		"Canonical bound identity: request=sha256:req",
		"Exact bound scope: workspace=ws-1 run=run-1 stage=stage-1",
	)
}

func TestArtifactsRouteUsesTypedReadAndInspectableModes(t *testing.T) {
	model := newArtifactsRouteModel(routeDefinition{ID: routeArtifacts, Label: "Artifacts"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeArtifacts})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)

	mustContainAll(t, view,
		"Typed detail mode:",
		"Inspectable content is supplemental evidence, not authoritative run/approval truth.",
		"diff preview (secrets redacted):",
		"token=[REDACTED]",
	)

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	view = updated.View(120, 40, focusContent)
	if !strings.Contains(view, "log preview (secrets redacted):") {
		t.Fatalf("expected log preview mode after m, got %q", view)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	view = updated.View(120, 40, focusContent)
	if !strings.Contains(view, "result preview (secrets redacted):") {
		t.Fatalf("expected result preview mode after second m, got %q", view)
	}
}

func TestApprovalsRouteSupportsTypedResolveFlowPath(t *testing.T) {
	spy := newRecordingBrokerClient(&fakeBrokerClient{})
	model := newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, spy)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeApprovals})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Flow path: workspace=ws-1 run=run-1 stage=stage-1 action=promotion") {
		t.Fatalf("expected typed flow-path summary in view, got %q", view)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("expected typed resolve command when pressing a")
	}
	updated, cmd = updated.Update(cmd())
	if cmd == nil {
		t.Fatal("expected post-resolve reload command")
	}
	updated, _ = updated.Update(cmd())

	calls := spy.Calls()
	if !containsCall(calls, "ApprovalResolve") {
		t.Fatalf("expected ApprovalResolve typed call, got %v", calls)
	}

	view = updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Status: Approval ap-1 resolved via typed ApprovalResolve; run can continue.") {
		t.Fatalf("expected resolve status in view, got %q", view)
	}
}

func containsCall(calls []string, want string) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}
	return false
}

func mustContainAll(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			t.Fatalf("expected %q in view, got %q", needle, haystack)
		}
	}
}
