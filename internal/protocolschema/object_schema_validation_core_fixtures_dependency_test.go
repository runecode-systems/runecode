package protocolschema

func validDependencyFetchRequest() map[string]any {
	registryIdentity := validDestinationDescriptor("registry")
	registryIdentity["descriptor_kind"] = "package_registry"
	delete(registryIdentity, "git_repository_identity")
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.DependencyFetchRequest",
		"schema_version":    "0.1.0",
		"request_kind":      "package_version_fetch",
		"registry_identity": registryIdentity,
		"ecosystem":         "npm",
		"package_name":      "left-pad",
		"package_version":   "1.3.0",
	}
}

func invalidDependencyFetchRequestWithNonRegistryDescriptorKind() map[string]any {
	request := validDependencyFetchRequest()
	registryIdentity := request["registry_identity"].(map[string]any)
	registryIdentity["descriptor_kind"] = "model_endpoint"
	return request
}

func validDependencyFetchBatchRequest() map[string]any {
	return map[string]any{
		"schema_id":           "runecode.protocol.v0.DependencyFetchBatchRequest",
		"schema_version":      "0.1.0",
		"lockfile_kind":       "npm_package_lock",
		"lockfile_digest":     testDigestValue("b"),
		"request_set_hash":    testDigestValue("c"),
		"dependency_requests": []any{validDependencyFetchRequest()},
	}
}

func invalidDependencyFetchBatchRequestWithoutRequests() map[string]any {
	request := validDependencyFetchBatchRequest()
	delete(request, "dependency_requests")
	return request
}

func validDependencyResolvedUnitManifest() map[string]any {
	return map[string]any{
		"schema_id":            "runecode.protocol.v0.DependencyResolvedUnitManifest",
		"schema_version":       "0.1.0",
		"request_hash":         testDigestValue("d"),
		"resolved_unit_digest": testDigestValue("e"),
		"dependency_request":   validDependencyFetchRequest(),
		"payload_artifacts": []any{
			map[string]any{
				"schema_id":               "runecode.protocol.v0.ArtifactReference",
				"schema_version":          "0.4.0",
				"digest":                  testDigestValue("f"),
				"size_bytes":              1024,
				"content_type":            "application/octet-stream",
				"data_class":              "dependency_resolved_payload",
				"provenance_receipt_hash": testDigestValue("1"),
			},
		},
		"integrity": map[string]any{
			"verification_state": "verified",
		},
		"materialization": map[string]any{
			"derived_only":       true,
			"read_only_required": true,
		},
	}
}

func invalidDependencyResolvedUnitManifestWithWrongPayloadDataClass() map[string]any {
	manifest := validDependencyResolvedUnitManifest()
	payload := manifest["payload_artifacts"].([]any)[0].(map[string]any)
	payload["data_class"] = "build_logs"
	return manifest
}

func validDependencyFetchBatchResult() map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.DependencyFetchBatchResult",
		"schema_version":     "0.1.0",
		"batch_request_hash": testDigestValue("2"),
		"resolution_state":   "complete",
		"cache_outcome":      "hit_exact",
		"resolved_units":     []any{validDependencyResolvedUnitManifest()},
		"materialization": map[string]any{
			"derived_only": true,
			"read_only":    true,
		},
	}
}

func invalidDependencyFetchBatchResultWithIncompleteResolution() map[string]any {
	result := validDependencyFetchBatchResult()
	result["resolution_state"] = "partial"
	return result
}

func validDependencyCacheEnsureRequest() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.DependencyCacheEnsureRequest",
		"schema_version": "0.1.0",
		"request_id":     "req-dependency-cache-ensure",
		"run_id":         "run-1",
		"batch_request":  validDependencyFetchBatchRequest(),
	}
}

func invalidDependencyCacheEnsureRequestWithoutRunID() map[string]any {
	request := validDependencyCacheEnsureRequest()
	delete(request, "run_id")
	return request
}

func validDependencyCacheEnsureResponse() map[string]any {
	return map[string]any{
		"schema_id":              "runecode.protocol.v0.DependencyCacheEnsureResponse",
		"schema_version":         "0.1.0",
		"request_id":             "req-dependency-cache-ensure",
		"batch_request_hash":     testDigestValue("4"),
		"batch_manifest_digest":  testDigestValue("5"),
		"resolution_state":       "complete",
		"cache_outcome":          "miss_filled",
		"resolved_unit_digests":  []any{testDigestValue("6")},
		"fetched_bytes":          1024,
		"registry_request_count": 1,
	}
}

func invalidDependencyCacheEnsureResponseWithoutCacheOutcome() map[string]any {
	response := validDependencyCacheEnsureResponse()
	delete(response, "cache_outcome")
	return response
}

func validDependencyFetchRegistryRequest() map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.DependencyFetchRegistryRequest",
		"schema_version":     "0.1.0",
		"request_id":         "req-dependency-fetch-registry",
		"run_id":             "run-1",
		"dependency_request": validDependencyFetchRequest(),
		"request_hash":       testDigestValue("7"),
	}
}

func invalidDependencyFetchRegistryRequestWithoutRequestHash() map[string]any {
	request := validDependencyFetchRegistryRequest()
	delete(request, "request_hash")
	return request
}

func validDependencyFetchRegistryResponse() map[string]any {
	return map[string]any{
		"schema_id":              "runecode.protocol.v0.DependencyFetchRegistryResponse",
		"schema_version":         "0.1.0",
		"request_id":             "req-dependency-fetch-registry",
		"request_hash":           testDigestValue("8"),
		"resolved_unit_digest":   testDigestValue("9"),
		"manifest_digest":        testDigestValue("a"),
		"payload_digests":        []any{testDigestValue("b")},
		"cache_outcome":          "hit_exact",
		"fetched_bytes":          0,
		"registry_request_count": 0,
	}
}

func invalidDependencyFetchRegistryResponseWithoutPayloadDigests() map[string]any {
	response := validDependencyFetchRegistryResponse()
	delete(response, "payload_digests")
	return response
}

func validDependencyCacheHandoffRequest() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.DependencyCacheHandoffRequest",
		"schema_version": "0.1.0",
		"request_id":     "req-dependency-cache-handoff",
		"request_digest": testDigestValue("c"),
		"consumer_role":  "workspace",
	}
}

func invalidDependencyCacheHandoffRequestWithoutConsumerRole() map[string]any {
	request := validDependencyCacheHandoffRequest()
	delete(request, "consumer_role")
	return request
}

func validDependencyCacheHandoffMetadata() map[string]any {
	return map[string]any{
		"schema_id":            "runecode.protocol.v0.DependencyCacheHandoffMetadata",
		"schema_version":       "0.1.0",
		"request_digest":       testDigestValue("d"),
		"resolved_unit_digest": testDigestValue("e"),
		"manifest_digest":      testDigestValue("f"),
		"payload_digests":      []any{testDigestValue("1")},
		"materialization_mode": "derived_read_only",
		"handoff_mode":         "broker_internal_artifact_handoff",
	}
}

func invalidDependencyCacheHandoffMetadataWithUnsupportedMode() map[string]any {
	metadata := validDependencyCacheHandoffMetadata()
	metadata["handoff_mode"] = "unsupported_mode"
	return metadata
}

func validDependencyCacheHandoffResponseFound() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.DependencyCacheHandoffResponse",
		"schema_version": "0.1.0",
		"request_id":     "req-dependency-cache-handoff",
		"found":          true,
		"handoff":        validDependencyCacheHandoffMetadata(),
	}
}

func validDependencyCacheHandoffResponseNotFound() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.DependencyCacheHandoffResponse",
		"schema_version": "0.1.0",
		"request_id":     "req-dependency-cache-handoff",
		"found":          false,
	}
}

func invalidDependencyCacheHandoffResponseFoundWithoutHandoff() map[string]any {
	response := validDependencyCacheHandoffResponseFound()
	delete(response, "handoff")
	return response
}
