package brokerapi

import (
	"context"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type ApprovalBoundScope struct {
	SchemaID           string `json:"schema_id"`
	SchemaVersion      string `json:"schema_version"`
	WorkspaceID        string `json:"workspace_id,omitempty"`
	InstanceID         string `json:"instance_id,omitempty"`
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
	ScopeDigest            string             `json:"scope_digest,omitempty"`
	ArtifactSetDigest      string             `json:"artifact_set_digest,omitempty"`
	DiffDigest             string             `json:"diff_digest,omitempty"`
	SummaryPreviewDigest   string             `json:"summary_preview_digest,omitempty"`
	ConsumedActionHash     string             `json:"consumed_action_hash,omitempty"`
	ConsumedArtifactDigest string             `json:"consumed_artifact_digest,omitempty"`
	ConsumptionLinkDigest  string             `json:"consumption_link_digest,omitempty"`
}

type ApprovalDetail struct {
	SchemaID                string                           `json:"schema_id"`
	SchemaVersion           string                           `json:"schema_version"`
	ApprovalID              string                           `json:"approval_id"`
	PolicyReasonCode        string                           `json:"policy_reason_code,omitempty"`
	LifecycleDetail         ApprovalLifecycleDetail          `json:"lifecycle_detail"`
	BindingKind             string                           `json:"binding_kind"`
	BoundActionHash         string                           `json:"bound_action_hash,omitempty"`
	BoundStageSummaryHash   string                           `json:"bound_stage_summary_hash,omitempty"`
	BackendPostureSelection *ApprovalBackendPostureSelection `json:"backend_posture_selection,omitempty"`
	WhatChangesIfApproved   ApprovalWhatChangesIfApproved    `json:"what_changes_if_approved"`
	BlockedWorkScope        ApprovalBlockedWorkScope         `json:"blocked_work_scope"`
	BoundIdentity           ApprovalBoundIdentity            `json:"bound_identity"`
}

type ApprovalBackendPostureSelection struct {
	SchemaID                     string `json:"schema_id"`
	SchemaVersion                string `json:"schema_version"`
	TargetInstanceID             string `json:"target_instance_id"`
	TargetBackendKind            string `json:"target_backend_kind"`
	SelectionMode                string `json:"selection_mode"`
	ChangeKind                   string `json:"change_kind"`
	RequestedPosture             string `json:"requested_posture"`
	AssuranceChangeKind          string `json:"assurance_change_kind"`
	OptInKind                    string `json:"opt_in_kind"`
	ReducedAssuranceAcknowledged bool   `json:"reduced_assurance_acknowledged"`
}

type ApprovalLifecycleDetail struct {
	SchemaID               string `json:"schema_id"`
	SchemaVersion          string `json:"schema_version"`
	LifecycleState         string `json:"lifecycle_state"`
	LifecycleReasonCode    string `json:"lifecycle_reason_code"`
	Stale                  bool   `json:"stale"`
	StaleReasonCode        string `json:"stale_reason_code,omitempty"`
	SupersededByApprovalID string `json:"superseded_by_approval_id,omitempty"`
	ExpiresAt              string `json:"expires_at,omitempty"`
	DecidedAt              string `json:"decided_at,omitempty"`
	ConsumedAt             string `json:"consumed_at,omitempty"`
}

type ApprovalWhatChangesIfApproved struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	Summary       string `json:"summary"`
	EffectKind    string `json:"effect_kind"`
}

type ApprovalBlockedWorkScope struct {
	SchemaID       string `json:"schema_id"`
	SchemaVersion  string `json:"schema_version"`
	ScopeKind      string `json:"scope_kind"`
	WorkspaceID    string `json:"workspace_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	StageID        string `json:"stage_id,omitempty"`
	StepID         string `json:"step_id,omitempty"`
	RoleInstanceID string `json:"role_instance_id,omitempty"`
	ActionKind     string `json:"action_kind"`
}

type ApprovalBoundIdentity struct {
	SchemaID                   string                         `json:"schema_id"`
	SchemaVersion              string                         `json:"schema_version"`
	ApprovalRequestDigest      string                         `json:"approval_request_digest"`
	ApprovalDecisionDigest     string                         `json:"approval_decision_digest,omitempty"`
	PolicyDecisionHash         string                         `json:"policy_decision_hash,omitempty"`
	ManifestHash               string                         `json:"manifest_hash"`
	RelevantArtifactHashes     []string                       `json:"relevant_artifact_hashes,omitempty"`
	BindingKind                string                         `json:"binding_kind"`
	BoundActionHash            string                         `json:"bound_action_hash,omitempty"`
	BoundStageSummaryHash      string                         `json:"bound_stage_summary_hash,omitempty"`
	DecisionApprover           *trustpolicy.PrincipalIdentity `json:"decision_approver,omitempty"`
	DecisionVerifierKeyID      string                         `json:"decision_verifier_key_id,omitempty"`
	DecisionVerifierKeyIDValue string                         `json:"decision_verifier_key_id_value,omitempty"`
	ScopeDigest                string                         `json:"scope_digest,omitempty"`
	ArtifactSetDigest          string                         `json:"artifact_set_digest,omitempty"`
	DiffDigest                 string                         `json:"diff_digest,omitempty"`
	SummaryPreviewDigest       string                         `json:"summary_preview_digest,omitempty"`
	ConsumedActionHash         string                         `json:"consumed_action_hash,omitempty"`
	ConsumedArtifactDigest     string                         `json:"consumed_artifact_digest,omitempty"`
	ConsumptionLinkDigest      string                         `json:"consumption_link_digest,omitempty"`
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
	ApprovalDetail         ApprovalDetail                    `json:"approval_detail"`
	SignedApprovalRequest  *trustpolicy.SignedObjectEnvelope `json:"signed_approval_request,omitempty"`
	SignedApprovalDecision *trustpolicy.SignedObjectEnvelope `json:"signed_approval_decision,omitempty"`
}

type ApprovalResolveRequest struct {
	SchemaID               string                           `json:"schema_id"`
	SchemaVersion          string                           `json:"schema_version"`
	RequestID              string                           `json:"request_id"`
	ApprovalID             string                           `json:"approval_id,omitempty"`
	BoundScope             ApprovalBoundScope               `json:"bound_scope"`
	ResolutionDetails      ApprovalResolveDetails           `json:"resolution_details"`
	SignedApprovalRequest  trustpolicy.SignedObjectEnvelope `json:"signed_approval_request"`
	SignedApprovalDecision trustpolicy.SignedObjectEnvelope `json:"signed_approval_decision"`

	// Legacy internal fields retained for compatibility with existing tests.
	UnapprovedDigest      string `json:"-"`
	Approver              string `json:"-"`
	RepoPath              string `json:"-"`
	Commit                string `json:"-"`
	ExtractorToolVersion  string `json:"-"`
	FullContentVisible    bool   `json:"-"`
	ExplicitViewFull      bool   `json:"-"`
	BulkRequest           bool   `json:"-"`
	BulkApprovalConfirmed bool   `json:"-"`
}

type ApprovalResolveDetails struct {
	SchemaID                string                                        `json:"schema_id"`
	SchemaVersion           string                                        `json:"schema_version"`
	Promotion               *ApprovalResolvePromotionDetails              `json:"promotion,omitempty"`
	BackendPostureSelection *ApprovalResolveBackendPostureSelectionDetail `json:"backend_posture_selection,omitempty"`
}

type ApprovalResolvePromotionDetails struct {
	SchemaID              string `json:"schema_id"`
	SchemaVersion         string `json:"schema_version"`
	UnapprovedDigest      string `json:"unapproved_digest"`
	Approver              string `json:"approver"`
	RepoPath              string `json:"repo_path"`
	Commit                string `json:"commit"`
	ExtractorToolVersion  string `json:"extractor_tool_version"`
	FullContentVisible    bool   `json:"full_content_visible"`
	ExplicitViewFull      bool   `json:"explicit_view_full"`
	BulkRequest           bool   `json:"bulk_request"`
	BulkApprovalConfirmed bool   `json:"bulk_approval_confirmed"`
}

type ApprovalResolveBackendPostureSelectionDetail struct {
	SchemaID          string `json:"schema_id"`
	SchemaVersion     string `json:"schema_version"`
	TargetInstanceID  string `json:"target_instance_id"`
	TargetBackendKind string `json:"target_backend_kind"`
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

type ApprovalWatchRequest struct {
	SchemaID        string             `json:"schema_id"`
	SchemaVersion   string             `json:"schema_version"`
	RequestID       string             `json:"request_id"`
	StreamID        string             `json:"stream_id"`
	ApprovalID      string             `json:"approval_id,omitempty"`
	RunID           string             `json:"run_id,omitempty"`
	WorkspaceID     string             `json:"workspace_id,omitempty"`
	Status          string             `json:"status,omitempty"`
	Follow          bool               `json:"follow"`
	IncludeSnapshot bool               `json:"include_snapshot"`
	RequestCtx      context.Context    `json:"-"`
	Cancel          context.CancelFunc `json:"-"`
	Release         func()             `json:"-"`
}

type ApprovalWatchEvent struct {
	SchemaID       string           `json:"schema_id"`
	SchemaVersion  string           `json:"schema_version"`
	StreamID       string           `json:"stream_id"`
	RequestID      string           `json:"request_id"`
	Seq            int64            `json:"seq"`
	EventType      string           `json:"event_type"`
	Approval       *ApprovalSummary `json:"approval,omitempty"`
	Terminal       bool             `json:"terminal,omitempty"`
	TerminalStatus string           `json:"terminal_status,omitempty"`
	Error          *ProtocolError   `json:"error,omitempty"`
}
