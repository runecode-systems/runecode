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
