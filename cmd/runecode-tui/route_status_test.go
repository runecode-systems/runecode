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
		"Broker lifecycle posture:",
		"attach_mode=full",
		"attachable=true",
		"normal_operation_allowed=true",
		"Broker lifecycle blocked reasons: none",
		"Broker lifecycle degraded reasons: none",
		"Attach guidance: full attach and normal operation are allowed.",
		"Project substrate posture:",
		"compatibility=supported_with_upgrade_available",
		"Project substrate remediation:",
		"Project substrate upgrade:",
		"a adopt substrate",
		"i init preview",
		"I init apply",
		"u upgrade preview",
		"U upgrade apply",
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
	key             rune
	expectedStatus  string
	expectedRPCCall []string
}

func statusRouteActionCases() []statusRouteActionCase {
	return []statusRouteActionCase{
		{key: 'a', expectedStatus: "Project substrate adopt status=", expectedRPCCall: []string{"ProjectSubstrateAdopt"}},
		{key: 'i', expectedStatus: "Project substrate init preview status=", expectedRPCCall: []string{"ProjectSubstrateInitPreview"}},
		{key: 'I', expectedStatus: "Project substrate init apply status=", expectedRPCCall: []string{"ProjectSubstrateInitApply"}},
		{key: 'u', expectedStatus: "Project substrate upgrade preview status=", expectedRPCCall: []string{"ProjectSubstrateUpgradePreview"}},
		{key: 'U', expectedStatus: "Project substrate upgrade apply status=", expectedRPCCall: []string{"ProjectSubstrateUpgradeApply"}},
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
	if cmd == nil {
		t.Fatalf("expected reload command after key %q", string(tc.key))
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, tc.expectedStatus) {
		t.Fatalf("expected status %q in view after key %q, got %q", tc.expectedStatus, string(tc.key), view)
	}
	assertStatusRouteViewRedactsPreviewHandles(t, view, tc.key)
	afterCalls := recording.Calls()[before:]
	assertStatusRouteCallsInclude(t, afterCalls, tc.expectedRPCCall, tc.key)
	if !containsCall(afterCalls, "ProjectSubstratePostureGet") {
		t.Fatalf("expected post-action posture reload after key %q; got %v", string(tc.key), afterCalls)
	}
	if !containsCall(afterCalls, "ProductLifecyclePostureGet") {
		t.Fatalf("expected post-action lifecycle posture reload after key %q; got %v", string(tc.key), afterCalls)
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
	if cmd == nil {
		t.Fatal("expected reload command after adopt")
	}
	updated, _ = updated.Update(cmd())
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
		"attach_mode=diagnostics_only",
		"lifecycle_posture=blocked",
		"attachable=true",
		"normal_operation_allowed=false",
		"Broker lifecycle blocked reasons: project_substrate_unsupported_too_new",
		"Broker lifecycle degraded reasons: project_substrate_upgrade_available",
		"Attach guidance: diagnostics/remediation-only attach is available; normal operation is blocked by current project-substrate posture.",
	)
}
