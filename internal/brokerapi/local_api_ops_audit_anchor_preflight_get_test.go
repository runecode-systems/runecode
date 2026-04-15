package brokerapi

import (
	"context"
	"strings"
	"testing"
)

func TestHandleAuditAnchorPreflightGetReturnsLatestSealAndReadiness(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	resp, errResp := service.HandleAuditAnchorPreflightGet(context.Background(), AuditAnchorPreflightGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPreflightGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-preflight",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorPreflightGet returned error response: %+v", errResp)
	}
	if resp.LatestAnchorableSeal == nil {
		t.Fatal("latest_anchorable_seal = nil, want populated")
	}
	if got := strings.TrimSpace(resp.LatestAnchorableSeal.SegmentID); got == "" {
		t.Fatal("latest_anchorable_seal.segment_id empty")
	}
	if !resp.SignerReadiness.Ready {
		t.Fatalf("signer_readiness.ready = false, want true: %+v", resp.SignerReadiness)
	}
	if !resp.VerifierReadiness.Ready {
		t.Fatalf("verifier_readiness.ready = false, want true: %+v", resp.VerifierReadiness)
	}
	if !resp.PresenceRequirements.Required {
		t.Fatalf("presence_requirements.required = false, want true for os_confirmation")
	}
	if !resp.PresenceRequirements.AttestationReady {
		t.Fatalf("presence_requirements.attestation_ready = false, want true")
	}
}

func TestHandleAuditAnchorPreflightGetFailsWhenLedgerMissing(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	service.auditLedger = nil
	_, errResp := service.HandleAuditAnchorPreflightGet(context.Background(), AuditAnchorPreflightGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPreflightGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-preflight-no-ledger",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditAnchorPreflightGet expected typed error response")
	}
	if errResp.Error.Code != auditAnchorErrorCodeLedgerUnavailable {
		t.Fatalf("error code = %q, want %q", errResp.Error.Code, auditAnchorErrorCodeLedgerUnavailable)
	}
}

func TestHandleAuditAnchorPreflightGetSignerUnavailableWhenSecretsMissing(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	service.secretsSvc = nil
	resp, errResp := service.HandleAuditAnchorPreflightGet(context.Background(), AuditAnchorPreflightGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPreflightGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-preflight-no-signer",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorPreflightGet returned error response: %+v", errResp)
	}
	if resp.SignerReadiness.Ready {
		t.Fatalf("signer_readiness.ready = true, want false")
	}
	if resp.SignerReadiness.ReasonCode != "signer_unavailable" {
		t.Fatalf("signer_readiness.reason_code = %q, want signer_unavailable", resp.SignerReadiness.ReasonCode)
	}
}
