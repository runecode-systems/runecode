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
	workflowDefinitionVersion    = "0.4.0"
	workflowDefinitionSchemaPath = "objects/WorkflowDefinition.schema.json"
	processDefinitionSchemaID    = "runecode.protocol.v0.ProcessDefinition"
	processDefinitionVersion     = "0.4.0"
	processDefinitionSchemaPath  = "objects/ProcessDefinition.schema.json"
	runPlanSchemaID              = "runecode.protocol.v0.RunPlan"
	runPlanVersion               = "0.3.0"
	runPlanSchemaPath            = "objects/RunPlan.schema.json"
	gateDefinitionSchemaID       = "runecode.protocol.v0.GateDefinition"
	gateDefinitionVersion        = "0.2.0"
)

type CompileInput struct {
	RunID                        string
	PlanID                       string
	SupersedesPlanID             string
	CompiledAt                   time.Time
	WorkflowDefinitionBytes      []byte
	ProcessDefinitionBytes       []byte
	ProjectContextIdentityDigest string
	PolicyContextHash            string
	ExecutorRegistry             policyengine.ExecutorRegistryProjection
}

func Compile(input CompileInput) (RunPlan, error) {
	workflow, process, workflowHash, processHash, err := decodeCompileDefinitions(input)
	if err != nil {
		return RunPlan{}, err
	}
	if err := validateCompileInput(input, workflow, process); err != nil {
		return RunPlan{}, err
	}
	if err := validateBindingsAgainstTrustedRegistry(process.ExecutorBindings, input.ExecutorRegistry); err != nil {
		return RunPlan{}, err
	}
	compiledBindings, gates, roleIDs, dependencyEdges, err := compileProcessPlanShape(process)
	if err != nil {
		return RunPlan{}, err
	}
	plan := newRunPlan(input, workflow, process, workflowHash, processHash, roleIDs, compiledBindings, gates, dependencyEdges)
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

func compileProcessPlanShape(process ProcessDefinition) ([]ExecutorBinding, []GateDefinition, []string, []DependencyEdge, error) {
	compiledBindings, err := compileExecutorBindings(process.ExecutorBindings)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	gates, roleIDs, err := compileGateDefinitions(process.GateDefinitions, compiledBindings)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	dependencyEdges, err := compileProcessDependencyEdges(gates, process.DependencyEdges)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return compiledBindings, gates, roleIDs, dependencyEdges, nil
}

func newRunPlan(input CompileInput, workflow WorkflowDefinition, process ProcessDefinition, workflowHash string, processHash string, roleIDs []string, compiledBindings []ExecutorBinding, gates []GateDefinition, dependencyEdges []DependencyEdge) RunPlan {
	compiledAt := compiledAtRFC3339(input.CompiledAt)
	return RunPlan{
		SchemaID:                     runPlanSchemaID,
		SchemaVersion:                runPlanVersion,
		PlanID:                       strings.TrimSpace(input.PlanID),
		SupersedesPlanID:             strings.TrimSpace(input.SupersedesPlanID),
		RunID:                        strings.TrimSpace(input.RunID),
		ProjectContextIdentityDigest: strings.TrimSpace(input.ProjectContextIdentityDigest),
		WorkflowID:                   workflow.WorkflowID,
		WorkflowVersion:              workflow.WorkflowVersion,
		ProcessID:                    process.ProcessID,
		ApprovalProfile:              workflow.ApprovalProfile,
		AutonomyPosture:              workflow.AutonomyPosture,
		PolicyBindingID:              workflow.PolicyBindingID,
		WorkflowDefinitionHash:       workflowHash,
		ProcessDefinitionHash:        processHash,
		PolicyContextHash:            strings.TrimSpace(input.PolicyContextHash),
		CompiledAt:                   compiledAt,
		RoleInstanceIDs:              roleIDs,
		ExecutorBindings:             compiledBindings,
		GateDefinitions:              gates,
		DependencyEdges:              dependencyEdges,
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
	canonicalPayload, err := canonicalizeCompileDefinitionPayload(payload, "workflow definition")
	if err != nil {
		return WorkflowDefinition{}, "", err
	}
	if err := policyengine.ValidateObjectPayloadAgainstSchema(canonicalPayload, workflowDefinitionSchemaPath); err != nil {
		return WorkflowDefinition{}, "", fmt.Errorf("workflow definition schema validation failed: %w", err)
	}
	value := WorkflowDefinition{}
	if err := json.Unmarshal(canonicalPayload, &value); err != nil {
		return WorkflowDefinition{}, "", fmt.Errorf("decode workflow definition: %w", err)
	}
	hash := policyengine.HashCanonicalJSONBytes(canonicalPayload)
	if value.SchemaID != workflowDefinitionSchemaID {
		return WorkflowDefinition{}, "", fmt.Errorf("workflow definition schema_id %q does not match %q", value.SchemaID, workflowDefinitionSchemaID)
	}
	if value.SchemaVersion != workflowDefinitionVersion {
		return WorkflowDefinition{}, "", fmt.Errorf("workflow definition schema_version %q does not match %q", value.SchemaVersion, workflowDefinitionVersion)
	}
	return value, hash, nil
}

func decodeProcessDefinition(payload []byte) (ProcessDefinition, string, error) {
	canonicalPayload, err := canonicalizeCompileDefinitionPayload(payload, "process definition")
	if err != nil {
		return ProcessDefinition{}, "", err
	}
	if err := policyengine.ValidateObjectPayloadAgainstSchema(canonicalPayload, processDefinitionSchemaPath); err != nil {
		return ProcessDefinition{}, "", fmt.Errorf("process definition schema validation failed: %w", err)
	}
	value := ProcessDefinition{}
	if err := json.Unmarshal(canonicalPayload, &value); err != nil {
		return ProcessDefinition{}, "", fmt.Errorf("decode process definition: %w", err)
	}
	hash := policyengine.HashCanonicalJSONBytes(canonicalPayload)
	if value.SchemaID != processDefinitionSchemaID {
		return ProcessDefinition{}, "", fmt.Errorf("process definition schema_id %q does not match %q", value.SchemaID, processDefinitionSchemaID)
	}
	if value.SchemaVersion != processDefinitionVersion {
		return ProcessDefinition{}, "", fmt.Errorf("process definition schema_version %q does not match %q", value.SchemaVersion, processDefinitionVersion)
	}
	return value, hash, nil
}

func canonicalizeCompileDefinitionPayload(payload []byte, definitionKind string) ([]byte, error) {
	canonicalPayload, err := policyengine.CanonicalizeJSONBytes(payload)
	if err != nil {
		return nil, fmt.Errorf("canonicalize %s payload before validation/hash: %w", definitionKind, err)
	}
	return canonicalPayload, nil
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
	if err := validateWorkflowProcessSelection(workflow, process); err != nil {
		return err
	}
	if err := validateCompileInputPolicyContextHash(input.PolicyContextHash); err != nil {
		return err
	}
	if err := validateCompileInputProjectContextIdentityDigest(input.ProjectContextIdentityDigest); err != nil {
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
	if strings.TrimSpace(workflow.WorkflowVersion) == "" {
		return fmt.Errorf("workflow_version is required")
	}
	if strings.TrimSpace(workflow.SelectedProcessID) == "" {
		return fmt.Errorf("selected_process_id is required")
	}
	if strings.TrimSpace(workflow.ApprovalProfile) == "" {
		return fmt.Errorf("approval_profile is required")
	}
	if strings.TrimSpace(workflow.AutonomyPosture) == "" {
		return fmt.Errorf("autonomy_posture is required")
	}
	if strings.TrimSpace(process.ProcessID) == "" {
		return fmt.Errorf("process_id is required")
	}
	return nil
}

func validateWorkflowProcessSelection(workflow WorkflowDefinition, process ProcessDefinition) error {
	selected := strings.TrimSpace(workflow.SelectedProcessID)
	processID := strings.TrimSpace(process.ProcessID)
	if selected != processID {
		return fmt.Errorf("workflow selected_process_id %q does not match process_id %q", selected, processID)
	}
	for _, reviewed := range workflow.ReviewedProcessIDs {
		if strings.TrimSpace(reviewed) == processID {
			return nil
		}
	}
	return fmt.Errorf("workflow reviewed_process_ids must include selected process_id %q", processID)
}

func validateCompileInputNonEmptyCollections(workflow WorkflowDefinition, process ProcessDefinition) error {
	if len(workflow.ReviewedProcessIDs) == 0 || len(process.GateDefinitions) == 0 || len(process.ExecutorBindings) == 0 {
		return fmt.Errorf("workflow reviewed_process_ids, process gate_definitions, and process executor_bindings must be non-empty")
	}
	return nil
}
