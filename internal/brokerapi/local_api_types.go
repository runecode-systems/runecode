package brokerapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	runSummarySchemaPath                   = "objects/RunSummary.schema.json"
	runDetailSchemaPath                    = "objects/RunDetail.schema.json"
	runStageSummarySchemaPath              = "objects/RunStageSummary.schema.json"
	runRoleSummarySchemaPath               = "objects/RunRoleSummary.schema.json"
	runCoordinationSummarySchemaPath       = "objects/RunCoordinationSummary.schema.json"
	approvalBoundScopeSchemaPath           = "objects/ApprovalBoundScope.schema.json"
	approvalSummarySchemaPath              = "objects/ApprovalSummary.schema.json"
	artifactSummarySchemaPath              = "objects/ArtifactSummary.schema.json"
	brokerReadinessSchemaPath              = "objects/BrokerReadiness.schema.json"
	brokerVersionInfoSchemaPath            = "objects/BrokerVersionInfo.schema.json"
	runListRequestSchemaPath               = "objects/RunListRequest.schema.json"
	runListResponseSchemaPath              = "objects/RunListResponse.schema.json"
	runGetRequestSchemaPath                = "objects/RunGetRequest.schema.json"
	runGetResponseSchemaPath               = "objects/RunGetResponse.schema.json"
	approvalListRequestSchemaPath          = "objects/ApprovalListRequest.schema.json"
	approvalListResponseSchemaPath         = "objects/ApprovalListResponse.schema.json"
	approvalGetRequestSchemaPath           = "objects/ApprovalGetRequest.schema.json"
	approvalGetResponseSchemaPath          = "objects/ApprovalGetResponse.schema.json"
	approvalResolveRequestSchemaPath       = "objects/ApprovalResolveRequest.schema.json"
	approvalResolveResponseSchemaPath      = "objects/ApprovalResolveResponse.schema.json"
	artifactListRequestSchemaPath          = "objects/ArtifactListRequest.schema.json"
	artifactListResponseSchemaPath         = "objects/ArtifactListResponse.schema.json"
	artifactHeadRequestSchemaPath          = "objects/ArtifactHeadRequest.schema.json"
	artifactHeadResponseSchemaPath         = "objects/ArtifactHeadResponse.schema.json"
	artifactReadRequestSchemaPath          = "objects/ArtifactReadRequest.schema.json"
	artifactStreamEventSchemaPath          = "objects/ArtifactStreamEvent.schema.json"
	logStreamEventSchemaPath               = "objects/LogStreamEvent.schema.json"
	auditTimelineRequestSchemaPath         = "objects/AuditTimelineRequest.schema.json"
	auditTimelineResponseSchemaPath        = "objects/AuditTimelineResponse.schema.json"
	auditVerificationGetRequestSchemaPath  = "objects/AuditVerificationGetRequest.schema.json"
	auditVerificationGetResponseSchemaPath = "objects/AuditVerificationGetResponse.schema.json"
	logStreamRequestSchemaPath             = "objects/LogStreamRequest.schema.json"
	readinessGetRequestSchemaPath          = "objects/ReadinessGetRequest.schema.json"
	readinessGetResponseSchemaPath         = "objects/ReadinessGetResponse.schema.json"
	versionInfoGetRequestSchemaPath        = "objects/VersionInfoGetRequest.schema.json"
	versionInfoGetResponseSchemaPath       = "objects/VersionInfoGetResponse.schema.json"
)

type RunSummary struct {
	SchemaID               string `json:"schema_id"`
	SchemaVersion          string `json:"schema_version"`
	RunID                  string `json:"run_id"`
	WorkspaceID            string `json:"workspace_id"`
	WorkflowKind           string `json:"workflow_kind"`
	WorkflowDefinitionHash string `json:"workflow_definition_hash"`
	CreatedAt              string `json:"created_at"`
	StartedAt              string `json:"started_at,omitempty"`
	UpdatedAt              string `json:"updated_at"`
	FinishedAt             string `json:"finished_at,omitempty"`
	LifecycleState         string `json:"lifecycle_state"`
	CurrentStageID         string `json:"current_stage_id,omitempty"`
	PendingApprovalCount   int    `json:"pending_approval_count"`
	ApprovalProfile        string `json:"approval_profile"`
	BackendKind            string `json:"backend_kind"`
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

type ArtifactSummary struct {
	SchemaID             string                      `json:"schema_id"`
	SchemaVersion        string                      `json:"schema_version"`
	Reference            artifacts.ArtifactReference `json:"reference"`
	CreatedAt            string                      `json:"created_at"`
	CreatedByRole        string                      `json:"created_by_role"`
	RunID                string                      `json:"run_id,omitempty"`
	StageID              string                      `json:"stage_id,omitempty"`
	StepID               string                      `json:"step_id,omitempty"`
	ApprovalOfDigest     string                      `json:"approval_of_digest,omitempty"`
	ApprovalDecisionHash string                      `json:"approval_decision_hash,omitempty"`
}

type BrokerReadiness struct {
	SchemaID                  string `json:"schema_id"`
	SchemaVersion             string `json:"schema_version"`
	Ready                     bool   `json:"ready"`
	LocalOnly                 bool   `json:"local_only"`
	ConsumptionChannel        string `json:"consumption_channel"`
	RecoveryComplete          bool   `json:"recovery_complete"`
	AppendPositionStable      bool   `json:"append_position_stable"`
	CurrentSegmentWritable    bool   `json:"current_segment_writable"`
	VerifierMaterialAvailable bool   `json:"verifier_material_available"`
	DerivedIndexCaughtUp      bool   `json:"derived_index_caught_up"`
}

type BrokerVersionInfo struct {
	SchemaID                    string   `json:"schema_id"`
	SchemaVersion               string   `json:"schema_version"`
	ProductVersion              string   `json:"product_version"`
	BuildRevision               string   `json:"build_revision"`
	BuildTime                   string   `json:"build_time"`
	ProtocolBundleVersion       string   `json:"protocol_bundle_version"`
	ProtocolBundleManifestHash  string   `json:"protocol_bundle_manifest_hash"`
	APIFamily                   string   `json:"api_family"`
	APIVersion                  string   `json:"api_version"`
	SupportedTransportEncodings []string `json:"supported_transport_encodings"`
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

type LocalArtifactListRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Order         string `json:"order,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	StepID        string `json:"step_id,omitempty"`
	DataClass     string `json:"data_class,omitempty"`
}

type LocalArtifactListResponse struct {
	SchemaID      string            `json:"schema_id"`
	SchemaVersion string            `json:"schema_version"`
	RequestID     string            `json:"request_id"`
	Order         string            `json:"order"`
	Artifacts     []ArtifactSummary `json:"artifacts"`
	NextCursor    string            `json:"next_cursor,omitempty"`
}

type LocalArtifactHeadRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Digest        string `json:"digest"`
}

type LocalArtifactHeadResponse struct {
	SchemaID      string          `json:"schema_id"`
	SchemaVersion string          `json:"schema_version"`
	RequestID     string          `json:"request_id"`
	Artifact      ArtifactSummary `json:"artifact"`
}

type ArtifactReadRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Digest        string `json:"digest"`
	ProducerRole  string `json:"producer_role"`
	ConsumerRole  string `json:"consumer_role"`
	ManifestOptIn bool   `json:"manifest_opt_in,omitempty"`
	DataClass     string `json:"data_class,omitempty"`
	RangeStart    *int64 `json:"range_start,omitempty"`
	RangeEnd      *int64 `json:"range_end,omitempty"`
	StreamID      string `json:"stream_id,omitempty"`
	ChunkBytes    int    `json:"chunk_bytes,omitempty"`
}

type ArtifactReadHandle struct {
	RequestID  string
	Digest     string
	DataClass  artifacts.DataClass
	StreamID   string
	ChunkBytes int
	Reader     io.ReadCloser
}

type ArtifactStreamEvent struct {
	SchemaID       string         `json:"schema_id"`
	SchemaVersion  string         `json:"schema_version"`
	StreamID       string         `json:"stream_id"`
	RequestID      string         `json:"request_id"`
	Seq            int64          `json:"seq"`
	EventType      string         `json:"event_type"`
	Digest         string         `json:"digest,omitempty"`
	DataClass      string         `json:"data_class,omitempty"`
	ChunkBase64    string         `json:"chunk_base64,omitempty"`
	ChunkBytes     int            `json:"chunk_bytes,omitempty"`
	EOF            bool           `json:"eof,omitempty"`
	Terminal       bool           `json:"terminal,omitempty"`
	TerminalStatus string         `json:"terminal_status,omitempty"`
	Error          *ProtocolError `json:"error,omitempty"`
}

type LogStreamEvent struct {
	SchemaID       string         `json:"schema_id"`
	SchemaVersion  string         `json:"schema_version"`
	StreamID       string         `json:"stream_id"`
	RequestID      string         `json:"request_id"`
	Seq            int64          `json:"seq"`
	EventType      string         `json:"event_type"`
	RunID          string         `json:"run_id,omitempty"`
	RoleInstanceID string         `json:"role_instance_id,omitempty"`
	Cursor         string         `json:"cursor,omitempty"`
	Timestamp      string         `json:"timestamp,omitempty"`
	Level          string         `json:"level,omitempty"`
	Message        string         `json:"message,omitempty"`
	Terminal       bool           `json:"terminal,omitempty"`
	TerminalStatus string         `json:"terminal_status,omitempty"`
	Error          *ProtocolError `json:"error,omitempty"`
}

type logStreamRecord struct {
	RunID          string
	RoleInstanceID string
	Cursor         string
	Timestamp      string
	Level          string
	Message        string
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

type LogStreamRequest struct {
	SchemaID       string `json:"schema_id"`
	SchemaVersion  string `json:"schema_version"`
	RequestID      string `json:"request_id"`
	StreamID       string `json:"stream_id"`
	RunID          string `json:"run_id,omitempty"`
	RoleInstanceID string `json:"role_instance_id,omitempty"`
	StartCursor    string `json:"start_cursor,omitempty"`
	Follow         bool   `json:"follow"`
	IncludeBacklog bool   `json:"include_backlog"`
}

type ReadinessGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ReadinessGetResponse struct {
	SchemaID      string          `json:"schema_id"`
	SchemaVersion string          `json:"schema_version"`
	RequestID     string          `json:"request_id"`
	Readiness     BrokerReadiness `json:"readiness"`
}

type VersionInfoGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type VersionInfoGetResponse struct {
	SchemaID      string            `json:"schema_id"`
	SchemaVersion string            `json:"schema_version"`
	RequestID     string            `json:"request_id"`
	VersionInfo   BrokerVersionInfo `json:"version_info"`
}

type pageCursor struct {
	Offset int `json:"offset"`
}

func decodeCursor(raw string) (pageCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return pageCursor{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return pageCursor{}, fmt.Errorf("decode cursor: %w", err)
	}
	var c pageCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return pageCursor{}, fmt.Errorf("decode cursor payload: %w", err)
	}
	if c.Offset < 0 {
		return pageCursor{}, fmt.Errorf("cursor offset must be >= 0")
	}
	return c, nil
}

func encodeCursor(c pageCursor) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func normalizeLimit(limit int, fallback int, max int) int {
	if limit <= 0 {
		limit = fallback
	}
	if limit > max {
		limit = max
	}
	return limit
}

func paginate[T any](items []T, cursor string, limit int) ([]T, string, error) {
	c, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	if c.Offset >= len(items) {
		return []T{}, "", nil
	}
	end := c.Offset + limit
	if end > len(items) {
		end = len(items)
	}
	page := items[c.Offset:end]
	if end == len(items) {
		return page, "", nil
	}
	next, err := encodeCursor(pageCursor{Offset: end})
	if err != nil {
		return nil, "", err
	}
	return page, next, nil
}

func toArtifactSummary(record artifacts.ArtifactRecord) ArtifactSummary {
	return ArtifactSummary{
		SchemaID:             "runecode.protocol.v0.ArtifactSummary",
		SchemaVersion:        "0.1.0",
		Reference:            record.Reference,
		CreatedAt:            record.CreatedAt.UTC().Format(time.RFC3339),
		CreatedByRole:        record.CreatedByRole,
		RunID:                record.RunID,
		StepID:               record.StepID,
		ApprovalOfDigest:     record.ApprovalOfDigest,
		ApprovalDecisionHash: record.ApprovalDecisionHash,
	}
}

func sortArtifactSummariesNewestFirst(items []ArtifactSummary) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].Reference.Digest > items[j].Reference.Digest
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
}
