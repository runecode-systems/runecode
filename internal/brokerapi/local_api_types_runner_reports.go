package brokerapi

type RunnerCheckpointReport struct {
	SchemaID               string         `json:"schema_id"`
	SchemaVersion          string         `json:"schema_version"`
	LifecycleState         string         `json:"lifecycle_state"`
	CheckpointCode         string         `json:"checkpoint_code"`
	OccurredAt             string         `json:"occurred_at"`
	IdempotencyKey         string         `json:"idempotency_key"`
	PlanCheckpointCode     string         `json:"plan_checkpoint_code,omitempty"`
	PlanOrderIndex         int            `json:"plan_order_index,omitempty"`
	GateEvidenceRef        string         `json:"gate_evidence_ref,omitempty"`
	GateID                 string         `json:"gate_id,omitempty"`
	GateKind               string         `json:"gate_kind,omitempty"`
	GateVersion            string         `json:"gate_version,omitempty"`
	GateLifecycleState     string         `json:"gate_lifecycle_state,omitempty"`
	StageID                string         `json:"stage_id,omitempty"`
	StepID                 string         `json:"step_id,omitempty"`
	RoleInstanceID         string         `json:"role_instance_id,omitempty"`
	StageAttemptID         string         `json:"stage_attempt_id,omitempty"`
	StepAttemptID          string         `json:"step_attempt_id,omitempty"`
	GateAttemptID          string         `json:"gate_attempt_id,omitempty"`
	NormalizedInputDigests []string       `json:"normalized_input_digests,omitempty"`
	PendingApprovalCount   int            `json:"pending_approval_count,omitempty"`
	Details                map[string]any `json:"details,omitempty"`
}

type RunnerResultReport struct {
	SchemaID                  string         `json:"schema_id"`
	SchemaVersion             string         `json:"schema_version"`
	LifecycleState            string         `json:"lifecycle_state"`
	ResultCode                string         `json:"result_code"`
	OccurredAt                string         `json:"occurred_at"`
	IdempotencyKey            string         `json:"idempotency_key"`
	PlanCheckpointCode        string         `json:"plan_checkpoint_code,omitempty"`
	PlanOrderIndex            int            `json:"plan_order_index,omitempty"`
	GateEvidence              *GateEvidence  `json:"gate_evidence,omitempty"`
	GateEvidenceRef           string         `json:"gate_evidence_ref,omitempty"`
	GateID                    string         `json:"gate_id,omitempty"`
	GateKind                  string         `json:"gate_kind,omitempty"`
	GateVersion               string         `json:"gate_version,omitempty"`
	GateLifecycleState        string         `json:"gate_lifecycle_state,omitempty"`
	StageID                   string         `json:"stage_id,omitempty"`
	StepID                    string         `json:"step_id,omitempty"`
	RoleInstanceID            string         `json:"role_instance_id,omitempty"`
	StageAttemptID            string         `json:"stage_attempt_id,omitempty"`
	StepAttemptID             string         `json:"step_attempt_id,omitempty"`
	GateAttemptID             string         `json:"gate_attempt_id,omitempty"`
	NormalizedInputDigests    []string       `json:"normalized_input_digests,omitempty"`
	FailureReasonCode         string         `json:"failure_reason_code,omitempty"`
	OverriddenFailedResultRef string         `json:"overridden_failed_result_ref,omitempty"`
	OverrideActionRequestHash string         `json:"override_action_request_hash,omitempty"`
	OverridePolicyDecisionRef string         `json:"override_policy_decision_ref,omitempty"`
	Details                   map[string]any `json:"details,omitempty"`
}

type GateEvidence struct {
	SchemaID                  string         `json:"schema_id"`
	SchemaVersion             string         `json:"schema_version"`
	GateID                    string         `json:"gate_id"`
	GateKind                  string         `json:"gate_kind"`
	GateVersion               string         `json:"gate_version"`
	PlanCheckpointCode        string         `json:"plan_checkpoint_code,omitempty"`
	PlanOrderIndex            int            `json:"plan_order_index,omitempty"`
	RunID                     string         `json:"run_id"`
	StageID                   string         `json:"stage_id,omitempty"`
	StepID                    string         `json:"step_id,omitempty"`
	RoleInstanceID            string         `json:"role_instance_id,omitempty"`
	GateAttemptID             string         `json:"gate_attempt_id"`
	StartedAt                 string         `json:"started_at"`
	FinishedAt                string         `json:"finished_at"`
	NormalizedInputDigests    []string       `json:"normalized_input_digests,omitempty"`
	Runtime                   map[string]any `json:"runtime"`
	Outcome                   map[string]any `json:"outcome"`
	OutputArtifactDigests     []string       `json:"output_artifact_digests,omitempty"`
	PolicyDecisionRefs        []string       `json:"policy_decision_refs,omitempty"`
	OverriddenFailedResultRef string         `json:"overridden_failed_result_ref,omitempty"`
	OverrideActionRequestHash string         `json:"override_action_request_hash,omitempty"`
	OverridePolicyDecisionRef string         `json:"override_policy_decision_ref,omitempty"`
	FailureReasonCode         string         `json:"failure_reason_code,omitempty"`
}

type RunnerCheckpointReportRequest struct {
	SchemaID      string                 `json:"schema_id"`
	SchemaVersion string                 `json:"schema_version"`
	RequestID     string                 `json:"request_id"`
	RunID         string                 `json:"run_id"`
	Report        RunnerCheckpointReport `json:"report"`
}

type RunnerCheckpointReportResponse struct {
	SchemaID                string `json:"schema_id"`
	SchemaVersion           string `json:"schema_version"`
	RequestID               string `json:"request_id"`
	RunID                   string `json:"run_id"`
	Accepted                bool   `json:"accepted"`
	CanonicalLifecycleState string `json:"canonical_lifecycle_state"`
	AcceptedAt              string `json:"accepted_at"`
	IdempotencyKey          string `json:"idempotency_key"`
}

type RunnerResultReportRequest struct {
	SchemaID      string             `json:"schema_id"`
	SchemaVersion string             `json:"schema_version"`
	RequestID     string             `json:"request_id"`
	RunID         string             `json:"run_id"`
	Report        RunnerResultReport `json:"report"`
}

type RunnerResultReportResponse struct {
	SchemaID                string `json:"schema_id"`
	SchemaVersion           string `json:"schema_version"`
	RequestID               string `json:"request_id"`
	RunID                   string `json:"run_id"`
	Accepted                bool   `json:"accepted"`
	CanonicalLifecycleState string `json:"canonical_lifecycle_state"`
	AcceptedAt              string `json:"accepted_at"`
	IdempotencyKey          string `json:"idempotency_key"`
}
