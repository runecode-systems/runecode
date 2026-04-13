package brokerapi

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var errLinkedPathComponent = errors.New("linked path component")

func rejectLinkedPathComponents(path string) error {
	current, remainder := linkedPathWalkState(path)
	if remainder == "" {
		return nil
	}
	for _, part := range strings.Split(remainder, string(os.PathSeparator)) {
		if skipPathPart(part) {
			continue
		}
		next, err := validatePathComponent(current, part)
		if err != nil {
			return err
		}
		if next == "" {
			return nil
		}
		current = next
	}
	return nil
}

func validatePathComponent(current string, part string) (string, error) {
	next := filepath.Join(current, part)
	info, err := lstatExistingPath(next)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	linked, err := pathEntryIsLinkOrReparse(next, info)
	if err != nil {
		return "", err
	}
	if linked {
		return "", errLinkedPathComponent
	}
	return next, nil
}

func linkedPathWalkState(path string) (string, string) {
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	remainder := strings.TrimPrefix(clean, volume)
	if strings.HasPrefix(remainder, string(os.PathSeparator)) {
		return volume + string(os.PathSeparator), strings.TrimPrefix(remainder, string(os.PathSeparator))
	}
	return volume, remainder
}

func skipPathPart(part string) bool {
	return part == "" || part == "."
}

func lstatExistingPath(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return info, nil
}
