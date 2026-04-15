package main

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type fakeRPCInvoker struct {
	invokeFn func(ctx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse
}

type runListDialProbe struct {
	wantCfg    brokerapi.LocalIPCConfig
	dialedCfg  brokerapi.LocalIPCConfig
	gotOp      string
	assertions func(t *testing.T, req brokerapi.RunListRequest)
}

func (f *fakeRPCInvoker) Invoke(ctx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse {
	if f.invokeFn != nil {
		return f.invokeFn(ctx, operation, request, out)
	}
	return nil
}

func (f *fakeRPCInvoker) Close() error { return nil }

func TestRPCBrokerClientUsesLocalIPCDialAndTypedRequestContract(t *testing.T) {
	origConfigProvider := localIPCConfigProvider
	origDialer := localRPCDialer
	t.Cleanup(func() {
		localIPCConfigProvider = origConfigProvider
		localRPCDialer = origDialer
	})

	probe := runListDialProbe{
		wantCfg: brokerapi.LocalIPCConfig{RuntimeDir: "/tmp/test-runtime", SocketName: "broker.sock"},
		assertions: func(t *testing.T, req brokerapi.RunListRequest) {
			t.Helper()
			if req.SchemaID != "runecode.protocol.v0.RunListRequest" {
				t.Fatalf("expected typed schema id, got %q", req.SchemaID)
			}
			if req.SchemaVersion != localAPISchemaVersion {
				t.Fatalf("expected schema version %q, got %q", localAPISchemaVersion, req.SchemaVersion)
			}
			if !strings.HasPrefix(req.RequestID, "run-list-") {
				t.Fatalf("expected typed request id prefix, got %q", req.RequestID)
			}
			if req.Limit != 12 {
				t.Fatalf("expected limit 12, got %d", req.Limit)
			}
		},
	}
	configureRunListDialProbe(t, &probe)

	client := &rpcBrokerClient{}
	if _, err := client.RunList(context.Background(), 12); err != nil {
		t.Fatalf("RunList returned error: %v", err)
	}
	if probe.dialedCfg != probe.wantCfg {
		t.Fatalf("expected dial config %+v, got %+v", probe.wantCfg, probe.dialedCfg)
	}
	if probe.gotOp != "run_list" {
		t.Fatalf("expected typed operation run_list, got %q", probe.gotOp)
	}
}

func configureRunListDialProbe(t *testing.T, probe *runListDialProbe) {
	t.Helper()
	localIPCConfigProvider = func() (brokerapi.LocalIPCConfig, error) {
		return probe.wantCfg, nil
	}
	localRPCDialer = func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (localRPCInvoker, error) {
		_ = ctx
		probe.dialedCfg = cfg
		return &fakeRPCInvoker{invokeFn: func(ctx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse {
			_ = ctx
			_ = out
			probe.gotOp = operation
			req, ok := request.(brokerapi.RunListRequest)
			if !ok {
				t.Fatalf("expected brokerapi.RunListRequest, got %T", request)
			}
			probe.assertions(t, req)
			return nil
		}}, nil
	}
}

func TestLocalBrokerBoundaryPostureMentionsLocalIPCAndPeerAuth(t *testing.T) {
	posture := localBrokerBoundaryPosture()
	if !strings.Contains(posture, "Local broker API only") {
		t.Fatalf("expected local API posture, got %q", posture)
	}
	if !strings.Contains(posture, "local IPC") {
		t.Fatalf("expected local IPC mention, got %q", posture)
	}
	if !strings.Contains(posture, "OS peer auth") {
		t.Fatalf("expected OS peer auth mention, got %q", posture)
	}
}

func TestRPCBrokerClientInvokeSurfacesActionableLocalIPCConfigError(t *testing.T) {
	origConfigProvider := localIPCConfigProvider
	origDialer := localRPCDialer
	t.Cleanup(func() {
		localIPCConfigProvider = origConfigProvider
		localRPCDialer = origDialer
	})

	localIPCConfigProvider = func() (brokerapi.LocalIPCConfig, error) {
		return brokerapi.LocalIPCConfig{}, errors.New("local ipc listener is linux-only for MVP")
	}

	client := &rpcBrokerClient{}
	_, err := client.RunList(context.Background(), 1)
	if err == nil {
		t.Fatal("expected RunList to fail")
	}
	if got := err.Error(); got != "local ipc listener is linux-only for MVP" {
		t.Fatalf("expected actionable config error, got %q", got)
	}
}

func TestRPCBrokerClientInvokeSurfacesActionableLocalIPCDialError(t *testing.T) {
	origConfigProvider := localIPCConfigProvider
	origDialer := localRPCDialer
	t.Cleanup(func() {
		localIPCConfigProvider = origConfigProvider
		localRPCDialer = origDialer
	})

	localIPCConfigProvider = func() (brokerapi.LocalIPCConfig, error) {
		return brokerapi.LocalIPCConfig{RuntimeDir: "/tmp/runtime", SocketName: "broker.sock"}, nil
	}
	localRPCDialer = func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (localRPCInvoker, error) {
		_ = ctx
		_ = cfg
		return nil, errors.New("runtime directory is required")
	}

	client := &rpcBrokerClient{}
	_, err := client.RunList(context.Background(), 1)
	if err == nil {
		t.Fatal("expected RunList to fail")
	}
	if got := err.Error(); got != "runtime directory is required" {
		t.Fatalf("expected actionable dial error, got %q", got)
	}
}

func TestRPCBrokerClientInvokeFallsBackForOpaqueLocalIPCErrors(t *testing.T) {
	origConfigProvider := localIPCConfigProvider
	origDialer := localRPCDialer
	t.Cleanup(func() {
		localIPCConfigProvider = origConfigProvider
		localRPCDialer = origDialer
	})

	localIPCConfigProvider = func() (brokerapi.LocalIPCConfig, error) {
		return brokerapi.LocalIPCConfig{}, errors.New("unexpected filesystem path /tmp/private/socket")
	}

	client := &rpcBrokerClient{}
	_, err := client.RunList(context.Background(), 1)
	if err == nil {
		t.Fatal("expected RunList to fail")
	}
	if got := err.Error(); got != "local_ipc_config_error" {
		t.Fatalf("expected fallback config error code, got %q", got)
	}
}

func TestRPCBrokerClientInvokeReturnsDialFallbackWhenDialerReturnsNilClient(t *testing.T) {
	origConfigProvider := localIPCConfigProvider
	origDialer := localRPCDialer
	t.Cleanup(func() {
		localIPCConfigProvider = origConfigProvider
		localRPCDialer = origDialer
	})

	localIPCConfigProvider = func() (brokerapi.LocalIPCConfig, error) {
		return brokerapi.LocalIPCConfig{RuntimeDir: "/tmp/runtime", SocketName: "broker.sock"}, nil
	}
	localRPCDialer = func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (localRPCInvoker, error) {
		_ = ctx
		_ = cfg
		return nil, nil
	}

	client := &rpcBrokerClient{}
	_, err := client.RunList(context.Background(), 1)
	if err == nil {
		t.Fatal("expected RunList to fail")
	}
	if got := err.Error(); got != "local_ipc_dial_error" {
		t.Fatalf("expected local_ipc_dial_error fallback for nil client, got %q", got)
	}
}

func TestDecodeArtifactStreamRequiresTerminalEvent(t *testing.T) {
	chunk := base64.StdEncoding.EncodeToString([]byte("partial"))
	_, err := decodeArtifactStream([]brokerapi.ArtifactStreamEvent{{EventType: "artifact_stream_chunk", ChunkBase64: chunk}})
	if err == nil {
		t.Fatal("expected decodeArtifactStream to reject missing terminal event")
	}
	if got := err.Error(); got != "artifact_stream_incomplete" {
		t.Fatalf("expected artifact_stream_incomplete, got %q", got)
	}
}

func TestLocalIPCConfigProviderWithOverridesUsesFallbackWhenBaseUnavailable(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	provider := localIPCConfigProviderWithOverrides(
		func() (brokerapi.LocalIPCConfig, error) {
			return brokerapi.LocalIPCConfig{}, errors.New("local ipc listener is linux-only for MVP")
		},
		runtimeDir,
		"broker.dev.sock",
	)
	cfg, err := provider()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if cfg.RuntimeDir != runtimeDir || cfg.SocketName != "broker.dev.sock" {
		t.Fatalf("provider cfg = %+v, want override values", cfg)
	}
}

func TestLocalIPCConfigProviderWithOverridesPropagatesBaseErrorWhenOnlyOneOverrideProvided(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	provider := localIPCConfigProviderWithOverrides(
		func() (brokerapi.LocalIPCConfig, error) {
			return brokerapi.LocalIPCConfig{}, errors.New("local ipc listener is linux-only for MVP")
		},
		runtimeDir,
		"",
	)
	_, err := provider()
	if err == nil {
		t.Fatal("provider expected error")
	}
	if got := err.Error(); got != "local ipc listener is linux-only for MVP" {
		t.Fatalf("provider error = %q", got)
	}
}

func TestLocalIPCConfigProviderWithOverridesMergesSuccessfulBaseConfig(t *testing.T) {
	baseRuntimeDir := filepath.Join(t.TempDir(), "base-runtime")
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	provider := localIPCConfigProviderWithOverrides(
		func() (brokerapi.LocalIPCConfig, error) {
			return brokerapi.LocalIPCConfig{RuntimeDir: baseRuntimeDir, SocketName: "broker.sock"}, nil
		},
		runtimeDir,
		"broker.dev.sock",
	)
	cfg, err := provider()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if cfg.RuntimeDir != runtimeDir || cfg.SocketName != "broker.dev.sock" {
		t.Fatalf("provider cfg = %+v, want merged override values", cfg)
	}
}

func TestLocalIPCConfigProviderWithOverridesPropagatesBaseErrorWhenFallbackRuntimeDirInvalid(t *testing.T) {
	provider := localIPCConfigProviderWithOverrides(
		func() (brokerapi.LocalIPCConfig, error) {
			return brokerapi.LocalIPCConfig{}, errors.New("local ipc listener is linux-only for MVP")
		},
		"relative/runtime",
		"broker.dev.sock",
	)
	_, err := provider()
	if err == nil {
		t.Fatal("provider expected base error")
	}
	if got := err.Error(); got != "local ipc listener is linux-only for MVP" {
		t.Fatalf("provider error = %q", got)
	}
}

func TestLocalIPCConfigProviderWithOverridesRejectsInvalidMergedConfigWhenBaseSucceeds(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	provider := localIPCConfigProviderWithOverrides(
		func() (brokerapi.LocalIPCConfig, error) {
			return brokerapi.LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"}, nil
		},
		"relative/runtime",
		"broker.dev.sock",
	)
	_, err := provider()
	if err == nil {
		t.Fatal("provider expected validation error")
	}
	if got := err.Error(); got != "runtime directory must be absolute" {
		t.Fatalf("provider error = %q", got)
	}
}

func TestValidatedLocalIPCConfigRejectsRootRuntimeDir(t *testing.T) {
	_, err := validatedLocalIPCConfig(brokerapi.LocalIPCConfig{RuntimeDir: string(filepath.Separator), SocketName: "broker.sock"})
	if err == nil {
		t.Fatal("validatedLocalIPCConfig expected error")
	}
	if got := err.Error(); got != "runtime directory must be a non-root absolute path" {
		t.Fatalf("validatedLocalIPCConfig error = %q", got)
	}
}

func TestLocalIPCConfigProviderWithOverridesPropagatesBaseErrorWhenFallbackSocketNameInvalid(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	provider := localIPCConfigProviderWithOverrides(
		func() (brokerapi.LocalIPCConfig, error) {
			return brokerapi.LocalIPCConfig{}, errors.New("local ipc listener is linux-only for MVP")
		},
		runtimeDir,
		"nested/broker.sock",
	)
	_, err := provider()
	if err == nil {
		t.Fatal("provider expected base error")
	}
	if got := err.Error(); got != "local ipc listener is linux-only for MVP" {
		t.Fatalf("provider error = %q", got)
	}
}

func TestLocalIPCConfigProviderWithOverridesPropagatesBaseErrorWithoutOverrides(t *testing.T) {
	provider := localIPCConfigProviderWithOverrides(
		func() (brokerapi.LocalIPCConfig, error) {
			return brokerapi.LocalIPCConfig{}, errors.New("local ipc listener is linux-only for MVP")
		},
		"",
		"",
	)
	_, err := provider()
	if err == nil {
		t.Fatal("provider expected error")
	}
	if got := err.Error(); got != "local ipc listener is linux-only for MVP" {
		t.Fatalf("provider error = %q", got)
	}
}
