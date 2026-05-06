package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}
	root, ok := findRepoRoot(cwd)
	if !ok {
		return "", fmt.Errorf("resolve repo root from %s: required markers go.mod, justfile, and %s", cwd, specDirRelative)
	}
	return root, nil
}

func findRepoRoot(start string) (string, bool) {
	dir := start
	for {
		if looksLikeRepoRoot(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func looksLikeRepoRoot(path string) bool {
	if !fileExists(filepath.Join(path, "go.mod")) {
		return false
	}
	if !fileExists(filepath.Join(path, "justfile")) {
		return false
	}
	return dirExists(filepath.Join(path, specDirRelative))
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat dir %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory: %s", path)
	}
	return nil
}

func ensureFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file, got directory: %s", path)
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
