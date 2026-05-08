package protocolschema

import "testing"

func TestRunPlanValidMinimalFixtureUsesRequiredFieldsOnly(t *testing.T) {
	fixture := loadJSONMap(t, fixturePath(t, "schema/run-plan.valid-minimal.json"))
	assertRunPlanFixtureTopLevelMinimal(t, fixture)
	assertRunPlanFixtureExecutorBindingMinimal(t, fixture)
	assertRunPlanFixtureGateDefinitionMinimal(t, fixture)
	assertRunPlanFixtureEntryMinimal(t, fixture)
	assertRunPlanFixtureGateDefinitionNormalizedInputsEmpty(t, fixture)
}

func assertRunPlanFixtureTopLevelMinimal(t *testing.T, fixture map[string]any) {
	t.Helper()

	assertSameStringSet(t, sortedKeys(fixture), []string{
		"approval_profile",
		"autonomy_posture",
		"compiled_at",
		"dependency_edges",
		"entries",
		"executor_bindings",
		"gate_definitions",
		"plan_id",
		"policy_context_hash",
		"process_definition_hash",
		"process_id",
		"role_instance_ids",
		"run_id",
		"schema_id",
		"schema_version",
		"workflow_definition_hash",
		"workflow_id",
		"workflow_version",
	})
}

func assertRunPlanFixtureExecutorBindingMinimal(t *testing.T, fixture map[string]any) {
	t.Helper()
	executorBindings, err := requiredArrayValue(fixture, "executor_bindings")
	if err != nil {
		t.Fatalf("requiredArrayValue(executor_bindings): %v", err)
	}
	executorBinding, err := objectFromFixtureValue(executorBindings[0], "executor_bindings[0]")
	if err != nil {
		t.Fatalf("objectFromFixtureValue(executor_bindings[0]): %v", err)
	}
	assertSameStringSet(t, sortedKeys(executorBinding), []string{
		"allowed_role_kinds",
		"binding_id",
		"executor_class",
		"executor_id",
	})
}

func assertRunPlanFixtureGateDefinitionMinimal(t *testing.T, fixture map[string]any) {
	t.Helper()
	gateDefinitions, err := requiredArrayValue(fixture, "gate_definitions")
	if err != nil {
		t.Fatalf("requiredArrayValue(gate_definitions): %v", err)
	}
	gateDefinition, err := objectFromFixtureValue(gateDefinitions[0], "gate_definitions[0]")
	if err != nil {
		t.Fatalf("objectFromFixtureValue(gate_definitions[0]): %v", err)
	}
	assertSameStringSet(t, sortedKeys(gateDefinition), []string{
		"checkpoint_code",
		"executor_binding_id",
		"gate",
		"order_index",
		"role_instance_id",
		"schema_id",
		"schema_version",
		"stage_id",
		"step_id",
	})
	assertRunPlanFixtureGateContractMinimal(t, objectValue(t, gateDefinition, "gate"), "gate_definitions[0].gate")
}

func assertRunPlanFixtureEntryMinimal(t *testing.T, fixture map[string]any) {
	t.Helper()
	entries, err := requiredArrayValue(fixture, "entries")
	if err != nil {
		t.Fatalf("requiredArrayValue(entries): %v", err)
	}
	entry, err := objectFromFixtureValue(entries[0], "entries[0]")
	if err != nil {
		t.Fatalf("objectFromFixtureValue(entries[0]): %v", err)
	}
	assertSameStringSet(t, sortedKeys(entry), []string{
		"blocks_entry_ids",
		"checkpoint_code",
		"depends_on_entry_ids",
		"entry_id",
		"entry_kind",
		"executor_binding_id",
		"gate",
		"order_index",
		"role_instance_id",
		"stage_id",
		"step_id",
		"supported_wait_kinds",
	})
	assertRunPlanFixtureGateContractMinimal(t, objectValue(t, entry, "gate"), "entries[0].gate")
}

func assertRunPlanFixtureGateDefinitionNormalizedInputsEmpty(t *testing.T, fixture map[string]any) {
	t.Helper()
	gateDefinitions, err := requiredArrayValue(fixture, "gate_definitions")
	if err != nil {
		t.Fatalf("requiredArrayValue(gate_definitions): %v", err)
	}
	gateDefinition, err := objectFromFixtureValue(gateDefinitions[0], "gate_definitions[0]")
	if err != nil {
		t.Fatalf("objectFromFixtureValue(gate_definitions[0]): %v", err)
	}
	normalizedInputs, err := requiredArrayValue(objectValue(t, gateDefinition, "gate"), "normalized_inputs")
	if err != nil {
		t.Fatalf("requiredArrayValue(gate_definitions[0].gate.normalized_inputs): %v", err)
	}
	if len(normalizedInputs) != 0 {
		t.Fatalf("gate_definitions[0].gate.normalized_inputs length = %d, want 0", len(normalizedInputs))
	}
}

func assertRunPlanFixtureGateContractMinimal(t *testing.T, gate map[string]any, location string) {
	t.Helper()

	assertSameStringSet(t, sortedKeys(gate), []string{
		"gate_id",
		"gate_kind",
		"gate_version",
		"normalized_inputs",
		"override_semantics",
		"plan_binding",
		"retry_semantics",
		"schema_id",
		"schema_version",
	})

	normalizedInputs, err := requiredArrayValue(gate, "normalized_inputs")
	if err != nil {
		t.Fatalf("requiredArrayValue(%s.normalized_inputs): %v", location, err)
	}
	if len(normalizedInputs) != 0 {
		t.Fatalf("%s.normalized_inputs length = %d, want 0", location, len(normalizedInputs))
	}
}
