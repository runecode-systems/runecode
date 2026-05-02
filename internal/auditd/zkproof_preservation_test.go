package auditd

import (
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestProofBackfillEvidenceSnapshotIncludesProofAndAuditSidecars(t *testing.T) {
	ledger, receiptDigest, bindingDigest, bindingPayload, artifactDigest, verificationDigest := mustPrepareProofBackfillEvidenceSnapshotFixture(t)
	snapshot := mustProofBackfillEvidenceSnapshot(t, ledger)
	assertBaseProofBackfillEvidenceSnapshot(t, snapshot)
	assertProofBackfillBindingEvidenceSnapshot(t, snapshot, receiptDigest, bindingDigest, bindingPayload)
	assertStringSliceContains(t, snapshot.ZKProofArtifactDigests, mustDigestIdentity(artifactDigest))
	assertStringSliceContains(t, snapshot.ZKProofVerificationDigests, mustDigestIdentity(verificationDigest))
}

func mustPrepareProofBackfillEvidenceSnapshotFixture(t *testing.T) (*Ledger, trustpolicy.Digest, trustpolicy.Digest, trustpolicy.AuditProofBindingPayload, trustpolicy.Digest, trustpolicy.Digest) {
	t.Helper()
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReport(t, ledger, validReportFixture("segment-000001"))
	receiptDigest := mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, sealResult.SealEnvelopeDigest))
	recordDigest := mustRecordDigestForTest(t, ledger)
	bindingDigest, bindingPayload := mustPersistAuditProofBindingFixture(t, ledger, recordDigest, sealResult.SealEnvelopeDigest)
	artifactDigest, verificationDigest := mustPersistZKProofFixture(t, ledger, bindingDigest)
	return ledger, receiptDigest, bindingDigest, bindingPayload, artifactDigest, verificationDigest
}

func mustPersistAuditProofBindingFixture(t *testing.T, ledger *Ledger, recordDigest, sealDigest trustpolicy.Digest) (trustpolicy.Digest, trustpolicy.AuditProofBindingPayload) {
	t.Helper()
	bindingDigest, _, err := ledger.PersistAuditProofBinding(validAuditProofBindingPayloadFixture(recordDigest, sealDigest))
	if err != nil {
		t.Fatalf("PersistAuditProofBinding returned error: %v", err)
	}
	bindingPayload, found, err := ledger.AuditProofBindingByDigest(bindingDigest)
	if err != nil {
		t.Fatalf("AuditProofBindingByDigest returned error: %v", err)
	}
	if !found {
		t.Fatal("AuditProofBindingByDigest returned !found")
	}
	return bindingDigest, bindingPayload
}

func mustPersistZKProofFixture(t *testing.T, ledger *Ledger, bindingDigest trustpolicy.Digest) (trustpolicy.Digest, trustpolicy.Digest) {
	t.Helper()
	artifact := validZKProofArtifactPayloadFixture(bindingDigest)
	artifactDigest, err := ledger.PersistZKProofArtifact(artifact)
	if err != nil {
		t.Fatalf("PersistZKProofArtifact returned error: %v", err)
	}
	verificationDigest, err := ledger.PersistZKProofVerificationRecord(validZKProofVerificationRecordPayloadFixture(artifactDigest, artifact))
	if err != nil {
		t.Fatalf("PersistZKProofVerificationRecord returned error: %v", err)
	}
	return artifactDigest, verificationDigest
}

func mustProofBackfillEvidenceSnapshot(t *testing.T, ledger *Ledger) ProofBackfillEvidenceSnapshot {
	t.Helper()
	snapshot, err := ledger.ProofBackfillEvidenceSnapshot()
	if err != nil {
		t.Fatalf("ProofBackfillEvidenceSnapshot returned error: %v", err)
	}
	return snapshot
}

func assertBaseProofBackfillEvidenceSnapshot(t *testing.T, snapshot ProofBackfillEvidenceSnapshot) {
	t.Helper()
	if len(snapshot.SegmentIDs) == 0 {
		t.Fatal("SegmentIDs empty")
	}
	if len(snapshot.VerifierRecordDigests) == 0 {
		t.Fatal("VerifierRecordDigests empty")
	}
	if len(snapshot.EventContractCatalogDigests) == 0 {
		t.Fatal("EventContractCatalogDigests empty")
	}
}

func assertProofBackfillBindingEvidenceSnapshot(t *testing.T, snapshot ProofBackfillEvidenceSnapshot, receiptDigest, bindingDigest trustpolicy.Digest, bindingPayload trustpolicy.AuditProofBindingPayload) {
	t.Helper()
	assertStringSliceContains(t, snapshot.ProtocolBundleManifestHashes, mustDigestIdentity(bindingPayload.ProtocolBundleManifest))
	assertStringSliceContains(t, snapshot.RuntimeImageDescriptorDigests, bindingPayload.ProjectedPublicBindings.RuntimeImageDescriptorDigest)
	assertStringSliceContains(t, snapshot.AttestationEvidenceDigests, bindingPayload.ProjectedPublicBindings.AttestationEvidenceDigest)
	assertStringSliceContains(t, snapshot.AppliedHardeningPostureDigests, bindingPayload.ProjectedPublicBindings.AppliedHardeningPostureDigest)
	assertStringSliceContains(t, snapshot.SessionBindingDigests, bindingPayload.ProjectedPublicBindings.SessionBindingDigest)
	assertStringSliceContains(t, snapshot.ProjectSubstrateSnapshotDigests, bindingPayload.ProjectedPublicBindings.ProjectSubstrateSnapshotDigest)
	assertStringSliceContains(t, snapshot.AuditReceiptDigests, mustDigestIdentity(receiptDigest))
	assertStringSliceContains(t, snapshot.AuditProofBindingDigests, mustDigestIdentity(bindingDigest))
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
	assertStringSliceContains(t, snapshot.TypedRequestHashes, "sha256:"+strings.Repeat("d", 64))
	assertStringSliceContains(t, snapshot.ActionRequestHashes, "sha256:"+strings.Repeat("e", 64))
	assertStringSliceContains(t, snapshot.PolicyDecisionHashes, "sha256:"+strings.Repeat("f", 64))
	assertStringSliceContains(t, snapshot.RequiredApprovalIDs, "approval-1")
	assertStringSliceContains(t, snapshot.ApprovalRequestHashes, "sha256:"+strings.Repeat("1", 64))
	assertStringSliceContains(t, snapshot.ApprovalDecisionHashes, "sha256:"+strings.Repeat("2", 64))
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
	evidenceDigest, _, err := ledger.PersistExternalAnchorEvidence(ExternalAnchorEvidenceRequest{RecordedAtRFC3339: time.Now().UTC().Format(time.RFC3339), RunID: "run-1", PreparedMutationID: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ExecutionAttemptID: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", CanonicalTargetKind: "transparency_log", CanonicalTargetDigest: targetDigest, CanonicalTargetIdentity: targetIdentity, TargetRequirement: trustpolicy.ExternalAnchorTargetRequirementOptional, AnchoringSubjectFamily: trustpolicy.AuditSegmentAnchoringSubjectSeal, AnchoringSubjectDigest: sealDigest, OutboundPayloadDigest: &outbound, Outcome: trustpolicy.ExternalAnchorOutcomeDeferred, OutcomeReasonCode: "external_anchor_execution_deferred", TypedRequestHash: digestPtr("d"), ActionRequestHash: digestPtr("e"), PolicyDecisionHash: digestPtr("f"), RequiredApprovalID: "approval-1", ApprovalRequestHash: digestPtr("1"), ApprovalDecisionHash: digestPtr("2"), ProofDigest: proofDigest, ProofSchemaID: "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0", ProofKind: "transparency_log_receipt_v0"})
	if err != nil {
		t.Fatalf("PersistExternalAnchorEvidence returned error: %v", err)
	}
	return evidenceDigest
}

func digestPtr(ch string) *trustpolicy.Digest {
	return &trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat(ch, 64)}
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
