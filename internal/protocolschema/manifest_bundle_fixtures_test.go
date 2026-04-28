package protocolschema

import (
	"encoding/json"
	"strings"
	"testing"
)

func cloneFixtureMap(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	payload := mustJSONBytes(t, value)
	return loadJSONMapFromBytes(t, payload)
}

func mustJSONBytes(t *testing.T, value map[string]any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	return payload
}

func loadJSONMapFromBytes(t *testing.T, payload []byte) map[string]any {
	t.Helper()
	parsed := map[string]any{}
	err := json.Unmarshal(payload, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	return parsed
}

func workflowDefinitionFixtureWithRequiredGates() map[string]any {
	return map[string]any{
		"schema_id":                        "runecode.protocol.v0.WorkflowDefinition",
		"schema_version":                   "0.5.0",
		"workflow_id":                      "workflow_main",
		"workflow_version":                 "1.0.0",
		"selected_process_id":              "process_default",
		"selected_process_definition_hash": "sha256:" + strings.Repeat("b", 64),
		"reviewed_process_artifacts": []any{
			map[string]any{
				"process_id":              "process_default",
				"process_definition_hash": "sha256:" + strings.Repeat("b", 64),
			},
		},
		"policy_binding_id": "policy_binding_default",
		"approval_profile":  "moderate",
		"autonomy_posture":  "balanced",
	}
}

func processDefinitionFixtureWithRequiredGates() map[string]any {
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.ProcessDefinition",
		"schema_version":    "0.4.0",
		"process_id":        "process_default",
		"executor_bindings": []any{executorBindingFixtureWithRequiredGates()},
		"gate_definitions":  []any{gateDefinitionFixtureWithRequiredGateContract()},
		"dependency_edges":  []any{},
	}
}

func runPlanFixtureWithRequiredGates() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.RunPlan",
		"schema_version":           "0.4.0",
		"plan_id":                  "plan_run_123_0001",
		"run_id":                   "run_123",
		"workflow_id":              "workflow_main",
		"workflow_version":         "1.0.0",
		"process_id":               "process_default",
		"approval_profile":         "moderate",
		"autonomy_posture":         "balanced",
		"policy_binding_id":        "policy_binding_default",
		"workflow_definition_hash": "sha256:" + strings.Repeat("a", 64),
		"process_definition_hash":  "sha256:" + strings.Repeat("b", 64),
		"policy_context_hash":      "sha256:" + strings.Repeat("c", 64),
		"compiled_at":              "2026-04-10T12:00:00Z",
		"role_instance_ids":        []any{"workspace_editor_1"},
		"executor_bindings":        []any{executorBindingFixtureWithRequiredGates()},
		"gate_definitions":         []any{gateDefinitionFixtureWithRequiredGateContract()},
		"dependency_edges":         []any{},
		"entries": []any{map[string]any{
			"entry_id":             "validation_build",
			"entry_kind":           "gate",
			"order_index":          0,
			"stage_id":             "validation",
			"step_id":              "validation_build",
			"role_instance_id":     "workspace_editor_1",
			"executor_binding_id":  "binding_workspace_runner",
			"checkpoint_code":      "step_validation_started",
			"gate":                 gateContractFixtureWithRequiredFields(),
			"depends_on_entry_ids": []any{},
			"blocks_entry_ids":     []any{},
			"supported_wait_kinds": []any{"waiting_operator_input", "waiting_approval"},
		}},
	}
}

func executorBindingFixtureWithRequiredGates() map[string]any {
	return map[string]any{
		"binding_id":         "binding_workspace_runner",
		"executor_id":        "workspace-runner",
		"executor_class":     "workspace_ordinary",
		"allowed_role_kinds": []any{"workspace-edit", "workspace-test"},
	}
}

func gateDefinitionFixtureWithRequiredGateContract() map[string]any {
	return map[string]any{
		"schema_id":           "runecode.protocol.v0.GateDefinition",
		"schema_version":      "0.2.0",
		"checkpoint_code":     "step_validation_started",
		"order_index":         0,
		"stage_id":            "validation",
		"step_id":             "validation_build",
		"role_instance_id":    "workspace_editor_1",
		"executor_binding_id": "binding_workspace_runner",
		"dependency_cache_handoffs": []any{
			map[string]any{
				"request_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)},
				"consumer_role":  "workspace",
				"required":       true,
			},
		},
		"gate": gateContractFixtureWithRequiredFields(),
	}
}

func gateContractFixtureWithRequiredFields() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GateContract",
		"schema_version": "0.1.0",
		"gate_id":        "build_gate",
		"gate_kind":      "build",
		"gate_version":   "1.0.0",
		"normalized_inputs": []any{
			map[string]any{"input_id": "source_tree", "input_digest": "sha256:" + strings.Repeat("1", 64)},
		},
		"plan_binding":       map[string]any{"checkpoint_code": "step_validation_started", "order_index": 0},
		"retry_semantics":    map[string]any{"retry_mode": "new_attempt_required", "max_attempts": 3},
		"override_semantics": map[string]any{"override_mode": "policy_action_required", "action_kind": "action_gate_override", "approval_trigger_code": "gate_override"},
	}
}
