package main

import (
	"context"
	"fmt"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/localbootstrap"
)

type liveIPCClientDeps struct {
	resolveScope func(localbootstrap.ResolveInput) (localbootstrap.RepoScope, error)
	dialClient   func(context.Context, brokerapi.LocalIPCConfig) (*brokerapi.LocalRPCClient, error)
}

func defaultLiveIPCClientDeps() liveIPCClientDeps {
	return liveIPCClientDeps{
		resolveScope: localbootstrap.ResolveRepoScope,
		dialClient:   brokerapi.DialLocalRPC,
	}
}

func newLiveIPCLocalAPIClient(ctx context.Context) (brokerLocalAPI, error) {
	return newLiveIPCLocalAPIClientWithDeps(ctx, defaultLiveIPCClientDeps())
}

func newLiveIPCLocalAPIClientWithDeps(ctx context.Context, deps liveIPCClientDeps) (brokerLocalAPI, error) {
	resolveScope := deps.resolveScope
	if resolveScope == nil {
		resolveScope = localbootstrap.ResolveRepoScope
	}
	dialClient := deps.dialClient
	if dialClient == nil {
		dialClient = brokerapi.DialLocalRPC
	}
	scope, err := resolveScope(localbootstrap.ResolveInput{})
	if err != nil {
		return nil, fmt.Errorf("resolve repo-scoped local broker target")
	}
	client, err := dialClient(normalizedLocalRPCContext(ctx), brokerapi.LocalIPCConfig{
		RuntimeDir:     scope.LocalRuntimeDir,
		SocketName:     scope.LocalSocketName,
		RepositoryRoot: scope.RepositoryRoot,
	})
	if err != nil {
		return nil, fmt.Errorf("repo-scoped local broker is not reachable")
	}
	return &localAPIClient{
		invoke: func(invokeCtx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse {
			return client.Invoke(normalizedLocalRPCContext(invokeCtx), operation, request, out)
		},
		invokeSecret: func(invokeCtx context.Context, operation string, request any, secret []byte, out any) *brokerapi.ErrorResponse {
			return client.InvokeSecretIngress(normalizedLocalRPCContext(invokeCtx), operation, request, secret, out)
		},
	}, nil
}
