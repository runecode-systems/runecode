package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestBrokerServiceUsesTempFallbackWhenUserDirsUnavailable(t *testing.T) {
	originalFactory := brokerServiceFactory
	defer func() { brokerServiceFactory = originalFactory }()

	t.Setenv("HOME", "")
	if err := os.Unsetenv("XDG_CACHE_HOME"); err != nil {
		t.Fatalf("Unsetenv(XDG_CACHE_HOME) error: %v", err)
	}
	if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
		t.Fatalf("Unsetenv(XDG_CONFIG_HOME) error: %v", err)
	}

	root := defaultBrokerStoreRoot()
	if root == "" {
		t.Fatal("defaultBrokerStoreRoot returned empty path")
	}
	if !filepath.IsAbs(root) {
		t.Fatalf("defaultBrokerStoreRoot = %q, want absolute path", root)
	}
	if !strings.Contains(filepath.ToSlash(root), "/runecode/artifact-store") {
		t.Fatalf("defaultBrokerStoreRoot = %q, want path containing runecode/artifact-store", root)
	}
	if _, err := brokerapi.NewService(root, filepath.Join(root, "audit-ledger")); err != nil {
		t.Fatalf("NewService(%q) error: %v", root, err)
	}
}
