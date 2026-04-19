package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func providerProfileFromSetupBegin(req ProviderSetupSessionBeginRequest) ProviderProfile {
	family := strings.TrimSpace(strings.ToLower(req.ProviderFamily))
	adapterKind := strings.TrimSpace(strings.ToLower(req.AdapterKind))
	if adapterKind == "" {
		adapterKind = defaultAdapterKindForProviderFamily(family)
	}
	host := strings.TrimSpace(strings.ToLower(req.CanonicalHost))
	pathPrefix := strings.TrimSpace(req.CanonicalPathPrefix)
	if pathPrefix == "" {
		pathPrefix = "/v1"
	}
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	posture := ProviderReadinessPosture{ConfigurationState: "configured", CredentialState: "missing", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"}
	posture.ReasonCodes = providerReadinessReasonCodes(posture)
	return ProviderProfile{DisplayLabel: strings.TrimSpace(req.DisplayLabel), ProviderFamily: family, AdapterKind: adapterKind, DestinationIdentity: policyengine.DestinationDescriptor{SchemaID: "runecode.protocol.v0.DestinationDescriptor", SchemaVersion: "0.1.0", DescriptorKind: "model_endpoint", CanonicalHost: host, CanonicalPathPrefix: pathPrefix, ProviderOrNamespace: family, TLSRequired: true, PrivateRangeBlocking: "enforced", DNSRebindingProtection: "enforced"}, SupportedAuthModes: []string{"direct_credential"}, CurrentAuthMode: "direct_credential", AllowlistedModelIDs: req.AllowlistedModelIDs, ModelCatalogPosture: ProviderModelCatalogPosture{SelectionAuthority: "manual_allowlist_canonical", DiscoveryPosture: "advisory", CompatibilityProbePosture: "advisory"}, CompatibilityPosture: "unverified", AuthMaterial: ProviderAuthMaterial{MaterialKind: "direct_credential", MaterialState: "missing"}, ReadinessPosture: posture}
}

func (s *Service) providerProfileByID(profileID string) (ProviderProfile, bool) {
	profiles := s.providerSubstrate.snapshotProfiles()
	for _, profile := range profiles {
		if profile.ProviderProfileID == profileID {
			return profile, true
		}
	}
	return ProviderProfile{}, false
}

func providerReadinessReasonCodes(posture ProviderReadinessPosture) []string {
	reasons := []string{}
	reasons = append(reasons, providerConfigurationReasons(posture)...)
	reasons = append(reasons, providerCredentialReasons(posture)...)
	reasons = append(reasons, providerConnectivityReasons(posture)...)
	reasons = append(reasons, providerCompatibilityReasons(posture)...)
	return normalizedStringSet(reasons)
}

func providerConfigurationReasons(posture ProviderReadinessPosture) []string {
	if strings.TrimSpace(posture.ConfigurationState) == "configured" {
		return nil
	}
	return []string{"provider_configuration_required"}
}

func providerCredentialReasons(posture ProviderReadinessPosture) []string {
	credentialState := strings.TrimSpace(posture.CredentialState)
	if credentialState == "present" {
		return nil
	}
	reasons := []string{"secret_ingress_required"}
	if credentialState == "expired" {
		reasons = append(reasons, "credential_expired")
	}
	if credentialState == "invalid" {
		reasons = append(reasons, "credential_invalid")
	}
	return reasons
}

func providerConnectivityReasons(posture ProviderReadinessPosture) []string {
	connectivityState := strings.TrimSpace(posture.ConnectivityState)
	if connectivityState == "unknown" {
		return []string{"connectivity_validation_pending"}
	}
	if connectivityState == "degraded" || connectivityState == "unreachable" {
		return []string{"connectivity_unhealthy"}
	}
	return nil
}

func providerCompatibilityReasons(posture ProviderReadinessPosture) []string {
	compatibilityState := strings.TrimSpace(posture.CompatibilityState)
	if compatibilityState == "unknown" {
		return []string{"compatibility_probe_pending"}
	}
	if compatibilityState == "incompatible" {
		return []string{"adapter_compatibility_failed"}
	}
	return nil
}

func destinationRefFromHostAndPath(host, pathPrefix string) string {
	resolvedHost := strings.TrimSpace(strings.ToLower(host))
	resolvedPath := strings.TrimSpace(pathPrefix)
	if resolvedPath == "" {
		resolvedPath = "/v1"
	}
	if !strings.HasPrefix(resolvedPath, "/") {
		resolvedPath = "/" + resolvedPath
	}
	return resolvedHost + resolvedPath
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func withUpdatedReadinessPosture(profile ProviderProfile, posture ProviderReadinessPosture) ProviderProfile {
	profile.ReadinessPosture = posture
	return profile
}

func buildProviderValidationPosture(previous ProviderReadinessPosture, req ProviderValidationCommitRequest) ProviderReadinessPosture {
	next := previous
	if strings.TrimSpace(req.ConfigurationState) != "" {
		next.ConfigurationState = req.ConfigurationState
	}
	if strings.TrimSpace(req.CredentialState) != "" {
		next.CredentialState = req.CredentialState
	}
	next.ConnectivityState = req.ConnectivityState
	next.CompatibilityState = req.CompatibilityState
	reasons := append([]string{}, providerReadinessReasonCodes(next)...)
	reasons = append(reasons, normalizedStringSet(req.ReasonCodes)...)
	next.ReasonCodes = normalizedStringSet(reasons)
	next.EffectiveReadiness = providerEffectiveReadiness(next)
	return next
}

func providerEffectiveReadiness(posture ProviderReadinessPosture) string {
	if strings.TrimSpace(posture.ConfigurationState) == "configured" && strings.TrimSpace(posture.CredentialState) == "present" && strings.TrimSpace(posture.ConnectivityState) == "reachable" && strings.TrimSpace(posture.CompatibilityState) == "compatible" {
		return "ready"
	}
	return "not_ready"
}

func compatibilityPostureFromReadiness(compatibilityState string) string {
	switch strings.TrimSpace(compatibilityState) {
	case "compatible":
		return "compatible"
	case "incompatible":
		return "incompatible"
	default:
		return "unverified"
	}
}
