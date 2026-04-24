package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/localbootstrap"
)

func TestNewLiveIPCLocalAPIClientUsesResolvedRepoScopeForDial(t *testing.T) {
	called := false
	client, err := newLiveIPCLocalAPIClientWithDeps(context.Background(), liveIPCClientDeps{
		resolveScope: func(localbootstrap.ResolveInput) (localbootstrap.RepoScope, error) {
			return localbootstrap.RepoScope{
				RepositoryRoot:  "/repo/root",
				LocalRuntimeDir: "/runtime/dir",
				LocalSocketName: "broker.sock",
			}, nil
		},
		dialClient: func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (*brokerapi.LocalRPCClient, error) {
			called = true
			if ctx == nil {
				t.Fatal("DialLocalRPC ctx = nil")
			}
			if cfg.RepositoryRoot != "/repo/root" {
				t.Fatalf("RepositoryRoot = %q, want /repo/root", cfg.RepositoryRoot)
			}
			if cfg.RuntimeDir != "/runtime/dir" {
				t.Fatalf("RuntimeDir = %q, want /runtime/dir", cfg.RuntimeDir)
			}
			if cfg.SocketName != "broker.sock" {
				t.Fatalf("SocketName = %q, want broker.sock", cfg.SocketName)
			}
			return &brokerapi.LocalRPCClient{}, nil
		},
	})
	if err != nil {
		t.Fatalf("newLiveIPCLocalAPIClient returned error: %v", err)
	}
	if client == nil {
		t.Fatal("newLiveIPCLocalAPIClient returned nil client")
	}
	if !called {
		t.Fatal("DialLocalRPC was not called")
	}
}

func TestRunLiveIPCCommandSkipsStoreFactory(t *testing.T) {
	originalFactory := brokerServiceFactory
	originalResolver := localAPIClientModeResolver
	t.Cleanup(func() {
		brokerServiceFactory = originalFactory
		localAPIClientModeResolver = originalResolver
	})

	serviceFactoryCalled := false
	brokerServiceFactory = func(brokerServiceRoots) (*brokerapi.Service, error) {
		serviceFactoryCalled = true
		return nil, nil
	}
	localAPIClientModeResolver = func() (brokerLocalAPIClientFactory, error) {
		return func(_ *brokerapi.Service) brokerLocalAPI {
			return &localAPIClient{invoke: func(_ context.Context, operation string, _ any, out any) *brokerapi.ErrorResponse {
				if operation != "run_list" {
					t.Fatalf("operation = %q, want run_list", operation)
				}
				resp, ok := out.(*brokerapi.RunListResponse)
				if !ok {
					t.Fatalf("out type = %T, want *brokerapi.RunListResponse", out)
				}
				*resp = brokerapi.RunListResponse{SchemaID: "runecode.protocol.v0.RunListResponse", SchemaVersion: "0.1.0", RequestID: "req-live", Runs: []brokerapi.RunSummary{}}
				return nil
			}, invokeSecret: func(context.Context, string, any, []byte, any) *brokerapi.ErrorResponse { return nil }}
		}, nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"run-list"}, stdout, stderr); err != nil {
		t.Fatalf("run(run-list) returned error: %v", err)
	}
	if serviceFactoryCalled {
		t.Fatal("brokerServiceFactory was called for live IPC command")
	}
}

func TestRunLiveIPCCommandDialsRepoScopedLocalIPC(t *testing.T) {
	originalFactory := brokerServiceFactory
	originalResolver := localAPIClientModeResolver
	t.Cleanup(func() {
		brokerServiceFactory = originalFactory
		localAPIClientModeResolver = originalResolver
	})

	serviceFactoryCalled := false
	brokerServiceFactory = func(brokerServiceRoots) (*brokerapi.Service, error) {
		serviceFactoryCalled = true
		return nil, nil
	}
	dialCalled := false
	localAPIClientModeResolver = func() (brokerLocalAPIClientFactory, error) {
		client, err := newLiveIPCLocalAPIClientWithDeps(context.Background(), liveIPCClientDeps{
			resolveScope: func(localbootstrap.ResolveInput) (localbootstrap.RepoScope, error) {
				return localbootstrap.RepoScope{
					RepositoryRoot:  "/repo/live",
					LocalRuntimeDir: "/runtime/live",
					LocalSocketName: "broker.sock",
				}, nil
			},
			dialClient: func(_ context.Context, cfg brokerapi.LocalIPCConfig) (*brokerapi.LocalRPCClient, error) {
				dialCalled = true
				if cfg.RepositoryRoot != "/repo/live" {
					t.Fatalf("RepositoryRoot = %q, want /repo/live", cfg.RepositoryRoot)
				}
				if cfg.RuntimeDir != "/runtime/live" {
					t.Fatalf("RuntimeDir = %q, want /runtime/live", cfg.RuntimeDir)
				}
				if cfg.SocketName != "broker.sock" {
					t.Fatalf("SocketName = %q, want broker.sock", cfg.SocketName)
				}
				return &brokerapi.LocalRPCClient{}, nil
			},
		})
		if err != nil {
			return nil, err
		}
		return func(_ *brokerapi.Service) brokerLocalAPI { return client }, nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"session-list"}, stdout, stderr)
	if err == nil {
		t.Fatal("session-list expected error for disconnected local rpc test client")
	}
	if !dialCalled {
		t.Fatal("dialBrokerLocalRPC was not called for live IPC command")
	}
	if serviceFactoryCalled {
		t.Fatal("brokerServiceFactory was called for live IPC command")
	}
}

func TestRunInProcessCommandBuildsServiceWithoutLiveIPCDial(t *testing.T) {
	setBrokerServiceForTest(t)
	originalFactory := brokerServiceFactory
	originalResolver := localAPIClientModeResolver
	t.Cleanup(func() {
		brokerServiceFactory = originalFactory
		localAPIClientModeResolver = originalResolver
	})

	serviceFactoryCalled := false
	brokerServiceFactory = func(roots brokerServiceRoots) (*brokerapi.Service, error) {
		serviceFactoryCalled = true
		return originalFactory(roots)
	}
	dialCalled := false
	localAPIClientModeResolver = func() (brokerLocalAPIClientFactory, error) {
		return func(service *brokerapi.Service) brokerLocalAPI {
			dialCalled = true
			return newInProcessLocalAPIClient(service)
		}, nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"show-policy"}, stdout, stderr)
	if err != nil {
		t.Fatalf("run(show-policy) returned error: %v", err)
	}
	if !serviceFactoryCalled {
		t.Fatal("brokerServiceFactory was not called for in-process command")
	}
	if dialCalled {
		t.Fatal("localAPIClientModeResolver should not be consulted for in-process command")
	}
}

func installLiveSessionAuthorityStub(t *testing.T) *bool {
	t.Helper()
	originalFactory := brokerServiceFactory
	originalResolver := localAPIClientModeResolver
	serviceFactoryCalled := false
	t.Cleanup(func() {
		brokerServiceFactory = originalFactory
		localAPIClientModeResolver = originalResolver
	})
	brokerServiceFactory = func(brokerServiceRoots) (*brokerapi.Service, error) {
		serviceFactoryCalled = true
		return nil, nil
	}
	localAPIClientModeResolver = func() (brokerLocalAPIClientFactory, error) {
		return func(_ *brokerapi.Service) brokerLocalAPI {
			return newLiveSessionAuthorityClientForTest()
		}, nil
	}
	return &serviceFactoryCalled
}

func newLiveSessionAuthorityClientForTest() brokerLocalAPI {
	return &localAPIClient{
		invoke: func(_ context.Context, operation string, _ any, out any) *brokerapi.ErrorResponse {
			switch operation {
			case "session_list":
				resp := out.(*brokerapi.SessionListResponse)
				resp.SchemaID = "runecode.protocol.v0.SessionListResponse"
				resp.SchemaVersion = "0.1.0"
				resp.RequestID = "req-live-session-list"
				resp.Sessions = []brokerapi.SessionSummary{testLiveSessionSummary()}
				return nil
			case "session_get":
				resp := out.(*brokerapi.SessionGetResponse)
				resp.SchemaID = "runecode.protocol.v0.SessionGetResponse"
				resp.SchemaVersion = "0.1.0"
				resp.RequestID = "req-live-session-get"
				resp.Session = testLiveSessionDetail()
				return nil
			default:
				err := brokerapi.ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: "req-live-op", Error: brokerapi.ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "broker_validation_operation_invalid", Category: "validation", Message: "unexpected op"}}
				return &err
			}
		},
		invokeSecret: func(context.Context, string, any, []byte, any) *brokerapi.ErrorResponse { return nil },
	}
}

func testLiveSessionSummary() brokerapi.SessionSummary {
	return brokerapi.SessionSummary{
		SchemaID:              "runecode.protocol.v0.SessionSummary",
		SchemaVersion:         "0.1.0",
		Identity:              brokerapi.SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-live-001", WorkspaceID: "workspace-live", CreatedAt: "2026-01-01T00:00:00Z"},
		UpdatedAt:             "2026-01-01T00:00:00Z",
		Status:                "active",
		WorkPosture:           "running",
		LastActivityKind:      "run_progress",
		TurnCount:             1,
		LinkedRunCount:        1,
		LinkedApprovalCount:   0,
		LinkedArtifactCount:   0,
		LinkedAuditEventCount: 0,
		HasIncompleteTurn:     false,
	}
}

func testLiveSessionDetail() brokerapi.SessionDetail {
	return brokerapi.SessionDetail{
		SchemaID:      "runecode.protocol.v0.SessionDetail",
		SchemaVersion: "0.1.0",
		Summary:       testLiveSessionSummary(),
	}
}

func TestSessionVisibilityUsesLiveAuthorityOverInProcessState(t *testing.T) {
	serviceFactoryCalled := installLiveSessionAuthorityStub(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"session-list"}, stdout, stderr); err != nil {
		t.Fatalf("run(session-list) returned error: %v", err)
	}
	list := []brokerapi.SessionSummary{}
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		t.Fatalf("session-list output decode error: %v", err)
	}
	if len(list) != 1 || list[0].Identity.SessionID != "sess-live-001" {
		t.Fatalf("session-list output = %+v, want live session sess-live-001", list)
	}

	stdout.Reset()
	if err := run([]string{"session-get", "--session-id", "sess-live-001"}, stdout, stderr); err != nil {
		t.Fatalf("run(session-get) returned error: %v", err)
	}
	detail := brokerapi.SessionDetail{}
	if err := json.Unmarshal(stdout.Bytes(), &detail); err != nil {
		t.Fatalf("session-get output decode error: %v", err)
	}
	if detail.Summary.Identity.SessionID != "sess-live-001" || detail.Summary.Identity.WorkspaceID != "workspace-live" {
		t.Fatalf("session-get output = %+v, want live authority session", detail)
	}
	if *serviceFactoryCalled {
		t.Fatal("brokerServiceFactory was called for live-authority session commands")
	}
}

func TestLiveIPCCommandFailsDeterministicallyWhenBrokerUnavailable(t *testing.T) {
	originalFactory := brokerServiceFactory
	originalResolver := localAPIClientModeResolver
	t.Cleanup(func() {
		brokerServiceFactory = originalFactory
		localAPIClientModeResolver = originalResolver
	})

	brokerServiceFactory = func(brokerServiceRoots) (*brokerapi.Service, error) {
		t.Fatal("brokerServiceFactory should not be called for live IPC command")
		return nil, nil
	}
	localAPIClientModeResolver = func() (brokerLocalAPIClientFactory, error) {
		_, err := newLiveIPCLocalAPIClientWithDeps(context.Background(), liveIPCClientDeps{
			resolveScope: func(localbootstrap.ResolveInput) (localbootstrap.RepoScope, error) {
				return localbootstrap.RepoScope{RepositoryRoot: "/repo/live", LocalRuntimeDir: "/runtime/live", LocalSocketName: "broker.sock"}, nil
			},
			dialClient: func(context.Context, brokerapi.LocalIPCConfig) (*brokerapi.LocalRPCClient, error) {
				return nil, errors.New("no live broker reachable")
			},
		})
		return nil, err
	}

	err := run([]string{"session-list"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("session-list expected deterministic live IPC unavailability error")
	}
	const want = "gateway_failure: repo-scoped local broker is not reachable"
	if err.Error() != want {
		t.Fatalf("session-list error = %q, want %q", err.Error(), want)
	}
}

func TestLiveIPCCommandClassificationIncludesSessionCommands(t *testing.T) {
	handlers := commandHandlers()
	for _, command := range []string{"session-list", "session-get", "session-execution-trigger", "provider-credential-lease-issue"} {
		spec, ok := handlers[command]
		if !ok {
			t.Fatalf("missing command handler for %q", command)
		}
		if spec.apiMode != brokerCommandAPIModeLiveIPC {
			t.Fatalf("%s apiMode = %q, want %q", command, spec.apiMode, brokerCommandAPIModeLiveIPC)
		}
		if spec.requiresStore {
			t.Fatalf("%s requiresStore = true, want false for live IPC", command)
		}
	}

	for _, command := range []string{"put-artifact", "seed-dev-manual-scenario"} {
		spec, ok := handlers[command]
		if !ok {
			t.Fatalf("missing command handler for %q", command)
		}
		if spec.apiMode != brokerCommandAPIModeInProcess {
			t.Fatalf("%s apiMode = %q, want %q", command, spec.apiMode, brokerCommandAPIModeInProcess)
		}
		if !spec.requiresStore {
			t.Fatalf("%s requiresStore = false, want true for local/admin command", command)
		}
	}
}

func TestLiveIPCCommandsRejectGlobalStoreOverrideFlags(t *testing.T) {
	setBrokerServiceForTest(t)

	err := run([]string{"--state-root", "/tmp/custom-state", "session-list"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("session-list expected usage error when --state-root is set")
	}
	usageErr, ok := err.(*usageError)
	if !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
	if got := usageErr.Error(); got != "session-list uses repo-scoped live broker IPC; --state-root and --audit-ledger-root are only supported for local in-process commands" {
		t.Fatalf("error message = %q", got)
	}
}
