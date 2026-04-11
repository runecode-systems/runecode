package policyengine

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
	SchemaID        string `json:"schema_id"`
	SchemaVersion   string `json:"schema_version"`
	GatewayRoleKind string `json:"gateway_role_kind"`
	DestinationKind string `json:"destination_kind"`
	DestinationRef  string `json:"destination_ref"`
	EgressDataClass string `json:"egress_data_class"`
	Operation       string `json:"operation,omitempty"`
	PayloadHash     string `json:"payload_hash,omitempty"`
}

type backendPosturePayload struct {
	SchemaID         string `json:"schema_id"`
	SchemaVersion    string `json:"schema_version"`
	BackendClass     string `json:"backend_class"`
	ChangeKind       string `json:"change_kind"`
	RequestedPosture string `json:"requested_posture"`
	RequiresOptIn    bool   `json:"requires_opt_in"`
}

type promotionPayload struct {
	SchemaID            string `json:"schema_id"`
	SchemaVersion       string `json:"schema_version"`
	PromotionKind       string `json:"promotion_kind"`
	TargetDataClass     string `json:"target_data_class"`
	AuthoritativeImport bool   `json:"authoritative_import,omitempty"`
}
