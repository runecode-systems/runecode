package brokerapi

import (
	"context"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestHandleZKProofGenerateAndVerifyEndToEnd(t *testing.T) {
	service, recordDigest := mustSetupZKProofE2EService(t)
	generateResp := mustGenerateZKProof(t, service, recordDigest)
	assertGeneratedZKProofResponse(t, generateResp)
	verifyResp := mustVerifyZKProof(t, service, generateResp.ZKProofArtifactDigest)
	assertVerifiedZKProofResponse(t, verifyResp)
}

func mustSetupZKProofE2EService(t *testing.T) (*Service, trustpolicy.Digest) {
	t.Helper()
	storeRoot := t.TempDir()
	ledgerRoot := t.TempDir()
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t), ZKProof: ZKProofConfig{EnableFixtureBackend: true}})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	surface, err := service.LatestAuditVerificationSurface(1)
	if err != nil {
		t.Fatalf("LatestAuditVerificationSurface returned error: %v", err)
	}
	if len(surface.Views) == 0 {
		t.Fatal("expected at least one audit view")
	}
	return service, surface.Views[0].RecordDigest
}

func mustGenerateZKProof(t *testing.T, service *Service, recordDigest trustpolicy.Digest) ZKProofGenerateResponse {
	t.Helper()
	resp, errResp := service.HandleZKProofGenerate(context.Background(), ZKProofGenerateRequest{SchemaID: "runecode.protocol.v0.ZKProofGenerateRequest", SchemaVersion: "0.1.0", RequestID: "req-zk-generate", RecordDigest: recordDigest}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleZKProofGenerate error response: %+v", errResp)
	}
	return resp
}

func assertGeneratedZKProofResponse(t *testing.T, resp ZKProofGenerateResponse) {
	t.Helper()
	if resp.StatementFamily != zkProofStatementFamilyV0 {
		t.Fatalf("statement_family = %q, want %q", resp.StatementFamily, zkProofStatementFamilyV0)
	}
	if resp.ZKProofVerificationDigest == nil {
		t.Fatal("zk_proof_verification_record_digest missing")
	}
	if !resp.UserCheckInRequired {
		t.Fatal("user_check_in_required=false, want true")
	}
}

func mustVerifyZKProof(t *testing.T, service *Service, artifactDigest trustpolicy.Digest) ZKProofVerifyResponse {
	t.Helper()
	resp, errResp := service.HandleZKProofVerify(context.Background(), ZKProofVerifyRequest{SchemaID: "runecode.protocol.v0.ZKProofVerifyRequest", SchemaVersion: "0.1.0", RequestID: "req-zk-verify", ZKProofArtifactDigest: artifactDigest}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleZKProofVerify error response: %+v", errResp)
	}
	return resp
}

func assertVerifiedZKProofResponse(t *testing.T, resp ZKProofVerifyResponse) {
	t.Helper()
	if resp.VerificationOutcome != trustpolicy.ProofVerificationOutcomeVerified {
		t.Fatalf("verification_outcome = %q, want %q", resp.VerificationOutcome, trustpolicy.ProofVerificationOutcomeVerified)
	}
	if len(resp.ReasonCodes) == 0 || resp.ReasonCodes[0] != trustpolicy.ProofVerificationReasonVerified {
		t.Fatalf("reason_codes = %v, want first code %q", resp.ReasonCodes, trustpolicy.ProofVerificationReasonVerified)
	}
	if resp.EvaluationGate == "" {
		t.Fatal("evaluation_gate is empty")
	}
}

func TestHandleZKProofVerifyNotFound(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t), ZKProof: ZKProofConfig{EnableFixtureBackend: true}})
	_, errResp := s.HandleZKProofVerify(context.Background(), ZKProofVerifyRequest{
		SchemaID:              "runecode.protocol.v0.ZKProofVerifyRequest",
		SchemaVersion:         "0.1.0",
		RequestID:             "req-zk-missing",
		ZKProofArtifactDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleZKProofVerify expected not-found error")
	}
	if errResp.Error.Code != "broker_not_found_artifact" {
		t.Fatalf("error code = %q, want broker_not_found_artifact", errResp.Error.Code)
	}
}

func TestHandleZKProofGenerateFailsClosedWhenFixtureBackendDisabled(t *testing.T) {
	storeRoot := t.TempDir()
	ledgerRoot := t.TempDir()
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
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
