package main

import (
	"context"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func (c *localAPIClient) AuditVerificationGet(ctx context.Context, req brokerapi.AuditVerificationGetRequest) (brokerapi.AuditVerificationGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.AuditVerificationGetResponse{}
	return resp, c.invoke(ctx, "audit_verification_get", req, &resp)
}

func (c *localAPIClient) AuditFinalizeVerify(ctx context.Context, req brokerapi.AuditFinalizeVerifyRequest) (brokerapi.AuditFinalizeVerifyResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.AuditFinalizeVerifyResponse{}
	return resp, c.invoke(ctx, "audit_finalize_verify", req, &resp)
}

func (c *localAPIClient) AuditRecordGet(ctx context.Context, req brokerapi.AuditRecordGetRequest) (brokerapi.AuditRecordGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.AuditRecordGetResponse{}
	return resp, c.invoke(ctx, "audit_record_get", req, &resp)
}

func (c *localAPIClient) AuditAnchorPreflightGet(ctx context.Context, req brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.AuditAnchorPreflightGetResponse{}
	return resp, c.invoke(ctx, "audit_anchor_preflight_get", req, &resp)
}

func (c *localAPIClient) AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.AuditAnchorPresenceGetResponse{}
	return resp, c.invoke(ctx, "audit_anchor_presence_get", req, &resp)
}

func (c *localAPIClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.AuditAnchorSegmentResponse{}
	return resp, c.invoke(ctx, "audit_anchor_segment", req, &resp)
}

func (c *localAPIClient) ZKProofGenerate(ctx context.Context, req brokerapi.ZKProofGenerateRequest) (brokerapi.ZKProofGenerateResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ZKProofGenerateResponse{}
	return resp, c.invoke(ctx, "zk_proof_generate", req, &resp)
}

func (c *localAPIClient) ZKProofVerify(ctx context.Context, req brokerapi.ZKProofVerifyRequest) (brokerapi.ZKProofVerifyResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ZKProofVerifyResponse{}
	return resp, c.invoke(ctx, "zk_proof_verify", req, &resp)
}

func (c *localAPIClient) GitSetupGet(ctx context.Context, req brokerapi.GitSetupGetRequest) (brokerapi.GitSetupGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.GitSetupGetResponse{}
	return resp, c.invoke(ctx, "git_setup_get", req, &resp)
}

func (c *localAPIClient) GitSetupAuthBootstrap(ctx context.Context, req brokerapi.GitSetupAuthBootstrapRequest) (brokerapi.GitSetupAuthBootstrapResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.GitSetupAuthBootstrapResponse{}
	return resp, c.invoke(ctx, "git_setup_auth_bootstrap", req, &resp)
}

func (c *localAPIClient) GitSetupIdentityUpsert(ctx context.Context, req brokerapi.GitSetupIdentityUpsertRequest) (brokerapi.GitSetupIdentityUpsertResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.GitSetupIdentityUpsertResponse{}
	return resp, c.invoke(ctx, "git_setup_identity_upsert", req, &resp)
}

func (c *localAPIClient) ProviderSetupSessionBegin(ctx context.Context, req brokerapi.ProviderSetupSessionBeginRequest) (brokerapi.ProviderSetupSessionBeginResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProviderSetupSessionBeginResponse{}
	return resp, c.invoke(ctx, "provider_setup_session_begin", req, &resp)
}

func (c *localAPIClient) ProviderSetupSecretIngressPrepare(ctx context.Context, req brokerapi.ProviderSetupSecretIngressPrepareRequest) (brokerapi.ProviderSetupSecretIngressPrepareResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProviderSetupSecretIngressPrepareResponse{}
	return resp, c.invoke(ctx, "provider_setup_secret_ingress_prepare", req, &resp)
}

func (c *localAPIClient) ProviderSetupSecretIngressSubmit(ctx context.Context, req brokerapi.ProviderSetupSecretIngressSubmitRequest, secret []byte) (brokerapi.ProviderSetupSecretIngressSubmitResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProviderSetupSecretIngressSubmitResponse{}
	if c.invokeSecret == nil {
		errResp := brokerapi.ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: "cli-local-rpc-secret", Error: brokerapi.ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "gateway_failure", Category: "internal", Retryable: false, Message: "secret ingress transport unavailable"}}
		return resp, &errResp
	}
	return resp, c.invokeSecret(ctx, "provider_setup_secret_ingress_submit", req, secret, &resp)
}

func (c *localAPIClient) ProviderValidationBegin(ctx context.Context, req brokerapi.ProviderValidationBeginRequest) (brokerapi.ProviderValidationBeginResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProviderValidationBeginResponse{}
	return resp, c.invoke(ctx, "provider_validation_begin", req, &resp)
}

func (c *localAPIClient) ProviderValidationCommit(ctx context.Context, req brokerapi.ProviderValidationCommitRequest) (brokerapi.ProviderValidationCommitResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProviderValidationCommitResponse{}
	return resp, c.invoke(ctx, "provider_validation_commit", req, &resp)
}

func (c *localAPIClient) ProviderCredentialLeaseIssue(ctx context.Context, req brokerapi.ProviderCredentialLeaseIssueRequest) (brokerapi.ProviderCredentialLeaseIssueResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProviderCredentialLeaseIssueResponse{}
	return resp, c.invoke(ctx, "provider_credential_lease_issue", req, &resp)
}

func (c *localAPIClient) ProviderProfileList(ctx context.Context, req brokerapi.ProviderProfileListRequest) (brokerapi.ProviderProfileListResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProviderProfileListResponse{}
	return resp, c.invoke(ctx, "provider_profile_list", req, &resp)
}

func (c *localAPIClient) ProviderProfileGet(ctx context.Context, req brokerapi.ProviderProfileGetRequest) (brokerapi.ProviderProfileGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProviderProfileGetResponse{}
	return resp, c.invoke(ctx, "provider_profile_get", req, &resp)
}

func (c *localAPIClient) ProjectSubstrateGet(ctx context.Context, req brokerapi.ProjectSubstrateGetRequest) (brokerapi.ProjectSubstrateGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProjectSubstrateGetResponse{}
	return resp, c.invoke(ctx, "project_substrate_get", req, &resp)
}

func (c *localAPIClient) ProjectSubstratePostureGet(ctx context.Context, req brokerapi.ProjectSubstratePostureGetRequest) (brokerapi.ProjectSubstratePostureGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProjectSubstratePostureGetResponse{}
	return resp, c.invoke(ctx, "project_substrate_posture_get", req, &resp)
}

func (c *localAPIClient) ProjectSubstrateAdopt(ctx context.Context, req brokerapi.ProjectSubstrateAdoptRequest) (brokerapi.ProjectSubstrateAdoptResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProjectSubstrateAdoptResponse{}
	return resp, c.invoke(ctx, "project_substrate_adopt", req, &resp)
}

func (c *localAPIClient) ProjectSubstrateInitPreview(ctx context.Context, req brokerapi.ProjectSubstrateInitPreviewRequest) (brokerapi.ProjectSubstrateInitPreviewResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProjectSubstrateInitPreviewResponse{}
	return resp, c.invoke(ctx, "project_substrate_init_preview", req, &resp)
}

func (c *localAPIClient) ProjectSubstrateInitApply(ctx context.Context, req brokerapi.ProjectSubstrateInitApplyRequest) (brokerapi.ProjectSubstrateInitApplyResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProjectSubstrateInitApplyResponse{}
	return resp, c.invoke(ctx, "project_substrate_init_apply", req, &resp)
}

func (c *localAPIClient) ProjectSubstrateUpgradePreview(ctx context.Context, req brokerapi.ProjectSubstrateUpgradePreviewRequest) (brokerapi.ProjectSubstrateUpgradePreviewResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProjectSubstrateUpgradePreviewResponse{}
	return resp, c.invoke(ctx, "project_substrate_upgrade_preview", req, &resp)
}

func (c *localAPIClient) ProjectSubstrateUpgradeApply(ctx context.Context, req brokerapi.ProjectSubstrateUpgradeApplyRequest) (brokerapi.ProjectSubstrateUpgradeApplyResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ProjectSubstrateUpgradeApplyResponse{}
	return resp, c.invoke(ctx, "project_substrate_upgrade_apply", req, &resp)
}

func (c *localAPIClient) GitRemoteMutationPrepare(ctx context.Context, req brokerapi.GitRemoteMutationPrepareRequest) (brokerapi.GitRemoteMutationPrepareResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.GitRemoteMutationPrepareResponse{}
	return resp, c.invoke(ctx, "git_remote_mutation_prepare", req, &resp)
}

func (c *localAPIClient) GitRemoteMutationGet(ctx context.Context, req brokerapi.GitRemoteMutationGetRequest) (brokerapi.GitRemoteMutationGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.GitRemoteMutationGetResponse{}
	return resp, c.invoke(ctx, "git_remote_mutation_get", req, &resp)
}

func (c *localAPIClient) GitRemoteMutationIssueExecuteLease(ctx context.Context, req brokerapi.GitRemoteMutationIssueExecuteLeaseRequest) (brokerapi.GitRemoteMutationIssueExecuteLeaseResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.GitRemoteMutationIssueExecuteLeaseResponse{}
	return resp, c.invoke(ctx, "git_remote_mutation_issue_execute_lease", req, &resp)
}

func (c *localAPIClient) GitRemoteMutationExecute(ctx context.Context, req brokerapi.GitRemoteMutationExecuteRequest) (brokerapi.GitRemoteMutationExecuteResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.GitRemoteMutationExecuteResponse{}
	return resp, c.invoke(ctx, "git_remote_mutation_execute", req, &resp)
}

func (c *localAPIClient) ExternalAnchorMutationPrepare(ctx context.Context, req brokerapi.ExternalAnchorMutationPrepareRequest) (brokerapi.ExternalAnchorMutationPrepareResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ExternalAnchorMutationPrepareResponse{}
	return resp, c.invoke(ctx, "external_anchor_mutation_prepare", req, &resp)
}

func (c *localAPIClient) ExternalAnchorMutationGet(ctx context.Context, req brokerapi.ExternalAnchorMutationGetRequest) (brokerapi.ExternalAnchorMutationGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ExternalAnchorMutationGetResponse{}
	return resp, c.invoke(ctx, "external_anchor_mutation_get", req, &resp)
}

func (c *localAPIClient) ExternalAnchorMutationIssueExecuteLease(ctx context.Context, req brokerapi.ExternalAnchorMutationIssueExecuteLeaseRequest) (brokerapi.ExternalAnchorMutationIssueExecuteLeaseResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ExternalAnchorMutationIssueExecuteLeaseResponse{}
	return resp, c.invoke(ctx, "external_anchor_mutation_issue_execute_lease", req, &resp)
}

func (c *localAPIClient) ExternalAnchorMutationExecute(ctx context.Context, req brokerapi.ExternalAnchorMutationExecuteRequest) (brokerapi.ExternalAnchorMutationExecuteResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ExternalAnchorMutationExecuteResponse{}
	return resp, c.invoke(ctx, "external_anchor_mutation_execute", req, &resp)
}

func (c *localAPIClient) DependencyCacheEnsure(ctx context.Context, req brokerapi.DependencyCacheEnsureRequest) (brokerapi.DependencyCacheEnsureResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.DependencyCacheEnsureResponse{}
	return resp, c.invoke(ctx, "dependency_cache_ensure", req, &resp)
}

func (c *localAPIClient) DependencyFetchRegistry(ctx context.Context, req brokerapi.DependencyFetchRegistryRequest) (brokerapi.DependencyFetchRegistryResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.DependencyFetchRegistryResponse{}
	return resp, c.invoke(ctx, "dependency_fetch_registry", req, &resp)
}

func (c *localAPIClient) DependencyCacheHandoff(ctx context.Context, req brokerapi.DependencyCacheHandoffRequest) (brokerapi.DependencyCacheHandoffResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.DependencyCacheHandoffResponse{}
	return resp, c.invoke(ctx, "dependency_cache_handoff", req, &resp)
}
