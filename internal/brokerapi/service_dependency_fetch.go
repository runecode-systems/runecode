package brokerapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
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
		fetcher:    publicRegistryDeterministicFetcher{},
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

func (s *dependencyFetchService) EnsureBatch(ctx context.Context, requestID string, req DependencyCacheEnsureRequest) (DependencyCacheEnsureResponse, error) {
	batchHash, err := canonicalDependencyBatchIdentity(req.BatchRequest)
	if err != nil {
		return DependencyCacheEnsureResponse{}, err
	}
	runID := strings.TrimSpace(req.RunID)
	resolutions, err := s.resolveBatchRequests(ctx, runID, req.BatchRequest.DependencyRequests)
	if err != nil {
		return DependencyCacheEnsureResponse{}, err
	}
	batchManifestPayload, cacheOutcome, fetchedBytes, registryRequests, err := s.buildBatchManifestPayload(batchHash, resolutions)
	if err != nil {
		return DependencyCacheEnsureResponse{}, err
	}
	batchManifestRef, err := s.putBatchManifest(runID, batchHash, batchManifestPayload)
	if err != nil {
		return DependencyCacheEnsureResponse{}, err
	}
	summary := summarizeDependencyBatchResolutions(s.owner.now().UTC(), resolutions)
	batchRecord := s.buildBatchRecord(req.BatchRequest, batchHash, batchManifestRef.Digest, cacheOutcome)
	if err := s.owner.RecordDependencyCacheBatch(batchRecord, summary.units); err != nil {
		return DependencyCacheEnsureResponse{}, err
	}
	s.appendDependencyCacheEnsureAudit(requestID, runID, batchHash, batchManifestRef.Digest, cacheOutcome, fetchedBytes, registryRequests, summary)
	return DependencyCacheEnsureResponse{
		SchemaID:             "runecode.protocol.v0.DependencyCacheEnsureResponse",
		SchemaVersion:        "0.1.0",
		RequestID:            requestID,
		BatchRequestHash:     mustDigestObjectFromIdentity(batchHash),
		BatchManifestDigest:  mustDigestObjectFromIdentity(batchManifestRef.Digest),
		ResolutionState:      "complete",
		CacheOutcome:         cacheOutcome,
		ResolvedUnitDigests:  mapDigestIdentities(summary.resolvedDigests),
		FetchedBytes:         fetchedBytes,
		RegistryRequestCount: registryRequests,
	}, nil
}

func (s *dependencyFetchService) putBatchManifest(runID, batchHash string, batchManifestPayload map[string]any) (artifacts.ArtifactReference, error) {
	batchManifestBytes, err := canonicalJSONBytesForValue(batchManifestPayload)
	if err != nil {
		return artifacts.ArtifactReference{}, err
	}
	return s.owner.Put(artifacts.PutRequest{
		Payload:               batchManifestBytes,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassDependencyBatchManifest,
		ProvenanceReceiptHash: artifacts.DigestBytes([]byte("dependency-cache-batch:" + batchHash)),
		CreatedByRole:         "dependency-fetch",
		TrustedSource:         true,
		RunID:                 runID,
		StepID:                "dependency_fetch",
	})
}

func (s *dependencyFetchService) buildBatchRecord(batchReq DependencyFetchBatchRequestObject, batchHash, manifestDigest, cacheOutcome string) artifacts.DependencyCacheBatchRecord {
	return artifacts.DependencyCacheBatchRecord{
		BatchRequestDigest:  batchHash,
		BatchManifestDigest: manifestDigest,
		LockfileDigest:      mustDigestIdentity(batchReq.LockfileDigest),
		RequestSetDigest:    mustDigestIdentity(batchReq.RequestSetHash),
		ResolutionState:     "complete",
		CacheOutcome:        cacheOutcome,
		CreatedAt:           s.owner.now().UTC(),
	}
}

func summarizeDependencyBatchResolutions(now time.Time, resolutions []dependencyUnitResolution) dependencyBatchEnsureSummary {
	summary := dependencyBatchEnsureSummary{
		units:           make([]artifacts.DependencyCacheResolvedUnitRecord, 0, len(resolutions)),
		resolvedDigests: make([]string, 0, len(resolutions)),
		startedAt:       now,
		completedAt:     now,
	}
	if len(resolutions) == 0 {
		return summary
	}
	summary.startedAt = resolutions[0].startedAt
	summary.completedAt = resolutions[0].completedAt
	for _, resolution := range resolutions {
		summary.units = append(summary.units, resolution.unit)
		summary.resolvedDigests = append(summary.resolvedDigests, resolution.unit.ResolvedUnitDigest)
		summary.destinationKinds = append(summary.destinationKinds, resolution.destinationKind)
		summary.destinationRefs = append(summary.destinationRefs, resolution.destinationRef)
		if resolution.matchedAllowlistRef != "" {
			summary.allowlistRefs = append(summary.allowlistRefs, resolution.matchedAllowlistRef)
		}
		if resolution.matchedAllowlistID != "" {
			summary.allowlistEntryIDs = append(summary.allowlistEntryIDs, resolution.matchedAllowlistID)
		}
		summary.requestBindings = append(summary.requestBindings, dependencyResolutionRequestBinding(resolution))
		if resolution.startedAt.Before(summary.startedAt) {
			summary.startedAt = resolution.startedAt
		}
		if resolution.completedAt.After(summary.completedAt) {
			summary.completedAt = resolution.completedAt
		}
	}
	sort.Strings(summary.resolvedDigests)
	return summary
}

func dependencyResolutionRequestBinding(resolution dependencyUnitResolution) map[string]any {
	binding := map[string]any{
		"request_hash":               resolution.requestHash,
		"payload_hash":               resolution.requestHash,
		"request_payload_hash_bound": true,
		"destination_kind":           resolution.destinationKind,
		"destination_ref":            resolution.destinationRef,
		"registry_auth_posture":      resolution.registryAuthPosture,
		"cache_outcome":              resolution.cacheOutcome,
		"resolved_unit_digest":       resolution.unit.ResolvedUnitDigest,
		"payload_digests":            append([]string{}, resolution.unit.PayloadDigest...),
		"fetched_bytes":              resolution.fetchedBytes,
		"started_at":                 resolution.startedAt.Format(time.RFC3339),
		"completed_at":               resolution.completedAt.Format(time.RFC3339),
	}
	if resolution.actionRequestHash != "" {
		binding["action_request_hash"] = resolution.actionRequestHash
	}
	if resolution.policyDecisionHash != "" {
		binding["policy_decision_hash"] = resolution.policyDecisionHash
	}
	if resolution.matchedAllowlistRef != "" {
		binding["matched_allowlist_ref"] = resolution.matchedAllowlistRef
	}
	if resolution.matchedAllowlistID != "" {
		binding["matched_allowlist_entry_id"] = resolution.matchedAllowlistID
	}
	return binding
}

func (s *dependencyFetchService) appendDependencyCacheEnsureAudit(requestID, runID, batchHash, batchManifestDigest, cacheOutcome string, fetchedBytes int64, registryRequests int, summary dependencyBatchEnsureSummary) {
	_ = s.owner.AppendTrustedAuditEvent("dependency_cache_ensure", "brokerapi", map[string]any{
		"request_id":                  requestID,
		"run_id":                      runID,
		"gateway_role_kind":           "dependency-fetch",
		"operation":                   "fetch_dependency",
		"audit_outcome":               "succeeded",
		"started_at":                  summary.startedAt.Format(time.RFC3339),
		"completed_at":                summary.completedAt.Format(time.RFC3339),
		"duration_ms":                 summary.completedAt.Sub(summary.startedAt).Milliseconds(),
		"outbound_bytes":              fetchedBytes,
		"fetched_bytes":               fetchedBytes,
		"registry_request_count":      registryRequests,
		"batch_request_hash":          batchHash,
		"batch_manifest_digest":       batchManifestDigest,
		"cache_outcome":               cacheOutcome,
		"resolved_unit_digests":       summary.resolvedDigests,
		"destination_kinds":           uniqueSortedStrings(summary.destinationKinds),
		"destination_refs":            uniqueSortedStrings(summary.destinationRefs),
		"matched_allowlist_refs":      uniqueSortedStrings(summary.allowlistRefs),
		"matched_allowlist_entry_ids": uniqueSortedStrings(summary.allowlistEntryIDs),
		"request_bindings":            summary.requestBindings,
	})
}

func (s *dependencyFetchService) FetchSingle(ctx context.Context, requestID string, req DependencyFetchRegistryRequest) (DependencyFetchRegistryResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	requestHash, err := canonicalDependencyRequestIdentity(req.DependencyRequest)
	if err != nil {
		return DependencyFetchRegistryResponse{}, err
	}
	requestHashIdentity, err := req.RequestHash.Identity()
	if err != nil {
		return DependencyFetchRegistryResponse{}, err
	}
	if requestHashIdentity != requestHash {
		return DependencyFetchRegistryResponse{}, fmt.Errorf("request_hash does not match dependency_request canonical identity")
	}
	resolution, err := s.resolveDependencyRequest(ctx, runID, req.DependencyRequest)
	if err != nil {
		return DependencyFetchRegistryResponse{}, err
	}
	_ = s.owner.AppendTrustedAuditEvent("dependency_registry_fetch", "brokerapi", map[string]any{
		"request_id":                 requestID,
		"run_id":                     runID,
		"gateway_role_kind":          "dependency-fetch",
		"destination_kind":           resolution.destinationKind,
		"destination_ref":            resolution.destinationRef,
		"operation":                  "fetch_dependency",
		"audit_outcome":              "succeeded",
		"started_at":                 resolution.startedAt.Format(time.RFC3339),
		"completed_at":               resolution.completedAt.Format(time.RFC3339),
		"duration_ms":                resolution.completedAt.Sub(resolution.startedAt).Milliseconds(),
		"outbound_bytes":             resolution.fetchedBytes,
		"request_hash":               requestHash,
		"payload_hash":               requestHash,
		"request_payload_hash_bound": true,
		"registry_auth_posture":      resolution.registryAuthPosture,
		"resolved_unit_digest":       resolution.unit.ResolvedUnitDigest,
		"payload_digests":            append([]string{}, resolution.unit.PayloadDigest...),
		"cache_outcome":              resolution.cacheOutcome,
		"fetched_bytes":              resolution.fetchedBytes,
		"registry_request_count":     resolution.registryRequests,
		"action_request_hash":        resolution.actionRequestHash,
		"policy_decision_hash":       resolution.policyDecisionHash,
		"matched_allowlist_ref":      resolution.matchedAllowlistRef,
		"matched_allowlist_entry_id": resolution.matchedAllowlistID,
	})
	return DependencyFetchRegistryResponse{
		SchemaID:             "runecode.protocol.v0.DependencyFetchRegistryResponse",
		SchemaVersion:        "0.1.0",
		RequestID:            requestID,
		RequestHash:          mustDigestObjectFromIdentity(requestHash),
		ResolvedUnitDigest:   mustDigestObjectFromIdentity(resolution.unit.ResolvedUnitDigest),
		ManifestDigest:       mustDigestObjectFromIdentity(resolution.unit.ManifestDigest),
		PayloadDigests:       mapDigestIdentities(resolution.unit.PayloadDigest),
		CacheOutcome:         resolution.cacheOutcome,
		FetchedBytes:         resolution.fetchedBytes,
		RegistryRequestCount: resolution.registryRequests,
	}, nil
}

func (s *dependencyFetchService) resolveBatchRequests(ctx context.Context, runID string, requests []DependencyFetchRequestObject) ([]dependencyUnitResolution, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("dependency_requests must be non-empty")
	}
	resolutions := make([]dependencyUnitResolution, len(requests))
	errCh := make(chan error, len(requests))
	var wg sync.WaitGroup
	for i := range requests {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			resolution, err := s.resolveDependencyRequest(ctx, runID, requests[i])
			if err != nil {
				errCh <- err
				return
			}
			resolutions[i] = resolution
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}
	return resolutions, nil
}

func (s *dependencyFetchService) resolveDependencyRequest(ctx context.Context, runID string, req DependencyFetchRequestObject) (dependencyUnitResolution, error) {
	startedAt := s.owner.now().UTC()
	requestHash, err := canonicalDependencyRequestIdentity(req)
	if err != nil {
		return dependencyUnitResolution{}, err
	}
	flight, leader := s.acquireFlight(requestHash)
	if !leader {
		select {
		case <-ctx.Done():
			s.releaseFlightWaiter(requestHash)
			return dependencyUnitResolution{}, ctx.Err()
		case <-flight.done:
			defer s.releaseFlightWaiter(requestHash)
			return flight.result, flight.err
		}
	}
	var result dependencyUnitResolution
	var resultErr error
	defer func() {
		flight.result = result
		flight.err = resultErr
		close(flight.done)
		s.releaseFlightWaiter(requestHash)
	}()
	hitUnit, ok, err := s.owner.DependencyCacheResolvedUnitByRequest(requestHash)
	if err != nil {
		resultErr = err
		return dependencyUnitResolution{}, resultErr
	}
	if ok {
		manifest, manifestErr := s.resolvedManifestObject(req, requestHash, hitUnit)
		if manifestErr != nil {
			resultErr = manifestErr
			return dependencyUnitResolution{}, resultErr
		}
		result = dependencyUnitResolution{
			unit:                 hitUnit,
			requestHash:          requestHash,
			resolvedUnitManifest: manifest,
			registryAuthPosture:  string(dependencyRegistryAuthPosturePublicNoAuth),
			cacheOutcome:         "hit_exact",
			startedAt:            startedAt,
			completedAt:          s.owner.now().UTC(),
		}
		s.enrichPolicyAndAllowlistLinkage(runID, req, &result)
		return result, nil
	}
	result, err = s.executeMissFetch(ctx, req, requestHash)
	if err != nil {
		resultErr = err
		return dependencyUnitResolution{}, resultErr
	}
	if result.startedAt.IsZero() {
		result.startedAt = startedAt
	}
	if result.completedAt.IsZero() {
		result.completedAt = s.owner.now().UTC()
	}
	s.enrichPolicyAndAllowlistLinkage(runID, req, &result)
	return result, nil
}

func (s *dependencyFetchService) executeMissFetch(ctx context.Context, req DependencyFetchRequestObject, requestHash string) (dependencyUnitResolution, error) {
	startedAt := s.owner.now().UTC()
	if err := s.acquireFetchToken(ctx); err != nil {
		return dependencyUnitResolution{}, err
	}
	defer s.releaseFetchToken()
	lease, err := s.acquireDependencyRegistryLease(ctx, req)
	if err != nil {
		return dependencyUnitResolution{}, err
	}
	payloadRef, metadata, err := s.fetchDependencyPayload(ctx, req, requestHash, lease)
	if err != nil {
		return dependencyUnitResolution{}, err
	}
	resolvedDigest, err := s.computeResolvedUnitDigest(req, requestHash, payloadRef.Digest)
	if err != nil {
		return dependencyUnitResolution{}, err
	}
	manifestPayload, manifestRef, err := s.putResolvedUnitManifest(req, requestHash, resolvedDigest, payloadRef, metadata)
	if err != nil {
		return dependencyUnitResolution{}, err
	}
	unit := artifacts.DependencyCacheResolvedUnitRecord{
		ResolvedUnitDigest:   resolvedDigest,
		RequestDigest:        requestHash,
		ManifestDigest:       manifestRef.Digest,
		PayloadDigest:        []string{payloadRef.Digest},
		IntegrityState:       "verified",
		MaterializationState: "derived_read_only",
		CreatedAt:            s.owner.now().UTC(),
	}
	return dependencyUnitResolution{
		unit:                 unit,
		requestHash:          requestHash,
		resolvedUnitManifest: manifestPayload,
		registryAuthPosture:  string(lease.Posture()),
		cacheOutcome:         "miss_filled",
		fetchedBytes:         payloadRef.SizeBytes,
		registryRequests:     1,
		startedAt:            startedAt,
		completedAt:          s.owner.now().UTC(),
	}, nil
}

func (s *dependencyFetchService) acquireDependencyRegistryLease(ctx context.Context, req DependencyFetchRequestObject) (dependencyRegistryAuthLease, error) {
	if s.authSource == nil {
		s.authSource = publicRegistryNoAuthSource{}
	}
	lease, err := s.authSource.AcquireLease(ctx, req)
	if err != nil {
		return nil, err
	}
	return lease, nil
}

func (s *dependencyFetchService) fetchDependencyPayload(ctx context.Context, req DependencyFetchRequestObject, requestHash string, lease dependencyRegistryAuthLease) (artifacts.ArtifactReference, dependencyRegistryFetchMetadata, error) {
	reader, metadata, err := s.fetcher.Fetch(ctx, req, lease)
	if err != nil {
		return artifacts.ArtifactReference{}, dependencyRegistryFetchMetadata{}, err
	}
	defer reader.Close()
	payloadRef, err := s.owner.PutStream(artifacts.PutStreamRequest{
		Reader:                reader,
		ContentType:           coalesceString(metadata.ContentType, "application/octet-stream"),
		DataClass:             artifacts.DataClassDependencyPayloadUnit,
		ProvenanceReceiptHash: artifacts.DigestBytes([]byte("dependency-registry-fetch:" + requestHash)),
		CreatedByRole:         "dependency-fetch",
		TrustedSource:         true,
		StepID:                "dependency_fetch",
	})
	if err != nil {
		return artifacts.ArtifactReference{}, dependencyRegistryFetchMetadata{}, err
	}
	if expected := strings.TrimSpace(metadata.ExpectedPayloadDigest); expected != "" && expected != payloadRef.Digest {
		_ = s.owner.DeleteDigest(payloadRef.Digest)
		return artifacts.ArtifactReference{}, dependencyRegistryFetchMetadata{}, artifacts.ErrDependencyCacheUnverifiableIdentity
	}
	return payloadRef, metadata, nil
}

func (s *dependencyFetchService) putResolvedUnitManifest(req DependencyFetchRequestObject, requestHash, resolvedDigest string, payloadRef artifacts.ArtifactReference, metadata dependencyRegistryFetchMetadata) (map[string]any, artifacts.ArtifactReference, error) {
	manifestPayload, err := s.buildResolvedUnitManifestPayload(req, requestHash, resolvedDigest, payloadRef, metadata)
	if err != nil {
		return nil, artifacts.ArtifactReference{}, err
	}
	manifestBytes, err := canonicalJSONBytesForValue(manifestPayload)
	if err != nil {
		return nil, artifacts.ArtifactReference{}, err
	}
	manifestRef, err := s.owner.Put(artifacts.PutRequest{
		Payload:               manifestBytes,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassDependencyResolvedUnit,
		ProvenanceReceiptHash: artifacts.DigestBytes([]byte("dependency-resolved-unit:" + requestHash)),
		CreatedByRole:         "dependency-fetch",
		TrustedSource:         true,
		StepID:                "dependency_fetch",
	})
	if err != nil {
		return nil, artifacts.ArtifactReference{}, err
	}
	return manifestPayload, manifestRef, nil
}

func (s *dependencyFetchService) enrichPolicyAndAllowlistLinkage(runID string, req DependencyFetchRequestObject, resolution *dependencyUnitResolution) {
	if resolution == nil {
		return
	}
	resolution.destinationKind = strings.TrimSpace(req.RegistryIdentity.DescriptorKind)
	resolution.destinationRef = destinationRefFromDescriptor(req.RegistryIdentity)
	policyRef, actionHash, ok := s.latestDependencyFetchPolicyDecision(runID, resolution.requestHash)
	if ok {
		resolution.policyDecisionHash = policyRef
		resolution.actionRequestHash = actionHash
	}
	runtime := policyRuntime{service: s.owner}
	compileInput, err := runtime.loadCompileInput(strings.TrimSpace(runID))
	if err != nil {
		return
	}
	payload := gatewayActionPayloadRuntime{
		GatewayRoleKind: "dependency-fetch",
		DestinationKind: resolution.destinationKind,
		DestinationRef:  resolution.destinationRef,
		Operation:       "fetch_dependency",
	}
	_, match, found, _ := findAllowlistEntryForGatewayPayload(compileInput.Allowlists, payload)
	if !found {
		return
	}
	resolution.matchedAllowlistRef = strings.TrimSpace(match.AllowlistRef)
	resolution.matchedAllowlistID = strings.TrimSpace(match.EntryID)
}

func (s *dependencyFetchService) latestDependencyFetchPolicyDecision(runID, requestHash string) (string, string, bool) {
	runID = strings.TrimSpace(runID)
	requestHash = strings.TrimSpace(requestHash)
	if runID == "" || requestHash == "" {
		return "", "", false
	}
	latestRef := ""
	latestAction := ""
	var latestRecordedAt time.Time
	for _, ref := range s.owner.PolicyDecisionRefsForRun(runID) {
		rec, ok := s.owner.PolicyDecisionGet(ref)
		if !ok || !matchesDependencyFetchPolicyDecision(rec, requestHash) {
			continue
		}
		recordedAt := rec.RecordedAt.UTC()
		if latestRef == "" || recordedAt.After(latestRecordedAt) || (recordedAt.Equal(latestRecordedAt) && ref > latestRef) {
			latestRef = ref
			latestAction = strings.TrimSpace(rec.ActionRequestHash)
			latestRecordedAt = recordedAt
		}
	}
	if latestRef == "" {
		return "", "", false
	}
	return latestRef, latestAction, true
}

func matchesDependencyFetchPolicyDecision(rec artifacts.PolicyDecisionRecord, requestHash string) bool {
	if strings.TrimSpace(rec.ActionRequestHash) == "" {
		return false
	}
	if !containsStringIdentity(rec.RelevantArtifactHashes, requestHash) {
		return false
	}
	if operation, ok := rec.Details["operation"].(string); ok && strings.TrimSpace(operation) != "" && strings.TrimSpace(operation) != "fetch_dependency" {
		return false
	}
	if role, ok := rec.Details["gateway_role_kind"].(string); ok && strings.TrimSpace(role) != "" && strings.TrimSpace(role) != "dependency-fetch" {
		return false
	}
	if kind, ok := rec.Details["destination_kind"].(string); ok && strings.TrimSpace(kind) != "" && strings.TrimSpace(kind) != "package_registry" {
		return false
	}
	return true
}

func (s *dependencyFetchService) buildBatchManifestPayload(batchHash string, resolutions []dependencyUnitResolution) (map[string]any, string, int64, int, error) {
	resolvedUnits := make([]any, 0, len(resolutions))
	cacheOutcome := "hit_exact"
	var fetchedBytes int64
	registryRequests := 0
	for _, resolution := range resolutions {
		resolvedUnits = append(resolvedUnits, resolution.resolvedUnitManifest)
		if resolution.cacheOutcome == "miss_filled" {
			cacheOutcome = "miss_filled"
		}
		fetchedBytes += resolution.fetchedBytes
		registryRequests += resolution.registryRequests
	}
	payload := map[string]any{
		"schema_id":             "runecode.protocol.v0.DependencyFetchBatchResult",
		"schema_version":        "0.1.0",
		"batch_request_hash":    digestObjectForIdentity(batchHash),
		"batch_manifest_digest": digestObjectForIdentity("sha256:" + strings.Repeat("0", 64)),
		"resolution_state":      "complete",
		"cache_outcome":         cacheOutcome,
		"resolved_units":        resolvedUnits,
		"materialization": map[string]any{
			"derived_only": true,
			"read_only":    true,
		},
		"fetched_bytes": fetchedBytes,
		"completed_at":  s.owner.now().UTC().Format(time.RFC3339),
	}
	if err := validateJSONEnvelope(payload, dependencyFetchBatchResultSchemaPath); err != nil {
		return nil, "", 0, 0, err
	}
	return payload, cacheOutcome, fetchedBytes, registryRequests, nil
}

func (s *dependencyFetchService) resolvedManifestObject(req DependencyFetchRequestObject, requestHash string, unit artifacts.DependencyCacheResolvedUnitRecord) (map[string]any, error) {
	if len(unit.PayloadDigest) == 0 {
		return nil, artifacts.ErrDependencyCacheIncompleteState
	}
	payloadArtifact, err := s.owner.Head(unit.PayloadDigest[0])
	if err != nil {
		return nil, err
	}
	return s.buildResolvedUnitManifestPayload(req, requestHash, unit.ResolvedUnitDigest, payloadArtifact.Reference, dependencyRegistryFetchMetadata{})
}

func (s *dependencyFetchService) buildResolvedUnitManifestPayload(req DependencyFetchRequestObject, requestHash, resolvedDigest string, payloadRef artifacts.ArtifactReference, metadata dependencyRegistryFetchMetadata) (map[string]any, error) {
	payload := map[string]any{
		"schema_id":            "runecode.protocol.v0.DependencyResolvedUnitManifest",
		"schema_version":       "0.1.0",
		"request_hash":         digestObjectForIdentity(requestHash),
		"resolved_unit_digest": digestObjectForIdentity(resolvedDigest),
		"dependency_request":   req,
		"payload_artifacts": []any{map[string]any{
			"schema_id":               "runecode.protocol.v0.ArtifactReference",
			"schema_version":          "0.4.0",
			"digest":                  digestObjectForIdentity(payloadRef.Digest),
			"size_bytes":              payloadRef.SizeBytes,
			"content_type":            payloadRef.ContentType,
			"data_class":              string(payloadRef.DataClass),
			"provenance_receipt_hash": digestObjectForIdentity(payloadRef.ProvenanceReceiptHash),
		}},
		"integrity": map[string]any{
			"verification_state": "verified",
		},
		"materialization": map[string]any{
			"derived_only":       true,
			"read_only_required": true,
		},
	}
	if digest := strings.TrimSpace(metadata.UpstreamManifestDigest); digest != "" {
		payload["integrity"].(map[string]any)["upstream_manifest_digest"] = digestObjectForIdentity(digest)
	}
	if err := validateJSONEnvelope(payload, dependencyResolvedUnitManifestSchemaPath); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *dependencyFetchService) computeResolvedUnitDigest(req DependencyFetchRequestObject, requestHash, payloadDigest string) (string, error) {
	input := map[string]any{
		"request_hash":       requestHash,
		"dependency_request": req,
		"payload_digests":    []string{payloadDigest},
	}
	identity, err := canonicalDigestIdentity(input)
	if err != nil {
		return "", err
	}
	return identity, nil
}

func (s *dependencyFetchService) acquireFlight(requestHash string) (*dependencyFetchFlight, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.inflight[requestHash]; ok {
		current.waiters++
		return current, false
	}
	flight := &dependencyFetchFlight{done: make(chan struct{}), waiters: 1}
	s.inflight[requestHash] = flight
	return flight, true
}

func (s *dependencyFetchService) releaseFlightWaiter(requestHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	flight, ok := s.inflight[requestHash]
	if !ok {
		return
	}
	flight.waiters--
	if flight.waiters <= 0 {
		delete(s.inflight, requestHash)
	}
}

func (s *dependencyFetchService) acquireFetchToken(ctx context.Context) error {
	select {
	case s.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *dependencyFetchService) releaseFetchToken() {
	select {
	case <-s.sem:
	default:
	}
}

func canonicalDigestIdentity(value any) (string, error) {
	d, err := canonicalDigestForValue(value)
	if err != nil {
		return "", err
	}
	return d.Identity()
}

func canonicalDependencyRequestIdentity(req DependencyFetchRequestObject) (string, error) {
	identity := map[string]any{
		"schema_id":         req.SchemaID,
		"schema_version":    req.SchemaVersion,
		"request_kind":      req.RequestKind,
		"registry_identity": req.RegistryIdentity,
		"ecosystem":         req.Ecosystem,
		"package_name":      req.PackageName,
		"package_version":   req.PackageVersion,
	}
	return canonicalDigestIdentity(identity)
}

func canonicalDependencyBatchIdentity(req DependencyFetchBatchRequestObject) (string, error) {
	identity := map[string]any{
		"schema_id":           req.SchemaID,
		"schema_version":      req.SchemaVersion,
		"lockfile_kind":       req.LockfileKind,
		"lockfile_digest":     req.LockfileDigest,
		"request_set_hash":    req.RequestSetHash,
		"dependency_requests": req.DependencyRequests,
	}
	return canonicalDigestIdentity(identity)
}

func coalesceString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func canonicalJSONBytesForValue(value any) ([]byte, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(b)
}

func mapDigestIdentities(identities []string) []trustpolicy.Digest {
	out := make([]trustpolicy.Digest, 0, len(identities))
	for _, identity := range identities {
		out = append(out, mustDigestObjectFromIdentity(identity))
	}
	return out
}

func mustDigestObjectFromIdentity(identity string) trustpolicy.Digest {
	d, err := digestFromIdentity(identity)
	if err != nil {
		panic(err)
	}
	return d
}

func mustDigestIdentity(d trustpolicy.Digest) string {
	identity, err := d.Identity()
	if err != nil {
		panic(err)
	}
	return identity
}

type publicRegistryDeterministicFetcher struct{}

func (publicRegistryDeterministicFetcher) Fetch(_ context.Context, req DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if req.RegistryIdentity.DescriptorKind != "package_registry" {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("dependency registry descriptor_kind must be package_registry")
	}
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("dependency registry auth lease is required")
	}
	if lease.Posture() != dependencyRegistryAuthPosturePublicNoAuth {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("public registry fetcher requires public_no_auth posture")
	}
	payload := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", strings.TrimSpace(req.Ecosystem), strings.TrimSpace(req.PackageName), strings.TrimSpace(req.PackageVersion), strings.TrimSpace(req.RegistryIdentity.CanonicalHost), strings.TrimSpace(req.RegistryIdentity.CanonicalPathPrefix))
	payloadDigest := artifacts.DigestBytes([]byte(payload))
	upstreamDigest := artifacts.DigestBytes([]byte("manifest:" + payload))
	return io.NopCloser(strings.NewReader(payload)), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream", ExpectedPayloadDigest: payloadDigest, UpstreamManifestDigest: upstreamDigest}, nil
}

type publicRegistryNoAuthSource struct{}

func (publicRegistryNoAuthSource) AcquireLease(_ context.Context, _ DependencyFetchRequestObject) (dependencyRegistryAuthLease, error) {
	return publicRegistryNoAuthLease{}, nil
}

type publicRegistryNoAuthLease struct{}

func (publicRegistryNoAuthLease) Posture() dependencyRegistryAuthPosture {
	return dependencyRegistryAuthPosturePublicNoAuth
}
func (publicRegistryNoAuthLease) LeaseID() string      { return "" }
func (publicRegistryNoAuthLease) ExpiresAt() time.Time { return time.Time{} }
