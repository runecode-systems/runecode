package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRestoreBackupAcceptsManifestPathInput(t *testing.T) {
	t.Setenv(backupHMACKeyEnv, "")
	t.Setenv(backupHMACKeyFileEnv, "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("a"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup-manifest-input")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup returned error: %v", err)
	}

	restoreStore := newTestStore(t)
	manifestPath := filepath.Join(backupPath, backupBundleManifestFile)
	if err := restoreStore.RestoreBackup(manifestPath); err != nil {
		t.Fatalf("RestoreBackup(manifest path) returned error: %v", err)
	}
	if _, err := restoreStore.Head(ref.Digest); err != nil {
		t.Fatalf("Head returned error after manifest-path restore: %v", err)
	}
}

func TestPersistentBackupKeyReturnsCanonicalPersistedKeyWhenFileExists(t *testing.T) {
	t.Setenv(backupHMACKeyEnv, "")
	t.Setenv(backupHMACKeyFileEnv, filepath.Join(t.TempDir(), "backup-hmac-key"))
	path, ok, err := persistentBackupKeyPath()
	if err != nil {
		t.Fatalf("persistentBackupKeyPath returned error: %v", err)
	}
	if !ok {
		t.Fatal("persistentBackupKeyPath reported unavailable path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte("persisted-key\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	key, err := persistentBackupKey("generated-key")
	if err != nil {
		t.Fatalf("persistentBackupKey returned error: %v", err)
	}
	if key != "persisted-key" {
		t.Fatalf("persistentBackupKey = %q, want %q", key, "persisted-key")
	}
}

func TestExportBackupBundlesReferencedBlobs(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup-bundle")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup error: %v", err)
	}
	hexDigest, ok := trimSHA256Digest(ref.Digest)
	if !ok {
		t.Fatalf("digest %q failed validation", ref.Digest)
	}
	bundleBlob := filepath.Join(backupPath, backupBundleBlobsDir, backupBundleSHA256Dir, hexDigest)
	b, err := os.ReadFile(bundleBlob)
	if err != nil {
		t.Fatalf("read bundled blob error: %v", err)
	}
	if digestBytes(b) != ref.Digest {
		t.Fatalf("bundled blob digest = %q, want %q", digestBytes(b), ref.Digest)
	}
}

func TestExportBackupRejectsCorruptedStoredBlobWithMatchingSize(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	rec, err := store.Head(ref.Digest)
	if err != nil {
		t.Fatalf("Head error: %v", err)
	}
	if err := os.WriteFile(rec.BlobPath, []byte("tamperd"), 0o600); err != nil {
		t.Fatalf("WriteFile tampered blob error: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup-corrupted-source-blob")
	err = store.ExportBackup(backupPath)
	if err == nil || !strings.Contains(err.Error(), "backup digest mismatch") {
		t.Fatalf("ExportBackup error = %v, want backup digest mismatch", err)
	}
}

func TestExportBackupRemovesBundledBlobWhenBlobCopyFails(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	rec, err := store.Head(ref.Digest)
	if err != nil {
		t.Fatalf("Head error: %v", err)
	}
	if err := os.WriteFile(rec.BlobPath, []byte("tamperd"), 0o600); err != nil {
		t.Fatalf("WriteFile tampered blob error: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup-bundle-copy-failure-cleanup")
	err = store.ExportBackup(backupPath)
	if err == nil || !strings.Contains(err.Error(), "backup digest mismatch") {
		t.Fatalf("ExportBackup error = %v, want backup digest mismatch", err)
	}

	bundleBlob := bundledBlobPathForDigest(t, backupPath, ref.Digest)
	if _, statErr := os.Stat(bundleBlob); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("bundled blob stat error = %v, want not-exist", statErr)
	}
}

func TestExportBackupRemovesEarlierBundledBlobsWhenLaterBlobFails(t *testing.T) {
	store := newTestStore(t)
	first, err := store.Put(PutRequest{Payload: []byte("payload-1"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put(first) error: %v", err)
	}
	second, err := store.Put(PutRequest{Payload: []byte("payload-2"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("2"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put(second) error: %v", err)
	}
	secondRec, err := store.Head(second.Digest)
	if err != nil {
		t.Fatalf("Head(second) error: %v", err)
	}
	if err := os.WriteFile(secondRec.BlobPath, []byte("tamperd-2"), 0o600); err != nil {
		t.Fatalf("WriteFile tampered second blob error: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup-bundle-multi-copy-failure-cleanup")
	err = store.ExportBackup(backupPath)
	if err == nil || !strings.Contains(err.Error(), "backup digest mismatch") {
		t.Fatalf("ExportBackup error = %v, want backup digest mismatch", err)
	}

	firstBundleBlob := bundledBlobPathForDigest(t, backupPath, first.Digest)
	if _, statErr := os.Stat(firstBundleBlob); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("first bundled blob stat error = %v, want not-exist", statErr)
	}
	secondBundleBlob := bundledBlobPathForDigest(t, backupPath, second.Digest)
	if _, statErr := os.Stat(secondBundleBlob); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("second bundled blob stat error = %v, want not-exist", statErr)
	}
}

func TestRestoreRejectsMissingBundledBlobAndKeepsStoreUnchanged(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup-missing-bundle-blob")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup error: %v", err)
	}
	removeBundledBlobForDigest(t, backupPath, ref.Digest)

	restoreStore := newTestStore(t)
	err = restoreStore.RestoreBackup(backupPath)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("RestoreBackup error = %v, want not-exist", err)
	}
	if len(restoreStore.state.Artifacts) != 0 {
		t.Fatalf("restored artifacts count = %d, want 0", len(restoreStore.state.Artifacts))
	}
	assertNoRestoreBackupAuditEvent(t, restoreStore)
}

func TestRestoreRejectsBundledBlobDigestMismatch(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup-digest-mismatch")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup error: %v", err)
	}
	bundleBlob := bundledBlobPathForDigest(t, backupPath, ref.Digest)
	if err := os.WriteFile(bundleBlob, []byte("tampered"), 0o600); err != nil {
		t.Fatalf("write tampered bundled blob error: %v", err)
	}

	restoreStore := newTestStore(t)
	err = restoreStore.RestoreBackup(backupPath)
	if err == nil || !strings.Contains(err.Error(), "backup digest mismatch") {
		t.Fatalf("RestoreBackup error = %v, want backup digest mismatch", err)
	}
	if len(restoreStore.state.Artifacts) != 0 {
		t.Fatalf("restored artifacts count = %d, want 0", len(restoreStore.state.Artifacts))
	}
}

func TestRestoreRejectsBundledBlobSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliable on Windows")
	}
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("payload"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup-symlink-blob")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup error: %v", err)
	}
	bundleBlob := bundledBlobPathForDigest(t, backupPath, ref.Digest)
	if err := os.Remove(bundleBlob); err != nil {
		t.Fatalf("remove bundled blob error: %v", err)
	}
	target := filepath.Join(t.TempDir(), "outside-source")
	if err := os.WriteFile(target, []byte("payload"), 0o600); err != nil {
		t.Fatalf("WriteFile target error: %v", err)
	}
	if err := os.Symlink(target, bundleBlob); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	restoreStore := newTestStore(t)
	err = restoreStore.RestoreBackup(backupPath)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("RestoreBackup error = %v, want not a regular file", err)
	}
}

func bundledBlobPathForDigest(t *testing.T, backupPath, digest string) string {
	t.Helper()
	hexDigest, ok := trimSHA256Digest(digest)
	if !ok {
		t.Fatalf("digest %q failed validation", digest)
	}
	return filepath.Join(backupPath, backupBundleBlobsDir, backupBundleSHA256Dir, hexDigest)
}

func removeBundledBlobForDigest(t *testing.T, backupPath, digest string) {
	t.Helper()
	bundleBlob := bundledBlobPathForDigest(t, backupPath, digest)
	if err := os.Remove(bundleBlob); err != nil {
		t.Fatalf("remove bundled blob error: %v", err)
	}
}

func assertNoRestoreBackupAuditEvent(t *testing.T, store *Store) {
	t.Helper()
	events, err := store.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents error: %v", err)
	}
	for _, event := range events {
		if event.Type != "artifact_retention_action" {
			continue
		}
		if action, _ := event.Details["action"].(string); action == "restore_backup" {
			t.Fatalf("unexpected restore audit event on failed restore")
		}
	}
}
