package auditd

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestBuildEvidenceRetentionReviewSeededFixtureIsFullySatisfied(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seal := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, seal.SealEnvelopeDigest))
	_ = mustPersistReport(t, ledger, validReportFixture("segment-000001"))

	_, _, review, err := ledger.BuildEvidenceRetentionReview(AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"})
	if err != nil {
		t.Fatalf("BuildEvidenceRetentionReview returned error: %v", err)
	}
	if !review.FullySatisfied {
		t.Fatalf("completeness review = %+v, want fully satisfied for seeded complete fixture", review)
	}
	if len(review.Missing) != 0 {
		t.Fatalf("missing = %+v, want no completeness gaps for seeded complete fixture", review.Missing)
	}
}

func TestEvaluateEvidenceRetentionCompletenessReportsRuntimeEvidenceGapWhenObjectMissing(t *testing.T) {
	runtime := "sha256:" + strings.Repeat("b", 64)
	review := EvaluateEvidenceRetentionCompleteness(
		AuditEvidenceSnapshot{RuntimeEvidenceDigests: []string{runtime}},
		AuditEvidenceBundleManifest{},
	)
	if review.FullySatisfied {
		t.Fatalf("review = %+v, want incomplete when runtime evidence digest has no included object", review)
	}
	if !containsCompletenessFamily(review.Missing, "runtime_evidence_digest") {
		t.Fatalf("missing = %+v, want runtime_evidence_digest completeness gap", review.Missing)
	}
}

func TestEvaluateEvidenceRetentionCompletenessTreatsDeclaredRedactionsSeparately(t *testing.T) {
	receipt := "sha256:" + strings.Repeat("a", 64)
	manifest := AuditEvidenceBundleManifest{
		Redactions: []AuditEvidenceBundleRedaction{{Path: "sidecar/receipts/*", ReasonCode: "profile_digest_metadata_default"}},
	}
	snapshot := AuditEvidenceSnapshot{AuditReceiptDigests: []string{receipt}}
	review := EvaluateEvidenceRetentionCompleteness(snapshot, manifest)
	if review.FullySatisfied {
		t.Fatalf("review = %+v, want not fully satisfied when required identities are redacted", review)
	}
	if containsCompletenessFamily(review.Missing, "audit_receipt_digest") {
		t.Fatalf("missing = %+v, want redacted identities tracked separately", review.Missing)
	}
	if !containsCompletenessFamily(review.DeclaredRedactions, "audit_receipt_digest") {
		t.Fatalf("declared_redactions = %+v, want audit_receipt_digest", review.DeclaredRedactions)
	}
}

func TestEvaluateEvidenceRetentionCompletenessManifestDeclaresButDoesNotSubstituteEvidence(t *testing.T) {
	receipt := "sha256:" + strings.Repeat("a", 64)
	manifest := AuditEvidenceBundleManifest{
		RootDigests: []string{receipt},
		SealReferences: []AuditEvidenceBundleSealReference{
			{SegmentID: "segment-000001", SealDigest: receipt, SealChainIndex: 0},
		},
		IncludedObjects: nil,
	}
	snapshot := AuditEvidenceSnapshot{AuditReceiptDigests: []string{receipt}}
	review := EvaluateEvidenceRetentionCompleteness(snapshot, manifest)
	if review.FullySatisfied {
		t.Fatalf("review = %+v, want incomplete when receipt evidence object is missing", review)
	}
	if !containsCompletenessFamily(review.Missing, "audit_receipt_digest") {
		t.Fatalf("missing = %+v, want audit_receipt_digest even when manifest contains root/seal digests", review.Missing)
	}
}

func containsCompletenessFamily(entries []AuditEvidenceSnapshotCompleteness, family string) bool {
	for i := range entries {
		if entries[i].Family == family {
			return true
		}
	}
	return false
}

func persistExternalAnchorEvidenceForSealWithProjectContext(t *testing.T, ledger *Ledger, sealDigest trustpolicy.Digest, targetDigest trustpolicy.Digest, projectContextDigest trustpolicy.Digest) error {
	t.Helper()
	proofDigest, err := ledger.PersistExternalAnchorSidecar(trustpolicy.ExternalAnchorSidecarKindProofBytes, map[string]any{"schema_id": "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0", "schema_version": "0.1.0", "proof": "fixture"})
	if err != nil {
		return err
	}
	targetIdentity, err := targetDigest.Identity()
	if err != nil {
		return err
	}
	outbound := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("c", 64)}
	_, _, err = ledger.PersistExternalAnchorEvidence(ExternalAnchorEvidenceRequest{
		RunID:                   "run-1",
		PreparedMutationID:      "sha256:" + strings.Repeat("4", 64),
		ExecutionAttemptID:      "sha256:" + strings.Repeat("5", 64),
		CanonicalTargetKind:     "transparency_log",
		CanonicalTargetDigest:   targetDigest,
		CanonicalTargetIdentity: targetIdentity,
		TargetRequirement:       trustpolicy.ExternalAnchorTargetRequirementOptional,
		AnchoringSubjectFamily:  trustpolicy.AuditSegmentAnchoringSubjectSeal,
		AnchoringSubjectDigest:  sealDigest,
		OutboundPayloadDigest:   &outbound,
		OutboundBytes:           128,
		Outcome:                 trustpolicy.ExternalAnchorOutcomeDeferred,
		OutcomeReasonCode:       "external_anchor_execution_deferred",
		ProofDigest:             proofDigest,
		ProofSchemaID:           "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0",
		ProofKind:               "transparency_log_receipt_v0",
		ProjectContextIdentity:  &projectContextDigest,
	})
	return err
}
