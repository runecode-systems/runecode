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

func (f *fakeRPCInvoker) InvokeSecretIngress(ctx context.Context, operation string, request any, secret []byte, out any) *brokerapi.ErrorResponse {
	_ = secret
	return f.Invoke(ctx, operation, request, out)
}

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

func TestRPCBrokerClientGitRemoteMutationMethodsUseTypedContracts(t *testing.T) {
	origConfigProvider := localIPCConfigProvider
	origDialer := localRPCDialer
	t.Cleanup(func() {
		localIPCConfigProvider = origConfigProvider
		localRPCDialer = origDialer
	})

	localIPCConfigProvider = func() (brokerapi.LocalIPCConfig, error) {
		return brokerapi.LocalIPCConfig{RuntimeDir: "/tmp/test-runtime", SocketName: "broker.sock"}, nil
	}
	operations := make([]string, 0, 7)
	localRPCDialer = func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (localRPCInvoker, error) {
		_ = ctx
		_ = cfg
		return &fakeRPCInvoker{invokeFn: func(_ context.Context, operation string, request any, out any) *brokerapi.ErrorResponse {
			operations = append(operations, operation)
			assertGitRemoteMutationRequestContract(t, operation, request)
			return nil
		}}, nil
	}

	client := &rpcBrokerClient{}
	if _, err := client.GitRemoteMutationPrepare(context.Background(), brokerapi.GitRemoteMutationPrepareRequest{RunID: "run-1", Provider: "github", TypedRequest: map[string]any{"request_kind": "git_ref_update"}}); err != nil {
		t.Fatalf("GitRemoteMutationPrepare returned error: %v", err)
	}
	if _, err := client.GitRemoteMutationGet(context.Background(), brokerapi.GitRemoteMutationGetRequest{PreparedMutationID: "sha256:" + strings.Repeat("1", 64)}); err != nil {
		t.Fatalf("GitRemoteMutationGet returned error: %v", err)
	}
	if _, err := client.GitRemoteMutationIssueExecuteLease(context.Background(), brokerapi.GitRemoteMutationIssueExecuteLeaseRequest{PreparedMutationID: "sha256:" + strings.Repeat("1", 64)}); err != nil {
		t.Fatalf("GitRemoteMutationIssueExecuteLease returned error: %v", err)
	}
	if _, err := client.GitRemoteMutationExecute(context.Background(), brokerapi.GitRemoteMutationExecuteRequest{PreparedMutationID: "sha256:" + strings.Repeat("1", 64), ApprovalID: "sha256:" + strings.Repeat("a", 64)}); err != nil {
		t.Fatalf("GitRemoteMutationExecute returned error: %v", err)
	}
	if _, err := client.ExternalAnchorMutationPrepare(context.Background(), brokerapi.ExternalAnchorMutationPrepareRequest{RunID: "run-1", TypedRequest: map[string]any{"schema_id": "runecode.protocol.v0.ExternalAnchorSubmitRequest", "schema_version": "0.1.0", "request_kind": "external_anchor_submit_v0", "target_kind": "transparency_log", "target_descriptor": map[string]any{"descriptor_schema_id": "runecode.protocol.audit.anchor_target.transparency_log.v0", "log_id": "tui-client-test-log", "log_public_key_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "entry_encoding_profile": "jcs_v1"}, "target_descriptor_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("2", 64)}, "seal_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("3", 64)}, "outbound_payload_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("4", 64)}}}); err != nil {
		t.Fatalf("ExternalAnchorMutationPrepare returned error: %v", err)
	}
	if _, err := client.ExternalAnchorMutationGet(context.Background(), brokerapi.ExternalAnchorMutationGetRequest{PreparedMutationID: "sha256:" + strings.Repeat("1", 64)}); err != nil {
		t.Fatalf("ExternalAnchorMutationGet returned error: %v", err)
	}
	if _, err := client.ExternalAnchorMutationExecute(context.Background(), brokerapi.ExternalAnchorMutationExecuteRequest{PreparedMutationID: "sha256:" + strings.Repeat("1", 64), ApprovalID: "sha256:" + strings.Repeat("a", 64)}); err != nil {
		t.Fatalf("ExternalAnchorMutationExecute returned error: %v", err)
	}

	if got := strings.Join(operations, ","); got != "git_remote_mutation_prepare,git_remote_mutation_get,git_remote_mutation_issue_execute_lease,git_remote_mutation_execute,external_anchor_mutation_prepare,external_anchor_mutation_get,external_anchor_mutation_execute" {
		t.Fatalf("operations=%q", got)
	}
}

func TestRPCBrokerClientDependencyMethodsUseTypedContracts(t *testing.T) {
	origConfigProvider := localIPCConfigProvider
	origDialer := localRPCDialer
	t.Cleanup(func() {
		localIPCConfigProvider = origConfigProvider
		localRPCDialer = origDialer
	})

	localIPCConfigProvider = func() (brokerapi.LocalIPCConfig, error) {
		return brokerapi.LocalIPCConfig{RuntimeDir: "/tmp/test-runtime", SocketName: "broker.sock"}, nil
	}
	operations := make([]string, 0, 3)
	localRPCDialer = func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (localRPCInvoker, error) {
		_ = ctx
		_ = cfg
		return &fakeRPCInvoker{invokeFn: func(_ context.Context, operation string, request any, out any) *brokerapi.ErrorResponse {
			operations = append(operations, operation)
			assertDependencyRequestContract(t, operation, request)
			return nil
		}}, nil
	}

	client := &rpcBrokerClient{}
	cacheReq := brokerapi.DependencyCacheEnsureRequest{RunID: "run-1"}
	if _, err := client.DependencyCacheEnsure(context.Background(), cacheReq); err != nil {
		t.Fatalf("DependencyCacheEnsure returned error: %v", err)
	}
	fetchReq := brokerapi.DependencyFetchRegistryRequest{RunID: "run-1"}
	if _, err := client.DependencyFetchRegistry(context.Background(), fetchReq); err != nil {
		t.Fatalf("DependencyFetchRegistry returned error: %v", err)
	}
	handoffReq := brokerapi.DependencyCacheHandoffRequest{RequestDigest: parseDigestIdentity("sha256:" + strings.Repeat("a", 64)), ConsumerRole: "workspace"}
	if _, err := client.DependencyCacheHandoff(context.Background(), handoffReq); err != nil {
		t.Fatalf("DependencyCacheHandoff returned error: %v", err)
	}

	if got := strings.Join(operations, ","); got != "dependency_cache_ensure,dependency_fetch_registry,dependency_cache_handoff" {
		t.Fatalf("operations=%q", got)
	}
}

func assertDependencyRequestContract(t *testing.T, operation string, request any) {
	t.Helper()
	switch operation {
	case "dependency_cache_ensure":
		req, ok := request.(brokerapi.DependencyCacheEnsureRequest)
		if !ok {
			t.Fatalf("dependency_cache_ensure request type=%T", request)
		}
		assertDependencyRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.DependencyCacheEnsureRequest", "dependency-cache-ensure-")
	case "dependency_fetch_registry":
		req, ok := request.(brokerapi.DependencyFetchRegistryRequest)
		if !ok {
			t.Fatalf("dependency_fetch_registry request type=%T", request)
		}
		assertDependencyRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.DependencyFetchRegistryRequest", "dependency-fetch-registry-")
	case "dependency_cache_handoff":
		req, ok := request.(brokerapi.DependencyCacheHandoffRequest)
		if !ok {
			t.Fatalf("dependency_cache_handoff request type=%T", request)
		}
		assertDependencyRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.DependencyCacheHandoffRequest", "dependency-cache-handoff-")
	default:
		t.Fatalf("unexpected operation %q", operation)
	}
}

func assertDependencyRequestMetadata(t *testing.T, schemaID, schemaVersion, requestID, wantSchemaID, wantPrefix string) {
	t.Helper()
	if schemaID != wantSchemaID || schemaVersion != localAPISchemaVersion || !strings.HasPrefix(requestID, wantPrefix) {
		t.Fatalf("dependency request metadata invalid: schema_id=%q schema_version=%q request_id=%q", schemaID, schemaVersion, requestID)
	}
}

func assertGitRemoteMutationRequestContract(t *testing.T, operation string, request any) {
	t.Helper()
	switch operation {
	case "git_remote_mutation_prepare":
		req, ok := request.(brokerapi.GitRemoteMutationPrepareRequest)
		if !ok {
			t.Fatalf("prepare request type=%T", request)
		}
		assertGitRemoteMutationRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.GitRemoteMutationPrepareRequest", "git-remote-mutation-prepare-")
	case "git_remote_mutation_get":
		req, ok := request.(brokerapi.GitRemoteMutationGetRequest)
		if !ok {
			t.Fatalf("get request type=%T", request)
		}
		assertGitRemoteMutationRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.GitRemoteMutationGetRequest", "git-remote-mutation-get-")
	case "git_remote_mutation_issue_execute_lease":
		req, ok := request.(brokerapi.GitRemoteMutationIssueExecuteLeaseRequest)
		if !ok {
			t.Fatalf("issue execute lease request type=%T", request)
		}
		assertGitRemoteMutationRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.GitRemoteMutationIssueExecuteLeaseRequest", "git-remote-mutation-issue-execute-lease-")
	case "git_remote_mutation_execute":
		req, ok := request.(brokerapi.GitRemoteMutationExecuteRequest)
		if !ok {
			t.Fatalf("execute request type=%T", request)
		}
		assertGitRemoteMutationRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.GitRemoteMutationExecuteRequest", "git-remote-mutation-execute-")
	case "external_anchor_mutation_prepare":
		req, ok := request.(brokerapi.ExternalAnchorMutationPrepareRequest)
		if !ok {
			t.Fatalf("external prepare request type=%T", request)
		}
		assertGitRemoteMutationRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.ExternalAnchorMutationPrepareRequest", "external-anchor-mutation-prepare-")
	case "external_anchor_mutation_get":
		req, ok := request.(brokerapi.ExternalAnchorMutationGetRequest)
		if !ok {
			t.Fatalf("external get request type=%T", request)
		}
		assertGitRemoteMutationRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.ExternalAnchorMutationGetRequest", "external-anchor-mutation-get-")
	case "external_anchor_mutation_execute":
		req, ok := request.(brokerapi.ExternalAnchorMutationExecuteRequest)
		if !ok {
			t.Fatalf("external execute request type=%T", request)
		}
		assertGitRemoteMutationRequestMetadata(t, req.SchemaID, req.SchemaVersion, req.RequestID, "runecode.protocol.v0.ExternalAnchorMutationExecuteRequest", "external-anchor-mutation-execute-")
	default:
		t.Fatalf("unexpected operation %q", operation)
	}
}

func assertGitRemoteMutationRequestMetadata(t *testing.T, schemaID, schemaVersion, requestID, wantSchemaID, wantPrefix string) {
	t.Helper()
	if schemaID != wantSchemaID || schemaVersion != localAPISchemaVersion || !strings.HasPrefix(requestID, wantPrefix) {
		t.Fatalf("request metadata invalid: schema_id=%q schema_version=%q request_id=%q", schemaID, schemaVersion, requestID)
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
	_, err := validatedLocalIPCConfig(brokerapi.LocalIPCConfig{RuntimeDir: localIPCRootRuntimeDirForPlatform(t), SocketName: "broker.sock"})
	if err == nil {
		t.Fatal("validatedLocalIPCConfig expected error")
	}
	if got := err.Error(); got != "runtime directory must be a non-root absolute path" {
		t.Fatalf("validatedLocalIPCConfig error = %q", got)
	}
}

func localIPCRootRuntimeDirForPlatform(t *testing.T) string {
	t.Helper()
	if volume := filepath.VolumeName(filepath.Clean(t.TempDir())); volume != "" {
		return volume + string(filepath.Separator)
	}
	return string(filepath.Separator)
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
