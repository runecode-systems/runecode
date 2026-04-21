package projectsubstrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writeCanonicalConfig(configPath, nextConfig string) error {
	configFile, tempPath, err := openAtomicConfigTemp(configPath)
	if err != nil {
		return fmt.Errorf("open canonical config: %w", err)
	}
	_, writeErr := configFile.WriteString(nextConfig)
	closeErr := configFile.Close()
	if writeErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("write canonical config: %w", writeErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close canonical config: %w", closeErr)
	}
	if err := replaceConfigFile(tempPath, configPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("rename canonical config: %w", err)
	}
	return nil
}

func openAtomicConfigTemp(configPath string) (*os.File, string, error) {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	configFile, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return nil, "", err
	}
	if err := configFile.Chmod(0o644); err != nil {
		tempPath := configFile.Name()
		_ = configFile.Close()
		_ = os.Remove(tempPath)
		return nil, "", err
	}
	return configFile, configFile.Name(), nil
}

var replaceConfigLocksMu sync.Mutex
var replaceConfigLocks = map[string]*sync.Mutex{}
var renameConfigFile = os.Rename
var removeConfigBackup = os.Remove
var writeCanonicalConfigFile = writeCanonicalConfig

func replaceConfigFile(src, dst string) error {
	release := lockReplaceConfigTarget(dst)
	defer release()
	if err := renameConfigFile(src, dst); err == nil {
		return nil
	}
	backup, err := createConfigBackup(dst)
	if err != nil {
		return err
	}
	if err := renameConfigFile(src, dst); err != nil {
		if restoreErr := restoreConfigBackup(backup, dst); restoreErr != nil {
			return fmt.Errorf("replace %s: rename failed: %w (restore backup: %v)", dst, err, restoreErr)
		}
		return err
	}
	if backup != "" {
		if err := removeConfigBackup(backup); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("replace %s: replacement applied but remove backup %s failed: %w", dst, backup, err)
		}
	}
	return nil
}

func lockReplaceConfigTarget(path string) func() {
	key := filepath.Clean(path)
	replaceConfigLocksMu.Lock()
	mu, ok := replaceConfigLocks[key]
	if !ok {
		mu = &sync.Mutex{}
		replaceConfigLocks[key] = mu
	}
	replaceConfigLocksMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

func createConfigBackup(dst string) (string, error) {
	if _, err := os.Stat(dst); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	backup := dst + ".bak"
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := os.Rename(dst, backup); err != nil {
		return "", err
	}
	return backup, nil
}

func restoreConfigBackup(backup, dst string) error {
	if backup == "" {
		return nil
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(backup, dst)
}
