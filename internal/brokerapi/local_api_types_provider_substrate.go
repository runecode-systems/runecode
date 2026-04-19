package brokerapi

import "github.com/runecode-ai/runecode/internal/policyengine"

type ProviderAuthMaterial struct {
	SchemaID         string `json:"schema_id"`
	SchemaVersion    string `json:"schema_version"`
	MaterialKind     string `json:"material_kind"`
	MaterialState    string `json:"material_state"`
	SecretRef        string `json:"secret_ref,omitempty"`
	LeasePolicyRef   string `json:"lease_policy_ref,omitempty"`
	SessionBindingID string `json:"session_binding_id,omitempty"`
	LastRotatedAt    string `json:"last_rotated_at,omitempty"`
}

type ProviderReadinessPosture struct {
	SchemaID            string   `json:"schema_id"`
	SchemaVersion       string   `json:"schema_version"`
	ConfigurationState  string   `json:"configuration_state"`
	CredentialState     string   `json:"credential_state"`
	ConnectivityState   string   `json:"connectivity_state"`
	CompatibilityState  string   `json:"compatibility_state"`
	EffectiveReadiness  string   `json:"effective_readiness"`
	ReasonCodes         []string `json:"reason_codes,omitempty"`
	LastValidationAt    string   `json:"last_validation_at,omitempty"`
	ValidationAttemptID string   `json:"validation_attempt_id,omitempty"`
}

type ProviderModelCatalogPosture struct {
	SchemaID                  string   `json:"schema_id"`
	SchemaVersion             string   `json:"schema_version"`
	SelectionAuthority        string   `json:"selection_authority"`
	DiscoveryPosture          string   `json:"discovery_posture"`
	CompatibilityProbePosture string   `json:"compatibility_probe_posture"`
	DiscoveredModelIDs        []string `json:"discovered_model_ids,omitempty"`
	ProbeCompatibleModelIDs   []string `json:"probe_compatible_model_ids,omitempty"`
	LastDiscoveryAt           string   `json:"last_discovery_at,omitempty"`
	LastProbeAt               string   `json:"last_probe_at,omitempty"`
}

type ProviderLifecycleMetadata struct {
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
	LastValidationAt        string `json:"last_validation_at,omitempty"`
	ValidationAttemptCount  int64  `json:"validation_attempt_count"`
	LastValidationSucceeded bool   `json:"last_validation_succeeded"`
}

type ProviderProfile struct {
	SchemaID             string                             `json:"schema_id"`
	SchemaVersion        string                             `json:"schema_version"`
	ProviderProfileID    string                             `json:"provider_profile_id"`
	DisplayLabel         string                             `json:"display_label"`
	ProviderFamily       string                             `json:"provider_family"`
	AdapterKind          string                             `json:"adapter_kind"`
	DestinationIdentity  policyengine.DestinationDescriptor `json:"destination_identity"`
	DestinationRef       string                             `json:"destination_ref"`
	SupportedAuthModes   []string                           `json:"supported_auth_modes"`
	CurrentAuthMode      string                             `json:"current_auth_mode"`
	AllowlistedModelIDs  []string                           `json:"allowlisted_model_ids"`
	ModelCatalogPosture  ProviderModelCatalogPosture        `json:"model_catalog_posture"`
	CompatibilityPosture string                             `json:"compatibility_posture"`
	QuotaProfileKind     string                             `json:"quota_profile_kind"`
	RequestBindingKind   string                             `json:"request_binding_kind"`
	SurfaceChannel       string                             `json:"surface_channel"`
	AuthMaterial         ProviderAuthMaterial               `json:"auth_material"`
	ReadinessPosture     ProviderReadinessPosture           `json:"readiness_posture"`
	Lifecycle            ProviderLifecycleMetadata          `json:"lifecycle"`
}

func (p ProviderProfile) projected() ProviderProfile {
	p.AuthMaterial = p.AuthMaterial.projected()
	return p
}

func (m ProviderAuthMaterial) projected() ProviderAuthMaterial {
	m.SecretRef = ""
	m.LeasePolicyRef = ""
	m.SessionBindingID = ""
	return m
}
