package artifacts

import (
	"fmt"
	"sort"
	"strings"
)

func (s *Store) UpsertProviderProfile(profile ProviderProfileDurableState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	profileID := strings.TrimSpace(profile.ProviderProfileID)
	if profileID == "" {
		return fmt.Errorf("provider_profile_id is required")
	}
	profile.ProviderProfileID = profileID
	s.state.ProviderProfiles[profileID] = cloneProviderProfileDurableState(profile)
	return s.saveStateLocked()
}

func (s *Store) ProviderProfiles() map[string]ProviderProfileDurableState {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]ProviderProfileDurableState, len(s.state.ProviderProfiles))
	for profileID, profile := range s.state.ProviderProfiles {
		out[profileID] = cloneProviderProfileDurableState(profile)
	}
	return out
}

func (s *Store) UpsertProviderSetupSession(session ProviderSetupSessionDurableState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	setupSessionID := strings.TrimSpace(session.SetupSessionID)
	if setupSessionID == "" {
		return fmt.Errorf("setup_session_id is required")
	}
	session.SetupSessionID = setupSessionID
	s.state.ProviderSetupSessions[setupSessionID] = cloneProviderSetupSessionDurableState(session)
	return s.saveStateLocked()
}

func (s *Store) ProviderSetupSessions() map[string]ProviderSetupSessionDurableState {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]ProviderSetupSessionDurableState, len(s.state.ProviderSetupSessions))
	for setupSessionID, session := range s.state.ProviderSetupSessions {
		out[setupSessionID] = cloneProviderSetupSessionDurableState(session)
	}
	return out
}

func cloneProviderProfileDurableState(profile ProviderProfileDurableState) ProviderProfileDurableState {
	profile.SupportedAuthModes = append([]string{}, profile.SupportedAuthModes...)
	profile.AllowlistedModelIDs = append([]string{}, profile.AllowlistedModelIDs...)
	profile.ModelCatalogPosture.DiscoveredModelIDs = append([]string{}, profile.ModelCatalogPosture.DiscoveredModelIDs...)
	profile.ModelCatalogPosture.ProbeCompatibleModelIDs = append([]string{}, profile.ModelCatalogPosture.ProbeCompatibleModelIDs...)
	profile.ReadinessPosture.ReasonCodes = append([]string{}, profile.ReadinessPosture.ReasonCodes...)
	return profile
}

func cloneProviderSetupSessionDurableState(session ProviderSetupSessionDurableState) ProviderSetupSessionDurableState {
	session.SupportedAuthModes = append([]string{}, session.SupportedAuthModes...)
	return session
}

func sortedProviderProfiles(profiles map[string]ProviderProfileDurableState) []ProviderProfileDurableState {
	if len(profiles) == 0 {
		return []ProviderProfileDurableState{}
	}
	keys := make([]string, 0, len(profiles))
	for key := range profiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ProviderProfileDurableState, 0, len(keys))
	for _, key := range keys {
		out = append(out, cloneProviderProfileDurableState(profiles[key]))
	}
	return out
}

func sortedProviderSetupSessions(sessions map[string]ProviderSetupSessionDurableState) []ProviderSetupSessionDurableState {
	if len(sessions) == 0 {
		return []ProviderSetupSessionDurableState{}
	}
	keys := make([]string, 0, len(sessions))
	for key := range sessions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ProviderSetupSessionDurableState, 0, len(keys))
	for _, key := range keys {
		out = append(out, cloneProviderSetupSessionDurableState(sessions[key]))
	}
	return out
}
