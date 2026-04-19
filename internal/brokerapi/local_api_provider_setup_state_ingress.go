package brokerapi

import (
	"fmt"
	"strings"
	"time"
)

func (s *providerSetupState) prepareIngress(setupSessionID, ingressChannel, credentialField string, ttl time.Duration) (ProviderSetupSession, providerSetupIngressRecord, error) {
	if s == nil {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("provider setup state unavailable")
	}
	id := strings.TrimSpace(setupSessionID)
	if id == "" {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("setup_session_id is required")
	}
	channel, field, err := normalizeProviderIngressRequest(ingressChannel, credentialField)
	if err != nil {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, err
	}
	rec, err := s.newIngressRecord(id, channel, field, ttl)
	if err != nil {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, err
	}
	entry, err := s.prepareIngressSession(rec)
	if err != nil {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, err
	}
	return entry, rec, nil
}

func normalizeProviderIngressRequest(ingressChannel, credentialField string) (string, string, error) {
	channel := strings.TrimSpace(ingressChannel)
	field := strings.TrimSpace(credentialField)
	if channel == "" {
		channel = "cli_stdin"
	}
	if field == "" {
		field = "api_key"
	}
	if channel == "environment_variable" || channel == "cli_argument" {
		return "", "", fmt.Errorf("ingress_channel %q is forbidden", channel)
	}
	return channel, field, nil
}

func (s *providerSetupState) newIngressRecord(sessionID, channel, field string, ttl time.Duration) (providerSetupIngressRecord, error) {
	token, err := randomProviderSetupToken(s.rand, "provider-secret-ingress-")
	if err != nil {
		return providerSetupIngressRecord{}, err
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := s.nowFn().UTC()
	return providerSetupIngressRecord{Token: token, SetupSessionID: sessionID, CredentialField: field, IngressChannel: channel, ExpiresAt: now.Add(ttl)}, nil
}

func (s *providerSetupState) prepareIngressSession(rec providerSetupIngressRecord) (ProviderSetupSession, error) {
	now := s.nowFn().UTC().Format(time.RFC3339)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.session[rec.SetupSessionID]
	if !ok {
		return ProviderSetupSession{}, fmt.Errorf("setup session not found")
	}
	entry = providerSessionReadyForIngress(entry, now)
	if err := s.persistProviderSetupSession(entry); err != nil {
		return ProviderSetupSession{}, err
	}
	s.invalidateIngressForSessionLocked(rec.SetupSessionID)
	s.session[rec.SetupSessionID] = entry
	s.ingress[rec.Token] = rec
	return entry, nil
}

func providerSessionReadyForIngress(entry ProviderSetupSession, updatedAt string) ProviderSetupSession {
	entry.CurrentPhase = providerSetupPhaseSecretIngressReady
	entry.SecretIngressReady = true
	entry.ValidationStatus = providerSetupValidationStatusNotStarted
	entry.ValidationAttemptID = ""
	entry.ReadinessCommitted = false
	entry.UpdatedAt = updatedAt
	return entry
}

func (s *providerSetupState) consumeIngress(token string) (ProviderSetupSession, providerSetupIngressRecord, error) {
	if s == nil {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("provider setup state unavailable")
	}
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("secret_ingress_token is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, entry, err := s.lookupActiveIngress(trimmed, s.nowFn().UTC())
	if err != nil {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, err
	}
	rec.Used = true
	s.ingress[trimmed] = rec
	return entry, rec, nil
}

func (s *providerSetupState) completeIngress(token string) (ProviderSetupSession, error) {
	if s == nil {
		return ProviderSetupSession{}, fmt.Errorf("provider setup state unavailable")
	}
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ProviderSetupSession{}, fmt.Errorf("secret_ingress_token is required")
	}
	now := s.nowFn().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.ingress[trimmed]
	if !ok {
		return ProviderSetupSession{}, fmt.Errorf("secret ingress token not found")
	}
	if !rec.Used {
		return ProviderSetupSession{}, fmt.Errorf("secret ingress token not consumed")
	}
	entry, ok := s.session[rec.SetupSessionID]
	if !ok {
		return ProviderSetupSession{}, fmt.Errorf("setup session not found")
	}
	s.invalidateIngressForSessionLocked(rec.SetupSessionID)
	entry = providerSessionCredentialCommitted(entry, now.Format(time.RFC3339))
	if err := s.persistProviderSetupSession(entry); err != nil {
		return ProviderSetupSession{}, err
	}
	s.session[entry.SetupSessionID] = entry
	return entry, nil
}

func providerSessionCredentialCommitted(entry ProviderSetupSession, updatedAt string) ProviderSetupSession {
	entry.CurrentPhase = providerSetupPhaseCredentialCommitted
	entry.SecretIngressReady = false
	entry.ValidationStatus = providerSetupValidationStatusNotStarted
	entry.ValidationAttemptID = ""
	entry.ReadinessCommitted = false
	entry.UpdatedAt = updatedAt
	return entry
}

func (s *providerSetupState) lookupActiveIngress(token string, now time.Time) (providerSetupIngressRecord, ProviderSetupSession, error) {
	rec, ok := s.ingress[token]
	if !ok {
		return providerSetupIngressRecord{}, ProviderSetupSession{}, fmt.Errorf("secret ingress token not found")
	}
	if rec.Used || !rec.ExpiresAt.After(now) {
		return providerSetupIngressRecord{}, ProviderSetupSession{}, fmt.Errorf("secret ingress token expired")
	}
	entry, ok := s.session[rec.SetupSessionID]
	if !ok {
		return providerSetupIngressRecord{}, ProviderSetupSession{}, fmt.Errorf("setup session not found")
	}
	return rec, entry, nil
}
