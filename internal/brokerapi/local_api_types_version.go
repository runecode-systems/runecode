package brokerapi

type BrokerVersionInfo struct {
	SchemaID                        string                          `json:"schema_id"`
	SchemaVersion                   string                          `json:"schema_version"`
	ProductVersion                  string                          `json:"product_version"`
	BuildRevision                   string                          `json:"build_revision"`
	BuildTime                       string                          `json:"build_time"`
	ProtocolBundleVersion           string                          `json:"protocol_bundle_version"`
	ProtocolBundleManifestHash      string                          `json:"protocol_bundle_manifest_hash"`
	APIFamily                       string                          `json:"api_family"`
	APIVersion                      string                          `json:"api_version"`
	SupportedTransportEncodings     []string                        `json:"supported_transport_encodings"`
	ProjectSubstrateContractID      string                          `json:"project_substrate_contract_id,omitempty"`
	ProjectSubstrateContractVersion string                          `json:"project_substrate_contract_version,omitempty"`
	ProjectSubstrateVersion         string                          `json:"project_substrate_version,omitempty"`
	ProjectSubstrateValidationState string                          `json:"project_substrate_validation_state,omitempty"`
	ProjectSubstratePosture         string                          `json:"project_substrate_posture,omitempty"`
	ProjectSubstrateBlockedReasons  []string                        `json:"project_substrate_blocked_reasons,omitempty"`
	ProjectSubstrateSupportedMin    string                          `json:"project_substrate_supported_min,omitempty"`
	ProjectSubstrateSupportedMax    string                          `json:"project_substrate_supported_max,omitempty"`
	ProjectSubstrateRecommended     string                          `json:"project_substrate_recommended,omitempty"`
	ProjectContextIdentityDigest    string                          `json:"project_context_identity_digest,omitempty"`
	ProjectSubstratePostureSummary  *ProjectSubstratePostureSummary `json:"project_substrate_posture_summary,omitempty"`
}

type VersionInfoGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type VersionInfoGetResponse struct {
	SchemaID      string            `json:"schema_id"`
	SchemaVersion string            `json:"schema_version"`
	RequestID     string            `json:"request_id"`
	VersionInfo   BrokerVersionInfo `json:"version_info"`
}
