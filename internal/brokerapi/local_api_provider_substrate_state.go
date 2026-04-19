package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type providerSubstrateState struct {
	mu             sync.RWMutex
	profiles       map[string]ProviderProfile
	nowFn          func() time.Time
	persistProfile func(ProviderProfile) error
}

func newProviderSubstrateState(nowFn func() time.Time) *providerSubstrateState {
	if nowFn == nil {
		nowFn = time.Now
	}
	return &providerSubstrateState{profiles: map[string]ProviderProfile{}, nowFn: nowFn}
}

func (s *providerSubstrateState) setNowFunc(nowFn func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if nowFn == nil {
		s.nowFn = time.Now
		return
	}
	s.nowFn = nowFn
}

func (s *providerSubstrateState) setPersistFunc(persistFn func(ProviderProfile) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persistProfile = persistFn
}

func (s *providerSubstrateState) restoreProfiles(profiles []ProviderProfile) error {
	if s == nil {
		return fmt.Errorf("provider substrate state unavailable")
	}
	next := make(map[string]ProviderProfile, len(profiles))
	for _, profile := range profiles {
		normalized, err := normalizeProviderProfile(profile)
		if err != nil {
			return err
		}
		next[normalized.ProviderProfileID] = normalized
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profiles = next
	return nil
}

func (s *providerSubstrateState) upsertProfile(profile ProviderProfile) (ProviderProfile, bool, error) {
	if s == nil {
		return ProviderProfile{}, false, fmt.Errorf("provider substrate state unavailable")
	}
	normalized, err := normalizeProviderProfile(profile)
	if err != nil {
		return ProviderProfile{}, false, err
	}
	now := s.nowFn().UTC().Format(time.RFC3339)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, existed := s.profiles[normalized.ProviderProfileID]
	normalized = mergedProviderProfileForUpsert(normalized, s.profiles[normalized.ProviderProfileID], now)
	if s.persistProfile != nil {
		if err := s.persistProfile(normalized); err != nil {
			return ProviderProfile{}, false, err
		}
	}
	s.profiles[normalized.ProviderProfileID] = normalized
	return normalized, !existed, nil
}

func mergedProviderProfileForUpsert(normalized, existing ProviderProfile, now string) ProviderProfile {
	if strings.TrimSpace(existing.ProviderProfileID) == "" {
		normalized.Lifecycle.CreatedAt = now
		normalized.Lifecycle.UpdatedAt = now
		return normalized
	}
	normalized.Lifecycle.CreatedAt = existing.Lifecycle.CreatedAt
	normalized.Lifecycle.ValidationAttemptCount = existing.Lifecycle.ValidationAttemptCount
	normalized.Lifecycle.LastValidationAt = preservedProviderValue(normalized.Lifecycle.LastValidationAt, existing.Lifecycle.LastValidationAt)
	normalized.Lifecycle.LastValidationSucceeded = existing.Lifecycle.LastValidationSucceeded
	normalized.ReadinessPosture.LastValidationAt = preservedProviderValue(normalized.ReadinessPosture.LastValidationAt, existing.ReadinessPosture.LastValidationAt)
	normalized.ReadinessPosture.ValidationAttemptID = preservedProviderValue(normalized.ReadinessPosture.ValidationAttemptID, existing.ReadinessPosture.ValidationAttemptID)
	normalized.Lifecycle.UpdatedAt = now
	return normalized
}

func preservedProviderValue(current, existing string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	return existing
}

func (s *providerSubstrateState) setAuthMaterial(profileID string, material ProviderAuthMaterial) (ProviderProfile, error) {
	if s == nil {
		return ProviderProfile{}, fmt.Errorf("provider substrate state unavailable")
	}
	id := strings.TrimSpace(profileID)
	if id == "" {
		return ProviderProfile{}, fmt.Errorf("provider_profile_id is required")
	}
	normalizedMaterial := normalizeProviderAuthMaterial(material)
	now := s.nowFn().UTC().Format(time.RFC3339)
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[id]
	if !ok {
		return ProviderProfile{}, fmt.Errorf("provider profile not found")
	}
	profile.AuthMaterial = normalizedMaterial
	profile.Lifecycle.UpdatedAt = now
	if s.persistProfile != nil {
		if err := s.persistProfile(profile); err != nil {
			return ProviderProfile{}, err
		}
	}
	s.profiles[id] = profile
	return profile, nil
}

func (s *providerSubstrateState) recordValidation(profileID string, posture ProviderReadinessPosture) (ProviderProfile, error) {
	if s == nil {
		return ProviderProfile{}, fmt.Errorf("provider substrate state unavailable")
	}
	id := strings.TrimSpace(profileID)
	if id == "" {
		return ProviderProfile{}, fmt.Errorf("provider_profile_id is required")
	}
	normalized := normalizeProviderReadinessPosture(posture)
	now := s.nowFn().UTC().Format(time.RFC3339)
	normalized.LastValidationAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[id]
	if !ok {
		return ProviderProfile{}, fmt.Errorf("provider profile not found")
	}
	profile.ReadinessPosture = normalized
	profile.Lifecycle.ValidationAttemptCount++
	profile.Lifecycle.LastValidationAt = now
	profile.Lifecycle.LastValidationSucceeded = normalized.EffectiveReadiness == "ready"
	profile.Lifecycle.UpdatedAt = now
	if s.persistProfile != nil {
		if err := s.persistProfile(profile); err != nil {
			return ProviderProfile{}, err
		}
	}
	s.profiles[id] = profile
	return profile, nil
}

func (s *providerSubstrateState) snapshotProfiles() []ProviderProfile {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ProviderProfile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		out = append(out, profile)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ProviderProfileID < out[j].ProviderProfileID
	})
	return out
}

func normalizeProviderProfile(profile ProviderProfile) (ProviderProfile, error) {
	profile.SchemaID = "runecode.protocol.v0.ProviderProfile"
	profile.SchemaVersion = "0.1.0"
	var err error
	profile, err = normalizeProviderProfileIdentity(profile)
	if err != nil {
		return ProviderProfile{}, err
	}
	profile.SupportedAuthModes = normalizeAuthModes(profile.SupportedAuthModes, profile.CurrentAuthMode)
	profile.AllowlistedModelIDs = normalizedStringSet(profile.AllowlistedModelIDs)
	profile.ModelCatalogPosture = normalizeProviderModelCatalogPosture(profile.ModelCatalogPosture)
	profile = applyProviderProfileDefaults(profile)
	profile.AuthMaterial = normalizeProviderAuthMaterial(profile.AuthMaterial)
	profile.ReadinessPosture = normalizeProviderReadinessPosture(profile.ReadinessPosture)
	profile.Lifecycle.ValidationAttemptCount = maxInt64(profile.Lifecycle.ValidationAttemptCount, 0)
	return profile, nil
}

func stableProviderProfileID(providerFamily, destinationRef string) string {
	seed := strings.ToLower(strings.TrimSpace(providerFamily)) + "|" + strings.ToLower(strings.TrimSpace(destinationRef))
	sum := sha256.Sum256([]byte(seed))
	return "provider-profile-" + hex.EncodeToString(sum[:12])
}
