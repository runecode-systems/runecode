package artifacts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type stateFileStamp struct {
	exists      bool
	modTime     time.Time
	size        int64
	contentHash [32]byte
}

var replaceStateFileLocksMu sync.Mutex
var replaceStateFileLocks = map[string]*sync.Mutex{}

func (s stateFileStamp) differsFrom(other stateFileStamp) bool {
	if s.exists != other.exists {
		return true
	}
	if !s.exists {
		return false
	}
	if !s.modTime.Equal(other.modTime) {
		return true
	}
	if s.size != other.size {
		return true
	}
	return s.contentHash != other.contentHash
}

func (s *storeIO) stateFileStamp() (stateFileStamp, error) {
	stamp := stateFileStamp{}
	info, err := os.Stat(s.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return stamp, nil
		}
		return stateFileStamp{}, err
	}
	b, err := os.ReadFile(s.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return stamp, nil
		}
		return stateFileStamp{}, err
	}
	stamp.exists = true
	stamp.modTime = info.ModTime().UTC()
	stamp.size = info.Size()
	stamp.contentHash = sha256.Sum256(b)
	return stamp, nil
}

func (s *storeIO) saveStateFile(state StoreState) (stateFileStamp, error) {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return stateFileStamp{}, err
	}
	if err := writeFileAtomic(s.statePath, b, 0o600); err != nil {
		return stateFileStamp{}, err
	}
	info, err := os.Stat(s.statePath)
	if err != nil {
		return stateFileStamp{}, err
	}
	return stateFileStamp{
		exists:      true,
		modTime:     info.ModTime().UTC(),
		size:        int64(len(b)),
		contentHash: sha256.Sum256(b),
	}, nil
}

func writeFileAtomic(path string, b []byte, mode os.FileMode) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmpFile.Write(b); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return replaceFile(tmpPath, path)
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
		return err
	}
	return nil
}

func lockReplaceTarget(path string) func() {
	key := filepath.Clean(path)
	replaceStateFileLocksMu.Lock()
	mu, ok := replaceStateFileLocks[key]
	if !ok {
		mu = &sync.Mutex{}
		replaceStateFileLocks[key] = mu
	}
	replaceStateFileLocksMu.Unlock()
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
