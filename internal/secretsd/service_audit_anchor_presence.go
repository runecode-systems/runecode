package secretsd

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	auditAnchorPresenceKeyBytes   = 32
	auditAnchorChallengeMinLength = 16
)

func validateAuditAnchorPresenceAttestation(s *Service, req AuditAnchorSignRequest, mode string) error {
	mode = strings.TrimSpace(mode)
	if !presenceAttestationRequired(mode) {
		return nil
	}
	if s == nil {
		return fmt.Errorf("audit anchor signer service is required for presence attestation validation")
	}
	if req.PresenceAttestation == nil {
		return fmt.Errorf("audit anchor presence attestation is required for presence_mode %q", mode)
	}
	challenge := strings.TrimSpace(req.PresenceAttestation.Challenge)
	if challenge == "" {
		return fmt.Errorf("audit anchor presence attestation challenge is required for presence_mode %q", mode)
	}
	if len(challenge) < auditAnchorChallengeMinLength {
		return fmt.Errorf("audit anchor presence attestation challenge must be at least %d characters", auditAnchorChallengeMinLength)
	}
	providedToken := strings.TrimSpace(req.PresenceAttestation.AcknowledgmentToken)
	if providedToken == "" {
		return fmt.Errorf("audit anchor presence attestation acknowledgment_token is required for presence_mode %q", mode)
	}
	expectedToken, err := s.computeAuditAnchorPresenceAcknowledgmentTokenLocked(mode, req.TargetSealDigest, challenge)
	if err != nil {
		return fmt.Errorf("audit anchor presence attestation: %w", err)
	}
	if !hmac.Equal([]byte(expectedToken), []byte(providedToken)) {
		return fmt.Errorf("audit anchor presence attestation acknowledgment_token is invalid for presence_mode %q", mode)
	}
	return nil
}

func presenceAttestationRequired(mode string) bool {
	mode = strings.TrimSpace(mode)
	return mode == "os_confirmation" || mode == "hardware_touch"
}

func (s *Service) ComputeAuditAnchorPresenceAcknowledgmentToken(mode string, targetSealDigest trustpolicy.Digest, challenge string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("service is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.computeAuditAnchorPresenceAcknowledgmentTokenLocked(mode, targetSealDigest, challenge)
}

func (s *Service) computeAuditAnchorPresenceAcknowledgmentTokenLocked(mode string, targetSealDigest trustpolicy.Digest, challenge string) (string, error) {
	mode = strings.TrimSpace(mode)
	if !presenceAttestationRequired(mode) {
		return "", fmt.Errorf("presence_mode %q does not support acknowledgment token generation", mode)
	}
	key, err := s.auditAnchorPresenceKeyLocked()
	if err != nil {
		return "", err
	}
	defer zeroBytes(key)
	sealIdentity, err := targetSealDigest.Identity()
	if err != nil {
		return "", fmt.Errorf("target_seal_digest: %w", err)
	}
	challenge = strings.TrimSpace(challenge)
	if challenge == "" {
		return "", fmt.Errorf("challenge is required")
	}
	if len(challenge) < auditAnchorChallengeMinLength {
		return "", fmt.Errorf("challenge must be at least %d characters", auditAnchorChallengeMinLength)
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("audit_anchor_presence_v1|"))
	_, _ = mac.Write([]byte(mode))
	_, _ = mac.Write([]byte("|"))
	_, _ = mac.Write([]byte(sealIdentity))
	_, _ = mac.Write([]byte("|"))
	_, _ = mac.Write([]byte(challenge))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *Service) auditAnchorPresenceKeyLocked() ([]byte, error) {
	if inline := strings.TrimSpace(os.Getenv(envAuditAnchorPresenceKeyB64)); inline != "" {
		return decodeAuditAnchorPresenceKey(inline)
	}
	material, err := loadOrCreateAuditAnchorPresenceKey(filepath.Join(s.root, auditAnchorPresenceKeyFileName), s.rand)
	if err != nil {
		return nil, err
	}
	return decodeAuditAnchorPresenceKey(strings.TrimSpace(string(material)))
}

func loadOrCreateAuditAnchorPresenceKey(path string, randSource io.Reader) ([]byte, error) {
	material, err := os.ReadFile(path)
	if err == nil {
		return material, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	key := make([]byte, auditAnchorPresenceKeyBytes)
	if _, err := io.ReadFull(randSource, key); err != nil {
		return nil, err
	}
	defer zeroBytes(key)
	encoded := base64.StdEncoding.EncodeToString(key)
	if err := writeFileAtomic(path, []byte(encoded), 0o600); err != nil {
		return nil, err
	}
	return []byte(encoded), nil
}

func decodeAuditAnchorPresenceKey(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode audit anchor presence key: %w", err)
	}
	defer zeroBytes(decoded)
	if len(decoded) != auditAnchorPresenceKeyBytes {
		return nil, fmt.Errorf("audit anchor presence key must be %d bytes", auditAnchorPresenceKeyBytes)
	}
	return append([]byte(nil), decoded...), nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
