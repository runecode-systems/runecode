package brokerapi

import (
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestProviderSubstrateSupportsMultipleProfiles(t *testing.T) {
	state := newProviderSubstrateState(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) })
	p1, err := state.upsertProfile(providerProfileFixture("OpenAI Prod", "openai_compatible", "api.openai.com", "/v1"))
	if err != nil {
		t.Fatalf("upsertProfile(p1) error: %v", err)
	}
	p2, err := state.upsertProfile(providerProfileFixture("OpenAI Staging", "openai_compatible", "staging.openai.example", "/v1"))
	if err != nil {
		t.Fatalf("upsertProfile(p2) error: %v", err)
	}
	if p1.ProviderProfileID == p2.ProviderProfileID {
		t.Fatalf("provider_profile_id collision: %q", p1.ProviderProfileID)
	}
	if got := len(state.snapshotProfiles()); got != 2 {
		t.Fatalf("snapshot profile count = %d, want 2", got)
	}
}

func TestProviderSubstrateProfileIDStableAcrossAuthModeChanges(t *testing.T) {
	state := newProviderSubstrateState(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) })
	first, err := state.upsertProfile(providerProfileFixture("Anthropic", "anthropic_compatible", "api.anthropic.com", "/v1"))
	if err != nil {
		t.Fatalf("upsertProfile(first) error: %v", err)
	}
	updated := providerProfileFixture("Anthropic", "anthropic_compatible", "api.anthropic.com", "/v1")
	updated.CurrentAuthMode = "oauth_derived"
	updated.SupportedAuthModes = []string{"direct_credential", "oauth_derived"}
	second, err := state.upsertProfile(updated)
	if err != nil {
		t.Fatalf("upsertProfile(second) error: %v", err)
	}
	if first.ProviderProfileID != second.ProviderProfileID {
		t.Fatalf("provider_profile_id changed across auth-mode update: before=%q after=%q", first.ProviderProfileID, second.ProviderProfileID)
	}
}

func TestProviderSubstrateCredentialRotationAndValidationRetriesKeepProfileIdentity(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	state := newProviderSubstrateState(func() time.Time { return now })
	profile, err := state.upsertProfile(providerProfileFixture("OpenAI", "openai_compatible", "api.openai.com", "/v1"))
	if err != nil {
		t.Fatalf("upsertProfile error: %v", err)
	}
	profileID := profile.ProviderProfileID
	now = now.Add(2 * time.Minute)
	profile = mustSetProviderAuthMaterial(t, state, profileID, "secrets/model-providers/openai/key-v1")
	if profile.ProviderProfileID != profileID {
		t.Fatalf("profile id changed after first credential set: before=%q after=%q", profileID, profile.ProviderProfileID)
	}
	now = now.Add(2 * time.Minute)
	profile = mustSetProviderAuthMaterial(t, state, profileID, "secrets/model-providers/openai/key-v2")
	if profile.ProviderProfileID != profileID {
		t.Fatalf("profile id changed after credential rotation: before=%q after=%q", profileID, profile.ProviderProfileID)
	}
	now = now.Add(2 * time.Minute)
	profile = mustRecordProviderValidation(t, state, profileID, ProviderReadinessPosture{ConfigurationState: "configured", CredentialState: "present", ConnectivityState: "degraded", CompatibilityState: "incompatible", EffectiveReadiness: "not_ready", ReasonCodes: []string{"validation_retry_required"}})
	now = now.Add(2 * time.Minute)
	profile = mustRecordProviderValidation(t, state, profileID, ProviderReadinessPosture{ConfigurationState: "configured", CredentialState: "present", ConnectivityState: "reachable", CompatibilityState: "compatible", EffectiveReadiness: "ready"})
	if profile.ProviderProfileID != profileID {
		t.Fatalf("profile id changed after validation retries: before=%q after=%q", profileID, profile.ProviderProfileID)
	}
	if got := profile.Lifecycle.ValidationAttemptCount; got != 2 {
		t.Fatalf("validation_attempt_count = %d, want 2", got)
	}
	if got := profile.ReadinessPosture.EffectiveReadiness; got != "ready" {
		t.Fatalf("effective_readiness = %q, want ready", got)
	}
}

func mustSetProviderAuthMaterial(t *testing.T, state *providerSubstrateState, profileID, secretRef string) ProviderProfile {
	t.Helper()
	profile, err := state.setAuthMaterial(profileID, ProviderAuthMaterial{MaterialKind: "direct_credential", MaterialState: "present", SecretRef: secretRef})
	if err != nil {
		t.Fatalf("setAuthMaterial(%s) error: %v", secretRef, err)
	}
	return profile
}

func mustRecordProviderValidation(t *testing.T, state *providerSubstrateState, profileID string, posture ProviderReadinessPosture) ProviderProfile {
	t.Helper()
	profile, err := state.recordValidation(profileID, posture)
	if err != nil {
		t.Fatalf("recordValidation error: %v", err)
	}
	return profile
}

func providerProfileFixture(label, family, host, pathPrefix string) ProviderProfile {
	adapterKind := providerAdapterKindOpenAIChatCompletionsV0
	if strings.TrimSpace(strings.ToLower(family)) == providerFamilyAnthropicCompatible {
		adapterKind = providerAdapterKindAnthropicMessagesV0
	}
	return ProviderProfile{
		DisplayLabel:   label,
		ProviderFamily: family,
		AdapterKind:    adapterKind,
		DestinationIdentity: policyengine.DestinationDescriptor{
			SchemaID:               "runecode.protocol.v0.DestinationDescriptor",
			SchemaVersion:          "0.1.0",
			DescriptorKind:         "model_endpoint",
			CanonicalHost:          host,
			CanonicalPathPrefix:    pathPrefix,
			ProviderOrNamespace:    family,
			TLSRequired:            true,
			PrivateRangeBlocking:   "enforced",
			DNSRebindingProtection: "enforced",
		},
		SupportedAuthModes:   []string{"direct_credential"},
		CurrentAuthMode:      "direct_credential",
		AllowlistedModelIDs:  []string{"gpt-4.1-mini"},
		ModelCatalogPosture:  ProviderModelCatalogPosture{SelectionAuthority: "manual_allowlist_canonical", DiscoveryPosture: "advisory", CompatibilityProbePosture: "advisory"},
		CompatibilityPosture: "unverified",
		QuotaProfileKind:     "hybrid",
		AuthMaterial:         ProviderAuthMaterial{MaterialKind: "direct_credential", MaterialState: "missing"},
		ReadinessPosture:     ProviderReadinessPosture{ConfigurationState: "configured", CredentialState: "missing", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"},
	}
}
