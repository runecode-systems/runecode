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

func containsHardFloorClass(classes []HardFloorOperationClass, target HardFloorOperationClass) bool {
	for _, class := range classes {
		if class == target {
			return true
		}
	}
	return false
}
