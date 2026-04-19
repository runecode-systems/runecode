package brokerapi

import "github.com/runecode-ai/runecode/internal/secretsd"

type ProviderSetupSession struct {
	SchemaID            string   `json:"schema_id"`
	SchemaVersion       string   `json:"schema_version"`
	SetupSessionID      string   `json:"setup_session_id"`
	ProviderProfileID   string   `json:"provider_profile_id"`
	ProviderFamily      string   `json:"provider_family"`
	SupportedAuthModes  []string `json:"supported_auth_modes"`
	CurrentPhase        string   `json:"current_phase"`
	CurrentAuthMode     string   `json:"current_auth_mode"`
	ValidationStatus    string   `json:"validation_status"`
	ValidationAttemptID string   `json:"validation_attempt_id,omitempty"`
	ReadinessCommitted  bool     `json:"readiness_committed"`
	SecretIngressReady  bool     `json:"secret_ingress_ready"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

type ProviderSetupSessionBeginRequest struct {
	SchemaID            string   `json:"schema_id"`
	SchemaVersion       string   `json:"schema_version"`
	RequestID           string   `json:"request_id"`
	DisplayLabel        string   `json:"display_label"`
	ProviderFamily      string   `json:"provider_family"`
	AdapterKind         string   `json:"adapter_kind"`
	CanonicalHost       string   `json:"canonical_host"`
	CanonicalPathPrefix string   `json:"canonical_path_prefix"`
	AllowlistedModelIDs []string `json:"allowlisted_model_ids,omitempty"`
}

type ProviderSetupSessionBeginResponse struct {
	SchemaID      string               `json:"schema_id"`
	SchemaVersion string               `json:"schema_version"`
	RequestID     string               `json:"request_id"`
	SetupSession  ProviderSetupSession `json:"setup_session"`
	Profile       ProviderProfile      `json:"profile"`
}

type ProviderSetupSecretIngressPrepareRequest struct {
	SchemaID        string `json:"schema_id"`
	SchemaVersion   string `json:"schema_version"`
	RequestID       string `json:"request_id"`
	SetupSessionID  string `json:"setup_session_id"`
	IngressChannel  string `json:"ingress_channel"`
	CredentialField string `json:"credential_field"`
}

type ProviderSetupSecretIngressPrepareResponse struct {
	SchemaID           string               `json:"schema_id"`
	SchemaVersion      string               `json:"schema_version"`
	RequestID          string               `json:"request_id"`
	SetupSession       ProviderSetupSession `json:"setup_session"`
	SecretIngressToken string               `json:"secret_ingress_token"`
	ExpiresAt          string               `json:"expires_at"`
}

type ProviderSetupSecretIngressSubmitRequest struct {
	SchemaID           string `json:"schema_id"`
	SchemaVersion      string `json:"schema_version"`
	RequestID          string `json:"request_id"`
	SecretIngressToken string `json:"secret_ingress_token"`
}

type ProviderSetupSecretIngressSubmitResponse struct {
	SchemaID      string               `json:"schema_id"`
	SchemaVersion string               `json:"schema_version"`
	RequestID     string               `json:"request_id"`
	SetupSession  ProviderSetupSession `json:"setup_session"`
	Profile       ProviderProfile      `json:"profile"`
}

type ProviderValidationBeginRequest struct {
	SchemaID            string `json:"schema_id"`
	SchemaVersion       string `json:"schema_version"`
	RequestID           string `json:"request_id"`
	ProviderProfileID   string `json:"provider_profile_id"`
	ValidationAttemptID string `json:"validation_attempt_id,omitempty"`
	ValidationReason    string `json:"validation_reason,omitempty"`
}

type ProviderValidationBeginResponse struct {
	SchemaID            string               `json:"schema_id"`
	SchemaVersion       string               `json:"schema_version"`
	RequestID           string               `json:"request_id"`
	ProviderProfileID   string               `json:"provider_profile_id"`
	ValidationAttemptID string               `json:"validation_attempt_id"`
	SetupSession        ProviderSetupSession `json:"setup_session"`
	Profile             ProviderProfile      `json:"profile"`
}

type ProviderValidationCommitRequest struct {
	SchemaID            string   `json:"schema_id"`
	SchemaVersion       string   `json:"schema_version"`
	RequestID           string   `json:"request_id"`
	ProviderProfileID   string   `json:"provider_profile_id"`
	ValidationAttemptID string   `json:"validation_attempt_id"`
	ConfigurationState  string   `json:"configuration_state,omitempty"`
	CredentialState     string   `json:"credential_state,omitempty"`
	ConnectivityState   string   `json:"connectivity_state"`
	CompatibilityState  string   `json:"compatibility_state"`
	ReasonCodes         []string `json:"reason_codes,omitempty"`
}

type ProviderValidationCommitResponse struct {
	SchemaID            string               `json:"schema_id"`
	SchemaVersion       string               `json:"schema_version"`
	RequestID           string               `json:"request_id"`
	ProviderProfileID   string               `json:"provider_profile_id"`
	ValidationAttemptID string               `json:"validation_attempt_id"`
	ValidationOutcome   string               `json:"validation_outcome"`
	SetupSession        ProviderSetupSession `json:"setup_session"`
	Profile             ProviderProfile      `json:"profile"`
}

type ProviderCredentialLeaseIssueRequest struct {
	SchemaID          string `json:"schema_id"`
	SchemaVersion     string `json:"schema_version"`
	RequestID         string `json:"request_id"`
	ProviderProfileID string `json:"provider_profile_id"`
	RunID             string `json:"run_id"`
	TTLSeconds        int    `json:"ttl_seconds,omitempty"`
}

type ProviderCredentialLeaseIssueResponse struct {
	SchemaID            string         `json:"schema_id"`
	SchemaVersion       string         `json:"schema_version"`
	RequestID           string         `json:"request_id"`
	ProviderProfileID   string         `json:"provider_profile_id"`
	ProviderAuthLeaseID string         `json:"provider_auth_lease_id"`
	Lease               secretsd.Lease `json:"lease"`
}

type ProviderProfileListRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ProviderProfileListResponse struct {
	SchemaID      string            `json:"schema_id"`
	SchemaVersion string            `json:"schema_version"`
	RequestID     string            `json:"request_id"`
	Profiles      []ProviderProfile `json:"profiles"`
}

type ProviderProfileGetRequest struct {
	SchemaID          string `json:"schema_id"`
	SchemaVersion     string `json:"schema_version"`
	RequestID         string `json:"request_id"`
	ProviderProfileID string `json:"provider_profile_id"`
}

type ProviderProfileGetResponse struct {
	SchemaID      string          `json:"schema_id"`
	SchemaVersion string          `json:"schema_version"`
	RequestID     string          `json:"request_id"`
	Profile       ProviderProfile `json:"profile"`
}
