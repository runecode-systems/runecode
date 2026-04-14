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
	if got, _ := decision.Details["invariant"].(string); got != "no_escalation_in_place" {
		t.Fatalf("invariant = %v, want no_escalation_in_place", decision.Details["invariant"])
	}
}

func TestDenyIfInvalidGatewayFamilyWorkspaceContextIncludesArtifactRouteGuidance(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validGatewayEgressActionRequest("cap_stage", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	actionHash, err := canonicalHashValue(action)
	if err != nil {
		t.Fatalf("canonicalHashValue returned error: %v", err)
	}
	decision, blocked := denyIfInvalidGatewayFamily(compiled, action, actionHash)
	if !blocked {
		t.Fatal("denyIfInvalidGatewayFamily blocked = false, want true")
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["required_cross_boundary_route"].(string); got != "artifact_io" {
		t.Fatalf("required_cross_boundary_route = %v, want artifact_io", decision.Details["required_cross_boundary_route"])
	}
	routeActions, ok := decision.Details["artifact_route_actions"].([]string)
	if !ok || len(routeActions) != 1 || routeActions[0] != ActionKindArtifactRead {
		t.Fatalf("artifact_route_actions = %#v, want [artifact_read]", decision.Details["artifact_route_actions"])
	}
}

func TestEvaluateExecutorRunNetworkDenyForWorkspaceRoleIncludesArtifactRouteGuidance(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"python", "script.py"})
	action.ActionPayload["executor_id"] = "python"
	action.ActionPayload["network_access"] = "gateway_only"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["invariant"].(string); got == "" {
		t.Fatalf("invariant missing in decision details: %#v", decision.Details)
	}
	if got, _ := decision.Details["required_cross_boundary_route"].(string); got != "artifact_io" {
		t.Fatalf("required_cross_boundary_route = %v, want artifact_io", decision.Details["required_cross_boundary_route"])
	}
	routeActions, ok := decision.Details["artifact_route_actions"].([]string)
	if !ok || len(routeActions) != 1 || routeActions[0] != ActionKindArtifactRead {
		t.Fatalf("artifact_route_actions = %#v, want [artifact_read]", decision.Details["artifact_route_actions"])
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

func TestEvaluateDeniesSecretAccessForGatewayRole(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validSecretAccessActionRequest("cap_gateway")
	action.RoleFamily = "gateway"
	action.RoleKind = "model-gateway"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["invariant"].(string); got != "gateway_no_long_lived_secret_storage" {
		t.Fatalf("invariant = %v, want gateway_no_long_lived_secret_storage", decision.Details["invariant"])
	}
}

func TestEvaluateDeniesArtifactReadForGatewayRole(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validArtifactReadActionRequest("cap_gateway")
	action.RoleFamily = "gateway"
	action.RoleKind = "model-gateway"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["invariant"].(string); got != "gateway_no_workspace_access" {
		t.Fatalf("invariant = %v, want gateway_no_workspace_access", decision.Details["invariant"])
	}
}

func TestEvaluateDeniesModelGatewayAuthProviderDestination(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-auth", "model-gateway", "auth_provider", "exchange_auth_code", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "auth_provider", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-auth.example.com"
	action.ActionPayload["operation"] = "exchange_auth_code"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["invariant"].(string); got != "gateway_role_separation" {
		t.Fatalf("invariant = %v, want gateway_role_separation", decision.Details["invariant"])
	}
}

func TestEvaluateDeniesModelGatewayAuthRefreshOperation(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "refresh_auth_token", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	action.ActionPayload["operation"] = "refresh_auth_token"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := decision.Details["invariant"].(string); got != "gateway_role_separation" {
		t.Fatalf("invariant = %v, want gateway_role_separation", decision.Details["invariant"])
	}
}

func TestEvaluateExecutorSystemModifyingDeniedForWorkspaceRoleMatrix(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "system_modifying", []string{"apt-get", "install", "jq"})
	action.ActionPayload["executor_id"] = "workspace-runner"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
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
	action.ActionPayload["target_backend_kind"] = "container"
	action.ActionPayload["assurance_change_kind"] = "reduce_assurance"
	action.ActionPayload["opt_in_kind"] = "none"
	action.ActionPayload["reduced_assurance_acknowledged"] = true
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
	action.ActionPayload["target_backend_kind"] = "container"
	action.ActionPayload["selection_mode"] = "automatic_fallback_attempt"
	action.ActionPayload["assurance_change_kind"] = "reduce_assurance"
	action.ActionPayload["opt_in_kind"] = "exact_action_approval"
	action.ActionPayload["reduced_assurance_acknowledged"] = true
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.PolicyReasonCode != "deny_container_automatic_fallback" {
		t.Fatalf("policy_reason_code = %q", decision.PolicyReasonCode)
	}
}

func TestEvaluateBackendSelectionRulesContainerRequiresReducedAssuranceAcknowledgment(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validBackendPostureActionRequest("cap_stage")
	action.ActionPayload["target_backend_kind"] = "container"
	action.ActionPayload["selection_mode"] = "explicit_selection"
	action.ActionPayload["assurance_change_kind"] = "reduce_assurance"
	action.ActionPayload["opt_in_kind"] = "exact_action_approval"
	action.ActionPayload["reduced_assurance_acknowledged"] = false
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("decision_outcome = %q, want deny", decision.DecisionOutcome)
	}
	if decision.PolicyReasonCode != "deny_container_acknowledgment_required" {
		t.Fatalf("policy_reason_code = %q, want deny_container_acknowledgment_required", decision.PolicyReasonCode)
	}
}

func TestEvaluateBackendSelectionRulesContainerExplicitSelectionBlockedByDefaultWithExactActionApproval(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := explicitContainerSelectionAction("cap_stage")
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	assertContainerSelectionRequiresApproval(t, decision)
	assertBackendSelectionApprovalScope(t, decision)
	assertBackendSelectionApprovalHashes(t, decision)
}

func explicitContainerSelectionAction(capabilityToken string) ActionRequest {
	action := validBackendPostureActionRequest(capabilityToken)
	action.ActionPayload["target_backend_kind"] = "container"
	action.ActionPayload["selection_mode"] = "explicit_selection"
	action.ActionPayload["change_kind"] = "select_backend"
	action.ActionPayload["assurance_change_kind"] = "reduce_assurance"
	action.ActionPayload["opt_in_kind"] = "exact_action_approval"
	action.ActionPayload["reduced_assurance_acknowledged"] = true
	return action
}

func assertContainerSelectionRequiresApproval(t *testing.T, decision PolicyDecision) {
	t.Helper()
	if decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("decision_outcome = %q, want %q", decision.DecisionOutcome, DecisionRequireHumanApproval)
	}
	if decision.PolicyReasonCode != "approval_required" {
		t.Fatalf("policy_reason_code = %q, want approval_required", decision.PolicyReasonCode)
	}
	if got, _ := decision.Details["invariant"].(string); got != "backend_selection_rules" {
		t.Fatalf("invariant = %v, want backend_selection_rules", decision.Details["invariant"])
	}
	if got, _ := decision.Details["target_backend_kind"].(string); got != "container" {
		t.Fatalf("target_backend_kind = %v, want container", decision.Details["target_backend_kind"])
	}
	if got, _ := decision.Details["opt_in_kind"].(string); got != "exact_action_approval" {
		t.Fatalf("opt_in_kind = %v, want exact_action_approval", decision.Details["opt_in_kind"])
	}
	if decision.RequiredApprovalSchemaID != requiredApprovalModerateBackendSchemaID {
		t.Fatalf("required_approval_schema_id = %q, want %q", decision.RequiredApprovalSchemaID, requiredApprovalModerateBackendSchemaID)
	}
}

func assertBackendSelectionApprovalScope(t *testing.T, decision PolicyDecision) {
	t.Helper()
	scope, ok := decision.RequiredApproval["scope"].(map[string]any)
	if !ok {
		t.Fatalf("required_approval.scope = %T, want map", decision.RequiredApproval["scope"])
	}
	if scope["action_kind"] != ActionKindBackendPosture {
		t.Fatalf("required_approval.scope.action_kind = %v, want %q", scope["action_kind"], ActionKindBackendPosture)
	}
}

func assertBackendSelectionApprovalHashes(t *testing.T, decision PolicyDecision) {
	t.Helper()
	related, ok := decision.RequiredApproval["related_hashes"].(map[string]any)
	if !ok {
		t.Fatalf("required_approval.related_hashes = %T, want map", decision.RequiredApproval["related_hashes"])
	}
	if related["action_request_hash"] != decision.ActionRequestHash {
		t.Fatalf("required_approval.related_hashes.action_request_hash = %v, want %q", related["action_request_hash"], decision.ActionRequestHash)
	}
}

func TestEvaluateWorkspaceOfflineInvariantStillAppliesAfterContainerBackendOptInRequest(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	backendAction := validBackendPostureActionRequest("cap_stage")
	backendAction.ActionPayload["target_backend_kind"] = "container"
	backendAction.ActionPayload["selection_mode"] = "explicit_selection"
	backendAction.ActionPayload["change_kind"] = "select_backend"
	backendAction.ActionPayload["assurance_change_kind"] = "reduce_assurance"
	backendAction.ActionPayload["opt_in_kind"] = "exact_action_approval"
	backendAction.ActionPayload["reduced_assurance_acknowledged"] = true

	backendDecision, err := Evaluate(compiled, backendAction)
	if err != nil {
		t.Fatalf("Evaluate(backend posture) returned error: %v", err)
	}
	if backendDecision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("backend decision_outcome = %q, want %q", backendDecision.DecisionOutcome, DecisionRequireHumanApproval)
	}

	execAction := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"python", "script.py"})
	execAction.ActionPayload["executor_id"] = "python"
	execAction.ActionPayload["network_access"] = "gateway_only"
	execDecision, err := Evaluate(compiled, execAction)
	if err != nil {
		t.Fatalf("Evaluate(executor run) returned error: %v", err)
	}
	if execDecision.DecisionOutcome != DecisionDeny {
		t.Fatalf("executor decision_outcome = %q, want %q", execDecision.DecisionOutcome, DecisionDeny)
	}
	if got, _ := execDecision.Details["required_cross_boundary_route"].(string); got != "artifact_io" {
		t.Fatalf("required_cross_boundary_route = %v, want artifact_io", execDecision.Details["required_cross_boundary_route"])
	}
}

func TestEvaluateWorkspaceRoleActionMatrixExplicitlyRejectsExecutorRunForWorkspaceRead(t *testing.T) {
	role := validRoleManifestPayload()
	role["role_kind"] = "workspace-read"
	role["capability_opt_ins"] = []any{"cap_artifact_read"}
	rolePrincipal := role["principal"].(map[string]any)
	rolePrincipal["role_kind"] = "workspace-read"

	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{"cap_artifact_read"}
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_kind"] = "workspace-read"

	stage := validStageCapabilityManifestPayload()
	stage["capability_opt_ins"] = []any{"cap_artifact_read"}
	stagePrincipal := stage["principal"].(map[string]any)
	stagePrincipal["role_kind"] = "workspace-read"

	compiled := mustCompile(t, CompileInput{
		RoleManifest:  testManifestInput(t, role, ""),
		RunManifest:   testManifestInput(t, run, ""),
		StageManifest: ptr(testManifestInput(t, stage, "")),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-c"), ""),
		},
	})
	action := validExecutorRunActionRequest("cap_artifact_read", "workspace_ordinary", []string{"go", "test", "./..."})
	action.RoleKind = "workspace-read"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateWorkspaceWriteBoundaryRejectsWorkspaceTestWriteOutsideBuildOutput(t *testing.T) {
	role := validRoleManifestPayload()
	role["role_kind"] = "workspace-test"
	role["capability_opt_ins"] = []any{"cap_stage"}
	rolePrincipal := role["principal"].(map[string]any)
	rolePrincipal["role_kind"] = "workspace-test"

	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{"cap_stage"}
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_kind"] = "workspace-test"

	stage := validStageCapabilityManifestPayload()
	stage["capability_opt_ins"] = []any{"cap_stage"}
	stagePrincipal := stage["principal"].(map[string]any)
	stagePrincipal["role_kind"] = "workspace-test"

	compiled := mustCompile(t, CompileInput{
		RoleManifest:  testManifestInput(t, role, ""),
		RunManifest:   testManifestInput(t, run, ""),
		StageManifest: ptr(testManifestInput(t, stage, "")),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-c"), ""),
		},
	})
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.RoleKind = "workspace-test"
	action.ActionPayload["target_path"] = "src/main.go"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateWorkspaceWriteBoundaryAllowsWorkspaceTestWriteInsideBuildOutput(t *testing.T) {
	role := validRoleManifestPayload()
	role["role_kind"] = "workspace-test"
	role["capability_opt_ins"] = []any{"cap_stage"}
	rolePrincipal := role["principal"].(map[string]any)
	rolePrincipal["role_kind"] = "workspace-test"

	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{"cap_stage"}
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_kind"] = "workspace-test"

	stage := validStageCapabilityManifestPayload()
	stage["capability_opt_ins"] = []any{"cap_stage"}
	stagePrincipal := stage["principal"].(map[string]any)
	stagePrincipal["role_kind"] = "workspace-test"

	compiled := mustCompile(t, CompileInput{
		RoleManifest:  testManifestInput(t, role, ""),
		RunManifest:   testManifestInput(t, run, ""),
		StageManifest: ptr(testManifestInput(t, stage, "")),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-c"), ""),
		},
	})
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.RoleKind = "workspace-test"
	action.ActionPayload["target_path"] = "build-output/test.log"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionAllow)
	}
}
