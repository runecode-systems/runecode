package brokerapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeApprovedImplementationFile(write approvedImplementationWorkspaceWrite) error {
	if strings.TrimSpace(write.writeMode) == "create" {
		return writeBrokerOwnedCreateFile(write.targetAbsolutePath, write.content, 0o644)
	}
	if err := validateApprovedImplementationWriteMode(write.targetAbsolutePath, write.writeMode); err != nil {
		return err
	}
	return writeBrokerOwnedDraftPromoteFile(write.targetAbsolutePath, write.content, 0o644)
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
