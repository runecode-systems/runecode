package auditd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestAuditRecordInclusionAndProofLookupHelpers(t *testing.T) {
	_, ledger, _, recordDigest, sealResult := setupZKProofLookupFixture(t)
	assertAuditRecordInclusion(t, ledger, recordDigest)
	bindingPayload, bindingDigest := persistAndAssertAuditProofBinding(t, ledger, recordDigest, sealResult.SealEnvelopeDigest)
	artifactPayload, artifactDigest := persistAndAssertZKProofArtifact(t, ledger, bindingDigest)
	assertLatestAuditProofBinding(t, ledger, recordDigest, bindingPayload, bindingDigest)
	assertMatchingVerificationRecord(t, ledger, artifactDigest, artifactPayload)
}

func setupZKProofLookupFixture(t *testing.T) (string, *Ledger, auditFixtureKey, trustpolicy.Digest, SealResult) {
	t.Helper()
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	recordDigest := mustRecordDigestForTest(t, ledger)
	return root, ledger, fixture, recordDigest, sealResult
}

func assertAuditRecordInclusion(t *testing.T, ledger *Ledger, recordDigest trustpolicy.Digest) {
	t.Helper()
	inclusion, found, err := ledger.AuditRecordInclusion(mustDigestIdentity(recordDigest))
	if err != nil {
		t.Fatalf("AuditRecordInclusion returned error: %v", err)
	}
	if !found {
		t.Fatal("AuditRecordInclusion found=false, want true")
	}
	if err := inclusion.Validate(); err != nil {
		t.Fatalf("inclusion.Validate returned error: %v", err)
	}
}

func persistAndAssertAuditProofBinding(t *testing.T, ledger *Ledger, recordDigest, sealDigest trustpolicy.Digest) (trustpolicy.AuditProofBindingPayload, trustpolicy.Digest) {
	t.Helper()
	bindingPayload := validAuditProofBindingPayloadFixture(recordDigest, sealDigest)
	bindingDigest, _, err := ledger.PersistAuditProofBinding(bindingPayload)
	if err != nil {
		t.Fatalf("PersistAuditProofBinding returned error: %v", err)
	}
	loadedBinding, found, err := ledger.AuditProofBindingByDigest(bindingDigest)
	if err != nil {
		t.Fatalf("AuditProofBindingByDigest returned error: %v", err)
	}
	if !found {
		t.Fatal("AuditProofBindingByDigest found=false, want true")
	}
	if loadedBinding.StatementFamily != bindingPayload.StatementFamily {
		t.Fatalf("binding statement_family = %q, want %q", loadedBinding.StatementFamily, bindingPayload.StatementFamily)
	}
	return bindingPayload, bindingDigest
}

func persistAndAssertZKProofArtifact(t *testing.T, ledger *Ledger, bindingDigest trustpolicy.Digest) (trustpolicy.ZKProofArtifactPayload, trustpolicy.Digest) {
	t.Helper()
	artifactPayload := validZKProofArtifactPayloadFixture(bindingDigest)
	artifactDigest, err := ledger.PersistZKProofArtifact(artifactPayload)
	if err != nil {
		t.Fatalf("PersistZKProofArtifact returned error: %v", err)
	}
	loadedArtifact, found, err := ledger.ZKProofArtifactByDigest(artifactDigest)
	if err != nil {
		t.Fatalf("ZKProofArtifactByDigest returned error: %v", err)
	}
	if !found {
		t.Fatal("ZKProofArtifactByDigest found=false, want true")
	}
	if loadedArtifact.CircuitID != artifactPayload.CircuitID {
		t.Fatalf("artifact circuit_id = %q, want %q", loadedArtifact.CircuitID, artifactPayload.CircuitID)
	}
	return artifactPayload, artifactDigest
}

func assertLatestAuditProofBinding(t *testing.T, ledger *Ledger, recordDigest trustpolicy.Digest, bindingPayload trustpolicy.AuditProofBindingPayload, bindingDigest trustpolicy.Digest) {
	t.Helper()
	latestDigest, latestBinding, found, err := ledger.LatestAuditProofBindingForRecord(recordDigest, bindingPayload.StatementFamily, bindingPayload.SchemeAdapterID)
	if err != nil {
		t.Fatalf("LatestAuditProofBindingForRecord returned error: %v", err)
	}
	if !found {
		t.Fatal("LatestAuditProofBindingForRecord found=false, want true")
	}
	if mustDigestIdentity(latestDigest) != mustDigestIdentity(bindingDigest) {
		t.Fatalf("latest digest = %q, want %q", mustDigestIdentity(latestDigest), mustDigestIdentity(bindingDigest))
	}
	if latestBinding.StatementFamily != bindingPayload.StatementFamily {
		t.Fatalf("latest binding statement_family = %q, want %q", latestBinding.StatementFamily, bindingPayload.StatementFamily)
	}
}

func assertMatchingVerificationRecord(t *testing.T, ledger *Ledger, artifactDigest trustpolicy.Digest, artifactPayload trustpolicy.ZKProofArtifactPayload) {
	t.Helper()
	verification := validZKProofVerificationRecordPayloadFixture(artifactDigest, artifactPayload)
	verificationDigest, err := ledger.PersistZKProofVerificationRecord(verification)
	if err != nil {
		t.Fatalf("PersistZKProofVerificationRecord returned error: %v", err)
	}
	foundDigest, foundRecord, found, err := ledger.FindMatchingZKProofVerificationRecord(verification)
	if err != nil {
		t.Fatalf("FindMatchingZKProofVerificationRecord returned error: %v", err)
	}
	if !found {
		t.Fatal("FindMatchingZKProofVerificationRecord found=false, want true")
	}
	if mustDigestIdentity(foundDigest) != mustDigestIdentity(verificationDigest) {
		t.Fatalf("verification digest = %q, want %q", mustDigestIdentity(foundDigest), mustDigestIdentity(verificationDigest))
	}
	if foundRecord.VerificationOutcome != verification.VerificationOutcome {
		t.Fatalf("verification_outcome = %q, want %q", foundRecord.VerificationOutcome, verification.VerificationOutcome)
	}
}

func TestSameStringSetRejectsDuplicatesAndMismatches(t *testing.T) {
	if sameStringSet([]string{"a", "a"}, []string{"a", "a"}) {
		t.Fatal("sameStringSet accepted duplicates")
	}
	if sameStringSet([]string{"a"}, []string{"b"}) {
		t.Fatal("sameStringSet accepted mismatch")
	}
	if !sameStringSet([]string{"a", "b"}, []string{"b", "a"}) {
		t.Fatal("sameStringSet should accept same unordered set")
	}
}

func TestZKProofArtifactAndBindingLookupNotFound(t *testing.T) {
	_, ledger, _ := setupLedgerWithAdmissionFixture(t)
	notFound := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)}
	assertArtifactAndBindingLookupsNotFound(t, ledger, notFound)
	assertVerificationLookupNotFound(t, ledger, notFound)
}

func assertArtifactAndBindingLookupsNotFound(t *testing.T, ledger *Ledger, notFound trustpolicy.Digest) {
	t.Helper()
	_, found, err := ledger.ZKProofArtifactByDigest(notFound)
	if err != nil {
		t.Fatalf("ZKProofArtifactByDigest returned error: %v", err)
	}
	if found {
		t.Fatal("ZKProofArtifactByDigest found=true, want false")
	}
	_, found, err = ledger.AuditProofBindingByDigest(notFound)
	if err != nil {
		t.Fatalf("AuditProofBindingByDigest returned error: %v", err)
	}
	if found {
		t.Fatal("AuditProofBindingByDigest found=true, want false")
	}
}

func assertVerificationLookupNotFound(t *testing.T, ledger *Ledger, notFound trustpolicy.Digest) {
	t.Helper()
	_, _, found, err := ledger.FindMatchingZKProofVerificationRecord(notFoundVerificationRecordFixture(notFound))
	if err != nil {
		t.Fatalf("FindMatchingZKProofVerificationRecord returned error: %v", err)
	}
	if found {
		t.Fatal("FindMatchingZKProofVerificationRecord found=true, want false")
	}
}

func notFoundVerificationRecordFixture(notFound trustpolicy.Digest) trustpolicy.ZKProofVerificationRecordPayload {
	return trustpolicy.ZKProofVerificationRecordPayload{SchemaID: trustpolicy.ZKProofVerificationRecordSchemaID, SchemaVersion: trustpolicy.ZKProofVerificationRecordSchemaVersion, ProofDigest: notFound, StatementFamily: "audit.isolate_session_bound.attested_runtime_membership.v0", StatementVersion: "v0", SchemeID: "groth16", CurveID: "bn254", CircuitID: "runecode.zk.circuit.audit.isolate_session_bound.attested_runtime_membership.v0", ConstraintSystemDigest: notFound, VerifierKeyDigest: notFound, SetupProvenanceDigest: notFound, NormalizationProfileID: "runecode.zk.normalize.audit.isolate_session_bound.attested_runtime.v0", SchemeAdapterID: "runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v0", PublicInputsDigest: notFound, VerifierImplementationID: "runecode.trusted.zk.verifier.gnark.v0", VerifiedAt: time.Now().UTC().Format(time.RFC3339), VerificationOutcome: trustpolicy.ProofVerificationOutcomeRejected, ReasonCodes: []string{trustpolicy.ProofVerificationReasonProofInvalid}, CacheProvenance: "fresh"}
}

func TestLatestAuditProofBindingForRecordReturnsNewestMatch(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	recordDigest := mustRecordDigestForTest(t, ledger)

	first := validAuditProofBindingPayloadFixture(recordDigest, sealResult.SealEnvelopeDigest)
	first.BindingCommitment = "sha256:" + strings.Repeat("a", 64)
	firstDigest, created, err := ledger.PersistAuditProofBinding(first)
	if err != nil {
		t.Fatalf("PersistAuditProofBinding(first) returned error: %v", err)
	}
	if !created {
		t.Fatal("PersistAuditProofBinding(first) created=false, want true")
	}
	time.Sleep(2 * time.Millisecond)
	second := first
	second.BindingCommitment = "sha256:" + strings.Repeat("b", 64)
	secondDigest, created, err := ledger.PersistAuditProofBinding(second)
	if err != nil {
		t.Fatalf("PersistAuditProofBinding(second) returned error: %v", err)
	}
	if !created {
		t.Fatal("PersistAuditProofBinding(second) created=false, want true")
	}

	latestDigest, latestBinding, found, err := ledger.LatestAuditProofBindingForRecord(recordDigest, first.StatementFamily, first.SchemeAdapterID)
	if err != nil {
		t.Fatalf("LatestAuditProofBindingForRecord returned error: %v", err)
	}
	if !found {
		t.Fatal("LatestAuditProofBindingForRecord found=false, want true")
	}
	if mustDigestIdentity(latestDigest) != mustDigestIdentity(secondDigest) {
		t.Fatalf("latest digest = %q, want %q (first was %q)", mustDigestIdentity(latestDigest), mustDigestIdentity(secondDigest), mustDigestIdentity(firstDigest))
	}
	if latestBinding.BindingCommitment != second.BindingCommitment {
		t.Fatalf("latest binding commitment = %q, want %q", latestBinding.BindingCommitment, second.BindingCommitment)
	}
}

func TestLatestAuditProofBindingForRecordIgnoresFilesystemModTime(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	recordDigest := mustRecordDigestForTest(t, ledger)
	first, firstDigest, second, secondDigest := persistSkewedBindingPair(t, ledger, recordDigest, sealResult.SealEnvelopeDigest)
	firstPath := filepath.Join(root, sidecarDirName, proofBindingsDirName, strings.TrimPrefix(mustDigestIdentity(firstDigest), "sha256:")+".json")
	secondPath := filepath.Join(root, sidecarDirName, proofBindingsDirName, strings.TrimPrefix(mustDigestIdentity(secondDigest), "sha256:")+".json")
	skewBindingPairModTimes(t, firstPath, secondPath)

	latestDigest, _, found, err := ledger.LatestAuditProofBindingForRecord(recordDigest, first.StatementFamily, first.SchemeAdapterID)
	if err != nil {
		t.Fatalf("LatestAuditProofBindingForRecord returned error: %v", err)
	}
	if !found {
		t.Fatal("LatestAuditProofBindingForRecord found=false, want true")
	}
	if mustDigestIdentity(latestDigest) != mustDigestIdentity(secondDigest) {
		t.Fatalf("latest digest = %q, want %q after modtime skew", mustDigestIdentity(latestDigest), mustDigestIdentity(secondDigest))
	}
	_ = second
}

func TestZKProofLookupsWorkAfterReopenWithPersistedLookupIndex(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	lookup := persistProofLookupIndexFixture(t, ledger, fixture)

	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(reopen) returned error: %v", err)
	}
	latestDigest, _, found, err := reopened.LatestAuditProofBindingForRecord(lookup.recordDigest, lookup.bindingPayload.StatementFamily, lookup.bindingPayload.SchemeAdapterID)
	if err != nil {
		t.Fatalf("LatestAuditProofBindingForRecord(reopen) returned error: %v", err)
	}
	if !found || mustDigestIdentity(latestDigest) != mustDigestIdentity(lookup.bindingDigest) {
		t.Fatalf("latest digest after reopen = %q, want %q", mustDigestIdentity(latestDigest), mustDigestIdentity(lookup.bindingDigest))
	}
	foundDigest, _, found, err := reopened.FindMatchingZKProofVerificationRecord(lookup.verificationPayload)
	if err != nil {
		t.Fatalf("FindMatchingZKProofVerificationRecord(reopen) returned error: %v", err)
	}
	if !found || mustDigestIdentity(foundDigest) != mustDigestIdentity(lookup.verificationDigest) {
		t.Fatalf("verification digest after reopen = %q, want %q", mustDigestIdentity(foundDigest), mustDigestIdentity(lookup.verificationDigest))
	}
	inclusion, found, err := reopened.AuditRecordInclusion(mustDigestIdentity(lookup.recordDigest))
	if err != nil {
		t.Fatalf("AuditRecordInclusion(reopen) returned error: %v", err)
	}
	if !found {
		t.Fatal("AuditRecordInclusion(reopen) found=false, want true")
	}
	if err := inclusion.Validate(); err != nil {
		t.Fatalf("inclusion.Validate after reopen returned error: %v", err)
	}
}

func persistSkewedBindingPair(t *testing.T, ledger *Ledger, recordDigest, sealDigest trustpolicy.Digest) (trustpolicy.AuditProofBindingPayload, trustpolicy.Digest, trustpolicy.AuditProofBindingPayload, trustpolicy.Digest) {
	t.Helper()
	first := validAuditProofBindingPayloadFixture(recordDigest, sealDigest)
	first.BindingCommitment = "sha256:" + strings.Repeat("a", 64)
	firstDigest := mustPersistBinding(t, ledger, first, "first")
	second := first
	second.BindingCommitment = "sha256:" + strings.Repeat("b", 64)
	secondDigest := mustPersistBinding(t, ledger, second, "second")
	return first, firstDigest, second, secondDigest
}

func mustPersistBinding(t *testing.T, ledger *Ledger, payload trustpolicy.AuditProofBindingPayload, label string) trustpolicy.Digest {
	t.Helper()
	digest, created, err := ledger.PersistAuditProofBinding(payload)
	if err != nil {
		t.Fatalf("PersistAuditProofBinding(%s) returned error: %v", label, err)
	}
	if !created {
		t.Fatalf("PersistAuditProofBinding(%s) created=false, want true", label)
	}
	return digest
}

func skewBindingPairModTimes(t *testing.T, firstPath, secondPath string) {
	t.Helper()
	now := time.Now().UTC()
	if err := os.Chtimes(firstPath, now.Add(2*time.Hour), now.Add(2*time.Hour)); err != nil {
		t.Fatalf("Chtimes(first) returned error: %v", err)
	}
	if err := os.Chtimes(secondPath, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("Chtimes(second) returned error: %v", err)
	}
}
