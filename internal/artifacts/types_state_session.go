package artifacts

import "time"

type SessionDurableState struct {
	SessionID                        string                                              `json:"session_id"`
	WorkspaceID                      string                                              `json:"workspace_id"`
	CreatedAt                        time.Time                                           `json:"created_at"`
	CreatedByRunID                   string                                              `json:"created_by_run_id,omitempty"`
	UpdatedAt                        time.Time                                           `json:"updated_at"`
	Status                           string                                              `json:"status"`
	WorkPosture                      string                                              `json:"work_posture"`
	WorkPostureReason                string                                              `json:"work_posture_reason_code,omitempty"`
	LastActivityAt                   time.Time                                           `json:"last_activity_at"`
	LastActivityKind                 string                                              `json:"last_activity_kind"`
	LastActivityPreview              string                                              `json:"last_activity_preview,omitempty"`
	LastInteractionSequence          int64                                               `json:"last_interaction_sequence"`
	TurnCount                        int                                                 `json:"turn_count"`
	HasIncompleteTurn                bool                                                `json:"has_incomplete_turn"`
	TranscriptTurns                  []SessionTranscriptTurnDurableState                 `json:"transcript_turns,omitempty"`
	IdempotencyByKey                 map[string]SessionIdempotencyRecord                 `json:"idempotency_by_key,omitempty"`
	ExecutionTriggers                []SessionExecutionTriggerDurableState               `json:"execution_triggers,omitempty"`
	TurnExecutions                   []SessionTurnExecutionDurableState                  `json:"turn_executions,omitempty"`
	ExecutionTriggerIdempotencyByKey map[string]SessionExecutionTriggerIdempotencyRecord `json:"execution_trigger_idempotency_by_key,omitempty"`
	LinkedRunIDs                     []string                                            `json:"linked_run_ids,omitempty"`
}

type SessionExecutionTriggerDurableState struct {
	TriggerID              string    `json:"trigger_id"`
	SessionID              string    `json:"session_id"`
	TriggerIndex           int       `json:"trigger_index"`
	TriggerSource          string    `json:"trigger_source"`
	RequestedOperation     string    `json:"requested_operation"`
	UserMessageContentText string    `json:"user_message_content_text,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
}

type SessionTurnExecutionDurableState struct {
	TurnID                               string    `json:"turn_id"`
	SessionID                            string    `json:"session_id"`
	ExecutionIndex                       int       `json:"execution_index"`
	OrchestrationScopeID                 string    `json:"orchestration_scope_id,omitempty"`
	DependsOnScopeIDs                    []string  `json:"depends_on_scope_ids,omitempty"`
	TriggerID                            string    `json:"trigger_id"`
	TriggerSource                        string    `json:"trigger_source"`
	RequestedOperation                   string    `json:"requested_operation"`
	ExecutionState                       string    `json:"execution_state"`
	WaitKind                             string    `json:"wait_kind,omitempty"`
	WaitState                            string    `json:"wait_state,omitempty"`
	ApprovalProfile                      string    `json:"approval_profile"`
	AutonomyPosture                      string    `json:"autonomy_posture"`
	PrimaryRunID                         string    `json:"primary_run_id,omitempty"`
	PendingApprovalID                    string    `json:"pending_approval_id,omitempty"`
	LinkedRunIDs                         []string  `json:"linked_run_ids,omitempty"`
	LinkedApprovalIDs                    []string  `json:"linked_approval_ids,omitempty"`
	LinkedArtifactDigests                []string  `json:"linked_artifact_digests,omitempty"`
	LinkedAuditRecordDigests             []string  `json:"linked_audit_record_digests,omitempty"`
	BoundValidatedProjectSubstrateDigest string    `json:"bound_validated_project_substrate_digest,omitempty"`
	BlockedReasonCode                    string    `json:"blocked_reason_code,omitempty"`
	TerminalOutcome                      string    `json:"terminal_outcome,omitempty"`
	CreatedAt                            time.Time `json:"created_at"`
	UpdatedAt                            time.Time `json:"updated_at"`
}

type SessionExecutionTriggerIdempotencyRecord struct {
	RequestHash string `json:"request_hash"`
	TriggerID   string `json:"trigger_id"`
	TurnID      string `json:"turn_id,omitempty"`
	Seq         int64  `json:"seq"`
}

type SessionTranscriptLinksDurableState struct {
	RunIDs             []string `json:"run_ids,omitempty"`
	ApprovalIDs        []string `json:"approval_ids,omitempty"`
	ArtifactDigests    []string `json:"artifact_digests,omitempty"`
	AuditRecordDigests []string `json:"audit_record_digests,omitempty"`
}

type SessionTranscriptMessageDurableState struct {
	MessageID    string                             `json:"message_id"`
	TurnID       string                             `json:"turn_id"`
	SessionID    string                             `json:"session_id"`
	MessageIndex int                                `json:"message_index"`
	Role         string                             `json:"role"`
	CreatedAt    time.Time                          `json:"created_at"`
	ContentText  string                             `json:"content_text"`
	RelatedLinks SessionTranscriptLinksDurableState `json:"related_links"`
}

type SessionTranscriptTurnDurableState struct {
	TurnID      string                                 `json:"turn_id"`
	SessionID   string                                 `json:"session_id"`
	TurnIndex   int                                    `json:"turn_index"`
	StartedAt   time.Time                              `json:"started_at"`
	CompletedAt *time.Time                             `json:"completed_at,omitempty"`
	Status      string                                 `json:"status"`
	Messages    []SessionTranscriptMessageDurableState `json:"messages,omitempty"`
}

type SessionIdempotencyRecord struct {
	RequestHash string `json:"request_hash"`
	TurnID      string `json:"turn_id"`
	MessageID   string `json:"message_id"`
	Seq         int64  `json:"seq"`
}
