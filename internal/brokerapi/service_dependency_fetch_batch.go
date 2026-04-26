package brokerapi

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

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
		"schema_id":          "runecode.protocol.v0.DependencyFetchBatchResult",
		"schema_version":     "0.1.0",
		"batch_request_hash": digestObjectForIdentity(batchHash),
		"resolution_state":   "complete",
		"cache_outcome":      cacheOutcome,
		"resolved_units":     resolvedUnits,
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
