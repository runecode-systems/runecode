package policyengine

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestEvaluateUsesPrecedenceDenyThenApprovalThenAllow(t *testing.T) { /* moved unchanged */
	rules := validRuleSetPayload()
	rules["rules"] = []any{
		map[string]any{"rule_id": "allow-1", "effect": "allow", "action_kind": "workspace_write", "capability_id": "cap_stage", "reason_code": "allow_manifest_opt_in", "details_schema_id": "runecode.protocol.details.policy.allow.v0"},
		map[string]any{"rule_id": "approval-1", "effect": "require_human_approval", "action_kind": "workspace_write", "capability_id": "cap_stage", "reason_code": "approval_required", "details_schema_id": "runecode.protocol.details.policy.approval.v0"},
		map[string]any{"rule_id": "deny-1", "effect": "deny", "action_kind": "workspace_write", "capability_id": "cap_stage", "reason_code": "deny_by_default", "details_schema_id": "runecode.protocol.details.policy.deny.v0"},
	}
	compiled := mustCompile(t, CompileInput{FixedInvariants: FixedInvariants{}, RoleManifest: testManifestInput(t, validRoleManifestPayload(), ""), RunManifest: testManifestInput(t, validRunCapabilityManifestPayload(), ""), StageManifest: ptr(testManifestInput(t, validStageCapabilityManifestPayload(), "")), Allowlists: []ManifestInput{testManifestInput(t, validAllowlistPayload("allowlist-a"), ""), testManifestInput(t, validAllowlistPayload("allowlist-b"), ""), testManifestInput(t, validAllowlistPayload("allowlist-c"), "")}, RuleSet: ptr(testManifestInput(t, rules, ""))})
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
	if !ok || evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error=%v, want validation schema error", err)
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
	if !ok || evalErr.Code != ErrCodeBrokerValidationSchema {
		t.Fatalf("error=%v, want validation schema error", err)
	}
}

func TestNewStageSummarySignOffActionFailsWithoutPlanID(t *testing.T) {
	manifestHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	stageSummaryHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)}

	_, err := NewStageSummarySignOffAction(StageSummarySignOffActionInput{
		ActionEnvelope:   ActionEnvelope{CapabilityID: "cap_stage", Actor: ActionActor{ActorKind: "daemon", RoleFamily: "workspace", RoleKind: "workspace-edit"}},
		RunID:            "run-1",
		StageID:          "stage-1",
		ManifestHash:     manifestHash,
		StageSummaryHash: stageSummaryHash,
		ApprovalProfile:  "moderate",
	})
	if err == nil {
		t.Fatal("NewStageSummarySignOffAction returned nil error without plan_id")
	}
}
