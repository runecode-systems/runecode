package policyengine

import "testing"

func TestEvaluateGateOverrideMatchesMultipleHardFloorClasses(t *testing.T) {
	compiled := mustCompile(t, compileInputWithOneCapability("cap_stage"))
	action := validGateOverrideActionRequest("cap_stage")
	decision, err := Evaluate(compiled, action)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if decision.DecisionOutcome != DecisionRequireHumanApproval {
		t.Fatalf("DecisionOutcome = %q, want %q", decision.DecisionOutcome, DecisionRequireHumanApproval)
	}
	classes, ok := decision.Details["hard_floor_operation_classes"].([]string)
	if !ok || len(classes) != 2 {
		t.Fatalf("hard_floor_operation_classes = %v", decision.Details["hard_floor_operation_classes"])
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

func TestClassifyHardFloorOperationDetectsSystemModifyingThroughWrapperChains(t *testing.T) {
	fixtures := []struct {
		name string
		argv []string
	}{
		{name: "env_command_nohup_chain", argv: []string{"env", "CI=1", "command", "nohup", "apt-get", "install", "jq"}},
		{name: "timeout_nice_chain", argv: []string{"timeout", "30", "nice", "-n", "10", "docker", "run", "alpine", "true"}},
		{name: "single_token_embedded_command", argv: []string{"workspace-runner", "exec", "--", "apt-get install jq"}},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			action := validExecutorRunActionRequest("cap_stage", "workspace_ordinary", fixture.argv)
			classes, floor := classifyHardFloorOperation(action, &executorRunPayload{ExecutorClass: "workspace_ordinary", Argv: fixture.argv})
			if !containsHardFloorClass(classes, HardFloorSecurityPostureWeakening) {
				t.Fatalf("classes = %v, want security_posture_weakening", classes)
			}
			if floor != ApprovalAssuranceReauthenticated {
				t.Fatalf("floor = %q, want %q", floor, ApprovalAssuranceReauthenticated)
			}
		})
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
