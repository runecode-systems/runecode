package runplan

import (
	"encoding/json"
	"fmt"
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
	workflow, process, workflowHash, processHash, err := decodeCompileDefinitions(input)
	if err != nil {
		return RunPlan{}, err
	}
	if err := validateCompileInput(input, workflow, process); err != nil {
		return RunPlan{}, err
	}
	if err := validateCompileBindings(input.ExecutorRegistry, workflow.ExecutorBindings, process.ExecutorBindings); err != nil {
		return RunPlan{}, err
	}
	compiledBindings, gates, roleIDs, err := compileMergedPlanShape(workflow, process)
	if err != nil {
		return RunPlan{}, err
	}
	plan := newRunPlan(input, workflow, process, workflowHash, processHash, roleIDs, compiledBindings, gates)
	if err := ValidateRunPlan(plan); err != nil {
		return RunPlan{}, err
	}
	return plan, nil
}

func decodeCompileDefinitions(input CompileInput) (WorkflowDefinition, ProcessDefinition, string, string, error) {
	workflow, workflowHash, err := decodeWorkflowDefinition(input.WorkflowDefinitionBytes)
	if err != nil {
		return WorkflowDefinition{}, ProcessDefinition{}, "", "", err
	}
	process, processHash, err := decodeProcessDefinition(input.ProcessDefinitionBytes)
	if err != nil {
		return WorkflowDefinition{}, ProcessDefinition{}, "", "", err
	}
	return workflow, process, workflowHash, processHash, nil
}

func validateCompileBindings(registry policyengine.ExecutorRegistryProjection, workflowBindings []ExecutorBinding, processBindings []ExecutorBinding) error {
	if err := validateBindingsAgainstTrustedRegistry(workflowBindings, registry); err != nil {
		return err
	}
	return validateBindingsAgainstTrustedRegistry(processBindings, registry)
}

func compileMergedPlanShape(workflow WorkflowDefinition, process ProcessDefinition) ([]ExecutorBinding, []GateDefinition, []string, error) {
	compiledBindings, err := mergeExecutorBindings(workflow.ExecutorBindings, process.ExecutorBindings)
	if err != nil {
		return nil, nil, nil, err
	}
	gates, roleIDs, err := compileGateDefinitions(workflow.GateDefinitions, process.GateDefinitions, compiledBindings)
	if err != nil {
		return nil, nil, nil, err
	}
	return compiledBindings, gates, roleIDs, nil
}

func newRunPlan(input CompileInput, workflow WorkflowDefinition, process ProcessDefinition, workflowHash string, processHash string, roleIDs []string, compiledBindings []ExecutorBinding, gates []GateDefinition) RunPlan {
	compiledAt := compiledAtRFC3339(input.CompiledAt)
	return RunPlan{
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
}

func compiledAtRFC3339(compiledAt time.Time) string {
	resolved := compiledAt
	if resolved.IsZero() {
		resolved = time.Now().UTC()
	}
	return resolved.UTC().Format(time.RFC3339)
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
	if err := validateCompileInputRequiredIDs(input, workflow, process); err != nil {
		return err
	}
	if err := validateCompileInputPolicyContextHash(input.PolicyContextHash); err != nil {
		return err
	}
	if err := validateCompileInputSupersedesPlanID(input.SupersedesPlanID, input.PlanID); err != nil {
		return err
	}
	if err := validateCompileInputNonEmptyCollections(workflow, process); err != nil {
		return err
	}
	return nil
}

func validateCompileInputRequiredIDs(input CompileInput, workflow WorkflowDefinition, process ProcessDefinition) error {
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
	return nil
}

func validateCompileInputPolicyContextHash(policyContextHash string) error {
	trimmed := strings.TrimSpace(policyContextHash)
	if trimmed == "" {
		return fmt.Errorf("policy_context_hash is required")
	}
	if _, err := policyengine.NormalizeHashIdentity(trimmed); err != nil {
		return fmt.Errorf("policy_context_hash invalid: %w", err)
	}
	return nil
}

func validateCompileInputSupersedesPlanID(supersedesPlanID string, planID string) error {
	trimmedSupersedes := strings.TrimSpace(supersedesPlanID)
	if trimmedSupersedes != "" && trimmedSupersedes == strings.TrimSpace(planID) {
		return fmt.Errorf("supersedes_plan_id must differ from plan_id")
	}
	return nil
}

func validateCompileInputNonEmptyCollections(workflow WorkflowDefinition, process ProcessDefinition) error {
	if len(workflow.GateDefinitions) == 0 || len(process.GateDefinitions) == 0 {
		return fmt.Errorf("workflow/process gate_definitions must be non-empty")
	}
	if len(workflow.ExecutorBindings) == 0 || len(process.ExecutorBindings) == 0 {
		return fmt.Errorf("workflow/process executor_bindings must be non-empty")
	}
	return nil
}
