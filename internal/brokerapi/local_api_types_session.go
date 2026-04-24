package brokerapi

import "context"

type SessionIdentity struct {
	SchemaID       string `json:"schema_id"`
	SchemaVersion  string `json:"schema_version"`
	SessionID      string `json:"session_id"`
	WorkspaceID    string `json:"workspace_id"`
	CreatedAt      string `json:"created_at"`
	CreatedByRunID string `json:"created_by_run_id,omitempty"`
}

type SessionSummary struct {
	SchemaID              string          `json:"schema_id"`
	SchemaVersion         string          `json:"schema_version"`
	Identity              SessionIdentity `json:"identity"`
	UpdatedAt             string          `json:"updated_at"`
	Status                string          `json:"status"`
	WorkPosture           string          `json:"work_posture"`
	WorkPostureReasonCode string          `json:"work_posture_reason_code,omitempty"`
	LastActivityAt        string          `json:"last_activity_at,omitempty"`
	LastActivityKind      string          `json:"last_activity_kind"`
	LastActivityPreview   string          `json:"last_activity_preview,omitempty"`
	TurnCount             int             `json:"turn_count"`
	LinkedRunCount        int             `json:"linked_run_count"`
	LinkedApprovalCount   int             `json:"linked_approval_count"`
	LinkedArtifactCount   int             `json:"linked_artifact_count"`
	LinkedAuditEventCount int             `json:"linked_audit_event_count"`
	HasIncompleteTurn     bool            `json:"has_incomplete_turn"`
}

type SessionDetail struct {
	SchemaID                 string                  `json:"schema_id"`
	SchemaVersion            string                  `json:"schema_version"`
	Summary                  SessionSummary          `json:"summary"`
	TranscriptTurns          []SessionTranscriptTurn `json:"transcript_turns"`
	CurrentTurnExecution     *SessionTurnExecution   `json:"current_turn_execution,omitempty"`
	LatestTurnExecution      *SessionTurnExecution   `json:"latest_turn_execution,omitempty"`
	PendingTurnExecutions    []SessionTurnExecution  `json:"pending_turn_executions,omitempty"`
	LinkedRunIDs             []string                `json:"linked_run_ids"`
	LinkedApprovalIDs        []string                `json:"linked_approval_ids"`
	LinkedArtifactDigests    []string                `json:"linked_artifact_digests"`
	LinkedAuditRecordDigests []string                `json:"linked_audit_record_digests"`
}

type SessionTurnExecution struct {
	SchemaID                             string   `json:"schema_id"`
	SchemaVersion                        string   `json:"schema_version"`
	TurnID                               string   `json:"turn_id"`
	SessionID                            string   `json:"session_id"`
	ExecutionIndex                       int      `json:"execution_index"`
	OrchestrationScopeID                 string   `json:"orchestration_scope_id,omitempty"`
	DependsOnScopeIDs                    []string `json:"depends_on_scope_ids,omitempty"`
	TriggerID                            string   `json:"trigger_id"`
	TriggerSource                        string   `json:"trigger_source"`
	RequestedOperation                   string   `json:"requested_operation"`
	ExecutionState                       string   `json:"execution_state"`
	WaitKind                             string   `json:"wait_kind,omitempty"`
	WaitState                            string   `json:"wait_state,omitempty"`
	ApprovalProfile                      string   `json:"approval_profile"`
	AutonomyPosture                      string   `json:"autonomy_posture"`
	PrimaryRunID                         string   `json:"primary_run_id,omitempty"`
	PendingApprovalID                    string   `json:"pending_approval_id,omitempty"`
	LinkedRunIDs                         []string `json:"linked_run_ids,omitempty"`
	LinkedApprovalIDs                    []string `json:"linked_approval_ids,omitempty"`
	LinkedArtifactDigests                []string `json:"linked_artifact_digests,omitempty"`
	LinkedAuditRecordDigests             []string `json:"linked_audit_record_digests,omitempty"`
	BoundValidatedProjectSubstrateDigest string   `json:"bound_validated_project_substrate_digest,omitempty"`
	BlockedReasonCode                    string   `json:"blocked_reason_code,omitempty"`
	TerminalOutcome                      string   `json:"terminal_outcome,omitempty"`
	CreatedAt                            string   `json:"created_at"`
	UpdatedAt                            string   `json:"updated_at"`
}

type SessionTranscriptLinks struct {
	SchemaID           string   `json:"schema_id"`
	SchemaVersion      string   `json:"schema_version"`
	RunIDs             []string `json:"run_ids"`
	ApprovalIDs        []string `json:"approval_ids"`
	ArtifactDigests    []string `json:"artifact_digests"`
	AuditRecordDigests []string `json:"audit_record_digests"`
}

type SessionTranscriptMessage struct {
	SchemaID      string                 `json:"schema_id"`
	SchemaVersion string                 `json:"schema_version"`
	MessageID     string                 `json:"message_id"`
	TurnID        string                 `json:"turn_id"`
	SessionID     string                 `json:"session_id"`
	MessageIndex  int                    `json:"message_index"`
	Role          string                 `json:"role"`
	CreatedAt     string                 `json:"created_at"`
	ContentText   string                 `json:"content_text"`
	RelatedLinks  SessionTranscriptLinks `json:"related_links"`
}

type SessionTranscriptTurn struct {
	SchemaID      string                     `json:"schema_id"`
	SchemaVersion string                     `json:"schema_version"`
	TurnID        string                     `json:"turn_id"`
	SessionID     string                     `json:"session_id"`
	TurnIndex     int                        `json:"turn_index"`
	StartedAt     string                     `json:"started_at"`
	CompletedAt   string                     `json:"completed_at,omitempty"`
	Status        string                     `json:"status"`
	Messages      []SessionTranscriptMessage `json:"messages"`
}

type SessionListRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Order         string `json:"order,omitempty"`
}

type SessionListResponse struct {
	SchemaID      string           `json:"schema_id"`
	SchemaVersion string           `json:"schema_version"`
	RequestID     string           `json:"request_id"`
	Order         string           `json:"order"`
	Sessions      []SessionSummary `json:"sessions"`
	NextCursor    string           `json:"next_cursor,omitempty"`
}

type SessionGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	SessionID     string `json:"session_id"`
}

type SessionGetResponse struct {
	SchemaID      string        `json:"schema_id"`
	SchemaVersion string        `json:"schema_version"`
	RequestID     string        `json:"request_id"`
	Session       SessionDetail `json:"session"`
}

type SessionSendMessageRequest struct {
	SchemaID       string                  `json:"schema_id"`
	SchemaVersion  string                  `json:"schema_version"`
	RequestID      string                  `json:"request_id"`
	SessionID      string                  `json:"session_id"`
	Role           string                  `json:"role"`
	ContentText    string                  `json:"content_text"`
	IdempotencyKey string                  `json:"idempotency_key,omitempty"`
	RelatedLinks   *SessionTranscriptLinks `json:"related_links,omitempty"`
}

type SessionSendMessageResponse struct {
	SchemaID      string                   `json:"schema_id"`
	SchemaVersion string                   `json:"schema_version"`
	RequestID     string                   `json:"request_id"`
	SessionID     string                   `json:"session_id"`
	Turn          SessionTranscriptTurn    `json:"turn"`
	Message       SessionTranscriptMessage `json:"message"`
	EventType     string                   `json:"event_type"`
	StreamID      string                   `json:"stream_id"`
	Seq           int64                    `json:"seq"`
}

type SessionExecutionTriggerRequest struct {
	SchemaID               string `json:"schema_id"`
	SchemaVersion          string `json:"schema_version"`
	RequestID              string `json:"request_id"`
	SessionID              string `json:"session_id"`
	TurnID                 string `json:"turn_id,omitempty"`
	TriggerSource          string `json:"trigger_source"`
	RequestedOperation     string `json:"requested_operation"`
	ApprovalProfile        string `json:"approval_profile,omitempty"`
	AutonomyPosture        string `json:"autonomy_posture,omitempty"`
	UserMessageContentText string `json:"user_message_content_text,omitempty"`
	IdempotencyKey         string `json:"idempotency_key,omitempty"`
}

type SessionExecutionTriggerResponse struct {
	SchemaID               string `json:"schema_id"`
	SchemaVersion          string `json:"schema_version"`
	RequestID              string `json:"request_id"`
	SessionID              string `json:"session_id"`
	TriggerID              string `json:"trigger_id"`
	TurnID                 string `json:"turn_id"`
	TriggerSource          string `json:"trigger_source"`
	RequestedOperation     string `json:"requested_operation"`
	ApprovalProfile        string `json:"approval_profile"`
	AutonomyPosture        string `json:"autonomy_posture"`
	ExecutionState         string `json:"execution_state"`
	UserMessageContentText string `json:"user_message_content_text,omitempty"`
	EventType              string `json:"event_type"`
	StreamID               string `json:"stream_id"`
	Seq                    int64  `json:"seq"`
}

type SessionWatchRequest struct {
	SchemaID         string             `json:"schema_id"`
	SchemaVersion    string             `json:"schema_version"`
	RequestID        string             `json:"request_id"`
	StreamID         string             `json:"stream_id"`
	SessionID        string             `json:"session_id,omitempty"`
	WorkspaceID      string             `json:"workspace_id,omitempty"`
	Status           string             `json:"status,omitempty"`
	LastActivityKind string             `json:"last_activity_kind,omitempty"`
	Follow           bool               `json:"follow"`
	IncludeSnapshot  bool               `json:"include_snapshot"`
	RequestCtx       context.Context    `json:"-"`
	Cancel           context.CancelFunc `json:"-"`
	Release          func()             `json:"-"`
}

type SessionWatchEvent struct {
	SchemaID       string          `json:"schema_id"`
	SchemaVersion  string          `json:"schema_version"`
	StreamID       string          `json:"stream_id"`
	RequestID      string          `json:"request_id"`
	Seq            int64           `json:"seq"`
	EventType      string          `json:"event_type"`
	Session        *SessionSummary `json:"session,omitempty"`
	Terminal       bool            `json:"terminal,omitempty"`
	TerminalStatus string          `json:"terminal_status,omitempty"`
	Error          *ProtocolError  `json:"error,omitempty"`
}

type SessionTurnExecutionWatchRequest struct {
	SchemaID        string             `json:"schema_id"`
	SchemaVersion   string             `json:"schema_version"`
	RequestID       string             `json:"request_id"`
	StreamID        string             `json:"stream_id"`
	SessionID       string             `json:"session_id,omitempty"`
	WorkspaceID     string             `json:"workspace_id,omitempty"`
	TurnID          string             `json:"turn_id,omitempty"`
	ExecutionState  string             `json:"execution_state,omitempty"`
	WaitKind        string             `json:"wait_kind,omitempty"`
	Follow          bool               `json:"follow"`
	IncludeSnapshot bool               `json:"include_snapshot"`
	RequestCtx      context.Context    `json:"-"`
	Cancel          context.CancelFunc `json:"-"`
	Release         func()             `json:"-"`
}

type SessionTurnExecutionWatchEvent struct {
	SchemaID       string                `json:"schema_id"`
	SchemaVersion  string                `json:"schema_version"`
	StreamID       string                `json:"stream_id"`
	RequestID      string                `json:"request_id"`
	Seq            int64                 `json:"seq"`
	EventType      string                `json:"event_type"`
	TurnExecution  *SessionTurnExecution `json:"turn_execution,omitempty"`
	Terminal       bool                  `json:"terminal,omitempty"`
	TerminalStatus string                `json:"terminal_status,omitempty"`
	Error          *ProtocolError        `json:"error,omitempty"`
}
