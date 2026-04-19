package brokerapi

import (
	"fmt"
	"strings"
	"time"
)

func (s *providerSetupState) startValidationForProfile(profileID, attemptID string) (ProviderSetupSession, error) {
	if s == nil {
		return ProviderSetupSession{}, fmt.Errorf("provider setup state unavailable")
	}
	id := strings.TrimSpace(profileID)
	if id == "" {
		return ProviderSetupSession{}, fmt.Errorf("provider_profile_id is required")
	}
	validationAttemptID, err := s.resolveValidationAttemptID(attemptID)
	if err != nil {
		return ProviderSetupSession{}, err
	}
	now := s.nowFn().UTC().Format(time.RFC3339)
	s.mu.Lock()
	defer s.mu.Unlock()
	key, entry, ok := s.latestSessionByProfileIDLocked(id)
	if !ok {
		return ProviderSetupSession{}, fmt.Errorf("setup session not found")
	}
	entry = providerSessionValidationInProgress(entry, validationAttemptID, now)
	if err := s.persistProviderSetupSession(entry); err != nil {
		return ProviderSetupSession{}, err
	}
	s.session[key] = entry
	return entry, nil
}

func (s *providerSetupState) commitValidationForProfile(profileID, attemptID, outcome string) (ProviderSetupSession, error) {
	if s == nil {
		return ProviderSetupSession{}, fmt.Errorf("provider setup state unavailable")
	}
	validationAttemptID, status, err := normalizeValidationCommitRequest(profileID, attemptID, outcome)
	if err != nil {
		return ProviderSetupSession{}, err
	}
	now := s.nowFn().UTC().Format(time.RFC3339)
	s.mu.Lock()
	defer s.mu.Unlock()
	key, entry, ok := s.latestSessionByProfileIDLocked(strings.TrimSpace(profileID))
	if !ok {
		return ProviderSetupSession{}, fmt.Errorf("setup session not found")
	}
	if err := validateValidationCommitSession(entry, validationAttemptID); err != nil {
		return ProviderSetupSession{}, err
	}
	entry = providerSessionReadinessCommitted(entry, validationAttemptID, status, now)
	if err := s.persistProviderSetupSession(entry); err != nil {
		return ProviderSetupSession{}, err
	}
	s.session[key] = entry
	return entry, nil
}

func (s *providerSetupState) resolveValidationAttemptID(attemptID string) (string, error) {
	validationAttemptID := strings.TrimSpace(attemptID)
	if validationAttemptID != "" {
		return validationAttemptID, nil
	}
	return randomProviderSetupToken(s.rand, "provider-validation-attempt-")
}

func normalizeValidationCommitRequest(profileID, attemptID, outcome string) (string, string, error) {
	if strings.TrimSpace(profileID) == "" {
		return "", "", fmt.Errorf("provider_profile_id is required")
	}
	validationAttemptID := strings.TrimSpace(attemptID)
	if validationAttemptID == "" {
		return "", "", fmt.Errorf("validation_attempt_id is required")
	}
	resolvedOutcome := strings.TrimSpace(strings.ToLower(outcome))
	status := providerSetupValidationStatusFailed
	if resolvedOutcome == providerSetupValidationStatusSucceeded || resolvedOutcome == "ready" {
		status = providerSetupValidationStatusSucceeded
	}
	return validationAttemptID, status, nil
}

func validateValidationCommitSession(entry ProviderSetupSession, validationAttemptID string) error {
	if strings.TrimSpace(entry.CurrentPhase) != providerSetupPhaseValidationInProgress || strings.TrimSpace(entry.ValidationStatus) != providerSetupValidationStatusInProgress {
		return fmt.Errorf("%w: validation session is not in progress", errProviderValidationCommitPrecondition)
	}
	if strings.TrimSpace(entry.ValidationAttemptID) != validationAttemptID {
		return fmt.Errorf("%w: validation_attempt_id does not match in-progress session", errProviderValidationCommitPrecondition)
	}
	return nil
}

func providerSessionValidationInProgress(entry ProviderSetupSession, validationAttemptID, updatedAt string) ProviderSetupSession {
	entry.CurrentPhase = providerSetupPhaseValidationInProgress
	entry.ValidationStatus = providerSetupValidationStatusInProgress
	entry.ValidationAttemptID = validationAttemptID
	entry.ReadinessCommitted = false
	entry.UpdatedAt = updatedAt
	return entry
}

func providerSessionReadinessCommitted(entry ProviderSetupSession, validationAttemptID, status, updatedAt string) ProviderSetupSession {
	entry.CurrentPhase = providerSetupPhaseReadinessCommitted
	entry.ValidationStatus = status
	entry.ValidationAttemptID = validationAttemptID
	entry.ReadinessCommitted = true
	entry.SecretIngressReady = false
	entry.UpdatedAt = updatedAt
	return entry
}
