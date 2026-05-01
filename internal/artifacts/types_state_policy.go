package artifacts

import (
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type StoreState struct {
	Artifacts                     map[string]ArtifactRecord                                       `json:"artifacts"`
	Sessions                      map[string]SessionDurableState                                  `json:"sessions,omitempty"`
	PolicyDecisions               map[string]PolicyDecisionRecord                                 `json:"policy_decisions,omitempty"`
	RunPolicyDecisionRefs         map[string][]string                                             `json:"run_policy_decision_refs,omitempty"`
	Approvals                     map[string]ApprovalRecord                                       `json:"approvals,omitempty"`
	RunApprovalRefs               map[string][]string                                             `json:"run_approval_refs,omitempty"`
	GitRemotePrepared             map[string]GitRemotePreparedMutationRecord                      `json:"git_remote_prepared,omitempty"`
	RunGitRemotePreparedRefs      map[string][]string                                             `json:"run_git_remote_prepared_refs,omitempty"`
	ExternalAnchorPrepared        map[string]ExternalAnchorPreparedMutationRecord                 `json:"external_anchor_prepared,omitempty"`
	RunExternalAnchorPreparedRefs map[string][]string                                             `json:"run_external_anchor_prepared_refs,omitempty"`
	RuntimeFactsByRun             map[string]launcherbackend.RuntimeFactsSnapshot                 `json:"runtime_facts_by_run,omitempty"`
	RuntimeEvidenceByRun          map[string]launcherbackend.RuntimeEvidenceSnapshot              `json:"runtime_evidence_by_run,omitempty"`
	AttestationVerificationCache  map[string]launcherbackend.IsolateAttestationVerificationRecord `json:"attestation_verification_cache,omitempty"`
	RuntimeLifecycleByRun         map[string]launcherbackend.RuntimeLifecycleState                `json:"runtime_lifecycle_by_run,omitempty"`
	RuntimeAuditStateByRun        map[string]RuntimeAuditEmissionState                            `json:"runtime_audit_state_by_run,omitempty"`
	RunnerAdvisoryByRun           map[string]RunnerAdvisoryState                                  `json:"runner_advisory_by_run,omitempty"`
	DependencyCacheBatches        map[string]DependencyCacheBatchRecord                           `json:"dependency_cache_batches,omitempty"`
	DependencyCacheUnits          map[string]DependencyCacheResolvedUnitRecord                    `json:"dependency_cache_units,omitempty"`
	DependencyCacheByRequest      map[string][]string                                             `json:"dependency_cache_by_request,omitempty"`
	ProviderProfiles              map[string]ProviderProfileDurableState                          `json:"provider_profiles,omitempty"`
	ProviderSetupSessions         map[string]ProviderSetupSessionDurableState                     `json:"provider_setup_sessions,omitempty"`
	RunPlanAuthorities            map[string]RunPlanAuthorityRecord                               `json:"run_plan_authorities,omitempty"`
	RunPlanRefsByRun              map[string][]string                                             `json:"run_plan_refs_by_run,omitempty"`
	RunPlanCompilations           map[string]RunPlanCompilationRecord                             `json:"run_plan_compilations,omitempty"`
	RunPlanCompilationByCacheKey  map[string]string                                               `json:"run_plan_compilation_by_cache_key,omitempty"`
	Policy                        Policy                                                          `json:"policy"`
	Runs                          map[string]string                                               `json:"runs"`
	PromotionEventsByActor        map[string][]time.Time                                          `json:"promotion_events_by_actor"`
	LastAuditSequence             int64                                                           `json:"last_audit_sequence"`
	StorageProtectionPosture      string                                                          `json:"storage_protection_posture"`
	BackupHMACKey                 string                                                          `json:"backup_hmac_key"`
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

type ExternalAnchorPreparedMutationRecord struct {
	PreparedMutationID           string                                `json:"prepared_mutation_id"`
	RunID                        string                                `json:"run_id"`
	DestinationRef               string                                `json:"destination_ref"`
	PrimaryTarget                ExternalAnchorPreparedTargetBinding   `json:"primary_target"`
	TargetSet                    []ExternalAnchorPreparedTargetBinding `json:"target_set,omitempty"`
	RequestKind                  string                                `json:"request_kind"`
	TypedRequestSchemaID         string                                `json:"typed_request_schema_id"`
	TypedRequestSchemaVer        string                                `json:"typed_request_schema_version"`
	TypedRequest                 map[string]any                        `json:"typed_request"`
	TypedRequestHash             string                                `json:"typed_request_hash"`
	ActionRequestHash            string                                `json:"action_request_hash"`
	PolicyDecisionHash           string                                `json:"policy_decision_hash"`
	RequiredApprovalID           string                                `json:"required_approval_id,omitempty"`
	RequiredApprovalReqHash      string                                `json:"required_approval_request_hash,omitempty"`
	RequiredApprovalDecHash      string                                `json:"required_approval_decision_hash,omitempty"`
	LifecycleState               string                                `json:"lifecycle_state"`
	LifecycleReasonCode          string                                `json:"lifecycle_reason_code,omitempty"`
	ExecutionState               string                                `json:"execution_state,omitempty"`
	ExecutionReasonCode          string                                `json:"execution_reason_code,omitempty"`
	CreatedAt                    time.Time                             `json:"created_at"`
	UpdatedAt                    time.Time                             `json:"updated_at"`
	LastPrepareRequestID         string                                `json:"last_prepare_request_id,omitempty"`
	LastGetRequestID             string                                `json:"last_get_request_id,omitempty"`
	LastExecuteRequestID         string                                `json:"last_execute_request_id,omitempty"`
	LastExecuteTargetAuthLeaseID string                                `json:"last_execute_target_auth_lease_id,omitempty"`
	LastExecuteAttemptID         string                                `json:"last_execute_attempt_id,omitempty"`
	LastExecuteAttemptSealDigest string                                `json:"last_execute_attempt_seal_digest,omitempty"`
	LastExecuteAttemptTargetID   string                                `json:"last_execute_attempt_target_descriptor_digest,omitempty"`
	LastExecuteAttemptRequestID  string                                `json:"last_execute_attempt_typed_request_hash,omitempty"`
	LastExecuteSnapshotSegmentID string                                `json:"last_execute_snapshot_segment_id,omitempty"`
	LastExecuteSnapshotSealID    string                                `json:"last_execute_snapshot_seal_digest,omitempty"`
	LastExecuteDeferredPolls     int                                   `json:"last_execute_deferred_polls_remaining,omitempty"`
	LastExecuteDeferredClaimID   string                                `json:"last_execute_deferred_claim_id,omitempty"`
	LastExecuteDeferredClaimedAt *time.Time                            `json:"last_execute_deferred_claimed_at,omitempty"`
	LastExecuteApprovalID        string                                `json:"last_execute_approval_id,omitempty"`
	LastExecuteApprovalReqID     string                                `json:"last_execute_approval_request_hash,omitempty"`
	LastExecuteApprovalDecID     string                                `json:"last_execute_approval_decision_hash,omitempty"`
	LastAnchorReceiptDigest      string                                `json:"last_anchor_receipt_digest,omitempty"`
	LastAnchorEvidenceDigest     string                                `json:"last_anchor_evidence_digest,omitempty"`
	LastAnchorVerificationDigest string                                `json:"last_anchor_verification_digest,omitempty"`
	LastAnchorProofDigest        string                                `json:"last_anchor_proof_digest,omitempty"`
	LastAnchorProviderReceipt    string                                `json:"last_anchor_provider_receipt_digest,omitempty"`
	LastAnchorTranscriptDigest   string                                `json:"last_anchor_transcript_digest,omitempty"`
}

type ExternalAnchorPreparedTargetBinding struct {
	TargetKind             string         `json:"target_kind"`
	TargetRequirement      string         `json:"target_requirement,omitempty"`
	TargetDescriptor       map[string]any `json:"target_descriptor"`
	TargetDescriptorDigest string         `json:"target_descriptor_digest"`
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
	Schema                       string                                                          `json:"schema"`
	ExportedAt                   time.Time                                                       `json:"exported_at"`
	StorageProtection            string                                                          `json:"storage_protection"`
	Policy                       Policy                                                          `json:"policy"`
	Artifacts                    []ArtifactRecord                                                `json:"artifacts"`
	DependencyCacheBatches       []DependencyCacheBatchRecord                                    `json:"dependency_cache_batches,omitempty"`
	DependencyCacheUnits         []DependencyCacheResolvedUnitRecord                             `json:"dependency_cache_units,omitempty"`
	Sessions                     []SessionDurableState                                           `json:"sessions,omitempty"`
	PolicyDecisions              []PolicyDecisionRecord                                          `json:"policy_decisions,omitempty"`
	Approvals                    []ApprovalRecord                                                `json:"approvals,omitempty"`
	GitRemotePrepared            []GitRemotePreparedMutationRecord                               `json:"git_remote_prepared,omitempty"`
	ExternalAnchorPrepared       []ExternalAnchorPreparedMutationRecord                          `json:"external_anchor_prepared,omitempty"`
	RuntimeFactsByRun            map[string]launcherbackend.RuntimeFactsSnapshot                 `json:"runtime_facts_by_run,omitempty"`
	RuntimeEvidenceByRun         map[string]launcherbackend.RuntimeEvidenceSnapshot              `json:"runtime_evidence_by_run,omitempty"`
	AttestationVerificationCache map[string]launcherbackend.IsolateAttestationVerificationRecord `json:"attestation_verification_cache,omitempty"`
	RuntimeLifecycleByRun        map[string]launcherbackend.RuntimeLifecycleState                `json:"runtime_lifecycle_by_run,omitempty"`
	RuntimeAuditStateByRun       map[string]RuntimeAuditEmissionState                            `json:"runtime_audit_state_by_run,omitempty"`
	RunnerAdvisoryByRun          map[string]RunnerAdvisoryState                                  `json:"runner_advisory_by_run,omitempty"`
	ProviderProfiles             []ProviderProfileDurableState                                   `json:"provider_profiles,omitempty"`
	ProviderSetupSessions        []ProviderSetupSessionDurableState                              `json:"provider_setup_sessions,omitempty"`
	RunPlanAuthorities           []RunPlanAuthorityRecord                                        `json:"run_plan_authorities,omitempty"`
	RunPlanCompilations          []RunPlanCompilationRecord                                      `json:"run_plan_compilations,omitempty"`
	Runs                         map[string]string                                               `json:"runs"`
}
