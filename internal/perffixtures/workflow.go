package perffixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	FixtureWorkflowFirstPartyMinimal = "workflow.first-party-minimal.v1"
	FixtureWorkflowCHG050Compile     = "workflow.chg050-compile.v1"
)

type WorkflowFixtureResult struct {
	FixtureID string
	RootDir   string
	RunPlan   string
	Workspace string
}

func BuildWorkflowFixture(rootDir, fixtureID string) (WorkflowFixtureResult, error) {
	workspace := filepath.Join(rootDir, "workspace")
	runplan := filepath.Join(rootDir, "runplan.json")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return WorkflowFixtureResult{}, err
	}
	var runplanContent []byte
	switch fixtureID {
	case FixtureWorkflowFirstPartyMinimal:
		raw, err := json.MarshalIndent(validRunPlanFixture("workflow.first-party-minimal.v1", "workflow_first_party_minimal", "process_first_party_minimal"), "", "  ")
		if err != nil {
			return WorkflowFixtureResult{}, err
		}
		runplanContent = raw
	case FixtureWorkflowCHG050Compile:
		raw, err := json.MarshalIndent(validRunPlanFixture("workflow.chg050-compile.v1", "workflow_chg050_compile", "process_chg050_compile"), "", "  ")
		if err != nil {
			return WorkflowFixtureResult{}, err
		}
		runplanContent = raw
	default:
		return WorkflowFixtureResult{}, fmt.Errorf("%w: %s", ErrUnsupportedFixtureID, fixtureID)
	}
	if err := os.WriteFile(runplan, runplanContent, 0o644); err != nil {
		return WorkflowFixtureResult{}, err
	}
	return WorkflowFixtureResult{FixtureID: fixtureID, RootDir: rootDir, RunPlan: runplan, Workspace: workspace}, nil
}

func validRunPlanFixture(fixtureID, workflowID, processID string) map[string]any {
	gate := runPlanGateContract()
	gateDefinition := runPlanGateDefinition(gate)
	entryDefinition := runPlanEntryDefinition(gate)
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.RunPlan",
		"schema_version":           "0.4.0",
		"plan_id":                  "plan_" + workflowID,
		"run_id":                   "run_" + workflowID,
		"workflow_id":              workflowID,
		"workflow_version":         "1.0.0",
		"process_id":               processID,
		"approval_profile":         "moderate",
		"autonomy_posture":         "balanced",
		"workflow_definition_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"process_definition_hash":  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"policy_context_hash":      "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"compiled_at":              "2026-01-01T00:00:00Z",
		"role_instance_ids":        []string{"role_alpha"},
		"executor_bindings": []map[string]any{{
			"binding_id":         "binding_alpha",
			"executor_id":        "executor_alpha",
			"executor_class":     "workspace_ordinary",
			"allowed_role_kinds": []string{"developer"},
		}},
		"gate_definitions": []map[string]any{gateDefinition},
		"dependency_edges": []any{},
		"entries":          []map[string]any{entryDefinition},
	}
}

func runPlanGateDefinition(gate map[string]any) map[string]any {
	return map[string]any{
		"schema_id":                 "runecode.protocol.v0.GateDefinition",
		"schema_version":            "0.2.0",
		"gate":                      gate,
		"checkpoint_code":           "quality",
		"order_index":               0,
		"stage_id":                  "quality_stage",
		"step_id":                   "quality_lint",
		"role_instance_id":          "role_alpha",
		"executor_binding_id":       "binding_alpha",
		"dependency_cache_handoffs": fixtureDependencyCacheHandoffs(),
	}
}

func runPlanEntryDefinition(gate map[string]any) map[string]any {
	return map[string]any{
		"entry_id":                  "quality_lint",
		"entry_kind":                "gate",
		"order_index":               0,
		"stage_id":                  "quality_stage",
		"step_id":                   "quality_lint",
		"role_instance_id":          "role_alpha",
		"executor_binding_id":       "binding_alpha",
		"checkpoint_code":           "quality",
		"gate":                      gate,
		"dependency_cache_handoffs": fixtureDependencyCacheHandoffs(),
		"depends_on_entry_ids":      []string{},
		"blocks_entry_ids":          []string{},
		"supported_wait_kinds":      []string{"waiting_operator_input", "waiting_approval"},
	}
}

func runPlanGateContract() map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.GateContract",
		"schema_version":     "0.1.0",
		"gate_id":            "lint",
		"gate_kind":          "lint",
		"gate_version":       "0.1.0",
		"normalized_inputs":  []any{},
		"plan_binding":       map[string]any{"checkpoint_code": "quality", "order_index": 0},
		"retry_semantics":    map[string]any{"retry_mode": "new_attempt_required", "max_attempts": 2},
		"override_semantics": map[string]any{"override_mode": "policy_action_required", "action_kind": "action_gate_override", "approval_trigger_code": "gate_override"},
	}
}

func fixtureDependencyCacheHandoffs() []map[string]any {
	return []map[string]any{{
		"request_digest": map[string]any{"hash_alg": "sha256", "hash": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		"consumer_role":  "workspace",
		"required":       true,
	}}
}
