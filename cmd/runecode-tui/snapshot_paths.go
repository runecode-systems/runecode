//go:build runecode_tui_snapshot

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const snapshotDefaultDirName = ".tui-snapshots"

func snapshotDefaultOutputDir() string {
	workspaceRoot, err := snapshotWorkspaceRoot()
	if err == nil {
		return filepath.Join(workspaceRoot, snapshotDefaultDirName)
	}
	cwd, err := os.Getwd()
	if err == nil {
		return filepath.Join(cwd, snapshotDefaultDirName)
	}
	return filepath.Join(os.TempDir(), snapshotDefaultDirName)
}

func normalizeSnapshotOutputDir(outputDir string) (string, error) {
	resolvedOutputDir, err := snapshotCanonicalPath(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve snapshot output directory: %w", err)
	}
	trustedRoots, err := snapshotTrustedOutputRoots()
	if err != nil {
		return "", err
	}
	for _, trustedRoot := range trustedRoots {
		if snapshotPathWithinBase(trustedRoot, resolvedOutputDir) {
			return resolvedOutputDir, nil
		}
	}
	return "", fmt.Errorf("snapshot output directory must stay under one of %s", strings.Join(trustedRoots, ", "))
}

func snapshotTrustedOutputRoots() ([]string, error) {
	roots := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)

	addRoot := func(path string) error {
		if path == "" {
			return nil
		}
		resolvedPath, err := snapshotCanonicalPath(path)
		if err != nil {
			return err
		}
		if _, ok := seen[resolvedPath]; ok {
			return nil
		}
		seen[resolvedPath] = struct{}{}
		roots = append(roots, resolvedPath)
		return nil
	}

	if err := addRoot(os.TempDir()); err != nil {
		return nil, fmt.Errorf("resolve temporary directory: %w", err)
	}
	if _, err := os.Stat("/tmp"); err == nil {
		if err := addRoot("/tmp"); err != nil {
			return nil, fmt.Errorf("resolve /tmp temporary directory: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("stat /tmp: %w", err)
	}
	if err := addRoot(snapshotDefaultOutputDir()); err != nil {
		return nil, fmt.Errorf("resolve default snapshot directory: %w", err)
	}

	return roots, nil
}

func snapshotWorkspaceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	resolvedCWD, err := snapshotCanonicalPath(cwd)
	if err != nil {
		return "", err
	}
	dir := resolvedCWD
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return resolvedCWD, nil
		}
		dir = parent
	}
}

func snapshotCanonicalPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolvedPath, err := snapshotEvalSymlinksAllowMissing(filepath.Clean(absPath))
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolvedPath), nil
}

func snapshotEvalSymlinksAllowMissing(path string) (string, error) {
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

func snapshotPathWithinBase(base string, candidate string) bool {
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
