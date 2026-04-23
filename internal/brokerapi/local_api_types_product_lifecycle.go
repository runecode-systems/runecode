package brokerapi

type ProductLifecyclePostureGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ProductLifecyclePostureGetResponse struct {
	SchemaID         string                        `json:"schema_id"`
	SchemaVersion    string                        `json:"schema_version"`
	RequestID        string                        `json:"request_id"`
	ProductLifecycle BrokerProductLifecyclePosture `json:"product_lifecycle"`
}

type BrokerProductLifecyclePosture struct {
	SchemaID                     string   `json:"schema_id"`
	SchemaVersion                string   `json:"schema_version"`
	ProductInstanceID            string   `json:"product_instance_id"`
	LifecycleGeneration          string   `json:"lifecycle_generation"`
	AttachMode                   string   `json:"attach_mode"`
	LifecyclePosture             string   `json:"lifecycle_posture"`
	Attachable                   bool     `json:"attachable"`
	NormalOperationAllowed       bool     `json:"normal_operation_allowed"`
	BlockedReasonCodes           []string `json:"blocked_reason_codes,omitempty"`
	DegradedReasonCodes          []string `json:"degraded_reason_codes,omitempty"`
	RepositoryRoot               string   `json:"repository_root"`
	ProjectContextIdentityDigest string   `json:"project_context_identity_digest,omitempty"`
	ActiveSessionCount           int      `json:"active_session_count"`
	ActiveRunCount               int      `json:"active_run_count"`
}
