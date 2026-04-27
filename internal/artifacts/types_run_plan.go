package artifacts

import "time"

type RunPlanDependencyCacheHandoffRecord struct {
	RequestDigest string `json:"request_digest"`
	ConsumerRole  string `json:"consumer_role"`
	Required      bool   `json:"required"`
}

type RunPlanGateEntryRecord struct {
	EntryID                 string                                `json:"entry_id"`
	EntryKind               string                                `json:"entry_kind"`
	PlanCheckpointCode      string                                `json:"plan_checkpoint_code"`
	PlanOrderIndex          int                                   `json:"plan_order_index"`
	GateID                  string                                `json:"gate_id"`
	GateKind                string                                `json:"gate_kind"`
	GateVersion             string                                `json:"gate_version"`
	StageID                 string                                `json:"stage_id"`
	StepID                  string                                `json:"step_id"`
	RoleInstanceID          string                                `json:"role_instance_id"`
	MaxAttempts             int                                   `json:"max_attempts"`
	ExpectedInputDigests    []string                              `json:"expected_input_digests,omitempty"`
	DependencyCacheHandoffs []RunPlanDependencyCacheHandoffRecord `json:"dependency_cache_handoffs,omitempty"`
}

type RunPlanAuthorityRecord struct {
	RunID                        string                   `json:"run_id"`
	PlanID                       string                   `json:"plan_id"`
	SupersedesPlanID             string                   `json:"supersedes_plan_id,omitempty"`
	RunPlanDigest                string                   `json:"run_plan_digest"`
	WorkflowDefinitionHash       string                   `json:"workflow_definition_hash"`
	ProcessDefinitionHash        string                   `json:"process_definition_hash"`
	PolicyContextHash            string                   `json:"policy_context_hash"`
	ProjectContextIdentityDigest string                   `json:"project_context_identity_digest,omitempty"`
	CompiledAt                   time.Time                `json:"compiled_at"`
	RecordedAt                   time.Time                `json:"recorded_at"`
	Entries                      []RunPlanGateEntryRecord `json:"entries,omitempty"`
}

type RunPlanCompilationRecord struct {
	RunID                        string    `json:"run_id"`
	PlanID                       string    `json:"plan_id"`
	RunPlanDigest                string    `json:"run_plan_digest"`
	SupersedesPlanID             string    `json:"supersedes_plan_id,omitempty"`
	WorkflowDefinitionRef        string    `json:"workflow_definition_ref"`
	ProcessDefinitionRef         string    `json:"process_definition_ref"`
	WorkflowDefinitionHash       string    `json:"workflow_definition_hash"`
	ProcessDefinitionHash        string    `json:"process_definition_hash"`
	PolicyContextHash            string    `json:"policy_context_hash"`
	ProjectContextIdentityDigest string    `json:"project_context_identity_digest,omitempty"`
	BindingDigest                string    `json:"binding_digest"`
	RecordDigest                 string    `json:"record_digest"`
	CompiledAt                   time.Time `json:"compiled_at"`
	RecordedAt                   time.Time `json:"recorded_at"`
}
