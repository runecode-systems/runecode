package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const (
	segmentsDirName            = "segments"
	sidecarDirName             = "sidecar"
	sealsDirName               = "segment-seals"
	receiptsDirName            = "receipts"
	externalAnchorEvidenceDir  = "external-anchor-evidence"
	externalAnchorSidecarsDir  = "external-anchor-sidecars"
	verificationReportsDirName = "verification-reports"
	indexDirName               = "index"
	stateFileName              = "state.json"
	auditEvidenceIndexFileName = "audit-evidence-index.json"
)

var renameFile = os.Rename
var removeFile = os.Remove

var replaceFileLocksMu sync.Mutex
var replaceFileLocks = map[string]*sync.Mutex{}

func (l *Ledger) ensureLayout() error {
	paths := []string{
		l.rootDir,
		filepath.Join(l.rootDir, segmentsDirName),
		filepath.Join(l.rootDir, sidecarDirName),
		filepath.Join(l.rootDir, sidecarDirName, sealsDirName),
		filepath.Join(l.rootDir, sidecarDirName, receiptsDirName),
		filepath.Join(l.rootDir, sidecarDirName, externalAnchorEvidenceDir),
		filepath.Join(l.rootDir, sidecarDirName, externalAnchorSidecarsDir),
		filepath.Join(l.rootDir, sidecarDirName, verificationReportsDirName),
		filepath.Join(l.rootDir, indexDirName),
	}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
		if err := os.Chmod(path, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledger) loadState() (ledgerState, error) {
	path := filepath.Join(l.rootDir, stateFileName)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ledgerState{}, err
		}
		return ledgerState{}, err
	}
	state := ledgerState{}
	if err := readJSONFile(path, &state); err != nil {
		return ledgerState{}, err
	}
	return state, nil
}

func (l *Ledger) saveState(state ledgerState) error {
	state.SchemaVersion = stateSchemaVersion
	return writeCanonicalJSONFile(filepath.Join(l.rootDir, stateFileName), state)
}

func writeCanonicalJSONFile(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmp)
	}()
	if _, err := tmpFile.Write(canonical); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := replaceFile(tmp, path); err != nil {
		return err
	}
	return nil
}

func replaceFile(src, dst string) error {
	release := lockReplaceTarget(dst)
	defer release()

	if err := renameFile(src, dst); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		return err
	}
	backup, err := createReplaceBackup(dst)
	if err != nil {
		return err
	}
	if err := renameFile(src, dst); err != nil {
		if restoreErr := restoreReplaceBackup(backup, dst); restoreErr != nil {
			return fmt.Errorf("replace %s: rename failed: %w (restore backup: %v)", dst, err, restoreErr)
		}
		return err
	}
	if backup == "" {
		return nil
	}
	if err := removeFile(backup); err != nil && !os.IsNotExist(err) {
		return err
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
	if err := removeFile(backup); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := renameFile(dst, backup); err != nil {
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
	if err := removeFile(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return renameFile(backup, dst)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o700)
}

func readJSONFile(path string, target any) error {
	release := lockReplaceTarget(path)
	defer release()

	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
