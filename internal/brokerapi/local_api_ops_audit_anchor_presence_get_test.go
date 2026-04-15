package brokerapi

import (
	"context"
	"strings"
	"testing"
)

func TestHandleAuditAnchorPresenceGetReturnsAttestationForOSConfirmation(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorPresenceGet(context.Background(), AuditAnchorPresenceGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPresenceGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-presence-get",
		SealDigest:    sealDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorPresenceGet returned error response: %+v", errResp)
	}
	if got := strings.TrimSpace(resp.PresenceMode); got != "os_confirmation" {
		t.Fatalf("presence_mode = %q, want os_confirmation", got)
	}
	if resp.PresenceAttestation == nil {
		t.Fatal("presence_attestation = nil, want attestation for os_confirmation")
	}
	if strings.TrimSpace(resp.PresenceAttestation.Challenge) == "" {
		t.Fatal("presence_attestation.challenge is empty")
	}
	if len(resp.PresenceAttestation.AcknowledgmentToken) != 64 {
		t.Fatalf("presence_attestation.acknowledgment_token length = %d, want 64", len(resp.PresenceAttestation.AcknowledgmentToken))
	}
	expected, err := service.secretsSvc.ComputeAuditAnchorPresenceAcknowledgmentToken(resp.PresenceMode, sealDigest, resp.PresenceAttestation.Challenge)
	if err != nil {
		t.Fatalf("ComputeAuditAnchorPresenceAcknowledgmentToken returned error: %v", err)
	}
	if resp.PresenceAttestation.AcknowledgmentToken != expected {
		t.Fatalf("presence_attestation.acknowledgment_token mismatch")
	}
}

func TestHandleAuditAnchorPresenceGetReturnsNoAttestationForPassphrase(t *testing.T) {
	t.Setenv("RUNE_AUDIT_ANCHOR_PRESENCE_MODE", "passphrase")
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorPresenceGet(context.Background(), AuditAnchorPresenceGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPresenceGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-presence-passphrase",
		SealDigest:    sealDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorPresenceGet returned error response: %+v", errResp)
	}
	if got := strings.TrimSpace(resp.PresenceMode); got != "passphrase" {
		t.Fatalf("presence_mode = %q, want passphrase", got)
	}
	if resp.PresenceAttestation != nil {
		t.Fatalf("presence_attestation = %+v, want nil for passphrase", resp.PresenceAttestation)
	}
}

func TestHandleAuditAnchorPresenceGetRejectsInvalidSealDigest(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	_, errResp := service.HandleAuditAnchorPresenceGet(context.Background(), AuditAnchorPresenceGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPresenceGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-presence-invalid-seal",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditAnchorPresenceGet expected validation error response")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestHandleAuditAnchorPresenceGetReturnsTypedSignerUnavailableError(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	service.secretsSvc = nil
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	_, errResp := service.HandleAuditAnchorPresenceGet(context.Background(), AuditAnchorPresenceGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPresenceGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-presence-no-signer",
		SealDigest:    sealDigest,
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditAnchorPresenceGet expected typed error response")
	}
	if errResp.Error.Code != auditAnchorErrorCodeSignerUnavailable {
		t.Fatalf("error code = %q, want %q", errResp.Error.Code, auditAnchorErrorCodeSignerUnavailable)
	}
}
