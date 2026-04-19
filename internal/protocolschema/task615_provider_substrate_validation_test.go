package protocolschema

import "testing"

func TestProviderSubstrateSchemasValidateSharedProfileModel(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	profileSchema := mustCompileObjectSchema(t, bundle, "objects/ProviderProfile.schema.json")
	materialSchema := mustCompileObjectSchema(t, bundle, "objects/ProviderAuthMaterial.schema.json")
	catalogSchema := mustCompileObjectSchema(t, bundle, "objects/ProviderModelCatalogPosture.schema.json")
	readinessSchema := mustCompileObjectSchema(t, bundle, "objects/ProviderReadinessPosture.schema.json")

	if err := materialSchema.Validate(validProviderAuthMaterial()); err != nil {
		t.Fatalf("ProviderAuthMaterial valid fixture failed validation: %v", err)
	}
	if err := catalogSchema.Validate(validProviderModelCatalogPosture()); err != nil {
		t.Fatalf("ProviderModelCatalogPosture valid fixture failed validation: %v", err)
	}
	if err := readinessSchema.Validate(validProviderReadinessPosture()); err != nil {
		t.Fatalf("ProviderReadinessPosture valid fixture failed validation: %v", err)
	}
	if err := profileSchema.Validate(validProviderProfile()); err != nil {
		t.Fatalf("ProviderProfile valid fixture failed validation: %v", err)
	}

	invalidDestination := validProviderProfile()
	invalidDestination["destination_identity"].(map[string]any)["descriptor_kind"] = "auth_provider"
	if err := profileSchema.Validate(invalidDestination); err == nil {
		t.Fatal("ProviderProfile accepted non-model destination_identity descriptor_kind")
	}

	invalidMaterial := validProviderAuthMaterial()
	invalidMaterial["material_kind"] = "raw_api_key"
	if err := materialSchema.Validate(invalidMaterial); err == nil {
		t.Fatal("ProviderAuthMaterial accepted unsupported material_kind")
	}

	invalidCatalog := validProviderModelCatalogPosture()
	invalidCatalog["selection_authority"] = "provider_discovery"
	if err := catalogSchema.Validate(invalidCatalog); err == nil {
		t.Fatal("ProviderModelCatalogPosture accepted non-manual selection_authority")
	}
}

func validProviderProfile() map[string]any {
	return map[string]any{
		"schema_id":            "runecode.protocol.v0.ProviderProfile",
		"schema_version":       "0.1.0",
		"provider_profile_id":  "provider-profile-123",
		"display_label":        "OpenAI Production",
		"provider_family":      "openai_compatible",
		"adapter_kind":         "chat_completions_v0",
		"destination_identity": validModelEndpointDestination(),
		"destination_ref":      "api.openai.com/v1",
		"supported_auth_modes": []any{"direct_credential", "oauth_derived", "bridge_session"},
		"current_auth_mode":    "direct_credential",
		"allowlisted_model_ids": []any{
			"gpt-4.1-mini",
			"gpt-4.1",
		},
		"model_catalog_posture": validProviderModelCatalogPosture(),
		"compatibility_posture": "unverified",
		"quota_profile_kind":    "hybrid",
		"request_binding_kind":  "canonical_llm_request_digest",
		"surface_channel":       "broker_local_api",
		"auth_material":         validProviderAuthMaterial(),
		"readiness_posture":     validProviderReadinessPosture(),
		"lifecycle": map[string]any{
			"created_at":                "2026-04-18T12:00:00Z",
			"updated_at":                "2026-04-18T12:01:00Z",
			"last_validation_at":        "2026-04-18T12:01:00Z",
			"validation_attempt_count":  int64(2),
			"last_validation_succeeded": true,
		},
	}
}

func validProviderModelCatalogPosture() map[string]any {
	return map[string]any{
		"schema_id":                   "runecode.protocol.v0.ProviderModelCatalogPosture",
		"schema_version":              "0.1.0",
		"selection_authority":         "manual_allowlist_canonical",
		"discovery_posture":           "advisory",
		"compatibility_probe_posture": "advisory",
		"discovered_model_ids":        []any{"gpt-4.1-mini", "gpt-4.1"},
		"probe_compatible_model_ids":  []any{"gpt-4.1-mini"},
	}
}

func validProviderAuthMaterial() map[string]any {
	return map[string]any{
		"schema_id":        "runecode.protocol.v0.ProviderAuthMaterial",
		"schema_version":   "0.1.0",
		"material_kind":    "direct_credential",
		"material_state":   "present",
		"secret_ref":       "secrets/model-providers/openai/key",
		"lease_policy_ref": "secretsd://lease-policy/model-provider-default",
		"last_rotated_at":  "2026-04-18T12:00:00Z",
	}
}

func validProviderReadinessPosture() map[string]any {
	return map[string]any{
		"schema_id":           "runecode.protocol.v0.ProviderReadinessPosture",
		"schema_version":      "0.1.0",
		"configuration_state": "configured",
		"credential_state":    "present",
		"connectivity_state":  "reachable",
		"compatibility_state": "compatible",
		"effective_readiness": "ready",
	}
}

func validModelEndpointDestination() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.DestinationDescriptor",
		"schema_version":           "0.1.0",
		"descriptor_kind":          "model_endpoint",
		"canonical_host":           "api.openai.com",
		"canonical_path_prefix":    "/v1",
		"provider_or_namespace":    "openai",
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
}
