package runplan

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func compileGateDefinitions(process []GateDefinition, bindings []ExecutorBinding) ([]GateDefinition, []string, error) {
	bindingMap := bindingsByID(bindings)
	keySet, err := collectCompiledGateDefinitions(bindingMap, process)
	if err != nil {
		return nil, nil, err
	}
	return sortedCompiledGateDefinitions(keySet)
}

func bindingsByID(bindings []ExecutorBinding) map[string]ExecutorBinding {
	out := map[string]ExecutorBinding{}
	for _, binding := range bindings {
		out[binding.BindingID] = binding
	}
	return out
}

func collectCompiledGateDefinitions(bindingMap map[string]ExecutorBinding, process []GateDefinition) (map[string]GateDefinition, error) {
	keySet := map[string]GateDefinition{}
	for _, gate := range process {
		key, err := validatedCompiledGateKey(bindingMap, gate)
		if err != nil {
			return nil, err
		}
		if err := mergeCompiledGateDefinition(keySet, key, gate); err != nil {
			return nil, err
		}
	}
	return keySet, nil
}

func validatedCompiledGateKey(bindingMap map[string]ExecutorBinding, gate GateDefinition) (string, error) {
	if gate.SchemaID != gateDefinitionSchemaID {
		return "", fmt.Errorf("gate definition schema_id %q does not match %q", gate.SchemaID, gateDefinitionSchemaID)
	}
	if gate.SchemaVersion != gateDefinitionVersion {
		return "", fmt.Errorf("gate definition schema_version %q does not match %q", gate.SchemaVersion, gateDefinitionVersion)
	}
	if gate.Gate.SchemaID != "runecode.protocol.v0.GateContract" {
		return "", fmt.Errorf("gate %q gate.schema_id %q does not match runecode.protocol.v0.GateContract", gate.Gate.GateID, gate.Gate.SchemaID)
	}
	if gate.Gate.SchemaVersion != "0.1.0" {
		return "", fmt.Errorf("gate %q gate.schema_version %q does not match 0.1.0", gate.Gate.GateID, gate.Gate.SchemaVersion)
	}
	if strings.TrimSpace(gate.StageID) == "" {
		return "", fmt.Errorf("gate %q stage_id is required", gate.Gate.GateID)
	}
	if strings.TrimSpace(gate.StepID) == "" {
		return "", fmt.Errorf("gate %q step_id is required", gate.Gate.GateID)
	}
	if strings.TrimSpace(gate.RoleInstanceID) == "" {
		return "", fmt.Errorf("gate %q role_instance_id is required", gate.Gate.GateID)
	}
	binding, ok := bindingMap[gate.ExecutorBindingID]
	if !ok {
		return "", fmt.Errorf("gate %q references unknown executor_binding_id %q", gate.Gate.GateID, gate.ExecutorBindingID)
	}
	if err := validateGatePlanBinding(gate); err != nil {
		return "", err
	}
	if err := validateGateDependencyCacheHandoffs(gate, binding); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s|%09d|%s|%s|%s|%s", gate.StageID, gate.OrderIndex, gate.StepID, gate.Gate.GateID, gate.Gate.GateKind, gate.Gate.GateVersion), nil
}

func validateGatePlanBinding(gate GateDefinition) error {
	checkpointCode, _ := gate.Gate.PlanBinding["checkpoint_code"].(string)
	if strings.TrimSpace(checkpointCode) != strings.TrimSpace(gate.CheckpointCode) {
		return fmt.Errorf("gate %q plan_binding.checkpoint_code %q must match checkpoint_code %q", gate.Gate.GateID, checkpointCode, gate.CheckpointCode)
	}
	orderIndexRaw, ok := gate.Gate.PlanBinding["order_index"].(float64)
	if !ok {
		return fmt.Errorf("gate %q plan_binding.order_index is required", gate.Gate.GateID)
	}
	if int(orderIndexRaw) != gate.OrderIndex {
		return fmt.Errorf("gate %q plan_binding.order_index %d must match order_index %d", gate.Gate.GateID, int(orderIndexRaw), gate.OrderIndex)
	}
	return nil
}

func validateGateDependencyCacheHandoffs(gate GateDefinition, binding ExecutorBinding) error {
	if len(gate.DependencyCacheHandoffs) == 0 {
		return nil
	}
	if err := validateExecutorForDependencyHandoffs(gate, binding); err != nil {
		return err
	}
	allowedRoles := buildAllowedRoleSet(binding.AllowedRoleKinds)
	return validateDependencyHandoffEntries(gate, allowedRoles)
}

func validateExecutorForDependencyHandoffs(gate GateDefinition, binding ExecutorBinding) error {
	if strings.TrimSpace(binding.ExecutorClass) != "workspace_ordinary" {
		return fmt.Errorf("gate %q dependency_cache_handoffs require workspace_ordinary executor binding", gate.Gate.GateID)
	}
	return nil
}

func buildAllowedRoleSet(roles []string) map[string]struct{} {
	allowedRoles := map[string]struct{}{}
	for _, role := range roles {
		allowedRoles[strings.TrimSpace(role)] = struct{}{}
	}
	return allowedRoles
}

func validateDependencyHandoffEntries(gate GateDefinition, allowedRoles map[string]struct{}) error {
	seen := map[string]struct{}{}
	for _, handoff := range gate.DependencyCacheHandoffs {
		if err := validateSingleDependencyHandoff(gate, handoff, allowedRoles, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateSingleDependencyHandoff(gate GateDefinition, handoff DependencyCacheHandoff, allowedRoles map[string]struct{}, seen map[string]struct{}) error {
	digestIdentity, err := handoff.RequestDigest.Identity()
	if err != nil {
		return fmt.Errorf("gate %q dependency_cache_handoffs.request_digest invalid: %w", gate.Gate.GateID, err)
	}
	consumerRole := strings.TrimSpace(handoff.ConsumerRole)
	if !isSupportedDependencyHandoffConsumerRole(consumerRole) {
		return fmt.Errorf("gate %q dependency_cache_handoffs consumer_role %q is not supported", gate.Gate.GateID, consumerRole)
	}
	if _, ok := allowedRoles[consumerRole]; !ok {
		return fmt.Errorf("gate %q dependency_cache_handoffs consumer_role %q not allowed by executor binding", gate.Gate.GateID, consumerRole)
	}
	if !handoff.Required {
		return fmt.Errorf("gate %q dependency_cache_handoffs are fail-closed and require required=true", gate.Gate.GateID)
	}
	key := digestIdentity + "|" + consumerRole
	if _, dup := seen[key]; dup {
		return fmt.Errorf("gate %q dependency_cache_handoffs entry %q is duplicated", gate.Gate.GateID, key)
	}
	seen[key] = struct{}{}
	return nil
}

func isSupportedDependencyHandoffConsumerRole(role string) bool {
	switch strings.TrimSpace(role) {
	case "workspace", "workspace-read", "workspace-edit", "workspace-test":
		return true
	default:
		return false
	}
}

func mergeCompiledGateDefinition(keySet map[string]GateDefinition, key string, gate GateDefinition) error {
	existing, seen := keySet[key]
	if !seen {
		keySet[key] = gate
		return nil
	}
	if !reflect.DeepEqual(existing, gate) {
		return fmt.Errorf("gate definition %q conflicts within process definition", gate.Gate.GateID)
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
