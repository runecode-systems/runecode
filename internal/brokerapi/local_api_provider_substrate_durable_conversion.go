package brokerapi

import (
	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func destinationIdentityToDurable(identity policyengine.DestinationDescriptor) artifacts.ProviderDestinationIdentityDurableState {
	return artifacts.ProviderDestinationIdentityDurableState{
		SchemaID:               identity.SchemaID,
		SchemaVersion:          identity.SchemaVersion,
		DescriptorKind:         identity.DescriptorKind,
		CanonicalHost:          identity.CanonicalHost,
		CanonicalPathPrefix:    identity.CanonicalPathPrefix,
		ProviderOrNamespace:    identity.ProviderOrNamespace,
		TLSRequired:            identity.TLSRequired,
		PrivateRangeBlocking:   identity.PrivateRangeBlocking,
		DNSRebindingProtection: identity.DNSRebindingProtection,
	}
}

func destinationIdentityFromDurable(identity artifacts.ProviderDestinationIdentityDurableState) policyengine.DestinationDescriptor {
	return policyengine.DestinationDescriptor{
		SchemaID:               identity.SchemaID,
		SchemaVersion:          identity.SchemaVersion,
		DescriptorKind:         identity.DescriptorKind,
		CanonicalHost:          identity.CanonicalHost,
		CanonicalPathPrefix:    identity.CanonicalPathPrefix,
		ProviderOrNamespace:    identity.ProviderOrNamespace,
		TLSRequired:            identity.TLSRequired,
		PrivateRangeBlocking:   identity.PrivateRangeBlocking,
		DNSRebindingProtection: identity.DNSRebindingProtection,
	}
}

func modelCatalogPostureToDurable(posture ProviderModelCatalogPosture) artifacts.ProviderModelCatalogPostureDurableState {
	return artifacts.ProviderModelCatalogPostureDurableState{
		SchemaID:                  posture.SchemaID,
		SchemaVersion:             posture.SchemaVersion,
		SelectionAuthority:        posture.SelectionAuthority,
		DiscoveryPosture:          posture.DiscoveryPosture,
		CompatibilityProbePosture: posture.CompatibilityProbePosture,
		DiscoveredModelIDs:        append([]string{}, posture.DiscoveredModelIDs...),
		ProbeCompatibleModelIDs:   append([]string{}, posture.ProbeCompatibleModelIDs...),
		LastDiscoveryAt:           posture.LastDiscoveryAt,
		LastProbeAt:               posture.LastProbeAt,
	}
}

func modelCatalogPostureFromDurable(posture artifacts.ProviderModelCatalogPostureDurableState) ProviderModelCatalogPosture {
	return ProviderModelCatalogPosture{
		SchemaID:                  posture.SchemaID,
		SchemaVersion:             posture.SchemaVersion,
		SelectionAuthority:        posture.SelectionAuthority,
		DiscoveryPosture:          posture.DiscoveryPosture,
		CompatibilityProbePosture: posture.CompatibilityProbePosture,
		DiscoveredModelIDs:        append([]string{}, posture.DiscoveredModelIDs...),
		ProbeCompatibleModelIDs:   append([]string{}, posture.ProbeCompatibleModelIDs...),
		LastDiscoveryAt:           posture.LastDiscoveryAt,
		LastProbeAt:               posture.LastProbeAt,
	}
}

func authMaterialToDurable(material ProviderAuthMaterial) artifacts.ProviderAuthMaterialDurableState {
	return artifacts.ProviderAuthMaterialDurableState{
		SchemaID:         material.SchemaID,
		SchemaVersion:    material.SchemaVersion,
		MaterialKind:     material.MaterialKind,
		MaterialState:    material.MaterialState,
		SecretRef:        material.SecretRef,
		LeasePolicyRef:   material.LeasePolicyRef,
		SessionBindingID: material.SessionBindingID,
		LastRotatedAt:    material.LastRotatedAt,
	}
}

func authMaterialFromDurable(material artifacts.ProviderAuthMaterialDurableState) ProviderAuthMaterial {
	return ProviderAuthMaterial{
		SchemaID:         material.SchemaID,
		SchemaVersion:    material.SchemaVersion,
		MaterialKind:     material.MaterialKind,
		MaterialState:    material.MaterialState,
		SecretRef:        material.SecretRef,
		LeasePolicyRef:   material.LeasePolicyRef,
		SessionBindingID: material.SessionBindingID,
		LastRotatedAt:    material.LastRotatedAt,
	}
}

func readinessPostureToDurable(posture ProviderReadinessPosture) artifacts.ProviderReadinessPostureDurableState {
	return artifacts.ProviderReadinessPostureDurableState{
		SchemaID:            posture.SchemaID,
		SchemaVersion:       posture.SchemaVersion,
		ConfigurationState:  posture.ConfigurationState,
		CredentialState:     posture.CredentialState,
		ConnectivityState:   posture.ConnectivityState,
		CompatibilityState:  posture.CompatibilityState,
		EffectiveReadiness:  posture.EffectiveReadiness,
		ReasonCodes:         append([]string{}, posture.ReasonCodes...),
		LastValidationAt:    posture.LastValidationAt,
		ValidationAttemptID: posture.ValidationAttemptID,
	}
}

func readinessPostureFromDurable(posture artifacts.ProviderReadinessPostureDurableState) ProviderReadinessPosture {
	return ProviderReadinessPosture{
		SchemaID:            posture.SchemaID,
		SchemaVersion:       posture.SchemaVersion,
		ConfigurationState:  posture.ConfigurationState,
		CredentialState:     posture.CredentialState,
		ConnectivityState:   posture.ConnectivityState,
		CompatibilityState:  posture.CompatibilityState,
		EffectiveReadiness:  posture.EffectiveReadiness,
		ReasonCodes:         append([]string{}, posture.ReasonCodes...),
		LastValidationAt:    posture.LastValidationAt,
		ValidationAttemptID: posture.ValidationAttemptID,
	}
}

func lifecycleMetadataToDurable(lifecycle ProviderLifecycleMetadata) artifacts.ProviderLifecycleMetadataDurableState {
	return artifacts.ProviderLifecycleMetadataDurableState{
		CreatedAt:               lifecycle.CreatedAt,
		UpdatedAt:               lifecycle.UpdatedAt,
		LastValidationAt:        lifecycle.LastValidationAt,
		ValidationAttemptCount:  lifecycle.ValidationAttemptCount,
		LastValidationSucceeded: lifecycle.LastValidationSucceeded,
	}
}

func lifecycleMetadataFromDurable(lifecycle artifacts.ProviderLifecycleMetadataDurableState) ProviderLifecycleMetadata {
	return ProviderLifecycleMetadata{
		CreatedAt:               lifecycle.CreatedAt,
		UpdatedAt:               lifecycle.UpdatedAt,
		LastValidationAt:        lifecycle.LastValidationAt,
		ValidationAttemptCount:  lifecycle.ValidationAttemptCount,
		LastValidationSucceeded: lifecycle.LastValidationSucceeded,
	}
}
