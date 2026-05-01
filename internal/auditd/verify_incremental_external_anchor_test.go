package auditd

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestVerifyCurrentSegmentIncrementalWithPreverifiedSealPersistsReport(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.VerifyCurrentSegmentAndPersist(); err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist baseline returned error: %v", err)
	}
	anchorSigner := newAuditFixtureKey(t)
	receiptEnvelope := fixtureAnchorReceiptEnvelopeForSubject(t, anchorSigner, sealResult.SealEnvelopeDigest)
	receiptDigest, err := ledger.PersistReceiptEnvelope(receiptEnvelope)
	if err != nil {
		t.Fatalf("PersistReceiptEnvelope returned error: %v", err)
	}
	assertDigestSidecarExists(t, root+"/sidecar/receipts", mustDigestIdentity(receiptDigest))

	reportDigest, err := ledger.VerifyCurrentSegmentIncrementalWithPreverifiedSeal(sealResult.SealEnvelopeDigest, fixtureAuditAnchorVerifierRecordForKey(anchorSigner, anchorSigner.keyIDValue))
	if err != nil {
		t.Fatalf("VerifyCurrentSegmentIncrementalWithPreverifiedSeal returned error: %v", err)
	}
	assertDigestSidecarExists(t, root+"/sidecar/verification-reports", mustDigestIdentity(reportDigest))
	report, err := ledger.LatestVerificationReport()
	if err != nil {
		t.Fatalf("LatestVerificationReport returned error: %v", err)
	}
	if report.VerificationScope.ScopeKind != trustpolicy.AuditVerificationScopeSegment {
		t.Fatalf("verification_scope.scope_kind=%q, want segment", report.VerificationScope.ScopeKind)
	}
	if report.VerificationScope.LastSegmentID == "" {
		t.Fatal("verification_scope.last_segment_id empty")
	}
}

func TestVerifyCurrentSegmentIncrementalWithPreverifiedSealRejectsMismatchedSeal(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	_ = mustSealFixtureSegment(t, ledger, fixture)
	anchorSigner := newAuditFixtureKey(t)
	wrong := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)}
	if _, err := ledger.VerifyCurrentSegmentIncrementalWithPreverifiedSeal(wrong, fixtureAuditAnchorVerifierRecordForKey(anchorSigner, anchorSigner.keyIDValue)); err == nil {
		t.Fatal("VerifyCurrentSegmentIncrementalWithPreverifiedSeal expected seal mismatch error")
	}
}

func TestVerifyCurrentSegmentIncrementalWithPreverifiedSealRequiresBaselineFoundation(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	anchorSigner := newAuditFixtureKey(t)
	if _, err := ledger.VerifyCurrentSegmentIncrementalWithPreverifiedSeal(sealResult.SealEnvelopeDigest, fixtureAuditAnchorVerifierRecordForKey(anchorSigner, anchorSigner.keyIDValue)); err == nil || !strings.Contains(err.Error(), "foundation missing") {
		t.Fatalf("VerifyCurrentSegmentIncrementalWithPreverifiedSeal error=%v, want incremental foundation requirement", err)
	}
}

func TestVerifyCurrentSegmentIncrementalWithPreverifiedSealUsesSealScopedFoundationInputs(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.VerifyCurrentSegmentAndPersist(); err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist baseline returned error: %v", err)
	}
	anchorSigner := newAuditFixtureKey(t)
	receiptEnvelope := fixtureAnchorReceiptEnvelopeForSubject(t, anchorSigner, sealResult.SealEnvelopeDigest)
	if _, err := ledger.PersistReceiptEnvelope(receiptEnvelope); err != nil {
		t.Fatalf("PersistReceiptEnvelope returned error: %v", err)
	}
	strayReceiptPath := filepath.Join(root, sidecarDirName, receiptsDirName, strings.Repeat("f", 64)+".json")
	if err := os.WriteFile(strayReceiptPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile stray receipt returned error: %v", err)
	}
	reportDigest, err := ledger.VerifyCurrentSegmentIncrementalWithPreverifiedSeal(sealResult.SealEnvelopeDigest, fixtureAuditAnchorVerifierRecordForKey(anchorSigner, anchorSigner.keyIDValue))
	if err != nil {
		t.Fatalf("VerifyCurrentSegmentIncrementalWithPreverifiedSeal returned error with stray unrelated receipt file present: %v", err)
	}
	assertDigestSidecarExists(t, root+"/sidecar/verification-reports", mustDigestIdentity(reportDigest))
}

func TestVerifyCurrentSegmentIncrementalWithPreverifiedSealFailsClosedOnTamperedSidecarContent(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.VerifyCurrentSegmentAndPersist(); err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist baseline returned error: %v", err)
	}
	anchorSigner := newAuditFixtureKey(t)
	receiptEnvelope := fixtureAnchorReceiptEnvelopeForSubject(t, anchorSigner, sealResult.SealEnvelopeDigest)
	if _, err := ledger.PersistReceiptEnvelope(receiptEnvelope); err != nil {
		t.Fatalf("PersistReceiptEnvelope returned error: %v", err)
	}
	tamperExternalAnchorSidecarContentForSeal(t, root, ledger, sealResult.SealEnvelopeDigest)

	if _, err := ledger.VerifyCurrentSegmentIncrementalWithPreverifiedSeal(sealResult.SealEnvelopeDigest, fixtureAuditAnchorVerifierRecordForKey(anchorSigner, anchorSigner.keyIDValue)); err == nil || !strings.Contains(err.Error(), "external anchor sidecar digest mismatch") {
		t.Fatalf("VerifyCurrentSegmentIncrementalWithPreverifiedSeal error=%v, want sidecar digest mismatch", err)
	}
}

func tamperExternalAnchorSidecarContentForSeal(t *testing.T, root string, ledger *Ledger, sealDigest trustpolicy.Digest) {
	t.Helper()
	proofDigest := persistExternalAnchorEvidenceForTamperTest(t, ledger, sealDigest)
	proofIdentity := mustDigestIdentity(proofDigest)
	tamperedPath := filepath.Join(root, sidecarDirName, externalAnchorSidecarsDir, strings.TrimPrefix(proofIdentity, "sha256:")+".json")
	tampered := ExternalAnchorSidecarPayload{
		SchemaID:      externalAnchorSidecarSchemaID,
		SchemaVersion: externalAnchorSidecarSchemaVersion,
		EvidenceKind:  trustpolicy.ExternalAnchorSidecarKindProofBytes,
		Payload: map[string]any{
			"schema_id":      "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0",
			"schema_version": "0.1.0",
			"proof":          "tampered",
		},
	}
	if err := writeCanonicalJSONFile(tamperedPath, tampered); err != nil {
		t.Fatalf("writeCanonicalJSONFile(tampered sidecar) returned error: %v", err)
	}
}

func persistExternalAnchorEvidenceForTamperTest(t *testing.T, ledger *Ledger, sealDigest trustpolicy.Digest) trustpolicy.Digest {
	t.Helper()
	proofDigest, err := ledger.PersistExternalAnchorSidecar(trustpolicy.ExternalAnchorSidecarKindProofBytes, map[string]any{"schema_id": "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0", "schema_version": "0.1.0", "proof": "fixture"})
	if err != nil {
		t.Fatalf("PersistExternalAnchorSidecar returned error: %v", err)
	}
	targetDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("8", 64)}
	targetIdentity, err := targetDigest.Identity()
	if err != nil {
		t.Fatalf("targetDigest.Identity returned error: %v", err)
	}
	outbound := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("c", 64)}
	if _, _, err := ledger.PersistExternalAnchorEvidence(ExternalAnchorEvidenceRequest{
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
	}); err != nil {
		t.Fatalf("PersistExternalAnchorEvidence returned error: %v", err)
	}
	return proofDigest
}

func TestCurrentVerificationContextUsesPersistedExternalAnchorTargetSet(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.VerifyCurrentSegmentAndPersist(); err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist baseline returned error: %v", err)
	}
	anchorSigner := newAuditFixtureKey(t)
	receiptEnvelope := fixtureAnchorReceiptEnvelopeForSubject(t, anchorSigner, sealResult.SealEnvelopeDigest)
	if _, err := ledger.PersistReceiptEnvelope(receiptEnvelope); err != nil {
		t.Fatalf("PersistReceiptEnvelope returned error: %v", err)
	}
	targetDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("9", 64)}
	if err := persistExternalAnchorEvidenceForSeal(t, ledger, sealResult.SealEnvelopeDigest, targetDigest, trustpolicy.ExternalAnchorTargetRequirementOptional, trustpolicy.ExternalAnchorOutcomeDeferred); err != nil {
		t.Fatalf("persistExternalAnchorEvidenceForSeal returned error: %v", err)
	}
	forceFoundationTargetRequirementForSeal(t, root, sealResult.SealEnvelopeDigest, targetDigest, trustpolicy.ExternalAnchorTargetRequirementRequired)

	ledger.mu.Lock()
	_, input, err := ledger.currentVerificationContextLocked()
	ledger.mu.Unlock()
	if err != nil {
		t.Fatalf("currentVerificationContextLocked returned error: %v", err)
	}
	if len(input.ExternalAnchorTargetSet) != 1 {
		t.Fatalf("ExternalAnchorTargetSet length=%d, want 1", len(input.ExternalAnchorTargetSet))
	}
	if mustDigestIdentity(input.ExternalAnchorTargetSet[0].TargetDescriptorDigest) != mustDigestIdentity(targetDigest) {
		t.Fatalf("ExternalAnchorTargetSet[0].target_descriptor_digest=%q, want %q", mustDigestIdentity(input.ExternalAnchorTargetSet[0].TargetDescriptorDigest), mustDigestIdentity(targetDigest))
	}
	if input.ExternalAnchorTargetSet[0].TargetRequirement != trustpolicy.ExternalAnchorTargetRequirementRequired {
		t.Fatalf("ExternalAnchorTargetSet[0].target_requirement=%q, want required", input.ExternalAnchorTargetSet[0].TargetRequirement)
	}
}

func TestCurrentVerificationContextLoadsProofSidecarsFromDedicatedLocation(t *testing.T) {
	_, ledger := setupVerificationContextWithExternalAnchorEvidenceForTest(t)
	input := currentVerificationContextForTest(t, ledger)
	assertExternalAnchorVerificationInputCounts(t, input, 1, 1)

	proofDigest, evidenceDigest := externalAnchorProofAndEvidenceDigestsForVerificationContextTest(t, ledger)
	assertDigestListContains(t, input.ExternalAnchorSidecars, proofDigest)
	assertDigestListExcludes(t, input.ExternalAnchorSidecars, evidenceDigest)
}

func TestVerifyCurrentSegmentIncrementalWithPreverifiedSealEvidenceFileCannotSatisfySidecarLookup(t *testing.T) {
	ledger, sealDigest, verifier := setupIncrementalVerificationWithEvidenceSidecarSpoof(t)
	if _, err := ledger.VerifyCurrentSegmentIncrementalWithPreverifiedSeal(sealDigest, verifier); err == nil || !strings.Contains(err.Error(), "external anchor sidecar missing or unreadable") {
		t.Fatalf("VerifyCurrentSegmentIncrementalWithPreverifiedSeal error=%v, want sidecar missing/unreadable", err)
	}
}

func setupIncrementalVerificationWithEvidenceSidecarSpoof(t *testing.T) (*Ledger, trustpolicy.Digest, trustpolicy.VerifierRecord) {
	t.Helper()
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.VerifyCurrentSegmentAndPersist(); err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist baseline returned error: %v", err)
	}
	anchorSigner := newAuditFixtureKey(t)
	if _, err := ledger.PersistReceiptEnvelope(fixtureAnchorReceiptEnvelopeForSubject(t, anchorSigner, sealResult.SealEnvelopeDigest)); err != nil {
		t.Fatalf("PersistReceiptEnvelope returned error: %v", err)
	}
	proofDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("7", 64)}
	persistExternalAnchorEvidenceWithProofDigest(t, ledger, sealResult.SealEnvelopeDigest, proofDigest)
	writeExternalAnchorSidecarSpoofToEvidenceDir(t, root, proofDigest)
	return ledger, sealResult.SealEnvelopeDigest, fixtureAuditAnchorVerifierRecordForKey(anchorSigner, anchorSigner.keyIDValue)
}

func persistExternalAnchorEvidenceWithProofDigest(t *testing.T, ledger *Ledger, sealDigest trustpolicy.Digest, proofDigest trustpolicy.Digest) {
	t.Helper()
	targetDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("8", 64)}
	targetIdentity, err := targetDigest.Identity()
	if err != nil {
		t.Fatalf("targetDigest.Identity returned error: %v", err)
	}
	outbound := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("c", 64)}
	if _, _, err := ledger.PersistExternalAnchorEvidence(ExternalAnchorEvidenceRequest{
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
	}); err != nil {
		t.Fatalf("PersistExternalAnchorEvidence returned error: %v", err)
	}
}

func writeExternalAnchorSidecarSpoofToEvidenceDir(t *testing.T, root string, proofDigest trustpolicy.Digest) {
	t.Helper()
	proofIdentity := mustDigestIdentity(proofDigest)
	evidencePath := filepath.Join(root, sidecarDirName, externalAnchorEvidenceDir, strings.TrimPrefix(proofIdentity, "sha256:")+".json")
	if err := writeCanonicalJSONFile(evidencePath, ExternalAnchorSidecarPayload{
		SchemaID:      externalAnchorSidecarSchemaID,
		SchemaVersion: externalAnchorSidecarSchemaVersion,
		EvidenceKind:  trustpolicy.ExternalAnchorSidecarKindProofBytes,
		Payload: map[string]any{
			"schema_id":      "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0",
			"schema_version": "0.1.0",
			"proof":          "evidence-dir-spoof",
		},
	}); err != nil {
		t.Fatalf("writeCanonicalJSONFile(evidence sidecar spoof) returned error: %v", err)
	}
}

func setupVerificationContextWithExternalAnchorEvidenceForTest(t *testing.T) (string, *Ledger) {
	t.Helper()
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.VerifyCurrentSegmentAndPersist(); err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist baseline returned error: %v", err)
	}
	anchorSigner := newAuditFixtureKey(t)
	receiptEnvelope := fixtureAnchorReceiptEnvelopeForSubject(t, anchorSigner, sealResult.SealEnvelopeDigest)
	if _, err := ledger.PersistReceiptEnvelope(receiptEnvelope); err != nil {
		t.Fatalf("PersistReceiptEnvelope returned error: %v", err)
	}
	targetDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("8", 64)}
	if err := persistExternalAnchorEvidenceForSeal(t, ledger, sealResult.SealEnvelopeDigest, targetDigest, trustpolicy.ExternalAnchorTargetRequirementOptional, trustpolicy.ExternalAnchorOutcomeDeferred); err != nil {
		t.Fatalf("persistExternalAnchorEvidenceForSeal returned error: %v", err)
	}
	return root, ledger
}

func externalAnchorProofAndEvidenceDigestsForVerificationContextTest(t *testing.T, ledger *Ledger) (trustpolicy.Digest, trustpolicy.Digest) {
	t.Helper()
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	segment, _, _, _, _, err := ledger.currentSegmentEvidenceLocked()
	if err != nil {
		t.Fatalf("currentSegmentEvidenceLocked returned error: %v", err)
	}
	currentSnapshot, err := ledger.currentSealSnapshotLocked()
	if err != nil {
		t.Fatalf("currentSealSnapshotLocked returned error: %v", err)
	}
	snapshot, err := ledger.requireSealIncrementalFoundationEntryLocked(segment.Header.SegmentID, currentSnapshot.sealDigest, true)
	if err != nil {
		t.Fatalf("requireSealIncrementalFoundationEntryLocked returned error: %v", err)
	}
	if snapshot == nil {
		t.Fatal("requireSealIncrementalFoundationEntryLocked returned nil snapshot")
	}
	if len(snapshot.ExternalAnchorSidecarDigests) != 1 {
		t.Fatalf("ExternalAnchorSidecarDigests length=%d, want 1", len(snapshot.ExternalAnchorSidecarDigests))
	}
	if len(snapshot.ExternalAnchorEvidenceDigests) != 1 {
		t.Fatalf("ExternalAnchorEvidenceDigests length=%d, want 1", len(snapshot.ExternalAnchorEvidenceDigests))
	}
	proofDigest, err := digestFromIdentity(snapshot.ExternalAnchorSidecarDigests[0])
	if err != nil {
		t.Fatalf("digestFromIdentity(sidecar) returned error: %v", err)
	}
	evidenceDigest, err := digestFromIdentity(snapshot.ExternalAnchorEvidenceDigests[0])
	if err != nil {
		t.Fatalf("digestFromIdentity(evidence) returned error: %v", err)
	}
	return proofDigest, evidenceDigest
}
func TestCurrentVerificationContextFailsClosedOnMalformedExternalAnchorEvidence(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	_ = mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.VerifyCurrentSegmentAndPersist(); err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist baseline returned error: %v", err)
	}

	malformedPath := filepath.Join(root, sidecarDirName, externalAnchorEvidenceDir, strings.Repeat("e", 64)+".json")
	malformedPayload, err := json.Marshal(map[string]any{
		"schema_id":      trustpolicy.ExternalAnchorEvidenceSchemaID,
		"schema_version": trustpolicy.ExternalAnchorEvidenceSchemaVersion,
	})
	if err != nil {
		t.Fatalf("Marshal malformed payload returned error: %v", err)
	}
	if err := os.WriteFile(malformedPath, malformedPayload, 0o600); err != nil {
		t.Fatalf("WriteFile malformed external-anchor file returned error: %v", err)
	}

	ledger.mu.Lock()
	_, _, err = ledger.currentVerificationContextLocked()
	ledger.mu.Unlock()
	if err == nil {
		t.Fatal("currentVerificationContextLocked expected malformed external anchor evidence error")
	}
}

func TestLoadSealScopedVerificationDurableInputsIncludesPersistedExternalAnchorTargetSet(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.VerifyCurrentSegmentAndPersist(); err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist baseline returned error: %v", err)
	}
	anchorSigner := newAuditFixtureKey(t)
	receiptEnvelope := fixtureAnchorReceiptEnvelopeForSubject(t, anchorSigner, sealResult.SealEnvelopeDigest)
	if _, err := ledger.PersistReceiptEnvelope(receiptEnvelope); err != nil {
		t.Fatalf("PersistReceiptEnvelope returned error: %v", err)
	}
	targetDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)}
	if err := persistExternalAnchorEvidenceForSeal(t, ledger, sealResult.SealEnvelopeDigest, targetDigest, trustpolicy.ExternalAnchorTargetRequirementOptional, trustpolicy.ExternalAnchorOutcomeDeferred); err != nil {
		t.Fatalf("persistExternalAnchorEvidenceForSeal returned error: %v", err)
	}
	forceFoundationTargetRequirementForSeal(t, root, sealResult.SealEnvelopeDigest, targetDigest, trustpolicy.ExternalAnchorTargetRequirementRequired)

	ledger.mu.Lock()
	segment, _, _, _, _, err := ledger.currentSegmentEvidenceLocked()
	if err != nil {
		ledger.mu.Unlock()
		t.Fatalf("currentSegmentEvidenceLocked returned error: %v", err)
	}
	inputs, err := ledger.loadSealScopedVerificationDurableInputsLocked(segment.Header.SegmentID, sealResult.SealEnvelopeDigest)
	ledger.mu.Unlock()
	if err != nil {
		t.Fatalf("loadSealScopedVerificationDurableInputsLocked returned error: %v", err)
	}
	if len(inputs.externalAnchorTargetSet) != 1 {
		t.Fatalf("externalAnchorTargetSet length=%d, want 1", len(inputs.externalAnchorTargetSet))
	}
	if inputs.externalAnchorTargetSet[0].TargetRequirement != trustpolicy.ExternalAnchorTargetRequirementRequired {
		t.Fatalf("externalAnchorTargetSet[0].target_requirement=%q, want required", inputs.externalAnchorTargetSet[0].TargetRequirement)
	}
	if mustDigestIdentity(inputs.externalAnchorTargetSet[0].TargetDescriptorDigest) != mustDigestIdentity(targetDigest) {
		t.Fatalf("externalAnchorTargetSet[0].target_descriptor_digest=%q, want %q", mustDigestIdentity(inputs.externalAnchorTargetSet[0].TargetDescriptorDigest), mustDigestIdentity(targetDigest))
	}
}

func fixtureAuditAnchorVerifierRecordForKey(f auditFixtureKey, keyIDValue string) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(f.publicKey)},
		LogicalPurpose:         "audit_anchor",
		LogicalScope:           "node",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "secretsd", InstanceID: "secretsd-1"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Status:                 "active",
	}
}

func fixtureAnchorReceiptEnvelopeForSubject(t *testing.T, f auditFixtureKey, subjectDigest trustpolicy.Digest) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeWithFixtureKey(t, f.privateKey, f.keyIDValue, trustpolicy.AuditReceiptSchemaID, trustpolicy.AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 trustpolicy.AuditReceiptSchemaID,
		"schema_version":            trustpolicy.AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        "anchor",
		"subject_family":            trustpolicy.AuditSegmentAnchoringSubjectSeal,
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"recorded_at":               "2026-03-13T12:25:00Z",
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.anchor.v0",
		"receipt_payload": map[string]any{
			"anchor_kind":            "local_user_presence_signature",
			"key_protection_posture": "os_keystore",
			"presence_mode":          "os_confirmation",
			"anchor_witness": map[string]any{
				"witness_kind":   "local_user_presence_signature_v0",
				"witness_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
			},
		},
	})
}

func signEnvelopeWithFixtureKey(t *testing.T, privateKey ed25519.PrivateKey, keyID, payloadSchemaID, payloadSchemaVersion string, payload map[string]any) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(payloadBytes)
	if err != nil {
		t.Fatalf("canonicalize payload returned error: %v", err)
	}
	signature := ed25519.Sign(privateKey, canonicalPayload)
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: payloadSchemaID, PayloadSchemaVersion: payloadSchemaVersion, Payload: payloadBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyID, Signature: base64.StdEncoding.EncodeToString(signature)}}
}

func persistExternalAnchorEvidenceForSeal(t *testing.T, ledger *Ledger, sealDigest trustpolicy.Digest, targetDigest trustpolicy.Digest, requirement string, outcome string) error {
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
		TargetRequirement:       requirement,
		AnchoringSubjectFamily:  trustpolicy.AuditSegmentAnchoringSubjectSeal,
		AnchoringSubjectDigest:  sealDigest,
		OutboundPayloadDigest:   &outbound,
		OutboundBytes:           128,
		Outcome:                 outcome,
		OutcomeReasonCode:       "external_anchor_execution_deferred",
		ProofDigest:             proofDigest,
		ProofSchemaID:           "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0",
		ProofKind:               "transparency_log_receipt_v0",
	})
	return err
}

func forceFoundationTargetRequirementForSeal(t *testing.T, root string, sealDigest trustpolicy.Digest, targetDigest trustpolicy.Digest, requirement string) {
	t.Helper()
	foundationPath := filepath.Join(root, indexDirName, externalAnchorIncrementalFoundationFileName)
	foundation := externalAnchorIncrementalFoundation{}
	if err := readJSONFile(foundationPath, &foundation); err != nil {
		t.Fatalf("readJSONFile(foundation) returned error: %v", err)
	}
	sealID := mustDigestIdentity(sealDigest)
	entry := foundation.Seals[sealID]
	entry.ExternalAnchorTargets = []externalAnchorVerificationTargetSnapshot{{
		TargetKind:             "transparency_log",
		TargetDescriptorDigest: mustDigestIdentity(targetDigest),
		TargetRequirement:      requirement,
	}}
	foundation.Seals[sealID] = entry
	if err := writeCanonicalJSONFile(foundationPath, foundation); err != nil {
		t.Fatalf("writeCanonicalJSONFile(foundation) returned error: %v", err)
	}
}
