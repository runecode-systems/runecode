package artifacts

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const backupHMACKeyEnv = "RUNE_BACKUP_HMAC_KEY"

const backupHMACKeyFileEnv = "RUNE_BACKUP_HMAC_KEY_FILE"

func ensureBackupKey(state StoreState) (StoreState, error) {
	if configured := strings.TrimSpace(os.Getenv(backupHMACKeyEnv)); configured != "" {
		state.BackupHMACKey = configured
		return state, nil
	}
	persisted, err := persistentBackupKey(state.BackupHMACKey)
	if err != nil {
		return state, err
	}
	state.BackupHMACKey = persisted
	return state, nil
}

func persistentBackupKey(existing string) (string, error) {
	path, ok, err := persistentBackupKeyPath()
	if err != nil {
		return "", err
	}
	if !ok {
		if strings.TrimSpace(existing) != "" {
			return strings.TrimSpace(existing), nil
		}
		return randomBackupKey()
	}
	key, found, err := readPersistentBackupKey(path)
	if err != nil {
		return "", err
	}
	if found {
		return key, nil
	}
	seed := strings.TrimSpace(existing)
	if seed == "" {
		seed, err = randomBackupKey()
		if err != nil {
			return "", err
		}
	}
	persisted, err := writePersistentBackupKey(path, seed)
	if err != nil {
		return "", err
	}
	return persisted, nil
}

func persistentBackupKeyPath() (string, bool, error) {
	if explicit := strings.TrimSpace(os.Getenv(backupHMACKeyFileEnv)); explicit != "" {
		return filepath.Clean(explicit), true, nil
	}
	if configDir, err := os.UserConfigDir(); err == nil && strings.TrimSpace(configDir) != "" {
		return filepath.Join(configDir, "runecode", "backup-hmac-key"), true, nil
	}
	if cacheDir, err := os.UserCacheDir(); err == nil && strings.TrimSpace(cacheDir) != "" {
		return filepath.Join(cacheDir, "runecode", "backup-hmac-key"), true, nil
	}
	return "", false, nil
}

func readPersistentBackupKey(path string) (string, bool, error) {
	b, err := os.ReadFile(path)
	if err == nil {
		key := strings.TrimSpace(string(b))
		if key == "" {
			return "", false, fmt.Errorf("persistent backup key file is empty")
		}
		return key, true, nil
	}
	if os.IsNotExist(err) {
		return "", false, nil
	}
	return "", false, err
}

func writePersistentBackupKey(path string, key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("persistent backup key is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "backup-key-*.tmp")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if _, err := tmp.WriteString(key + "\n"); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		if persisted, found, readErr := readPersistentBackupKey(path); readErr == nil && found {
			return persisted, nil
		}
		return "", err
	}
	return key, nil
}

func backupSignaturePath(path string) string {
	bundlePath := normalizeBackupBundlePath(path)
	return filepath.Join(bundlePath, backupBundleSignatureFile)
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

func normalizeBackupBundlePath(path string) string {
	cleaned := sanitizeBackupPath(path)
	if filepath.Base(cleaned) == backupBundleManifestFile {
		return filepath.Dir(cleaned)
	}
	return cleaned
}

func randomBackupKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "hmac-sha256:" + hex.EncodeToString(b), nil
}
