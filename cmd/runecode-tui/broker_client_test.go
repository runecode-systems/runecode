package main

import (
	"context"
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
