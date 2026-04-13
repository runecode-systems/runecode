package policyengine

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

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
	action.RelevantArtifactHashes = []trustpolicy.Digest{{HashAlg: "sha256", Hash: strings.Repeat("f", 64)}}
	return action
}

func gatewayPayloadHash() map[string]any {
	return mustDigestObject("sha256:" + strings.Repeat("f", 64))
}

func validGatewayAuditContext(requestHash map[string]any) map[string]any {
	return map[string]any{
		"schema_id":            "runecode.protocol.v0.GatewayAuditContext",
		"schema_version":       "0.1.0",
		"outbound_bytes":       float64(1024),
		"started_at":           "2026-03-13T12:00:00Z",
		"completed_at":         "2026-03-13T12:00:01Z",
		"outcome":              "succeeded",
		"request_hash":         requestHash,
		"response_hash":        mustDigestObject("sha256:" + strings.Repeat("a", 64)),
		"lease_id":             "lease-model-1",
		"policy_decision_hash": mustDigestObject("sha256:" + strings.Repeat("b", 64)),
	}
}

func validGatewayQuotaContextTokenMetered() map[string]any {
	return map[string]any{
		"schema_id":             "runecode.protocol.v0.GatewayQuotaContext",
		"schema_version":        "0.1.0",
		"quota_profile_kind":    "token_metered_api",
		"phase":                 "admission",
		"enforce_during_stream": false,
		"meters": map[string]any{
			"input_tokens":  float64(512),
			"output_tokens": float64(128),
		},
	}
}

func validDependencyFetchActionRequest(capabilityID string, roleKind string, refName string) ActionRequest {
	payloadHash := mustDigestObject("sha256:" + strings.Repeat("e", 64))
	auditContext := validGatewayDependencyAuditContext(payloadHash)
	action := newActionRequest(
		ActionKindDependencyFetch,
		capabilityID,
		actionPayloadGatewaySchemaID,
		newSchemaPayload(actionPayloadGatewaySchemaID, map[string]any{
			"gateway_role_kind": "dependency-fetch",
			"destination_kind":  "package_registry",
			"destination_ref":   refName + ".example.com",
			"egress_data_class": "spec_text",
			"operation":         "fetch_dependency",
			"timeout_seconds":   float64(60),
			"payload_hash":      payloadHash,
			"audit_context":     auditContext,
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
	action.RelevantArtifactHashes = []trustpolicy.Digest{{HashAlg: "sha256", Hash: strings.Repeat("e", 64)}}
	return action
}

func validGatewayDependencyAuditContext(requestHash map[string]any) map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GatewayAuditContext",
		"schema_version": "0.1.0",
		"outbound_bytes": float64(4096),
		"started_at":     "2026-03-13T12:00:00Z",
		"completed_at":   "2026-03-13T12:00:01Z",
		"outcome":        "succeeded",
		"request_hash":   requestHash,
		"response_hash":  mustDigestObject("sha256:" + strings.Repeat("d", 64)),
	}
}

func compileGatewayInputWithOneCapability(roleKind string, capability string, allowlist map[string]any) CompileInput {
	role := validRoleManifestPayload()
	role["role_family"] = "gateway"
	role["role_kind"] = roleKind
	role["capability_opt_ins"] = []any{capability}
	rolePrincipal := role["principal"].(map[string]any)
	rolePrincipal["role_family"] = "gateway"
	rolePrincipal["role_kind"] = roleKind
	role["allowlist_refs"] = []any{mustDigestObject(testAllowlistHash(nil, allowlist))}

	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{capability}
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_family"] = "gateway"
	runPrincipal["role_kind"] = roleKind
	run["allowlist_refs"] = []any{mustDigestObject(testAllowlistHash(nil, allowlist))}

	return CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(nil, role, ""),
		RunManifest:     testManifestInput(nil, run, ""),
		Allowlists:      []ManifestInput{testManifestInput(nil, allowlist, "")},
	}
}

func validAllowlistPayloadForGateway(entry string, gatewayRole string, descriptorKind string, operation string, dataClass string) map[string]any {
	return map[string]any{
		"schema_id":       policyAllowlistSchemaID,
		"schema_version":  policyAllowlistSchemaVersion,
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": gatewayScopeRuleSchemaID,
		"entries": []any{map[string]any{
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
		}},
	}
}

func validDestinationDescriptorForKind(name, kind string) map[string]any {
	return map[string]any{
		"schema_id":                destinationDescriptorSchemaID,
		"schema_version":           destinationDescriptorVersion,
		"descriptor_kind":          kind,
		"canonical_host":           name + ".example.com",
		"provider_or_namespace":    name,
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
}
