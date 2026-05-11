package brokerapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

type brokerOwnedMutationWriteIntent struct {
	targetAbsolutePath string
	targetRelativePath string
	writeMode          string
	contents           []byte
	expectedDigest     string
	mode               os.FileMode
}

type brokerOwnedPreparedMutationWrite struct {
	intent   brokerOwnedMutationWriteIntent
	snapshot brokerOwnedFileSnapshot
}

var brokerOwnedMutationPostWriteHookForTest func(path string) error

func prepareBrokerOwnedMutationWrites(intents []brokerOwnedMutationWriteIntent) ([]brokerOwnedPreparedMutationWrite, error) {
	prepared := make([]brokerOwnedPreparedMutationWrite, 0, len(intents))
	seenTargets := map[string]struct{}{}
	for _, intent := range intents {
		normalized, err := normalizeBrokerOwnedMutationIntent(intent)
		if err != nil {
			return nil, err
		}
		trimmedTarget := normalized.targetAbsolutePath
		if _, exists := seenTargets[trimmedTarget]; exists {
			return nil, fmt.Errorf("broker-owned mutation target %q is duplicated", trimmedTarget)
		}
		seenTargets[trimmedTarget] = struct{}{}
		if err := validateBrokerOwnedMutationWriteMode(normalized); err != nil {
			return nil, err
		}
		snapshot, err := captureBrokerOwnedFileSnapshot(trimmedTarget)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, brokerOwnedPreparedMutationWrite{intent: normalized, snapshot: snapshot})
	}
	return prepared, nil
}

func normalizeBrokerOwnedMutationIntent(intent brokerOwnedMutationWriteIntent) (brokerOwnedMutationWriteIntent, error) {
	trimmedTarget := filepath.Clean(strings.TrimSpace(intent.targetAbsolutePath))
	if trimmedTarget == "" {
		return brokerOwnedMutationWriteIntent{}, fmt.Errorf("broker-owned mutation target path is required")
	}
	if strings.TrimSpace(intent.expectedDigest) == "" {
		return brokerOwnedMutationWriteIntent{}, fmt.Errorf("broker-owned mutation expected digest is required for %q", trimmedTarget)
	}
	if intent.mode == 0 {
		intent.mode = 0o644
	}
	intent.targetAbsolutePath = trimmedTarget
	intent.contents = append([]byte(nil), intent.contents...)
	return intent, nil
}

func validateBrokerOwnedMutationWriteMode(intent brokerOwnedMutationWriteIntent) error {
	if strings.TrimSpace(intent.writeMode) == "" {
		return nil
	}
	return validateApprovedImplementationWriteMode(intent.targetAbsolutePath, intent.writeMode)
}

func finalizeBrokerOwnedMutationWrites(prepared []brokerOwnedPreparedMutationWrite, finalize func() error) error {
	if err := writePreparedBrokerOwnedMutationWrites(prepared); err != nil {
		return err
	}
	if finalize == nil {
		return nil
	}
	if err := finalize(); err != nil {
		return joinBrokerOwnedRollbackError(err, rollbackPreparedBrokerOwnedMutationWrites(prepared))
	}
	return nil
}

func writePreparedBrokerOwnedMutationWrites(prepared []brokerOwnedPreparedMutationWrite) error {
	for _, write := range prepared {
		if err := writeBrokerOwnedMutationFile(write.intent); err != nil {
			return joinBrokerOwnedRollbackError(err, rollbackPreparedBrokerOwnedMutationWrites(prepared))
		}
	}
	return nil
}

func rollbackPreparedBrokerOwnedMutationWrites(prepared []brokerOwnedPreparedMutationWrite) error {
	snapshots := make([]brokerOwnedFileSnapshot, 0, len(prepared))
	for _, write := range prepared {
		snapshots = append(snapshots, write.snapshot)
	}
	return rollbackBrokerOwnedFileSnapshots(snapshots)
}

func writeBrokerOwnedMutationFile(intent brokerOwnedMutationWriteIntent) error {
	if err := writeBrokerOwnedMutationFileUnverified(intent); err != nil {
		return err
	}
	if brokerOwnedMutationPostWriteHookForTest != nil {
		if err := brokerOwnedMutationPostWriteHookForTest(intent.targetAbsolutePath); err != nil {
			return fmt.Errorf("broker-owned mutation post-write hook: %w", err)
		}
	}
	if err := verifyBrokerOwnedMutationWrite(intent); err != nil {
		return err
	}
	return nil
}

func writeBrokerOwnedMutationFileUnverified(intent brokerOwnedMutationWriteIntent) error {
	switch strings.TrimSpace(intent.writeMode) {
	case "":
		return writeBrokerOwnedDraftPromoteFile(intent.targetAbsolutePath, intent.contents, intent.mode)
	case "create":
		return writeBrokerOwnedCreateFile(intent.targetAbsolutePath, intent.contents, intent.mode)
	case "update":
		return writeBrokerOwnedDraftPromoteFile(intent.targetAbsolutePath, intent.contents, intent.mode)
	default:
		return fmt.Errorf("broker-owned mutation write_mode %q is unsupported", strings.TrimSpace(intent.writeMode))
	}
}

func verifyBrokerOwnedMutationWrite(intent brokerOwnedMutationWriteIntent) error {
	payload, err := os.ReadFile(intent.targetAbsolutePath)
	if err != nil {
		return fmt.Errorf("read broker-owned mutation target after write: %w", err)
	}
	if got := artifacts.DigestBytes(payload); got != strings.TrimSpace(intent.expectedDigest) {
		path := strings.TrimSpace(intent.targetRelativePath)
		if path == "" {
			path = strings.TrimSpace(intent.targetAbsolutePath)
		}
		return fmt.Errorf("broker-owned mutation post-write digest drift for %q", path)
	}
	return nil
}

func writeBrokerOwnedCreateFile(path string, contents []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create approved implementation target parent: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create approved implementation target: %w", err)
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("chmod approved implementation target: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write approved implementation target: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close approved implementation target: %w", err)
	}
	return nil
}

func validateApprovedImplementationWriteMode(targetPath, writeMode string) error {
	switch strings.TrimSpace(writeMode) {
	case "create":
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("approved implementation create target already exists: %s", targetPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat approved implementation create target: %w", err)
		}
	case "update":
		if _, err := os.Stat(targetPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("approved implementation update target does not exist: %s", targetPath)
			}
			return fmt.Errorf("stat approved implementation update target: %w", err)
		}
	default:
		return fmt.Errorf("approved implementation write_mode %q is unsupported", strings.TrimSpace(writeMode))
	}
	return nil
}
