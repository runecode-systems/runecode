package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
)

func TestRuntimeSessionBindingRelationshipVerifierChecksAllNormalizedFields(t *testing.T) {
	fixture, err := buildSeedRuntimeEvidenceFixture("session-1")
	if err != nil {
		t.Fatalf("buildSeedRuntimeEvidenceFixture returned error: %v", err)
	}
	normalized, err := zkproof.CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0(zkproof.CompileAuditIsolateSessionBoundAttestedRuntimeInput{
		DeterministicVerification: true,
		VerifiedAuditEvent:        validAuditEventFixtureForVerifierTest(t, fixture.evidence),
		VerifiedAuditRecordDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("9", 64)},
		VerifiedAuditSegmentSeal: trustpolicy.AuditSegmentSealPayload{
			SchemaID:      trustpolicy.AuditSegmentSealSchemaID,
			SchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion,
			MerkleProfile: trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1,
			MerkleRoot:    mustMerkleRootForVerifierTest(t),
		},
		VerifiedAuditSegmentSealDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)},
		MerkleAuthenticationPath:       mustMerklePathForVerifierTest(t),
		BindingCommitmentDeriver:       zkproof.NewPoseidonBindingCommitmentDeriverV0(),
		SessionBindingRelationshipVerify: runtimeSessionBindingRelationshipVerifier{
			evidence: fixture.evidence,
		},
		NormalizationProfileID: zkproof.NormalizationProfileAuditIsolateSessionBoundAttestedRuntimeV0,
		SchemeAdapterID:        zkproof.SchemeAdapterGnarkGroth16IsolateSessionBoundV0,
	})
	if err != nil {
		t.Fatalf("CompileAudit... returned error: %v", err)
	}
	verifier := runtimeSessionBindingRelationshipVerifier{evidence: fixture.evidence}
	if err := verifier.VerifyNormalizedPrivateRemainderSessionBinding(normalized.WitnessInputs.PrivateRemainder, fixture.evidence.Session.EvidenceDigest); err != nil {
		t.Fatalf("VerifyNormalizedPrivateRemainderSessionBinding returned error: %v", err)
	}
	evidence := fixture.evidence
	evidence.Session = &launcherbackend.SessionRuntimeEvidence{}
	if err := (runtimeSessionBindingRelationshipVerifier{evidence: evidence}).VerifyNormalizedPrivateRemainderSessionBinding(normalized.WitnessInputs.PrivateRemainder, fixture.evidence.Session.EvidenceDigest); err == nil {
		t.Fatal("expected mismatch when authoritative session evidence fields drift")
	}
}

func TestFindCachedVerificationResponseReturnsBeforeCryptographicVerification(t *testing.T) {
	service, recordDigest := mustSetupZKProofE2EService(t)
	outcome := runEvaluationOnlyZKProofHarnessV0(t, service, recordDigest)
	artifact, found, err := service.auditLedger.ZKProofArtifactByDigest(outcome.ProofArtifactDigest)
	if err != nil {
		t.Fatalf("ZKProofArtifactByDigest returned error: %v", err)
	}
	if !found {
		t.Fatal("ZKProofArtifactByDigest found=false, want true")
	}
	publicInputsDigest, err := verifyArtifactPublicInputsDigest(artifact, artifact.PublicInputsDigest)
	if err != nil {
		t.Fatalf("verifyArtifactPublicInputsDigest returned error: %v", err)
	}
	if err := service.validateArtifactBindingAndAuthoritativeEvidence(artifact, publicInputsDigest); err != nil {
		t.Fatalf("validateArtifactBindingAndAuthoritativeEvidence returned error: %v", err)
	}
	base, err := service.buildVerificationRecordBase(outcome.ProofArtifactDigest, artifact, publicInputsDigest)
	if err != nil {
		t.Fatalf("buildVerificationRecordBase returned error: %v", err)
	}
	cached, found, err := service.findCachedVerificationResponse("req-zk-cache", outcome.ProofArtifactDigest, base)
	if err != nil {
		t.Fatalf("findCachedVerificationResponse returned error: %v", err)
	}
	if !found {
		t.Fatal("findCachedVerificationResponse found=false, want true")
	}
	if cached.CacheProvenance != "cache_hit" {
		t.Fatalf("cache_provenance = %q, want cache_hit", cached.CacheProvenance)
	}
	if _, err := service.finalizeVerificationRecord(base, artifact, trustpolicy.Digest{}); err == nil {
		t.Fatal("expected finalizeVerificationRecord to require valid public inputs digest")
	}
}

func TestValidateArtifactBindingAndAuthoritativeEvidenceRejectsBindingDrift(t *testing.T) {
	service, recordDigest := mustSetupZKProofE2EService(t)
	outcome := runEvaluationOnlyZKProofHarnessV0(t, service, recordDigest)
	artifact, found, err := service.auditLedger.ZKProofArtifactByDigest(outcome.ProofArtifactDigest)
	if err != nil {
		t.Fatalf("ZKProofArtifactByDigest returned error: %v", err)
	}
	if !found {
		t.Fatal("ZKProofArtifactByDigest found=false, want true")
	}
	artifact.PublicInputs["session_binding_digest"] = fmt.Sprintf("sha256:%s", strings.Repeat("1", 64))
	publicInputsDigest, err := canonicalMapDigest(artifact.PublicInputs)
	if err != nil {
		t.Fatalf("canonicalMapDigest returned error: %v", err)
	}
	err = service.validateArtifactBindingAndAuthoritativeEvidence(artifact, publicInputsDigest)
	if err == nil {
		t.Fatal("expected binding drift rejection, got nil")
	}
}

func validAuditEventFixtureForVerifierTest(t *testing.T, evidence launcherbackend.RuntimeEvidenceSnapshot) trustpolicy.AuditEventPayload {
	t.Helper()
	payload := trustpolicy.IsolateSessionBoundPayload{
		SchemaID:                      trustpolicy.IsolateSessionBoundPayloadSchemaID,
		SchemaVersion:                 trustpolicy.IsolateSessionBoundPayloadSchemaVersion,
		RunID:                         "run-1",
		IsolateID:                     "isolate-1",
		SessionID:                     "session-1",
		BackendKind:                   "microvm",
		IsolationAssuranceLevel:       "isolated",
		ProvisioningPosture:           "tofu",
		LaunchContextDigest:           evidence.Session.LaunchContextDigest,
		HandshakeTranscriptHash:       evidence.Session.HandshakeTranscriptHash,
		SessionBindingDigest:          evidence.Session.EvidenceDigest,
		RuntimeImageDescriptorDigest:  evidence.Launch.RuntimeImageDescriptorDigest,
		AppliedHardeningPostureDigest: evidence.Hardening.EvidenceDigest,
		AttestationEvidenceDigest:     evidence.Attestation.EvidenceDigest,
	}
	bytes, err := jsonMarshalForVerifierTest(payload)
	if err != nil {
		t.Fatalf("jsonMarshalForVerifierTest payload: %v", err)
	}
	return trustpolicy.AuditEventPayload{
		SchemaID:                   trustpolicy.AuditEventSchemaID,
		SchemaVersion:              trustpolicy.AuditEventSchemaVersion,
		AuditEventType:             "isolate_session_bound",
		EmitterStreamID:            "stream-1",
		Seq:                        1,
		OccurredAt:                 "2026-03-13T12:20:00Z",
		EventPayloadSchemaID:       trustpolicy.IsolateSessionBoundPayloadSchemaID,
		EventPayload:               bytes,
		EventPayloadHash:           trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("7", 64)},
		ProtocolBundleManifestHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("8", 64)},
	}
}

func mustMerkleRootForVerifierTest(t *testing.T) trustpolicy.Digest {
	t.Helper()
	root, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot([]trustpolicy.Digest{{HashAlg: "sha256", Hash: strings.Repeat("9", 64)}, {HashAlg: "sha256", Hash: strings.Repeat("f", 64)}})
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot: %v", err)
	}
	return root
}

func mustMerklePathForVerifierTest(t *testing.T) zkproof.MerkleAuthenticationPath {
	t.Helper()
	path, err := zkproof.DeriveAuditSegmentMerkleAuthenticationPathV0([]trustpolicy.Digest{{HashAlg: "sha256", Hash: strings.Repeat("9", 64)}, {HashAlg: "sha256", Hash: strings.Repeat("f", 64)}}, 0)
	if err != nil {
		t.Fatalf("DeriveAuditSegmentMerkleAuthenticationPathV0: %v", err)
	}
	return path
}

func jsonMarshalForVerifierTest(v any) ([]byte, error) {
	return json.Marshal(v)
}
