package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *dependencyFetchService) FetchSingle(ctx context.Context, requestID string, req DependencyFetchRegistryRequest) (DependencyFetchRegistryResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	requestHash, err := validateDependencyFetchRegistryRequest(req)
	if err != nil {
		return DependencyFetchRegistryResponse{}, err
	}
	resolution, err := s.resolveDependencyRequest(ctx, runID, req.DependencyRequest)
	if err != nil {
		return DependencyFetchRegistryResponse{}, err
	}
	s.appendDependencyRegistryFetchAudit(requestID, runID, requestHash, resolution)
	return dependencyFetchRegistryResponse(requestID, requestHash, resolution), nil
}

func validateDependencyFetchRegistryRequest(req DependencyFetchRegistryRequest) (string, error) {
	requestHash, err := canonicalDependencyRequestIdentity(req.DependencyRequest)
	if err != nil {
		return "", err
	}
	requestHashIdentity, err := req.RequestHash.Identity()
	if err != nil {
		return "", err
	}
	if requestHashIdentity != requestHash {
		return "", fmt.Errorf("request_hash does not match dependency_request canonical identity")
	}
	return requestHash, nil
}

func (s *dependencyFetchService) appendDependencyRegistryFetchAudit(requestID, runID, requestHash string, resolution dependencyUnitResolution) {
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
}

func dependencyFetchRegistryResponse(requestID, requestHash string, resolution dependencyUnitResolution) DependencyFetchRegistryResponse {
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
	}
}
