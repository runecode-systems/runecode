package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

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
