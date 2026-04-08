package policyengine

import (
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
