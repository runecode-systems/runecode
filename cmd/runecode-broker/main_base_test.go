package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/localbootstrap"
)

func TestResolveExplicitLiveIPCTargetConfigUsesDefaultConfigSeam(t *testing.T) {
	setDefaultLocalIPCConfigForTest(t, func() (brokerapi.LocalIPCConfig, error) {
		return brokerapi.LocalIPCConfig{RuntimeDir: "/default/runtime", SocketName: "default.sock", RepositoryRoot: "/repo/root"}, nil
	})

	cfg, err := resolveExplicitLiveIPCTargetConfig(brokerLiveIPCTargetOptions{runtimeDir: "/explicit/runtime", socketName: "explicit.sock"})
	if err != nil {
		t.Fatalf("resolveExplicitLiveIPCTargetConfig returned error: %v", err)
	}
	if cfg.RuntimeDir != "/explicit/runtime" {
		t.Fatalf("RuntimeDir = %q, want /explicit/runtime", cfg.RuntimeDir)
	}
	if cfg.SocketName != "explicit.sock" {
		t.Fatalf("SocketName = %q, want explicit.sock", cfg.SocketName)
	}
	if cfg.RepositoryRoot != "/repo/root" {
		t.Fatalf("RepositoryRoot = %q, want /repo/root", cfg.RepositoryRoot)
	}
}

func TestResolveExplicitLiveIPCTargetConfigDefaultsFromSeamAndPropagatesError(t *testing.T) {
	t.Run("defaults runtime dir and socket name", func(t *testing.T) {
		setDefaultLocalIPCConfigForTest(t, func() (brokerapi.LocalIPCConfig, error) {
			return brokerapi.LocalIPCConfig{RuntimeDir: "/default/runtime", SocketName: "default.sock", RepositoryRoot: "/repo/root"}, nil
		})

		cfg, err := resolveExplicitLiveIPCTargetConfig(brokerLiveIPCTargetOptions{})
		if err != nil {
			t.Fatalf("resolveExplicitLiveIPCTargetConfig returned error: %v", err)
		}
		if cfg.RuntimeDir != "/default/runtime" {
			t.Fatalf("RuntimeDir = %q, want /default/runtime", cfg.RuntimeDir)
		}
		if cfg.SocketName != "default.sock" {
			t.Fatalf("SocketName = %q, want default.sock", cfg.SocketName)
		}
		if cfg.RepositoryRoot != "/repo/root" {
			t.Fatalf("RepositoryRoot = %q, want /repo/root", cfg.RepositoryRoot)
		}
	})

	t.Run("propagates default config error", func(t *testing.T) {
		wantErr := errors.New("linux only")
		setDefaultLocalIPCConfigForTest(t, func() (brokerapi.LocalIPCConfig, error) {
			return brokerapi.LocalIPCConfig{}, wantErr
		})

		_, err := resolveExplicitLiveIPCTargetConfig(brokerLiveIPCTargetOptions{})
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})
}

func TestRunSeedsRepoRootEnvForSchemaCommandsOutsideRepoRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	stateRoot := filepath.Join(t.TempDir(), "state")
	auditRoot := filepath.Join(t.TempDir(), "audit")

	originalResolve := resolveBrokerRepoScope
	resolveBrokerRepoScope = func() (localbootstrap.RepoScope, error) {
		return localbootstrap.RepoScope{RepositoryRoot: repoRoot, StateRoot: stateRoot, AuditLedgerRoot: auditRoot}, nil
	}
	t.Cleanup(func() { resolveBrokerRepoScope = originalResolve })

	t.Setenv("RUNE_REPO_ROOT", "")
	outside := t.TempDir()
	if err := os.Chdir(outside); err != nil {
		t.Fatalf("Chdir outside repo root returned error: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(wd); chdirErr != nil {
			t.Fatalf("restore working directory returned error: %v", chdirErr)
		}
	})

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	err = run([]string{"--state-root", stateRoot, "--audit-ledger-root", auditRoot, "head-artifact", "--digest", "invalid"}, stdout, stderr)
	if err == nil {
		t.Fatal("head-artifact expected validation error for invalid digest")
	}
	if !strings.Contains(err.Error(), "broker_validation_schema_invalid") {
		t.Fatalf("error = %q, want typed schema validation code", err.Error())
	}
	if got := os.Getenv("RUNE_REPO_ROOT"); got != repoRoot {
		t.Fatalf("RUNE_REPO_ROOT = %q, want %q", got, repoRoot)
	}
}
