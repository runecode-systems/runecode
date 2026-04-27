package runplan

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type ExecutorBinding struct {
	BindingID        string   `json:"binding_id"`
	ExecutorID       string   `json:"executor_id"`
	ExecutorClass    string   `json:"executor_class"`
	AllowedRoleKinds []string `json:"allowed_role_kinds"`
	Description      string   `json:"description,omitempty"`
}

type WorkflowDefinition struct {
	SchemaID           string   `json:"schema_id"`
	SchemaVersion      string   `json:"schema_version"`
	WorkflowID         string   `json:"workflow_id"`
	WorkflowVersion    string   `json:"workflow_version"`
	SelectedProcessID  string   `json:"selected_process_id"`
	ReviewedProcessIDs []string `json:"reviewed_process_ids"`
	PolicyBindingID    string   `json:"policy_binding_id,omitempty"`
	ApprovalProfile    string   `json:"approval_profile"`
	AutonomyPosture    string   `json:"autonomy_posture"`
}

type ProcessDefinition struct {
	SchemaID         string            `json:"schema_id"`
	SchemaVersion    string            `json:"schema_version"`
	ProcessID        string            `json:"process_id"`
	ExecutorBindings []ExecutorBinding `json:"executor_bindings"`
	GateDefinitions  []GateDefinition  `json:"gate_definitions"`
	DependencyEdges  []DependencyEdge  `json:"dependency_edges"`
}

type DependencyEdge struct {
	UpstreamStepID   string `json:"upstream_step_id"`
	DownstreamStepID string `json:"downstream_step_id"`
	DependencyKind   string `json:"dependency_kind"`
}

type GateDefinition struct {
	SchemaID                string                   `json:"schema_id"`
	SchemaVersion           string                   `json:"schema_version"`
	Gate                    GateContract             `json:"gate"`
	CheckpointCode          string                   `json:"checkpoint_code"`
	OrderIndex              int                      `json:"order_index"`
	StageID                 string                   `json:"stage_id"`
	StepID                  string                   `json:"step_id"`
	RoleInstanceID          string                   `json:"role_instance_id"`
	ExecutorBindingID       string                   `json:"executor_binding_id"`
	DependencyCacheHandoffs []DependencyCacheHandoff `json:"dependency_cache_handoffs,omitempty"`
}

type DependencyCacheHandoff struct {
	RequestDigest trustpolicy.Digest `json:"request_digest"`
	ConsumerRole  string             `json:"consumer_role"`
	Required      bool               `json:"required"`
}

type GateContract struct {
	SchemaID          string           `json:"schema_id"`
	SchemaVersion     string           `json:"schema_version"`
	GateID            string           `json:"gate_id"`
	GateKind          string           `json:"gate_kind"`
	GateVersion       string           `json:"gate_version"`
	NormalizedInputs  []map[string]any `json:"normalized_inputs"`
	PlanBinding       map[string]any   `json:"plan_binding"`
	RetrySemantics    map[string]any   `json:"retry_semantics"`
	OverrideSemantics map[string]any   `json:"override_semantics"`
}

type RunPlan struct {
	SchemaID                     string            `json:"schema_id"`
	SchemaVersion                string            `json:"schema_version"`
	PlanID                       string            `json:"plan_id"`
	SupersedesPlanID             string            `json:"supersedes_plan_id,omitempty"`
	RunID                        string            `json:"run_id"`
	ProjectContextIdentityDigest string            `json:"project_context_identity_digest,omitempty"`
	WorkflowID                   string            `json:"workflow_id"`
	WorkflowVersion              string            `json:"workflow_version"`
	ProcessID                    string            `json:"process_id"`
	ApprovalProfile              string            `json:"approval_profile"`
	AutonomyPosture              string            `json:"autonomy_posture"`
	PolicyBindingID              string            `json:"policy_binding_id,omitempty"`
	WorkflowDefinitionHash       string            `json:"workflow_definition_hash"`
	ProcessDefinitionHash        string            `json:"process_definition_hash"`
	PolicyContextHash            string            `json:"policy_context_hash"`
	CompiledAt                   string            `json:"compiled_at"`
	RoleInstanceIDs              []string          `json:"role_instance_ids"`
	ExecutorBindings             []ExecutorBinding `json:"executor_bindings"`
	GateDefinitions              []GateDefinition  `json:"gate_definitions"`
	DependencyEdges              []DependencyEdge  `json:"dependency_edges"`
}
