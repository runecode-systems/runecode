package artifacts

import "time"

type RunnerCheckpointAdvisory struct {
	LifecycleState   string         `json:"lifecycle_state"`
	CheckpointCode   string         `json:"checkpoint_code"`
	OccurredAt       time.Time      `json:"occurred_at"`
	IdempotencyKey   string         `json:"idempotency_key"`
	PlanCheckpoint   string         `json:"plan_checkpoint_code,omitempty"`
	PlanOrderIndex   int            `json:"plan_order_index,omitempty"`
	GateID           string         `json:"gate_id,omitempty"`
	GateKind         string         `json:"gate_kind,omitempty"`
	GateVersion      string         `json:"gate_version,omitempty"`
	GateState        string         `json:"gate_lifecycle_state,omitempty"`
	StageID          string         `json:"stage_id,omitempty"`
	StepID           string         `json:"step_id,omitempty"`
	RoleInstanceID   string         `json:"role_instance_id,omitempty"`
	StageAttemptID   string         `json:"stage_attempt_id,omitempty"`
	StepAttemptID    string         `json:"step_attempt_id,omitempty"`
	GateAttemptID    string         `json:"gate_attempt_id,omitempty"`
	GateEvidenceRef  string         `json:"gate_evidence_ref,omitempty"`
	NormalizedInputs []string       `json:"normalized_input_digests,omitempty"`
	PendingApprovals int            `json:"pending_approval_count,omitempty"`
	Details          map[string]any `json:"details,omitempty"`
}

type RunnerResultAdvisory struct {
	LifecycleState     string         `json:"lifecycle_state"`
	ResultCode         string         `json:"result_code"`
	OccurredAt         time.Time      `json:"occurred_at"`
	IdempotencyKey     string         `json:"idempotency_key"`
	PlanCheckpoint     string         `json:"plan_checkpoint_code,omitempty"`
	PlanOrderIndex     int            `json:"plan_order_index,omitempty"`
	GateID             string         `json:"gate_id,omitempty"`
	GateKind           string         `json:"gate_kind,omitempty"`
	GateVersion        string         `json:"gate_version,omitempty"`
	GateState          string         `json:"gate_lifecycle_state,omitempty"`
	StageID            string         `json:"stage_id,omitempty"`
	StepID             string         `json:"step_id,omitempty"`
	RoleInstanceID     string         `json:"role_instance_id,omitempty"`
	StageAttemptID     string         `json:"stage_attempt_id,omitempty"`
	StepAttemptID      string         `json:"step_attempt_id,omitempty"`
	GateAttemptID      string         `json:"gate_attempt_id,omitempty"`
	NormalizedInputs   []string       `json:"normalized_input_digests,omitempty"`
	FailureReasonCode  string         `json:"failure_reason_code,omitempty"`
	OverrideFailedRef  string         `json:"overridden_failed_result_ref,omitempty"`
	OverrideActionHash string         `json:"override_action_request_hash,omitempty"`
	OverridePolicyRef  string         `json:"override_policy_decision_ref,omitempty"`
	ResultRef          string         `json:"gate_result_ref,omitempty"`
	GateEvidenceRef    string         `json:"gate_evidence_ref,omitempty"`
	Details            map[string]any `json:"details,omitempty"`
}

type RunnerAdvisoryState struct {
	LastCheckpoint *RunnerCheckpointAdvisory `json:"last_checkpoint,omitempty"`
	LastResult     *RunnerResultAdvisory     `json:"last_result,omitempty"`
	Lifecycle      *RunnerLifecycleHint      `json:"lifecycle,omitempty"`
	StepAttempts   map[string]RunnerStepHint `json:"step_attempts,omitempty"`
	GateAttempts   map[string]RunnerGateHint `json:"gate_attempts,omitempty"`
	ApprovalWaits  map[string]RunnerApproval `json:"approval_waits,omitempty"`
}

type RunnerGateHint struct {
	GateAttemptID      string    `json:"gate_attempt_id"`
	RunID              string    `json:"run_id"`
	PlanCheckpoint     string    `json:"plan_checkpoint_code,omitempty"`
	PlanOrderIndex     int       `json:"plan_order_index,omitempty"`
	GateID             string    `json:"gate_id"`
	GateKind           string    `json:"gate_kind"`
	GateVersion        string    `json:"gate_version"`
	GateState          string    `json:"gate_lifecycle_state"`
	StageID            string    `json:"stage_id,omitempty"`
	StepID             string    `json:"step_id,omitempty"`
	RoleInstanceID     string    `json:"role_instance_id,omitempty"`
	StageAttemptID     string    `json:"stage_attempt_id,omitempty"`
	StepAttemptID      string    `json:"step_attempt_id,omitempty"`
	GateEvidenceRef    string    `json:"gate_evidence_ref,omitempty"`
	FailureReasonCode  string    `json:"failure_reason_code,omitempty"`
	OverrideFailedRef  string    `json:"overridden_failed_result_ref,omitempty"`
	OverrideActionHash string    `json:"override_action_request_hash,omitempty"`
	OverridePolicyRef  string    `json:"override_policy_decision_ref,omitempty"`
	ResultRef          string    `json:"gate_result_ref,omitempty"`
	ResultCode         string    `json:"result_code,omitempty"`
	Terminal           bool      `json:"terminal"`
	StartedAt          time.Time `json:"started_at,omitempty"`
	FinishedAt         time.Time `json:"finished_at,omitempty"`
	LastUpdatedAt      time.Time `json:"last_updated_at"`
}

type RunnerLifecycleHint struct {
	LifecycleState string    `json:"lifecycle_state"`
	OccurredAt     time.Time `json:"occurred_at"`
	StageID        string    `json:"stage_id,omitempty"`
	StepID         string    `json:"step_id,omitempty"`
	RoleInstanceID string    `json:"role_instance_id,omitempty"`
	StageAttemptID string    `json:"stage_attempt_id,omitempty"`
	StepAttemptID  string    `json:"step_attempt_id,omitempty"`
	GateAttemptID  string    `json:"gate_attempt_id,omitempty"`
}

type RunnerStepHint struct {
	StepAttemptID   string    `json:"step_attempt_id"`
	RunID           string    `json:"run_id"`
	GateID          string    `json:"gate_id,omitempty"`
	GateKind        string    `json:"gate_kind,omitempty"`
	GateVersion     string    `json:"gate_version,omitempty"`
	GateState       string    `json:"gate_lifecycle_state,omitempty"`
	StageID         string    `json:"stage_id,omitempty"`
	StepID          string    `json:"step_id,omitempty"`
	RoleInstanceID  string    `json:"role_instance_id,omitempty"`
	StageAttemptID  string    `json:"stage_attempt_id,omitempty"`
	GateAttemptID   string    `json:"gate_attempt_id,omitempty"`
	GateEvidenceRef string    `json:"gate_evidence_ref,omitempty"`
	CurrentPhase    string    `json:"current_phase,omitempty"`
	PhaseStatus     string    `json:"phase_status,omitempty"`
	Status          string    `json:"status"`
	StartedAt       time.Time `json:"started_at,omitempty"`
	FinishedAt      time.Time `json:"finished_at,omitempty"`
	LastUpdatedAt   time.Time `json:"last_updated_at"`
}

type RunnerApproval struct {
	ApprovalID            string     `json:"approval_id"`
	RunID                 string     `json:"run_id"`
	StageID               string     `json:"stage_id,omitempty"`
	StepID                string     `json:"step_id,omitempty"`
	RoleInstanceID        string     `json:"role_instance_id,omitempty"`
	Status                string     `json:"status"`
	ApprovalType          string     `json:"approval_type"`
	BoundActionHash       string     `json:"bound_action_hash,omitempty"`
	BoundStageSummaryHash string     `json:"bound_stage_summary_hash,omitempty"`
	OccurredAt            time.Time  `json:"occurred_at"`
	ResolvedAt            *time.Time `json:"resolved_at,omitempty"`
	SupersededByApproval  string     `json:"superseded_by_approval,omitempty"`
}

type RunnerDurableSnapshot struct {
	Family        string                         `json:"family"`
	SchemaVersion int                            `json:"schema_version"`
	LastSequence  int64                          `json:"last_sequence"`
	Runs          map[string]RunnerAdvisoryState `json:"runs"`
	Idempotency   map[string]int64               `json:"idempotency"`
}

type RunnerDurableJournalRecord struct {
	Family         string                    `json:"family"`
	SchemaVersion  int                       `json:"schema_version"`
	Sequence       int64                     `json:"sequence"`
	RecordType     string                    `json:"record_type"`
	RunID          string                    `json:"run_id"`
	IdempotencyKey string                    `json:"idempotency_key"`
	OccurredAt     time.Time                 `json:"occurred_at"`
	Checkpoint     *RunnerCheckpointAdvisory `json:"checkpoint,omitempty"`
	Result         *RunnerResultAdvisory     `json:"result,omitempty"`
	Approval       *RunnerApproval           `json:"approval,omitempty"`
}

type RuntimeAuditEmissionState struct {
	LastIsolateSessionStartedDigest  string `json:"last_isolate_session_started_digest,omitempty"`
	LastIsolateSessionBoundDigest    string `json:"last_isolate_session_bound_digest,omitempty"`
	LastRuntimeLaunchAdmissionDigest string `json:"last_runtime_launch_admission_digest,omitempty"`
	LastRuntimeLaunchDeniedDigest    string `json:"last_runtime_launch_denied_digest,omitempty"`
}
