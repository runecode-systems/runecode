package secretsd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var replaceFileLocksMu sync.Mutex
var replaceFileLocks = map[string]*sync.Mutex{}

func (s *Service) persistState() error {
	b, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(s.root, stateFileName), append(b, '\n'), 0o600)
}

func writeFileAtomic(path string, b []byte, mode os.FileMode) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := tmpFile.Name()
	if _, err := tmpFile.Write(b); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := replaceFile(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func replaceFile(src, dst string) error {
	release := lockReplaceTarget(dst)
	defer release()
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		return err
	}
	backup, err := createReplaceBackup(dst)
	if err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		if restoreErr := restoreReplaceBackup(backup, dst); restoreErr != nil {
			return fmt.Errorf("replace %s: rename failed: %w (restore backup: %v)", dst, err, restoreErr)
		}
		return err
	}
	if backup == "" {
		return nil
	}
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return nil
	}
	return nil
}

func lockReplaceTarget(path string) func() {
	key := filepath.Clean(path)
	replaceFileLocksMu.Lock()
	mu, ok := replaceFileLocks[key]
	if !ok {
		mu = &sync.Mutex{}
		replaceFileLocks[key] = mu
	}
	replaceFileLocksMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

func createReplaceBackup(dst string) (string, error) {
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

func restoreReplaceBackup(backup, dst string) error {
	if backup == "" {
		return nil
	}
	if _, err := os.Stat(backup); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(backup, dst)
}
