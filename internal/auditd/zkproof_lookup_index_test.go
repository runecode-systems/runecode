package auditd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestProofLookupIndexBuildAndRecovery(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	lookup := persistProofLookupIndexFixture(t, ledger, fixture)

	lookupPath := filepath.Join(root, indexDirName, proofLookupIndexFileName)
	assertPathPresent(t, lookupPath, "proof lookup index missing")

	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(reopen) returned error: %v", err)
	}
	assertReopenedLookupIndexState(t, reopened, lookup)
}

func persistProofLookupIndexFixture(t *testing.T, ledger *Ledger, fixture auditFixtureKey) proofLookupIndexFixture {
	t.Helper()
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	recordDigest := mustRecordDigestForTest(t, ledger)
	bindingPayload, bindingDigest := persistAndAssertAuditProofBinding(t, ledger, recordDigest, sealResult.SealEnvelopeDigest)
	artifactPayload, artifactDigest := persistAndAssertZKProofArtifact(t, ledger, bindingDigest)
	verificationPayload := validZKProofVerificationRecordPayloadFixture(artifactDigest, artifactPayload)
	verificationDigest, err := ledger.PersistZKProofVerificationRecord(verificationPayload)
	if err != nil {
		t.Fatalf("PersistZKProofVerificationRecord returned error: %v", err)
	}
	return proofLookupIndexFixture{sealResult: sealResult, recordDigest: recordDigest, bindingPayload: bindingPayload, bindingDigest: bindingDigest, verificationPayload: verificationPayload, verificationDigest: verificationDigest}
}

type proofLookupIndexFixture struct {
	sealResult          SealResult
	recordDigest        trustpolicy.Digest
	bindingPayload      trustpolicy.AuditProofBindingPayload
	bindingDigest       trustpolicy.Digest
	verificationPayload trustpolicy.ZKProofVerificationRecordPayload
	verificationDigest  trustpolicy.Digest
}

func assertReopenedLookupIndexState(t *testing.T, reopened *Ledger, fixture proofLookupIndexFixture) {
	t.Helper()
	segmentID, sealDigest, err := reopened.LatestAnchorableSeal()
	if err != nil {
		t.Fatalf("LatestAnchorableSeal returned error: %v", err)
	}
	if segmentID != fixture.sealResult.SegmentID {
		t.Fatalf("LatestAnchorableSeal segment=%q, want %q", segmentID, fixture.sealResult.SegmentID)
	}
	if mustDigestIdentity(sealDigest) != mustDigestIdentity(fixture.sealResult.SealEnvelopeDigest) {
		t.Fatalf("LatestAnchorableSeal digest=%q, want %q", mustDigestIdentity(sealDigest), mustDigestIdentity(fixture.sealResult.SealEnvelopeDigest))
	}
	latestDigest, _, found, err := reopened.LatestAuditProofBindingForRecord(fixture.recordDigest, fixture.bindingPayload.StatementFamily, fixture.bindingPayload.SchemeAdapterID)
	if err != nil {
		t.Fatalf("LatestAuditProofBindingForRecord returned error: %v", err)
	}
	if !found || mustDigestIdentity(latestDigest) != mustDigestIdentity(fixture.bindingDigest) {
		t.Fatalf("latest binding digest=%q, want %q", mustDigestIdentity(latestDigest), mustDigestIdentity(fixture.bindingDigest))
	}
	foundDigest, _, found, err := reopened.FindMatchingZKProofVerificationRecord(fixture.verificationPayload)
	if err != nil {
		t.Fatalf("FindMatchingZKProofVerificationRecord returned error: %v", err)
	}
	if !found || mustDigestIdentity(foundDigest) != mustDigestIdentity(fixture.verificationDigest) {
		t.Fatalf("verification digest=%q, want %q", mustDigestIdentity(foundDigest), mustDigestIdentity(fixture.verificationDigest))
	}
}

func TestProofLookupIndexFindsPreviousSealDigestByChainIndex(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	firstSeal := mustSealFixtureSegment(t, ledger, fixture)
	secondSeal := appendAndSealSecondFixtureSegment(t, ledger, fixture, firstSeal)

	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	currentEnvelope, _, currentPayload, err := ledger.loadSealEnvelopeForSegmentLocked(secondSeal.SegmentID)
	if err != nil {
		t.Fatalf("loadSealEnvelopeForSegmentLocked(second) returned error: %v", err)
	}
	if currentPayload.SealChainIndex != 1 {
		t.Fatalf("second seal chain index=%d, want 1", currentPayload.SealChainIndex)
	}
	previousDigest, err := ledger.previousSealDigestByIndexLocked(currentPayload.SealChainIndex - 1)
	if err != nil {
		t.Fatalf("previousSealDigestByIndexLocked returned error: %v", err)
	}
	if previousDigest == nil || mustDigestIdentity(*previousDigest) != mustDigestIdentity(firstSeal.SealEnvelopeDigest) {
		t.Fatalf("previous digest=%v, want %q", previousDigest, mustDigestIdentity(firstSeal.SealEnvelopeDigest))
	}
	computed, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(currentEnvelope)
	if err != nil {
		t.Fatalf("ComputeSignedEnvelopeAuditRecordDigest returned error: %v", err)
	}
	if mustDigestIdentity(computed) != mustDigestIdentity(secondSeal.SealEnvelopeDigest) {
		t.Fatalf("second seal digest mismatch: computed=%q want=%q", mustDigestIdentity(computed), mustDigestIdentity(secondSeal.SealEnvelopeDigest))
	}
}

func appendAndSealSecondFixtureSegment(t *testing.T, ledger *Ledger, fixture auditFixtureKey, firstSeal SealResult) SealResult {
	t.Helper()
	request := validAdmissionRequestForLedger(t, fixture)
	if _, err := ledger.AppendAdmittedEvent(request); err != nil {
		t.Fatalf("AppendAdmittedEvent(second segment) returned error: %v", err)
	}
	segment, err := ledger.loadSegment(firstSeal.NextOpenSegmentID)
	if err != nil {
		t.Fatalf("loadSegment(next open) returned error: %v", err)
	}
	secondSealEnvelope := buildSealEnvelopeForSegment(t, fixture, ledger, segment, &firstSeal.SealEnvelopeDigest, 1)
	secondSeal, err := ledger.SealCurrentSegment(secondSealEnvelope)
	if err != nil {
		t.Fatalf("SealCurrentSegment(second) returned error: %v", err)
	}
	return secondSeal
}

func TestVerificationIdentityKeyNormalizesReasonCodes(t *testing.T) {
	digest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)}
	left := trustpolicy.ZKProofVerificationRecordPayload{
		ProofDigest:              digest,
		StatementFamily:          "x",
		StatementVersion:         "v0",
		SchemeID:                 "groth16",
		CurveID:                  "bn254",
		CircuitID:                "c",
		ConstraintSystemDigest:   digest,
		VerifierKeyDigest:        digest,
		SetupProvenanceDigest:    digest,
		NormalizationProfileID:   "n",
		SchemeAdapterID:          "a",
		PublicInputsDigest:       digest,
		VerifierImplementationID: "impl",
		VerificationOutcome:      trustpolicy.ProofVerificationOutcomeVerified,
		ReasonCodes:              []string{"verified", "proof_invalid"},
	}
	right := left
	right.ReasonCodes = []string{"proof_invalid", "verified"}
	leftKey, err := verificationIdentityKey(left)
	if err != nil {
		t.Fatalf("verificationIdentityKey(left) returned error: %v", err)
	}
	rightKey, err := verificationIdentityKey(right)
	if err != nil {
		t.Fatalf("verificationIdentityKey(right) returned error: %v", err)
	}
	if leftKey != rightKey {
		t.Fatalf("verification identity keys differ for equal reason code set")
	}
}
