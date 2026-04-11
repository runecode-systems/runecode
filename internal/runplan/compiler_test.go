package runplan

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestCompileBuildsDeterministicRunPlan(t *testing.T) {
	workflowPayload := workflowPayloadForTest(t, []any{gateDef("build_gate", "step_validation_started", 0)}, []string{"workspace-edit", "workspace-test"})
	processPayload := processPayloadForTest(t, []any{gateDef("build_gate", "step_validation_started", 0), gateDef("lint_gate", "step_validation_finished", 1)}, []string{"workspace-edit", "workspace-test"})

	plan, err := Compile(CompileInput{
		RunID:                   "run_123",
		PlanID:                  "plan_run_123_0001",
		SupersedesPlanID:        "plan_run_123_0000",
		CompiledAt:              time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		WorkflowDefinitionBytes: workflowPayload,
		ProcessDefinitionBytes:  processPayload,
		PolicyContextHash:       "sha256:" + strings.Repeat("a", 64),
		ExecutorRegistry:        policyengine.BuildExecutorRegistryProjection(),
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if plan.SchemaID != runPlanSchemaID {
		t.Fatalf("schema_id = %q, want %q", plan.SchemaID, runPlanSchemaID)
	}
	if plan.PlanID != "plan_run_123_0001" {
		t.Fatalf("plan_id = %q", plan.PlanID)
	}
	if plan.SupersedesPlanID != "plan_run_123_0000" {
		t.Fatalf("supersedes_plan_id = %q", plan.SupersedesPlanID)
	}
	if len(plan.ExecutorBindings) != 1 {
		t.Fatalf("executor_bindings len = %d, want 1", len(plan.ExecutorBindings))
	}
	if len(plan.GateDefinitions) != 2 {
		t.Fatalf("gate_definitions len = %d, want 2", len(plan.GateDefinitions))
	}
	if got := plan.GateDefinitions[0].Gate.GateID; got != "lint_gate" {
		t.Fatalf("first gate_id = %q, want lint_gate", got)
	}
	if got := plan.GateDefinitions[1].Gate.GateID; got != "build_gate" {
		t.Fatalf("second gate_id = %q, want build_gate", got)
	}
	if err := ValidateRunPlan(plan); err != nil {
		t.Fatalf("ValidateRunPlan returned error: %v", err)
	}
}

func TestCompileFailsClosedOnUnknownExecutorBinding(t *testing.T) {
	workflowPayload := mustJSON(t, map[string]any{
		"schema_id":      workflowDefinitionSchemaID,
		"schema_version": workflowDefinitionVersion,
		"workflow_id":    "workflow_main",
		"executor_bindings": []any{
			map[string]any{
				"binding_id":         "binding_unknown",
				"executor_id":        "unknown-executor",
				"executor_class":     "workspace_ordinary",
				"allowed_role_kinds": []any{"workspace-edit"},
			},
		},
		"gate_definitions": []any{gateDef("build_gate", "step_validation_started", 0)},
	})
	processPayload := mustJSON(t, map[string]any{
		"schema_id":      processDefinitionSchemaID,
		"schema_version": processDefinitionVersion,
		"process_id":     "process_default",
		"executor_bindings": []any{
			map[string]any{
				"binding_id":         "binding_unknown",
				"executor_id":        "unknown-executor",
				"executor_class":     "workspace_ordinary",
				"allowed_role_kinds": []any{"workspace-edit"},
			},
		},
		"gate_definitions": []any{gateDef("build_gate", "step_validation_started", 0)},
	})

	_, err := Compile(CompileInput{
		RunID:                   "run_123",
		PlanID:                  "plan_run_123_0001",
		WorkflowDefinitionBytes: workflowPayload,
		ProcessDefinitionBytes:  processPayload,
		PolicyContextHash:       "sha256:" + strings.Repeat("a", 64),
		ExecutorRegistry:        policyengine.BuildExecutorRegistryProjection(),
	})
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
}

func workflowPayloadForTest(t *testing.T, gates []any, roles []string) []byte {
	t.Helper()
	return mustJSON(t, map[string]any{
		"schema_id":         workflowDefinitionSchemaID,
		"schema_version":    workflowDefinitionVersion,
		"workflow_id":       "workflow_main",
		"executor_bindings": []any{executorBindingFixture("binding_workspace_runner", "workspace-runner", roles)},
		"gate_definitions":  gates,
	})
}

func processPayloadForTest(t *testing.T, gates []any, roles []string) []byte {
	t.Helper()
	return mustJSON(t, map[string]any{
		"schema_id":         processDefinitionSchemaID,
		"schema_version":    processDefinitionVersion,
		"process_id":        "process_default",
		"executor_bindings": []any{executorBindingFixture("binding_workspace_runner", "workspace-runner", roles)},
		"gate_definitions":  gates,
	})
}

func executorBindingFixture(bindingID, executorID string, roles []string) map[string]any {
	roleItems := make([]any, 0, len(roles))
	for _, role := range roles {
		roleItems = append(roleItems, role)
	}
	return map[string]any{
		"binding_id":         bindingID,
		"executor_id":        executorID,
		"executor_class":     "workspace_ordinary",
		"allowed_role_kinds": roleItems,
	}
}

func TestCompileRejectsSupersedesSameAsPlanID(t *testing.T) {
	workflowPayload := mustJSON(t, map[string]any{
		"schema_id":      workflowDefinitionSchemaID,
		"schema_version": workflowDefinitionVersion,
		"workflow_id":    "workflow_main",
		"executor_bindings": []any{
			map[string]any{
				"binding_id":         "binding_workspace_runner",
				"executor_id":        "workspace-runner",
				"executor_class":     "workspace_ordinary",
				"allowed_role_kinds": []any{"workspace-edit"},
			},
		},
		"gate_definitions": []any{gateDef("build_gate", "step_validation_started", 0)},
	})
	processPayload := mustJSON(t, map[string]any{
		"schema_id":      processDefinitionSchemaID,
		"schema_version": processDefinitionVersion,
		"process_id":     "process_default",
		"executor_bindings": []any{
			map[string]any{
				"binding_id":         "binding_workspace_runner",
				"executor_id":        "workspace-runner",
				"executor_class":     "workspace_ordinary",
				"allowed_role_kinds": []any{"workspace-edit"},
			},
		},
		"gate_definitions": []any{gateDef("build_gate", "step_validation_started", 0)},
	})

	_, err := Compile(CompileInput{
		RunID:                   "run_123",
		PlanID:                  "plan_run_123_0001",
		SupersedesPlanID:        "plan_run_123_0001",
		WorkflowDefinitionBytes: workflowPayload,
		ProcessDefinitionBytes:  processPayload,
		PolicyContextHash:       "sha256:" + strings.Repeat("a", 64),
		ExecutorRegistry:        policyengine.BuildExecutorRegistryProjection(),
	})
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
}

func TestCompileUsesCurrentTimeWhenCompiledAtZero(t *testing.T) {
	workflowPayload := mustJSON(t, map[string]any{
		"schema_id":      workflowDefinitionSchemaID,
		"schema_version": workflowDefinitionVersion,
		"workflow_id":    "workflow_main",
		"executor_bindings": []any{
			map[string]any{
				"binding_id":         "binding_workspace_runner",
				"executor_id":        "workspace-runner",
				"executor_class":     "workspace_ordinary",
				"allowed_role_kinds": []any{"workspace-edit"},
			},
		},
		"gate_definitions": []any{gateDef("build_gate", "step_validation_started", 0)},
	})
	processPayload := mustJSON(t, map[string]any{
		"schema_id":      processDefinitionSchemaID,
		"schema_version": processDefinitionVersion,
		"process_id":     "process_default",
		"executor_bindings": []any{
			map[string]any{
				"binding_id":         "binding_workspace_runner",
				"executor_id":        "workspace-runner",
				"executor_class":     "workspace_ordinary",
				"allowed_role_kinds": []any{"workspace-edit"},
			},
		},
		"gate_definitions": []any{gateDef("build_gate", "step_validation_started", 0)},
	})
	plan, err := Compile(CompileInput{
		RunID:                   "run_123",
		PlanID:                  "plan_run_123_0001",
		WorkflowDefinitionBytes: workflowPayload,
		ProcessDefinitionBytes:  processPayload,
		PolicyContextHash:       "sha256:" + strings.Repeat("a", 64),
		ExecutorRegistry:        policyengine.BuildExecutorRegistryProjection(),
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if plan.CompiledAt == "0001-01-01T00:00:00Z" {
		t.Fatal("compiled_at used zero time, want current timestamp fallback")
	}
	if _, err := time.Parse(time.RFC3339, plan.CompiledAt); err != nil {
		t.Fatalf("compiled_at parse error: %v", err)
	}
}

func gateDef(gateID, checkpoint string, order int) map[string]any {
	return map[string]any{
		"schema_id":           gateDefinitionSchemaID,
		"schema_version":      gateDefinitionVersion,
		"checkpoint_code":     checkpoint,
		"order_index":         order,
		"role_instance_id":    "workspace_editor_1",
		"executor_binding_id": "binding_workspace_runner",
		"gate": map[string]any{
			"schema_id":      "runecode.protocol.v0.GateContract",
			"schema_version": "0.1.0",
			"gate_id":        gateID,
			"gate_kind":      "build",
			"gate_version":   "1.0.0",
			"normalized_inputs": []any{
				map[string]any{"input_id": "source_tree", "input_digest": "sha256:" + strings.Repeat("1", 64)},
			},
			"plan_binding":       map[string]any{"checkpoint_code": checkpoint, "order_index": order},
			"retry_semantics":    map[string]any{"retry_mode": "new_attempt_required", "max_attempts": 3},
			"override_semantics": map[string]any{"override_mode": "policy_action_required", "action_kind": "action_gate_override", "approval_trigger_code": "gate_override"},
		},
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	return b
}
