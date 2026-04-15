package secretsd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func validateAuditAnchorPresenceAttestation(req AuditAnchorSignRequest, mode string) error {
	mode = strings.TrimSpace(mode)
	if !presenceAttestationRequired(mode) {
		return nil
	}
	if req.PresenceAttestation == nil {
		return fmt.Errorf("audit anchor presence attestation is required for presence_mode %q", mode)
	}
	challenge := strings.TrimSpace(req.PresenceAttestation.Challenge)
	if challenge == "" {
		return fmt.Errorf("audit anchor presence attestation challenge is required for presence_mode %q", mode)
	}
	providedToken := strings.TrimSpace(req.PresenceAttestation.AcknowledgmentToken)
	if providedToken == "" {
		return fmt.Errorf("audit anchor presence attestation acknowledgment_token is required for presence_mode %q", mode)
	}
	expectedToken, err := ComputeAuditAnchorPresenceAcknowledgmentToken(mode, req.TargetSealDigest, challenge)
	if err != nil {
		return fmt.Errorf("audit anchor presence attestation: %w", err)
	}
	if !strings.EqualFold(expectedToken, providedToken) {
		return fmt.Errorf("audit anchor presence attestation acknowledgment_token is invalid for presence_mode %q", mode)
	}
	return nil
}

func presenceAttestationRequired(mode string) bool {
	mode = strings.TrimSpace(mode)
	return mode == "os_confirmation" || mode == "hardware_touch"
}

func ComputeAuditAnchorPresenceAcknowledgmentToken(mode string, targetSealDigest trustpolicy.Digest, challenge string) (string, error) {
	mode = strings.TrimSpace(mode)
	if !presenceAttestationRequired(mode) {
		return "", fmt.Errorf("presence_mode %q does not support acknowledgment token generation", mode)
	}
	sealIdentity, err := targetSealDigest.Identity()
	if err != nil {
		return "", fmt.Errorf("target_seal_digest: %w", err)
	}
	challenge = strings.TrimSpace(challenge)
	if challenge == "" {
		return "", fmt.Errorf("challenge is required")
	}
	sum := sha256.Sum256([]byte("audit_anchor_presence_v1|" + mode + "|" + sealIdentity + "|" + challenge))
	return hex.EncodeToString(sum[:]), nil
}
