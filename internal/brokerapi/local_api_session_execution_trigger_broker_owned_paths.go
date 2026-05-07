package brokerapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func normalizeBrokerOwnedRelativeTargetPath(targetRelativePath string) (string, error) {
	trimmed := strings.TrimSpace(targetRelativePath)
	if trimmed == "" {
		return "", nil
	}
	normalized := filepath.ToSlash(filepath.Clean(filepath.FromSlash(trimmed)))
	if normalized == "." {
		return "", nil
	}
	if normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("target path escapes repository root")
	}
	return normalized, nil
}

func pathWithinAllowedPrefix(target, allowed string) bool {
	target = strings.TrimSpace(target)
	allowed = strings.TrimSpace(allowed)
	if target == "" || allowed == "" {
		return false
	}
	if target == allowed {
		return true
	}
	return strings.HasPrefix(target, allowed+"/")
}

func brokerOwnedDraftPromoteTargetPath(repoRoot, targetRelativePath string) (string, error) {
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))
	if repoRoot == "" {
		return "", fmt.Errorf("repository root is required")
	}
	targetPath := filepath.Clean(filepath.Join(repoRoot, filepath.FromSlash(strings.TrimSpace(targetRelativePath))))
	rel, err := filepath.Rel(repoRoot, targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve draft promote/apply target path: %w", err)
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("draft promote/apply target path escapes repository root")
	}
	return targetPath, nil
}

func writeBrokerOwnedDraftPromoteFile(path string, contents []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create draft promote/apply target parent: %w", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("open draft promote/apply temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod draft promote/apply temp file: %w", err)
	}
	if _, err := tmpFile.Write(contents); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write draft promote/apply temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close draft promote/apply temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename draft promote/apply temp file: %w", err)
	}
	return nil
}
