package main

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestStatusRouteRendersProjectSubstratePostureAndGuidance(t *testing.T) {
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"Managed operation",
		"Attach and normal managed operation are available.",
		"Overview: • broker connected • normal work available • local workspace mode",
		"Setup: setup validated • upgrade available",
		"Managed-operation summary:",
		"full managed access",
		"Normal work is available now",
		"Managed-operation blockers: none reported",
		"Managed-operation watchouts: none reported",
		"Project setup",
		"Project setup is usable, but a broker-owned upgrade is available.",
		"Project setup summary:",
		"Adopt (a): not currently available.",
		"Init (i/I): preview ready for review and optional apply (preview handle ready).",
		"Upgrade (u/U): preview ready for review and optional apply (preview handle ready).",
		"Project setup guidance",
		"r reload",
	)
}

func TestStatusRouteProjectSubstrateActionsUseTypedContracts(t *testing.T) {
	recording := newRecordingBrokerClient(&fakeBrokerClient{})
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, recording)
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	for _, tc := range statusRouteActionCases() {
		assertStatusRouteActionUsesTypedContracts(t, updated, recording, tc)
	}
}

type statusRouteActionCase struct {
	key              rune
	expectedStatus   string
	expectedSnippets []string
	expectedRPCCall  []string
	expectReload     bool
}

func statusRouteActionCases() []statusRouteActionCase {
	return []statusRouteActionCase{
		{key: 'a', expectedStatus: "Project setup adoption refreshed: status=compatible_existing", expectedSnippets: []string{"Compatible adoption", "existing compatible setup", "without changing repository files"}, expectedRPCCall: []string{"ProjectSubstrateAdopt"}},
		{key: 'i', expectedStatus: "Project setup init preview refreshed: status=ready_for_apply", expectedSnippets: []string{"Init preview", "Preview is preview ready for review and optional apply.", "Handle is <acquired>."}, expectedRPCCall: []string{"ProjectSubstrateInitPreview"}},
		{key: 'I', expectedStatus: "Project setup validation refreshed. Review the updated managed-operation and setup posture below.", expectedSnippets: []string{"Managed operation", "Project setup validation refreshed.", "Project setup guidance"}, expectedRPCCall: []string{"ProjectSubstrateInitApply"}, expectReload: true},
		{key: 'u', expectedStatus: "Project setup upgrade preview refreshed: status=ready_for_apply", expectedSnippets: []string{"Upgrade preview", "Preview is preview ready for review and optional apply.", "Digest is <acquired>."}, expectedRPCCall: []string{"ProjectSubstrateUpgradePreview"}},
		{key: 'U', expectedStatus: "Project setup validation refreshed. Review the updated managed-operation and setup posture below.", expectedSnippets: []string{"Managed operation", "Project setup validation refreshed.", "Project setup guidance"}, expectedRPCCall: []string{"ProjectSubstrateUpgradeApply"}, expectReload: true},
	}
}

func assertStatusRouteActionUsesTypedContracts(t *testing.T, model routeModel, recording *recordingBrokerClient, tc statusRouteActionCase) {
	t.Helper()
	before := len(recording.Calls())
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
	if cmd == nil {
		t.Fatalf("expected action command for key %q", string(tc.key))
	}
	updated, cmd = updated.Update(cmd())
	if tc.expectReload && cmd == nil {
		t.Fatalf("expected reload command after key %q", string(tc.key))
	}
	if tc.expectReload {
		updated, _ = updated.Update(cmd())
	}
	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, tc.expectedStatus) {
		t.Fatalf("expected status %q in view after key %q, got %q", tc.expectedStatus, string(tc.key), view)
	}
	for _, snippet := range tc.expectedSnippets {
		if !strings.Contains(view, snippet) {
			t.Fatalf("expected snippet %q in view after key %q, got %q", snippet, string(tc.key), view)
		}
	}
	assertStatusRouteViewRedactsPreviewHandles(t, view, tc.key)
	afterCalls := recording.Calls()[before:]
	assertStatusRouteCallsInclude(t, afterCalls, tc.expectedRPCCall, tc.key)
	if tc.expectReload && !containsCall(afterCalls, "ProjectSubstratePostureGet") {
		t.Fatalf("expected post-action posture reload after key %q; got %v", string(tc.key), afterCalls)
	}
	if tc.expectReload && !containsCall(afterCalls, "ProductLifecyclePostureGet") {
		t.Fatalf("expected post-action lifecycle posture reload after key %q; got %v", string(tc.key), afterCalls)
	}
}

type failingProjectSubstrateActionClient struct {
	*fakeBrokerClient
	initPreviewErr error
	postureErr     error
	postureCalls   int
}

func (f *failingProjectSubstrateActionClient) ProjectSubstrateInitPreview(ctx context.Context) (brokerapi.ProjectSubstrateInitPreviewResponse, error) {
	_, _ = f.fakeBrokerClient.ProjectSubstrateInitPreview(ctx)
	if f.initPreviewErr != nil {
		return brokerapi.ProjectSubstrateInitPreviewResponse{}, f.initPreviewErr
	}
	return f.fakeBrokerClient.ProjectSubstrateInitPreview(ctx)
}

func (f *failingProjectSubstrateActionClient) ProjectSubstratePostureGet(ctx context.Context) (brokerapi.ProjectSubstratePostureGetResponse, error) {
	f.postureCalls++
	if f.postureErr != nil && f.postureCalls > 1 {
		return brokerapi.ProjectSubstratePostureGetResponse{}, f.postureErr
	}
	return f.fakeBrokerClient.ProjectSubstratePostureGet(ctx)
}

func TestStatusRouteProjectSubstratePreviewFailureGuidesRetry(t *testing.T) {
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, &failingProjectSubstrateActionClient{fakeBrokerClient: &fakeBrokerClient{}, initPreviewErr: context.DeadlineExceeded})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if cmd == nil {
		t.Fatal("expected init preview action command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"Init preview",
		"RuneCode could not load the broker-owned init preview.",
		"Reload or retry init preview before any init apply.",
		"normal work blocked: project substrate posture blocks execution while validation is incompatible",
	)
}

type missingPreviewHandleClient struct {
	*fakeBrokerClient
}

func (f *missingPreviewHandleClient) ProjectSubstratePostureGet(ctx context.Context) (brokerapi.ProjectSubstratePostureGetResponse, error) {
	resp, err := f.fakeBrokerClient.ProjectSubstratePostureGet(ctx)
	if err != nil {
		return brokerapi.ProjectSubstratePostureGetResponse{}, err
	}
	resp.InitPreview.Status = ""
	resp.InitPreview.PreviewToken = ""
	return resp, nil
}

func TestStatusRouteProjectSubstrateApplyUnavailableExplainsPreviewRequirement(t *testing.T) {
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, &missingPreviewHandleClient{fakeBrokerClient: &fakeBrokerClient{}})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'I'}})
	if cmd == nil {
		t.Fatal("expected init apply action command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"Init apply",
		"Init apply is unavailable because no preview handle is currently published.",
		"Run init preview first, then apply only after reviewing the planned mutation.",
	)
}

func TestStatusRouteProjectSubstrateValidationReloadFailureStaysScoped(t *testing.T) {
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, &failingProjectSubstrateActionClient{fakeBrokerClient: &fakeBrokerClient{}, postureErr: context.DeadlineExceeded})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'U'}})
	if cmd == nil {
		t.Fatal("expected upgrade apply action command")
	}
	updated, cmd = updated.Update(cmd())
	if cmd == nil {
		t.Fatal("expected validation reload command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"Post-apply validation",
		"Project setup apply finished, but refreshed validation status is currently unavailable.",
		"Press r to retry validation refresh, then confirm managed-operation and setup posture before continuing.",
	)
	if strings.Contains(view, "Status is temporarily unavailable.") {
		t.Fatalf("expected scoped project-setup validation guidance, got %q", view)
	}
}

func TestStatusRouteSuccessfulValidationClearsFailureCard(t *testing.T) {
	model := statusRouteModel{
		def:                        routeDefinition{ID: routeStatus, Label: "Status"},
		validatingProjectSubstrate: true,
		loadSeq:                    1,
		projectSubstrateActionCard: projectSubstrateValidationFailureCard(brokerapi.ProjectSubstratePostureGetResponse{}, "open /home/user/.config/runecode/state.json: permission denied"),
	}
	updated, _ := model.Update(statusLoadedMsg{seq: 1, project: brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{ValidationState: "valid"}}})
	shell := updated.(statusRouteModel)
	if shell.projectSubstrateActionCard != nil {
		t.Fatal("expected successful validation reload to clear stale project substrate action card")
	}
	view := shell.View(120, 40, focusContent)
	if strings.Contains(view, "Post-apply validation") || strings.Contains(view, "permission denied") {
		t.Fatalf("expected stale validation failure card to be removed after success, got %q", view)
	}
}

func assertStatusRouteViewRedactsPreviewHandles(t *testing.T, view string, key rune) {
	t.Helper()
	if strings.Contains(view, "sha256:"+strings.Repeat("1", 64)) || strings.Contains(view, "sha256:"+strings.Repeat("0", 64)) {
		t.Fatalf("expected project substrate preview handles to stay redacted after key %q, got %q", string(key), view)
	}
}

func assertStatusRouteCallsInclude(t *testing.T, calls []string, want []string, key rune) {
	t.Helper()
	for _, name := range want {
		if !containsCall(calls, name) {
			t.Fatalf("expected call %q after key %q; got %v", name, string(key), calls)
		}
	}
}

func TestStatusRouteActivationUsesLifecyclePostureAndStatusContracts(t *testing.T) {
	recording := newRecordingBrokerClient(&fakeBrokerClient{})
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, recording)
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	assertStringSliceEqual(t, recording.Calls(), []string{"ReadinessGet", "VersionInfoGet", "ProjectSubstratePostureGet", "ProductLifecyclePostureGet", "BackendPostureGet"})

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("expected adopt action command")
	}
	updated, cmd = updated.Update(cmd())
	if cmd != nil {
		t.Fatal("did not expect reload command after adopt")
	}
	calls := recording.Calls()
	if !containsCall(calls, "ProjectSubstrateAdopt") {
		t.Fatalf("expected ProjectSubstrateAdopt call, got %v", calls)
	}
}

type diagnosticsOnlyLifecycleClient struct {
	*fakeBrokerClient
}

func (f *diagnosticsOnlyLifecycleClient) ProductLifecyclePostureGet(ctx context.Context) (brokerapi.ProductLifecyclePostureGetResponse, error) {
	_, _ = f.fakeBrokerClient.ProductLifecyclePostureGet(ctx)
	return brokerapi.ProductLifecyclePostureGetResponse{ProductLifecycle: brokerapi.BrokerProductLifecyclePosture{
		SchemaID:               "runecode.protocol.v0.BrokerProductLifecyclePosture",
		SchemaVersion:          "0.1.0",
		ProductInstanceID:      "repo-test",
		LifecycleGeneration:    "gen-blocked",
		AttachMode:             "diagnostics_only",
		LifecyclePosture:       "blocked",
		Attachable:             true,
		NormalOperationAllowed: false,
		BlockedReasonCodes:     []string{"project_substrate_unsupported_too_new"},
		DegradedReasonCodes:    []string{"project_substrate_upgrade_available"},
	}}, nil
}

func TestStatusRouteRendersDiagnosticsOnlyAttachGuidanceWhenNormalOperationBlocked(t *testing.T) {
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, &diagnosticsOnlyLifecycleClient{fakeBrokerClient: &fakeBrokerClient{}})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"Managed operation",
		"You can attach for diagnostics and remediation only; managed work stays blocked.",
		"diagnostics-only access",
		"Attach is available for inspection and remediation",
		"Managed-operation blockers: project substrate unsupported too new",
		"Managed-operation watchouts: project substrate upgrade available",
	)
}

type blockedProjectSubstrateStatusClient struct {
	*fakeBrokerClient
}

func (f *blockedProjectSubstrateStatusClient) ProductLifecyclePostureGet(ctx context.Context) (brokerapi.ProductLifecyclePostureGetResponse, error) {
	_, _ = f.fakeBrokerClient.ProductLifecyclePostureGet(ctx)
	return brokerapi.ProductLifecyclePostureGetResponse{ProductLifecycle: brokerapi.BrokerProductLifecyclePosture{
		SchemaID:               "runecode.protocol.v0.BrokerProductLifecyclePosture",
		SchemaVersion:          "0.1.0",
		ProductInstanceID:      "repo-test",
		LifecycleGeneration:    "gen-blocked-substrate",
		AttachMode:             "diagnostics_only",
		LifecyclePosture:       "blocked",
		Attachable:             true,
		NormalOperationAllowed: false,
		BlockedReasonCodes:     []string{"project_substrate_missing"},
	}}, nil
}

func (f *blockedProjectSubstrateStatusClient) ProjectSubstratePostureGet(ctx context.Context) (brokerapi.ProjectSubstratePostureGetResponse, error) {
	_, _ = f.fakeBrokerClient.ProjectSubstratePostureGet(ctx)
	return brokerapi.ProjectSubstratePostureGetResponse{
		SchemaID:       "runecode.protocol.v0.ProjectSubstratePostureGetResponse",
		SchemaVersion:  "0.1.0",
		RequestID:      "req-project-substrate-posture-blocked",
		RepositoryRoot: "/repo",
		PostureSummary: brokerapi.ProjectSubstratePostureSummary{
			SchemaID:               "runecode.protocol.v0.ProjectSubstratePostureSummary",
			SchemaVersion:          "0.1.0",
			ValidationState:        "missing",
			CompatibilityPosture:   "missing",
			NormalOperationAllowed: false,
			BlockedReasonCodes:     []string{"project_substrate_missing"},
		},
		BlockedExplanation:  "normal operation blocked by project substrate posture: project_substrate_missing",
		RemediationGuidance: []string{"inspect_project_substrate_posture", "initialize_canonical_runecontext_substrate", "revalidate_project_substrate"},
		InitPreview:         brokerapi.ProjectSubstrateInitPreviewResponse{Preview: brokerapi.ProjectSubstrateInitPreviewResponse{}.Preview}.Preview,
		UpgradePreview:      brokerapi.ProjectSubstrateUpgradePreviewResponse{}.Preview,
	}, nil
}

func TestStatusRouteRendersBlockedProjectSubstrateGuidance(t *testing.T) {
	model := newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, &blockedProjectSubstrateStatusClient{fakeBrokerClient: &fakeBrokerClient{}})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeStatus})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	mustContainAll(t, view,
		"Project setup",
		"Project setup needs attention before managed work can continue.",
		"Project setup summary:",
		"setup missing",
		"no compatible setup detected",
		"Normal work stays blocked until setup is fixed",
		"What blocks normal work: normal operation blocked by project substrate posture: project_substrate_missing",
		"Broker guidance: inspect project substrate posture, initialize canonical runecontext substrate, revalidate project substrate",
	)
}
