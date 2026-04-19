package brokerapi

import (
	"fmt"
	"sort"
	"strings"
)

func normalizeProviderProfileIdentity(profile ProviderProfile) (ProviderProfile, error) {
	profile.DisplayLabel = strings.TrimSpace(profile.DisplayLabel)
	profile.ProviderFamily = strings.TrimSpace(strings.ToLower(profile.ProviderFamily))
	profile.AdapterKind = strings.TrimSpace(strings.ToLower(profile.AdapterKind))
	profile.DestinationRef = strings.TrimSpace(profile.DestinationRef)
	if profile.ProviderFamily == "" {
		return ProviderProfile{}, fmt.Errorf("provider_family is required")
	}
	if profile.AdapterKind == "" {
		profile.AdapterKind = defaultAdapterKindForProviderFamily(profile.ProviderFamily)
	}
	if profile.AdapterKind == "" {
		return ProviderProfile{}, fmt.Errorf("adapter_kind is required")
	}
	if err := validateProviderAdapterFamily(profile.ProviderFamily, profile.AdapterKind); err != nil {
		return ProviderProfile{}, err
	}
	if profile.DisplayLabel == "" {
		profile.DisplayLabel = profile.ProviderFamily
	}
	if !isHardenedModelDestination(profile.DestinationIdentity) {
		return ProviderProfile{}, fmt.Errorf("destination_identity must be hardened model_endpoint descriptor")
	}
	if profile.DestinationRef == "" {
		profile.DestinationRef = destinationRefFromDescriptor(profile.DestinationIdentity)
	}
	if profile.ProviderProfileID == "" {
		profile.ProviderProfileID = stableProviderProfileID(profile.ProviderFamily, profile.DestinationRef)
	}
	if profile.CurrentAuthMode == "" {
		profile.CurrentAuthMode = "direct_credential"
	}
	return profile, nil
}

func applyProviderProfileDefaults(profile ProviderProfile) ProviderProfile {
	if strings.TrimSpace(profile.CompatibilityPosture) == "" {
		profile.CompatibilityPosture = "unverified"
	}
	if strings.TrimSpace(profile.QuotaProfileKind) == "" {
		profile.QuotaProfileKind = "hybrid"
	}
	profile.RequestBindingKind = "canonical_llm_request_digest"
	profile.SurfaceChannel = "broker_local_api"
	return profile
}

func normalizeProviderAuthMaterial(material ProviderAuthMaterial) ProviderAuthMaterial {
	material.SchemaID = "runecode.protocol.v0.ProviderAuthMaterial"
	material.SchemaVersion = "0.1.0"
	material.MaterialKind = strings.TrimSpace(material.MaterialKind)
	material.MaterialState = strings.TrimSpace(material.MaterialState)
	material.SecretRef = strings.TrimSpace(material.SecretRef)
	material.LeasePolicyRef = strings.TrimSpace(material.LeasePolicyRef)
	material.SessionBindingID = strings.TrimSpace(material.SessionBindingID)
	material.LastRotatedAt = strings.TrimSpace(material.LastRotatedAt)
	if material.MaterialKind == "" {
		material.MaterialKind = "direct_credential"
	}
	if material.MaterialState == "" {
		material.MaterialState = "missing"
	}
	return material
}

func normalizeProviderReadinessPosture(posture ProviderReadinessPosture) ProviderReadinessPosture {
	posture.SchemaID = "runecode.protocol.v0.ProviderReadinessPosture"
	posture.SchemaVersion = "0.1.0"
	if strings.TrimSpace(posture.ConfigurationState) == "" {
		posture.ConfigurationState = "not_configured"
	}
	if strings.TrimSpace(posture.CredentialState) == "" {
		posture.CredentialState = "missing"
	}
	if strings.TrimSpace(posture.ConnectivityState) == "" {
		posture.ConnectivityState = "unknown"
	}
	if strings.TrimSpace(posture.CompatibilityState) == "" {
		posture.CompatibilityState = "unknown"
	}
	if strings.TrimSpace(posture.EffectiveReadiness) == "" {
		posture.EffectiveReadiness = "not_ready"
	}
	posture.ReasonCodes = normalizedStringSet(posture.ReasonCodes)
	posture.LastValidationAt = strings.TrimSpace(posture.LastValidationAt)
	posture.ValidationAttemptID = strings.TrimSpace(posture.ValidationAttemptID)
	return posture
}

func normalizeProviderModelCatalogPosture(posture ProviderModelCatalogPosture) ProviderModelCatalogPosture {
	posture.SchemaID = "runecode.protocol.v0.ProviderModelCatalogPosture"
	posture.SchemaVersion = "0.1.0"
	if strings.TrimSpace(posture.SelectionAuthority) == "" {
		posture.SelectionAuthority = "manual_allowlist_canonical"
	}
	if strings.TrimSpace(posture.DiscoveryPosture) == "" {
		posture.DiscoveryPosture = "advisory"
	}
	if strings.TrimSpace(posture.CompatibilityProbePosture) == "" {
		posture.CompatibilityProbePosture = "advisory"
	}
	posture.DiscoveredModelIDs = normalizedStringSet(posture.DiscoveredModelIDs)
	posture.ProbeCompatibleModelIDs = normalizedStringSet(posture.ProbeCompatibleModelIDs)
	posture.LastDiscoveryAt = strings.TrimSpace(posture.LastDiscoveryAt)
	posture.LastProbeAt = strings.TrimSpace(posture.LastProbeAt)
	return posture
}

func normalizeAuthModes(supported []string, current string) []string {
	modes := append([]string{}, supported...)
	modes = append(modes, current)
	normalized := normalizedStringSet(modes)
	if len(normalized) == 0 {
		return []string{"direct_credential"}
	}
	return normalized
}

func normalizedStringSet(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func maxInt64(v, min int64) int64 {
	if v < min {
		return min
	}
	return v
}

func defaultAdapterKindForProviderFamily(family string) string {
	switch strings.TrimSpace(strings.ToLower(family)) {
	case providerFamilyOpenAICompatible:
		return providerAdapterKindOpenAIChatCompletionsV0
	case providerFamilyAnthropicCompatible:
		return providerAdapterKindAnthropicMessagesV0
	default:
		return ""
	}
}

func validateProviderAdapterFamily(family, adapterKind string) error {
	profile := ProviderProfile{ProviderFamily: family, AdapterKind: adapterKind}
	_, err := adapterForProviderProfile(profile)
	if err != nil {
		return err
	}
	return nil
}
