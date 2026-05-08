package brokerapi

import (
	"errors"
	"fmt"
	"os"
)

type brokerOwnedFileSnapshot struct {
	path     string
	existed  bool
	contents []byte
	mode     os.FileMode
}

func captureBrokerOwnedFileSnapshot(path string) (brokerOwnedFileSnapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return brokerOwnedFileSnapshot{path: path}, nil
		}
		return brokerOwnedFileSnapshot{}, fmt.Errorf("stat broker-owned write target: %w", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return brokerOwnedFileSnapshot{}, fmt.Errorf("read broker-owned write target snapshot: %w", err)
	}
	return brokerOwnedFileSnapshot{path: path, existed: true, contents: contents, mode: info.Mode()}, nil
}

func rollbackBrokerOwnedFileSnapshots(snapshots []brokerOwnedFileSnapshot) error {
	var joined error
	for i := len(snapshots) - 1; i >= 0; i-- {
		if err := rollbackBrokerOwnedFileSnapshot(snapshots[i]); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func rollbackBrokerOwnedFileSnapshot(snapshot brokerOwnedFileSnapshot) error {
	if !snapshot.existed {
		if err := os.Remove(snapshot.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove broker-owned write target during rollback: %w", err)
		}
		return nil
	}
	if err := writeBrokerOwnedDraftPromoteFile(snapshot.path, snapshot.contents, snapshot.mode); err != nil {
		return fmt.Errorf("restore broker-owned write target during rollback: %w", err)
	}
	return nil
}

func joinBrokerOwnedRollbackError(cause error, rollbackErr error) error {
	if rollbackErr == nil {
		return cause
	}
	return errors.Join(cause, fmt.Errorf("broker-owned write rollback failed: %w", rollbackErr))
}
