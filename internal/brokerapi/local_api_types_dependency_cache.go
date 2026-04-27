package brokerapi

import (
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type DependencyFetchRequestObject struct {
	SchemaID         string                             `json:"schema_id"`
	SchemaVersion    string                             `json:"schema_version"`
	RequestKind      string                             `json:"request_kind"`
	RegistryIdentity policyengine.DestinationDescriptor `json:"registry_identity"`
	Ecosystem        string                             `json:"ecosystem"`
	PackageName      string                             `json:"package_name"`
	PackageVersion   string                             `json:"package_version"`
}

type DependencyFetchBatchRequestObject struct {
	SchemaID            string                         `json:"schema_id"`
	SchemaVersion       string                         `json:"schema_version"`
	LockfileKind        string                         `json:"lockfile_kind"`
	LockfileDigest      trustpolicy.Digest             `json:"lockfile_digest"`
	RequestSetHash      trustpolicy.Digest             `json:"request_set_hash"`
	DependencyRequests  []DependencyFetchRequestObject `json:"dependency_requests"`
	BatchRequestID      string                         `json:"batch_request_id,omitempty"`
	LockfileLocatorHint string                         `json:"lockfile_locator_hint,omitempty"`
}

type DependencyCacheEnsureRequest struct {
	SchemaID      string                            `json:"schema_id"`
	SchemaVersion string                            `json:"schema_version"`
	RequestID     string                            `json:"request_id"`
	RunID         string                            `json:"run_id"`
	BatchRequest  DependencyFetchBatchRequestObject `json:"batch_request"`
}

type DependencyCacheEnsureResponse struct {
	SchemaID             string               `json:"schema_id"`
	SchemaVersion        string               `json:"schema_version"`
	RequestID            string               `json:"request_id"`
	BatchRequestHash     trustpolicy.Digest   `json:"batch_request_hash"`
	BatchManifestDigest  trustpolicy.Digest   `json:"batch_manifest_digest"`
	ResolutionState      string               `json:"resolution_state"`
	CacheOutcome         string               `json:"cache_outcome"`
	ResolvedUnitDigests  []trustpolicy.Digest `json:"resolved_unit_digests"`
	FetchedBytes         int64                `json:"fetched_bytes"`
	RegistryRequestCount int                  `json:"registry_request_count"`
}

type DependencyFetchRegistryRequest struct {
	SchemaID          string                       `json:"schema_id"`
	SchemaVersion     string                       `json:"schema_version"`
	RequestID         string                       `json:"request_id"`
	RunID             string                       `json:"run_id"`
	DependencyRequest DependencyFetchRequestObject `json:"dependency_request"`
	RequestHash       trustpolicy.Digest           `json:"request_hash"`
}

type DependencyFetchRegistryResponse struct {
	SchemaID             string               `json:"schema_id"`
	SchemaVersion        string               `json:"schema_version"`
	RequestID            string               `json:"request_id"`
	RequestHash          trustpolicy.Digest   `json:"request_hash"`
	ResolvedUnitDigest   trustpolicy.Digest   `json:"resolved_unit_digest"`
	ManifestDigest       trustpolicy.Digest   `json:"manifest_digest"`
	PayloadDigests       []trustpolicy.Digest `json:"payload_digests"`
	CacheOutcome         string               `json:"cache_outcome"`
	FetchedBytes         int64                `json:"fetched_bytes"`
	RegistryRequestCount int                  `json:"registry_request_count"`
}

type DependencyCacheHandoffRequest struct {
	SchemaID      string             `json:"schema_id"`
	SchemaVersion string             `json:"schema_version"`
	RequestID     string             `json:"request_id"`
	RequestDigest trustpolicy.Digest `json:"request_digest"`
	ConsumerRole  string             `json:"consumer_role"`
}

type DependencyCacheHandoffMetadata struct {
	SchemaID            string               `json:"schema_id"`
	SchemaVersion       string               `json:"schema_version"`
	RequestDigest       trustpolicy.Digest   `json:"request_digest"`
	ResolvedUnitDigest  trustpolicy.Digest   `json:"resolved_unit_digest"`
	ManifestDigest      trustpolicy.Digest   `json:"manifest_digest"`
	PayloadDigests      []trustpolicy.Digest `json:"payload_digests"`
	MaterializationMode string               `json:"materialization_mode"`
	HandoffMode         string               `json:"handoff_mode"`
}

type DependencyCacheHandoffResponse struct {
	SchemaID      string                          `json:"schema_id"`
	SchemaVersion string                          `json:"schema_version"`
	RequestID     string                          `json:"request_id"`
	Found         bool                            `json:"found"`
	Handoff       *DependencyCacheHandoffMetadata `json:"handoff,omitempty"`
}
