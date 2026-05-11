package main

import (
	"context"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func (f *reloadAwareBrokerClient) GitSetupGet(ctx context.Context, provider string) (brokerapi.GitSetupGetResponse, error) {
	return (&fakeBrokerClient{}).GitSetupGet(ctx, provider)
}

func (f *reloadAwareBrokerClient) ProviderSetupSessionBegin(ctx context.Context, req brokerapi.ProviderSetupSessionBeginRequest) (brokerapi.ProviderSetupSessionBeginResponse, error) {
	return (&fakeBrokerClient{}).ProviderSetupSessionBegin(ctx, req)
}

func (f *reloadAwareBrokerClient) ProviderSetupSecretIngressPrepare(ctx context.Context, req brokerapi.ProviderSetupSecretIngressPrepareRequest) (brokerapi.ProviderSetupSecretIngressPrepareResponse, error) {
	return (&fakeBrokerClient{}).ProviderSetupSecretIngressPrepare(ctx, req)
}

func (f *reloadAwareBrokerClient) ProviderSetupSecretIngressSubmit(ctx context.Context, req brokerapi.ProviderSetupSecretIngressSubmitRequest, secret []byte) (brokerapi.ProviderSetupSecretIngressSubmitResponse, error) {
	return (&fakeBrokerClient{}).ProviderSetupSecretIngressSubmit(ctx, req, secret)
}

func (f *reloadAwareBrokerClient) ProviderCredentialLeaseIssue(ctx context.Context, req brokerapi.ProviderCredentialLeaseIssueRequest) (brokerapi.ProviderCredentialLeaseIssueResponse, error) {
	return (&fakeBrokerClient{}).ProviderCredentialLeaseIssue(ctx, req)
}

func (f *reloadAwareBrokerClient) ProviderProfileList(ctx context.Context) (brokerapi.ProviderProfileListResponse, error) {
	return (&fakeBrokerClient{}).ProviderProfileList(ctx)
}

func (f *reloadAwareBrokerClient) ProviderProfileGet(ctx context.Context, providerProfileID string) (brokerapi.ProviderProfileGetResponse, error) {
	return (&fakeBrokerClient{}).ProviderProfileGet(ctx, providerProfileID)
}

func (f *reloadAwareBrokerClient) GitSetupAuthBootstrap(ctx context.Context, req brokerapi.GitSetupAuthBootstrapRequest) (brokerapi.GitSetupAuthBootstrapResponse, error) {
	return (&fakeBrokerClient{}).GitSetupAuthBootstrap(ctx, req)
}

func (f *reloadAwareBrokerClient) GitSetupIdentityUpsert(ctx context.Context, req brokerapi.GitSetupIdentityUpsertRequest) (brokerapi.GitSetupIdentityUpsertResponse, error) {
	return (&fakeBrokerClient{}).GitSetupIdentityUpsert(ctx, req)
}
