package brokerapi

type BrokerVersionInfo struct {
	SchemaID                    string   `json:"schema_id"`
	SchemaVersion               string   `json:"schema_version"`
	ProductVersion              string   `json:"product_version"`
	BuildRevision               string   `json:"build_revision"`
	BuildTime                   string   `json:"build_time"`
	ProtocolBundleVersion       string   `json:"protocol_bundle_version"`
	ProtocolBundleManifestHash  string   `json:"protocol_bundle_manifest_hash"`
	APIFamily                   string   `json:"api_family"`
	APIVersion                  string   `json:"api_version"`
	SupportedTransportEncodings []string `json:"supported_transport_encodings"`
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
