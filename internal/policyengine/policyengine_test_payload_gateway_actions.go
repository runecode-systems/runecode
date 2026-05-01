package policyengine

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func newGatewayRelevantArtifactHash(seed string) []trustpolicy.Digest {
	return []trustpolicy.Digest{{HashAlg: "sha256", Hash: strings.Repeat(seed, 64)}}
}

func validGatewayEgressActionRequest(capabilityID string, roleFamily string, roleKind string, gatewayRoleKind string, destinationKind string, actionKind string) ActionRequest {
	payloadHash := gatewayPayloadHash()
	action := newActionRequest(
		actionKind,
		capabilityID,
		actionPayloadGatewaySchemaID,
		newSchemaPayload(actionPayloadGatewaySchemaID, map[string]any{
			"gateway_role_kind": gatewayRoleKind,
			"destination_kind":  destinationKind,
			"destination_ref":   "provider.example.com",
			"egress_data_class": "spec_text",
			"operation":         "invoke_model",
			"timeout_seconds":   float64(60),
			"payload_hash":      payloadHash,
			"audit_context":     validGatewayAuditContext(payloadHash),
			"quota_context":     validGatewayQuotaContextTokenMetered(),
		}),
		roleFamily,
		roleKind,
	)
	action.RelevantArtifactHashes = newGatewayRelevantArtifactHash("f")
	return action
}

func validDependencyFetchActionRequest(capabilityID string, roleKind string, refName string) ActionRequest {
	dependencyRequest := validDependencyFetchRequest(refName)
	requestHashIdentity, err := canonicalHashValue(dependencyRequest)
	if err != nil {
		panic(err)
	}
	payloadHash := mustDigestObject(requestHashIdentity)
	auditContext := validGatewayDependencyAuditContext(payloadHash)
	action := newActionRequest(
		ActionKindDependencyFetch,
		capabilityID,
		actionPayloadGatewaySchemaID,
		newSchemaPayload(actionPayloadGatewaySchemaID, map[string]any{
			"gateway_role_kind":  "dependency-fetch",
			"destination_kind":   "package_registry",
			"destination_ref":    refName + ".example.com",
			"egress_data_class":  "spec_text",
			"operation":          "fetch_dependency",
			"timeout_seconds":    float64(60),
			"payload_hash":       payloadHash,
			"audit_context":      auditContext,
			"dependency_request": dependencyRequest,
			"quota_context": map[string]any{
				"schema_id":             "runecode.protocol.v0.GatewayQuotaContext",
				"schema_version":        "0.1.0",
				"quota_profile_kind":    "request_entitlement",
				"phase":                 "admission",
				"enforce_during_stream": false,
				"meters": map[string]any{
					"request_units":     float64(1),
					"entitlement_units": float64(1),
				},
			},
		}),
		"gateway",
		roleKind,
	)
	action.RelevantArtifactHashes = newGatewayRelevantArtifactHash("e")
	return action
}

func validDependencyFetchRequest(refName string) map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.DependencyFetchRequest",
		"schema_version": "0.1.0",
		"request_kind":   "package_version_fetch",
		"registry_identity": map[string]any{
			"schema_id":                destinationDescriptorSchemaID,
			"schema_version":           destinationDescriptorVersion,
			"descriptor_kind":          "package_registry",
			"canonical_host":           refName + ".example.com",
			"provider_or_namespace":    "npm",
			"tls_required":             true,
			"private_range_blocking":   "enforced",
			"dns_rebinding_protection": "enforced",
		},
		"ecosystem":       "npm",
		"package_name":    "left-pad",
		"package_version": "1.3.0",
	}
}

func validGitRemoteMutationActionRequest(capabilityID string, operation string) ActionRequest {
	gitRequest := validGitTypedRefUpdateRequest()
	if operation == "git_pull_request_create" {
		gitRequest = validGitTypedPullRequestCreateRequest()
	}
	requestHashIdentity, err := canonicalHashValue(gitRequest)
	if err != nil {
		panic(err)
	}
	payloadHash := mustDigestObject(requestHashIdentity)
	auditContext := validGitRemoteMutationAuditContext(payloadHash)
	runtimeProof := validGitRemoteMutationRuntimeProof(payloadHash)
	action := newActionRequest(
		ActionKindGatewayEgress,
		capabilityID,
		actionPayloadGatewaySchemaID,
		newSchemaPayload(actionPayloadGatewaySchemaID, map[string]any{
			"gateway_role_kind": "git-gateway",
			"destination_kind":  "git_remote",
			"destination_ref":   "git.example.com/org/repo",
			"egress_data_class": "diffs",
			"operation":         operation,
			"payload_hash":      payloadHash,
			"audit_context":     auditContext,
			"git_request":       gitRequest,
			"git_runtime_proof": runtimeProof,
		}),
		"gateway",
		"git-gateway",
	)
	action.RelevantArtifactHashes = newGatewayRelevantArtifactHash("9")
	return action
}

func validExternalAnchorSubmitActionRequest(capabilityID string, destinationRef string, targetDescriptorDigest string) ActionRequest {
	externalRequest := validExternalAnchorSubmitRequest(targetDescriptorDigest)
	requestHashIdentity, err := canonicalHashValue(externalRequest)
	if err != nil {
		panic(err)
	}
	payloadHash := mustDigestObject(requestHashIdentity)
	auditContext := validGitRemoteMutationAuditContext(payloadHash)
	action := newActionRequest(
		ActionKindGatewayEgress,
		capabilityID,
		actionPayloadGatewaySchemaID,
		newSchemaPayload(actionPayloadGatewaySchemaID, map[string]any{
			"gateway_role_kind":       "git-gateway",
			"destination_kind":        "git_remote",
			"destination_ref":         destinationRef,
			"egress_data_class":       "audit_events",
			"operation":               "external_anchor_submit",
			"payload_hash":            payloadHash,
			"audit_context":           auditContext,
			"external_anchor_request": externalRequest,
		}),
		"gateway",
		"git-gateway",
	)
	action.RelevantArtifactHashes = newGatewayRelevantArtifactHash("8")
	return action
}

func validExternalAnchorSubmitRequest(targetDescriptorDigest string) map[string]any {
	targetDescriptor := map[string]any{
		"descriptor_schema_id":   "runecode.protocol.audit.anchor_target.transparency_log.v0",
		"log_id":                 "policyengine-test-log",
		"log_public_key_digest":  mustDigestObject("sha256:" + strings.Repeat("d", 64)),
		"entry_encoding_profile": "jcs_v1",
	}
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.ExternalAnchorSubmitRequest",
		"schema_version":           "0.1.0",
		"request_kind":             "external_anchor_submit_v0",
		"target_kind":              "transparency_log",
		"target_descriptor":        targetDescriptor,
		"target_descriptor_digest": mustDigestObject(targetDescriptorDigest),
		"seal_digest":              mustDigestObject("sha256:" + strings.Repeat("1", 64)),
		"outbound_payload_digest":  mustDigestObject("sha256:" + strings.Repeat("2", 64)),
	}
}

func validGitTypedPullRequestCreateRequest() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GitPullRequestCreateRequest",
		"schema_version": "0.1.0",
		"request_kind":   "git_pull_request_create",
		"base_repository_identity": map[string]any{
			"schema_id":                destinationDescriptorSchemaID,
			"schema_version":           destinationDescriptorVersion,
			"descriptor_kind":          "git_remote",
			"canonical_host":           "git.example.com",
			"git_repository_identity":  "git.example.com/org/repo",
			"provider_or_namespace":    "org/repo",
			"tls_required":             true,
			"private_range_blocking":   "enforced",
			"dns_rebinding_protection": "enforced",
		},
		"base_ref": "refs/heads/main",
		"head_repository_identity": map[string]any{
			"schema_id":                destinationDescriptorSchemaID,
			"schema_version":           destinationDescriptorVersion,
			"descriptor_kind":          "git_remote",
			"canonical_host":           "git.example.com",
			"git_repository_identity":  "git.example.com/org/repo",
			"provider_or_namespace":    "org/repo",
			"tls_required":             true,
			"private_range_blocking":   "enforced",
			"dns_rebinding_protection": "enforced",
		},
		"head_ref":                          "refs/heads/runecode/feature-1",
		"title":                             "Apply approved patch flow",
		"body":                              "Created from typed pull-request contract.",
		"head_commit_or_tree_hash":          mustDigestObject("sha256:" + strings.Repeat("9", 64)),
		"referenced_patch_artifact_digests": []any{mustDigestObject("sha256:" + strings.Repeat("7", 64))},
		"expected_result_tree_hash":         mustDigestObject("sha256:" + strings.Repeat("6", 64)),
	}
}

func validAllowlistPayloadForGateway(entry string, gatewayRole string, descriptorKind string, operation string, dataClass string) map[string]any {
	entryPayload := map[string]any{
		"schema_id":                   gatewayScopeRuleSchemaID,
		"schema_version":              gatewayScopeRuleVersion,
		"scope_kind":                  "gateway_destination",
		"gateway_role_kind":           gatewayRole,
		"destination":                 validDestinationDescriptorForKind(entry, descriptorKind),
		"permitted_operations":        []any{operation},
		"allowed_egress_data_classes": []any{dataClass},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         float64(120),
		"max_response_bytes":          float64(16777216),
	}
	if descriptorKind == "git_remote" {
		entryPayload["git_ref_update_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "exact", "ref": "refs/heads/main"}}}
		entryPayload["git_tag_update_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "prefix_glob", "prefix": "refs/tags/releases/"}}}
		entryPayload["git_pull_request_base_ref_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "exact", "ref": "refs/heads/main"}}}
		entryPayload["git_pull_request_head_namespace_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "prefix_glob", "prefix": "refs/heads/runecode/"}}}
	}
	return map[string]any{
		"schema_id":       policyAllowlistSchemaID,
		"schema_version":  policyAllowlistSchemaVersion,
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": gatewayScopeRuleSchemaID,
		"entries":         []any{entryPayload},
	}
}

func validDestinationDescriptorForKind(name, kind string) map[string]any {
	destination := map[string]any{
		"schema_id":                destinationDescriptorSchemaID,
		"schema_version":           destinationDescriptorVersion,
		"descriptor_kind":          kind,
		"canonical_host":           name + ".example.com",
		"provider_or_namespace":    name,
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
	if kind == "git_remote" {
		destination["git_repository_identity"] = name + ".example.com/org/repo"
	}
	return destination
}
