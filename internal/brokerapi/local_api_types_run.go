package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type RunSummary struct {
	SchemaID                string `json:"schema_id"`
	SchemaVersion           string `json:"schema_version"`
	RunID                   string `json:"run_id"`
	WorkspaceID             string `json:"workspace_id"`
	WorkflowKind            string `json:"workflow_kind,omitempty"`
	WorkflowDefinitionHash  string `json:"workflow_definition_hash,omitempty"`
	CreatedAt               string `json:"created_at"`
	StartedAt               string `json:"started_at,omitempty"`
	UpdatedAt               string `json:"updated_at"`
	FinishedAt              string `json:"finished_at,omitempty"`
	LifecycleState          string `json:"lifecycle_state"`
	CurrentStageID          string `json:"current_stage_id,omitempty"`
	PendingApprovalCount    int    `json:"pending_approval_count"`
	ApprovalProfile         string `json:"approval_profile"`
	BackendKind             string `json:"backend_kind"`
	IsolationAssuranceLevel string `json:"isolation_assurance_level"`
	ProvisioningPosture     string `json:"provisioning_posture"`
	// AssuranceLevel is retained as a migration-compatible alias for
	// isolation_assurance_level.
	AssuranceLevel         string `json:"assurance_level"`
	BlockingReasonCode     string `json:"blocking_reason_code,omitempty"`
	AuditIntegrityStatus   string `json:"audit_integrity_status"`
	AuditAnchoringStatus   string `json:"audit_anchoring_status"`
	AuditCurrentlyDegraded bool   `json:"audit_currently_degraded"`
}

type RunStageSummary struct {
	SchemaID             string `json:"schema_id"`
	SchemaVersion        string `json:"schema_version"`
	StageID              string `json:"stage_id"`
	LifecycleState       string `json:"lifecycle_state"`
	StartedAt            string `json:"started_at,omitempty"`
	FinishedAt           string `json:"finished_at,omitempty"`
	PendingApprovalCount int    `json:"pending_approval_count"`
	ArtifactCount        int    `json:"artifact_count"`
}

type RunRoleSummary struct {
	SchemaID        string `json:"schema_id"`
	SchemaVersion   string `json:"schema_version"`
	RoleInstanceID  string `json:"role_instance_id"`
	RoleFamily      string `json:"role_family"`
	RoleKind        string `json:"role_kind"`
	LifecycleState  string `json:"lifecycle_state"`
	ActiveItemCount int    `json:"active_item_count"`
	WaitReasonCode  string `json:"wait_reason_code,omitempty"`
}

type RunCoordinationSummary struct {
	SchemaID         string `json:"schema_id"`
	SchemaVersion    string `json:"schema_version"`
	Blocked          bool   `json:"blocked"`
	WaitReasonCode   string `json:"wait_reason_code,omitempty"`
	LockCount        int    `json:"lock_count"`
	ConflictCount    int    `json:"conflict_count"`
	CoordinationMode string `json:"coordination_mode"`
}

type RunDetail struct {
	SchemaID                 string                                         `json:"schema_id"`
	SchemaVersion            string                                         `json:"schema_version"`
	Summary                  RunSummary                                     `json:"summary"`
	StageSummaries           []RunStageSummary                              `json:"stage_summaries"`
	RoleSummaries            []RunRoleSummary                               `json:"role_summaries"`
	Coordination             RunCoordinationSummary                         `json:"coordination"`
	AuditSummary             trustpolicy.DerivedRunAuditVerificationSummary `json:"audit_summary"`
	ArtifactCountsByClass    map[string]int                                 `json:"artifact_counts_by_class"`
	PendingApprovalIDs       []string                                       `json:"pending_approval_ids"`
	ActiveManifestHashes     []string                                       `json:"active_manifest_hashes"`
	LatestPolicyDecisionRefs []string                                       `json:"latest_policy_decision_refs"`
	AuthoritativeState       map[string]any                                 `json:"authoritative_state"`
	AdvisoryState            map[string]any                                 `json:"advisory_state"`
}

type RunListRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Order         string `json:"order,omitempty"`
}

type RunListResponse struct {
	SchemaID      string       `json:"schema_id"`
	SchemaVersion string       `json:"schema_version"`
	RequestID     string       `json:"request_id"`
	Order         string       `json:"order"`
	Runs          []RunSummary `json:"runs"`
	NextCursor    string       `json:"next_cursor,omitempty"`
}

type RunGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	RunID         string `json:"run_id"`
}

type RunGetResponse struct {
	SchemaID      string    `json:"schema_id"`
	SchemaVersion string    `json:"schema_version"`
	RequestID     string    `json:"request_id"`
	Run           RunDetail `json:"run"`
}
