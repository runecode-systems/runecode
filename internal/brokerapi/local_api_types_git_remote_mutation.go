package brokerapi

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type GitRemoteMutationDerivedSummary struct {
	SchemaID                      string               `json:"schema_id"`
	SchemaVersion                 string               `json:"schema_version"`
	RepositoryIdentity            string               `json:"repository_identity"`
	TargetRefs                    []string             `json:"target_refs"`
	ReferencedPatchArtifactHashes []trustpolicy.Digest `json:"referenced_patch_artifact_digests"`
	ExpectedResultTreeHash        trustpolicy.Digest   `json:"expected_result_tree_hash"`
	CommitSubject                 string               `json:"commit_subject,omitempty"`
	PullRequestTitle              string               `json:"pull_request_title,omitempty"`
	PullRequestBaseRef            string               `json:"pull_request_base_ref,omitempty"`
	PullRequestHeadRef            string               `json:"pull_request_head_ref,omitempty"`
}

type GitRemoteMutationPreparedState struct {
	SchemaID                     string                          `json:"schema_id"`
	SchemaVersion                string                          `json:"schema_version"`
	PreparedMutationID           string                          `json:"prepared_mutation_id"`
	RunID                        string                          `json:"run_id"`
	Provider                     string                          `json:"provider"`
	DestinationRef               string                          `json:"destination_ref"`
	RequestKind                  string                          `json:"request_kind"`
	TypedRequestSchemaID         string                          `json:"typed_request_schema_id"`
	TypedRequestSchemaVersion    string                          `json:"typed_request_schema_version"`
	TypedRequest                 map[string]any                  `json:"typed_request"`
	TypedRequestHash             trustpolicy.Digest              `json:"typed_request_hash"`
	ActionRequestHash            trustpolicy.Digest              `json:"action_request_hash"`
	PolicyDecisionHash           trustpolicy.Digest              `json:"policy_decision_hash"`
	RequiredApprovalID           string                          `json:"required_approval_id,omitempty"`
	RequiredApprovalRequestHash  *trustpolicy.Digest             `json:"required_approval_request_hash,omitempty"`
	RequiredApprovalDecisionHash *trustpolicy.Digest             `json:"required_approval_decision_hash,omitempty"`
	LifecycleState               string                          `json:"lifecycle_state"`
	LifecycleReasonCode          string                          `json:"lifecycle_reason_code,omitempty"`
	ExecutionState               string                          `json:"execution_state"`
	ExecutionReasonCode          string                          `json:"execution_reason_code,omitempty"`
	CreatedAt                    string                          `json:"created_at"`
	UpdatedAt                    string                          `json:"updated_at"`
	DerivedSummary               GitRemoteMutationDerivedSummary `json:"derived_summary"`
	LastPrepareRequestID         string                          `json:"last_prepare_request_id,omitempty"`
	LastGetRequestID             string                          `json:"last_get_request_id,omitempty"`
	LastExecuteRequestID         string                          `json:"last_execute_request_id,omitempty"`
	LastExecuteProviderLeaseID   string                          `json:"last_execute_provider_auth_lease_id,omitempty"`
	LastExecuteAttemptID         string                          `json:"last_execute_attempt_id,omitempty"`
	LastExecuteAttemptRequestID  *trustpolicy.Digest             `json:"last_execute_attempt_typed_request_hash,omitempty"`
	LastExecuteSnapshotSegmentID string                          `json:"last_execute_snapshot_segment_id,omitempty"`
	LastExecuteSnapshotSealID    *trustpolicy.Digest             `json:"last_execute_snapshot_seal_digest,omitempty"`
}

type GitRemoteMutationPrepareRequest struct {
	SchemaID       string         `json:"schema_id"`
	SchemaVersion  string         `json:"schema_version"`
	RequestID      string         `json:"request_id"`
	RunID          string         `json:"run_id"`
	Provider       string         `json:"provider"`
	DestinationRef string         `json:"destination_ref,omitempty"`
	TypedRequest   map[string]any `json:"typed_request"`
}

type GitRemoteMutationPrepareResponse struct {
	SchemaID           string                         `json:"schema_id"`
	SchemaVersion      string                         `json:"schema_version"`
	RequestID          string                         `json:"request_id"`
	PreparedMutationID string                         `json:"prepared_mutation_id"`
	TypedRequestHash   trustpolicy.Digest             `json:"typed_request_hash"`
	Prepared           GitRemoteMutationPreparedState `json:"prepared"`
}

type GitRemoteMutationGetRequest struct {
	SchemaID           string `json:"schema_id"`
	SchemaVersion      string `json:"schema_version"`
	RequestID          string `json:"request_id"`
	PreparedMutationID string `json:"prepared_mutation_id"`
}

type GitRemoteMutationGetResponse struct {
	SchemaID      string                         `json:"schema_id"`
	SchemaVersion string                         `json:"schema_version"`
	RequestID     string                         `json:"request_id"`
	Prepared      GitRemoteMutationPreparedState `json:"prepared"`
}

type GitRemoteMutationExecuteRequest struct {
	SchemaID             string             `json:"schema_id"`
	SchemaVersion        string             `json:"schema_version"`
	RequestID            string             `json:"request_id"`
	PreparedMutationID   string             `json:"prepared_mutation_id"`
	ApprovalID           string             `json:"approval_id"`
	ApprovalRequestHash  trustpolicy.Digest `json:"approval_request_hash"`
	ApprovalDecisionHash trustpolicy.Digest `json:"approval_decision_hash"`
	ProviderAuthLeaseID  string             `json:"provider_auth_lease_id"`
}

type GitRemoteMutationExecuteResponse struct {
	SchemaID           string                         `json:"schema_id"`
	SchemaVersion      string                         `json:"schema_version"`
	RequestID          string                         `json:"request_id"`
	PreparedMutationID string                         `json:"prepared_mutation_id"`
	ExecutionState     string                         `json:"execution_state"`
	Prepared           GitRemoteMutationPreparedState `json:"prepared"`
}
