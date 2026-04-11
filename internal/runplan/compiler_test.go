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

	plan, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	assertDeterministicRunPlanShape(t, plan)
	if err := ValidateRunPlan(plan); err != nil {
		t.Fatalf("ValidateRunPlan returned error: %v", err)
	}
}

func deterministicCompileInput(workflowPayload []byte, processPayload []byte) CompileInput {
	return CompileInput{
		RunID:                   "run_123",
		PlanID:                  "plan_run_123_0001",
		SupersedesPlanID:        "plan_run_123_0000",
		CompiledAt:              time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		WorkflowDefinitionBytes: workflowPayload,
		ProcessDefinitionBytes:  processPayload,
		PolicyContextHash:       "sha256:" + strings.Repeat("a", 64),
		ExecutorRegistry:        policyengine.BuildExecutorRegistryProjection(),
	}
}

func assertDeterministicRunPlanShape(t *testing.T, plan RunPlan) {
	t.Helper()
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
}

func TestCompileFailsClosedOnUnknownExecutorBinding(t *testing.T) {
	workflowPayload := workflowPayloadWithSingleBinding(t, "binding_unknown", "unknown-executor", []string{"workspace-edit"})
	processPayload := processPayloadWithSingleBinding(t, "binding_unknown", "unknown-executor", []string{"workspace-edit"})

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
	workflowPayload := workflowPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})

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
	workflowPayload := workflowPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
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

func TestCompileKeepsDistinctGateKindVersionVariants(t *testing.T) {
	workflowPayload := workflowPayloadForTest(t, []any{gateDefWithKindVersion("same_gate", "step_validation_started", 0, "build", "1.0.0")}, []string{"workspace-edit"})
	processPayload := processPayloadForTest(t, []any{gateDefWithKindVersion("same_gate", "step_validation_started", 0, "test", "2.0.0")}, []string{"workspace-edit"})

	plan, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(plan.GateDefinitions) != 2 {
		t.Fatalf("gate_definitions len = %d, want 2 distinct kind/version variants", len(plan.GateDefinitions))
	}
	seen := map[string]bool{}
	for _, gate := range plan.GateDefinitions {
		seen[gate.Gate.GateKind+"@"+gate.Gate.GateVersion] = true
	}
	if !seen["build@1.0.0"] || !seen["test@2.0.0"] {
		t.Fatalf("compiled gate variants = %+v, want build@1.0.0 and test@2.0.0", seen)
	}
}

func TestCompileRejectsConflictingGateDefinitionForSameDedupeKey(t *testing.T) {
	workflowPayload := workflowPayloadForTest(t, []any{gateDef("build_gate", "step_validation_started", 0)}, []string{"workspace-edit"})
	processGate := gateDef("build_gate", "step_validation_started", 0)
	processGate["executor_binding_id"] = "binding_workspace_runner_alt"
	processPayload := mustJSON(t, map[string]any{
		"schema_id":      processDefinitionSchemaID,
		"schema_version": processDefinitionVersion,
		"process_id":     "process_default",
		"executor_bindings": []any{
			executorBindingFixture("binding_workspace_runner", "workspace-runner", []string{"workspace-edit"}),
			executorBindingFixture("binding_workspace_runner_alt", "workspace-runner", []string{"workspace-edit"}),
		},
		"gate_definitions": []any{processGate},
	})

	_, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err == nil {
		t.Fatal("Compile error = nil, want conflict rejection for same dedupe key")
	}
	if !strings.Contains(err.Error(), "conflicts across workflow/process inputs") {
		t.Fatalf("Compile error = %v, want conflict error", err)
	}
}

func gateDef(gateID, checkpoint string, order int) map[string]any {
	return gateDefWithKindVersion(gateID, checkpoint, order, "build", "1.0.0")
}

func gateDefWithKindVersion(gateID, checkpoint string, order int, gateKind, gateVersion string) map[string]any {
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
			"gate_kind":      gateKind,
			"gate_version":   gateVersion,
			"normalized_inputs": []any{
				map[string]any{"input_id": "source_tree", "input_digest": "sha256:" + strings.Repeat("1", 64)},
			},
			"plan_binding":       map[string]any{"checkpoint_code": checkpoint, "order_index": order},
			"retry_semantics":    map[string]any{"retry_mode": "new_attempt_required", "max_attempts": 3},
			"override_semantics": map[string]any{"override_mode": "policy_action_required", "action_kind": "action_gate_override", "approval_trigger_code": "gate_override"},
		},
	}
}

func workflowPayloadWithSingleBinding(t *testing.T, bindingID, executorID string, roles []string) []byte {
	t.Helper()
	return mustJSON(t, map[string]any{
		"schema_id":         workflowDefinitionSchemaID,
		"schema_version":    workflowDefinitionVersion,
		"workflow_id":       "workflow_main",
		"executor_bindings": []any{executorBindingFixture(bindingID, executorID, roles)},
		"gate_definitions":  []any{gateDef("build_gate", "step_validation_started", 0)},
	})
}

func processPayloadWithSingleBinding(t *testing.T, bindingID, executorID string, roles []string) []byte {
	t.Helper()
	return mustJSON(t, map[string]any{
		"schema_id":         processDefinitionSchemaID,
		"schema_version":    processDefinitionVersion,
		"process_id":        "process_default",
		"executor_bindings": []any{executorBindingFixture(bindingID, executorID, roles)},
		"gate_definitions":  []any{gateDef("build_gate", "step_validation_started", 0)},
	})
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	return b
}
