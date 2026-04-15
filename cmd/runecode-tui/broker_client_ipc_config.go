package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func localIPCConfigProviderWithOverrides(base func() (brokerapi.LocalIPCConfig, error), runtimeDir, socketName string) func() (brokerapi.LocalIPCConfig, error) {
	return func() (brokerapi.LocalIPCConfig, error) {
		cfg, err := base()
		trimmedRuntimeDir := strings.TrimSpace(runtimeDir)
		trimmedSocketName := strings.TrimSpace(socketName)
		if err != nil {
			if trimmedRuntimeDir == "" || trimmedSocketName == "" {
				return brokerapi.LocalIPCConfig{}, err
			}
			fallback, fallbackErr := validatedLocalIPCConfig(brokerapi.LocalIPCConfig{RuntimeDir: trimmedRuntimeDir, SocketName: trimmedSocketName})
			if fallbackErr != nil {
				// Keep the authoritative base-provider failure as the public error when overrides cannot safely replace it.
				return brokerapi.LocalIPCConfig{}, err
			}
			return fallback, nil
		}
		if trimmedRuntimeDir != "" {
			cfg.RuntimeDir = trimmedRuntimeDir
		}
		if trimmedSocketName != "" {
			cfg.SocketName = trimmedSocketName
		}
		return validatedLocalIPCConfig(cfg)
	}
}

func validatedLocalIPCConfig(cfg brokerapi.LocalIPCConfig) (brokerapi.LocalIPCConfig, error) {
	cleanRuntimeDir, err := validatedLocalIPCRuntimeDir(cfg.RuntimeDir)
	if err != nil {
		return brokerapi.LocalIPCConfig{}, err
	}
	if err := validateLocalIPCSocketName(cfg.SocketName); err != nil {
		return brokerapi.LocalIPCConfig{}, err
	}
	return brokerapi.LocalIPCConfig{RuntimeDir: cleanRuntimeDir, SocketName: cfg.SocketName}, nil
}

func validatedLocalIPCRuntimeDir(runtimeDir string) (string, error) {
	cleanRuntimeDir := filepath.Clean(runtimeDir)
	if !filepath.IsAbs(cleanRuntimeDir) {
		return "", fmt.Errorf("runtime directory must be absolute")
	}
	if filepath.Dir(cleanRuntimeDir) == cleanRuntimeDir {
		return "", fmt.Errorf("runtime directory must be a non-root absolute path")
	}
	if cleanRuntimeDir != runtimeDir {
		return "", fmt.Errorf("runtime directory must be normalized")
	}
	return cleanRuntimeDir, nil
}

func validateLocalIPCSocketName(socketName string) error {
	if strings.TrimSpace(socketName) == "" {
		return fmt.Errorf("socket name is required")
	}
	if socketName == "." || socketName == ".." {
		return fmt.Errorf("socket name must not be dot path")
	}
	if strings.ContainsRune(socketName, 0) {
		return fmt.Errorf("socket name must not contain null bytes")
	}
	if socketName != filepath.Base(socketName) {
		return fmt.Errorf("socket name must not include path separators")
	}
	for _, r := range socketName {
		if isAllowedLocalIPCSocketRune(r) {
			continue
		}
		return fmt.Errorf("socket name contains unsupported characters")
	}
	return nil
}

func isAllowedLocalIPCSocketRune(r rune) bool {
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
		return true
	}
	switch r {
	case '-', '_', '.':
		return true
	default:
		return false
	}
}
