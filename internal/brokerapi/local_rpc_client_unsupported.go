//go:build !linux

package brokerapi

import "context"

func DialLocalRPC(_ context.Context, _ LocalIPCConfig) (*LocalRPCClient, error) {
	return nil, ErrPeerCredentialsUnavailable
}

func DialLocalRPCWithLimits(_ context.Context, _ LocalIPCConfig, _ Limits) (*LocalRPCClient, error) {
	return nil, ErrPeerCredentialsUnavailable
}

type LocalRPCClient struct{}

func (c *LocalRPCClient) Close() error { return ErrPeerCredentialsUnavailable }

func (c *LocalRPCClient) Invoke(_ context.Context, _ string, _ any, _ any) *ErrorResponse {
	err := toErrorResponse(defaultRequestIDFallback, "gateway_failure", "internal", false, "local rpc client is linux-only for MVP")
	return &err
}
