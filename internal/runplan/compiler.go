package runplan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

const (
	workflowDefinitionSchemaID   = "runecode.protocol.v0.WorkflowDefinition"
	workflowDefinitionVersion    = "0.2.0"
	workflowDefinitionSchemaPath = "objects/WorkflowDefinition.schema.json"
	processDefinitionSchemaID    = "runecode.protocol.v0.ProcessDefinition"
	processDefinitionVersion     = "0.2.0"
	processDefinitionSchemaPath  = "objects/ProcessDefinition.schema.json"
	runPlanSchemaID              = "runecode.protocol.v0.RunPlan"
	runPlanVersion               = "0.1.0"
	runPlanSchemaPath            = "objects/RunPlan.schema.json"
	gateDefinitionSchemaID       = "runecode.protocol.v0.GateDefinition"
	gateDefinitionVersion        = "0.1.0"
)

type CompileInput struct {
	RunID                   string
	PlanID                  string
	SupersedesPlanID        string
	CompiledAt              time.Time
	WorkflowDefinitionBytes []byte
	ProcessDefinitionBytes  []byte
	PolicyContextHash       string
	ExecutorRegistry        policyengine.ExecutorRegistryProjection
}

func Compile(input CompileInput) (RunPlan, error) {
	workflow, workflowHash, err := decodeWorkflowDefinition(input.WorkflowDefinitionBytes)
	if err != nil {
		return RunPlan{}, err
	}
	process, processHash, err := decodeProcessDefinition(input.ProcessDefinitionBytes)
	if err != nil {
		return RunPlan{}, err
	}
	if err := validateCompileInput(input, workflow, process); err != nil {
		return RunPlan{}, err
	}

	if err := validateBindingsAgainstTrustedRegistry(workflow.ExecutorBindings, input.ExecutorRegistry); err != nil {
		return RunPlan{}, err
	}
	if err := validateBindingsAgainstTrustedRegistry(process.ExecutorBindings, input.ExecutorRegistry); err != nil {
		return RunPlan{}, err
	}

	compiledBindings, err := mergeExecutorBindings(workflow.ExecutorBindings, process.ExecutorBindings)
	if err != nil {
		return RunPlan{}, err
	}
	gates, roleIDs, err := compileGateDefinitions(workflow.GateDefinitions, process.GateDefinitions, compiledBindings)
	if err != nil {
		return RunPlan{}, err
	}

	compiledAtTime := input.CompiledAt
	if compiledAtTime.IsZero() {
		compiledAtTime = time.Now().UTC()
	}
	compiledAt := compiledAtTime.UTC().Format(time.RFC3339)

	plan := RunPlan{
		SchemaID:               runPlanSchemaID,
		SchemaVersion:          runPlanVersion,
		PlanID:                 strings.TrimSpace(input.PlanID),
		SupersedesPlanID:       strings.TrimSpace(input.SupersedesPlanID),
		RunID:                  strings.TrimSpace(input.RunID),
		WorkflowID:             workflow.WorkflowID,
		ProcessID:              process.ProcessID,
		WorkflowDefinitionHash: workflowHash,
		ProcessDefinitionHash:  processHash,
		PolicyContextHash:      strings.TrimSpace(input.PolicyContextHash),
		CompiledAt:             compiledAt,
		RoleInstanceIDs:        roleIDs,
		ExecutorBindings:       compiledBindings,
		GateDefinitions:        gates,
	}
	if err := ValidateRunPlan(plan); err != nil {
		return RunPlan{}, err
	}
	return plan, nil
}

func decodeWorkflowDefinition(payload []byte) (WorkflowDefinition, string, error) {
	if err := policyengine.ValidateObjectPayloadAgainstSchema(payload, workflowDefinitionSchemaPath); err != nil {
		return WorkflowDefinition{}, "", fmt.Errorf("workflow definition schema validation failed: %w", err)
	}
	value := WorkflowDefinition{}
	if err := json.Unmarshal(payload, &value); err != nil {
		return WorkflowDefinition{}, "", fmt.Errorf("decode workflow definition: %w", err)
	}
	hash, err := policyengine.CanonicalHashBytes(payload)
	if err != nil {
		return WorkflowDefinition{}, "", fmt.Errorf("hash workflow definition: %w", err)
	}
	if value.SchemaID != workflowDefinitionSchemaID {
		return WorkflowDefinition{}, "", fmt.Errorf("workflow definition schema_id %q does not match %q", value.SchemaID, workflowDefinitionSchemaID)
	}
	if value.SchemaVersion != workflowDefinitionVersion {
		return WorkflowDefinition{}, "", fmt.Errorf("workflow definition schema_version %q does not match %q", value.SchemaVersion, workflowDefinitionVersion)
	}
	return value, hash, nil
}

func decodeProcessDefinition(payload []byte) (ProcessDefinition, string, error) {
	if err := policyengine.ValidateObjectPayloadAgainstSchema(payload, processDefinitionSchemaPath); err != nil {
		return ProcessDefinition{}, "", fmt.Errorf("process definition schema validation failed: %w", err)
	}
	value := ProcessDefinition{}
	if err := json.Unmarshal(payload, &value); err != nil {
		return ProcessDefinition{}, "", fmt.Errorf("decode process definition: %w", err)
	}
	hash, err := policyengine.CanonicalHashBytes(payload)
	if err != nil {
		return ProcessDefinition{}, "", fmt.Errorf("hash process definition: %w", err)
	}
	if value.SchemaID != processDefinitionSchemaID {
		return ProcessDefinition{}, "", fmt.Errorf("process definition schema_id %q does not match %q", value.SchemaID, processDefinitionSchemaID)
	}
	if value.SchemaVersion != processDefinitionVersion {
		return ProcessDefinition{}, "", fmt.Errorf("process definition schema_version %q does not match %q", value.SchemaVersion, processDefinitionVersion)
	}
	return value, hash, nil
}

func ValidateRunPlan(plan RunPlan) error {
	payload, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("marshal run plan: %w", err)
	}
	if err := policyengine.ValidateObjectPayloadAgainstSchema(payload, runPlanSchemaPath); err != nil {
		return fmt.Errorf("run plan schema validation failed: %w", err)
	}
	if strings.TrimSpace(plan.SupersedesPlanID) != "" && strings.TrimSpace(plan.SupersedesPlanID) == strings.TrimSpace(plan.PlanID) {
		return fmt.Errorf("supersedes_plan_id must differ from plan_id")
	}
	return nil
}

func validateCompileInput(input CompileInput, workflow WorkflowDefinition, process ProcessDefinition) error {
	if strings.TrimSpace(input.RunID) == "" {
		return fmt.Errorf("run_id is required")
	}
	if strings.TrimSpace(input.PlanID) == "" {
		return fmt.Errorf("plan_id is required")
	}
	if strings.TrimSpace(workflow.WorkflowID) == "" {
		return fmt.Errorf("workflow_id is required")
	}
	if strings.TrimSpace(process.ProcessID) == "" {
		return fmt.Errorf("process_id is required")
	}
	if strings.TrimSpace(input.PolicyContextHash) == "" {
		return fmt.Errorf("policy_context_hash is required")
	}
	if _, err := policyengine.NormalizeHashIdentity(strings.TrimSpace(input.PolicyContextHash)); err != nil {
		return fmt.Errorf("policy_context_hash invalid: %w", err)
	}
	if strings.TrimSpace(input.SupersedesPlanID) != "" && strings.TrimSpace(input.SupersedesPlanID) == strings.TrimSpace(input.PlanID) {
		return fmt.Errorf("supersedes_plan_id must differ from plan_id")
	}
	if len(workflow.GateDefinitions) == 0 || len(process.GateDefinitions) == 0 {
		return fmt.Errorf("workflow/process gate_definitions must be non-empty")
	}
	if len(workflow.ExecutorBindings) == 0 || len(process.ExecutorBindings) == 0 {
		return fmt.Errorf("workflow/process executor_bindings must be non-empty")
	}
	return nil
}

func validateBindingsAgainstTrustedRegistry(bindings []ExecutorBinding, registry policyengine.ExecutorRegistryProjection) error {
	byID := map[string]policyengine.ExecutorProjectionRecord{}
	for _, rec := range registry.Executors {
		byID[rec.ExecutorID] = rec
	}
	for _, binding := range bindings {
		rec, ok := byID[binding.ExecutorID]
		if !ok {
			return fmt.Errorf("executor_binding %q references unknown trusted executor_id %q", binding.BindingID, binding.ExecutorID)
		}
		if rec.ExecutorClass != binding.ExecutorClass {
			return fmt.Errorf("executor_binding %q class %q does not match trusted executor class %q", binding.BindingID, binding.ExecutorClass, rec.ExecutorClass)
		}
		allowed := toSet(rec.AllowedRoles)
		for _, roleKind := range binding.AllowedRoleKinds {
			if _, ok := allowed[roleKind]; !ok {
				return fmt.Errorf("executor_binding %q role_kind %q not allowed by trusted executor registry", binding.BindingID, roleKind)
			}
		}
	}
	return nil
}

func mergeExecutorBindings(workflow []ExecutorBinding, process []ExecutorBinding) ([]ExecutorBinding, error) {
	merged := map[string]ExecutorBinding{}
	for _, binding := range append(append([]ExecutorBinding{}, workflow...), process...) {
		if strings.TrimSpace(binding.BindingID) == "" {
			return nil, fmt.Errorf("executor binding_id is required")
		}
		existing, seen := merged[binding.BindingID]
		if !seen {
			copyBinding := binding
			copyBinding.AllowedRoleKinds = sortedUniqueStrings(binding.AllowedRoleKinds)
			merged[binding.BindingID] = copyBinding
			continue
		}
		if existing.ExecutorID != binding.ExecutorID || existing.ExecutorClass != binding.ExecutorClass {
			return nil, fmt.Errorf("executor binding %q conflicts across workflow/process inputs", binding.BindingID)
		}
		combinedRoles := append(existing.AllowedRoleKinds, binding.AllowedRoleKinds...)
		existing.AllowedRoleKinds = sortedUniqueStrings(combinedRoles)
		if existing.Description == "" {
			existing.Description = binding.Description
		}
		merged[binding.BindingID] = existing
	}
	bindingIDs := make([]string, 0, len(merged))
	for bindingID := range merged {
		bindingIDs = append(bindingIDs, bindingID)
	}
	sort.Strings(bindingIDs)
	out := make([]ExecutorBinding, 0, len(bindingIDs))
	for _, bindingID := range bindingIDs {
		out = append(out, merged[bindingID])
	}
	return out, nil
}

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
	return fmt.Sprintf("%s|%09d|%s", gate.CheckpointCode, gate.OrderIndex, gate.Gate.GateID), nil
}

func mergeCompiledGateDefinition(keySet map[string]GateDefinition, key string, gate GateDefinition) error {
	existing, seen := keySet[key]
	if !seen {
		keySet[key] = gate
		return nil
	}
	if existing.ExecutorBindingID != gate.ExecutorBindingID || existing.RoleInstanceID != gate.RoleInstanceID {
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

func toSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func sortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	set := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
