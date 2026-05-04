package trustpolicy

import "encoding/json"

type auditReceiptPayloadStrict struct {
	SchemaID             string          `json:"schema_id"`
	SchemaVersion        string          `json:"schema_version"`
	SubjectDigest        Digest          `json:"subject_digest"`
	AuditReceiptKind     string          `json:"audit_receipt_kind"`
	SubjectFamily        string          `json:"subject_family,omitempty"`
	Recorder             json.RawMessage `json:"recorder"`
	RecordedAt           string          `json:"recorded_at"`
	ReceiptPayloadSchema string          `json:"receipt_payload_schema_id,omitempty"`
	ReceiptPayload       json.RawMessage `json:"receipt_payload,omitempty"`
}

type importRestoreReceiptPayload struct {
	ProvenanceAction      string                     `json:"provenance_action"`
	SegmentFileHashScope  string                     `json:"segment_file_hash_scope"`
	ImportedSegments      []importRestoreSegmentLink `json:"imported_segments"`
	SourceManifestDigests []Digest                   `json:"source_manifest_digests"`
	SourceInstanceID      string                     `json:"source_instance_id,omitempty"`
	Operator              *PrincipalIdentity         `json:"operator,omitempty"`
	AuthorityContext      *importRestoreAuthority    `json:"authority_context,omitempty"`
}

type importRestoreAuthority struct {
	AuthorityKind               string  `json:"authority_kind"`
	AuthorityID                 string  `json:"authority_id"`
	AuthorizationManifestDigest *Digest `json:"authorization_manifest_digest,omitempty"`
	Note                        string  `json:"note,omitempty"`
}

type importRestoreSegmentLink struct {
	ImportedSegmentSealDigest Digest `json:"imported_segment_seal_digest"`
	ImportedSegmentRoot       Digest `json:"imported_segment_root"`
	SourceSegmentFileHash     Digest `json:"source_segment_file_hash"`
	LocalSegmentFileHash      Digest `json:"local_segment_file_hash"`
	ByteIdentityVerified      bool   `json:"byte_identity_verified"`
}

type anchorReceiptPayload struct {
	AnchorKind           string                 `json:"anchor_kind"`
	KeyProtectionPosture string                 `json:"key_protection_posture,omitempty"`
	PresenceMode         string                 `json:"presence_mode,omitempty"`
	ApprovalAssurance    string                 `json:"approval_assurance_level,omitempty"`
	ApprovalDecision     *Digest                `json:"approval_decision_digest,omitempty"`
	AnchorWitness        *anchorReceiptWitness  `json:"anchor_witness,omitempty"`
	ExternalAnchor       *anchorExternalPayload `json:"external_anchor,omitempty"`
}

type anchorReceiptWitness struct {
	WitnessKind   string `json:"witness_kind"`
	WitnessDigest Digest `json:"witness_digest"`
}

type anchorExternalPayload struct {
	TargetKind             string                 `json:"target_kind"`
	RuntimeAdapter         string                 `json:"runtime_adapter"`
	TargetDescriptor       json.RawMessage        `json:"target_descriptor"`
	TargetDescriptorDigest Digest                 `json:"target_descriptor_digest"`
	Proof                  anchorExternalProofRef `json:"proof"`
	DerivedExecution       json.RawMessage        `json:"derived_execution,omitempty"`
}

type anchorExternalProofRef struct {
	ProofKind     string `json:"proof_kind"`
	ProofSchemaID string `json:"proof_schema_id"`
	ProofDigest   Digest `json:"proof_digest"`
}

type transparencyLogTargetDescriptor struct {
	DescriptorSchemaID string `json:"descriptor_schema_id"`
	LogID              string `json:"log_id"`
	LogPublicKeyDigest Digest `json:"log_public_key_digest"`
	EntryEncoding      string `json:"entry_encoding_profile"`
}

type timestampAuthorityTargetDescriptor struct {
	DescriptorSchemaID     string `json:"descriptor_schema_id"`
	AuthorityID            string `json:"authority_id"`
	CertificateChainDigest Digest `json:"certificate_chain_digest"`
	TimestampProfile       string `json:"timestamp_profile"`
}

type publicChainTargetDescriptor struct {
	DescriptorSchemaID       string `json:"descriptor_schema_id"`
	ChainNamespace           string `json:"chain_namespace"`
	NetworkID                string `json:"network_id"`
	SettlementContractDigest Digest `json:"settlement_contract_digest"`
}

type transparencyLogDerivedExecution struct {
	SubmitEndpointURI string `json:"submit_endpoint_uri"`
}

type timestampAuthorityDerivedExecution struct {
	TSAEndpointURI string `json:"tsa_endpoint_uri"`
}

type publicChainDerivedExecution struct {
	RPCEndpointURI string `json:"rpc_endpoint_uri"`
}

type providerInvocationReceiptPayload struct {
	AuthorizationOutcome string                  `json:"authorization_outcome"`
	ProviderKind         string                  `json:"provider_kind"`
	ProviderProfileID    string                  `json:"provider_profile_id,omitempty"`
	ModelID              string                  `json:"model_id,omitempty"`
	EndpointIdentity     string                  `json:"endpoint_identity,omitempty"`
	GatewayRoleKind      string                  `json:"gateway_role_kind"`
	DestinationKind      string                  `json:"destination_kind"`
	Operation            string                  `json:"operation"`
	DecisionReasonCode   string                  `json:"decision_reason_code,omitempty"`
	DecisionReasonDetail string                  `json:"decision_reason_detail,omitempty"`
	RequestDigest        *Digest                 `json:"request_digest,omitempty"`
	ResponseDigest       *Digest                 `json:"response_digest,omitempty"`
	PayloadDigest        *Digest                 `json:"payload_digest,omitempty"`
	RequestPayloadBound  *bool                   `json:"request_payload_digest_bound,omitempty"`
	NetworkTarget        networkTargetDescriptor `json:"network_target"`
	NetworkTargetDigest  Digest                  `json:"network_target_digest"`
	PolicyDecisionDigest *Digest                 `json:"policy_decision_digest,omitempty"`
	AllowlistRefDigest   *Digest                 `json:"allowlist_ref_digest,omitempty"`
	AllowlistEntryID     string                  `json:"allowlist_entry_id,omitempty"`
	LeaseIDDigest        *Digest                 `json:"lease_id_digest,omitempty"`
	RunIDDigest          *Digest                 `json:"run_id_digest,omitempty"`
}

type networkTargetDescriptor struct {
	DescriptorSchemaID string `json:"descriptor_schema_id"`
	DestinationKind    string `json:"destination_kind"`
	Host               string `json:"host,omitempty"`
	Port               *int   `json:"port,omitempty"`
	PathPrefix         string `json:"path_prefix,omitempty"`
	DestinationRef     string `json:"destination_ref,omitempty"`
}

type secretLeaseReceiptPayload struct {
	LeaseAction         string  `json:"lease_action"`
	LeaseIDDigest       Digest  `json:"lease_id_digest"`
	SecretRefDigest     Digest  `json:"secret_ref_digest"`
	ConsumerIDDigest    Digest  `json:"consumer_id_digest"`
	RoleKind            string  `json:"role_kind"`
	ScopeDigest         Digest  `json:"scope_digest"`
	DeliveryKind        string  `json:"delivery_kind,omitempty"`
	IssuedAt            string  `json:"issued_at,omitempty"`
	RevokedAt           string  `json:"revoked_at,omitempty"`
	ReasonDigest        *Digest `json:"reason_digest,omitempty"`
	RepositoryIDDigest  *Digest `json:"repository_identity_digest,omitempty"`
	ActionRequestDigest *Digest `json:"action_request_digest,omitempty"`
	PolicyContextDigest *Digest `json:"policy_context_digest,omitempty"`
	RunIDDigest         *Digest `json:"run_id_digest,omitempty"`
}

type runtimeSummaryReceiptPayload struct {
	SummaryScopeKind          string `json:"summary_scope_kind"`
	ProviderInvocationCount   int64  `json:"provider_invocation_count"`
	SecretLeaseIssueCount     int64  `json:"secret_lease_issue_count"`
	SecretLeaseRevokeCount    int64  `json:"secret_lease_revoke_count"`
	NetworkEgressCount        int64  `json:"network_egress_count"`
	NoProviderInvocation      bool   `json:"no_provider_invocation"`
	NoSecretLeaseIssued       bool   `json:"no_secret_lease_issued"`
	ApprovalConsumptionCount  int64  `json:"approval_consumption_count,omitempty"`
	NoApprovalConsumed        bool   `json:"no_approval_consumed,omitempty"`
	BoundaryCrossingCount     int64  `json:"boundary_crossing_count,omitempty"`
	NoArtifactCrossedBoundary bool   `json:"no_artifact_crossed_boundary,omitempty"`
	BoundaryRoute             string `json:"boundary_route,omitempty"`
	BoundaryCrossingSupport   string `json:"boundary_crossing_support,omitempty"`
}

type degradedPostureSummaryReceiptPayload struct {
	SummaryScopeKind          string   `json:"summary_scope_kind"`
	Degraded                  bool     `json:"degraded"`
	DegradationCauseCode      string   `json:"degradation_cause_code"`
	DegradationReasonCodes    []string `json:"degradation_reason_codes,omitempty"`
	TrustClaimBefore          string   `json:"trust_claim_before"`
	TrustClaimAfter           string   `json:"trust_claim_after"`
	ChangedTrustClaim         bool     `json:"changed_trust_claim"`
	UserAcknowledged          bool     `json:"user_acknowledged"`
	AcknowledgmentEvidence    string   `json:"acknowledgment_evidence,omitempty"`
	ApprovalRequired          bool     `json:"approval_required"`
	ApprovalConsumed          bool     `json:"approval_consumed"`
	OverrideRequired          bool     `json:"override_required"`
	OverrideApplied           bool     `json:"override_applied"`
	ApprovalPolicyDecisionRef string   `json:"approval_policy_decision_ref,omitempty"`
	ApprovalConsumptionLink   string   `json:"approval_consumption_link,omitempty"`
	OverridePolicyDecisionRef string   `json:"override_policy_decision_ref,omitempty"`
	OverrideActionRequestHash string   `json:"override_action_request_hash,omitempty"`
	RunLifecycleState         string   `json:"run_lifecycle_state,omitempty"`
}

type negativeCapabilitySummaryReceiptPayload struct {
	SummaryScopeKind                   string `json:"summary_scope_kind"`
	NoSecretLeaseIssued                bool   `json:"no_secret_lease_issued"`
	NoNetworkEgress                    bool   `json:"no_network_egress"`
	NoApprovalConsumed                 bool   `json:"no_approval_consumed"`
	NoArtifactCrossedBoundary          bool   `json:"no_artifact_crossed_boundary"`
	BoundaryRoute                      string `json:"boundary_route,omitempty"`
	SecretLeaseEvidenceSupport         string `json:"secret_lease_evidence_support"`
	NetworkEgressEvidenceSupport       string `json:"network_egress_evidence_support"`
	ApprovalConsumptionEvidenceSupport string `json:"approval_consumption_evidence_support"`
	BoundaryCrossingEvidenceSupport    string `json:"boundary_crossing_evidence_support"`
}

type metaAuditActionReceiptPayload struct {
	ActionCode         string             `json:"action_code"`
	ActionFamily       string             `json:"action_family"`
	ScopeKind          string             `json:"scope_kind"`
	ScopeRefDigest     *Digest            `json:"scope_ref_digest,omitempty"`
	Result             string             `json:"result"`
	ManifestDigest     *Digest            `json:"manifest_digest,omitempty"`
	ObjectDigest       *Digest            `json:"object_digest,omitempty"`
	Operator           *PrincipalIdentity `json:"operator,omitempty"`
	ExternalSystemRef  string             `json:"external_system_ref,omitempty"`
	SensitiveViewClass string             `json:"sensitive_view_class,omitempty"`
}

type approvalEvidenceReceiptPayload struct {
	ApprovalID            string             `json:"approval_id"`
	ApprovalStatus        string             `json:"approval_status"`
	ResolutionReasonCode  string             `json:"resolution_reason_code,omitempty"`
	RequestDigest         Digest             `json:"request_digest"`
	DecisionDigest        *Digest            `json:"decision_digest,omitempty"`
	ScopeDigest           *Digest            `json:"scope_digest,omitempty"`
	ArtifactSetDigest     *Digest            `json:"artifact_set_digest,omitempty"`
	DiffDigest            *Digest            `json:"diff_digest,omitempty"`
	SummaryPreviewDigest  *Digest            `json:"summary_preview_digest,omitempty"`
	ConsumptionLinkDigest *Digest            `json:"consumption_link_digest,omitempty"`
	PolicyDecisionDigest  *Digest            `json:"policy_decision_digest,omitempty"`
	Approver              *PrincipalIdentity `json:"approver,omitempty"`
	RunIDDigest           *Digest            `json:"run_id_digest,omitempty"`
	ActionKind            string             `json:"action_kind,omitempty"`
	RecordedFrom          string             `json:"recorded_from,omitempty"`
}

type publicationEvidenceReceiptPayload struct {
	PublicationKind        string  `json:"publication_kind"`
	ArtifactDigest         Digest  `json:"artifact_digest"`
	SourceArtifactDigest   *Digest `json:"source_artifact_digest,omitempty"`
	ApprovalDecisionDigest *Digest `json:"approval_decision_digest,omitempty"`
	ApprovalLinkDigest     *Digest `json:"approval_link_digest,omitempty"`
	RunIDDigest            *Digest `json:"run_id_digest,omitempty"`
	ActionKind             string  `json:"action_kind,omitempty"`
}

type overrideEvidenceReceiptPayload struct {
	OverrideKind         string  `json:"override_kind"`
	PolicyDecisionDigest *Digest `json:"policy_decision_digest,omitempty"`
	ActionRequestDigest  *Digest `json:"action_request_digest,omitempty"`
	ApprovalLinkDigest   *Digest `json:"approval_link_digest,omitempty"`
	RunIDDigest          *Digest `json:"run_id_digest,omitempty"`
	ApprovalRequired     bool    `json:"approval_required"`
	ApprovalConsumed     bool    `json:"approval_consumed"`
}
