package brokerapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

var errProviderValidationCommitPrecondition = errors.New("provider validation commit precondition failed")

const (
	providerSetupPhaseMetadataConfigured   = "metadata_configured"
	providerSetupPhaseSecretIngressReady   = "secret_ingress_ready"
	providerSetupPhaseCredentialCommitted  = "credential_committed"
	providerSetupPhaseValidationInProgress = "validation_in_progress"
	providerSetupPhaseReadinessCommitted   = "readiness_committed"

	providerSetupValidationStatusNotStarted = "not_started"
	providerSetupValidationStatusInProgress = "in_progress"
	providerSetupValidationStatusSucceeded  = "succeeded"
	providerSetupValidationStatusFailed     = "failed"
)

type providerSetupState struct {
	mu             sync.Mutex
	nowFn          func() time.Time
	rand           io.Reader
	session        map[string]ProviderSetupSession
	ingress        map[string]providerSetupIngressRecord
	persistSession func(ProviderSetupSession) error
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

func (s *providerSetupState) setPersistFunc(persistFn func(ProviderSetupSession) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persistSession = persistFn
}

func (s *providerSetupState) restoreSessions(sessions []ProviderSetupSession) {
	if s == nil {
		return
	}
	next := make(map[string]ProviderSetupSession, len(sessions))
	for _, session := range sessions {
		normalized := normalizeSetupSessionForRestore(session)
		next[normalized.SetupSessionID] = normalized
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = next
	s.ingress = map[string]providerSetupIngressRecord{}
}

func normalizeSetupSessionForRestore(session ProviderSetupSession) ProviderSetupSession {
	session.SchemaID = "runecode.protocol.v0.ProviderSetupSession"
	session.SchemaVersion = "0.1.0"
	session.SetupSessionID = strings.TrimSpace(session.SetupSessionID)
	session.ProviderProfileID = strings.TrimSpace(session.ProviderProfileID)
	session.ProviderFamily = strings.TrimSpace(session.ProviderFamily)
	session.SupportedAuthModes = normalizedStringSet(session.SupportedAuthModes)
	session.CurrentPhase = strings.TrimSpace(session.CurrentPhase)
	session.CurrentAuthMode = strings.TrimSpace(session.CurrentAuthMode)
	session.ValidationStatus = strings.TrimSpace(session.ValidationStatus)
	session.ValidationAttemptID = strings.TrimSpace(session.ValidationAttemptID)
	session.CreatedAt = strings.TrimSpace(session.CreatedAt)
	session.UpdatedAt = strings.TrimSpace(session.UpdatedAt)
	if session.SecretIngressReady || session.CurrentPhase == providerSetupPhaseSecretIngressReady {
		session.SecretIngressReady = false
		session.CurrentPhase = providerSetupPhaseMetadataConfigured
	}
	if session.CurrentPhase == "" {
		session.CurrentPhase = providerSetupPhaseMetadataConfigured
	}
	if session.CurrentPhase == "configured" {
		session.CurrentPhase = providerSetupPhaseCredentialCommitted
	}
	if session.ValidationStatus == "" {
		session.ValidationStatus = providerSetupValidationStatusNotStarted
	}
	if session.CurrentPhase == providerSetupPhaseReadinessCommitted {
		session.ReadinessCommitted = true
	}
	return session
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
		ValidationStatus:   providerSetupValidationStatusNotStarted,
		ReadinessCommitted: false,
		SecretIngressReady: false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if s.persistSession != nil {
		if err := s.persistSession(entry); err != nil {
			return ProviderSetupSession{}, err
		}
	}
	s.session[entry.SetupSessionID] = entry
	return entry, nil
}

func (s *providerSetupState) latestSessionByProfileIDLocked(profileID string) (string, ProviderSetupSession, bool) {
	bestKey := ""
	best := ProviderSetupSession{}
	for key, entry := range s.session {
		if entry.ProviderProfileID != profileID {
			continue
		}
		if bestKey == "" || entry.UpdatedAt > best.UpdatedAt || (entry.UpdatedAt == best.UpdatedAt && key > bestKey) {
			bestKey = key
			best = entry
		}
	}
	if bestKey == "" {
		return "", ProviderSetupSession{}, false
	}
	return bestKey, best, true
}

func (s *providerSetupState) persistProviderSetupSession(entry ProviderSetupSession) error {
	if s.persistSession == nil {
		return nil
	}
	return s.persistSession(entry)
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
