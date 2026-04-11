package artifacts

import (
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type DataClass string

const (
	DataClassSpecText                DataClass = "spec_text"
	DataClassUnapprovedFileExcerpts  DataClass = "unapproved_file_excerpts"
	DataClassApprovedFileExcerpts    DataClass = "approved_file_excerpts"
	DataClassDiffs                   DataClass = "diffs"
	DataClassBuildLogs               DataClass = "build_logs"
	DataClassAuditEvents             DataClass = "audit_events"
	DataClassAuditVerificationReport DataClass = "audit_verification_report"
	DataClassGateEvidence            DataClass = "gate_evidence"
	DataClassWebQuery                DataClass = "web_query"
	DataClassWebCitations            DataClass = "web_citations"
)

var allDataClasses = map[DataClass]struct{}{
	DataClassSpecText:                {},
	DataClassUnapprovedFileExcerpts:  {},
	DataClassApprovedFileExcerpts:    {},
	DataClassDiffs:                   {},
	DataClassBuildLogs:               {},
	DataClassAuditEvents:             {},
	DataClassAuditVerificationReport: {},
	DataClassGateEvidence:            {},
	DataClassWebQuery:                {},
	DataClassWebCitations:            {},
}

type ArtifactReference struct {
	Digest                string    `json:"digest"`
	SizeBytes             int64     `json:"size_bytes"`
	ContentType           string    `json:"content_type"`
	DataClass             DataClass `json:"data_class"`
	ProvenanceReceiptHash string    `json:"provenance_receipt_hash"`
}

type ArtifactRecord struct {
	Reference             ArtifactReference `json:"reference"`
	BlobPath              string            `json:"blob_path"`
	CreatedAt             time.Time         `json:"created_at"`
	CreatedByRole         string            `json:"created_by_role"`
	RunID                 string            `json:"run_id,omitempty"`
	StepID                string            `json:"step_id,omitempty"`
	StorageProtection     string            `json:"storage_protection"`
	ApprovalOfDigest      string            `json:"approval_of_digest,omitempty"`
	ApprovalDecisionHash  string            `json:"approval_decision_hash,omitempty"`
	PromotionRequestHash  string            `json:"promotion_request_hash,omitempty"`
	PromotionApprovedBy   string            `json:"promotion_approved_by,omitempty"`
	PromotionApprovedAt   *time.Time        `json:"promotion_approved_at,omitempty"`
	RetentionProtectedRun []string          `json:"retention_protected_run,omitempty"`
}

type Quota struct {
	MaxArtifactCount      int   `json:"max_artifact_count"`
	MaxTotalBytes         int64 `json:"max_total_bytes"`
	MaxSingleArtifactSize int64 `json:"max_single_artifact_bytes"`
}

type FlowRule struct {
	ProducerRole       string      `json:"producer_role"`
	ConsumerRole       string      `json:"consumer_role"`
	AllowedDataClasses []DataClass `json:"allowed_data_classes"`
}

type Policy struct {
	HandOffReferenceMode                string           `json:"handoff_reference_mode"`
	ReservedClassesEnabled              bool             `json:"reserved_classes_enabled"`
	EncryptedAtRestDefault              bool             `json:"encrypted_at_rest_default"`
	DevPlaintextOverride                bool             `json:"dev_plaintext_override"`
	ExplicitHumanApprovalRequired       bool             `json:"explicit_human_approval_required"`
	PromotionMintsNewArtifactReference  bool             `json:"promotion_mints_new_artifact_reference"`
	MaxPromotionRequestBytes            int64            `json:"max_promotion_request_bytes"`
	MaxPromotionRequestsPerMinute       int              `json:"max_promotion_requests_per_minute"`
	BulkPromotionRequiresSeparateReview bool             `json:"bulk_promotion_requires_separate_approval"`
	FlowMatrix                          []FlowRule       `json:"flow_matrix"`
	RevokedApprovedExcerptHashes        map[string]bool  `json:"revoked_approved_excerpt_hashes"`
	PerRoleQuota                        map[string]Quota `json:"per_role_quota"`
	PerStepQuota                        map[string]Quota `json:"per_step_quota"`
	UnreferencedTTLSeconds              int64            `json:"unreferenced_ttl_seconds"`
	DeleteOnQuotaPressure               bool             `json:"delete_unreferenced_on_quota_pressure"`
	RequireOriginMetadata               []string         `json:"require_origin_metadata"`
	RequireFullContentVisibility        bool             `json:"require_full_content_visibility"`
	ApprovedExcerptEgressOptInOnly      bool             `json:"approved_excerpt_egress_opt_in_only"`
	UnapprovedExcerptEgressDenied       bool             `json:"unapproved_excerpt_egress_denied"`
}

type StoreState struct {
	Artifacts                map[string]ArtifactRecord                          `json:"artifacts"`
	PolicyDecisions          map[string]PolicyDecisionRecord                    `json:"policy_decisions,omitempty"`
	RunPolicyDecisionRefs    map[string][]string                                `json:"run_policy_decision_refs,omitempty"`
	Approvals                map[string]ApprovalRecord                          `json:"approvals,omitempty"`
	RunApprovalRefs          map[string][]string                                `json:"run_approval_refs,omitempty"`
	RuntimeFactsByRun        map[string]launcherbackend.RuntimeFactsSnapshot    `json:"runtime_facts_by_run,omitempty"`
	RuntimeEvidenceByRun     map[string]launcherbackend.RuntimeEvidenceSnapshot `json:"runtime_evidence_by_run,omitempty"`
	RuntimeLifecycleByRun    map[string]launcherbackend.RuntimeLifecycleState   `json:"runtime_lifecycle_by_run,omitempty"`
	RuntimeAuditStateByRun   map[string]RuntimeAuditEmissionState               `json:"runtime_audit_state_by_run,omitempty"`
	RunnerAdvisoryByRun      map[string]RunnerAdvisoryState                     `json:"runner_advisory_by_run,omitempty"`
	Policy                   Policy                                             `json:"policy"`
	Runs                     map[string]string                                  `json:"runs"`
	PromotionEventsByActor   map[string][]time.Time                             `json:"promotion_events_by_actor"`
	LastAuditSequence        int64                                              `json:"last_audit_sequence"`
	StorageProtectionPosture string                                             `json:"storage_protection_posture"`
	BackupHMACKey            string                                             `json:"backup_hmac_key"`
}

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
	LastIsolateSessionStartedDigest string `json:"last_isolate_session_started_digest,omitempty"`
	LastIsolateSessionBoundDigest   string `json:"last_isolate_session_bound_digest,omitempty"`
}

type ApprovalRecord struct {
	ApprovalID             string                            `json:"approval_id"`
	Status                 string                            `json:"status"`
	WorkspaceID            string                            `json:"workspace_id,omitempty"`
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

type PutRequest struct {
	Payload               []byte
	ContentType           string
	DataClass             DataClass
	ProvenanceReceiptHash string
	CreatedByRole         string
	TrustedSource         bool
	RunID                 string
	StepID                string
}

type FlowCheckRequest struct {
	ProducerRole  string
	ConsumerRole  string
	DataClass     DataClass
	Digest        string
	IsEgress      bool
	ManifestOptIn bool
}

type ArtifactReadRequest struct {
	Digest        string
	ProducerRole  string
	ConsumerRole  string
	DataClass     DataClass
	IsEgress      bool
	ManifestOptIn bool
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

type GCResult struct {
	DeletedDigests []string `json:"deleted_digests"`
	FreedBytes     int64    `json:"freed_bytes"`
}

type BackupManifest struct {
	Schema            string                 `json:"schema"`
	ExportedAt        time.Time              `json:"exported_at"`
	StorageProtection string                 `json:"storage_protection"`
	Policy            Policy                 `json:"policy"`
	Artifacts         []ArtifactRecord       `json:"artifacts"`
	PolicyDecisions   []PolicyDecisionRecord `json:"policy_decisions,omitempty"`
	Approvals         []ApprovalRecord       `json:"approvals,omitempty"`
	Runs              map[string]string      `json:"runs"`
}

type AuditEvent struct {
	Seq        int64                  `json:"seq"`
	Type       string                 `json:"type"`
	OccurredAt time.Time              `json:"occurred_at"`
	Actor      string                 `json:"actor"`
	Details    map[string]interface{} `json:"details"`
}

type BackupSignature struct {
	Schema         string    `json:"schema"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	HMACSHA256     string    `json:"hmac_sha256"`
	KeyID          string    `json:"key_id"`
	ExportedAt     time.Time `json:"exported_at"`
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
