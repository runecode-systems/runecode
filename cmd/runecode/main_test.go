package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/localbootstrap"
)

func TestBareRunecodeIsAttach(t *testing.T) {
	restore := stubRunecodeLifecycle(t)
	defer restore()
	out := &bytes.Buffer{}
	if err := run([]string{}, out, &bytes.Buffer{}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if got := out.String(); got == "" {
		t.Fatal("attach output empty")
	}
}

func TestStatusIsNonStartingWhenBrokerUnavailable(t *testing.T) {
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: "/runtime", LocalSocketName: "broker.sock"}
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldStart := startRepoBroker
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{}, errNoLiveBroker
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
	out := &bytes.Buffer{}
	if err := run([]string{"status"}, out, &bytes.Buffer{}); err != nil {
		t.Fatalf("run status returned error: %v", err)
	}
	if startCalled {
		t.Fatal("status started broker; expected non-starting behavior")
	}
	if got := out.String(); got == "" {
		t.Fatal("status output empty")
	}
	if strings.Contains(out.String(), "/repo") {
		t.Fatalf("status output leaked repository path: %q", out.String())
	}
}

func TestStartEnsuresLifecycleAndDoesNotLaunchTUI(t *testing.T) {
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldStart := startRepoBroker
	oldLaunch := launchTUI
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: "/runtime", LocalSocketName: "broker.sock"}
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	step := 0
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		if step == 0 {
			step++
			return brokerapi.BrokerProductLifecyclePosture{}, errNoLiveBroker
		}
		return brokerapi.BrokerProductLifecyclePosture{RepositoryRoot: "/repo", ProductInstanceID: "repo-a", Attachable: true, AttachMode: "full", LifecyclePosture: "ready"}, nil
	}
	started := false
	startRepoBroker = func(localbootstrap.RepoScope) error {
		started = true
		return nil
	}
	launched := false
	launchTUI = func(localbootstrap.RepoScope) error {
		launched = true
		return nil
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		startRepoBroker = oldStart
		launchTUI = oldLaunch
	})
	out := &bytes.Buffer{}
	if err := run([]string{"start"}, out, &bytes.Buffer{}); err != nil {
		t.Fatalf("run start returned error: %v", err)
	}
	if !started {
		t.Fatal("start did not invoke broker bootstrap")
	}
	if launched {
		t.Fatal("start launched tui")
	}
}

func TestUsageErrorForUnknownSubcommand(t *testing.T) {
	err := run([]string{"bogus"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected usage error")
	}
	var u *usageError
	if !errors.As(err, &u) {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestStopUsesLiveBrokerPeerPIDWithoutPIDFile(t *testing.T) {
	runtimeDir := t.TempDir()
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: runtimeDir, LocalSocketName: "broker.sock"}
	oldResolveProcess := resolveRepoBrokerProcess
	oldInterrupt := interruptBrokerProcess
	oldKill := killBrokerProcess
	oldQuery := queryProductLifecyclePosture
	resolveRepoBrokerProcess = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, int, error) {
		return brokerapi.BrokerProductLifecyclePosture{RepositoryRoot: "/repo", ProductInstanceID: "repo-a", Attachable: true, AttachMode: "full", LifecyclePosture: "ready"}, 4321, nil
	}
	interruptedPID := 0
	interruptBrokerProcess = func(pid int) error {
		interruptedPID = pid
		return nil
	}
	killCalled := false
	killBrokerProcess = func(pid int) error {
		killCalled = true
		return nil
	}
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{}, errNoLiveBroker
	}
	t.Cleanup(func() {
		resolveRepoBrokerProcess = oldResolveProcess
		interruptBrokerProcess = oldInterrupt
		killBrokerProcess = oldKill
		queryProductLifecyclePosture = oldQuery
	})
	if err := stopRepoBrokerProcess(scope); err != nil {
		t.Fatalf("stopRepoBrokerProcess returned error: %v", err)
	}
	if interruptedPID != 4321 {
		t.Fatalf("interrupted pid = %d, want 4321", interruptedPID)
	}
	if killCalled {
		t.Fatal("killBrokerProcess called, want graceful stop without forced kill")
	}
}

func TestStartCleansUpBrokerWhenPIDWriteFails(t *testing.T) {
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldStart := startRepoBroker
	oldWritePID := writeBrokerPID
	oldCleanup := cleanupStartedBrokerProcess
	resolveRepoScope = func() (localbootstrap.RepoScope, error) {
		return localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: t.TempDir(), LocalSocketName: "broker.sock"}, nil
	}
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{}, errNoLiveBroker
	}
	startCalled := false
	startRepoBroker = func(localbootstrap.RepoScope) error {
		startCalled = true
		return errors.New("persist broker pid: disk full")
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		startRepoBroker = oldStart
		writeBrokerPID = oldWritePID
		cleanupStartedBrokerProcess = oldCleanup
	})
	err := run([]string{"start"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run start error = nil, want failure")
	}
	if !startCalled {
		t.Fatal("startRepoBroker not invoked")
	}
	if !strings.Contains(err.Error(), "persist broker pid") {
		t.Fatalf("error = %q, want pid persistence failure", err.Error())
	}
}

func TestStopReturnsKillFailureAfterTimeout(t *testing.T) {
	runtimeDir := t.TempDir()
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: runtimeDir, LocalSocketName: "broker.sock"}
	oldResolveProcess := resolveRepoBrokerProcess
	oldInterrupt := interruptBrokerProcess
	oldKill := killBrokerProcess
	oldQuery := queryProductLifecyclePosture
	resolveRepoBrokerProcess = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, int, error) {
		return brokerapi.BrokerProductLifecyclePosture{RepositoryRoot: "/repo", ProductInstanceID: "repo-a", Attachable: true, AttachMode: "full", LifecyclePosture: "ready"}, 4321, nil
	}
	interruptBrokerProcess = func(int) error { return nil }
	killBrokerProcess = func(int) error { return errors.New("kill failed") }
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{RepositoryRoot: "/repo", ProductInstanceID: "repo-a", Attachable: true, AttachMode: "full", LifecyclePosture: "ready"}, nil
	}
	t.Cleanup(func() {
		resolveRepoBrokerProcess = oldResolveProcess
		interruptBrokerProcess = oldInterrupt
		killBrokerProcess = oldKill
		queryProductLifecyclePosture = oldQuery
	})
	err := stopRepoBrokerProcess(scope)
	if err == nil {
		t.Fatal("stopRepoBrokerProcess error = nil, want kill failure")
	}
	if !strings.Contains(err.Error(), "kill failed") {
		t.Fatalf("error = %q, want kill failure", err.Error())
	}
}

func stubRunecodeLifecycle(t *testing.T) func() {
	t.Helper()
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldStart := startRepoBroker
	oldLaunch := launchTUI
	oldStop := stopRepoBroker
	oldResolveProcess := resolveRepoBrokerProcess
	oldInterrupt := interruptBrokerProcess
	oldKill := killBrokerProcess
	resolveRepoScope = func() (localbootstrap.RepoScope, error) {
		return localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: "/runtime", LocalSocketName: "broker.sock"}, nil
	}
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{RepositoryRoot: "/repo", ProductInstanceID: "repo-a", Attachable: true, AttachMode: "full", LifecyclePosture: "ready"}, nil
	}
	resolveRepoBrokerProcess = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, int, error) {
		return brokerapi.BrokerProductLifecyclePosture{RepositoryRoot: "/repo", ProductInstanceID: "repo-a", Attachable: true, AttachMode: "full", LifecyclePosture: "ready"}, 1234, nil
	}
	startRepoBroker = func(localbootstrap.RepoScope) error { return nil }
	launchTUI = func(localbootstrap.RepoScope) error { return nil }
	stopRepoBroker = func(localbootstrap.RepoScope) error { return nil }
	interruptBrokerProcess = func(int) error { return nil }
	killBrokerProcess = func(int) error { return nil }
	return func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		startRepoBroker = oldStart
		launchTUI = oldLaunch
		stopRepoBroker = oldStop
		resolveRepoBrokerProcess = oldResolveProcess
		interruptBrokerProcess = oldInterrupt
		killBrokerProcess = oldKill
	}
}
