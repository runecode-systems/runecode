package brokerapi

import (
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type gatewayAuditContextPayload struct {
	OutboundBytes      int64               `json:"outbound_bytes"`
	StartedAt          string              `json:"started_at"`
	CompletedAt        string              `json:"completed_at"`
	Outcome            string              `json:"outcome"`
	RequestHash        *trustpolicy.Digest `json:"request_hash,omitempty"`
	ResponseHash       *trustpolicy.Digest `json:"response_hash,omitempty"`
	LeaseID            string              `json:"lease_id,omitempty"`
	PolicyDecisionHash *trustpolicy.Digest `json:"policy_decision_hash,omitempty"`
}

type gitRequestSummaryPayload struct {
	SchemaID                       string                             `json:"schema_id"`
	SchemaVersion                  string                             `json:"schema_version"`
	RequestKind                    string                             `json:"request_kind"`
	RepositoryIdentity             policyengine.DestinationDescriptor `json:"repository_identity"`
	TargetRefs                     []string                           `json:"target_refs"`
	ReferencedPatchArtifactDigests []trustpolicy.Digest               `json:"referenced_patch_artifact_digests"`
	ExpectedResultTreeHash         trustpolicy.Digest                 `json:"expected_result_tree_hash"`
}

type gitRuntimeProofPayload struct {
	SchemaID               string               `json:"schema_id"`
	SchemaVersion          string               `json:"schema_version"`
	TypedRequestHash       trustpolicy.Digest   `json:"typed_request_hash"`
	PatchArtifactDigests   []trustpolicy.Digest `json:"patch_artifact_digests,omitempty"`
	ExpectedOldObjectID    string               `json:"expected_old_object_id"`
	ObservedOldObjectID    string               `json:"observed_old_object_id"`
	ExpectedResultTreeHash trustpolicy.Digest   `json:"expected_result_tree_hash"`
	ObservedResultTreeHash trustpolicy.Digest   `json:"observed_result_tree_hash"`
	SparseCheckoutApplied  bool                 `json:"sparse_checkout_applied"`
	DriftDetected          bool                 `json:"drift_detected"`
	DestructiveRefMutation bool                 `json:"destructive_ref_mutation"`
	ProviderKind           string               `json:"provider_kind,omitempty"`
	PullRequestNumber      *int64               `json:"pull_request_number,omitempty"`
	PullRequestURL         string               `json:"pull_request_url,omitempty"`
	EvidenceRefs           []string             `json:"evidence_refs,omitempty"`
}

type gatewayQuotaContextPayload struct {
	QuotaProfileKind    string                    `json:"quota_profile_kind"`
	Phase               string                    `json:"phase"`
	EnforceDuringStream bool                      `json:"enforce_during_stream"`
	StreamLimitBytes    *int64                    `json:"stream_limit_bytes,omitempty"`
	Meters              gatewayQuotaMetersPayload `json:"meters"`
}

type gatewayQuotaMetersPayload struct {
	RequestUnits     *int64 `json:"request_units,omitempty"`
	InputTokens      *int64 `json:"input_tokens,omitempty"`
	OutputTokens     *int64 `json:"output_tokens,omitempty"`
	StreamedBytes    *int64 `json:"streamed_bytes,omitempty"`
	ConcurrencyUnits *int64 `json:"concurrency_units,omitempty"`
	SpendMicros      *int64 `json:"spend_micros,omitempty"`
	EntitlementUnits *int64 `json:"entitlement_units,omitempty"`
}

type gatewayAllowlistMatch struct {
	AllowlistRef string
	EntryID      string
}
