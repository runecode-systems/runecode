package brokerapi

import (
	"context"
	"testing"
)

func TestHandleReadinessGetIncludesConfiguredProviderProfiles(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, err := s.providerSubstrate.upsertProfile(providerProfileFixture("OpenAI Prod", "openai_compatible", "api.openai.com", "/v1"))
	if err != nil {
		t.Fatalf("upsertProfile returned error: %v", err)
	}
	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-provider-profiles",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	assertProjectedProviderProfile(t, resp)
	if err := s.validateResponse(resp, readinessGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(readinessGetResponse) returned error: %v", err)
	}
}

func assertProjectedProviderProfile(t *testing.T, resp ReadinessGetResponse) {
	t.Helper()
	if got := len(resp.Readiness.ProviderProfiles); got != 1 {
		t.Fatalf("provider_profiles count = %d, want 1", got)
	}
	profile := resp.Readiness.ProviderProfiles[0]
	if profile.ProviderProfileID == "" {
		t.Fatal("provider_profile_id empty, want stable identity")
	}
	if got := profile.DestinationIdentity.DescriptorKind; got != "model_endpoint" {
		t.Fatalf("destination_identity.descriptor_kind = %q, want model_endpoint", got)
	}
	if got := profile.CurrentAuthMode; got != "direct_credential" {
		t.Fatalf("current_auth_mode = %q, want direct_credential", got)
	}
	if got := profile.SupportedAuthModes; len(got) != 1 || got[0] != "direct_credential" {
		t.Fatalf("supported_auth_modes = %#v, want [direct_credential]", got)
	}
	if got := profile.ModelCatalogPosture.SelectionAuthority; got != "manual_allowlist_canonical" {
		t.Fatalf("model_catalog_posture.selection_authority = %q, want manual_allowlist_canonical", got)
	}
	if got := profile.AuthMaterial.SecretRef; got != "" {
		t.Fatalf("auth_material.secret_ref = %q, want redacted projection", got)
	}
}
