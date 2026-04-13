package brokerapi

type BrokerReadiness struct {
	SchemaID                  string                         `json:"schema_id"`
	SchemaVersion             string                         `json:"schema_version"`
	Ready                     bool                           `json:"ready"`
	LocalOnly                 bool                           `json:"local_only"`
	ConsumptionChannel        string                         `json:"consumption_channel"`
	RecoveryComplete          bool                           `json:"recovery_complete"`
	AppendPositionStable      bool                           `json:"append_position_stable"`
	CurrentSegmentWritable    bool                           `json:"current_segment_writable"`
	VerifierMaterialAvailable bool                           `json:"verifier_material_available"`
	DerivedIndexCaughtUp      bool                           `json:"derived_index_caught_up"`
	SecretsReady              bool                           `json:"secrets_ready"`
	SecretsHealthState        string                         `json:"secrets_health_state,omitempty"`
	SecretsOperationalMetrics *SecretsOperationalMetrics     `json:"secrets_operational_metrics,omitempty"`
	SecretsStoragePosture     *SecretStoragePosture          `json:"secrets_storage_posture,omitempty"`
	ModelGatewayReady         bool                           `json:"model_gateway_ready"`
	ModelGatewayHealthState   string                         `json:"model_gateway_health_state,omitempty"`
	ModelGatewayPosture       *ModelGatewayPostureProjection `json:"model_gateway_posture_projection,omitempty"`
}

type SecretsOperationalMetrics struct {
	LeaseIssueCount  int `json:"lease_issue_count"`
	LeaseRenewCount  int `json:"lease_renew_count"`
	LeaseRevokeCount int `json:"lease_revoke_count"`
	LeaseDeniedCount int `json:"lease_denied_count"`
	ActiveLeaseCount int `json:"active_lease_count"`
}

type SecretStoragePosture struct {
	SchemaID                               string                          `json:"schema_id"`
	SchemaVersion                          string                          `json:"schema_version"`
	LongLivedSecretStoreAuthority          string                          `json:"long_lived_secret_store_authority"`
	SecureStoragePreference                string                          `json:"secure_storage_preference"`
	SecureStorageAvailable                 bool                            `json:"secure_storage_available"`
	FailClosedWhenSecureStorageUnavailable bool                            `json:"fail_closed_when_secure_storage_unavailable"`
	EffectiveCustodyPosture                string                          `json:"effective_custody_posture"`
	PortablePassphraseOptIn                SecretStoragePortableOptIn      `json:"portable_passphrase_opt_in"`
	DurableState                           SecretStorageDurableState       `json:"durable_state"`
	OnboardingContract                     SecretStorageOnboardingContract `json:"onboarding_contract"`
}

type SecretStoragePortableOptIn struct {
	Enabled bool `json:"enabled"`
}

type SecretStorageDurableState struct {
	SecretMetadata  SecretStorageSecretMetadata  `json:"secret_metadata"`
	LeaseState      SecretStorageLeaseState      `json:"lease_state"`
	RevocationState SecretStorageRevocationState `json:"revocation_state"`
	LinkageMetadata SecretStorageLinkageMetadata `json:"linkage_metadata"`
}

type SecretStorageSecretMetadata struct {
	RecordCount              int    `json:"record_count"`
	EncryptedMaterialPresent bool   `json:"encrypted_material_present"`
	LastUpdatedAt            string `json:"last_updated_at"`
}

type SecretStorageLeaseState struct {
	ActiveLeaseCount  int    `json:"active_lease_count"`
	ExpiredLeaseCount int    `json:"expired_lease_count"`
	LastRecoveredAt   string `json:"last_recovered_at"`
}

type SecretStorageRevocationState struct {
	RevokedLeaseCount        int  `json:"revoked_lease_count"`
	RevocationIndexPersisted bool `json:"revocation_index_persisted"`
}

type SecretStorageLinkageMetadata struct {
	PolicyBindingHashCount int `json:"policy_binding_hash_count"`
	AuditLinkHashCount     int `json:"audit_link_hash_count"`
}

type SecretStorageOnboardingContract struct {
	CanonicalPortableSource   string   `json:"canonical_portable_source"`
	SupportedSources          []string `json:"supported_sources"`
	FileDescriptorSupport     string   `json:"file_descriptor_support"`
	AuditMetadataOnly         bool     `json:"audit_metadata_only"`
	AuditIncludesSecretValues bool     `json:"audit_includes_secret_values"`
}

type ModelGatewayPostureProjection struct {
	SchemaID             string `json:"schema_id"`
	SchemaVersion        string `json:"schema_version"`
	ProjectionKind       string `json:"projection_kind"`
	GatewayRoleKind      string `json:"gateway_role_kind"`
	DestinationScopeKind string `json:"destination_scope_kind"`
	ConfigurationState   string `json:"configuration_state"`
	EgressPolicyPosture  string `json:"egress_policy_posture"`
	SurfaceChannel       string `json:"surface_channel"`
}

type ReadinessGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ReadinessGetResponse struct {
	SchemaID      string          `json:"schema_id"`
	SchemaVersion string          `json:"schema_version"`
	RequestID     string          `json:"request_id"`
	Readiness     BrokerReadiness `json:"readiness"`
}
