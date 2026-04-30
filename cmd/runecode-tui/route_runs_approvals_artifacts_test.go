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
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	inspector := surface.Regions.Inspector.Body

	mustContainAll(t, inspector,
		"Summary: run=run-1 lifecycle=n/a pending_approvals=0",
		"Identity: run=run-1 backend=workspace",
		"Local actions: jump:approvals | jump:artifacts | jump:audit | copy:run_id",
		"Copy actions: run id | raw block",
		"backend_kind=workspace",
		"Runtime isolation assurance (authoritative): runtime isolation=sandboxed",
		"Provisioning/binding posture (authoritative): provisioning posture=attested",
		"PROVISIONING_OK",
		"Attestation posture (authoritative): attestation posture=valid",
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
	if strings.Contains(view, "Summary: run=run-1 lifecycle=n/a pending_approvals=0") {
		t.Fatalf("expected run detail only in inspector region, got %q", view)
	}
}

func TestApprovalsRouteDistinguishesCodesLifecycleAndBinding(t *testing.T) {
	model := newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeApprovals})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	inspector := surface.Regions.Inspector.Body

	mustContainAll(t, inspector,
		"Summary: approval=ap-1 status=pending trigger=policy_gate",
		"Identity: approval=ap-1 run=run-1",
		"Local actions: resolve:typed | jump:runs | jump:audit | copy:approval_id",
		"Copy actions: approval id | bound run id | raw block",
		"Approval type: exact-action approval (binding_kind=exact_action)",
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
	if !strings.Contains(view, "Approval safety strip") {
		t.Fatalf("expected approval safety strip in main view, got %q", view)
	}
	if strings.Contains(view, "Summary: approval=ap-1 status=pending trigger=policy_gate") {
		t.Fatalf("expected approval detail only in inspector region, got %q", view)
	}
}

func TestArtifactsRouteUsesTypedReadAndInspectableModes(t *testing.T) {
	model := newArtifactsRouteModel(routeDefinition{ID: routeArtifacts, Label: "Artifacts"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeArtifacts})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	inspector := surface.Regions.Inspector.Body

	mustContainAll(t, inspector,
		"Summary: artifact=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb class=diffs bytes=128",
		"Identity: digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"Local actions: jump:runs | jump:audit | copy:digest | copy:provenance_receipt",
		"Copy actions: artifact digest | provenance receipt | artifact preview",
		"Typed detail mode:",
		"Inspectable content is supplemental evidence, not authoritative run/approval truth.",
		"diff preview (secrets redacted):",
		"token=[REDACTED]",
	)
	if strings.Contains(view, "Summary: artifact=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
		t.Fatalf("expected artifact detail only in inspector region, got %q", view)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	surface = updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	if !strings.Contains(surface.Regions.Inspector.Body, "log preview (secrets redacted):") {
		t.Fatalf("expected log preview mode after m, got %q", surface.Regions.Inspector.Body)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	surface = updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	if !strings.Contains(surface.Regions.Inspector.Body, "result preview (secrets redacted):") {
		t.Fatalf("expected result preview mode after second m, got %q", surface.Regions.Inspector.Body)
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
	if !strings.Contains(view, "typed approval_resolve -> resume signal") {
		t.Fatalf("expected typed resolve copy in flow path, got %q", view)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatal("expected resolve to fail closed until typed origin metadata is available")
	}

	calls := spy.Calls()
	if containsCall(calls, "ApprovalResolve") {
		t.Fatalf("expected ApprovalResolve not to be called, got %v", calls)
	}

	view = updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Status: promotion approvals must be resolved via promote-excerpt to preserve exact promotion binding") {
		t.Fatalf("expected fail-closed approval status in view, got %q", view)
	}
}

func TestApprovalsRouteResolvesBackendPostureViaTypedApprovalResolve(t *testing.T) {
	base := &backendResolveReadyBrokerClient{fakeBrokerClient: &fakeBrokerClient{}}
	spy := newRecordingBrokerClient(base)
	model := newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, spy)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeApprovals})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected detail load command for selected backend-posture approval")
	}
	updated, _ = updated.Update(cmd())
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("expected resolve command for backend-posture approval")
	}
	updated, cmd = updated.Update(cmd())
	if cmd == nil {
		t.Fatal("expected post-resolve reload command")
	}
	updated, _ = updated.Update(cmd())

	calls := spy.Calls()
	if !containsCall(calls, "ApprovalResolve") {
		t.Fatalf("expected ApprovalResolve to be called, got %v", calls)
	}
	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "resolved via typed ApprovalResolve") {
		t.Fatalf("expected resolve success status in view, got %q", view)
	}
}

func TestRunsReloadKeepsSelectedDetailAligned(t *testing.T) {
	model := newRunsRouteModel(routeDefinition{ID: routeRuns, Label: "Runs"}, &reloadAwareBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeRuns})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected reload command")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "> run-2") {
		t.Fatalf("expected run-2 to remain selected after reload, got %q", view)
	}
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	if !strings.Contains(surface.Regions.Inspector.Body, "backend_kind=container") {
		t.Fatalf("expected run-2 detail to remain active after reload, got %q", surface.Regions.Inspector.Body)
	}
}

func TestApprovalsReloadKeepsSelectedDetailAligned(t *testing.T) {
	model := newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, &reloadAwareBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeApprovals})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected reload command")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "> ap-2") {
		t.Fatalf("expected ap-2 to remain selected after reload, got %q", view)
	}
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	if !strings.Contains(surface.Regions.Inspector.Body, "Policy reason code: stage_sign_off_required") {
		t.Fatalf("expected ap-2 detail to remain active after reload, got %q", surface.Regions.Inspector.Body)
	}
}

func TestArtifactsReloadKeepsSelectedDetailAligned(t *testing.T) {
	model := newArtifactsRouteModel(routeDefinition{ID: routeArtifacts, Label: "Artifacts"}, &reloadAwareBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeArtifacts})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected reload command")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "> sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc") {
		t.Fatalf("expected second artifact to remain selected after reload, got %q", view)
	}
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	if !strings.Contains(surface.Regions.Inspector.Body, "Data class: build_logs") {
		t.Fatalf("expected selected artifact detail to remain active after reload, got %q", surface.Regions.Inspector.Body)
	}
}

func TestRunsReloadFallsBackWhenSelectedRunDisappears(t *testing.T) {
	model := newRunsRouteModel(routeDefinition{ID: routeRuns, Label: "Runs"}, &reloadAwareBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeRuns})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, cmd = updated.Update(runsSelectRunMsg{RunID: "run-missing"})
	if cmd == nil {
		t.Fatal("expected load command for missing run selection")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	if strings.Contains(view, "Load failed") {
		t.Fatalf("expected graceful fallback instead of load failure, got %q", view)
	}
	if !strings.Contains(view, "> run-1") {
		t.Fatalf("expected fallback selection to available run, got %q", view)
	}
	if !strings.Contains(updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body, "run=run-1") {
		t.Fatalf("expected fallback detail for run-1, got %q", updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body)
	}
}

func TestApprovalsReloadFallsBackWhenSelectedApprovalDisappears(t *testing.T) {
	model := newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, &reloadAwareBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeApprovals})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, cmd = updated.Update(approvalsSelectMsg{ApprovalID: "ap-missing"})
	if cmd == nil {
		t.Fatal("expected load command for missing approval selection")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	if strings.Contains(view, "Load failed") {
		t.Fatalf("expected graceful fallback instead of load failure, got %q", view)
	}
	if !strings.Contains(view, "> ap-1") {
		t.Fatalf("expected fallback selection to available approval, got %q", view)
	}
	if !strings.Contains(updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body, "approval=ap-1") {
		t.Fatalf("expected fallback detail for ap-1, got %q", updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body)
	}
}

func TestArtifactsReloadFallsBackWhenSelectedArtifactDisappears(t *testing.T) {
	model := newArtifactsRouteModel(routeDefinition{ID: routeArtifacts, Label: "Artifacts"}, &reloadAwareBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeArtifacts})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, cmd = updated.Update(artifactsSelectDigestMsg{Digest: "sha256:missing"})
	if cmd == nil {
		t.Fatal("expected load command for missing artifact selection")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	if strings.Contains(view, "Load failed") {
		t.Fatalf("expected graceful fallback instead of load failure, got %q", view)
	}
	if !strings.Contains(view, "> sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
		t.Fatalf("expected fallback selection to first available artifact, got %q", view)
	}
	if !strings.Contains(updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body, "artifact=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
		t.Fatalf("expected fallback detail for first artifact, got %q", updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body)
	}
}

func TestRouteInspectorViewportScrollAndResizePersistence(t *testing.T) {
	runs := newRunsRouteModel(routeDefinition{ID: routeRuns, Label: "Runs"}, &fakeBrokerClient{})
	updated, cmd := runs.Update(routeActivatedMsg{RouteID: routeRuns})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(routeViewportResizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(routeViewportScrollMsg{Region: routeRegionInspector, Delta: 4})
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 28, Focus: focusContent, Focused: routeRegionInspector, Breakpoint: shellBreakpointWide})
	if !strings.Contains(surface.Regions.Inspector.Body, "offset=4") {
		t.Fatalf("expected inspector viewport offset after scroll, got %q", surface.Regions.Inspector.Body)
	}

	updated, _ = updated.Update(routeViewportResizeMsg{Width: 140, Height: 30})
	surface = updated.ShellSurface(routeShellContext{Width: 140, Height: 30, Focus: focusContent, Focused: routeRegionInspector, Breakpoint: shellBreakpointWide})
	if !strings.Contains(surface.Regions.Inspector.Body, "offset=4") {
		t.Fatalf("expected offset persisted across resize, got %q", surface.Regions.Inspector.Body)
	}
	if !strings.Contains(surface.Regions.Inspector.Body, "viewport") {
		t.Fatalf("expected viewport metadata after resize, got %q", surface.Regions.Inspector.Body)
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
