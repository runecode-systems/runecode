package artifacts

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestRestoreRejectsTamperedAttestationVerificationCacheRecord(t *testing.T) {
	store := newTestStore(t)
	runID := "run-restore-tampered-attestation-cache"
	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-restore", "policy-restore")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState returned error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup-attestation-cache-tampered.json")
	manifest := loadExportedBackupManifest(t, store, backupPath)
	if evidence.AttestationVerification == nil {
		t.Fatal("expected attestation verification record for fixture evidence")
	}
	cacheKey := attestationVerificationCacheKeyFromFields(
		evidence.Attestation.EvidenceDigest,
		evidence.Launch.AuthorityStateDigest,
		evidence.Attestation.MeasurementProfile,
	)
	if cacheKey == "" {
		t.Fatal("expected non-empty attestation verification cache key")
	}
	record := cloneAttestationVerificationRecord(*evidence.AttestationVerification)
	record.VerificationDigest = DigestBytes([]byte("tampered-verification"))
	manifest.AttestationVerificationCache = map[string]launcherbackend.IsolateAttestationVerificationRecord{cacheKey: record}
	writeBackupManifestWithSignature(t, store, backupPath, manifest)
	restoreStore := newTestStore(t)
	err := restoreStore.RestoreBackup(backupPath)
	if err == nil {
		t.Fatal("RestoreBackup expected tampered attestation verification cache error")
	}
	if !strings.Contains(err.Error(), "verification_digest does not match attestation verification record") {
		t.Fatalf("RestoreBackup error = %v, want verification digest mismatch", err)
	}
}

func TestRestoreRuntimeEvidencePrefersDerivedRuntimeFacts(t *testing.T) {
	store := newTestStore(t)
	runID := "run-restore-derived-runtime-evidence"
	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-derived", "policy-derived")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState returned error: %v", err)
	}
	restoreStore, restoredFacts, restoredEvidence := restoreStoreWithTamperedRuntimeEvidence(t, store, runID)
	_, _, _, _, ok := restoreStore.RuntimeEvidenceState(runID)
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want restored runtime state")
	}
	assertDerivedRuntimeEvidenceRestored(t, restoredFacts, restoredEvidence)
}

func restoreStoreWithTamperedRuntimeEvidence(t *testing.T, store *Store, runID string) (*Store, launcherbackend.RuntimeFactsSnapshot, launcherbackend.RuntimeEvidenceSnapshot) {
	t.Helper()
	backupPath := filepath.Join(t.TempDir(), "backup-derived-runtime-evidence.json")
	manifest := loadExportedBackupManifest(t, store, backupPath)
	tampered := manifest.RuntimeEvidenceByRun[runID]
	if tampered.AttestationVerification == nil {
		t.Fatal("expected attestation verification record in runtime evidence backup")
	}
	tamperedVerification := *tampered.AttestationVerification
	tamperedVerification.VerifierPolicyDigest = testDigest("forged-policy")
	tamperedVerification.VerificationDigest = testDigest("forged-verification")
	tampered.AttestationVerification = &tamperedVerification
	manifest.RuntimeEvidenceByRun[runID] = tampered
	writeBackupManifestWithSignature(t, store, backupPath, manifest)
	restoreStore := newTestStore(t)
	if err := restoreStore.RestoreBackup(backupPath); err != nil {
		t.Fatalf("RestoreBackup returned error: %v", err)
	}
	restoredFacts, restoredEvidence, _, _, ok := restoreStore.RuntimeEvidenceState(runID)
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want restored runtime state")
	}
	return restoreStore, restoredFacts, restoredEvidence
}

func assertDerivedRuntimeEvidenceRestored(t *testing.T, restoredFacts launcherbackend.RuntimeFactsSnapshot, restoredEvidence launcherbackend.RuntimeEvidenceSnapshot) {
	t.Helper()
	derivedEvidence, _, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(restoredFacts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if restoredEvidence.AttestationVerification == nil || derivedEvidence.AttestationVerification == nil {
		t.Fatalf("attestation verification missing after restore; restored=%#v derived=%#v", restoredEvidence.AttestationVerification, derivedEvidence.AttestationVerification)
	}
	if restoredEvidence.AttestationVerification.VerificationDigest != derivedEvidence.AttestationVerification.VerificationDigest {
		t.Fatalf("restored attestation verification digest = %q, want %q", restoredEvidence.AttestationVerification.VerificationDigest, derivedEvidence.AttestationVerification.VerificationDigest)
	}
	if restoredEvidence.AttestationVerification.VerifierPolicyDigest != derivedEvidence.AttestationVerification.VerifierPolicyDigest {
		t.Fatalf("restored attestation policy digest = %q, want %q", restoredEvidence.AttestationVerification.VerifierPolicyDigest, derivedEvidence.AttestationVerification.VerifierPolicyDigest)
	}
}

func TestRestoreRejectsInvalidAttestationVerificationCacheKey(t *testing.T) {
	store := newTestStore(t)
	runID := "run-restore-invalid-attestation-cache-key"
	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-invalid-key", "policy-invalid-key")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState returned error: %v", err)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("expected attestation verification record for fixture evidence")
	}
	backupPath := filepath.Join(t.TempDir(), "backup-attestation-cache-invalid-key.json")
	manifest := loadExportedBackupManifest(t, store, backupPath)
	record := cloneAttestationVerificationRecord(*evidence.AttestationVerification)
	manifest.AttestationVerificationCache = map[string]launcherbackend.IsolateAttestationVerificationRecord{"not-a-valid-key|still-invalid|profile": record}
	writeBackupManifestWithSignature(t, store, backupPath, manifest)
	restoreStore := newTestStore(t)
	err := restoreStore.RestoreBackup(backupPath)
	if err == nil {
		t.Fatal("RestoreBackup expected invalid attestation verification cache key error")
	}
	if !strings.Contains(err.Error(), "attestation verification cache key must be structurally valid") {
		t.Fatalf("RestoreBackup error = %v, want invalid cache key error", err)
	}
}

func TestRestoreRejectsCollidingAttestationVerificationCacheKeys(t *testing.T) {
	store := newTestStore(t)
	runID := "run-restore-colliding-attestation-cache-keys"
	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-collision", "policy-collision")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState returned error: %v", err)
	}
	if evidence.Attestation == nil || evidence.AttestationVerification == nil {
		t.Fatal("expected attestation evidence and verification for collision fixture")
	}
	backupPath := filepath.Join(t.TempDir(), "backup-attestation-cache-collision.json")
	manifest := loadExportedBackupManifest(t, store, backupPath)
	record := cloneAttestationVerificationRecord(*evidence.AttestationVerification)
	legacyKey := strings.Join([]string{evidence.Attestation.EvidenceDigest, evidence.Launch.AuthorityStateDigest, evidence.Attestation.MeasurementProfile}, "|")
	normalizedKey := attestationVerificationCacheKeyFromFields(evidence.Attestation.EvidenceDigest, evidence.Launch.AuthorityStateDigest, evidence.Attestation.MeasurementProfile)
	if normalizedKey == "" {
		t.Fatal("expected normalized attestation cache key")
	}
	modified := cloneAttestationVerificationRecord(record)
	modified.VerifierPolicyDigest = DigestBytes([]byte("policy-collision-other"))
	if err := launcherbackend.FinalizeIsolateAttestationVerificationRecord(&modified); err != nil {
		t.Fatalf("FinalizeIsolateAttestationVerificationRecord returned error: %v", err)
	}
	manifest.AttestationVerificationCache = map[string]launcherbackend.IsolateAttestationVerificationRecord{
		legacyKey:     record,
		normalizedKey: modified,
	}
	writeBackupManifestWithSignature(t, store, backupPath, manifest)
	restoreStore := newTestStore(t)
	err := restoreStore.RestoreBackup(backupPath)
	if err == nil {
		t.Fatal("RestoreBackup expected attestation verification cache collision error")
	}
	if !strings.Contains(err.Error(), "attestation verification cache key collision") {
		t.Fatalf("RestoreBackup error = %v, want cache collision error", err)
	}
}

func TestRestoreRejectsRuntimeEvidenceWithoutRuntimeFacts(t *testing.T) {
	store := newTestStore(t)
	runID := "run-restore-orphan-runtime-evidence"
	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-orphan", "policy-orphan")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState returned error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup-orphan-runtime-evidence.json")
	manifest := loadExportedBackupManifest(t, store, backupPath)
	delete(manifest.RuntimeFactsByRun, runID)
	writeBackupManifestWithSignature(t, store, backupPath, manifest)
	restoreStore := newTestStore(t)
	err := restoreStore.RestoreBackup(backupPath)
	if err == nil {
		t.Fatal("RestoreBackup expected missing runtime facts error")
	}
	if !strings.Contains(err.Error(), "requires runtime facts") {
		t.Fatalf("RestoreBackup error = %v, want runtime facts requirement error", err)
	}
}
