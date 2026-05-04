package brokerapi

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

func validateOfflineBundlePath(bundlePath string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(bundlePath))
	if clean == "." || clean == "" {
		return "", errOfflineBundlePathRequired
	}
	if !filepath.IsAbs(clean) {
		return "", errOfflineBundlePathAbsolute
	}
	if filepath.Ext(clean) != ".tar" {
		return "", errOfflineBundlePathTar
	}
	if err := rejectLinkedPathComponents(filepath.Dir(clean)); err != nil {
		if errors.Is(err, errLinkedPathComponent) {
			return "", errOfflineBundlePathLinkedComponents
		}
		return "", fmt.Errorf("%w: %v", errOfflineBundlePathNotAccessible, err)
	}
	return clean, nil
}
