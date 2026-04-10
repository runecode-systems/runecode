package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

const (
	runSummarySchemaPath              = "objects/RunSummary.schema.json"
	runDetailSchemaPath               = "objects/RunDetail.schema.json"
	runStageSummarySchemaPath         = "objects/RunStageSummary.schema.json"
	runRoleSummarySchemaPath          = "objects/RunRoleSummary.schema.json"
	runCoordinationSummarySchemaPath  = "objects/RunCoordinationSummary.schema.json"
	approvalBoundScopeSchemaPath      = "objects/ApprovalBoundScope.schema.json"
	approvalSummarySchemaPath         = "objects/ApprovalSummary.schema.json"
	runListRequestSchemaPath          = "objects/RunListRequest.schema.json"
	runListResponseSchemaPath         = "objects/RunListResponse.schema.json"
	runGetRequestSchemaPath           = "objects/RunGetRequest.schema.json"
	runGetResponseSchemaPath          = "objects/RunGetResponse.schema.json"
	approvalListRequestSchemaPath     = "objects/ApprovalListRequest.schema.json"
	approvalListResponseSchemaPath    = "objects/ApprovalListResponse.schema.json"
	approvalGetRequestSchemaPath      = "objects/ApprovalGetRequest.schema.json"
	approvalGetResponseSchemaPath     = "objects/ApprovalGetResponse.schema.json"
	approvalResolveRequestSchemaPath  = "objects/ApprovalResolveRequest.schema.json"
	approvalResolveResponseSchemaPath = "objects/ApprovalResolveResponse.schema.json"
)

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

type ApprovalBoundScope struct {
	SchemaID           string `json:"schema_id"`
	SchemaVersion      string `json:"schema_version"`
	WorkspaceID        string `json:"workspace_id,omitempty"`
	RunID              string `json:"run_id,omitempty"`
	StageID            string `json:"stage_id,omitempty"`
	StepID             string `json:"step_id,omitempty"`
	RoleInstanceID     string `json:"role_instance_id,omitempty"`
	ActionKind         string `json:"action_kind"`
	PolicyDecisionHash string `json:"policy_decision_hash,omitempty"`
}

type ApprovalSummary struct {
	SchemaID               string             `json:"schema_id"`
	SchemaVersion          string             `json:"schema_version"`
	ApprovalID             string             `json:"approval_id"`
	Status                 string             `json:"status"`
	RequestedAt            string             `json:"requested_at"`
	ExpiresAt              string             `json:"expires_at,omitempty"`
	DecidedAt              string             `json:"decided_at,omitempty"`
	ConsumedAt             string             `json:"consumed_at,omitempty"`
	ApprovalTriggerCode    string             `json:"approval_trigger_code"`
	ChangesIfApproved      string             `json:"changes_if_approved"`
	ApprovalAssuranceLevel string             `json:"approval_assurance_level"`
	PresenceMode           string             `json:"presence_mode"`
	BoundScope             ApprovalBoundScope `json:"bound_scope"`
	PolicyDecisionHash     string             `json:"policy_decision_hash,omitempty"`
	SupersededByApprovalID string             `json:"superseded_by_approval_id,omitempty"`
	RequestDigest          string             `json:"request_digest,omitempty"`
	DecisionDigest         string             `json:"decision_digest,omitempty"`
}

type ApprovalListRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Order         string `json:"order,omitempty"`
	Status        string `json:"status,omitempty"`
	RunID         string `json:"run_id,omitempty"`
}

type ApprovalListResponse struct {
	SchemaID      string            `json:"schema_id"`
	SchemaVersion string            `json:"schema_version"`
	RequestID     string            `json:"request_id"`
	Order         string            `json:"order"`
	Approvals     []ApprovalSummary `json:"approvals"`
	NextCursor    string            `json:"next_cursor,omitempty"`
}

type ApprovalGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	ApprovalID    string `json:"approval_id"`
}

type ApprovalGetResponse struct {
	SchemaID               string                            `json:"schema_id"`
	SchemaVersion          string                            `json:"schema_version"`
	RequestID              string                            `json:"request_id"`
	Approval               ApprovalSummary                   `json:"approval"`
	SignedApprovalRequest  *trustpolicy.SignedObjectEnvelope `json:"signed_approval_request,omitempty"`
	SignedApprovalDecision *trustpolicy.SignedObjectEnvelope `json:"signed_approval_decision,omitempty"`
}

type ApprovalResolveRequest struct {
	SchemaID               string                           `json:"schema_id"`
	SchemaVersion          string                           `json:"schema_version"`
	RequestID              string                           `json:"request_id"`
	ApprovalID             string                           `json:"approval_id,omitempty"`
	BoundScope             ApprovalBoundScope               `json:"bound_scope"`
	UnapprovedDigest       string                           `json:"unapproved_digest"`
	Approver               string                           `json:"approver"`
	RepoPath               string                           `json:"repo_path"`
	Commit                 string                           `json:"commit"`
	ExtractorToolVersion   string                           `json:"extractor_tool_version"`
	FullContentVisible     bool                             `json:"full_content_visible"`
	ExplicitViewFull       bool                             `json:"explicit_view_full"`
	BulkRequest            bool                             `json:"bulk_request"`
	BulkApprovalConfirmed  bool                             `json:"bulk_approval_confirmed"`
	SignedApprovalRequest  trustpolicy.SignedObjectEnvelope `json:"signed_approval_request"`
	SignedApprovalDecision trustpolicy.SignedObjectEnvelope `json:"signed_approval_decision"`
}

type ApprovalResolveResponse struct {
	SchemaID             string           `json:"schema_id"`
	SchemaVersion        string           `json:"schema_version"`
	RequestID            string           `json:"request_id"`
	ResolutionStatus     string           `json:"resolution_status"`
	ResolutionReasonCode string           `json:"resolution_reason_code,omitempty"`
	Approval             ApprovalSummary  `json:"approval"`
	ApprovedArtifact     *ArtifactSummary `json:"approved_artifact,omitempty"`
}

type AuditTimelineRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Order         string `json:"order,omitempty"`
}

type AuditTimelineResponse struct {
	SchemaID      string                             `json:"schema_id"`
	SchemaVersion string                             `json:"schema_version"`
	RequestID     string                             `json:"request_id"`
	Order         string                             `json:"order"`
	Views         []trustpolicy.AuditOperationalView `json:"views"`
	NextCursor    string                             `json:"next_cursor,omitempty"`
}

type AuditVerificationGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	ViewLimit     int    `json:"view_limit,omitempty"`
}

type AuditVerificationGetResponse struct {
	SchemaID      string                                         `json:"schema_id"`
	SchemaVersion string                                         `json:"schema_version"`
	RequestID     string                                         `json:"request_id"`
	Summary       trustpolicy.DerivedRunAuditVerificationSummary `json:"summary"`
	Report        trustpolicy.AuditVerificationReportPayload     `json:"report"`
	Views         []trustpolicy.AuditOperationalView             `json:"views"`
}
