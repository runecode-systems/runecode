package policyengine

import "strings"

func newActionRequest(actionKind, capabilityID, payloadSchemaID string, payload map[string]any, roleFamily, roleKind string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            actionKind,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: payloadSchemaID,
		ActionPayload:         payload,
		ActorKind:             "daemon",
		RoleFamily:            roleFamily,
		RoleKind:              roleKind,
	}
}

func newSchemaPayload(schemaID string, fields map[string]any) map[string]any {
	payload := map[string]any{
		"schema_id":      schemaID,
		"schema_version": "0.1.0",
	}
	for key, value := range fields {
		payload[key] = value
	}
	return payload
}

func validWorkspaceWriteActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindWorkspaceWrite,
		capabilityID,
		actionPayloadWorkspaceSchemaID,
		newSchemaPayload(actionPayloadWorkspaceSchemaID, map[string]any{
			"target_path": "src/main.go",
			"write_mode":  "update",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validExecutorRunActionRequest(capabilityID string, executorClass string, argv []string) ActionRequest {
	return newActionRequest(
		ActionKindExecutorRun,
		capabilityID,
		actionPayloadExecutorSchemaID,
		newSchemaPayload(actionPayloadExecutorSchemaID, map[string]any{
			"executor_class": executorClass,
			"executor_id":    "workspace-runner",
			"argv":           toAnySlice(argv),
			"network_access": "none",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validGatewayEgressActionRequest(capabilityID string, roleFamily string, roleKind string, gatewayRoleKind string, destinationKind string, actionKind string) ActionRequest {
	return newActionRequest(
		actionKind,
		capabilityID,
		actionPayloadGatewaySchemaID,
		newSchemaPayload(actionPayloadGatewaySchemaID, map[string]any{
			"gateway_role_kind": gatewayRoleKind,
			"destination_kind":  destinationKind,
			"destination_ref":   "provider.example.com",
			"egress_data_class": "spec_text",
			"operation":         "invoke_model",
		}),
		roleFamily,
		roleKind,
	)
}

func validDependencyFetchActionRequest(capabilityID string, roleKind string, refName string) ActionRequest {
	return newActionRequest(
		ActionKindDependencyFetch,
		capabilityID,
		actionPayloadGatewaySchemaID,
		newSchemaPayload(actionPayloadGatewaySchemaID, map[string]any{
			"gateway_role_kind": "dependency-fetch",
			"destination_kind":  "package_registry",
			"destination_ref":   refName + ".example.com",
			"egress_data_class": "spec_text",
			"operation":         "fetch_dependency",
		}),
		"gateway",
		roleKind,
	)
}

func validArtifactReadActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindArtifactRead,
		capabilityID,
		actionPayloadArtifactSchemaID,
		newSchemaPayload(actionPayloadArtifactSchemaID, map[string]any{
			"artifact_hash": mustDigestObject("sha256:" + strings.Repeat("3", 64)),
			"read_mode":     "head",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validPromotionActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindPromotion,
		capabilityID,
		actionPayloadPromotionSchemaID,
		newSchemaPayload(actionPayloadPromotionSchemaID, map[string]any{
			"promotion_kind":       "excerpt",
			"source_artifact_hash": mustDigestObject("sha256:" + strings.Repeat("4", 64)),
			"target_data_class":    "approved_file_excerpts",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validBackendPostureActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindBackendPosture,
		capabilityID,
		actionPayloadBackendSchemaID,
		newSchemaPayload(actionPayloadBackendSchemaID, map[string]any{
			"backend_class":     "microvm",
			"change_kind":       "select_backend",
			"requested_posture": "microvm_default",
			"requires_opt_in":   false,
		}),
		"workspace",
		"workspace-edit",
	)
}

func validGateOverrideActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindGateOverride,
		capabilityID,
		actionPayloadGateSchemaID,
		newSchemaPayload(actionPayloadGateSchemaID, map[string]any{
			"gate_id":                      "policy_engine_gate",
			"gate_kind":                    "policy",
			"gate_version":                 "1.0.0",
			"gate_attempt_id":              "gate-attempt-1",
			"overridden_failed_result_ref": "sha256:" + strings.Repeat("2", 64),
			"policy_context_hash":          "sha256:" + strings.Repeat("3", 64),
			"override_mode":                "break_glass",
			"justification":                "Emergency trust maintenance",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validStageSummarySignOffActionRequest(capabilityID, summaryHash string) ActionRequest {
	return newActionRequest(
		ActionKindStageSummarySign,
		capabilityID,
		actionPayloadStageSchemaID,
		newSchemaPayload(actionPayloadStageSchemaID, map[string]any{
			"run_id":             "run-1",
			"stage_id":           "stage-1",
			"stage_summary_hash": mustDigestObject(summaryHash),
			"approval_profile":   "moderate",
			"summary_revision":   float64(1),
		}),
		"workspace",
		"workspace-edit",
	)
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
