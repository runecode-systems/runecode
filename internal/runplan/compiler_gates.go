package runplan

import (
	"fmt"
	"reflect"
	"sort"
)

func compileGateDefinitions(workflow []GateDefinition, process []GateDefinition, bindings []ExecutorBinding) ([]GateDefinition, []string, error) {
	bindingSet := bindingIDSet(bindings)
	keySet, err := collectCompiledGateDefinitions(bindingSet, workflow, process)
	if err != nil {
		return nil, nil, err
	}
	return sortedCompiledGateDefinitions(keySet)
}

func bindingIDSet(bindings []ExecutorBinding) map[string]struct{} {
	out := map[string]struct{}{}
	for _, binding := range bindings {
		out[binding.BindingID] = struct{}{}
	}
	return out
}

func collectCompiledGateDefinitions(bindingSet map[string]struct{}, workflow []GateDefinition, process []GateDefinition) (map[string]GateDefinition, error) {
	keySet := map[string]GateDefinition{}
	for _, gate := range append(append([]GateDefinition{}, workflow...), process...) {
		key, err := validatedCompiledGateKey(bindingSet, gate)
		if err != nil {
			return nil, err
		}
		if err := mergeCompiledGateDefinition(keySet, key, gate); err != nil {
			return nil, err
		}
	}
	return keySet, nil
}

func validatedCompiledGateKey(bindingSet map[string]struct{}, gate GateDefinition) (string, error) {
	if gate.SchemaID != gateDefinitionSchemaID {
		return "", fmt.Errorf("gate definition schema_id %q does not match %q", gate.SchemaID, gateDefinitionSchemaID)
	}
	if gate.SchemaVersion != gateDefinitionVersion {
		return "", fmt.Errorf("gate definition schema_version %q does not match %q", gate.SchemaVersion, gateDefinitionVersion)
	}
	if _, ok := bindingSet[gate.ExecutorBindingID]; !ok {
		return "", fmt.Errorf("gate %q references unknown executor_binding_id %q", gate.Gate.GateID, gate.ExecutorBindingID)
	}
	return fmt.Sprintf("%s|%09d|%s|%s|%s", gate.CheckpointCode, gate.OrderIndex, gate.Gate.GateID, gate.Gate.GateKind, gate.Gate.GateVersion), nil
}

func mergeCompiledGateDefinition(keySet map[string]GateDefinition, key string, gate GateDefinition) error {
	existing, seen := keySet[key]
	if !seen {
		keySet[key] = gate
		return nil
	}
	if !reflect.DeepEqual(existing, gate) {
		return fmt.Errorf("gate definition %q conflicts across workflow/process inputs", gate.Gate.GateID)
	}
	return nil
}

func sortedCompiledGateDefinitions(keySet map[string]GateDefinition) ([]GateDefinition, []string, error) {
	keys := make([]string, 0, len(keySet))
	roleSet := map[string]struct{}{}
	for key, gate := range keySet {
		keys = append(keys, key)
		roleSet[gate.RoleInstanceID] = struct{}{}
	}
	sort.Strings(keys)
	gates := make([]GateDefinition, 0, len(keys))
	for _, key := range keys {
		gates = append(gates, keySet[key])
	}
	roleIDs := make([]string, 0, len(roleSet))
	for roleID := range roleSet {
		roleIDs = append(roleIDs, roleID)
	}
	sort.Strings(roleIDs)
	return gates, roleIDs, nil
}
