package brokerapi

import (
	"context"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestHandleZKProofGenerateAndVerifyEndToEnd(t *testing.T) {
	service, recordDigest := mustSetupZKProofE2EService(t)
	_, errResp := service.HandleZKProofGenerate(context.Background(), ZKProofGenerateRequest{SchemaID: "runecode.protocol.v0.ZKProofGenerateRequest", SchemaVersion: "0.1.0", RequestID: "req-zk-generate", RecordDigest: recordDigest}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleZKProofGenerate expected fail-closed backend-disabled error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func mustSetupZKProofE2EService(t *testing.T) (*Service, trustpolicy.Digest) {
	t.Helper()
	storeRoot := t.TempDir()
	ledgerRoot := t.TempDir()
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	recordSeedRuntimeEvidenceForSession(t, service, "session-1")
	surface, err := service.LatestAuditVerificationSurface(1)
	if err != nil {
		t.Fatalf("LatestAuditVerificationSurface returned error: %v", err)
	}
	if len(surface.Views) == 0 {
		t.Fatal("expected at least one audit view")
	}
	return service, surface.Views[0].RecordDigest
}

func TestHandleZKProofVerifyNotFound(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	_, errResp := s.HandleZKProofVerify(context.Background(), ZKProofVerifyRequest{
		SchemaID:              "runecode.protocol.v0.ZKProofVerifyRequest",
		SchemaVersion:         "0.1.0",
		RequestID:             "req-zk-missing",
		ZKProofArtifactDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleZKProofVerify expected not-found error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestHandleZKProofGenerateFailsClosedWhenBackendDisabledEvenWithRuntimeEvidence(t *testing.T) {
	storeRoot := t.TempDir()
	ledgerRoot := t.TempDir()
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	recordSeedRuntimeEvidenceForSession(t, service, "session-1")
	surface, err := service.LatestAuditVerificationSurface(1)
	if err != nil {
		t.Fatalf("LatestAuditVerificationSurface returned error: %v", err)
	}
	_, errResp := service.HandleZKProofGenerate(context.Background(), ZKProofGenerateRequest{
		SchemaID:      "runecode.protocol.v0.ZKProofGenerateRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-zk-generate-disabled",
		RecordDigest:  surface.Views[0].RecordDigest,
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleZKProofGenerate expected fail-closed backend-disabled error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}
