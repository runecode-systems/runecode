package protocolschema

func validRunPlan() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.RunPlan",
		"schema_version":           "0.1.0",
		"plan_id":                  "plan_run_123_0001",
		"run_id":                   "run_123",
		"workflow_id":              "workflow_main",
		"process_id":               "process_default",
		"workflow_definition_hash": testDigestString("a"),
		"process_definition_hash":  testDigestString("b"),
		"policy_context_hash":      testDigestString("c"),
		"compiled_at":              "2026-04-10T12:00:00Z",
		"role_instance_ids":        []any{"workspace_editor_1"},
		"executor_bindings":        []any{validExecutorBindingFixture()},
		"gate_definitions":         []any{validRunPlanGateDefinitionFixture()},
	}
}

func validRunPlanWithSupersedesPlanID() map[string]any {
	plan := validRunPlan()
	plan["supersedes_plan_id"] = "plan_run_123_0000"
	return plan
}

func invalidRunPlanWithoutBindings() map[string]any {
	plan := validRunPlan()
	plan["executor_bindings"] = []any{}
	return plan
}

func validWorkflowDefinition() map[string]any {
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.WorkflowDefinition",
		"schema_version":    "0.2.0",
		"workflow_id":       "workflow_main",
		"executor_bindings": []any{validExecutorBindingFixture()},
		"gate_definitions":  []any{validRunPlanGateDefinitionFixture()},
	}
}

func invalidWorkflowDefinitionWithoutExecutorBindings() map[string]any {
	workflow := validWorkflowDefinition()
	delete(workflow, "executor_bindings")
	return workflow
}

func validProcessDefinition() map[string]any {
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.ProcessDefinition",
		"schema_version":    "0.2.0",
		"process_id":        "process_default",
		"executor_bindings": []any{validExecutorBindingFixture()},
		"gate_definitions":  []any{validRunPlanGateDefinitionFixture()},
	}
}

func invalidProcessDefinitionWithoutProcessID() map[string]any {
	process := validProcessDefinition()
	delete(process, "process_id")
	return process
}

func validExecutorBindingFixture() map[string]any {
	return map[string]any{
		"binding_id":         "binding_workspace_runner",
		"executor_id":        "workspace-runner",
		"executor_class":     "workspace_ordinary",
		"allowed_role_kinds": []any{"workspace-edit", "workspace-test"},
	}
}

func validRunPlanGateDefinitionFixture() map[string]any {
	return map[string]any{
		"schema_id":           "runecode.protocol.v0.GateDefinition",
		"schema_version":      "0.1.0",
		"checkpoint_code":     "step_validation_started",
		"order_index":         0,
		"role_instance_id":    "workspace_editor_1",
		"executor_binding_id": "binding_workspace_runner",
		"gate": map[string]any{
			"schema_id":      "runecode.protocol.v0.GateContract",
			"schema_version": "0.1.0",
			"gate_id":        "build_gate",
			"gate_kind":      "build",
			"gate_version":   "1.0.0",
			"normalized_inputs": []any{
				map[string]any{
					"input_id":     "source_tree",
					"input_digest": testDigestString("1"),
				},
			},
			"plan_binding": map[string]any{
				"checkpoint_code": "step_validation_started",
				"order_index":     0,
			},
			"retry_semantics": map[string]any{
				"retry_mode":   "new_attempt_required",
				"max_attempts": 3,
			},
			"override_semantics": map[string]any{
				"override_mode":         "policy_action_required",
				"action_kind":           "action_gate_override",
				"approval_trigger_code": "gate_override",
			},
		},
	}
}
