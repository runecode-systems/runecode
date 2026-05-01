package brokerapi

import (
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type ExternalAnchorMutationPreparedTarget struct {
	TargetKind             string             `json:"target_kind"`
	TargetRequirement      string             `json:"target_requirement,omitempty"`
	TargetDescriptor       map[string]any     `json:"target_descriptor"`
	TargetDescriptorDigest trustpolicy.Digest `json:"target_descriptor_digest"`
}

type ExternalAnchorMutationPreparedState struct {
	SchemaID                     string                                 `json:"schema_id"`
	SchemaVersion                string                                 `json:"schema_version"`
	PreparedMutationID           string                                 `json:"prepared_mutation_id"`
	RunID                        string                                 `json:"run_id"`
	ExecutionPathway             string                                 `json:"execution_pathway"`
	AnchorPosture                string                                 `json:"anchor_posture"`
	DestinationRef               string                                 `json:"destination_ref"`
	PrimaryTarget                ExternalAnchorMutationPreparedTarget   `json:"primary_target"`
	TargetSet                    []ExternalAnchorMutationPreparedTarget `json:"target_set,omitempty"`
	RequestKind                  string                                 `json:"request_kind"`
	TypedRequestSchemaID         string                                 `json:"typed_request_schema_id"`
	TypedRequestSchemaVersion    string                                 `json:"typed_request_schema_version"`
	TypedRequest                 map[string]any                         `json:"typed_request"`
	TypedRequestHash             trustpolicy.Digest                     `json:"typed_request_hash"`
	ActionRequestHash            trustpolicy.Digest                     `json:"action_request_hash"`
	PolicyDecisionHash           trustpolicy.Digest                     `json:"policy_decision_hash"`
	RequiredApprovalID           string                                 `json:"required_approval_id,omitempty"`
	RequiredApprovalRequestHash  *trustpolicy.Digest                    `json:"required_approval_request_hash,omitempty"`
	RequiredApprovalDecisionHash *trustpolicy.Digest                    `json:"required_approval_decision_hash,omitempty"`
	LifecycleState               string                                 `json:"lifecycle_state"`
	LifecycleReasonCode          string                                 `json:"lifecycle_reason_code,omitempty"`
	ExecutionState               string                                 `json:"execution_state"`
	ExecutionReasonCode          string                                 `json:"execution_reason_code,omitempty"`
	CreatedAt                    string                                 `json:"created_at"`
	UpdatedAt                    string                                 `json:"updated_at"`
	LastPrepareRequestID         string                                 `json:"last_prepare_request_id,omitempty"`
	LastGetRequestID             string                                 `json:"last_get_request_id,omitempty"`
	LastExecuteRequestID         string                                 `json:"last_execute_request_id,omitempty"`
	LastExecuteTargetAuthLeaseID string                                 `json:"last_execute_target_auth_lease_id,omitempty"`
	LastExecuteAttemptID         string                                 `json:"last_execute_attempt_id,omitempty"`
	LastExecuteAttemptSealDigest *trustpolicy.Digest                    `json:"last_execute_attempt_seal_digest,omitempty"`
	LastExecuteAttemptTargetID   *trustpolicy.Digest                    `json:"last_execute_attempt_target_descriptor_digest,omitempty"`
	LastExecuteAttemptRequestID  *trustpolicy.Digest                    `json:"last_execute_attempt_typed_request_hash,omitempty"`
	LastExecuteSnapshotSegmentID string                                 `json:"last_execute_snapshot_segment_id,omitempty"`
	LastExecuteSnapshotSealID    *trustpolicy.Digest                    `json:"last_execute_snapshot_seal_digest,omitempty"`
	LastExecuteDeferredPolls     int                                    `json:"last_execute_deferred_polls_remaining,omitempty"`
	LastExecuteDeferredClaimID   string                                 `json:"last_execute_deferred_claim_id,omitempty"`
	LastAnchorReceiptDigest      *trustpolicy.Digest                    `json:"last_anchor_receipt_digest,omitempty"`
	LastAnchorEvidenceDigest     *trustpolicy.Digest                    `json:"last_anchor_evidence_digest,omitempty"`
	LastAnchorVerificationDigest *trustpolicy.Digest                    `json:"last_anchor_verification_digest,omitempty"`
	LastAnchorProofDigest        *trustpolicy.Digest                    `json:"last_anchor_proof_digest,omitempty"`
	LastAnchorProviderReceipt    *trustpolicy.Digest                    `json:"last_anchor_provider_receipt_digest,omitempty"`
	LastAnchorTranscriptDigest   *trustpolicy.Digest                    `json:"last_anchor_transcript_digest,omitempty"`
}

type ExternalAnchorMutationPrepareRequest struct {
	SchemaID      string         `json:"schema_id"`
	SchemaVersion string         `json:"schema_version"`
	RequestID     string         `json:"request_id"`
	RunID         string         `json:"run_id"`
	TypedRequest  map[string]any `json:"typed_request"`
}

type ExternalAnchorMutationPrepareResponse struct {
	SchemaID           string                              `json:"schema_id"`
	SchemaVersion      string                              `json:"schema_version"`
	RequestID          string                              `json:"request_id"`
	PreparedMutationID string                              `json:"prepared_mutation_id"`
	TypedRequestHash   trustpolicy.Digest                  `json:"typed_request_hash"`
	Prepared           ExternalAnchorMutationPreparedState `json:"prepared"`
}

type ExternalAnchorMutationGetRequest struct {
	SchemaID           string `json:"schema_id"`
	SchemaVersion      string `json:"schema_version"`
	RequestID          string `json:"request_id"`
	PreparedMutationID string `json:"prepared_mutation_id"`
}

type ExternalAnchorMutationGetResponse struct {
	SchemaID      string                              `json:"schema_id"`
	SchemaVersion string                              `json:"schema_version"`
	RequestID     string                              `json:"request_id"`
	Prepared      ExternalAnchorMutationPreparedState `json:"prepared"`
}

type ExternalAnchorMutationIssueExecuteLeaseRequest struct {
	SchemaID           string `json:"schema_id"`
	SchemaVersion      string `json:"schema_version"`
	RequestID          string `json:"request_id"`
	PreparedMutationID string `json:"prepared_mutation_id"`
	TTLSeconds         int    `json:"ttl_seconds,omitempty"`
}

type ExternalAnchorMutationIssueExecuteLeaseResponse struct {
	SchemaID           string         `json:"schema_id"`
	SchemaVersion      string         `json:"schema_version"`
	RequestID          string         `json:"request_id"`
	PreparedMutationID string         `json:"prepared_mutation_id"`
	Lease              secretsd.Lease `json:"lease"`
	TargetAuthLeaseID  string         `json:"target_auth_lease_id"`
}

type ExternalAnchorMutationExecuteRequest struct {
	SchemaID             string             `json:"schema_id"`
	SchemaVersion        string             `json:"schema_version"`
	RequestID            string             `json:"request_id"`
	PreparedMutationID   string             `json:"prepared_mutation_id"`
	ApprovalID           string             `json:"approval_id"`
	ApprovalRequestHash  trustpolicy.Digest `json:"approval_request_hash"`
	ApprovalDecisionHash trustpolicy.Digest `json:"approval_decision_hash"`
	TargetAuthLeaseID    string             `json:"target_auth_lease_id"`
	ExportReceiptCopy    bool               `json:"export_receipt_copy,omitempty"`
}

type ExternalAnchorMutationExecuteResponse struct {
	SchemaID           string                              `json:"schema_id"`
	SchemaVersion      string                              `json:"schema_version"`
	RequestID          string                              `json:"request_id"`
	PreparedMutationID string                              `json:"prepared_mutation_id"`
	ExecutionState     string                              `json:"execution_state"`
	Prepared           ExternalAnchorMutationPreparedState `json:"prepared"`
}
