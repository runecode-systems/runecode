package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRetentionGCAndBackupRestore(t *testing.T) {
	store, keep, backupPath := setupRetentionAndBackupFixture(t)
	assertRetentionAndRestore(t, store, keep, backupPath)
}

func setupRetentionAndBackupFixture(t *testing.T) (*Store, ArtifactReference, string) {
	store, now := setupRetentionStore(t)
	keep := seedRetentionArtifacts(t, store)
	runAndAssertGC(t, store, now, keep)
	backupPath := filepath.Join(t.TempDir(), "backup.json")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup error: %v", err)
	}
	return store, keep, backupPath
}

func setupRetentionStore(t *testing.T) (*Store, time.Time) {
	store := newTestStore(t)
	policy := store.Policy()
	policy.UnreferencedTTLSeconds = 1
	if err := store.SetPolicy(policy); err != nil {
		t.Fatalf("SetPolicy error: %v", err)
	}
	now := time.Now().UTC()
	store.nowFn = func() time.Time { return now }
	return store, now
}

func seedRetentionArtifacts(t *testing.T, store *Store) ArtifactReference {
	keep, err := store.Put(PutRequest{Payload: []byte("keep"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("9"), CreatedByRole: "workspace", RunID: "run-active"})
	if err != nil {
		t.Fatalf("Put keep error: %v", err)
	}
	if err := store.SetRunStatus("run-active", "active"); err != nil {
		t.Fatalf("SetRunStatus active error: %v", err)
	}
	if _, err := store.Put(PutRequest{Payload: []byte("drop"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("a"), CreatedByRole: "workspace", RunID: "run-closed"}); err != nil {
		t.Fatalf("Put drop error: %v", err)
	}
	if err := store.SetRunStatus("run-closed", "closed"); err != nil {
		t.Fatalf("SetRunStatus closed error: %v", err)
	}
	return keep
}

func runAndAssertGC(t *testing.T, store *Store, now time.Time, keep ArtifactReference) {
	store.nowFn = func() time.Time { return now.Add(5 * time.Second) }
	gcResult, err := store.GarbageCollect()
	if err != nil {
		t.Fatalf("GarbageCollect error: %v", err)
	}
	if gcResult.FreedBytes == 0 || len(gcResult.DeletedDigests) == 0 {
		t.Fatalf("expected GC to delete at least one artifact")
	}
	if _, err := store.Head(keep.Digest); err != nil {
		t.Fatalf("active run artifact should be retained: %v", err)
	}
}

func assertRetentionAndRestore(t *testing.T, sourceStore *Store, keep ArtifactReference, backupPath string) {
	b, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup error: %v", err)
	}
	var manifest BackupManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatalf("backup json parse error: %v", err)
	}
	if manifest.Schema != "runecode.backup.artifacts.v1" {
		t.Fatalf("backup schema = %q", manifest.Schema)
	}

	restoreStore := newTestStore(t)
	copyBlobsToStore(t, restoreStore, manifest.Artifacts)
	if err := restoreStore.RestoreBackup(backupPath); err != nil {
		t.Fatalf("RestoreBackup error: %v", err)
	}
	if _, err := restoreStore.Head(keep.Digest); err != nil {
		t.Fatalf("restored store missing retained artifact: %v", err)
	}
	_ = sourceStore
}

func TestRestoreRejectsForgedBackupRecord(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("c"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.json")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup error: %v", err)
	}
	b, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup error: %v", err)
	}
	manifest := BackupManifest{}
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatalf("parse backup error: %v", err)
	}
	manifest.Artifacts[0].Reference.Digest = testDigest("d")
	b, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal forged backup error: %v", err)
	}
	if err := os.WriteFile(backupPath, b, 0o600); err != nil {
		t.Fatalf("write forged backup error: %v", err)
	}
	restoreStore := newTestStore(t)
	copyBlobFile(t, store.storeIO.blobPath(ref.Digest), restoreStore.storeIO.blobPath(ref.Digest))
	err = restoreStore.RestoreBackup(backupPath)
	if err == nil {
		t.Fatal("RestoreBackup expected error for forged digest")
	}
}

func TestRestoreRejectsInvalidDigestBeforeBlobLookup(t *testing.T) {
	store := newTestStore(t)
	manifest := BackupManifest{
		Schema:            "runecode.backup.artifacts.v1",
		ExportedAt:        time.Now().UTC(),
		StorageProtection: "encrypted_at_rest_default",
		Policy:            DefaultPolicy(),
		Runs:              map[string]string{},
		Artifacts: []ArtifactRecord{{
			Reference: ArtifactReference{
				Digest:                "sha256:../../evil",
				SizeBytes:             1,
				ContentType:           "text/plain",
				DataClass:             DataClassSpecText,
				ProvenanceReceiptHash: testDigest("1"),
			},
			CreatedAt:         time.Now().UTC(),
			CreatedByRole:     "workspace",
			StorageProtection: "encrypted_at_rest_default",
		}},
	}
	backupPath := filepath.Join(t.TempDir(), "backup.json")
	if err := store.storeIO.writeBackup(backupPath, manifest); err != nil {
		t.Fatalf("write backup error: %v", err)
	}
	signature, err := computeBackupSignature(manifest, store.state.BackupHMACKey)
	if err != nil {
		t.Fatalf("compute signature error: %v", err)
	}
	if err := store.storeIO.writeBackupSignature(backupSignaturePath(backupPath), signature); err != nil {
		t.Fatalf("write signature error: %v", err)
	}
	restoreStore := newTestStore(t)
	err = restoreStore.RestoreBackup(backupPath)
	if err != ErrInvalidDigest {
		t.Fatalf("RestoreBackup error = %v, want %v", err, ErrInvalidDigest)
	}
}

func TestRestoreRejectsMissingBackupSignature(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"}); err != nil {
		t.Fatalf("Put error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.json")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup error: %v", err)
	}
	if err := os.Remove(backupSignaturePath(backupPath)); err != nil {
		t.Fatalf("remove signature error: %v", err)
	}
	restoreStore := newTestStore(t)
	err := restoreStore.RestoreBackup(backupPath)
	if err != ErrBackupSignatureMissing {
		t.Fatalf("RestoreBackup error = %v, want %v", err, ErrBackupSignatureMissing)
	}
}

func TestRestoreRejectsTamperedBackupSignature(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"}); err != nil {
		t.Fatalf("Put error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.json")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup error: %v", err)
	}
	b, err := os.ReadFile(backupSignaturePath(backupPath))
	if err != nil {
		t.Fatalf("read signature error: %v", err)
	}
	sig := BackupSignature{}
	if err := json.Unmarshal(b, &sig); err != nil {
		t.Fatalf("unmarshal signature error: %v", err)
	}
	sig.HMACSHA256 = strings.Repeat("0", len(sig.HMACSHA256))
	b, err = json.MarshalIndent(sig, "", "  ")
	if err != nil {
		t.Fatalf("marshal tampered signature error: %v", err)
	}
	if err := os.WriteFile(backupSignaturePath(backupPath), b, 0o600); err != nil {
		t.Fatalf("write tampered signature error: %v", err)
	}
	restoreStore := newTestStore(t)
	err = restoreStore.RestoreBackup(backupPath)
	if err != ErrBackupSignatureInvalid {
		t.Fatalf("RestoreBackup error = %v, want %v", err, ErrBackupSignatureInvalid)
	}
}

func TestBackupFilesArePrivateByDefault(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"}); err != nil {
		t.Fatalf("Put error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.json")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup error: %v", err)
	}
	assertMode(t, backupPath, 0o600)
	assertMode(t, backupSignaturePath(backupPath), 0o600)
}

func TestBackupRestorePreservesPolicyDecisionState(t *testing.T) {
	store := newTestStore(t)
	rec := basePolicyDecisionRecord("run-backup-policy", map[string]any{"precedence": "invariants_first"})
	if err := store.RecordPolicyDecision(rec); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	storedDecision, ok := firstPolicyDecisionRecord(store)
	if !ok {
		t.Fatal("policy decision missing from source state")
	}

	backupPath := filepath.Join(t.TempDir(), "backup-policy.json")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup returned error: %v", err)
	}

	restoreStore := newTestStore(t)
	if err := restoreStore.RestoreBackup(backupPath); err != nil {
		t.Fatalf("RestoreBackup returned error: %v", err)
	}

	restoredDecision, ok := restoreStore.state.PolicyDecisions[storedDecision.Digest]
	if !ok {
		t.Fatalf("restored policy decision %q missing", storedDecision.Digest)
	}
	if restoredDecision.RunID != rec.RunID {
		t.Fatalf("restored run_id = %q, want %q", restoredDecision.RunID, rec.RunID)
	}
	refs := restoreStore.PolicyDecisionRefsForRun(rec.RunID)
	if len(refs) != 1 || refs[0] != storedDecision.Digest {
		t.Fatalf("restored PolicyDecisionRefsForRun = %v, want [%s]", refs, storedDecision.Digest)
	}
}
