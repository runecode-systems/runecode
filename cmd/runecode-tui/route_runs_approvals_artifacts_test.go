package main

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestRunsRouteExplainsBrokerPostureAndStateTaxonomy(t *testing.T) {
	model := newRunsRouteModel(routeDefinition{ID: routeRuns, Label: "Runs"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeRuns})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 80, Focus: focusContent, Breakpoint: shellBreakpointWide})
	inspector := surface.Regions.Inspector.Body

	mustContainAll(t, inspector,
		"Summary: Run run-1 is active with 1 pending approval(s).",
		"Identity: run=run-1 workspace=ws-1 session=session-1",
		"Local actions: jump:session | jump:approvals | jump:artifacts | jump:audit | copy:run_id | copy:session_id",
		"Copy actions: run id | workspace id | session id | linked approval ids | audit refs | raw block",
		"Workflow operation: change_draft • stage=stage-1",
		"Plan authority: workflow definition hash",
		"Runner/reporting posture: runner=active • last checkpoint=approval_wait_entered",
		"Blocked/failure reason: approval wait",
		"Evidence links: approvals 1 • artifact classes 2 • active manifests 1 • policy refs 1",
		"Linked approvals: ap-1",
		"Artifacts trail: change_draft (1) • diffs (2)",
		"Audit evidence: manifest sha256:manifest • policy sha256:policy",
		"Navigation cues: session=session-1 approvals=1 artifacts=3 audit=2",
		"backend_kind=workspace",
		"Structured/raw modes expose workflow hashes",
		"Runtime assurance: Isolation: sandboxed",
	)
	mustContainAll(t, view,
		"Run overview",
		"Run run-1 is active and waiting on 1 approval.",
		"Next: Open Approvals to review 1 pending approval, then return here for the updated result.",
		"Safety and evidence",
		"Run directory",
	)
	if strings.Contains(view, "Summary: Run run-1 is active with 1 pending approval(s).") {
		t.Fatalf("expected run detail only in inspector region, got %q", view)
	}
	for _, banned := range []string{"Selected run", "workflow=", "operation=", "backend=", "audit=", "Modes:"} {
		if strings.Contains(view, banned) {
			t.Fatalf("expected %q to stay out of rendered runs pane, got %q", banned, view)
		}
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
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 80, Focus: focusContent, Breakpoint: shellBreakpointWide})
	inspector := surface.Regions.Inspector.Body

	mustContainAll(t, inspector,
		"Summary: approval=ap-1 state=approval required reason=policy requires operator review before promotion for run-1 can continue (requires_human_review)",
		"Identity: approval=ap-1 run=run-1 action=promotion",
		"Local actions: resolve:typed | jump:runs | jump:artifacts | jump:audit | copy:approval_id",
		"Copy actions: approval id | bound run id | request digest | decision digest | raw block",
		"Approval state: approval required",
		"Why this approval exists: policy requires operator review before promotion for run-1 can",
		"Exact gated object/action: run run-1 • stage stage-1 • action=promotion",
		"Review first: run evidence for run-1",
		"If approved next: Promotion continues (effect=unblock_next_stage)",
		"After approval route: Artifacts → Audit",
		"Resolve availability: unavailable here because promotion approvals must be completed in the prom",
		"Resolve unavailable because: promotion approvals must stay in the promotion flow",
		"Workflow posture: approval required; lifecycle=pending (stale)",
		"Structured/raw modes expose exact-action approval",
	)
	if !strings.Contains(view, "Approval review") {
		t.Fatalf("expected approval overview card in main view, got %q", view)
	}
	mustContainAll(t, view,
		"Decision workbench",
		"Approval queue",
	)
	if strings.Contains(view, "Summary: approval=ap-1 state=approval required") {
		t.Fatalf("expected approval detail only in inspector region, got %q", view)
	}
	for _, banned := range []string{"Modes:", "policy_reason_code=", "approval_trigger_code=", "reason=", "gate="} {
		if strings.Contains(view, banned) {
			t.Fatalf("expected %q to stay out of rendered approvals pane, got %q", banned, view)
		}
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
		"Summary: artifact=diffs for run-1 class=diffs bytes=128",
		"Identity: artifact=diffs for run-1 digest=sha256:bbbbbbbbbbbb",
		"Status: evidence_trail=run:run-1 -> artifact:sha256:bbbbbbbbbbbb -> audit -> verification posture",
		"Local actions: jump:runs | jump:audit | copy:digest | copy:provenance_receipt | copy:artifact_preview",
		"Copy actions: artifact digest | provenance receipt | artifact preview",
		"Evidence label: diffs for run-1",
		"Evidence trail: run run-1 -> artifact diffs for run-1 -> Audit for verification posture and anch",
		"Primary digest display: sha256:bbbbbbbbbbbb (copy raw digest below)",
		"Typed detail mode:",
		"Inspectable content is supplemental evidence, not authoritative run/approval truth.",
		"diff preview (secrets redacted):",
		"token=[REDACTED]",
	)
	mustContainAll(t, view,
		"Evidence workspace",
		"Evidence path: run run-1 → diffs for run-1 → Audit for verification posture and anchoring.",
		"Filter: all artifact classes",
		"diffs for run-1 sha256:bbbbbbbbbbbb • 128 bytes plain text preview",
	)
	if strings.Contains(view, "Summary: artifact=diffs for run-1") {
		t.Fatalf("expected artifact detail only in inspector region, got %q", view)
	}
	for _, banned := range []string{"Modes:", "class=", "bytes=", "run="} {
		if strings.Contains(view, banned) {
			t.Fatalf("expected %q to stay out of rendered artifacts pane, got %q", banned, view)
		}
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
	if !strings.Contains(view, "Evidence trail: review run evidence for run-1, then open Audit for policy and verification context.") {
		t.Fatalf("expected compact evidence guidance in view, got %q", view)
	}
	if !strings.Contains(view, "Decision path: review here, then continue in Artifacts → Audit for the final workflow-specific step.") {
		t.Fatalf("expected decision guidance in view, got %q", view)
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
	if !strings.Contains(view, "Status: Resolve unavailable: promotion approvals must be completed in the promotion flow so exact promotion binding stays intact.") {
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
	if !strings.Contains(view, "resolved; broker result=approval_consumed") {
		t.Fatalf("expected resolve success status in view, got %q", view)
	}
}

func TestRunInspectorContentIncludesPostureAndTrustCues(t *testing.T) {
	detail := mustFakeRunDetail(t)
	content := runInspectorContent(detail.Summary, detail, pendingApprovalStageCount(detail.StageSummaries), waitingRoleCount(detail.RoleSummaries), buildRunEvidenceLinks(detail), presentationRendered)
	mustContainAll(t, content,
		"Provisioning posture: Provisioning: attested",
		"PROVISIONING_OK",
		"Attestation truth: attestation posture=valid",
		"post-handshake verification succeeded; supported attested posture earned from verified post-handshake evidence",
		"Structured/raw modes expose workflow hashes",
	)
}

func TestApprovalInspectorContentIncludesPolicyAndTriggerCues(t *testing.T) {
	resp := mustFakeApprovalDetail(t)
	content := approvalInspectorContent(resp.Approval, resp.ApprovalDetail, resp.ApprovalDetail.BoundIdentity, resp.Approval.BoundScope, approvalBindingLabel(resp.ApprovalDetail.BindingKind), resp.ApprovalDetail.LifecycleDetail.LifecycleState, renderApprovalLifecycleFlags(resp.ApprovalDetail.LifecycleDetail), presentationRendered)
	mustContainAll(t, content,
		"Why this approval exists: policy requires operator review",
		"Safety cue: approval policy, lifecycle, and execution/system errors remain distinct",
		"Structured/raw modes expose exact-action approval, binding kind, trigger codes, policy reason codes",
	)
}

func TestAuditRouteInspectorShowsBoundedTrailAndCopyableDigest(t *testing.T) {
	model := newAuditRouteModel(routeDefinition{ID: routeAudit, Label: "Audit"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeAudit})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected record detail load command")
	}
	updated, _ = updated.Update(cmd())
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	inspector := surface.Regions.Inspector.Body
	mustContainAll(t, inspector,
		"Copy actions: record digest | linked references | raw block",
		"Primary digest display: sha256:aaaaaaaaaaaa (copy raw digest below)",
		"Evidence trail: workflow result -> artifacts -> audit records -> verification posture",
		"Trust posture: broker-linked references stay authoritative",
	)
	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Run state changed • sha256:aaaaaaaaaaaa • verification degraded") {
		t.Fatalf("expected product-language timeline directory row, got %q", view)
	}
	for _, banned := range []string{"digest=", "event=", "posture=", "refs=", "Modes:"} {
		if strings.Contains(view, banned) {
			t.Fatalf("expected %q to stay out of rendered audit pane, got %q", banned, view)
		}
	}
}

func mustFakeRunDetail(t *testing.T) *brokerapi.RunDetail {
	t.Helper()
	resp, err := (&fakeBrokerClient{}).RunGet(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("RunGet failed: %v", err)
	}
	return &resp.Run
}

func mustFakeApprovalDetail(t *testing.T) brokerapi.ApprovalGetResponse {
	t.Helper()
	resp, err := (&fakeBrokerClient{}).ApprovalGet(context.Background(), "ap-1")
	if err != nil {
		t.Fatalf("ApprovalGet failed: %v", err)
	}
	return resp
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
	mustContainAll(t, view,
		"Run overview",
		"Run run-2 is blocked.",
		"Run directory",
	)
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
	if !strings.Contains(surface.Regions.Inspector.Body, "Exact gated object/action: run run-2 • stage stage-2 • action=stage_summary_sign_off") {
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
	if !strings.Contains(view, "> build logs for run-2 sha256:cccccccccccc • 256 bytes plain text preview") {
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
	if !strings.Contains(view, "> diffs for run-1 sha256:bbbbbbbbbbbb • 128 bytes plain text preview") {
		t.Fatalf("expected fallback selection to first available artifact, got %q", view)
	}
	if !strings.Contains(updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide}).Regions.Inspector.Body, "artifact=diffs for run-1") {
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

func TestAttestationPostureCueClarifiesRuntimeIsolationAssurance(t *testing.T) {
	cue := renderAttestationPostureCue("valid", nil)
	if !strings.Contains(cue, "attestation posture=valid (evidence present, verification succeeded; isolation assurance varies by runtime posture)") {
		t.Fatalf("expected valid attestation cue to clarify runtime isolation assurance, got %q", cue)
	}
	if !strings.Contains(cue, "ATTESTATION_VALID") {
		t.Fatalf("expected ATTESTATION_VALID badge in cue, got %q", cue)
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
