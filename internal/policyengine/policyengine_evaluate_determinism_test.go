package policyengine

import (
	"strings"
	"testing"
)

func TestEvaluateDeterministicForCanonicalActionKinds(t *testing.T) {
	for _, tc := range deterministicActionKindCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			assertDeterministicDecision(t, tc.ctx, tc.action)
		})
	}
}

type deterministicCase struct {
	name   string
	ctx    *CompiledContext
	action ActionRequest
}

func deterministicActionKindCases(t *testing.T) []deterministicCase {
	return []deterministicCase{
		{name: "workspace_write", ctx: mustCompile(t, compileInputWithOneCapability("cap_stage")), action: validWorkspaceWriteActionRequest("cap_stage")},
		{name: "executor_run", ctx: mustCompile(t, compileInputWithOneCapability("cap_stage")), action: validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"go", "test", "./..."})},
		{name: "artifact_read", ctx: mustCompile(t, compileInputWithOneCapability("cap_stage")), action: validArtifactReadActionRequest("cap_stage")},
		{name: "promotion", ctx: mustCompile(t, compileInputWithOneCapability("cap_stage")), action: validPromotionActionRequest("cap_stage")},
		{name: "backend_posture_change", ctx: mustCompile(t, compileInputWithOneCapability("cap_stage")), action: validBackendPostureActionRequest("cap_stage")},
		{name: "action_gate_override", ctx: mustCompile(t, compileInputWithOneCapability("cap_stage")), action: validGateOverrideActionRequest("cap_stage")},
		{name: "stage_summary_sign_off", ctx: mustCompile(t, compileInputWithOneCapability("cap_stage")), action: validStageSummarySignOffActionRequest("cap_stage", "sha256:"+strings.Repeat("6", 64))},
		{name: "gateway_egress", ctx: mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text"))), action: validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)},
		{name: "dependency_fetch", ctx: mustCompile(t, compileGatewayInputWithOneCapability("dependency-fetch", "cap_dep", validAllowlistPayloadForGateway("allowlist-dep", "dependency-fetch", "package_registry", "fetch_dependency", "spec_text"))), action: validDependencyFetchActionRequest("cap_dep", "dependency-fetch", "allowlist-dep")},
	}
}

func assertDeterministicDecision(t *testing.T, ctx *CompiledContext, action ActionRequest) {
	t.Helper()
	decisionA, err := Evaluate(ctx, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	decisionB, err := Evaluate(ctx, action)
	if err != nil {
		t.Fatalf("Evaluate second pass returned error: %v", err)
	}
	if decisionA.ManifestHash == "" || decisionA.ActionRequestHash == "" {
		t.Fatalf("decision hashes must be set: %+v", decisionA)
	}
	if len(decisionA.PolicyInputHashes) == 0 {
		t.Fatal("policy_input_hashes must be non-empty")
	}
	if decisionA.ManifestHash != ctx.ManifestHash {
		t.Fatalf("manifest_hash = %q, want compiled %q", decisionA.ManifestHash, ctx.ManifestHash)
	}
	if decisionA.DecisionOutcome != decisionB.DecisionOutcome || decisionA.PolicyReasonCode != decisionB.PolicyReasonCode || decisionA.ActionRequestHash != decisionB.ActionRequestHash {
		t.Fatalf("decision not deterministic: first=%+v second=%+v", decisionA, decisionB)
	}
}

func TestActionRequestCanonicalHashIsStableAcrossPayloadMapOrder(t *testing.T) {
	actionA := validWorkspaceWriteActionRequest("cap_stage")
	actionB := validWorkspaceWriteActionRequest("cap_stage")
	actionA.ActionPayload = map[string]any{"schema_id": actionPayloadWorkspaceSchemaID, "schema_version": "0.1.0", "target_path": "src/main.go", "write_mode": "update", "bytes": float64(123)}
	actionB.ActionPayload = map[string]any{"bytes": float64(123), "write_mode": "update", "target_path": "src/main.go", "schema_version": "0.1.0", "schema_id": actionPayloadWorkspaceSchemaID}
	hashA, err := canonicalHashValue(actionA)
	if err != nil {
		t.Fatalf("canonicalHashValue(actionA) returned error: %v", err)
	}
	hashB, err := canonicalHashValue(actionB)
	if err != nil {
		t.Fatalf("canonicalHashValue(actionB) returned error: %v", err)
	}
	if hashA != hashB {
		t.Fatalf("canonical hash mismatch: %q != %q", hashA, hashB)
	}
}
