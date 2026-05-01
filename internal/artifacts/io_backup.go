package artifacts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	backupBundleManifestFile  = "manifest.json"
	backupBundleSignatureFile = "signature.json"
	backupBundleBlobsDir      = "blobs"
	backupBundleSHA256Dir     = "sha256"
)

func (s *storeIO) writeBackup(path string, manifest BackupManifest) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, backupBundleManifestFile), b, 0o600)
}

func (s *storeIO) writeBackupSignature(path string, signature BackupSignature) error {
	b, err := json.MarshalIndent(signature, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func (s *storeIO) readBackup(path string) (BackupManifest, error) {
	manifest := BackupManifest{}
	bundlePath := normalizeBackupBundlePath(path)
	b, err := os.ReadFile(filepath.Join(bundlePath, backupBundleManifestFile))
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		return manifest, err
	}
	if manifest.Schema != "runecode.backup.artifacts.v1" {
		return manifest, fmt.Errorf("unsupported backup schema")
	}
	return manifest, nil
}

func (s *storeIO) readBackupSignature(path string) (BackupSignature, error) {
	signature := BackupSignature{}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return signature, ErrBackupSignatureMissing
		}
		return signature, err
	}
	if err := json.Unmarshal(b, &signature); err != nil {
		return signature, err
	}
	return signature, nil
}

func (s *storeIO) writeBackupBlobs(path string, records []ArtifactRecord) error {
	bundleBlobDir := filepath.Join(normalizeBackupBundlePath(path), backupBundleBlobsDir, backupBundleSHA256Dir)
	if err := os.MkdirAll(bundleBlobDir, 0o700); err != nil {
		return err
	}
	writtenBlobs := make([]string, 0, len(records))
	for _, rec := range records {
		blobPath, err := s.writeBackupBlob(bundleBlobDir, rec)
		if err != nil {
			cleanupBackupBlobPaths(writtenBlobs)
			return err
		}
		writtenBlobs = append(writtenBlobs, blobPath)
	}
	return nil
}

func (s *storeIO) writeBackupBlob(bundleBlobDir string, rec ArtifactRecord) (string, error) {
	hexDigest, ok := trimSHA256Digest(rec.Reference.Digest)
	if !ok {
		return "", ErrInvalidDigest
	}
	src, err := s.validatedBlobPath(rec.Reference.Digest)
	if err != nil {
		return "", err
	}
	dst := filepath.Join(bundleBlobDir, hexDigest)
	if err := copyBackupBlob(src, dst, rec.Reference.Digest, rec.Reference.SizeBytes); err != nil {
		return "", err
	}
	return dst, nil
}

func copyBackupBlob(src, dst, expectedDigest string, expectedSize int64) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	cleanupDst := true
	defer func() {
		if cleanupDst {
			_ = os.Remove(dst)
		}
	}()
	h := newDigestWriter()
	written, copyErr := io.Copy(io.MultiWriter(out, h), in)
	closeErr := out.Close()
	if copyErr != nil {
		if closeErr != nil {
			return errors.Join(copyErr, closeErr)
		}
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written != expectedSize {
		return fmt.Errorf("backup size mismatch for %s", expectedDigest)
	}
	if h.identity() != expectedDigest {
		return fmt.Errorf("backup digest mismatch for %s", expectedDigest)
	}
	cleanupDst = false
	return nil
}

func cleanupBackupBlobPaths(paths []string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func (s *storeIO) restoreBackupBlobs(path string, records []ArtifactRecord) error {
	_, err := s.restoreBackupBlobsStaged(path, records)
	if err != nil {
		return err
	}
	return nil
}

func (s *storeIO) restoreBackupBlobsStaged(path string, records []ArtifactRecord) ([]string, error) {
	addedDigests := make([]string, 0, len(records))
	bundlePath := normalizeBackupBundlePath(path)
	for _, rec := range records {
		created, err := s.restoreBackupBlobStaged(bundlePath, rec)
		if err != nil {
			rollbackRestoredBlobDigests(s, addedDigests)
			return nil, err
		}
		if created {
			addedDigests = append(addedDigests, rec.Reference.Digest)
		}
	}
	return addedDigests, nil
}

func (s *storeIO) restoreBackupBlobStaged(bundlePath string, rec ArtifactRecord) (bool, error) {
	bundleBlobPath, err := backupBundleBlobPath(bundlePath, rec.Reference.Digest)
	if err != nil {
		return false, err
	}
	in, err := openRegularFile(bundleBlobPath)
	if err != nil {
		return false, err
	}
	tmpPath, digest, size, err := s.streamToTempBlob(in)
	closeErr := in.Close()
	if err != nil {
		return false, err
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return false, closeErr
	}
	if err := validateStreamedBlob(tmpPath, digest, size, rec.Reference.Digest, rec.Reference.SizeBytes); err != nil {
		return false, err
	}
	return s.persistBlobFromTempFile(tmpPath, rec.Reference.Digest)
}

func backupBundleBlobPath(bundlePath, digest string) (string, error) {
	hexDigest, ok := trimSHA256Digest(digest)
	if !ok {
		return "", ErrInvalidDigest
	}
	return filepath.Join(bundlePath, backupBundleBlobsDir, backupBundleSHA256Dir, hexDigest), nil
}

func openRegularFile(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("backup blob path is not a regular file: %s", path)
	}
	return os.Open(path)
}

func validateStreamedBlob(tmpPath, digest string, size int64, expectedDigest string, expectedSize int64) error {
	if digest != expectedDigest {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("backup digest mismatch for %s", expectedDigest)
	}
	if size != expectedSize {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("backup size mismatch for %s", expectedDigest)
	}
	return nil
}

func rollbackRestoredBlobDigests(s *storeIO, digests []string) {
	for _, digest := range digests {
		blobPath, err := s.validatedBlobPath(digest)
		if err != nil {
			continue
		}
		_ = s.removeBlob(blobPath)
	}
}

func trimSHA256Digest(digest string) (string, bool) {
	if !isValidDigest(digest) {
		return "", false
	}
	return strings.TrimPrefix(digest, "sha256:"), true
}
