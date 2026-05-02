package brokerapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
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

func TestEvaluationOnlyZKProofHarnessEndToEndWithoutAuthoritativeCommands(t *testing.T) {
	service, recordDigest := mustSetupZKProofE2EService(t)
	outcome := runEvaluationOnlyZKProofHarnessV0(t, service, recordDigest)
	if outcome.VerificationRecord.VerificationOutcome != trustpolicy.ProofVerificationOutcomeVerified {
		t.Fatalf("verification_outcome = %q, want %q", outcome.VerificationRecord.VerificationOutcome, trustpolicy.ProofVerificationOutcomeVerified)
	}
	if len(outcome.VerificationRecord.ReasonCodes) != 1 || outcome.VerificationRecord.ReasonCodes[0] != trustpolicy.ProofVerificationReasonVerified {
		t.Fatalf("reason_codes = %v, want [%q]", outcome.VerificationRecord.ReasonCodes, trustpolicy.ProofVerificationReasonVerified)
	}
	if outcome.CachedVerifyResponse.CacheProvenance != "cache_hit" {
		t.Fatalf("cache_provenance = %q, want cache_hit", outcome.CachedVerifyResponse.CacheProvenance)
	}
}

func TestEvaluationOnlyZKProofHarnessUsesPersistedBindingAndAuthoritativeEvidence(t *testing.T) {
	service, recordDigest := mustSetupZKProofE2EService(t)
	outcome := runEvaluationOnlyZKProofHarnessV0(t, service, recordDigest)
	bindingPayload, found, err := service.auditLedger.AuditProofBindingByDigest(outcome.AuditProofBindingDigest)
	if err != nil {
		t.Fatalf("AuditProofBindingByDigest returned error: %v", err)
	}
	if !found {
		t.Fatal("AuditProofBindingByDigest found=false, want true")
	}
	if bindingPayload.ProjectedPublicBindings.AttestationVerificationRecord == nil {
		t.Fatal("binding missing attestation_verification_record_digest")
	}
}

func mustSetupZKProofE2EService(t testing.TB) (*Service, trustpolicy.Digest) {
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

type evaluationHarnessOutcome struct {
	ProofArtifactDigest     trustpolicy.Digest
	VerificationDigest      trustpolicy.Digest
	VerificationRecord      trustpolicy.ZKProofVerificationRecordPayload
	CachedVerifyResponse    ZKProofVerifyResponse
	AuditProofBindingDigest trustpolicy.Digest
}

func runEvaluationOnlyZKProofHarnessV0(t testing.TB, service *Service, recordDigest trustpolicy.Digest) evaluationHarnessOutcome {
	t.Helper()
	artifactDigest, bindingDigest, record := mustRunEvaluationOnlyZKProofHarnessFresh(t, service, recordDigest)
	cached, found, err := service.findCachedVerificationResponse("req-zk-eval", artifactDigest, record)
	if err != nil {
		t.Fatalf("findCachedVerificationResponse returned error: %v", err)
	}
	if !found {
		t.Fatal("findCachedVerificationResponse found=false, want true")
	}
	verificationDigest, err := mustPersistedVerificationDigestForRecord(t, service, record)
	if err != nil {
		t.Fatalf("mustPersistedVerificationDigestForRecord returned error: %v", err)
	}
	return evaluationHarnessOutcome{ProofArtifactDigest: artifactDigest, VerificationDigest: verificationDigest, VerificationRecord: record, CachedVerifyResponse: cached, AuditProofBindingDigest: bindingDigest}
}

func mustRunEvaluationOnlyZKProofHarnessFresh(t testing.TB, service *Service, recordDigest trustpolicy.Digest) (trustpolicy.Digest, trustpolicy.Digest, trustpolicy.ZKProofVerificationRecordPayload) {
	t.Helper()
	artifact, artifactDigest, bindingDigest, backend, trusted, err := mustBuildEvaluationOnlyProofArtifact(t, service, recordDigest)
	if err != nil {
		t.Fatal(err)
	}
	mustVerifyEvaluationOnlyProofArtifact(t, service, artifact, backend, trusted)
	record := buildEvaluationOnlyVerificationRecord(artifact, artifactDigest)
	verificationDigest, err := service.auditLedger.PersistZKProofVerificationRecord(record)
	if err != nil {
		t.Fatalf("PersistZKProofVerificationRecord returned error: %v", err)
	}
	_ = verificationDigest
	return artifactDigest, bindingDigest, record
}

func mustBuildEvaluationOnlyProofArtifact(t testing.TB, service *Service, recordDigest trustpolicy.Digest) (trustpolicy.ZKProofArtifactPayload, trustpolicy.Digest, trustpolicy.Digest, zkproof.ProofProver, zkproof.TrustedVerifierPosture, error) {
	t.Helper()
	_, inclusion, err := service.loadAuditRecordInclusion(recordDigest)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, zkproof.TrustedVerifierPosture{}, fmt.Errorf("loadAuditRecordInclusion returned error: %w", err)
	}
	compiled, path, runtimeEvidence, err := service.compileZKProofInput(inclusion)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, zkproof.TrustedVerifierPosture{}, fmt.Errorf("compileZKProofInput returned error: %w", err)
	}
	bindingDigest, err := service.persistCompiledAuditProofBinding(compiled, path, inclusion, runtimeEvidence)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, zkproof.TrustedVerifierPosture{}, fmt.Errorf("persistCompiledAuditProofBinding returned error: %w", err)
	}
	backend, frozen, _, trusted, err := zkproof.NewEvaluationOnlyGroth16BackendForBenchmarkV0()
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, zkproof.TrustedVerifierPosture{}, fmt.Errorf("NewEvaluationOnlyGroth16BackendForBenchmarkV0 returned error: %w", err)
	}
	artifact, err := buildZKProofArtifact(compiled, bindingDigest, backend, frozen, trusted)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, zkproof.TrustedVerifierPosture{}, fmt.Errorf("buildZKProofArtifact returned error: %w", err)
	}
	artifactDigest, err := service.auditLedger.PersistZKProofArtifact(artifact)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, zkproof.TrustedVerifierPosture{}, fmt.Errorf("PersistZKProofArtifact returned error: %w", err)
	}
	return artifact, artifactDigest, bindingDigest, backend, trusted, nil
}

func mustVerifyEvaluationOnlyProofArtifact(t testing.TB, service *Service, artifact trustpolicy.ZKProofArtifactPayload, backend zkproof.ProofProver, trusted zkproof.TrustedVerifierPosture) {
	t.Helper()
	publicInputsDigest, err := verifyArtifactPublicInputsDigest(artifact, artifact.PublicInputsDigest)
	if err != nil {
		t.Fatalf("verifyArtifactPublicInputsDigest returned error: %v", err)
	}
	if err := service.validateArtifactBindingAndAuthoritativeEvidence(artifact, publicInputsDigest); err != nil {
		t.Fatalf("validateArtifactBindingAndAuthoritativeEvidence returned error: %v", err)
	}
	proofBytes, publicInputs, identity, err := decodeProofVerificationInputs(artifact, publicInputsDigest)
	if err != nil {
		t.Fatalf("decodeProofVerificationInputs returned error: %v", err)
	}
	if err := zkproof.VerifyProofWithTrustedPostureV0(backend, proofBytes, publicInputs, identity, trusted); err != nil {
		t.Fatalf("VerifyProofWithTrustedPostureV0 returned error: %v", err)
	}
}

func buildEvaluationOnlyVerificationRecord(artifact trustpolicy.ZKProofArtifactPayload, artifactDigest trustpolicy.Digest) trustpolicy.ZKProofVerificationRecordPayload {
	return trustpolicy.ZKProofVerificationRecordPayload{
		SchemaID:                 trustpolicy.ZKProofVerificationRecordSchemaID,
		SchemaVersion:            trustpolicy.ZKProofVerificationRecordSchemaVersion,
		ProofDigest:              artifactDigest,
		StatementFamily:          artifact.StatementFamily,
		StatementVersion:         artifact.StatementVersion,
		SchemeID:                 artifact.SchemeID,
		CurveID:                  artifact.CurveID,
		CircuitID:                artifact.CircuitID,
		ConstraintSystemDigest:   artifact.ConstraintSystemDigest,
		VerifierKeyDigest:        artifact.VerifierKeyDigest,
		SetupProvenanceDigest:    artifact.SetupProvenanceDigest,
		NormalizationProfileID:   artifact.NormalizationProfileID,
		SchemeAdapterID:          artifact.SchemeAdapterID,
		PublicInputsDigest:       artifact.PublicInputsDigest,
		VerifierImplementationID: "runecode.evaluation.zk.verifier.gnark.v0",
		VerifiedAt:               time.Now().UTC().Format(time.RFC3339),
		VerificationOutcome:      trustpolicy.ProofVerificationOutcomeVerified,
		ReasonCodes:              []string{trustpolicy.ProofVerificationReasonVerified},
		CacheProvenance:          "fresh",
	}
}

func mustPersistedVerificationDigestForRecord(t testing.TB, service *Service, record trustpolicy.ZKProofVerificationRecordPayload) (trustpolicy.Digest, error) {
	t.Helper()
	verificationDigest, _, found, err := service.auditLedger.FindMatchingZKProofVerificationRecord(record)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	if !found {
		return trustpolicy.Digest{}, fmt.Errorf("matching verification record not found")
	}
	return verificationDigest, nil
}
