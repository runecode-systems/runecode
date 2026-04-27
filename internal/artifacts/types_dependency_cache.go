package artifacts

import "time"

type DependencyCacheBatchRecord struct {
	BatchRequestDigest    string    `json:"batch_request_digest"`
	BatchManifestDigest   string    `json:"batch_manifest_digest"`
	ResolvedUnitDigests   []string  `json:"resolved_unit_digests"`
	LockfileDigest        string    `json:"lockfile_digest"`
	RequestSetDigest      string    `json:"request_set_digest"`
	ResolutionState       string    `json:"resolution_state"`
	CacheOutcome          string    `json:"cache_outcome"`
	MaterializationDigest []string  `json:"materialization_digests,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

type DependencyCacheResolvedUnitRecord struct {
	ResolvedUnitDigest   string    `json:"resolved_unit_digest"`
	RequestDigest        string    `json:"request_digest"`
	ManifestDigest       string    `json:"manifest_digest"`
	PayloadDigest        []string  `json:"payload_digests"`
	IntegrityState       string    `json:"integrity_state"`
	MaterializationState string    `json:"materialization_state"`
	CreatedAt            time.Time `json:"created_at"`
}

type DependencyCacheHitRequest struct {
	BatchRequestDigest string
	ResolvedUnitDigest string
	RequestDigest      string
}

type DependencyCacheHandoffRequest struct {
	RequestDigest string
	ConsumerRole  string
}

type DependencyCacheHandoff struct {
	RequestDigest       string
	ResolvedUnitDigest  string
	ManifestDigest      string
	PayloadDigests      []string
	MaterializationMode string
	HandoffMode         string
}
