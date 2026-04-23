//go:build unix

package main

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/localbootstrap"
)

func TestEnsureRepoLifecycleRecoversStaleRuntimeArtifactsBeforeStart(t *testing.T) {
	runtimeDir := shortRuntimeDir(t)
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: runtimeDir, LocalSocketName: "broker.sock"}
	socketPath := seedStaleRuntimeArtifacts(t, scope)

	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldStart := startRepoBroker
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryCalls := 0
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		queryCalls++
		if queryCalls <= 3 {
			return brokerapi.BrokerProductLifecyclePosture{}, errNoLiveBroker
		}
		return brokerapi.BrokerProductLifecyclePosture{RepositoryRoot: "/repo", ProductInstanceID: "repo-a", Attachable: true, AttachMode: "full", LifecyclePosture: "ready"}, nil
	}
	startCalled := false
	startRepoBroker = func(localbootstrap.RepoScope) error {
		startCalled = true
		assertRuntimeArtifactsRemoved(t, scope, socketPath)
		return nil
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		startRepoBroker = oldStart
	})

	_, posture, err := ensureRepoLifecycle()
	if err != nil {
		t.Fatalf("ensureRepoLifecycle returned error: %v", err)
	}
	if !startCalled {
		t.Fatal("startRepoBroker not called")
	}
	if posture.ProductInstanceID != "repo-a" {
		t.Fatalf("posture.ProductInstanceID = %q, want repo-a", posture.ProductInstanceID)
	}
}

func seedStaleRuntimeArtifacts(t *testing.T, scope localbootstrap.RepoScope) string {
	t.Helper()
	if err := os.WriteFile(pidFilePath(scope), []byte(strconv.Itoa(99999999)), 0o600); err != nil {
		t.Fatalf("WriteFile(pidfile) returned error: %v", err)
	}
	socketPath := filepath.Join(scope.LocalRuntimeDir, scope.LocalSocketName)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen(unix) returned error: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close returned error: %v", err)
	}
	return socketPath
}

func assertRuntimeArtifactsRemoved(t *testing.T, scope localbootstrap.RepoScope, socketPath string) {
	t.Helper()
	assertPathNotExists(t, pidFilePath(scope), "pidfile")
	assertPathNotExists(t, socketPath, "socket")
}

func assertPathNotExists(t *testing.T, path string, label string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			t.Fatalf("%s still present", label)
		}
		t.Fatalf("Stat(%s) returned error: %v", label, err)
	}
}

func TestEnsureRepoLifecycleDoesNotCleanArtifactsWhenBrokerReachable(t *testing.T) {
	runtimeDir := shortRuntimeDir(t)
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: runtimeDir, LocalSocketName: "broker.sock"}
	if err := os.WriteFile(pidFilePath(scope), []byte("not-a-pid"), 0o600); err != nil {
		t.Fatalf("WriteFile(pidfile) returned error: %v", err)
	}
	socketPath := filepath.Join(runtimeDir, scope.LocalSocketName)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen(unix) returned error: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldStart := startRepoBroker
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{RepositoryRoot: "/repo", ProductInstanceID: "repo-a", Attachable: true, AttachMode: "full", LifecyclePosture: "ready"}, nil
	}
	startCalled := false
	startRepoBroker = func(localbootstrap.RepoScope) error {
		startCalled = true
		return nil
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		startRepoBroker = oldStart
	})

	_, posture, err := ensureRepoLifecycle()
	if err != nil {
		t.Fatalf("ensureRepoLifecycle returned error: %v", err)
	}
	if posture.ProductInstanceID != "repo-a" {
		t.Fatalf("posture.ProductInstanceID = %q, want repo-a", posture.ProductInstanceID)
	}
	if startCalled {
		t.Fatal("startRepoBroker called with reachable broker")
	}
	if _, err := os.Stat(pidFilePath(scope)); err != nil {
		t.Fatalf("Stat(pidfile) returned error: %v", err)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("Stat(socket) returned error: %v", err)
	}
}
