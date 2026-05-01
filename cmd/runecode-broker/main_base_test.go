package main

import (
	"errors"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
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
