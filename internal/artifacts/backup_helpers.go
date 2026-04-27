package artifacts

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

const backupHMACKeyEnv = "RUNE_BACKUP_HMAC_KEY"

func ensureBackupKey(state StoreState) (StoreState, error) {
	if state.BackupHMACKey != "" {
		return state, nil
	}
	if configured := os.Getenv(backupHMACKeyEnv); configured != "" {
		state.BackupHMACKey = configured
		return state, nil
	}
	randomKey, err := randomBackupKey()
	if err != nil {
		return state, err
	}
	state.BackupHMACKey = randomKey
	return state, nil
}

func backupSignaturePath(manifestPath string) string {
	return manifestPath + ".sig"
}

func computeBackupSignature(manifest BackupManifest, key string) (BackupSignature, error) {
	canonical, err := canonicalBackupManifestBytes(manifest)
	if err != nil {
		return BackupSignature{}, err
	}
	manifestHash := sha256.Sum256(canonical)
	h := hmac.New(sha256.New, []byte(key))
	if _, err := h.Write(canonical); err != nil {
		return BackupSignature{}, err
	}
	mac := h.Sum(nil)
	return BackupSignature{
		Schema:         "runecode.backup.signature.v1",
		ManifestSHA256: hex.EncodeToString(manifestHash[:]),
		HMACSHA256:     hex.EncodeToString(mac),
		KeyID:          backupKeyID(key),
		ExportedAt:     manifest.ExportedAt,
	}, nil
}

func verifyBackupSignature(manifest BackupManifest, signature BackupSignature, key string) error {
	if signature.Schema != "runecode.backup.signature.v1" {
		return ErrBackupSignatureInvalid
	}
	if signature.KeyID != backupKeyID(key) {
		return ErrBackupSignatureInvalid
	}
	expected, err := computeBackupSignature(manifest, key)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected.ManifestSHA256), []byte(signature.ManifestSHA256)) {
		return ErrBackupSignatureInvalid
	}
	if !hmac.Equal([]byte(expected.HMACSHA256), []byte(signature.HMACSHA256)) {
		return ErrBackupSignatureInvalid
	}
	return nil
}

func canonicalBackupManifestBytes(manifest BackupManifest) ([]byte, error) {
	b, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	return canonicalizeJSONBytes(b)
}

func backupKeyID(key string) string {
	h := sha256.Sum256([]byte(key))
	return "sha256:" + hex.EncodeToString(h[:16])
}

func sanitizeBackupPath(filePath string) string {
	return filepath.Clean(filePath)
}

func randomBackupKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "hmac-sha256:" + hex.EncodeToString(b), nil
}
