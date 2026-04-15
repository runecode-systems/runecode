package brokerapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func mustAuditAnchorPresenceAttestation(t *testing.T, service *Service, mode string, sealDigest trustpolicy.Digest) *AuditAnchorPresenceAttestation {
	t.Helper()
	challenge := "presence-challenge-" + strings.Repeat("a", 16)
	token, err := auditAnchorPresenceTokenForBrokerTest(service, mode, sealDigest, challenge)
	if err != nil {
		t.Fatalf("auditAnchorPresenceTokenForBrokerTest returned error: %v", err)
	}
	return &AuditAnchorPresenceAttestation{Challenge: challenge, AcknowledgmentToken: token}
}

func auditAnchorPresenceTokenForBrokerTest(service *Service, mode string, sealDigest trustpolicy.Digest, challenge string) (string, error) {
	if service == nil || service.secretsSvc == nil {
		return "", fmt.Errorf("secrets service unavailable")
	}
	return service.secretsSvc.ComputeAuditAnchorPresenceAcknowledgmentToken(mode, sealDigest, challenge)
}
