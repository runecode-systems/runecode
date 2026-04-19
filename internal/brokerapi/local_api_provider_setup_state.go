package brokerapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	providerSetupPhaseMetadataConfigured = "metadata_configured"
	providerSetupPhaseSecretIngressReady = "secret_ingress_ready"
	providerSetupPhaseConfigured         = "configured"
)

type providerSetupState struct {
	mu      sync.Mutex
	nowFn   func() time.Time
	rand    io.Reader
	session map[string]ProviderSetupSession
	ingress map[string]providerSetupIngressRecord
}

type providerSetupIngressRecord struct {
	Token           string
	SetupSessionID  string
	CredentialField string
	IngressChannel  string
	ExpiresAt       time.Time
	Used            bool
}

func newProviderSetupState(nowFn func() time.Time) *providerSetupState {
	if nowFn == nil {
		nowFn = time.Now
	}
	return &providerSetupState{
		nowFn:   nowFn,
		rand:    rand.Reader,
		session: map[string]ProviderSetupSession{},
		ingress: map[string]providerSetupIngressRecord{},
	}
}

func (s *providerSetupState) setNowFunc(nowFn func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if nowFn == nil {
		s.nowFn = time.Now
		return
	}
	s.nowFn = nowFn
}

func (s *providerSetupState) begin(profile ProviderProfile) (ProviderSetupSession, error) {
	if s == nil {
		return ProviderSetupSession{}, fmt.Errorf("provider setup state unavailable")
	}
	if strings.TrimSpace(profile.ProviderProfileID) == "" {
		return ProviderSetupSession{}, fmt.Errorf("provider_profile_id is required")
	}
	now := s.nowFn().UTC().Format(time.RFC3339)
	token, err := randomProviderSetupToken(s.rand, "provider-setup-session-")
	if err != nil {
		return ProviderSetupSession{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := ProviderSetupSession{
		SchemaID:           "runecode.protocol.v0.ProviderSetupSession",
		SchemaVersion:      "0.1.0",
		SetupSessionID:     token,
		ProviderProfileID:  profile.ProviderProfileID,
		ProviderFamily:     profile.ProviderFamily,
		SupportedAuthModes: append([]string{}, profile.SupportedAuthModes...),
		CurrentPhase:       providerSetupPhaseMetadataConfigured,
		CurrentAuthMode:    profile.CurrentAuthMode,
		SecretIngressReady: false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.session[entry.SetupSessionID] = entry
	return entry, nil
}

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
	token, err := randomProviderSetupToken(s.rand, "provider-secret-ingress-")
	if err != nil {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, err
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := s.nowFn().UTC()
	rec := providerSetupIngressRecord{Token: token, SetupSessionID: id, CredentialField: field, IngressChannel: channel, ExpiresAt: now.Add(ttl)}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.session[id]
	if !ok {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("setup session not found")
	}
	s.invalidateIngressForSessionLocked(id)
	entry.CurrentPhase = providerSetupPhaseSecretIngressReady
	entry.SecretIngressReady = true
	entry.UpdatedAt = now.Format(time.RFC3339)
	s.session[id] = entry
	s.ingress[token] = rec
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

func (s *providerSetupState) consumeIngress(token string) (ProviderSetupSession, providerSetupIngressRecord, error) {
	if s == nil {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("provider setup state unavailable")
	}
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("secret_ingress_token is required")
	}
	now := s.nowFn().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.ingress[trimmed]
	if !ok {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("secret ingress token not found")
	}
	if rec.Used || !rec.ExpiresAt.After(now) {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("secret ingress token expired")
	}
	entry, ok := s.session[rec.SetupSessionID]
	if !ok {
		return ProviderSetupSession{}, providerSetupIngressRecord{}, fmt.Errorf("setup session not found")
	}
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
	if rec.Used || !rec.ExpiresAt.After(now) {
		return ProviderSetupSession{}, fmt.Errorf("secret ingress token expired")
	}
	entry, ok := s.session[rec.SetupSessionID]
	if !ok {
		return ProviderSetupSession{}, fmt.Errorf("setup session not found")
	}
	s.invalidateIngressForSessionLocked(rec.SetupSessionID)
	entry.CurrentPhase = providerSetupPhaseConfigured
	entry.SecretIngressReady = false
	entry.UpdatedAt = now.Format(time.RFC3339)
	s.session[entry.SetupSessionID] = entry
	return entry, nil
}

func (s *providerSetupState) invalidateIngressForSessionLocked(setupSessionID string) {
	for token, rec := range s.ingress {
		if rec.SetupSessionID == setupSessionID {
			delete(s.ingress, token)
		}
	}
}

func randomProviderSetupToken(reader io.Reader, prefix string) (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(reader, b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}
