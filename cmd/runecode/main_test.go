package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
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
	oldSubstrateQuery := queryProjectSubstratePosture
	oldStart := startRepoBroker
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{}, errNoLiveBroker
	}
	substrateQueried := false
	queryProjectSubstratePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.ProjectSubstratePostureGetResponse, error) {
		substrateQueried = true
		return brokerapi.ProjectSubstratePostureGetResponse{}, nil
	}
	startCalled := false
	startRepoBroker = func(localbootstrap.RepoScope) error {
		startCalled = true
		return nil
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		queryProjectSubstratePosture = oldSubstrateQuery
		startRepoBroker = oldStart
	})
	out := &bytes.Buffer{}
	if err := run([]string{"status"}, out, &bytes.Buffer{}); err != nil {
		t.Fatalf("run status returned error: %v", err)
	}
	if startCalled {
		t.Fatal("status started broker; expected non-starting behavior")
	}
	if substrateQueried {
		t.Fatal("status queried project substrate posture while broker unavailable")
	}
	if got := strings.TrimSpace(out.String()); got != "runecode status: no live product instance reachable" {
		t.Fatalf("status output = %q, want no-live message", got)
	}
	if strings.Contains(out.String(), "/repo") {
		t.Fatalf("status output leaked repository path: %q", out.String())
	}
}

func TestStatusIncludesBrokerLifecycleAndProjectSubstratePosture(t *testing.T) {
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: "/runtime", LocalSocketName: "broker.sock"}
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldSubstrateQuery := queryProjectSubstratePosture
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{
			RepositoryRoot:         "/repo",
			ProductInstanceID:      "repo-a",
			LifecycleGeneration:    "gen-123",
			AttachMode:             "full",
			LifecyclePosture:       "ready",
			Attachable:             true,
			NormalOperationAllowed: true,
		}, nil
	}
	queryProjectSubstratePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.ProjectSubstratePostureGetResponse, error) {
		return brokerapi.ProjectSubstratePostureGetResponse{
			RepositoryRoot: "/repo",
			PostureSummary: brokerapi.ProjectSubstratePostureSummary{
				CompatibilityPosture:   "supported_current",
				ValidationState:        "valid",
				NormalOperationAllowed: true,
			},
		}, nil
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		queryProjectSubstratePosture = oldSubstrateQuery
	})
	out := &bytes.Buffer{}
	if err := run([]string{"status"}, out, &bytes.Buffer{}); err != nil {
		t.Fatalf("run status returned error: %v", err)
	}
	got := strings.TrimSpace(out.String())
	wantParts := []string{
		"instance=repo-a",
		"generation=gen-123",
		"mode=full",
		"posture=ready",
		"attachable=true",
		"normal_operation_allowed=true",
		"blocked_reasons=none",
		"degraded_reasons=none",
		"project_substrate_posture=supported_current",
		"project_substrate_validation_state=valid",
		"project_substrate_normal_operation_allowed=true",
		"project_substrate_blocked_reasons=none",
		"project_substrate_remediation_guidance=none",
	}
	for _, want := range wantParts {
		if !strings.Contains(got, want) {
			t.Fatalf("status output missing %q: %q", want, got)
		}
	}
}

func TestStatusIncludesBlockedProjectSubstrateGuidanceForDiagnosticsAttach(t *testing.T) {
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: "/runtime", LocalSocketName: "broker.sock"}
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldSubstrateQuery := queryProjectSubstratePosture
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{
			RepositoryRoot:         "/repo",
			ProductInstanceID:      "repo-a",
			LifecycleGeneration:    "gen-456",
			AttachMode:             "diagnostics_only",
			LifecyclePosture:       "blocked",
			Attachable:             true,
			NormalOperationAllowed: false,
			BlockedReasonCodes:     []string{"project_substrate_unsupported_too_new"},
		}, nil
	}
	queryProjectSubstratePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.ProjectSubstratePostureGetResponse, error) {
		return brokerapi.ProjectSubstratePostureGetResponse{
			RepositoryRoot: "/repo",
			PostureSummary: brokerapi.ProjectSubstratePostureSummary{
				CompatibilityPosture:   "unsupported_too_new",
				ValidationState:        "invalid",
				NormalOperationAllowed: false,
				BlockedReasonCodes:     []string{"project_substrate_unsupported_too_new"},
			},
			RemediationGuidance: []string{"run project_substrate_upgrade_apply"},
		}, nil
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		queryProjectSubstratePosture = oldSubstrateQuery
	})
	out := &bytes.Buffer{}
	if err := run([]string{"status"}, out, &bytes.Buffer{}); err != nil {
		t.Fatalf("run status returned error: %v", err)
	}
	got := strings.TrimSpace(out.String())
	wantParts := []string{
		"mode=diagnostics_only",
		"posture=blocked",
		"normal_operation_allowed=false",
		"blocked_reasons=project_substrate_unsupported_too_new",
		"project_substrate_posture=unsupported_too_new",
		"project_substrate_validation_state=invalid",
		"project_substrate_normal_operation_allowed=false",
		"project_substrate_blocked_reasons=project_substrate_unsupported_too_new",
		"project_substrate_remediation_guidance=run project_substrate_upgrade_apply",
	}
	for _, want := range wantParts {
		if !strings.Contains(got, want) {
			t.Fatalf("status output missing %q: %q", want, got)
		}
	}
}

func TestStatusRejectsMismatchedBrokerIdentity(t *testing.T) {
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: "/runtime", LocalSocketName: "broker.sock"}
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{
			RepositoryRoot:         "/repo",
			ProductInstanceID:      "repo-b",
			LifecycleGeneration:    "gen-123",
			AttachMode:             "full",
			LifecyclePosture:       "ready",
			Attachable:             true,
			NormalOperationAllowed: true,
		}, nil
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
	})
	err := run([]string{"status"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run status error = nil, want identity mismatch")
	}
	if !strings.Contains(err.Error(), "reachable broker product instance does not match authoritative repository scope") {
		t.Fatalf("error = %q, want broker identity mismatch", err.Error())
	}
}

func TestStatusRejectsMismatchedProjectSubstrateIdentity(t *testing.T) {
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: "/runtime", LocalSocketName: "broker.sock"}
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldSubstrateQuery := queryProjectSubstratePosture
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{
			RepositoryRoot:         "/repo",
			ProductInstanceID:      "repo-a",
			LifecycleGeneration:    "gen-123",
			AttachMode:             "full",
			LifecyclePosture:       "ready",
			Attachable:             true,
			NormalOperationAllowed: true,
		}, nil
	}
	queryProjectSubstratePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.ProjectSubstratePostureGetResponse, error) {
		return brokerapi.ProjectSubstratePostureGetResponse{
			RepositoryRoot: "/other-repo",
			PostureSummary: brokerapi.ProjectSubstratePostureSummary{
				CompatibilityPosture:   "supported_current",
				ValidationState:        "valid",
				NormalOperationAllowed: true,
			},
		}, nil
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		queryProjectSubstratePosture = oldSubstrateQuery
	})
	err := run([]string{"status"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run status error = nil, want project substrate identity mismatch")
	}
	if !strings.Contains(err.Error(), "reachable broker repository root does not match authoritative repository root") {
		t.Fatalf("error = %q, want project substrate identity mismatch", err.Error())
	}
}

func TestStatusSanitizesBrokerSuppliedOutputFields(t *testing.T) {
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: "/runtime", LocalSocketName: "broker.sock"}
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldSubstrateQuery := queryProjectSubstratePosture
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		return brokerapi.BrokerProductLifecyclePosture{
			RepositoryRoot:         "/repo",
			ProductInstanceID:      "repo-a",
			LifecycleGeneration:    "gen-123\n",
			AttachMode:             "full\t",
			LifecyclePosture:       "ready\r",
			Attachable:             true,
			NormalOperationAllowed: true,
			BlockedReasonCodes:     []string{"blocked\x1b[0m"},
		}, nil
	}
	queryProjectSubstratePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.ProjectSubstratePostureGetResponse, error) {
		return brokerapi.ProjectSubstratePostureGetResponse{
			RepositoryRoot: "/repo",
			PostureSummary: brokerapi.ProjectSubstratePostureSummary{
				CompatibilityPosture:   "supported_current\n",
				ValidationState:        "valid\r",
				NormalOperationAllowed: true,
				BlockedReasonCodes:     []string{"reason\x1b[2J"},
			},
			RemediationGuidance: []string{"fix\x1b[31m-now"},
		}, nil
	}
	t.Cleanup(func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		queryProjectSubstratePosture = oldSubstrateQuery
	})
	out := &bytes.Buffer{}
	if err := run([]string{"status"}, out, &bytes.Buffer{}); err != nil {
		t.Fatalf("run status returned error: %v", err)
	}
	got := strings.TrimSuffix(out.String(), "\n")
	if strings.ContainsAny(got, "\x1b\n\r") {
		t.Fatalf("status output contains unsanitized control characters: %q", got)
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
		if step <= 2 {
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

func TestEnsureRepoLifecycleDoesNotStartBrokerWhenRecoveryFindsReachableBroker(t *testing.T) {
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: shortRuntimeDir(t), LocalSocketName: "broker.sock"}
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldStart := startRepoBroker
	resolveRepoScope = func() (localbootstrap.RepoScope, error) { return scope, nil }
	queryCalls := 0
	queryProductLifecyclePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		queryCalls++
		if queryCalls == 1 {
			return brokerapi.BrokerProductLifecyclePosture{}, errNoLiveBroker
		}
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
	if startCalled {
		t.Fatal("startRepoBroker called after recovery found reachable broker")
	}
	if posture.ProductInstanceID != "repo-a" {
		t.Fatalf("posture.ProductInstanceID = %q, want repo-a", posture.ProductInstanceID)
	}
}

func TestEnsureRepoLifecycleFailsClosedWhenBrokerPIDIsAliveButUnreachable(t *testing.T) {
	runtimeDir := shortRuntimeDir(t)
	scope := localbootstrap.RepoScope{RepositoryRoot: "/repo", ProductInstance: "repo-a", LocalRuntimeDir: runtimeDir, LocalSocketName: "broker.sock"}
	if err := os.WriteFile(pidFilePath(scope), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatalf("WriteFile(pidfile) returned error: %v", err)
	}
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

	_, _, err := ensureRepoLifecycle()
	if err == nil {
		t.Fatal("ensureRepoLifecycle error = nil, want fail-closed unreachable broker process")
	}
	if !errors.Is(err, errBrokerProcessUnreachable) {
		t.Fatalf("error = %v, want errBrokerProcessUnreachable", err)
	}
	if startCalled {
		t.Fatal("startRepoBroker called while unreachable broker pid remained alive")
	}
}

func stubRunecodeLifecycle(t *testing.T) func() {
	t.Helper()
	oldResolve := resolveRepoScope
	oldQuery := queryProductLifecyclePosture
	oldStart := startRepoBroker
	oldLaunch := launchTUI
	oldStop := stopRepoBroker
	oldSubstrateQuery := queryProjectSubstratePosture
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
	queryProjectSubstratePosture = func(context.Context, brokerapi.LocalIPCConfig) (brokerapi.ProjectSubstratePostureGetResponse, error) {
		return brokerapi.ProjectSubstratePostureGetResponse{RepositoryRoot: "/repo", PostureSummary: brokerapi.ProjectSubstratePostureSummary{CompatibilityPosture: "supported_current", ValidationState: "valid", NormalOperationAllowed: true}}, nil
	}
	interruptBrokerProcess = func(int) error { return nil }
	killBrokerProcess = func(int) error { return nil }
	return func() {
		resolveRepoScope = oldResolve
		queryProductLifecyclePosture = oldQuery
		startRepoBroker = oldStart
		launchTUI = oldLaunch
		stopRepoBroker = oldStop
		queryProjectSubstratePosture = oldSubstrateQuery
		resolveRepoBrokerProcess = oldResolveProcess
		interruptBrokerProcess = oldInterrupt
		killBrokerProcess = oldKill
	}
}

func shortRuntimeDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "rc-runtime-")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
