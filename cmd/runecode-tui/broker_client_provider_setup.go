package main

import (
	"context"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func (c *rpcBrokerClient) ProviderSetupSessionBegin(ctx context.Context, req brokerapi.ProviderSetupSessionBeginRequest) (brokerapi.ProviderSetupSessionBeginResponse, error) {
	req.SchemaID = "runecode.protocol.v0.ProviderSetupSessionBeginRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("provider-setup-session-begin")
	resp := brokerapi.ProviderSetupSessionBeginResponse{}
	return resp, c.invoke(ctx, "provider_setup_session_begin", req, &resp)
}

func (c *rpcBrokerClient) ProviderSetupSecretIngressPrepare(ctx context.Context, req brokerapi.ProviderSetupSecretIngressPrepareRequest) (brokerapi.ProviderSetupSecretIngressPrepareResponse, error) {
	req.SchemaID = "runecode.protocol.v0.ProviderSetupSecretIngressPrepareRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("provider-secret-ingress-prepare")
	resp := brokerapi.ProviderSetupSecretIngressPrepareResponse{}
	return resp, c.invoke(ctx, "provider_setup_secret_ingress_prepare", req, &resp)
}

func (c *rpcBrokerClient) ProviderSetupSecretIngressSubmit(ctx context.Context, req brokerapi.ProviderSetupSecretIngressSubmitRequest, secret []byte) (brokerapi.ProviderSetupSecretIngressSubmitResponse, error) {
	req.SchemaID = "runecode.protocol.v0.ProviderSetupSecretIngressSubmitRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("provider-secret-ingress-submit")
	resp := brokerapi.ProviderSetupSecretIngressSubmitResponse{}
	return resp, c.invokeWithSecret(ctx, "provider_setup_secret_ingress_submit", req, secret, &resp)
}

func (c *rpcBrokerClient) ProviderCredentialLeaseIssue(ctx context.Context, req brokerapi.ProviderCredentialLeaseIssueRequest) (brokerapi.ProviderCredentialLeaseIssueResponse, error) {
	req.SchemaID = "runecode.protocol.v0.ProviderCredentialLeaseIssueRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("provider-credential-lease-issue")
	resp := brokerapi.ProviderCredentialLeaseIssueResponse{}
	return resp, c.invoke(ctx, "provider_credential_lease_issue", req, &resp)
}

func (c *rpcBrokerClient) ProviderProfileList(ctx context.Context) (brokerapi.ProviderProfileListResponse, error) {
	req := brokerapi.ProviderProfileListRequest{SchemaID: "runecode.protocol.v0.ProviderProfileListRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("provider-profile-list")}
	resp := brokerapi.ProviderProfileListResponse{}
	return resp, c.invoke(ctx, "provider_profile_list", req, &resp)
}

func (c *rpcBrokerClient) ProviderProfileGet(ctx context.Context, providerProfileID string) (brokerapi.ProviderProfileGetResponse, error) {
	req := brokerapi.ProviderProfileGetRequest{SchemaID: "runecode.protocol.v0.ProviderProfileGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("provider-profile-get"), ProviderProfileID: providerProfileID}
	resp := brokerapi.ProviderProfileGetResponse{}
	return resp, c.invoke(ctx, "provider_profile_get", req, &resp)
}
