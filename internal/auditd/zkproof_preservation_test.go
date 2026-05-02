package auditd

import (
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestProofBackfillEvidenceSnapshotIncludesProofAndAuditSidecars(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReport(t, ledger, validReportFixture("segment-000001"))
	receiptDigest := mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, sealResult.SealEnvelopeDigest))

	recordDigest := mustRecordDigestForTest(t, ledger)
	bindingDigest, _, err := ledger.PersistAuditProofBinding(validAuditProofBindingPayloadFixture(recordDigest, sealResult.SealEnvelopeDigest))
	if err != nil {
		t.Fatalf("PersistAuditProofBinding returned error: %v", err)
	}
	artifact := validZKProofArtifactPayloadFixture(bindingDigest)
	artifactDigest, err := ledger.PersistZKProofArtifact(artifact)
	if err != nil {
		t.Fatalf("PersistZKProofArtifact returned error: %v", err)
	}
	verificationDigest, err := ledger.PersistZKProofVerificationRecord(validZKProofVerificationRecordPayloadFixture(artifactDigest, artifact))
	if err != nil {
		t.Fatalf("PersistZKProofVerificationRecord returned error: %v", err)
	}

	snapshot, err := ledger.ProofBackfillEvidenceSnapshot()
	if err != nil {
		t.Fatalf("ProofBackfillEvidenceSnapshot returned error: %v", err)
	}
	if len(snapshot.SegmentIDs) == 0 {
		t.Fatal("SegmentIDs empty")
	}
	if len(snapshot.VerifierRecordDigests) == 0 {
		t.Fatal("VerifierRecordDigests empty")
	}
	if len(snapshot.EventContractCatalogDigests) == 0 {
		t.Fatal("EventContractCatalogDigests empty")
	}
	assertStringSliceContains(t, snapshot.AuditReceiptDigests, mustDigestIdentity(receiptDigest))
	assertStringSliceContains(t, snapshot.AuditProofBindingDigests, mustDigestIdentity(bindingDigest))
	assertStringSliceContains(t, snapshot.ZKProofArtifactDigests, mustDigestIdentity(artifactDigest))
	assertStringSliceContains(t, snapshot.ZKProofVerificationDigests, mustDigestIdentity(verificationDigest))
}

func TestProofBackfillEvidenceSnapshotIncludesExternalAnchorEvidenceClasses(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	proofDigest := mustPersistExternalAnchorProofSidecar(t, ledger)
	evidenceDigest := mustPersistExternalAnchorEvidenceForSnapshot(t, ledger, sealResult.SealEnvelopeDigest, proofDigest)
	snapshot, err := ledger.ProofBackfillEvidenceSnapshot()
	if err != nil {
		t.Fatalf("ProofBackfillEvidenceSnapshot returned error: %v", err)
	}
	assertStringSliceContains(t, snapshot.ExternalAnchorEvidenceDigests, mustDigestIdentity(evidenceDigest))
	assertStringSliceContains(t, snapshot.ExternalAnchorSidecarDigests, mustDigestIdentity(proofDigest))
}

func mustPersistExternalAnchorProofSidecar(t *testing.T, ledger *Ledger) trustpolicy.Digest {
	t.Helper()
	proofDigest, err := ledger.PersistExternalAnchorSidecar(trustpolicy.ExternalAnchorSidecarKindProofBytes, map[string]any{"schema_id": "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0", "schema_version": "0.1.0", "proof": "fixture"})
	if err != nil {
		t.Fatalf("PersistExternalAnchorSidecar returned error: %v", err)
	}
	return proofDigest
}

func mustPersistExternalAnchorEvidenceForSnapshot(t *testing.T, ledger *Ledger, sealDigest, proofDigest trustpolicy.Digest) trustpolicy.Digest {
	t.Helper()
	targetDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: "9999999999999999999999999999999999999999999999999999999999999999"}
	targetIdentity, _ := targetDigest.Identity()
	outbound := trustpolicy.Digest{HashAlg: "sha256", Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	evidenceDigest, _, err := ledger.PersistExternalAnchorEvidence(ExternalAnchorEvidenceRequest{RecordedAtRFC3339: time.Now().UTC().Format(time.RFC3339), RunID: "run-1", PreparedMutationID: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ExecutionAttemptID: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", CanonicalTargetKind: "transparency_log", CanonicalTargetDigest: targetDigest, CanonicalTargetIdentity: targetIdentity, TargetRequirement: trustpolicy.ExternalAnchorTargetRequirementOptional, AnchoringSubjectFamily: trustpolicy.AuditSegmentAnchoringSubjectSeal, AnchoringSubjectDigest: sealDigest, OutboundPayloadDigest: &outbound, Outcome: trustpolicy.ExternalAnchorOutcomeDeferred, OutcomeReasonCode: "external_anchor_execution_deferred", ProofDigest: proofDigest, ProofSchemaID: "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0", ProofKind: "transparency_log_receipt_v0"})
	if err != nil {
		t.Fatalf("PersistExternalAnchorEvidence returned error: %v", err)
	}
	return evidenceDigest
}

func assertStringSliceContains(t *testing.T, values []string, want string) {
	t.Helper()
	for i := range values {
		if values[i] == want {
			return
		}
	}
	t.Fatalf("slice %v missing %q", values, want)
}
