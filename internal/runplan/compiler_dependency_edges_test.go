package runplan

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCompileRejectsDependencyEdgeWithUnknownStepIdentity(t *testing.T) {
	processPayload := mustJSON(t, map[string]any{
		"schema_id":         processDefinitionSchemaID,
		"schema_version":    processDefinitionVersion,
		"process_id":        "process_default",
		"executor_bindings": []any{executorBindingFixture("binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})},
		"gate_definitions": []any{
			gateDef("build_gate", "step_validation_started", 0),
		},
		"dependency_edges": []any{
			map[string]any{"upstream_step_id": "unknown_step", "downstream_step_id": "build_gate_build_step", "dependency_kind": "step_completed"},
		},
	})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	_, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err == nil {
		t.Fatal("Compile error = nil, want dependency edge rejection")
	}
	if !strings.Contains(err.Error(), "unknown upstream_step_id") {
		t.Fatalf("Compile error = %v, want unknown upstream_step_id error", err)
	}
}

func TestCompilePropagatesDependencyEdgesIntoRunPlan(t *testing.T) {
	processPayload := mustJSON(t, map[string]any{
		"schema_id":         processDefinitionSchemaID,
		"schema_version":    processDefinitionVersion,
		"process_id":        "process_default",
		"executor_bindings": []any{executorBindingFixture("binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})},
		"gate_definitions": []any{
			gateDef("build_gate", "step_validation_started", 0),
			gateDef("lint_gate", "step_validation_finished", 1),
		},
		"dependency_edges": []any{
			map[string]any{"upstream_step_id": "build_gate_build_step", "downstream_step_id": "lint_gate_build_step", "dependency_kind": "step_completed"},
		},
	})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	plan, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(plan.DependencyEdges) != 1 {
		t.Fatalf("dependency_edges len = %d, want 1", len(plan.DependencyEdges))
	}
	if got := plan.DependencyEdges[0].UpstreamStepID; got != "build_gate_build_step" {
		t.Fatalf("dependency_edges[0].upstream_step_id = %q, want build_gate_build_step", got)
	}
	if len(plan.Entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(plan.Entries))
	}
	if got := plan.Entries[1].DependsOnEntryIDs; len(got) != 1 || got[0] != "build_gate_build_step" {
		t.Fatalf("entries[1].depends_on_entry_ids = %+v, want [build_gate_build_step]", got)
	}
}

func TestCompileRejectsGatePlanBindingMismatch(t *testing.T) {
	gate := gateDef("build_gate", "step_validation_started", 0)
	gate["gate"].(map[string]any)["plan_binding"] = map[string]any{"checkpoint_code": "step_validation_finished", "order_index": 1}
	processPayload := processPayloadForTest(t, []any{gate}, []string{"workspace-edit"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	_, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err == nil {
		t.Fatal("Compile error=nil, want plan_binding mismatch rejection")
	}
	if !strings.Contains(err.Error(), "plan_binding") {
		t.Fatalf("Compile error = %v, want plan_binding rejection", err)
	}
}

func TestCompilePropagatesDependencyCacheHandoffsIntoRunPlan(t *testing.T) {
	gate := gateDef("build_gate", "step_validation_started", 0)
	gate["dependency_cache_handoffs"] = []any{map[string]any{"request_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "consumer_role": "workspace-edit", "required": true}}
	processPayload := processPayloadForTest(t, []any{gate}, []string{"workspace-edit", "workspace-test"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	plan, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(plan.GateDefinitions) != 1 {
		t.Fatalf("gate_definitions len = %d, want 1", len(plan.GateDefinitions))
	}
	if len(plan.GateDefinitions[0].DependencyCacheHandoffs) != 1 {
		t.Fatalf("dependency_cache_handoffs len = %d, want 1", len(plan.GateDefinitions[0].DependencyCacheHandoffs))
	}
	if got := plan.GateDefinitions[0].DependencyCacheHandoffs[0].ConsumerRole; got != "workspace-edit" {
		t.Fatalf("consumer_role = %q, want workspace-edit", got)
	}
}

func TestCompileRejectsDependencyCacheHandoffOutsideBindingRoles(t *testing.T) {
	gate := gateDef("build_gate", "step_validation_started", 0)
	gate["dependency_cache_handoffs"] = []any{map[string]any{"request_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "consumer_role": "workspace-test", "required": true}}
	processPayload := processPayloadForTest(t, []any{gate}, []string{"workspace-edit"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	_, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err == nil {
		t.Fatal("Compile error=nil, want dependency_cache_handoffs role rejection")
	}
	if !strings.Contains(err.Error(), "dependency_cache_handoffs") {
		t.Fatalf("Compile error = %v, want dependency_cache_handoffs rejection", err)
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
