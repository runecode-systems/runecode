package brokerapi

import (
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) configureProviderDurability() {
	if s == nil || s.store == nil {
		return
	}
	if s.providerSubstrate != nil {
		s.providerSubstrate.setPersistFunc(func(profile ProviderProfile) error {
			return s.store.UpsertProviderProfile(providerProfileToDurable(profile))
		})
	}
	if s.providerSetup != nil {
		s.providerSetup.setPersistFunc(func(session ProviderSetupSession) error {
			return s.store.UpsertProviderSetupSession(providerSetupSessionToDurable(session))
		})
	}
}

func (s *Service) reloadProviderDurableState() error {
	if s == nil || s.store == nil || s.providerSubstrate == nil || s.providerSetup == nil {
		return nil
	}
	profilesByID := s.store.ProviderProfiles()
	profileIDs := make([]string, 0, len(profilesByID))
	for profileID := range profilesByID {
		profileIDs = append(profileIDs, profileID)
	}
	sort.Strings(profileIDs)
	profiles := make([]ProviderProfile, 0, len(profileIDs))
	for _, profileID := range profileIDs {
		profiles = append(profiles, providerProfileFromDurable(profilesByID[profileID]))
	}
	if err := s.providerSubstrate.restoreProfiles(profiles); err != nil {
		return err
	}
	sessionsByID := s.store.ProviderSetupSessions()
	sessionIDs := make([]string, 0, len(sessionsByID))
	for setupSessionID := range sessionsByID {
		sessionIDs = append(sessionIDs, setupSessionID)
	}
	sort.Strings(sessionIDs)
	sessions := make([]ProviderSetupSession, 0, len(sessionIDs))
	for _, setupSessionID := range sessionIDs {
		sessions = append(sessions, providerSetupSessionFromDurable(sessionsByID[setupSessionID]))
	}
	s.providerSetup.restoreSessions(sessions)
	return s.reconcileProviderSecretReferences()
}

func (s *Service) reconcileProviderSecretReferences() error {
	if s == nil || s.providerSubstrate == nil || s.secretsSvc == nil {
		return nil
	}
	profiles := s.providerSubstrate.snapshotProfiles()
	for _, profile := range profiles {
		reconciled, changed := s.reconcileProviderSecretReference(profile)
		if changed {
			if _, _, err := s.providerSubstrate.upsertProfile(reconciled); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) reconcileProviderSecretReference(profile ProviderProfile) (ProviderProfile, bool) {
	if strings.TrimSpace(profile.AuthMaterial.MaterialKind) != "direct_credential" {
		return profile, false
	}
	if _, hasSecret := s.secretsSvc.LookupSecretMetadata(profile.AuthMaterial.SecretRef); hasSecret {
		return profile, false
	}
	reconciled := profile
	changed := false
	changed = assignIfDifferent(&reconciled.AuthMaterial.MaterialState, "missing") || changed
	readiness, readinessChanged := reconcileMissingSecretReadiness(reconciled.ReadinessPosture)
	reconciled.ReadinessPosture = readiness
	return reconciled, changed || readinessChanged
}

func reconcileMissingSecretReadiness(posture ProviderReadinessPosture) (ProviderReadinessPosture, bool) {
	reconciled := posture
	changed := false
	changed = assignIfDifferent(&reconciled.CredentialState, "missing") || changed
	changed = assignIfDifferent(&reconciled.EffectiveReadiness, "not_ready") || changed
	nextReasonCodes := providerReadinessReasonCodes(reconciled)
	if strings.Join(nextReasonCodes, ",") == strings.Join(reconciled.ReasonCodes, ",") {
		return reconciled, changed
	}
	reconciled.ReasonCodes = nextReasonCodes
	return reconciled, true
}

func assignIfDifferent(target *string, next string) bool {
	if *target == next {
		return false
	}
	*target = next
	return true
}

func providerProfileToDurable(profile ProviderProfile) artifacts.ProviderProfileDurableState {
	return artifacts.ProviderProfileDurableState{
		SchemaID:             profile.SchemaID,
		SchemaVersion:        profile.SchemaVersion,
		ProviderProfileID:    profile.ProviderProfileID,
		DisplayLabel:         profile.DisplayLabel,
		ProviderFamily:       profile.ProviderFamily,
		AdapterKind:          profile.AdapterKind,
		DestinationIdentity:  destinationIdentityToDurable(profile.DestinationIdentity),
		DestinationRef:       profile.DestinationRef,
		SupportedAuthModes:   append([]string{}, profile.SupportedAuthModes...),
		CurrentAuthMode:      profile.CurrentAuthMode,
		AllowlistedModelIDs:  append([]string{}, profile.AllowlistedModelIDs...),
		ModelCatalogPosture:  modelCatalogPostureToDurable(profile.ModelCatalogPosture),
		CompatibilityPosture: profile.CompatibilityPosture,
		QuotaProfileKind:     profile.QuotaProfileKind,
		RequestBindingKind:   profile.RequestBindingKind,
		SurfaceChannel:       profile.SurfaceChannel,
		AuthMaterial:         authMaterialToDurable(profile.AuthMaterial),
		ReadinessPosture:     readinessPostureToDurable(profile.ReadinessPosture),
		Lifecycle:            lifecycleMetadataToDurable(profile.Lifecycle),
	}
}

func providerProfileFromDurable(profile artifacts.ProviderProfileDurableState) ProviderProfile {
	return ProviderProfile{
		SchemaID:             profile.SchemaID,
		SchemaVersion:        profile.SchemaVersion,
		ProviderProfileID:    profile.ProviderProfileID,
		DisplayLabel:         profile.DisplayLabel,
		ProviderFamily:       profile.ProviderFamily,
		AdapterKind:          profile.AdapterKind,
		DestinationIdentity:  destinationIdentityFromDurable(profile.DestinationIdentity),
		DestinationRef:       profile.DestinationRef,
		SupportedAuthModes:   append([]string{}, profile.SupportedAuthModes...),
		CurrentAuthMode:      profile.CurrentAuthMode,
		AllowlistedModelIDs:  append([]string{}, profile.AllowlistedModelIDs...),
		ModelCatalogPosture:  modelCatalogPostureFromDurable(profile.ModelCatalogPosture),
		CompatibilityPosture: profile.CompatibilityPosture,
		QuotaProfileKind:     profile.QuotaProfileKind,
		RequestBindingKind:   profile.RequestBindingKind,
		SurfaceChannel:       profile.SurfaceChannel,
		AuthMaterial:         authMaterialFromDurable(profile.AuthMaterial),
		ReadinessPosture:     readinessPostureFromDurable(profile.ReadinessPosture),
		Lifecycle:            lifecycleMetadataFromDurable(profile.Lifecycle),
	}
}
