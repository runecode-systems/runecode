package policyengine

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type executorRunPayload struct {
	SchemaID       string            `json:"schema_id"`
	SchemaVersion  string            `json:"schema_version"`
	ExecutorClass  string            `json:"executor_class"`
	ExecutorID     string            `json:"executor_id"`
	Argv           []string          `json:"argv"`
	Environment    map[string]string `json:"environment,omitempty"`
	WorkingDir     string            `json:"working_directory,omitempty"`
	NetworkAccess  string            `json:"network_access,omitempty"`
	TimeoutSeconds *int              `json:"timeout_seconds,omitempty"`
}

type gatewayEgressPayload struct {
	SchemaID        string               `json:"schema_id"`
	SchemaVersion   string               `json:"schema_version"`
	GatewayRoleKind string               `json:"gateway_role_kind"`
	DestinationKind string               `json:"destination_kind"`
	DestinationRef  string               `json:"destination_ref"`
	EgressDataClass string               `json:"egress_data_class"`
	Operation       string               `json:"operation"`
	TimeoutSeconds  *int                 `json:"timeout_seconds,omitempty"`
	PayloadHash     *trustpolicy.Digest  `json:"payload_hash,omitempty"`
	AuditContext    *gatewayAuditContext `json:"audit_context,omitempty"`
	QuotaContext    *gatewayQuotaContext `json:"quota_context,omitempty"`
	GitRequest      map[string]any       `json:"git_request,omitempty"`
	GitRuntimeProof *gitRuntimeProof     `json:"git_runtime_proof,omitempty"`
}

type gitRefUpdateRequest struct {
	SchemaID                       string                `json:"schema_id"`
	SchemaVersion                  string                `json:"schema_version"`
	RequestKind                    string                `json:"request_kind"`
	RepositoryIdentity             DestinationDescriptor `json:"repository_identity"`
	TargetRef                      string                `json:"target_ref"`
	ExpectedOldRefHash             trustpolicy.Digest    `json:"expected_old_ref_hash"`
	ReferencedPatchArtifactDigests []trustpolicy.Digest  `json:"referenced_patch_artifact_digests"`
	CommitIntent                   gitCommitIntent       `json:"commit_intent"`
	ExpectedResultTreeHash         trustpolicy.Digest    `json:"expected_result_tree_hash"`
	AllowForcePush                 bool                  `json:"allow_force_push"`
	AllowRefDeletion               bool                  `json:"allow_ref_deletion"`
	RefPurpose                     string                `json:"ref_purpose,omitempty"`
	BaseRef                        string                `json:"base_ref,omitempty"`
}

type gitPullRequestCreateRequest struct {
	SchemaID                       string                `json:"schema_id"`
	SchemaVersion                  string                `json:"schema_version"`
	RequestKind                    string                `json:"request_kind"`
	BaseRepositoryIdentity         DestinationDescriptor `json:"base_repository_identity"`
	BaseRef                        string                `json:"base_ref"`
	HeadRepositoryIdentity         DestinationDescriptor `json:"head_repository_identity"`
	HeadRef                        string                `json:"head_ref"`
	Title                          string                `json:"title"`
	Body                           string                `json:"body"`
	HeadCommitOrTreeHash           trustpolicy.Digest    `json:"head_commit_or_tree_hash"`
	ReferencedPatchArtifactDigests []trustpolicy.Digest  `json:"referenced_patch_artifact_digests"`
	ExpectedResultTreeHash         trustpolicy.Digest    `json:"expected_result_tree_hash"`
}

type gitCommitIntent struct {
	Message   gitCommitMessage   `json:"message"`
	Trailers  []gitCommitTrailer `json:"trailers"`
	Author    gitIdentity        `json:"author"`
	Committer gitIdentity        `json:"committer"`
	Signoff   gitIdentity        `json:"signoff"`
}

type gitCommitMessage struct {
	Subject string `json:"subject"`
	Body    string `json:"body,omitempty"`
}

type gitCommitTrailer struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type gitIdentity struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type gitRuntimeProof struct {
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

type gatewayAuditContext struct {
	SchemaID           string              `json:"schema_id"`
	SchemaVersion      string              `json:"schema_version"`
	OutboundBytes      int64               `json:"outbound_bytes"`
	StartedAt          string              `json:"started_at"`
	CompletedAt        string              `json:"completed_at"`
	Outcome            string              `json:"outcome"`
	RequestHash        *trustpolicy.Digest `json:"request_hash,omitempty"`
	ResponseHash       *trustpolicy.Digest `json:"response_hash,omitempty"`
	LeaseID            string              `json:"lease_id,omitempty"`
	PolicyDecisionHash *trustpolicy.Digest `json:"policy_decision_hash,omitempty"`
}

type gatewayQuotaContext struct {
	SchemaID            string             `json:"schema_id"`
	SchemaVersion       string             `json:"schema_version"`
	QuotaProfileKind    string             `json:"quota_profile_kind"`
	Phase               string             `json:"phase"`
	EnforceDuringStream bool               `json:"enforce_during_stream"`
	StreamLimitBytes    *int64             `json:"stream_limit_bytes,omitempty"`
	Meters              gatewayQuotaMeters `json:"meters"`
}

type gatewayQuotaMeters struct {
	RequestUnits     *int64 `json:"request_units,omitempty"`
	InputTokens      *int64 `json:"input_tokens,omitempty"`
	OutputTokens     *int64 `json:"output_tokens,omitempty"`
	StreamedBytes    *int64 `json:"streamed_bytes,omitempty"`
	ConcurrencyUnits *int64 `json:"concurrency_units,omitempty"`
	SpendMicros      *int64 `json:"spend_micros,omitempty"`
	EntitlementUnits *int64 `json:"entitlement_units,omitempty"`
}

type backendPosturePayload struct {
	SchemaID                     string `json:"schema_id"`
	SchemaVersion                string `json:"schema_version"`
	TargetInstanceID             string `json:"target_instance_id"`
	TargetBackendKind            string `json:"target_backend_kind"`
	SelectionMode                string `json:"selection_mode"`
	ChangeKind                   string `json:"change_kind"`
	AssuranceChangeKind          string `json:"assurance_change_kind"`
	OptInKind                    string `json:"opt_in_kind"`
	ReducedAssuranceAcknowledged bool   `json:"reduced_assurance_acknowledged"`
}

type promotionPayload struct {
	SchemaID            string `json:"schema_id"`
	SchemaVersion       string `json:"schema_version"`
	PromotionKind       string `json:"promotion_kind"`
	TargetDataClass     string `json:"target_data_class"`
	AuthoritativeImport bool   `json:"authoritative_import,omitempty"`
}
