package main

import (
	"context"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func (c *rpcBrokerClient) GitSetupGet(ctx context.Context, provider string) (brokerapi.GitSetupGetResponse, error) {
	req := brokerapi.GitSetupGetRequest{SchemaID: "runecode.protocol.v0.GitSetupGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("git-setup-get"), Provider: provider}
	resp := brokerapi.GitSetupGetResponse{}
	return resp, c.invoke(ctx, "git_setup_get", req, &resp)
}

func (c *rpcBrokerClient) GitSetupAuthBootstrap(ctx context.Context, req brokerapi.GitSetupAuthBootstrapRequest) (brokerapi.GitSetupAuthBootstrapResponse, error) {
	req.SchemaID = "runecode.protocol.v0.GitSetupAuthBootstrapRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("git-setup-auth-bootstrap")
	resp := brokerapi.GitSetupAuthBootstrapResponse{}
	return resp, c.invoke(ctx, "git_setup_auth_bootstrap", req, &resp)
}

func (c *rpcBrokerClient) GitSetupIdentityUpsert(ctx context.Context, req brokerapi.GitSetupIdentityUpsertRequest) (brokerapi.GitSetupIdentityUpsertResponse, error) {
	req.SchemaID = "runecode.protocol.v0.GitSetupIdentityUpsertRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("git-setup-identity-upsert")
	resp := brokerapi.GitSetupIdentityUpsertResponse{}
	return resp, c.invoke(ctx, "git_setup_identity_upsert", req, &resp)
}
