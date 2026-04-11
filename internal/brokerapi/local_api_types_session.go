package brokerapi

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
	LinkedRunIDs             []string                `json:"linked_run_ids"`
	LinkedApprovalIDs        []string                `json:"linked_approval_ids"`
	LinkedArtifactDigests    []string                `json:"linked_artifact_digests"`
	LinkedAuditRecordDigests []string                `json:"linked_audit_record_digests"`
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
