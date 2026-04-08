package policyengine

import "strings"

func validWorkspaceWriteActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindWorkspaceWrite,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadWorkspaceSchemaID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadWorkspaceSchemaID,
			"schema_version": "0.1.0",
			"target_path":    "src/main.go",
			"write_mode":     "update",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validExecutorRunActionRequest(capabilityID string, executorClass string, argv []string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindExecutorRun,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadExecutorSchemaID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadExecutorSchemaID,
			"schema_version": "0.1.0",
			"executor_class": executorClass,
			"executor_id":    "workspace-runner",
			"argv":           toAnySlice(argv),
			"network_access": "none",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validGatewayEgressActionRequest(capabilityID string, roleFamily string, roleKind string, gatewayRoleKind string, destinationKind string, actionKind string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            actionKind,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadGatewaySchemaID,
		ActionPayload: map[string]any{
			"schema_id":         actionPayloadGatewaySchemaID,
			"schema_version":    "0.1.0",
			"gateway_role_kind": gatewayRoleKind,
			"destination_kind":  destinationKind,
			"destination_ref":   "provider.example.com",
			"egress_data_class": "spec_text",
			"operation":         "invoke_model",
		},
		ActorKind:  "daemon",
		RoleFamily: roleFamily,
		RoleKind:   roleKind,
	}
}

func validDependencyFetchActionRequest(capabilityID string, roleKind string, refName string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindDependencyFetch,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadGatewaySchemaID,
		ActionPayload: map[string]any{
			"schema_id":         actionPayloadGatewaySchemaID,
			"schema_version":    "0.1.0",
			"gateway_role_kind": "dependency-fetch",
			"destination_kind":  "package_registry",
			"destination_ref":   refName + ".example.com",
			"egress_data_class": "spec_text",
			"operation":         "fetch_dependency",
		},
		ActorKind:  "daemon",
		RoleFamily: "gateway",
		RoleKind:   roleKind,
	}
}

func validArtifactReadActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindArtifactRead,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadArtifactSchemaID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadArtifactSchemaID,
			"schema_version": "0.1.0",
			"artifact_hash":  mustDigestObject("sha256:" + strings.Repeat("3", 64)),
			"read_mode":      "head",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validPromotionActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindPromotion,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadPromotionSchemaID,
		ActionPayload: map[string]any{
			"schema_id":            actionPayloadPromotionSchemaID,
			"schema_version":       "0.1.0",
			"promotion_kind":       "excerpt",
			"source_artifact_hash": mustDigestObject("sha256:" + strings.Repeat("4", 64)),
			"target_data_class":    "approved_file_excerpts",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validBackendPostureActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindBackendPosture,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadBackendSchemaID,
		ActionPayload: map[string]any{
			"schema_id":         actionPayloadBackendSchemaID,
			"schema_version":    "0.1.0",
			"backend_class":     "microvm",
			"change_kind":       "select_backend",
			"requested_posture": "microvm_default",
			"requires_opt_in":   false,
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validGateOverrideActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindGateOverride,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadGateSchemaID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadGateSchemaID,
			"schema_version": "0.1.0",
			"gate_name":      "policy-engine",
			"override_mode":  "break_glass",
			"justification":  "Emergency trust maintenance",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validStageSummarySignOffActionRequest(capabilityID, summaryHash string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindStageSummarySign,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadStageSchemaID,
		ActionPayload: map[string]any{
			"schema_id":          actionPayloadStageSchemaID,
			"schema_version":     "0.1.0",
			"run_id":             "run-1",
			"stage_id":           "stage-1",
			"stage_summary_hash": mustDigestObject(summaryHash),
			"approval_profile":   "moderate",
			"summary_revision":   float64(1),
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
	}
}

func validSecretAccessActionRequest(capabilityID string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            ActionKindSecretAccess,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: actionPayloadSecretAccessID,
		ActionPayload: map[string]any{
			"schema_id":      actionPayloadSecretAccessID,
			"schema_version": "0.1.0",
			"secret_ref":     "secrets/prod/db-password",
			"access_mode":    "lease_issue",
		},
		ActorKind:  "daemon",
		RoleFamily: "workspace",
		RoleKind:   "workspace-edit",
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
	role["allowlist_refs"] = []any{mustDigestObject(mustAllowlistHash(allowlist))}

	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{capability}
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_family"] = "gateway"
	runPrincipal["role_kind"] = roleKind
	run["allowlist_refs"] = []any{mustDigestObject(mustAllowlistHash(allowlist))}

	return CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    mustManifestInput(role),
		RunManifest:     mustManifestInput(run),
		Allowlists:      []ManifestInput{mustManifestInput(allowlist)},
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
