package protocolschema

func validPolicyAllowlist() map[string]any {
	return map[string]any{
		"schema_id":       "runecode.protocol.v0.PolicyAllowlist",
		"schema_version":  "0.1.0",
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": "runecode.protocol.v0.GatewayScopeRule",
		"entries": []any{
			validGatewayScopeRule("provider-a"),
			validGatewayScopeRule("provider-b"),
		},
	}
}

func invalidPolicyAllowlistKind() map[string]any {
	allowlist := validPolicyAllowlist()
	allowlist["allowlist_kind"] = "gateway_destination"
	return allowlist
}

func invalidPolicyAllowlistEntrySchemaID() map[string]any {
	allowlist := validPolicyAllowlist()
	allowlist["entry_schema_id"] = "runecode.protocol.v0.DestinationDescriptor"
	return allowlist
}

func validGatewayScopeRule(provider string) map[string]any {
	destination := validDestinationDescriptor(provider)
	rule := map[string]any{
		"schema_id":                   "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":              "0.1.0",
		"scope_kind":                  "gateway_destination",
		"entry_id":                    "model_default",
		"gateway_role_kind":           "model-gateway",
		"destination":                 destination,
		"permitted_operations":        []any{"invoke_model"},
		"allowed_egress_data_classes": []any{"spec_text"},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         120,
		"max_response_bytes":          16777216,
	}
	if provider == "git" {
		destination["descriptor_kind"] = "git_remote"
		destination["canonical_host"] = "git.example.test"
		destination["git_repository_identity"] = "git.example.test/org/repo.git"
		rule["gateway_role_kind"] = "git-gateway"
		rule["permitted_operations"] = []any{"change_allowlist"}
		rule["allowed_egress_data_classes"] = []any{"diffs"}
		rule["git_ref_update_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "exact", "ref": "refs/heads/main"}}}
		rule["git_tag_update_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "prefix_glob", "prefix": "refs/tags/releases/"}}}
		rule["git_pull_request_base_ref_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "exact", "ref": "refs/heads/main"}}}
		rule["git_pull_request_head_namespace_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "prefix_glob", "prefix": "refs/heads/runecode/"}}}
	}
	return rule
}

func invalidGatewayScopeRuleKind() map[string]any {
	rule := validGatewayScopeRule("provider-a")
	rule["scope_kind"] = "gateway_destination_legacy"
	return rule
}

func validDestinationDescriptor(provider string) map[string]any {
	descriptor := map[string]any{
		"schema_id":                "runecode.protocol.v0.DestinationDescriptor",
		"schema_version":           "0.1.0",
		"descriptor_kind":          "model_endpoint",
		"canonical_host":           provider + ".example.com",
		"provider_or_namespace":    provider,
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
	if provider == "git" {
		descriptor["descriptor_kind"] = "git_remote"
		descriptor["canonical_host"] = "git.example.com"
		descriptor["provider_or_namespace"] = "org/repo"
		descriptor["git_repository_identity"] = "git.example.com/org/repo"
	}
	return descriptor
}

func invalidDestinationDescriptorKind() map[string]any {
	descriptor := validDestinationDescriptor("provider-a")
	descriptor["descriptor_kind"] = "raw_url"
	return descriptor
}

func invalidDestinationDescriptorGitMissingRepositoryIdentity() map[string]any {
	descriptor := validDestinationDescriptor("provider-a")
	descriptor["descriptor_kind"] = "git_remote"
	delete(descriptor, "git_repository_identity")
	return descriptor
}

func validActionPayloadGatewayEgressRequestOperation() map[string]any {
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.ActionPayloadGatewayEgress",
		"schema_version":    "0.1.0",
		"gateway_role_kind": "model-gateway",
		"destination_kind":  "model_endpoint",
		"destination_ref":   "provider-a.example.com/v1",
		"egress_data_class": "spec_text",
		"operation":         "invoke_model",
		"payload_hash":      testDigestValue("8"),
		"audit_context": map[string]any{
			"schema_id":            "runecode.protocol.v0.GatewayAuditContext",
			"schema_version":       "0.1.0",
			"outbound_bytes":       1024,
			"started_at":           "2026-03-13T12:00:00Z",
			"completed_at":         "2026-03-13T12:00:01Z",
			"outcome":              "succeeded",
			"request_hash":         testDigestValue("8"),
			"response_hash":        testDigestValue("7"),
			"lease_id":             "lease-model-1",
			"policy_decision_hash": testDigestValue("6"),
		},
		"quota_context": map[string]any{
			"schema_id":             "runecode.protocol.v0.GatewayQuotaContext",
			"schema_version":        "0.1.0",
			"quota_profile_kind":    "token_metered_api",
			"phase":                 "admission",
			"enforce_during_stream": false,
			"meters": map[string]any{
				"input_tokens":  512,
				"output_tokens": 128,
			},
		},
	}
}

func validActionPayloadGatewayEgressRequestOperationWithPortAndPath() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["destination_ref"] = "provider-a.example.com:8443/v1/chat/completions"
	return payload
}

func validActionPayloadGatewayEgressDependencyRequestOperation() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["gateway_role_kind"] = "dependency-fetch"
	payload["destination_kind"] = "package_registry"
	payload["destination_ref"] = "registry.example.com/npm"
	payload["operation"] = "fetch_dependency"
	payload["dependency_request"] = validDependencyFetchRequest()
	return payload
}

func validActionPayloadGatewayEgressScopeOperation() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["operation"] = "expand_scope"
	delete(payload, "payload_hash")
	return payload
}

func validActionPayloadGatewayEgressGitRemoteMutationOperation() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["gateway_role_kind"] = "git-gateway"
	payload["destination_kind"] = "git_remote"
	payload["destination_ref"] = "git.example.com/org/repo"
	payload["egress_data_class"] = "diffs"
	payload["operation"] = "git_ref_update"
	delete(payload, "quota_context")
	payload["git_request"] = validGitRefUpdateRequest()
	payload["git_runtime_proof"] = validGitRemoteMutationProofFixture()
	return payload
}

func validGitRemoteMutationProofFixture() map[string]any {
	return map[string]any{
		"schema_id":                 "runecode.protocol.v0.GitRuntimeProof",
		"schema_version":            "0.1.0",
		"typed_request_hash":        testDigestValue("8"),
		"expected_old_object_id":    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"observed_old_object_id":    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"patch_artifact_digests":    []any{testDigestValue("5")},
		"expected_result_tree_hash": testDigestValue("6"),
		"observed_result_tree_hash": testDigestValue("6"),
		"sparse_checkout_applied":   true,
		"drift_detected":            false,
		"destructive_ref_mutation":  false,
		"provider_kind":             "github",
		"evidence_refs":             []any{"artifact:gate-result"},
	}
}

func validActionPayloadGatewayEgressRequestOperationWithTimeout() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["timeout_seconds"] = 60
	return payload
}

func validActionPayloadGatewayEgressStreamQuotaOperation() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	quota := payload["quota_context"].(map[string]any)
	quota["phase"] = "stream"
	quota["enforce_during_stream"] = true
	quota["stream_limit_bytes"] = 2048
	meters := quota["meters"].(map[string]any)
	meters["streamed_bytes"] = 1024
	return payload
}

func invalidActionPayloadGatewayEgressUnknownOperation() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["operation"] = "provider_specific_passthrough"
	return payload
}

func invalidActionPayloadGatewayEgressRequestMissingPayloadHash() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	delete(payload, "payload_hash")
	return payload
}

func invalidActionPayloadGatewayEgressDependencyRequestMissingPayloadHash() map[string]any {
	payload := validActionPayloadGatewayEgressDependencyRequestOperation()
	delete(payload, "payload_hash")
	return payload
}

func invalidActionPayloadGatewayEgressDependencyRequestMissingTypedRequest() map[string]any {
	payload := validActionPayloadGatewayEgressDependencyRequestOperation()
	delete(payload, "dependency_request")
	return payload
}

func invalidActionPayloadGatewayEgressScopeWithPayloadHash() map[string]any {
	payload := validActionPayloadGatewayEgressScopeOperation()
	payload["payload_hash"] = testDigestValue("9")
	return payload
}

func invalidActionPayloadGatewayEgressGitRemoteMutationMissingSummary() map[string]any {
	payload := validActionPayloadGatewayEgressGitRemoteMutationOperation()
	delete(payload, "git_request")
	return payload
}

func invalidActionPayloadGatewayEgressRawURLDestinationRef() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["destination_ref"] = "https://provider-a.example.com/v1/chat/completions?model=test#frag"
	return payload
}

func invalidActionPayloadGatewayEgressTimeoutOutOfBounds() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["timeout_seconds"] = 301
	return payload
}

func invalidActionPayloadGatewayEgressMissingAuditContext() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	delete(payload, "audit_context")
	return payload
}

func invalidActionPayloadGatewayEgressMissingQuotaContext() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	delete(payload, "quota_context")
	return payload
}

func invalidActionPayloadGatewayEgressStreamPhaseWithoutLimit() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	quota := payload["quota_context"].(map[string]any)
	quota["phase"] = "stream"
	quota["enforce_during_stream"] = true
	delete(quota, "stream_limit_bytes")
	return payload
}

func invalidActionPayloadGatewayEgressAuthRequestMissingAuditContext() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["gateway_role_kind"] = "auth-gateway"
	payload["destination_kind"] = "auth_provider"
	payload["operation"] = "exchange_auth_code"
	delete(payload, "audit_context")
	return payload
}

func invalidActionPayloadGatewayEgressAuthRequestMissingQuotaContext() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["gateway_role_kind"] = "auth-gateway"
	payload["destination_kind"] = "auth_provider"
	payload["operation"] = "refresh_auth_token"
	delete(payload, "quota_context")
	return payload
}

func invalidActionPayloadGatewayEgressHybridQuotaMissingEntitlementMeter() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	quota := payload["quota_context"].(map[string]any)
	quota["quota_profile_kind"] = "hybrid"
	meters := quota["meters"].(map[string]any)
	delete(meters, "request_units")
	delete(meters, "entitlement_units")
	meters["input_tokens"] = 256
	return payload
}

func invalidActionPayloadGatewayEgressHybridQuotaMissingTokenMeter() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	quota := payload["quota_context"].(map[string]any)
	quota["quota_profile_kind"] = "hybrid"
	meters := quota["meters"].(map[string]any)
	delete(meters, "input_tokens")
	delete(meters, "output_tokens")
	meters["request_units"] = 1
	return payload
}

func invalidGatewayScopeRuleTimeoutOutOfBounds() map[string]any {
	rule := validGatewayScopeRule("provider-a")
	rule["max_timeout_seconds"] = 301
	return rule
}

func invalidGatewayScopeRuleGitMissingRefPolicies() map[string]any {
	rule := validGatewayScopeRule("git")
	delete(rule, "git_ref_update_policy")
	return rule
}

func validGitCommitIntent() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GitCommitIntent",
		"schema_version": "0.1.0",
		"message": map[string]any{
			"subject": "Apply approved patch",
			"body":    "Includes deterministic trailer rendering from structured identities.",
		},
		"trailers": []any{
			map[string]any{"key": "Signed-off-by", "value": "Signoff Example <signoff@example.com>"},
		},
		"author": map[string]any{
			"display_name": "Author Example",
			"email":        "author@example.com",
		},
		"committer": map[string]any{
			"display_name": "Committer Example",
			"email":        "committer@example.com",
		},
		"signoff": map[string]any{
			"display_name": "Signoff Example",
			"email":        "signoff@example.com",
		},
	}
}

func invalidGitCommitIntentWithoutSignoff() map[string]any {
	intent := validGitCommitIntent()
	delete(intent, "signoff")
	return intent
}

func validGitSignedPatchArtifact() map[string]any {
	return map[string]any{
		"schema_id":                    "runecode.protocol.v0.GitSignedPatchArtifact",
		"schema_version":               "0.1.0",
		"data_class":                   "diffs",
		"base_commit_hash":             testDigestValue("1"),
		"base_tree_hash":               testDigestValue("2"),
		"canonical_patch_payload_hash": testDigestValue("3"),
		"touched_paths":                []any{"README.md", "internal/policyengine/evaluate_gateway_binding.go"},
		"expected_result_tree_hash":    testDigestValue("4"),
		"patch_format":                 "unified_diff",
	}
}

func invalidGitSignedPatchArtifactWithoutDiffsDataClass() map[string]any {
	artifact := validGitSignedPatchArtifact()
	artifact["data_class"] = "approved_file_excerpts"
	return artifact
}

func validGitRefUpdateRequest() map[string]any {
	return map[string]any{
		"schema_id":                         "runecode.protocol.v0.GitRefUpdateRequest",
		"schema_version":                    "0.1.0",
		"request_kind":                      "git_ref_update",
		"repository_identity":               validDestinationDescriptor("git"),
		"target_ref":                        "refs/heads/main",
		"expected_old_ref_hash":             testDigestValue("5"),
		"referenced_patch_artifact_digests": []any{testDigestValue("6")},
		"commit_intent":                     validGitCommitIntent(),
		"expected_result_tree_hash":         testDigestValue("7"),
		"allow_force_push":                  false,
		"allow_ref_deletion":                false,
		"ref_purpose":                       "branch",
	}
}

func invalidGitRefUpdateRequestWithForcePushEnabled() map[string]any {
	request := validGitRefUpdateRequest()
	request["allow_force_push"] = true
	return request
}

func validGitPullRequestCreateRequest() map[string]any {
	return map[string]any{
		"schema_id":                         "runecode.protocol.v0.GitPullRequestCreateRequest",
		"schema_version":                    "0.1.0",
		"request_kind":                      "git_pull_request_create",
		"base_repository_identity":          validDestinationDescriptor("git"),
		"base_ref":                          "refs/heads/main",
		"head_repository_identity":          validDestinationDescriptor("git"),
		"head_ref":                          "refs/heads/runecode/feature-1",
		"title":                             "Apply approved patch flow",
		"body":                              "Created from typed pull-request contract.",
		"head_commit_or_tree_hash":          testDigestValue("8"),
		"referenced_patch_artifact_digests": []any{testDigestValue("9")},
		"expected_result_tree_hash":         testDigestValue("a"),
	}
}

func invalidGitPullRequestCreateRequestWithNonCanonicalHeadRef() map[string]any {
	request := validGitPullRequestCreateRequest()
	request["head_ref"] = "main"
	return request
}
