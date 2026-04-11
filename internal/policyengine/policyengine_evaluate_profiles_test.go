package policyengine

import (
	"errors"
	"strings"
	"testing"
)

func TestEvaluateGatewayRequiresSignedAllowlistDestinationMatch(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("allowlisted destination should allow, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateGatewayAllowsCaseInsensitiveDestinationHostMatch(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "ALLOWLIST-MODEL.EXAMPLE.COM"
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("case-insensitive host should allow, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateGatewayDeniesWhenCanonicalPortRequiredButMissing(t *testing.T) {
	allowlist := validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")
	entry := allowlist["entries"].([]any)[0].(map[string]any)
	destination := entry["destination"].(map[string]any)
	destination["canonical_port"] = float64(8443)

	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", allowlist))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateGatewayFailsClosedWhenOperationMissing(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	delete(action.ActionPayload, "operation")
	_, err := Evaluate(compiled, action)
	if err == nil {
		t.Fatal("Evaluate error = nil, want fail-closed schema validation error")
	}
	var evalErr *EvaluationError
	if !errors.As(err, &evalErr) {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerValidationSchema)
	}
}

func TestEvaluateGatewayDeniesEscapingPathPrefix(t *testing.T) {
	allowlist := validAllowlistPayloadForGateway("allowlist-model", "model-gateway", "model_endpoint", "invoke_model", "spec_text")
	entry := allowlist["entries"].([]any)[0].(map[string]any)
	destination := entry["destination"].(map[string]any)
	destination["canonical_path_prefix"] = "/v1/allowed/"

	compiled := mustCompile(t, compileGatewayInputWithOneCapability("model-gateway", "cap_gateway", allowlist))
	action := validGatewayEgressActionRequest("cap_gateway", "gateway", "model-gateway", "model-gateway", "model_endpoint", ActionKindGatewayEgress)
	action.ActionPayload["destination_ref"] = "allowlist-model.example.com/v1/allowed/../escape"
	decision, err := Evaluate(compiled, action)
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
	if err != nil || decision.DecisionOutcome != DecisionAllow {
		t.Fatalf("workspace edit should allow, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateModerateProfileRequiresApprovalForWindowsAbsoluteWorkspaceWrite(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validWorkspaceWriteActionRequest("cap_stage")
	action.ActionPayload["target_path"] = `C:\Windows\System32\drivers\etc\hosts`
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("absolute write should require approval, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateDeniesWrappedShellForWorkspaceOrdinaryExecutor(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"env", "FOO=bar", "sh", "-c", "whoami"})
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("wrapped shell should deny, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateDeniesShellPassthroughFlagsForWorkspaceOrdinaryExecutor(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"python", "-c", "print('hi')"})
	action.ActionPayload["executor_id"] = "python"
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("shell passthrough flags should deny, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateDeniesPowerShellCommandAliasForWorkspaceOrdinaryExecutor(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"pwsh", "-c", "Write-Output hi"})
	action.ActionPayload["executor_id"] = "workspace-runner"
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("powershell -c passthrough should deny, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateDeniesSudoLauncherForWorkspaceOrdinaryExecutor(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"sudo", "python", "script.py"})
	action.ActionPayload["executor_id"] = "python"
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("sudo launcher should deny, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateDeniesSystemModifyingWrapperChainsForWorkspaceOrdinaryExecutor(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	fixtures := []struct {
		name string
		argv []string
	}{
		{name: "env_command_nohup_chain", argv: []string{"env", "CI=1", "command", "nohup", "apt-get", "install", "jq"}},
		{name: "scheduler_priority_wrappers", argv: []string{"timeout", "30", "nice", "-n", "10", "docker", "run", "alpine", "true"}},
		{name: "single_token_passthrough", argv: []string{"workspace-runner", "exec", "--", "apt-get install jq"}},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", fixture.argv)
			decision, err := Evaluate(compiled, action)
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			if decision.DecisionOutcome != DecisionDeny {
				t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
			}
		})
	}
}

func TestExactActionApprovalBindingChangesWithActionHash(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	actionA := validGateOverrideActionRequest("cap_stage")
	actionB := validGateOverrideActionRequest("cap_stage")
	actionB.ActionPayload["justification"] = "Emergency trust maintenance - revised"
	decisionA, _ := Evaluate(compiled, actionA)
	decisionB, _ := Evaluate(compiled, actionB)
	if decisionA.ActionRequestHash == decisionB.ActionRequestHash {
		t.Fatalf("action_request_hash should differ when payload changes")
	}
}

func TestEvaluateDependencyFetchUsesDependencyTriggerCode(t *testing.T) {
	compiled := mustCompile(t, compileGatewayInputWithOneCapability("dependency-fetch", "cap_dep", validAllowlistPayloadForGateway("allowlist-dep", "dependency-fetch", "package_registry", "enable_dependency_fetch", "spec_text")))
	action := validDependencyFetchActionRequest("cap_dep", "dependency-fetch", "allowlist-dep")
	action.ActionPayload["operation"] = "enable_dependency_fetch"
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("dependency fetch should require approval, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestEvaluateSecretAccessRequiresModerateApproval(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validSecretAccessActionRequest("cap_stage")
	decision, err := Evaluate(compiled, action)
	if err != nil || decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("secret access should require approval, err=%v outcome=%q", err, decision.DecisionOutcome)
	}
}

func TestStageSignOffStaleBindingChangesWithSummaryHash(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	actionA := validStageSummarySignOffActionRequest("cap_stage", "sha256:"+strings.Repeat("7", 64))
	actionB := validStageSummarySignOffActionRequest("cap_stage", "sha256:"+strings.Repeat("8", 64))
	decisionA, _ := Evaluate(compiled, actionA)
	decisionB, _ := Evaluate(compiled, actionB)
	if decisionA.ActionRequestHash == decisionB.ActionRequestHash {
		t.Fatalf("action_request_hash should differ when stage summary hash changes")
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
}

func TestEvaluateWorkspaceExecutorContractsFailClosedOnUnknownExecutorID(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"go", "test", "./..."})
	action.ActionPayload["executor_id"] = "go"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateWorkspaceExecutorContractsRejectExecutorClassMismatch(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "system_modifying", []string{"python", "script.py"})
	action.ActionPayload["executor_id"] = "python"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateWorkspaceExecutorContractsRejectRoleNotAllowedForExecutorID(t *testing.T) {
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
	action := validExecutorRunActionRequest("cap_artifact_read", "workspace_ordinary", []string{"python", "script.py"})
	action.RoleKind = "workspace-read"
	action.ActionPayload["executor_id"] = "python"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateWorkspaceExecutorContractsRejectNetworkAccessOutsideContract(t *testing.T) {
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
}

func TestEvaluateWorkspaceExecutorContractsRejectArgvShapeMismatch(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"go", "test", "./..."})
	action.ActionPayload["executor_id"] = "python"
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateWorkspaceRunnerAllowsEnvFlagValueWithoutTreatingAsAssignment(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"env", "--chdir=.", "workspace-runner", "go", "test", "./..."})
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateWorkspaceExecutorContractsRejectUnknownOperationHead(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"workspace-runner", "exec", "python", "script.py"})
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}

func TestEvaluateWorkspaceExecutorContractsRejectEnvironmentOutsideContract(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", []string{"python", "script.py"})
	action.ActionPayload["executor_id"] = "python"
	action.ActionPayload["environment"] = map[string]any{"LD_PRELOAD": "/tmp/x"}
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionDeny {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionDeny)
	}
}
