package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const localSnapshotDirName = "runecode-tui-snapshots"

func canonicalPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolvedPath, err := evalSymlinksAllowMissing(filepath.Clean(absPath))
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolvedPath), nil
}

func evalSymlinksAllowMissing(path string) (string, error) {
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolvedPath, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	missingParts := []string{}
	existingPath := path
	for {
		if _, statErr := os.Stat(existingPath); statErr == nil {
			break
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return "", statErr
		}

		parent := filepath.Dir(existingPath)
		if parent == existingPath {
			return "", err
		}
		missingParts = append(missingParts, filepath.Base(existingPath))
		existingPath = parent
	}

	resolvedExistingPath, err := filepath.EvalSymlinks(existingPath)
	if err != nil {
		return "", err
	}
	for i := len(missingParts) - 1; i >= 0; i-- {
		resolvedExistingPath = filepath.Join(resolvedExistingPath, missingParts[i])
	}
	return resolvedExistingPath, nil
}

func prepareArtifactsDir(workDir string, outputDir string) (string, error) {
	artifactsDir := workDir
	if outputDir == "" {
		artifactsDir = filepath.Join(workDir, "artifacts")
	}
	resolvedArtifactsDir, err := canonicalPath(artifactsDir)
	if err != nil {
		return "", fmt.Errorf("resolve artifact directory: %w", err)
	}
	if err := requirePathWithinTrustedRoot("artifact directory", resolvedArtifactsDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(resolvedArtifactsDir, 0o755); err != nil {
		return "", fmt.Errorf("create artifact directory: %w", err)
	}
	return resolvedArtifactsDir, nil
}

func resolveLocalSnapshotDir(outputDir string) (string, error) {
	requestedDir := strings.TrimSpace(outputDir)
	if requestedDir == "" {
		requestedDir = filepath.Join(os.TempDir(), localSnapshotDirName)
	}
	resolvedOutputDir, err := canonicalPath(requestedDir)
	if err != nil {
		return "", fmt.Errorf("resolve local snapshot directory: %w", err)
	}
	if err := requirePathWithinTrustedRoot("local snapshot directory", resolvedOutputDir); err != nil {
		return "", err
	}
	return resolvedOutputDir, nil
}

func cleanupLocalSnapshotDir(outputDir string) error {
	resolvedOutputDir, err := resolveLocalSnapshotDir(outputDir)
	if err != nil {
		return err
	}
	if err := requirePathWithinTrustedRoot("local snapshot cleanup directory", resolvedOutputDir); err != nil {
		return err
	}
	trustedRoots, err := trustedTempRoots()
	if err != nil {
		return err
	}
	for _, trustedRoot := range trustedRoots {
		if resolvedOutputDir == trustedRoot {
			return fmt.Errorf("local snapshot cleanup directory must not be the trusted temporary root %q", trustedRoot)
		}
	}
	if err := os.RemoveAll(resolvedOutputDir); err != nil {
		return fmt.Errorf("remove local snapshot directory %q: %w", resolvedOutputDir, err)
	}
	return nil
}

func normalizeTrustedOutputDir(outputDir string) (string, error) {
	resolvedOutputDir, err := canonicalPath(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output directory: %w", err)
	}
	if err := requirePathWithinTrustedRoot("output directory", resolvedOutputDir); err != nil {
		return "", err
	}
	return resolvedOutputDir, nil
}

func requirePathWithinTrustedRoot(label string, candidate string) error {
	trustedRoots, err := trustedTempRoots()
	if err != nil {
		return err
	}
	for _, trustedRoot := range trustedRoots {
		if pathWithinBase(trustedRoot, candidate) {
			return nil
		}
	}
	return fmt.Errorf("%s must stay under %s", label, trustedRoots[0])
}

func trustedTempRoots() ([]string, error) {
	roots := []string{}
	primaryRoot, err := canonicalPath(os.TempDir())
	if err != nil {
		return nil, fmt.Errorf("resolve temporary directory: %w", err)
	}
	roots = append(roots, primaryRoot)
	if _, err := os.Stat("/tmp"); err == nil {
		tmpRoot, err := canonicalPath("/tmp")
		if err != nil {
			return nil, fmt.Errorf("resolve /tmp: %w", err)
		}
		if !slices.Contains(roots, tmpRoot) {
			roots = append(roots, tmpRoot)
		}
	}
	return roots, nil
}

func pathWithinBase(base string, candidate string) bool {
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
