package protocolschema

import (
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func TestProviderSubstrateSchemasValidateSharedProfileModel(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	schemas := compileProviderSubstrateSchemas(t, bundle)
	assertValidProviderSubstrateFixtures(t, schemas)
	assertInvalidProviderSubstrateFixturesRejected(t, schemas)
}

type providerSubstrateSchemas struct {
	profile              *jsonschema.Schema
	material             *jsonschema.Schema
	catalog              *jsonschema.Schema
	readiness            *jsonschema.Schema
	setupSession         *jsonschema.Schema
	validationBeginReq   *jsonschema.Schema
	validationBeginResp  *jsonschema.Schema
	validationCommitReq  *jsonschema.Schema
	validationCommitResp *jsonschema.Schema
}

func compileProviderSubstrateSchemas(t *testing.T, bundle compiledBundle) providerSubstrateSchemas {
	t.Helper()
	return providerSubstrateSchemas{
		profile:              mustCompileObjectSchema(t, bundle, "objects/ProviderProfile.schema.json"),
		material:             mustCompileObjectSchema(t, bundle, "objects/ProviderAuthMaterial.schema.json"),
		catalog:              mustCompileObjectSchema(t, bundle, "objects/ProviderModelCatalogPosture.schema.json"),
		readiness:            mustCompileObjectSchema(t, bundle, "objects/ProviderReadinessPosture.schema.json"),
		setupSession:         mustCompileObjectSchema(t, bundle, "objects/ProviderSetupSession.schema.json"),
		validationBeginReq:   mustCompileObjectSchema(t, bundle, "objects/ProviderValidationBeginRequest.schema.json"),
		validationBeginResp:  mustCompileObjectSchema(t, bundle, "objects/ProviderValidationBeginResponse.schema.json"),
		validationCommitReq:  mustCompileObjectSchema(t, bundle, "objects/ProviderValidationCommitRequest.schema.json"),
		validationCommitResp: mustCompileObjectSchema(t, bundle, "objects/ProviderValidationCommitResponse.schema.json"),
	}
}

func assertValidProviderSubstrateFixtures(t *testing.T, schemas providerSubstrateSchemas) {
	t.Helper()
	assertSchemaValid(t, schemas.material, validProviderAuthMaterial(), "ProviderAuthMaterial")
	assertSchemaValid(t, schemas.catalog, validProviderModelCatalogPosture(), "ProviderModelCatalogPosture")
	assertSchemaValid(t, schemas.readiness, validProviderReadinessPosture(), "ProviderReadinessPosture")
	assertSchemaValid(t, schemas.setupSession, validProviderSetupSession(), "ProviderSetupSession")
	assertSchemaValid(t, schemas.validationBeginReq, validProviderValidationBeginRequest(), "ProviderValidationBeginRequest")
	assertSchemaValid(t, schemas.validationBeginResp, validProviderValidationBeginResponse(), "ProviderValidationBeginResponse")
	assertSchemaValid(t, schemas.validationCommitReq, validProviderValidationCommitRequest(), "ProviderValidationCommitRequest")
	assertSchemaValid(t, schemas.validationCommitResp, validProviderValidationCommitResponse(), "ProviderValidationCommitResponse")
	assertSchemaValid(t, schemas.profile, validProviderProfile(), "ProviderProfile")
}

func assertInvalidProviderSubstrateFixturesRejected(t *testing.T, schemas providerSubstrateSchemas) {
	t.Helper()
	invalidDestination := validProviderProfile()
	invalidDestination["destination_identity"].(map[string]any)["descriptor_kind"] = "auth_provider"
	assertSchemaInvalid(t, schemas.profile, invalidDestination, "ProviderProfile accepted non-model destination_identity descriptor_kind")

	invalidMaterial := validProviderAuthMaterial()
	invalidMaterial["material_kind"] = "raw_api_key"
	assertSchemaInvalid(t, schemas.material, invalidMaterial, "ProviderAuthMaterial accepted unsupported material_kind")

	invalidCatalog := validProviderModelCatalogPosture()
	invalidCatalog["selection_authority"] = "provider_discovery"
	assertSchemaInvalid(t, schemas.catalog, invalidCatalog, "ProviderModelCatalogPosture accepted non-manual selection_authority")
}

func assertSchemaValid(t *testing.T, schema *jsonschema.Schema, value map[string]any, label string) {
	t.Helper()
	if err := schema.Validate(value); err != nil {
		t.Fatalf("%s valid fixture failed validation: %v", label, err)
	}
}

func assertSchemaInvalid(t *testing.T, schema *jsonschema.Schema, value map[string]any, message string) {
	t.Helper()
	if err := schema.Validate(value); err == nil {
		t.Fatal(message)
	}
}

func validProviderSetupSession() map[string]any {
	return map[string]any{
		"schema_id":             "runecode.protocol.v0.ProviderSetupSession",
		"schema_version":        "0.1.0",
		"setup_session_id":      "provider-setup-session-1",
		"provider_profile_id":   "provider-profile-123",
		"provider_family":       "openai_compatible",
		"supported_auth_modes":  []any{"direct_credential"},
		"current_phase":         "readiness_committed",
		"current_auth_mode":     "direct_credential",
		"validation_status":     "succeeded",
		"validation_attempt_id": "provider-validation-attempt-1",
		"readiness_committed":   true,
		"secret_ingress_ready":  false,
		"created_at":            "2026-04-18T12:00:00Z",
		"updated_at":            "2026-04-18T12:03:00Z",
	}
}

func validProviderValidationBeginRequest() map[string]any {
	return map[string]any{
		"schema_id":           "runecode.protocol.v0.ProviderValidationBeginRequest",
		"schema_version":      "0.1.0",
		"request_id":          "req-provider-validation-begin",
		"provider_profile_id": "provider-profile-123",
	}
}

func validProviderValidationBeginResponse() map[string]any {
	return map[string]any{
		"schema_id":             "runecode.protocol.v0.ProviderValidationBeginResponse",
		"schema_version":        "0.1.0",
		"request_id":            "req-provider-validation-begin",
		"provider_profile_id":   "provider-profile-123",
		"validation_attempt_id": "provider-validation-attempt-1",
		"setup_session":         validProviderSetupSession(),
		"profile":               validProviderProfile(),
	}
}

func validProviderValidationCommitRequest() map[string]any {
	return map[string]any{
		"schema_id":             "runecode.protocol.v0.ProviderValidationCommitRequest",
		"schema_version":        "0.1.0",
		"request_id":            "req-provider-validation-commit",
		"provider_profile_id":   "provider-profile-123",
		"validation_attempt_id": "provider-validation-attempt-1",
		"connectivity_state":    "reachable",
		"compatibility_state":   "compatible",
	}
}

func validProviderValidationCommitResponse() map[string]any {
	return map[string]any{
		"schema_id":             "runecode.protocol.v0.ProviderValidationCommitResponse",
		"schema_version":        "0.1.0",
		"request_id":            "req-provider-validation-commit",
		"provider_profile_id":   "provider-profile-123",
		"validation_attempt_id": "provider-validation-attempt-1",
		"validation_outcome":    "succeeded",
		"setup_session":         validProviderSetupSession(),
		"profile":               validProviderProfile(),
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
