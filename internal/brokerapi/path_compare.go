package brokerapi

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func pathsReferToSameLocation(first string, second string) (bool, error) {
	firstAbs, err := filepath.Abs(first)
	if err != nil {
		return false, err
	}
	secondAbs, err := filepath.Abs(second)
	if err != nil {
		return false, err
	}
	firstResolved, err := resolvedExistingPath(firstAbs)
	if err != nil {
		return false, err
	}
	secondResolved, err := resolvedExistingPath(secondAbs)
	if err != nil {
		return false, err
	}
	firstInfo, firstStatErr := os.Stat(firstResolved)
	secondInfo, secondStatErr := os.Stat(secondResolved)
	if firstStatErr == nil && secondStatErr == nil {
		return os.SameFile(firstInfo, secondInfo), nil
	}
	return sameFilesystemPath(firstResolved, secondResolved), nil
}

func resolvedExistingPath(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, nil
		}
		return "", err
	}
	return resolved, nil
}

func sameFilesystemPath(first string, second string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(first), filepath.Clean(second))
	}
	return filepath.Clean(first) == filepath.Clean(second)
}
