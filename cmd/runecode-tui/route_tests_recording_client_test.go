package main

import (
	"context"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type recordingBrokerClient struct {
	base  localBrokerClient
	calls []string
}

func newRecordingBrokerClient(base localBrokerClient) *recordingBrokerClient {
	return &recordingBrokerClient{base: base}
}

func (r *recordingBrokerClient) record(call string) {
	r.calls = append(r.calls, call)
}

func (r *recordingBrokerClient) Calls() []string {
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

func (r *recordingBrokerClient) RunList(ctx context.Context, limit int) (brokerapi.RunListResponse, error) {
	r.record("RunList")
	return r.base.RunList(ctx, limit)
}

func (r *recordingBrokerClient) RunGet(ctx context.Context, runID string) (brokerapi.RunGetResponse, error) {
	r.record("RunGet")
	return r.base.RunGet(ctx, runID)
}

func (r *recordingBrokerClient) RunWatch(ctx context.Context, req brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, error) {
	r.record("RunWatch")
	return r.base.RunWatch(ctx, req)
}

func (r *recordingBrokerClient) SessionList(ctx context.Context, limit int) (brokerapi.SessionListResponse, error) {
	r.record("SessionList")
	return r.base.SessionList(ctx, limit)
}

func (r *recordingBrokerClient) SessionGet(ctx context.Context, sessionID string) (brokerapi.SessionGetResponse, error) {
	r.record("SessionGet")
	return r.base.SessionGet(ctx, sessionID)
}

func (r *recordingBrokerClient) SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, error) {
	r.record("SessionSendMessage")
	return r.base.SessionSendMessage(ctx, req)
}

func (r *recordingBrokerClient) SessionExecutionTrigger(ctx context.Context, req brokerapi.SessionExecutionTriggerRequest) (brokerapi.SessionExecutionTriggerResponse, error) {
	r.record("SessionExecutionTrigger")
	return r.base.SessionExecutionTrigger(ctx, req)
}

func (r *recordingBrokerClient) SessionWatch(ctx context.Context, req brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, error) {
	r.record("SessionWatch")
	return r.base.SessionWatch(ctx, req)
}

func (r *recordingBrokerClient) SessionTurnExecutionWatch(ctx context.Context, req brokerapi.SessionTurnExecutionWatchRequest) ([]brokerapi.SessionTurnExecutionWatchEvent, error) {
	r.record("SessionTurnExecutionWatch")
	return r.base.SessionTurnExecutionWatch(ctx, req)
}

func (r *recordingBrokerClient) ApprovalList(ctx context.Context, limit int) (brokerapi.ApprovalListResponse, error) {
	r.record("ApprovalList")
	return r.base.ApprovalList(ctx, limit)
}

func (r *recordingBrokerClient) ApprovalGet(ctx context.Context, approvalID string) (brokerapi.ApprovalGetResponse, error) {
	r.record("ApprovalGet")
	return r.base.ApprovalGet(ctx, approvalID)
}

func (r *recordingBrokerClient) ApprovalResolve(ctx context.Context, req brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, error) {
	r.record("ApprovalResolve")
	return r.base.ApprovalResolve(ctx, req)
}

func (r *recordingBrokerClient) ApprovalWatch(ctx context.Context, req brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, error) {
	r.record("ApprovalWatch")
	return r.base.ApprovalWatch(ctx, req)
}

func (r *recordingBrokerClient) BackendPostureGet(ctx context.Context) (brokerapi.BackendPostureGetResponse, error) {
	r.record("BackendPostureGet")
	return r.base.BackendPostureGet(ctx)
}

func (r *recordingBrokerClient) BackendPostureChange(ctx context.Context, req brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, error) {
	r.record("BackendPostureChange")
	return r.base.BackendPostureChange(ctx, req)
}

func (r *recordingBrokerClient) ArtifactList(ctx context.Context, limit int, dataClass string) (brokerapi.LocalArtifactListResponse, error) {
	r.record("ArtifactList")
	return r.base.ArtifactList(ctx, limit, dataClass)
}

func (r *recordingBrokerClient) ArtifactHead(ctx context.Context, digest string) (brokerapi.LocalArtifactHeadResponse, error) {
	r.record("ArtifactHead")
	return r.base.ArtifactHead(ctx, digest)
}

func (r *recordingBrokerClient) ArtifactRead(ctx context.Context, req brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, error) {
	r.record("ArtifactRead")
	return r.base.ArtifactRead(ctx, req)
}

func (r *recordingBrokerClient) DependencyCacheEnsure(ctx context.Context, req brokerapi.DependencyCacheEnsureRequest) (brokerapi.DependencyCacheEnsureResponse, error) {
	r.record("DependencyCacheEnsure")
	return r.base.DependencyCacheEnsure(ctx, req)
}

func (r *recordingBrokerClient) DependencyFetchRegistry(ctx context.Context, req brokerapi.DependencyFetchRegistryRequest) (brokerapi.DependencyFetchRegistryResponse, error) {
	r.record("DependencyFetchRegistry")
	return r.base.DependencyFetchRegistry(ctx, req)
}

func (r *recordingBrokerClient) DependencyCacheHandoff(ctx context.Context, req brokerapi.DependencyCacheHandoffRequest) (brokerapi.DependencyCacheHandoffResponse, error) {
	r.record("DependencyCacheHandoff")
	return r.base.DependencyCacheHandoff(ctx, req)
}

func (r *recordingBrokerClient) LLMInvoke(ctx context.Context, req brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, error) {
	r.record("LLMInvoke")
	return r.base.LLMInvoke(ctx, req)
}

func (r *recordingBrokerClient) LLMStream(ctx context.Context, req brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, error) {
	r.record("LLMStream")
	return r.base.LLMStream(ctx, req)
}

func (r *recordingBrokerClient) AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error) {
	r.record("AuditTimeline")
	return r.base.AuditTimeline(ctx, limit, cursor)
}

func (r *recordingBrokerClient) AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error) {
	r.record("AuditVerificationGet")
	return r.base.AuditVerificationGet(ctx, viewLimit)
}

func (r *recordingBrokerClient) AuditFinalizeVerify(ctx context.Context) (brokerapi.AuditFinalizeVerifyResponse, error) {
	r.record("AuditFinalizeVerify")
	return r.base.AuditFinalizeVerify(ctx)
}

func (r *recordingBrokerClient) AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error) {
	r.record("AuditRecordGet")
	return r.base.AuditRecordGet(ctx, digest)
}

func (r *recordingBrokerClient) AuditAnchorPreflightGet(ctx context.Context, req brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, error) {
	r.record("AuditAnchorPreflightGet")
	return r.base.AuditAnchorPreflightGet(ctx, req)
}

func (r *recordingBrokerClient) AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, error) {
	r.record("AuditAnchorPresenceGet")
	return r.base.AuditAnchorPresenceGet(ctx, req)
}

func (r *recordingBrokerClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error) {
	r.record("AuditAnchorSegment")
	return r.base.AuditAnchorSegment(ctx, req)
}

func (r *recordingBrokerClient) GitSetupGet(ctx context.Context, provider string) (brokerapi.GitSetupGetResponse, error) {
	r.record("GitSetupGet")
	return r.base.GitSetupGet(ctx, provider)
}

func (r *recordingBrokerClient) GitSetupAuthBootstrap(ctx context.Context, req brokerapi.GitSetupAuthBootstrapRequest) (brokerapi.GitSetupAuthBootstrapResponse, error) {
	r.record("GitSetupAuthBootstrap")
	return r.base.GitSetupAuthBootstrap(ctx, req)
}

func (r *recordingBrokerClient) GitSetupIdentityUpsert(ctx context.Context, req brokerapi.GitSetupIdentityUpsertRequest) (brokerapi.GitSetupIdentityUpsertResponse, error) {
	r.record("GitSetupIdentityUpsert")
	return r.base.GitSetupIdentityUpsert(ctx, req)
}

func (r *recordingBrokerClient) ProviderSetupSessionBegin(ctx context.Context, req brokerapi.ProviderSetupSessionBeginRequest) (brokerapi.ProviderSetupSessionBeginResponse, error) {
	r.record("ProviderSetupSessionBegin")
	return r.base.ProviderSetupSessionBegin(ctx, req)
}

func (r *recordingBrokerClient) ProviderSetupSecretIngressPrepare(ctx context.Context, req brokerapi.ProviderSetupSecretIngressPrepareRequest) (brokerapi.ProviderSetupSecretIngressPrepareResponse, error) {
	r.record("ProviderSetupSecretIngressPrepare")
	return r.base.ProviderSetupSecretIngressPrepare(ctx, req)
}

func (r *recordingBrokerClient) ProviderSetupSecretIngressSubmit(ctx context.Context, req brokerapi.ProviderSetupSecretIngressSubmitRequest, secret []byte) (brokerapi.ProviderSetupSecretIngressSubmitResponse, error) {
	r.record("ProviderSetupSecretIngressSubmit")
	return r.base.ProviderSetupSecretIngressSubmit(ctx, req, secret)
}

func (r *recordingBrokerClient) ProviderCredentialLeaseIssue(ctx context.Context, req brokerapi.ProviderCredentialLeaseIssueRequest) (brokerapi.ProviderCredentialLeaseIssueResponse, error) {
	r.record("ProviderCredentialLeaseIssue")
	return r.base.ProviderCredentialLeaseIssue(ctx, req)
}

func (r *recordingBrokerClient) ProviderProfileList(ctx context.Context) (brokerapi.ProviderProfileListResponse, error) {
	r.record("ProviderProfileList")
	return r.base.ProviderProfileList(ctx)
}

func (r *recordingBrokerClient) ProviderProfileGet(ctx context.Context, providerProfileID string) (brokerapi.ProviderProfileGetResponse, error) {
	r.record("ProviderProfileGet")
	return r.base.ProviderProfileGet(ctx, providerProfileID)
}

func (r *recordingBrokerClient) ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error) {
	r.record("ReadinessGet")
	return r.base.ReadinessGet(ctx)
}

func (r *recordingBrokerClient) VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error) {
	r.record("VersionInfoGet")
	return r.base.VersionInfoGet(ctx)
}

func (r *recordingBrokerClient) ProductLifecyclePostureGet(ctx context.Context) (brokerapi.ProductLifecyclePostureGetResponse, error) {
	r.record("ProductLifecyclePostureGet")
	return r.base.ProductLifecyclePostureGet(ctx)
}

func (r *recordingBrokerClient) ProjectSubstratePostureGet(ctx context.Context) (brokerapi.ProjectSubstratePostureGetResponse, error) {
	r.record("ProjectSubstratePostureGet")
	return r.base.ProjectSubstratePostureGet(ctx)
}

func (r *recordingBrokerClient) ProjectSubstrateAdopt(ctx context.Context) (brokerapi.ProjectSubstrateAdoptResponse, error) {
	r.record("ProjectSubstrateAdopt")
	return r.base.ProjectSubstrateAdopt(ctx)
}

func (r *recordingBrokerClient) ProjectSubstrateInitPreview(ctx context.Context) (brokerapi.ProjectSubstrateInitPreviewResponse, error) {
	r.record("ProjectSubstrateInitPreview")
	return r.base.ProjectSubstrateInitPreview(ctx)
}

func (r *recordingBrokerClient) ProjectSubstrateInitApply(ctx context.Context, expectedPreviewToken string) (brokerapi.ProjectSubstrateInitApplyResponse, error) {
	r.record("ProjectSubstrateInitApply")
	return r.base.ProjectSubstrateInitApply(ctx, expectedPreviewToken)
}

func (r *recordingBrokerClient) ProjectSubstrateUpgradePreview(ctx context.Context) (brokerapi.ProjectSubstrateUpgradePreviewResponse, error) {
	r.record("ProjectSubstrateUpgradePreview")
	return r.base.ProjectSubstrateUpgradePreview(ctx)
}

func (r *recordingBrokerClient) ProjectSubstrateUpgradeApply(ctx context.Context, expectedPreviewDigest string) (brokerapi.ProjectSubstrateUpgradeApplyResponse, error) {
	r.record("ProjectSubstrateUpgradeApply")
	return r.base.ProjectSubstrateUpgradeApply(ctx, expectedPreviewDigest)
}
