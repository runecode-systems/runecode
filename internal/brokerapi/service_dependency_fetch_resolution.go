package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

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
	flightKey := dependencyFetchFlightKey(runID, requestHash)
	flight, leader := s.acquireFlight(flightKey)
	if !leader {
		return s.waitForDependencyFetchFlight(ctx, flightKey, flight)
	}
	return s.resolveDependencyRequestLeader(ctx, runID, req, startedAt, requestHash, flightKey, flight)
}

func dependencyFetchFlightKey(runID, requestHash string) string {
	return strings.TrimSpace(runID) + "|" + strings.TrimSpace(requestHash)
}

func (s *dependencyFetchService) waitForDependencyFetchFlight(ctx context.Context, flightKey string, flight *dependencyFetchFlight) (dependencyUnitResolution, error) {
	select {
	case <-ctx.Done():
		s.releaseFlightWaiter(flightKey)
		return dependencyUnitResolution{}, ctx.Err()
	case <-flight.done:
		defer s.releaseFlightWaiter(flightKey)
		return flight.result, flight.err
	}
}

func (s *dependencyFetchService) resolveDependencyRequestLeader(ctx context.Context, runID string, req DependencyFetchRequestObject, startedAt time.Time, requestHash string, flightKey string, flight *dependencyFetchFlight) (dependencyUnitResolution, error) {
	var result dependencyUnitResolution
	var resultErr error
	defer s.finishDependencyFetchFlight(flightKey, flight, &result, &resultErr)
	authz, err := s.authorizeDependencyFetch(runID, req, requestHash)
	if err != nil {
		resultErr = err
		return dependencyUnitResolution{}, err
	}
	result, hit, err := s.resolveDependencyRequestFromCache(req, requestHash, startedAt)
	if err != nil {
		resultErr = err
		return dependencyUnitResolution{}, err
	}
	if !hit {
		result, err = s.executeMissFetch(ctx, req, requestHash, authz.maxResponseBytes)
		if err != nil {
			resultErr = err
			return dependencyUnitResolution{}, err
		}
	}
	finalizeDependencyResolutionTimestamps(s.owner.now().UTC(), startedAt, &result)
	s.applyDependencyAuthorization(&result, authz)
	return result, nil
}

func (s *dependencyFetchService) finishDependencyFetchFlight(flightKey string, flight *dependencyFetchFlight, result *dependencyUnitResolution, resultErr *error) {
	flight.result = *result
	flight.err = *resultErr
	close(flight.done)
	s.releaseFlightWaiter(flightKey)
}

func (s *dependencyFetchService) resolveDependencyRequestFromCache(req DependencyFetchRequestObject, requestHash string, startedAt time.Time) (dependencyUnitResolution, bool, error) {
	hitUnit, ok, err := s.owner.DependencyCacheResolvedUnitByRequest(requestHash)
	if err != nil || !ok {
		return dependencyUnitResolution{}, ok, err
	}
	manifest, err := s.resolvedManifestObject(req, requestHash, hitUnit)
	if err != nil {
		return dependencyUnitResolution{}, true, err
	}
	return dependencyUnitResolution{
		unit:                 hitUnit,
		requestHash:          requestHash,
		resolvedUnitManifest: manifest,
		registryAuthPosture:  string(dependencyRegistryAuthPosturePublicNoAuth),
		cacheOutcome:         "hit_exact",
		startedAt:            startedAt,
		completedAt:          s.owner.now().UTC(),
	}, true, nil
}

func finalizeDependencyResolutionTimestamps(now, startedAt time.Time, result *dependencyUnitResolution) {
	if result == nil {
		return
	}
	if result.startedAt.IsZero() {
		result.startedAt = startedAt
	}
	if result.completedAt.IsZero() {
		result.completedAt = now
	}
}

func (s *dependencyFetchService) executeMissFetch(ctx context.Context, req DependencyFetchRequestObject, requestHash string, maxResponseBytes int64) (dependencyUnitResolution, error) {
	startedAt := s.owner.now().UTC()
	if err := s.acquireFetchToken(ctx); err != nil {
		return dependencyUnitResolution{}, err
	}
	defer s.releaseFetchToken()
	lease, err := s.acquireDependencyRegistryLease(ctx, req)
	if err != nil {
		return dependencyUnitResolution{}, err
	}
	payloadRef, metadata, err := s.fetchDependencyPayload(ctx, req, requestHash, lease, maxResponseBytes)
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
	return s.buildMissDependencyResolution(requestHash, resolvedDigest, payloadRef, manifestRef.Digest, manifestPayload, lease, startedAt), nil
}

func (s *dependencyFetchService) buildMissDependencyResolution(requestHash, resolvedDigest string, payloadRef artifacts.ArtifactReference, manifestDigest string, manifestPayload map[string]any, lease dependencyRegistryAuthLease, startedAt time.Time) dependencyUnitResolution {
	return dependencyUnitResolution{
		unit: artifacts.DependencyCacheResolvedUnitRecord{
			ResolvedUnitDigest:   resolvedDigest,
			RequestDigest:        requestHash,
			ManifestDigest:       manifestDigest,
			PayloadDigest:        []string{payloadRef.Digest},
			IntegrityState:       "verified",
			MaterializationState: "derived_read_only",
			CreatedAt:            s.owner.now().UTC(),
		},
		requestHash:          requestHash,
		resolvedUnitManifest: manifestPayload,
		registryAuthPosture:  string(lease.Posture()),
		cacheOutcome:         "miss_filled",
		fetchedBytes:         payloadRef.SizeBytes,
		registryRequests:     1,
		startedAt:            startedAt,
		completedAt:          s.owner.now().UTC(),
	}
}

func (s *dependencyFetchService) acquireDependencyRegistryLease(ctx context.Context, req DependencyFetchRequestObject) (dependencyRegistryAuthLease, error) {
	if s.authSource == nil {
		s.authSource = publicRegistryNoAuthSource{}
	}
	return s.authSource.AcquireLease(ctx, req)
}

func (s *dependencyFetchService) fetchDependencyPayload(ctx context.Context, req DependencyFetchRequestObject, requestHash string, lease dependencyRegistryAuthLease, maxResponseBytes int64) (artifacts.ArtifactReference, dependencyRegistryFetchMetadata, error) {
	reader, metadata, err := s.fetcher.Fetch(ctx, req, lease)
	if err != nil {
		return artifacts.ArtifactReference{}, dependencyRegistryFetchMetadata{}, err
	}
	defer reader.Close()
	if maxResponseBytes <= 0 {
		maxResponseBytes = int64(gatewayRuntimeMaxResponseBytes)
	}
	boundedReader := newStreamingSizeLimitReader(reader, maxResponseBytes)
	payloadRef, err := s.owner.PutStream(artifacts.PutStreamRequest{
		Reader:                boundedReader,
		ContentType:           coalesceString(metadata.ContentType, "application/octet-stream"),
		DataClass:             artifacts.DataClassDependencyPayloadUnit,
		ProvenanceReceiptHash: artifacts.DigestBytes([]byte("dependency-registry-fetch:" + requestHash)),
		CreatedByRole:         "dependency-fetch",
		TrustedSource:         true,
		StepID:                "dependency_fetch",
	})
	if err != nil {
		if isStreamingSizeLimitError(err) {
			return artifacts.ArtifactReference{}, dependencyRegistryFetchMetadata{}, err
		}
		return artifacts.ArtifactReference{}, dependencyRegistryFetchMetadata{}, err
	}
	if metadata.ExpectedPayloadDigest != "" && metadata.ExpectedPayloadDigest != payloadRef.Digest {
		_ = s.owner.DeleteDigest(payloadRef.Digest)
		return artifacts.ArtifactReference{}, dependencyRegistryFetchMetadata{}, artifacts.ErrDependencyCacheUnverifiableIdentity
	}
	return payloadRef, metadata, nil
}

func (s *dependencyFetchService) applyDependencyAuthorization(resolution *dependencyUnitResolution, authz dependencyFetchAuthorization) {
	if resolution == nil {
		return
	}
	resolution.destinationKind = strings.TrimSpace(authz.destinationKind)
	resolution.destinationRef = strings.TrimSpace(authz.destinationRef)
	resolution.actionRequestHash = strings.TrimSpace(authz.actionRequestHash)
	resolution.policyDecisionHash = strings.TrimSpace(authz.policyDecisionHash)
	resolution.matchedAllowlistRef = strings.TrimSpace(authz.matchedAllowlistRef)
	resolution.matchedAllowlistID = strings.TrimSpace(authz.matchedAllowlistID)
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
