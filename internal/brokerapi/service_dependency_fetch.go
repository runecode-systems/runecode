package brokerapi

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

const (
	dependencyResolvedUnitManifestSchemaPath = "objects/DependencyResolvedUnitManifest.schema.json"
	dependencyFetchBatchResultSchemaPath     = "objects/DependencyFetchBatchResult.schema.json"
)

type dependencyRegistryFetchMetadata struct {
	ContentType            string
	ExpectedPayloadDigest  string
	UpstreamManifestDigest string
}

type dependencyRegistryAuthPosture string

const (
	dependencyRegistryAuthPosturePublicNoAuth dependencyRegistryAuthPosture = "public_no_auth"
)

// dependencyRegistryAuthLease is broker-internal leased auth material.
// Implementations must represent short-lived leases only.
type dependencyRegistryAuthLease interface {
	Posture() dependencyRegistryAuthPosture
	LeaseID() string
	ExpiresAt() time.Time
}

// dependencyRegistryAuthSource is a trusted-domain-only credential source.
// Long-lived credentials remain external (for example secretsd); broker code
// only consumes short-lived lease material through this interface.
type dependencyRegistryAuthSource interface {
	AcquireLease(ctx context.Context, req DependencyFetchRequestObject) (dependencyRegistryAuthLease, error)
}

type dependencyRegistryFetcher interface {
	Fetch(ctx context.Context, req DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error)
}

type dependencyFetchService struct {
	owner      *Service
	fetcher    dependencyRegistryFetcher
	authSource dependencyRegistryAuthSource
	sem        chan struct{}
	mu         sync.Mutex
	inflight   map[string]*dependencyFetchFlight
}

type dependencyFetchFlight struct {
	done    chan struct{}
	result  dependencyUnitResolution
	err     error
	waiters int
}

type dependencyUnitResolution struct {
	unit                 artifacts.DependencyCacheResolvedUnitRecord
	requestHash          string
	resolvedUnitManifest map[string]any
	cacheOutcome         string
	fetchedBytes         int64
	registryRequests     int
	startedAt            time.Time
	completedAt          time.Time
	destinationKind      string
	destinationRef       string
	registryAuthPosture  string
	actionRequestHash    string
	policyDecisionHash   string
	matchedAllowlistRef  string
	matchedAllowlistID   string
}

type dependencyBatchEnsureSummary struct {
	units             []artifacts.DependencyCacheResolvedUnitRecord
	resolvedDigests   []string
	startedAt         time.Time
	completedAt       time.Time
	destinationKinds  []string
	destinationRefs   []string
	allowlistRefs     []string
	allowlistEntryIDs []string
	requestBindings   []any
}

func newDependencyFetchService(owner *Service, maxParallel int) *dependencyFetchService {
	if maxParallel <= 0 {
		maxParallel = 4
	}
	return &dependencyFetchService{
		owner:      owner,
		fetcher:    newPublicRegistryHTTPFetcher(),
		authSource: publicRegistryNoAuthSource{},
		sem:        make(chan struct{}, maxParallel),
		inflight:   map[string]*dependencyFetchFlight{},
	}
}

func (s *Service) SetDependencyRegistryFetcherForTests(fetcher dependencyRegistryFetcher) {
	if s == nil || s.dependencyFetchService == nil || fetcher == nil {
		return
	}
	s.dependencyFetchService.fetcher = fetcher
}

func (s *Service) SetDependencyRegistryAuthSourceForTests(source dependencyRegistryAuthSource) {
	if s == nil || s.dependencyFetchService == nil || source == nil {
		return
	}
	s.dependencyFetchService.authSource = source
}
