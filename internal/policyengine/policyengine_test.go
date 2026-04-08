package policyengine

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestCompileBuildsEffectiveContextWithFrozenPrecedenceAndHashes(t *testing.T) {
	input := CompileInput{
		FixedInvariants: FixedInvariants{DeniedCapabilities: []string{"always_denied"}, DeniedActionKinds: []string{"backend_posture_change"}},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		StageManifest:   ptr(testManifestInput(t, validStageCapabilityManifestPayload(), "")),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-c"), ""),
		},
	}

	compiled, err := Compile(input)
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if compiled.ManifestHash == "" {
		t.Fatal("ManifestHash must be set")
	}
	if len(compiled.PolicyInputHashes) != 6 {
		t.Fatalf("PolicyInputHashes len = %d, want 6", len(compiled.PolicyInputHashes))
	}
	if got := compiled.Context.EffectiveCapabilities; len(got) != 1 || got[0] != "cap_stage" {
		t.Fatalf("EffectiveCapabilities = %v, want [cap_stage]", got)
	}
	if got := compiled.Context.ActiveAllowlistRefs; len(got) != 3 {
		t.Fatalf("ActiveAllowlistRefs len = %d, want 3", len(got))
	}
}

func TestCompileFailsClosedWhenActiveAllowlistMissing(t *testing.T) {
	input := CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists:      []ManifestInput{},
	}

	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	var evalErr *EvaluationError
	ok := errors.As(err, &evalErr)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerLimitPolicyReject {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerLimitPolicyReject)
	}
}

func TestCompileRejectsSchemaVersionMismatch(t *testing.T) {
	role := validRoleManifestPayload()
	role["schema_version"] = "9.9.9"
	input := CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, role, ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}

	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	var evalErr *EvaluationError
	ok := errors.As(err, &evalErr)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeUnsupportedSchemaVersion {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeUnsupportedSchemaVersion)
	}
}

func TestCompileFailsClosedOnUnknownGatewayScopeKind(t *testing.T) {
	allowlist := validAllowlistPayload("allowlist-a")
	entries := allowlist["entries"].([]any)
	rule := entries[0].(map[string]any)
	rule["scope_kind"] = "gateway_destination_legacy"

	input := CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, allowlist, ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}

	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	var evalErr *EvaluationError
	ok := errors.As(err, &evalErr)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema && evalErr.Code != ErrCodeBrokerValidationOperation {
		t.Fatalf("error code = %q, want schema/operation validation fail-closed", evalErr.Code)
	}
}

func TestCompileFailsClosedOnUnknownDestinationDescriptorKind(t *testing.T) {
	allowlist := validAllowlistPayload("allowlist-a")
	entries := allowlist["entries"].([]any)
	rule := entries[0].(map[string]any)
	destination := rule["destination"].(map[string]any)
	destination["descriptor_kind"] = "raw_url"

	input := CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, allowlist, ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}

	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	var evalErr *EvaluationError
	ok := errors.As(err, &evalErr)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema && evalErr.Code != ErrCodeBrokerValidationOperation {
		t.Fatalf("error code = %q, want schema/operation validation fail-closed", evalErr.Code)
	}
}

func TestEvaluateUsesPrecedenceDenyThenApprovalThenAllow(t *testing.T) {
	rules := validRuleSetPayload()
	rules["rules"] = []any{
		map[string]any{"rule_id": "allow-1", "effect": "allow", "action_kind": "workspace_write", "capability_id": "cap_stage", "reason_code": "allow_manifest_opt_in", "details_schema_id": "runecode.protocol.details.policy.allow.v0"},
		map[string]any{"rule_id": "approval-1", "effect": "require_human_approval", "action_kind": "workspace_write", "capability_id": "cap_stage", "reason_code": "approval_required", "details_schema_id": "runecode.protocol.details.policy.approval.v0"},
		map[string]any{"rule_id": "deny-1", "effect": "deny", "action_kind": "workspace_write", "capability_id": "cap_stage", "reason_code": "deny_by_default", "details_schema_id": "runecode.protocol.details.policy.deny.v0"},
	}

	compiled := mustCompile(t, CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		StageManifest:   ptr(testManifestInput(t, validStageCapabilityManifestPayload(), "")),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-c"), ""),
		},
		RuleSet: ptr(testManifestInput(t, rules, "")),
	})

	decision, err := Evaluate(compiled, validWorkspaceWriteActionRequest("cap_stage"))
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateBindsDecisionToCompiledContextHash(t *testing.T) {
	compiledA := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	compiledB := mustCompile(t, compileInputWithOneCapability("cap_other"))

	decisionA, err := Evaluate(compiledA, validWorkspaceWriteActionRequest("cap_stage"))
	if err != nil {
		t.Fatalf("Evaluate A returned error: %v", err)
	}
	decisionB, err := Evaluate(compiledB, validWorkspaceWriteActionRequest("cap_other"))
	if err != nil {
		t.Fatalf("Evaluate B returned error: %v", err)
	}
	if decisionA.ManifestHash == decisionB.ManifestHash {
		t.Fatalf("ManifestHash collision: both were %q", decisionA.ManifestHash)
	}
}

func TestEvaluateFailsClosedOnUnknownActionKind(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.ActionKind = "unknown_kind"
	action.ActionPayloadSchemaID = actionPayloadWorkspaceSchemaID

	_, err := Evaluate(compiled, action)
	if err == nil {
		t.Fatal("Evaluate returned nil error, want failure")
	}
	evalErr, ok := err.(*EvaluationError)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerValidationSchema)
	}
}

func TestEvaluateFailsClosedOnUnknownActionPayloadSchemaID(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.ActionPayloadSchemaID = "runecode.protocol.v0.ActionPayloadUnknown"

	_, err := Evaluate(compiled, action)
	if err == nil {
		t.Fatal("Evaluate returned nil error, want failure")
	}
	evalErr, ok := err.(*EvaluationError)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerValidationSchema)
	}
}

func TestEvaluateDeniesNetworkEgressForWorkspaceRole(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validGatewayEgressActionRequest("cap_stage", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got := decision.Details["invariant"]; got != "no_escalation_in_place" {
		t.Fatalf("invariant = %v, want no_escalation_in_place", got)
	}
}

func TestEvaluateDeniesDependencyFetchWhenNotUsingDependencyRoleAndKind(t *testing.T) {
	role := validRoleManifestPayload()
	role["role_family"] = "gateway"
	role["role_kind"] = "dependency-fetch"
	principal := role["principal"].(map[string]any)
	principal["role_family"] = "gateway"
	principal["role_kind"] = "dependency-fetch"
	run := validRunCapabilityManifestPayload()
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_family"] = "gateway"
	runPrincipal["role_kind"] = "dependency-fetch"

	compiled := mustCompile(t, CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    mustManifestInput(role),
		RunManifest:     mustManifestInput(run),
		Allowlists: []ManifestInput{
			mustManifestInput(validAllowlistPayload("allowlist-a")),
			mustManifestInput(validAllowlistPayload("allowlist-b")),
		},
	})
	action := validGatewayEgressActionRequest("cap_run", "gateway", "dependency-fetch", "dependency-fetch", "model_endpoint", ActionKindDependencyFetch)

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got := decision.Details["invariant"]; got != "dependency_behavior_split" {
		t.Fatalf("invariant = %v, want dependency_behavior_split", got)
	}
}

func TestEvaluateExecutorSystemModifyingRequiresHardFloorApproval(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "system_modifying", []string{"apt-get", "install", "jq"})

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionRequireHumanApproval)
	}
	classes, ok := decision.Details["hard_floor_operation_classes"].([]string)
	if !ok {
		t.Fatalf("hard_floor_operation_classes type = %T, want []string", decision.Details["hard_floor_operation_classes"])
	}
	if len(classes) != 1 || classes[0] != string(HardFloorSecurityPostureWeakening) {
		t.Fatalf("hard_floor_operation_classes = %v, want [%s]", classes, HardFloorSecurityPostureWeakening)
	}
	if got := decision.Details["required_assurance_floor"]; got != string(ApprovalAssuranceReauthenticated) {
		t.Fatalf("required_assurance_floor = %v, want %s", got, ApprovalAssuranceReauthenticated)
	}
}

func TestEvaluateBackendSelectionRulesMicroVMDefaultAllow(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindBackendPosture,
		CapabilityID:          "cap_stage",
		ActionPayloadSchemaID: actionPayloadBackendSchemaID,
		ActionPayload: map[string]any{
			"schema_id":         actionPayloadBackendSchemaID,
			"schema_version":    "0.1.0",
			"backend_class":     "microvm",
			"change_kind":       "select_backend",
			"requested_posture": "microvm_default",
			"requires_opt_in":   false,
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionAllow)
	}
	if decision.PolicyReasonCode != "allow_microvm_default" {
		t.Fatalf("policy_reason_code = %q, want allow_microvm_default", decision.PolicyReasonCode)
	}
}

func TestEvaluateBackendSelectionRulesContainerRequiresExplicitOptIn(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindBackendPosture,
		CapabilityID:          "cap_stage",
		ActionPayloadSchemaID: actionPayloadBackendSchemaID,
		ActionPayload: map[string]any{
			"schema_id":         actionPayloadBackendSchemaID,
			"schema_version":    "0.1.0",
			"backend_class":     "container",
			"change_kind":       "select_backend",
			"requested_posture": "container_opt_in",
			"requires_opt_in":   false,
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if decision.PolicyReasonCode != "deny_container_opt_in_required" {
		t.Fatalf("policy_reason_code = %q, want deny_container_opt_in_required", decision.PolicyReasonCode)
	}
}

func TestEvaluateBackendSelectionRulesNoAutomaticFallback(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindBackendPosture,
		CapabilityID:          "cap_stage",
		ActionPayloadSchemaID: actionPayloadBackendSchemaID,
		ActionPayload: map[string]any{
			"schema_id":         actionPayloadBackendSchemaID,
			"schema_version":    "0.1.0",
			"backend_class":     "container",
			"change_kind":       "select_backend",
			"requested_posture": "container_fallback",
			"requires_opt_in":   true,
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if decision.PolicyReasonCode != "deny_container_automatic_fallback" {
		t.Fatalf("policy_reason_code = %q, want deny_container_automatic_fallback", decision.PolicyReasonCode)
	}
}

func TestEvaluatePolicyDecisionCarriesRelevantArtifactHashes(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.RelevantArtifactHashes = []trustpolicy.Digest{
		{HashAlg: "sha256", Hash: strings.Repeat("1", 64)},
		{HashAlg: "sha256", Hash: strings.Repeat("2", 64)},
	}

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(decision.RelevantArtifactHashes) != 2 {
		t.Fatalf("relevant_artifact_hashes len = %d, want 2", len(decision.RelevantArtifactHashes))
	}
	if decision.RelevantArtifactHashes[0] != "sha256:"+strings.Repeat("1", 64) {
		t.Fatalf("first relevant_artifact_hash = %q, want sha256:...1", decision.RelevantArtifactHashes[0])
	}
}

func TestEvaluateGateOverrideMatchesMultipleHardFloorClasses(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindGateOverride,
		CapabilityID:          "cap_stage",
		ActionPayloadSchemaID: actionPayloadGateSchemaID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadGateSchemaID,
			"schema_version": "0.1.0",
			"gate_name":      "policy-engine",
			"override_mode":  "break_glass",
			"justification":  "Emergency trust maintenance",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionRequireHumanApproval)
	}
	classes, ok := decision.Details["hard_floor_operation_classes"].([]string)
	if !ok {
		t.Fatalf("hard_floor_operation_classes type = %T, want []string", decision.Details["hard_floor_operation_classes"])
	}
	if len(classes) != 2 {
		t.Fatalf("hard_floor_operation_classes len = %d, want 2", len(classes))
	}
	if got := decision.Details["required_assurance_floor"]; got != string(ApprovalAssuranceHardwareBacked) {
		t.Fatalf("required_assurance_floor = %v, want %s", got, ApprovalAssuranceHardwareBacked)
	}
}

func TestEvaluateDeterministicForCanonicalActionKinds(t *testing.T) {
	tests := []struct {
		name   string
		ctx    *CompiledContext
		action ActionRequest
	}{
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decisionA, err := Evaluate(tc.ctx, tc.action)
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			decisionB, err := Evaluate(tc.ctx, tc.action)
			if err != nil {
				t.Fatalf("Evaluate second pass returned error: %v", err)
			}
			if decisionA.ManifestHash == "" || decisionA.ActionRequestHash == "" {
				t.Fatalf("decision hashes must be set: %+v", decisionA)
			}
			if len(decisionA.PolicyInputHashes) == 0 {
				t.Fatal("policy_input_hashes must be non-empty")
			}
			if decisionA.ManifestHash != tc.ctx.ManifestHash {
				t.Fatalf("manifest_hash = %q, want compiled %q", decisionA.ManifestHash, tc.ctx.ManifestHash)
			}
			if decisionA.DecisionOutcome != decisionB.DecisionOutcome || decisionA.PolicyReasonCode != decisionB.PolicyReasonCode || decisionA.ActionRequestHash != decisionB.ActionRequestHash {
				t.Fatalf("decision not deterministic: first=%+v second=%+v", decisionA, decisionB)
			}
		})
	}
}

func TestEvaluateGatewayRequiresSignedAllowlistDestinationMatch(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionAllow)
	}

	action.ActionPayload["destination_ref"] = "not-allowlisted.example.com"
	decision, err = Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got := decision.Details["reason"]; got != "destination_not_allowlisted" {
		t.Fatalf("reason = %v, want destination_not_allowlisted", got)
	}

	action.ActionPayload["destination_ref"] = "allowlist-model.example.com:8443"
	decision, err = Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateModerateProfileAllowsTypicalOfflineWorkspaceEditWithoutIntermediateApproval(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.ActionPayload["target_path"] = "src/offline_edit.go"

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionAllow)
	}
}

func TestEvaluateModerateProfileRequiresApprovalForWindowsAbsoluteWorkspaceWrite(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.ActionPayload["target_path"] = `C:\Windows\System32\drivers\etc\hosts`

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionRequireHumanApproval)
	}
	if trigger, _ := decision.RequiredApproval["approval_trigger_code"].(string); trigger != "out_of_workspace_write" {
		t.Fatalf("approval_trigger_code = %q, want out_of_workspace_write", trigger)
	}
}

func TestEvaluateDeniesWrappedShellForWorkspaceOrdinaryExecutor(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"env", "FOO=bar", "sh", "-c", "whoami"})

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestExactActionApprovalBindingChangesWithActionHash(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	actionA := validGateOverrideActionRequest("cap_stage")
	actionB := validGateOverrideActionRequest("cap_stage")
	actionB.ActionPayload["justification"] = "Emergency trust maintenance - revised"

	decisionA, err := Evaluate(compiled, actionA)
	if err != nil {
		t.Fatalf("Evaluate(actionA) returned error: %v", err)
	}
	decisionB, err := Evaluate(compiled, actionB)
	if err != nil {
		t.Fatalf("Evaluate(actionB) returned error: %v", err)
	}
	if decisionA.ActionRequestHash == decisionB.ActionRequestHash {
		t.Fatalf("action_request_hash should differ when exact action payload changes: %q", decisionA.ActionRequestHash)
	}
}

func TestCompileFailsClosedOnUnknownApprovalProfile(t *testing.T) {
	role := validRoleManifestPayload()
	role["approval_profile"] = "legacy"
	input := CompileInput{
		RoleManifest: testManifestInput(t, role, ""),
		RunManifest:  testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}
	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	evalErr, ok := err.(*EvaluationError)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationOperation {
		if evalErr.Code != ErrCodeBrokerValidationSchema {
			t.Fatalf("error code = %q, want fail-closed validation rejection", evalErr.Code)
		}
	}
}

func TestEvaluateFailsClosedOnUnknownRoleKind(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.RoleKind = "workspace-admin"

	_, err := Evaluate(compiled, action)
	if err == nil {
		t.Fatal("Evaluate returned nil error, want failure")
	}
	evalErr, ok := err.(*EvaluationError)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerValidationSchema)
	}
}

func TestEvaluateDependencyFetchUsesDependencyTriggerCode(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("dependency-fetch", "cap_dep", validAllowlistPayloadForGateway("allowlist-dep", "dependency-fetch", "package_registry", "enable_dependency_fetch", "spec_text")))
	action := validDependencyFetchActionRequest("cap_dep", "dependency-fetch", "allowlist-dep")
	action.ActionPayload["operation"] = "enable_dependency_fetch"

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionRequireHumanApproval)
	}
	if trigger, _ := decision.RequiredApproval["approval_trigger_code"].(string); trigger != "dependency_network_fetch" {
		t.Fatalf("approval_trigger_code = %q, want dependency_network_fetch", trigger)
	}
}

func TestEvaluateGatewayEgressNonCheckpointActionAllowedWithoutIntermediateApproval(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionAllow)
	}
}

func TestEvaluateSecretAccessRequiresModerateApproval(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validSecretAccessActionRequest("cap_stage")

	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionRequireHumanApproval)
	}
	if trigger, _ := decision.RequiredApproval["approval_trigger_code"].(string); trigger != "secret_access_lease" {
		t.Fatalf("approval_trigger_code = %q, want secret_access_lease", trigger)
	}
}

func TestClassifyHardFloorOperationCoversBackendAndAuthoritativePromotion(t *testing.T) {
	promotion := validPromotionActionRequest("cap_stage")
	promotion.ActionPayload["authoritative_import"] = true
	classes, _ := classifyHardFloorOperation(promotion, nil)
	if !containsHardFloorClass(classes, HardFloorAuthoritativeStateReconciliation) {
		t.Fatalf("classes = %v, want authoritative_state_reconciliation", classes)
	}
}

func containsHardFloorClass(classes []HardFloorOperationClass, target HardFloorOperationClass) bool {
	for _, class := range classes {
		if class == target {
			return true
		}
	}
	return false
}

func TestStageSignOffStaleBindingChangesWithSummaryHash(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	actionA := validStageSummarySignOffActionRequest("cap_stage", "sha256:"+strings.Repeat("7", 64))
	actionB := validStageSummarySignOffActionRequest("cap_stage", "sha256:"+strings.Repeat("8", 64))

	decisionA, err := Evaluate(compiled, actionA)
	if err != nil {
		t.Fatalf("Evaluate(actionA) returned error: %v", err)
	}
	decisionB, err := Evaluate(compiled, actionB)
	if err != nil {
		t.Fatalf("Evaluate(actionB) returned error: %v", err)
	}
	if decisionA.ActionRequestHash == decisionB.ActionRequestHash {
		t.Fatalf("action_request_hash should differ when stage summary hash changes: %q", decisionA.ActionRequestHash)
	}
	if got := decisionA.RequiredApproval["stage_summary_staleness_posture"]; got != "invalidate_on_bound_input_change" {
		t.Fatalf("stage_summary_staleness_posture = %v, want invalidate_on_bound_input_change", got)
	}
}

func TestActionRequestCanonicalHashIsStableAcrossPayloadMapOrder(t *testing.T) {
	actionA := validWorkspaceWriteActionRequest("cap_stage")
	actionB := validWorkspaceWriteActionRequest("cap_stage")
	actionA.ActionPayload = map[string]any{
		"schema_id":      actionPayloadWorkspaceSchemaID,
		"schema_version": "0.1.0",
		"target_path":    "src/main.go",
		"write_mode":     "update",
		"bytes":          float64(123),
	}
	actionB.ActionPayload = map[string]any{
		"bytes":          float64(123),
		"write_mode":     "update",
		"target_path":    "src/main.go",
		"schema_version": "0.1.0",
		"schema_id":      actionPayloadWorkspaceSchemaID,
	}

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

func mustCompile(t *testing.T, input CompileInput) *CompiledContext {
	t.Helper()
	compiled, err := Compile(input)
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	return compiled
}

func compileInputWithOneCapability(capability string) CompileInput {
	role := validRoleManifestPayload()
	role["capability_opt_ins"] = []any{capability}
	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{capability}
	stage := validStageCapabilityManifestPayload()
	stage["capability_opt_ins"] = []any{capability}
	return CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    mustManifestInput(role),
		RunManifest:     mustManifestInput(run),
		StageManifest:   ptr(mustManifestInput(stage)),
		Allowlists: []ManifestInput{
			mustManifestInput(validAllowlistPayload("allowlist-a")),
			mustManifestInput(validAllowlistPayload("allowlist-b")),
			mustManifestInput(validAllowlistPayload("allowlist-c")),
		},
	}
}

func mustManifestInput(value map[string]any) ManifestInput {
	b, _ := json.Marshal(value)
	h, _ := canonicalHashBytes(b)
	return ManifestInput{Payload: b, ExpectedHash: h}
}

func testManifestInput(t *testing.T, value map[string]any, expectedHash string) ManifestInput {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if expectedHash == "" {
		expectedHash, err = canonicalHashBytes(b)
		if err != nil {
			t.Fatalf("canonicalHashBytes returned error: %v", err)
		}
	}
	return ManifestInput{Payload: b, ExpectedHash: expectedHash}
}

func validRoleManifestPayload() map[string]any {
	return map[string]any{
		"schema_id":          roleManifestSchemaID,
		"schema_version":     roleManifestSchemaVersion,
		"principal":          map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1", "role_family": "workspace", "role_kind": "workspace-edit"},
		"role_family":        "workspace",
		"role_kind":          "workspace-edit",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_run", "cap_stage", "always_denied"},
		"allowlist_refs":     []any{mustDigestObject(mustAllowlistHash(validAllowlistPayload("allowlist-a")))},
		"signatures":         []any{map[string]any{"alg": "ed25519", "key_id": "key_sha256", "key_id_value": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "signature": "c2ln"}},
	}
}

func validRunCapabilityManifestPayload() map[string]any {
	return map[string]any{
		"schema_id":          capabilityManifestSchemaID,
		"schema_version":     capabilityManifestVersion,
		"principal":          map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1", "role_family": "workspace", "role_kind": "workspace-edit", "run_id": "run-1"},
		"manifest_scope":     "run",
		"run_id":             "run-1",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_run", "cap_stage"},
		"allowlist_refs":     []any{mustDigestObject(mustAllowlistHash(validAllowlistPayload("allowlist-b")))},
		"signatures":         []any{map[string]any{"alg": "ed25519", "key_id": "key_sha256", "key_id_value": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "signature": "c2ln"}},
	}
}

func validStageCapabilityManifestPayload() map[string]any {
	return map[string]any{
		"schema_id":          capabilityManifestSchemaID,
		"schema_version":     capabilityManifestVersion,
		"principal":          map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1", "role_family": "workspace", "role_kind": "workspace-edit", "run_id": "run-1", "stage_id": "stage-1"},
		"manifest_scope":     "stage",
		"run_id":             "run-1",
		"stage_id":           "stage-1",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_stage"},
		"allowlist_refs":     []any{mustDigestObject(mustAllowlistHash(validAllowlistPayload("allowlist-c")))},
		"signatures":         []any{map[string]any{"alg": "ed25519", "key_id": "key_sha256", "key_id_value": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "signature": "c2ln"}},
	}
}

func validAllowlistPayload(entry string) map[string]any {
	return map[string]any{
		"schema_id":       policyAllowlistSchemaID,
		"schema_version":  policyAllowlistSchemaVersion,
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": gatewayScopeRuleSchemaID,
		"entries": []any{map[string]any{
			"schema_id":                   gatewayScopeRuleSchemaID,
			"schema_version":              gatewayScopeRuleVersion,
			"scope_kind":                  "gateway_destination",
			"gateway_role_kind":           "model-gateway",
			"destination":                 validDestinationDescriptor(entry),
			"permitted_operations":        []any{"invoke_model"},
			"allowed_egress_data_classes": []any{"spec_text"},
			"redirect_posture":            "allowlist_only",
		}},
	}
}

func validDestinationDescriptor(name string) map[string]any {
	return map[string]any{
		"schema_id":                destinationDescriptorSchemaID,
		"schema_version":           destinationDescriptorVersion,
		"descriptor_kind":          "model_endpoint",
		"canonical_host":           name + ".example.com",
		"provider_or_namespace":    name,
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
}

func validRuleSetPayload() map[string]any {
	return map[string]any{
		"schema_id":      policyRuleSetSchemaID,
		"schema_version": policyRuleSetSchemaVersion,
		"rules": []any{
			map[string]any{"rule_id": "allow-1", "effect": "allow", "action_kind": "workspace_write", "capability_id": "cap_stage", "reason_code": "allow_manifest_opt_in", "details_schema_id": "runecode.protocol.details.policy.allow.v0"},
		},
	}
}

func mustAllowlistHash(value map[string]any) string {
	b, _ := json.Marshal(value)
	h, _ := canonicalHashBytes(b)
	return h
}

func mustDigestObject(identity string) map[string]any {
	return map[string]any{"hash_alg": "sha256", "hash": identity[len("sha256:"):]}
}

func validWorkspaceWriteActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindWorkspaceWrite,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadWorkspaceSchemaID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadWorkspaceSchemaID,
			"schema_version": "0.1.0",
			"target_path":    "src/main.go",
			"write_mode":     "update",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validExecutorRunActionRequest(capabilityID string, executorClass string, argv []string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindExecutorRun,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadExecutorSchemaID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadExecutorSchemaID,
			"schema_version": "0.1.0",
			"executor_class": executorClass,
			"executor_id":    "workspace-runner",
			"argv":           toAnySlice(argv),
			"network_access": "none",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validGatewayEgressActionRequest(capabilityID string, roleFamily string, roleKind string, gatewayRoleKind string, destinationKind string, actionKind string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            actionKind,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadGatewaySchemaID,
		ActionPayload: map[string]any{
			"schema_id":         actionPayloadGatewaySchemaID,
			"schema_version":    "0.1.0",
			"gateway_role_kind": gatewayRoleKind,
			"destination_kind":  destinationKind,
			"destination_ref":   "provider.example.com",
			"egress_data_class": "spec_text",
			"operation":         "invoke_model",
		},
		ActorKind:  "daemon",
		RoleFamily: roleFamily,
		RoleKind:   roleKind,
	}
}

func validDependencyFetchActionRequest(capabilityID string, roleKind string, refName string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindDependencyFetch,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadGatewaySchemaID,
		ActionPayload: map[string]any{
			"schema_id":         actionPayloadGatewaySchemaID,
			"schema_version":    "0.1.0",
			"gateway_role_kind": "dependency-fetch",
			"destination_kind":  "package_registry",
			"destination_ref":   refName + ".example.com",
			"egress_data_class": "spec_text",
			"operation":         "fetch_dependency",
		},
		ActorKind:  "daemon",
		RoleFamily: "gateway",
		RoleKind:   roleKind,
	}
}

func validArtifactReadActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindArtifactRead,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadArtifactSchemaID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadArtifactSchemaID,
			"schema_version": "0.1.0",
			"artifact_hash":  mustDigestObject("sha256:" + strings.Repeat("3", 64)),
			"read_mode":      "head",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validPromotionActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindPromotion,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadPromotionSchemaID,
		ActionPayload: map[string]any{
			"schema_id":            actionPayloadPromotionSchemaID,
			"schema_version":       "0.1.0",
			"promotion_kind":       "excerpt",
			"source_artifact_hash": mustDigestObject("sha256:" + strings.Repeat("4", 64)),
			"target_data_class":    "approved_file_excerpts",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validBackendPostureActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindBackendPosture,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadBackendSchemaID,
		ActionPayload: map[string]any{
			"schema_id":         actionPayloadBackendSchemaID,
			"schema_version":    "0.1.0",
			"backend_class":     "microvm",
			"change_kind":       "select_backend",
			"requested_posture": "microvm_default",
			"requires_opt_in":   false,
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validGateOverrideActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindGateOverride,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadGateSchemaID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadGateSchemaID,
			"schema_version": "0.1.0",
			"gate_name":      "policy-engine",
			"override_mode":  "break_glass",
			"justification":  "Emergency trust maintenance",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validStageSummarySignOffActionRequest(capabilityID, summaryHash string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindStageSummarySign,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadStageSchemaID,
		ActionPayload: map[string]any{
			"schema_id":          actionPayloadStageSchemaID,
			"schema_version":     "0.1.0",
			"run_id":             "run-1",
			"stage_id":           "stage-1",
			"stage_summary_hash": mustDigestObject(summaryHash),
			"approval_profile":   "moderate",
			"summary_revision":   float64(1),
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validSecretAccessActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindSecretAccess,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadSecretAccessID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadSecretAccessID,
			"schema_version": "0.1.0",
			"secret_ref":     "secrets/prod/db-password",
			"access_mode":    "lease_issue",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func compileGatewayInputWithOneCapability(roleKind string, capability string, allowlist map[string]any) CompileInput {
	role := validRoleManifestPayload()
	role["role_family"] = "gateway"
	role["role_kind"] = roleKind
	role["capability_opt_ins"] = []any{capability}
	rolePrincipal := role["principal"].(map[string]any)
	rolePrincipal["role_family"] = "gateway"
	rolePrincipal["role_kind"] = roleKind
	role["allowlist_refs"] = []any{mustDigestObject(mustAllowlistHash(allowlist))}

	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{capability}
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_family"] = "gateway"
	runPrincipal["role_kind"] = roleKind
	run["allowlist_refs"] = []any{mustDigestObject(mustAllowlistHash(allowlist))}

	return CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    mustManifestInput(role),
		RunManifest:     mustManifestInput(run),
		Allowlists:      []ManifestInput{mustManifestInput(allowlist)},
	}
}

func validAllowlistPayloadForGateway(entry string, gatewayRole string, descriptorKind string, operation string, dataClass string) map[string]any {
	return map[string]any{
		"schema_id":       policyAllowlistSchemaID,
		"schema_version":  policyAllowlistSchemaVersion,
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": gatewayScopeRuleSchemaID,
		"entries": []any{map[string]any{
			"schema_id":                   gatewayScopeRuleSchemaID,
			"schema_version":              gatewayScopeRuleVersion,
			"scope_kind":                  "gateway_destination",
			"gateway_role_kind":           gatewayRole,
			"destination":                 validDestinationDescriptorForKind(entry, descriptorKind),
			"permitted_operations":        []any{operation},
			"allowed_egress_data_classes": []any{dataClass},
			"redirect_posture":            "allowlist_only",
		}},
	}
}

func validDestinationDescriptorForKind(name, kind string) map[string]any {
	return map[string]any{
		"schema_id":                destinationDescriptorSchemaID,
		"schema_version":           destinationDescriptorVersion,
		"descriptor_kind":          kind,
		"canonical_host":           name + ".example.com",
		"provider_or_namespace":    name,
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
}

func toAnySlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return out
}

func ptr[T any](v T) *T { return &v }
