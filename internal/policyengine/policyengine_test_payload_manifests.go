package policyengine

func validRoleManifestPayload() map[string]any {
	return map[string]any{
		"schema_id":          roleManifestSchemaID,
		"schema_version":     roleManifestSchemaVersion,
		"principal":          map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1", "role_family": "workspace", "role_kind": "workspace-edit"},
		"role_family":        "workspace",
		"role_kind":          "workspace-edit",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_run", "cap_stage", "always_denied"},
		"allowlist_refs":     []any{mustDigestObject(mustAllowlistHash(validAllowlistPayload("allowlist-a")))},
		"signatures":         []any{map[string]any{"alg": "ed25519", "key_id": "key_sha256", "key_id_value": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "signature": "c2ln"}},
	}
}

func validRunCapabilityManifestPayload() map[string]any {
	return map[string]any{
		"schema_id":          capabilityManifestSchemaID,
		"schema_version":     capabilityManifestVersion,
		"principal":          map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1", "role_family": "workspace", "role_kind": "workspace-edit", "run_id": "run-1"},
		"manifest_scope":     "run",
		"run_id":             "run-1",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_run", "cap_stage"},
		"allowlist_refs":     []any{mustDigestObject(mustAllowlistHash(validAllowlistPayload("allowlist-b")))},
		"signatures":         []any{map[string]any{"alg": "ed25519", "key_id": "key_sha256", "key_id_value": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "signature": "c2ln"}},
	}
}

func validStageCapabilityManifestPayload() map[string]any {
	return map[string]any{
		"schema_id":          capabilityManifestSchemaID,
		"schema_version":     capabilityManifestVersion,
		"principal":          map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1", "role_family": "workspace", "role_kind": "workspace-edit", "run_id": "run-1", "stage_id": "stage-1"},
		"manifest_scope":     "stage",
		"run_id":             "run-1",
		"stage_id":           "stage-1",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_stage"},
		"allowlist_refs":     []any{mustDigestObject(mustAllowlistHash(validAllowlistPayload("allowlist-c")))},
		"signatures":         []any{map[string]any{"alg": "ed25519", "key_id": "key_sha256", "key_id_value": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "signature": "c2ln"}},
	}
}

func validAllowlistPayload(entry string) map[string]any {
	return map[string]any{
		"schema_id":       policyAllowlistSchemaID,
		"schema_version":  policyAllowlistSchemaVersion,
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": gatewayScopeRuleSchemaID,
		"entries": []any{map[string]any{
			"schema_id":                   gatewayScopeRuleSchemaID,
			"schema_version":              gatewayScopeRuleVersion,
			"scope_kind":                  "gateway_destination",
			"gateway_role_kind":           "model-gateway",
			"destination":                 validDestinationDescriptor(entry),
			"permitted_operations":        []any{"invoke_model"},
			"allowed_egress_data_classes": []any{"spec_text"},
			"redirect_posture":            "allowlist_only",
		}},
	}
}

func validDestinationDescriptor(name string) map[string]any {
	return map[string]any{
		"schema_id":                destinationDescriptorSchemaID,
		"schema_version":           destinationDescriptorVersion,
		"descriptor_kind":          "model_endpoint",
		"canonical_host":           name + ".example.com",
		"provider_or_namespace":    name,
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
}

func validRuleSetPayload() map[string]any {
	return map[string]any{
		"schema_id":      policyRuleSetSchemaID,
		"schema_version": policyRuleSetSchemaVersion,
		"rules": []any{
			map[string]any{"rule_id": "allow-1", "effect": "allow", "action_kind": "workspace_write", "capability_id": "cap_stage", "reason_code": "allow_manifest_opt_in", "details_schema_id": "runecode.protocol.details.policy.allow.v0"},
		},
	}
}
