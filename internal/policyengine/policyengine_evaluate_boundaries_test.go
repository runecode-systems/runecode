package policyengine

import "testing"

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
	compiled := mustCompile(t, CompileInput{FixedInvariants: FixedInvariants{}, RoleManifest: testManifestInput(t, role, ""), RunManifest: testManifestInput(t, run, ""), Allowlists: []ManifestInput{testManifestInput(t, validAllowlistPayload("allowlist-a"), ""), testManifestInput(t, validAllowlistPayload("allowlist-b"), "")}})
	action := validGatewayEgressActionRequest("cap_run", "gateway", "dependency-fetch", "dependency-fetch", "model_endpoint", ActionKindDependencyFetch)
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
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
}

func TestEvaluateBackendSelectionRulesMicroVMDefaultAllow(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validBackendPostureActionRequest("cap_stage")
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionAllow)
	}
}

func TestEvaluateBackendSelectionRulesContainerRequiresExplicitOptIn(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validBackendPostureActionRequest("cap_stage")
	action.ActionPayload["backend_class"] = "container"
	action.ActionPayload["requested_posture"] = "container_opt_in"
	action.ActionPayload["requires_opt_in"] = false
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.PolicyReasonCode != "deny_container_opt_in_required" {
		t.Fatalf("policy_reason_code = %q", decision.PolicyReasonCode)
	}
}

func TestEvaluateBackendSelectionRulesNoAutomaticFallback(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validBackendPostureActionRequest("cap_stage")
	action.ActionPayload["backend_class"] = "container"
	action.ActionPayload["requested_posture"] = "container_fallback"
	action.ActionPayload["requires_opt_in"] = true
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.PolicyReasonCode != "deny_container_automatic_fallback" {
		t.Fatalf("policy_reason_code = %q", decision.PolicyReasonCode)
	}
}
