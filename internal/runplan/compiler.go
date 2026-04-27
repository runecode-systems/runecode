package runplan

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

const (
	workflowDefinitionSchemaID   = "runecode.protocol.v0.WorkflowDefinition"
	workflowDefinitionVersion    = "0.5.0"
	workflowDefinitionSchemaPath = "objects/WorkflowDefinition.schema.json"
	processDefinitionSchemaID    = "runecode.protocol.v0.ProcessDefinition"
	processDefinitionVersion     = "0.4.0"
	processDefinitionSchemaPath  = "objects/ProcessDefinition.schema.json"
	runPlanSchemaID              = "runecode.protocol.v0.RunPlan"
	runPlanVersion               = "0.4.0"
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
	if err := validateCompileInput(input, workflow, process, processHash); err != nil {
		return RunPlan{}, err
	}
	if err := validateBindingsAgainstTrustedRegistry(process.ExecutorBindings, input.ExecutorRegistry); err != nil {
		return RunPlan{}, err
	}
	compiledBindings, gates, roleIDs, dependencyEdges, entries, err := compileProcessPlanShape(process)
	if err != nil {
		return RunPlan{}, err
	}
	plan := newRunPlan(input, workflow, process, workflowHash, processHash, roleIDs, compiledBindings, gates, dependencyEdges, entries)
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

func compileProcessPlanShape(process ProcessDefinition) ([]ExecutorBinding, []GateDefinition, []string, []DependencyEdge, []Entry, error) {
	compiledBindings, err := compileExecutorBindings(process.ExecutorBindings)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	gates, roleIDs, err := compileGateDefinitions(process.GateDefinitions, compiledBindings)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	dependencyEdges, err := compileProcessDependencyEdges(gates, process.DependencyEdges)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	entries, err := compileRunPlanEntries(gates, dependencyEdges)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return compiledBindings, gates, roleIDs, dependencyEdges, entries, nil
}

func newRunPlan(input CompileInput, workflow WorkflowDefinition, process ProcessDefinition, workflowHash string, processHash string, roleIDs []string, compiledBindings []ExecutorBinding, gates []GateDefinition, dependencyEdges []DependencyEdge, entries []Entry) RunPlan {
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
		Entries:                      entries,
	}
}

func validateCompileInput(input CompileInput, workflow WorkflowDefinition, process ProcessDefinition, processHash string) error {
	if err := validateCompileInputRequiredIDs(input, workflow, process); err != nil {
		return err
	}
	if err := validateWorkflowProcessSelection(workflow, process, processHash); err != nil {
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
	if strings.TrimSpace(workflow.SelectedProcessDefinitionHash) == "" {
		return fmt.Errorf("selected_process_definition_hash is required")
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

func validateWorkflowProcessSelection(workflow WorkflowDefinition, process ProcessDefinition, processHash string) error {
	selected := strings.TrimSpace(workflow.SelectedProcessID)
	processID := strings.TrimSpace(process.ProcessID)
	selectedHash := strings.TrimSpace(workflow.SelectedProcessDefinitionHash)
	if err := validateSelectedProcessBinding(selected, processID, selectedHash, processHash); err != nil {
		return err
	}
	reviewedArtifacts, err := collectReviewedProcessArtifacts(workflow.ReviewedProcessArtifacts)
	if err != nil {
		return err
	}
	if reviewedHash, ok := reviewedArtifacts[processID]; ok && reviewedHash == selectedHash {
		return nil
	}
	return fmt.Errorf("workflow reviewed_process_artifacts must include selected process artifact for process_id %q", processID)
}

func validateSelectedProcessBinding(selectedProcessID string, processID string, selectedHash string, processHash string) error {
	if _, err := policyengine.NormalizeHashIdentity(selectedHash); err != nil {
		return fmt.Errorf("selected_process_definition_hash invalid: %w", err)
	}
	if selectedProcessID != processID {
		return fmt.Errorf("workflow selected_process_id %q does not match process_id %q", selectedProcessID, processID)
	}
	if selectedHash != strings.TrimSpace(processHash) {
		return fmt.Errorf("workflow selected_process_definition_hash %q does not match compiled process_definition_hash %q", selectedHash, processHash)
	}
	return nil
}

func collectReviewedProcessArtifacts(reviewedArtifacts []ReviewedProcessArtifact) (map[string]string, error) {
	seenReviewedArtifacts := map[string]string{}
	for _, reviewed := range reviewedArtifacts {
		reviewedProcessID, reviewedHash, err := normalizeReviewedProcessArtifact(reviewed)
		if err != nil {
			return nil, err
		}
		if err := includeReviewedProcessArtifact(seenReviewedArtifacts, reviewedProcessID, reviewedHash); err != nil {
			return nil, err
		}
	}
	return seenReviewedArtifacts, nil
}

func normalizeReviewedProcessArtifact(reviewed ReviewedProcessArtifact) (string, string, error) {
	reviewedProcessID := strings.TrimSpace(reviewed.ProcessID)
	reviewedHash := strings.TrimSpace(reviewed.ProcessDefinitionHash)
	if reviewedHash == "" {
		return reviewedProcessID, reviewedHash, nil
	}
	if _, err := policyengine.NormalizeHashIdentity(reviewedHash); err != nil {
		return "", "", fmt.Errorf("reviewed_process_artifacts.process_definition_hash invalid: %w", err)
	}
	return reviewedProcessID, reviewedHash, nil
}

func includeReviewedProcessArtifact(seenReviewedArtifacts map[string]string, reviewedProcessID, reviewedHash string) error {
	priorHash, ok := seenReviewedArtifacts[reviewedProcessID]
	if !ok {
		seenReviewedArtifacts[reviewedProcessID] = reviewedHash
		return nil
	}
	if priorHash != reviewedHash {
		return fmt.Errorf("workflow reviewed_process_artifacts contains conflicting process_definition_hash values for process_id %q", reviewedProcessID)
	}
	return fmt.Errorf("workflow reviewed_process_artifacts contains duplicate process_id %q", reviewedProcessID)
}

func validateCompileInputNonEmptyCollections(workflow WorkflowDefinition, process ProcessDefinition) error {
	if len(workflow.ReviewedProcessArtifacts) == 0 || len(process.GateDefinitions) == 0 || len(process.ExecutorBindings) == 0 {
		return fmt.Errorf("workflow reviewed_process_artifacts, process gate_definitions, and process executor_bindings must be non-empty")
	}
	return nil
}
