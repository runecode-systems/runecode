package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const (
	segmentsDirName            = "segments"
	sidecarDirName             = "sidecar"
	sealsDirName               = "segment-seals"
	receiptsDirName            = "receipts"
	verificationReportsDirName = "verification-reports"
	indexDirName               = "index"
	stateFileName              = "state.json"
	indexFileName              = "timeline-index.json"
)

func (l *Ledger) ensureLayout() error {
	paths := []string{
		l.rootDir,
		filepath.Join(l.rootDir, segmentsDirName),
		filepath.Join(l.rootDir, sidecarDirName),
		filepath.Join(l.rootDir, sidecarDirName, sealsDirName),
		filepath.Join(l.rootDir, sidecarDirName, receiptsDirName),
		filepath.Join(l.rootDir, sidecarDirName, verificationReportsDirName),
		filepath.Join(l.rootDir, indexDirName),
	}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
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
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, canonical, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return nil
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func readJSONFile(path string, target any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
