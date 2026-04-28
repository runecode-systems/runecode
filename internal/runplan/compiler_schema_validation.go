package runplan

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

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
