package artifacts

import (
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type StoreState struct {
	Artifacts                map[string]ArtifactRecord                          `json:"artifacts"`
	Sessions                 map[string]SessionDurableState                     `json:"sessions,omitempty"`
	PolicyDecisions          map[string]PolicyDecisionRecord                    `json:"policy_decisions,omitempty"`
	RunPolicyDecisionRefs    map[string][]string                                `json:"run_policy_decision_refs,omitempty"`
	Approvals                map[string]ApprovalRecord                          `json:"approvals,omitempty"`
	RunApprovalRefs          map[string][]string                                `json:"run_approval_refs,omitempty"`
	GitRemotePrepared        map[string]GitRemotePreparedMutationRecord         `json:"git_remote_prepared,omitempty"`
	RunGitRemotePreparedRefs map[string][]string                                `json:"run_git_remote_prepared_refs,omitempty"`
	RuntimeFactsByRun        map[string]launcherbackend.RuntimeFactsSnapshot    `json:"runtime_facts_by_run,omitempty"`
	RuntimeEvidenceByRun     map[string]launcherbackend.RuntimeEvidenceSnapshot `json:"runtime_evidence_by_run,omitempty"`
	RuntimeLifecycleByRun    map[string]launcherbackend.RuntimeLifecycleState   `json:"runtime_lifecycle_by_run,omitempty"`
	RuntimeAuditStateByRun   map[string]RuntimeAuditEmissionState               `json:"runtime_audit_state_by_run,omitempty"`
	RunnerAdvisoryByRun      map[string]RunnerAdvisoryState                     `json:"runner_advisory_by_run,omitempty"`
	ProviderProfiles         map[string]ProviderProfileDurableState             `json:"provider_profiles,omitempty"`
	ProviderSetupSessions    map[string]ProviderSetupSessionDurableState        `json:"provider_setup_sessions,omitempty"`
	Policy                   Policy                                             `json:"policy"`
	Runs                     map[string]string                                  `json:"runs"`
	PromotionEventsByActor   map[string][]time.Time                             `json:"promotion_events_by_actor"`
	LastAuditSequence        int64                                              `json:"last_audit_sequence"`
	StorageProtectionPosture string                                             `json:"storage_protection_posture"`
	BackupHMACKey            string                                             `json:"backup_hmac_key"`
}

type SessionDurableState struct {
	SessionID           string                              `json:"session_id"`
	WorkspaceID         string                              `json:"workspace_id"`
	CreatedAt           time.Time                           `json:"created_at"`
	CreatedByRunID      string                              `json:"created_by_run_id,omitempty"`
	UpdatedAt           time.Time                           `json:"updated_at"`
	Status              string                              `json:"status"`
	WorkPosture         string                              `json:"work_posture"`
	WorkPostureReason   string                              `json:"work_posture_reason_code,omitempty"`
	LastActivityAt      time.Time                           `json:"last_activity_at"`
	LastActivityKind    string                              `json:"last_activity_kind"`
	LastActivityPreview string                              `json:"last_activity_preview,omitempty"`
	TurnCount           int                                 `json:"turn_count"`
	HasIncompleteTurn   bool                                `json:"has_incomplete_turn"`
	TranscriptTurns     []SessionTranscriptTurnDurableState `json:"transcript_turns,omitempty"`
	IdempotencyByKey    map[string]SessionIdempotencyRecord `json:"idempotency_by_key,omitempty"`
	LinkedRunIDs        []string                            `json:"linked_run_ids,omitempty"`
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

type ApprovalRecord struct {
	ApprovalID             string                            `json:"approval_id"`
	Status                 string                            `json:"status"`
	WorkspaceID            string                            `json:"workspace_id,omitempty"`
	InstanceID             string                            `json:"instance_id,omitempty"`
	RunID                  string                            `json:"run_id,omitempty"`
	StageID                string                            `json:"stage_id,omitempty"`
	StepID                 string                            `json:"step_id,omitempty"`
	RoleInstanceID         string                            `json:"role_instance_id,omitempty"`
	ActionKind             string                            `json:"action_kind"`
	RequestedAt            time.Time                         `json:"requested_at"`
	ExpiresAt              *time.Time                        `json:"expires_at,omitempty"`
	DecidedAt              *time.Time                        `json:"decided_at,omitempty"`
	ConsumedAt             *time.Time                        `json:"consumed_at,omitempty"`
	ApprovalTriggerCode    string                            `json:"approval_trigger_code"`
	ChangesIfApproved      string                            `json:"changes_if_approved"`
	ApprovalAssuranceLevel string                            `json:"approval_assurance_level"`
	PresenceMode           string                            `json:"presence_mode"`
	PolicyDecisionHash     string                            `json:"policy_decision_hash,omitempty"`
	SupersededByApprovalID string                            `json:"superseded_by_approval_id,omitempty"`
	ManifestHash           string                            `json:"manifest_hash"`
	ActionRequestHash      string                            `json:"action_request_hash"`
	RelevantArtifactHashes []string                          `json:"relevant_artifact_hashes,omitempty"`
	RequestDigest          string                            `json:"request_digest,omitempty"`
	DecisionDigest         string                            `json:"decision_digest,omitempty"`
	SourceDigest           string                            `json:"source_digest,omitempty"`
	RequestEnvelope        *trustpolicy.SignedObjectEnvelope `json:"request_envelope,omitempty"`
	DecisionEnvelope       *trustpolicy.SignedObjectEnvelope `json:"decision_envelope,omitempty"`
	AuditEventType         string                            `json:"audit_event_type,omitempty"`
	AuditEventSeq          int64                             `json:"audit_event_seq,omitempty"`
}

type PolicyDecisionRecord struct {
	Digest                   string         `json:"digest"`
	RunID                    string         `json:"run_id,omitempty"`
	SchemaID                 string         `json:"schema_id"`
	SchemaVersion            string         `json:"schema_version"`
	DecisionOutcome          string         `json:"decision_outcome"`
	PolicyReasonCode         string         `json:"policy_reason_code"`
	ManifestHash             string         `json:"manifest_hash"`
	ActionRequestHash        string         `json:"action_request_hash"`
	PolicyInputHashes        []string       `json:"policy_input_hashes"`
	RelevantArtifactHashes   []string       `json:"relevant_artifact_hashes"`
	DetailsSchemaID          string         `json:"details_schema_id"`
	Details                  map[string]any `json:"details"`
	RequiredApprovalSchemaID string         `json:"required_approval_schema_id,omitempty"`
	RequiredApproval         map[string]any `json:"required_approval,omitempty"`
	RecordedAt               time.Time      `json:"recorded_at"`
	AuditEventType           string         `json:"audit_event_type"`
	AuditEventSeq            int64          `json:"audit_event_seq"`
}

type GitRemotePreparedMutationRecord struct {
	PreparedMutationID       string         `json:"prepared_mutation_id"`
	RunID                    string         `json:"run_id"`
	Provider                 string         `json:"provider"`
	DestinationRef           string         `json:"destination_ref"`
	RequestKind              string         `json:"request_kind"`
	TypedRequestSchemaID     string         `json:"typed_request_schema_id"`
	TypedRequestSchemaVer    string         `json:"typed_request_schema_version"`
	TypedRequest             map[string]any `json:"typed_request"`
	TypedRequestHash         string         `json:"typed_request_hash"`
	ActionRequestHash        string         `json:"action_request_hash"`
	PolicyDecisionHash       string         `json:"policy_decision_hash"`
	RequiredApprovalID       string         `json:"required_approval_id,omitempty"`
	RequiredApprovalReqHash  string         `json:"required_approval_request_hash,omitempty"`
	RequiredApprovalDecHash  string         `json:"required_approval_decision_hash,omitempty"`
	LifecycleState           string         `json:"lifecycle_state"`
	LifecycleReasonCode      string         `json:"lifecycle_reason_code,omitempty"`
	ExecutionState           string         `json:"execution_state,omitempty"`
	ExecutionReasonCode      string         `json:"execution_reason_code,omitempty"`
	DerivedSummary           map[string]any `json:"derived_summary"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	LastPrepareRequestID     string         `json:"last_prepare_request_id,omitempty"`
	LastGetRequestID         string         `json:"last_get_request_id,omitempty"`
	LastExecuteRequestID     string         `json:"last_execute_request_id,omitempty"`
	LastExecuteApprovalID    string         `json:"last_execute_approval_id,omitempty"`
	LastExecuteApprovalReqID string         `json:"last_execute_approval_request_hash,omitempty"`
	LastExecuteApprovalDecID string         `json:"last_execute_approval_decision_hash,omitempty"`
}

type PromotionRequest struct {
	UnapprovedDigest      string
	Approver              string
	RepoPath              string
	Commit                string
	ExtractorToolVersion  string
	FullContentVisible    bool
	ExplicitViewFull      bool
	BulkRequest           bool
	BulkApprovalConfirmed bool
	ApprovalRequest       *trustpolicy.SignedObjectEnvelope
	ApprovalDecision      *trustpolicy.SignedObjectEnvelope
}

type BackupManifest struct {
	Schema                string                             `json:"schema"`
	ExportedAt            time.Time                          `json:"exported_at"`
	StorageProtection     string                             `json:"storage_protection"`
	Policy                Policy                             `json:"policy"`
	Artifacts             []ArtifactRecord                   `json:"artifacts"`
	Sessions              []SessionDurableState              `json:"sessions,omitempty"`
	PolicyDecisions       []PolicyDecisionRecord             `json:"policy_decisions,omitempty"`
	Approvals             []ApprovalRecord                   `json:"approvals,omitempty"`
	ProviderProfiles      []ProviderProfileDurableState      `json:"provider_profiles,omitempty"`
	ProviderSetupSessions []ProviderSetupSessionDurableState `json:"provider_setup_sessions,omitempty"`
	Runs                  map[string]string                  `json:"runs"`
}

func DefaultPolicy() Policy {
	return Policy{
		HandOffReferenceMode:                "hash_only",
		ReservedClassesEnabled:              false,
		EncryptedAtRestDefault:              true,
		DevPlaintextOverride:                false,
		ExplicitHumanApprovalRequired:       true,
		PromotionMintsNewArtifactReference:  true,
		MaxPromotionRequestBytes:            1024 * 1024,
		MaxPromotionRequestsPerMinute:       30,
		BulkPromotionRequiresSeparateReview: true,
		FlowMatrix: []FlowRule{
			{ProducerRole: "workspace", ConsumerRole: "model_gateway", AllowedDataClasses: []DataClass{DataClassSpecText, DataClassApprovedFileExcerpts}},
			{ProducerRole: "workspace", ConsumerRole: "auditd", AllowedDataClasses: []DataClass{DataClassAuditEvents, DataClassAuditVerificationReport, DataClassGateEvidence, DataClassBuildLogs, DataClassDiffs, DataClassSpecText, DataClassUnapprovedFileExcerpts, DataClassApprovedFileExcerpts}},
		},
		RevokedApprovedExcerptHashes: map[string]bool{},
		PerRoleQuota: map[string]Quota{
			"workspace":     {MaxArtifactCount: 4096, MaxTotalBytes: 512 * 1024 * 1024, MaxSingleArtifactSize: 64 * 1024 * 1024},
			"model_gateway": {MaxArtifactCount: 4096, MaxTotalBytes: 512 * 1024 * 1024, MaxSingleArtifactSize: 64 * 1024 * 1024},
		},
		PerStepQuota:                   map[string]Quota{},
		UnreferencedTTLSeconds:         7 * 24 * 3600,
		DeleteOnQuotaPressure:          true,
		RequireOriginMetadata:          []string{"repo_path", "commit", "extractor_tool_version"},
		RequireFullContentVisibility:   true,
		ApprovedExcerptEgressOptInOnly: true,
		UnapprovedExcerptEgressDenied:  true,
	}
}
