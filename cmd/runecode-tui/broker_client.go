package main

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const localAPISchemaVersion = "0.1.0"
const localAPIFamily = "broker_local_api"

var requestSeq uint64

var localIPCConfigProvider = brokerapi.DefaultLocalIPCConfig

var localRPCDialer = func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (localRPCInvoker, error) {
	return brokerapi.DialLocalRPC(ctx, cfg)
}

var localBrokerClientFactory = func() localBrokerClient {
	return &rpcBrokerClient{}
}

type localBrokerClient interface {
	RunList(ctx context.Context, limit int) (brokerapi.RunListResponse, error)
	RunGet(ctx context.Context, runID string) (brokerapi.RunGetResponse, error)
	RunWatch(ctx context.Context, req brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, error)
	SessionList(ctx context.Context, limit int) (brokerapi.SessionListResponse, error)
	SessionGet(ctx context.Context, sessionID string) (brokerapi.SessionGetResponse, error)
	SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, error)
	SessionExecutionTrigger(ctx context.Context, req brokerapi.SessionExecutionTriggerRequest) (brokerapi.SessionExecutionTriggerResponse, error)
	SessionWatch(ctx context.Context, req brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, error)
	SessionTurnExecutionWatch(ctx context.Context, req brokerapi.SessionTurnExecutionWatchRequest) ([]brokerapi.SessionTurnExecutionWatchEvent, error)
	ApprovalList(ctx context.Context, limit int) (brokerapi.ApprovalListResponse, error)
	ApprovalGet(ctx context.Context, approvalID string) (brokerapi.ApprovalGetResponse, error)
	ApprovalResolve(ctx context.Context, req brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, error)
	ApprovalWatch(ctx context.Context, req brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, error)
	BackendPostureGet(ctx context.Context) (brokerapi.BackendPostureGetResponse, error)
	BackendPostureChange(ctx context.Context, req brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, error)
	ArtifactList(ctx context.Context, limit int, dataClass string) (brokerapi.LocalArtifactListResponse, error)
	ArtifactHead(ctx context.Context, digest string) (brokerapi.LocalArtifactHeadResponse, error)
	ArtifactRead(ctx context.Context, req brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, error)
	DependencyCacheEnsure(ctx context.Context, req brokerapi.DependencyCacheEnsureRequest) (brokerapi.DependencyCacheEnsureResponse, error)
	DependencyFetchRegistry(ctx context.Context, req brokerapi.DependencyFetchRegistryRequest) (brokerapi.DependencyFetchRegistryResponse, error)
	DependencyCacheHandoff(ctx context.Context, req brokerapi.DependencyCacheHandoffRequest) (brokerapi.DependencyCacheHandoffResponse, error)
	LLMInvoke(ctx context.Context, req brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, error)
	LLMStream(ctx context.Context, req brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, error)
	AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error)
	AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error)
	AuditFinalizeVerify(ctx context.Context) (brokerapi.AuditFinalizeVerifyResponse, error)
	AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error)
	AuditAnchorPreflightGet(ctx context.Context, req brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, error)
	AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, error)
	AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error)
	GitSetupGet(ctx context.Context, provider string) (brokerapi.GitSetupGetResponse, error)
	GitSetupAuthBootstrap(ctx context.Context, req brokerapi.GitSetupAuthBootstrapRequest) (brokerapi.GitSetupAuthBootstrapResponse, error)
	GitSetupIdentityUpsert(ctx context.Context, req brokerapi.GitSetupIdentityUpsertRequest) (brokerapi.GitSetupIdentityUpsertResponse, error)
	ProviderSetupSessionBegin(ctx context.Context, req brokerapi.ProviderSetupSessionBeginRequest) (brokerapi.ProviderSetupSessionBeginResponse, error)
	ProviderSetupSecretIngressPrepare(ctx context.Context, req brokerapi.ProviderSetupSecretIngressPrepareRequest) (brokerapi.ProviderSetupSecretIngressPrepareResponse, error)
	ProviderSetupSecretIngressSubmit(ctx context.Context, req brokerapi.ProviderSetupSecretIngressSubmitRequest, secret []byte) (brokerapi.ProviderSetupSecretIngressSubmitResponse, error)
	ProviderCredentialLeaseIssue(ctx context.Context, req brokerapi.ProviderCredentialLeaseIssueRequest) (brokerapi.ProviderCredentialLeaseIssueResponse, error)
	ProviderProfileList(ctx context.Context) (brokerapi.ProviderProfileListResponse, error)
	ProviderProfileGet(ctx context.Context, providerProfileID string) (brokerapi.ProviderProfileGetResponse, error)
	GitRemoteMutationPrepare(ctx context.Context, req brokerapi.GitRemoteMutationPrepareRequest) (brokerapi.GitRemoteMutationPrepareResponse, error)
	GitRemoteMutationGet(ctx context.Context, req brokerapi.GitRemoteMutationGetRequest) (brokerapi.GitRemoteMutationGetResponse, error)
	GitRemoteMutationIssueExecuteLease(ctx context.Context, req brokerapi.GitRemoteMutationIssueExecuteLeaseRequest) (brokerapi.GitRemoteMutationIssueExecuteLeaseResponse, error)
	GitRemoteMutationExecute(ctx context.Context, req brokerapi.GitRemoteMutationExecuteRequest) (brokerapi.GitRemoteMutationExecuteResponse, error)
	ExternalAnchorMutationPrepare(ctx context.Context, req brokerapi.ExternalAnchorMutationPrepareRequest) (brokerapi.ExternalAnchorMutationPrepareResponse, error)
	ExternalAnchorMutationGet(ctx context.Context, req brokerapi.ExternalAnchorMutationGetRequest) (brokerapi.ExternalAnchorMutationGetResponse, error)
	ExternalAnchorMutationExecute(ctx context.Context, req brokerapi.ExternalAnchorMutationExecuteRequest) (brokerapi.ExternalAnchorMutationExecuteResponse, error)
	ProjectSubstratePostureGet(ctx context.Context) (brokerapi.ProjectSubstratePostureGetResponse, error)
	ProjectSubstrateAdopt(ctx context.Context) (brokerapi.ProjectSubstrateAdoptResponse, error)
	ProjectSubstrateInitPreview(ctx context.Context) (brokerapi.ProjectSubstrateInitPreviewResponse, error)
	ProjectSubstrateInitApply(ctx context.Context, expectedPreviewToken string) (brokerapi.ProjectSubstrateInitApplyResponse, error)
	ProjectSubstrateUpgradePreview(ctx context.Context) (brokerapi.ProjectSubstrateUpgradePreviewResponse, error)
	ProjectSubstrateUpgradeApply(ctx context.Context, expectedPreviewDigest string) (brokerapi.ProjectSubstrateUpgradeApplyResponse, error)
	ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error)
	VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error)
	ProductLifecyclePostureGet(ctx context.Context) (brokerapi.ProductLifecyclePostureGetResponse, error)
}

type localRPCInvoker interface {
	Invoke(ctx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse
	InvokeSecretIngress(ctx context.Context, operation string, request any, secret []byte, out any) *brokerapi.ErrorResponse
	Close() error
}

type rpcBrokerClient struct{}

func newLocalBrokerClient() localBrokerClient {
	return localBrokerClientFactory()
}

func (c *rpcBrokerClient) RunList(ctx context.Context, limit int) (brokerapi.RunListResponse, error) {
	req := brokerapi.RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("run-list"), Limit: limit}
	resp := brokerapi.RunListResponse{}
	return resp, c.invoke(ctx, "run_list", req, &resp)
}

func (c *rpcBrokerClient) RunGet(ctx context.Context, runID string) (brokerapi.RunGetResponse, error) {
	req := brokerapi.RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("run-get"), RunID: runID}
	resp := brokerapi.RunGetResponse{}
	return resp, c.invoke(ctx, "run_get", req, &resp)
}

func (c *rpcBrokerClient) RunWatch(ctx context.Context, req brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, error) {
	req.SchemaID = "runecode.protocol.v0.RunWatchRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("run-watch")
	events := []brokerapi.RunWatchEvent{}
	return events, c.invoke(ctx, "run_watch", req, &events)
}

func (c *rpcBrokerClient) SessionList(ctx context.Context, limit int) (brokerapi.SessionListResponse, error) {
	req := brokerapi.SessionListRequest{SchemaID: "runecode.protocol.v0.SessionListRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("session-list"), Limit: limit}
	resp := brokerapi.SessionListResponse{}
	return resp, c.invoke(ctx, "session_list", req, &resp)
}

func (c *rpcBrokerClient) SessionGet(ctx context.Context, sessionID string) (brokerapi.SessionGetResponse, error) {
	req := brokerapi.SessionGetRequest{SchemaID: "runecode.protocol.v0.SessionGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("session-get"), SessionID: sessionID}
	resp := brokerapi.SessionGetResponse{}
	return resp, c.invoke(ctx, "session_get", req, &resp)
}

func (c *rpcBrokerClient) SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, error) {
	req.SchemaID = "runecode.protocol.v0.SessionSendMessageRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("session-send")
	resp := brokerapi.SessionSendMessageResponse{}
	return resp, c.invoke(ctx, "session_send_message", req, &resp)
}

func (c *rpcBrokerClient) SessionExecutionTrigger(ctx context.Context, req brokerapi.SessionExecutionTriggerRequest) (brokerapi.SessionExecutionTriggerResponse, error) {
	req.SchemaID = "runecode.protocol.v0.SessionExecutionTriggerRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("session-trigger")
	resp := brokerapi.SessionExecutionTriggerResponse{}
	return resp, c.invoke(ctx, "session_execution_trigger", req, &resp)
}

func (c *rpcBrokerClient) SessionWatch(ctx context.Context, req brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, error) {
	req.SchemaID = "runecode.protocol.v0.SessionWatchRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("session-watch")
	events := []brokerapi.SessionWatchEvent{}
	return events, c.invoke(ctx, "session_watch", req, &events)
}

func (c *rpcBrokerClient) SessionTurnExecutionWatch(ctx context.Context, req brokerapi.SessionTurnExecutionWatchRequest) ([]brokerapi.SessionTurnExecutionWatchEvent, error) {
	req.SchemaID = "runecode.protocol.v0.SessionTurnExecutionWatchRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("session-turn-execution-watch")
	events := []brokerapi.SessionTurnExecutionWatchEvent{}
	return events, c.invoke(ctx, "session_turn_execution_watch", req, &events)
}

func (c *rpcBrokerClient) ApprovalList(ctx context.Context, limit int) (brokerapi.ApprovalListResponse, error) {
	req := brokerapi.ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("approval-list"), Limit: limit}
	resp := brokerapi.ApprovalListResponse{}
	return resp, c.invoke(ctx, "approval_list", req, &resp)
}

func (c *rpcBrokerClient) ApprovalGet(ctx context.Context, approvalID string) (brokerapi.ApprovalGetResponse, error) {
	req := brokerapi.ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("approval-get"), ApprovalID: approvalID}
	resp := brokerapi.ApprovalGetResponse{}
	return resp, c.invoke(ctx, "approval_get", req, &resp)
}

func (c *rpcBrokerClient) ApprovalResolve(ctx context.Context, req brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, error) {
	req.SchemaID = "runecode.protocol.v0.ApprovalResolveRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("approval-resolve")
	resp := brokerapi.ApprovalResolveResponse{}
	return resp, c.invoke(ctx, "approval_resolve", req, &resp)
}

func (c *rpcBrokerClient) ApprovalWatch(ctx context.Context, req brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, error) {
	req.SchemaID = "runecode.protocol.v0.ApprovalWatchRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("approval-watch")
	events := []brokerapi.ApprovalWatchEvent{}
	return events, c.invoke(ctx, "approval_watch", req, &events)
}

func (c *rpcBrokerClient) BackendPostureGet(ctx context.Context) (brokerapi.BackendPostureGetResponse, error) {
	req := brokerapi.BackendPostureGetRequest{SchemaID: "runecode.protocol.v0.BackendPostureGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("backend-posture-get")}
	resp := brokerapi.BackendPostureGetResponse{}
	return resp, c.invoke(ctx, "backend_posture_get", req, &resp)
}

func (c *rpcBrokerClient) BackendPostureChange(ctx context.Context, req brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, error) {
	req.SchemaID = "runecode.protocol.v0.BackendPostureChangeRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("backend-posture-change")
	resp := brokerapi.BackendPostureChangeResponse{}
	return resp, c.invoke(ctx, "backend_posture_change", req, &resp)
}

func (c *rpcBrokerClient) ArtifactList(ctx context.Context, limit int, dataClass string) (brokerapi.LocalArtifactListResponse, error) {
	req := brokerapi.LocalArtifactListRequest{SchemaID: "runecode.protocol.v0.ArtifactListRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("artifact-list"), Limit: limit, DataClass: dataClass}
	resp := brokerapi.LocalArtifactListResponse{}
	return resp, c.invoke(ctx, "artifact_list", req, &resp)
}

func (c *rpcBrokerClient) ArtifactHead(ctx context.Context, digest string) (brokerapi.LocalArtifactHeadResponse, error) {
	req := brokerapi.LocalArtifactHeadRequest{SchemaID: "runecode.protocol.v0.ArtifactHeadRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("artifact-head"), Digest: digest}
	resp := brokerapi.LocalArtifactHeadResponse{}
	return resp, c.invoke(ctx, "artifact_head", req, &resp)
}

func (c *rpcBrokerClient) ArtifactRead(ctx context.Context, req brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, error) {
	req.SchemaID = "runecode.protocol.v0.ArtifactReadRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("artifact-read")
	events := []brokerapi.ArtifactStreamEvent{}
	return events, c.invoke(ctx, "artifact_read", req, &events)
}

func (c *rpcBrokerClient) DependencyCacheEnsure(ctx context.Context, req brokerapi.DependencyCacheEnsureRequest) (brokerapi.DependencyCacheEnsureResponse, error) {
	req.SchemaID = "runecode.protocol.v0.DependencyCacheEnsureRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("dependency-cache-ensure")
	resp := brokerapi.DependencyCacheEnsureResponse{}
	return resp, c.invoke(ctx, "dependency_cache_ensure", req, &resp)
}

func (c *rpcBrokerClient) DependencyFetchRegistry(ctx context.Context, req brokerapi.DependencyFetchRegistryRequest) (brokerapi.DependencyFetchRegistryResponse, error) {
	req.SchemaID = "runecode.protocol.v0.DependencyFetchRegistryRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("dependency-fetch-registry")
	resp := brokerapi.DependencyFetchRegistryResponse{}
	return resp, c.invoke(ctx, "dependency_fetch_registry", req, &resp)
}

func (c *rpcBrokerClient) DependencyCacheHandoff(ctx context.Context, req brokerapi.DependencyCacheHandoffRequest) (brokerapi.DependencyCacheHandoffResponse, error) {
	req.SchemaID = "runecode.protocol.v0.DependencyCacheHandoffRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("dependency-cache-handoff")
	resp := brokerapi.DependencyCacheHandoffResponse{}
	return resp, c.invoke(ctx, "dependency_cache_handoff", req, &resp)
}

func (c *rpcBrokerClient) LLMInvoke(ctx context.Context, req brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, error) {
	req.SchemaID = "runecode.protocol.v0.LLMInvokeRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("llm-invoke")
	resp := brokerapi.LLMInvokeResponse{}
	return resp, c.invoke(ctx, "llm_invoke", req, &resp)
}

func (c *rpcBrokerClient) LLMStream(ctx context.Context, req brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, error) {
	req.SchemaID = "runecode.protocol.v0.LLMStreamRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("llm-stream")
	resp := brokerapi.LLMStreamEnvelope{}
	return resp, c.invoke(ctx, "llm_stream", req, &resp)
}

func (c *rpcBrokerClient) AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error) {
	req := brokerapi.AuditTimelineRequest{SchemaID: "runecode.protocol.v0.AuditTimelineRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("audit-timeline"), Limit: limit, Cursor: cursor}
	resp := brokerapi.AuditTimelineResponse{}
	return resp, c.invoke(ctx, "audit_timeline", req, &resp)
}

func (c *rpcBrokerClient) AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error) {
	req := brokerapi.AuditVerificationGetRequest{SchemaID: "runecode.protocol.v0.AuditVerificationGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("audit-verification"), ViewLimit: viewLimit}
	resp := brokerapi.AuditVerificationGetResponse{}
	return resp, c.invoke(ctx, "audit_verification_get", req, &resp)
}

func (c *rpcBrokerClient) AuditFinalizeVerify(ctx context.Context) (brokerapi.AuditFinalizeVerifyResponse, error) {
	req := brokerapi.AuditFinalizeVerifyRequest{SchemaID: "runecode.protocol.v0.AuditFinalizeVerifyRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("audit-finalize-verify")}
	resp := brokerapi.AuditFinalizeVerifyResponse{}
	return resp, c.invoke(ctx, "audit_finalize_verify", req, &resp)
}

func (c *rpcBrokerClient) AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error) {
	req := brokerapi.AuditRecordGetRequest{SchemaID: "runecode.protocol.v0.AuditRecordGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("audit-record"), RecordDigest: parseDigestIdentity(digest)}
	resp := brokerapi.AuditRecordGetResponse{}
	return resp, c.invoke(ctx, "audit_record_get", req, &resp)
}

func (c *rpcBrokerClient) AuditAnchorPreflightGet(ctx context.Context, req brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, error) {
	req.SchemaID = "runecode.protocol.v0.AuditAnchorPreflightGetRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("audit-anchor-preflight")
	resp := brokerapi.AuditAnchorPreflightGetResponse{}
	return resp, c.invoke(ctx, "audit_anchor_preflight_get", req, &resp)
}

func (c *rpcBrokerClient) AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, error) {
	req.SchemaID = "runecode.protocol.v0.AuditAnchorPresenceGetRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("audit-anchor-presence")
	resp := brokerapi.AuditAnchorPresenceGetResponse{}
	return resp, c.invoke(ctx, "audit_anchor_presence_get", req, &resp)
}

func (c *rpcBrokerClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error) {
	req.SchemaID = "runecode.protocol.v0.AuditAnchorSegmentRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("audit-anchor")
	resp := brokerapi.AuditAnchorSegmentResponse{}
	return resp, c.invoke(ctx, "audit_anchor_segment", req, &resp)
}

func (c *rpcBrokerClient) GitRemoteMutationPrepare(ctx context.Context, req brokerapi.GitRemoteMutationPrepareRequest) (brokerapi.GitRemoteMutationPrepareResponse, error) {
	req.SchemaID = "runecode.protocol.v0.GitRemoteMutationPrepareRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("git-remote-mutation-prepare")
	resp := brokerapi.GitRemoteMutationPrepareResponse{}
	return resp, c.invoke(ctx, "git_remote_mutation_prepare", req, &resp)
}

func (c *rpcBrokerClient) GitRemoteMutationGet(ctx context.Context, req brokerapi.GitRemoteMutationGetRequest) (brokerapi.GitRemoteMutationGetResponse, error) {
	req.SchemaID = "runecode.protocol.v0.GitRemoteMutationGetRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("git-remote-mutation-get")
	resp := brokerapi.GitRemoteMutationGetResponse{}
	return resp, c.invoke(ctx, "git_remote_mutation_get", req, &resp)
}

func (c *rpcBrokerClient) GitRemoteMutationIssueExecuteLease(ctx context.Context, req brokerapi.GitRemoteMutationIssueExecuteLeaseRequest) (brokerapi.GitRemoteMutationIssueExecuteLeaseResponse, error) {
	req.SchemaID = "runecode.protocol.v0.GitRemoteMutationIssueExecuteLeaseRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("git-remote-mutation-issue-execute-lease")
	resp := brokerapi.GitRemoteMutationIssueExecuteLeaseResponse{}
	return resp, c.invoke(ctx, "git_remote_mutation_issue_execute_lease", req, &resp)
}

func (c *rpcBrokerClient) GitRemoteMutationExecute(ctx context.Context, req brokerapi.GitRemoteMutationExecuteRequest) (brokerapi.GitRemoteMutationExecuteResponse, error) {
	req.SchemaID = "runecode.protocol.v0.GitRemoteMutationExecuteRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("git-remote-mutation-execute")
	resp := brokerapi.GitRemoteMutationExecuteResponse{}
	return resp, c.invoke(ctx, "git_remote_mutation_execute", req, &resp)
}

func (c *rpcBrokerClient) ExternalAnchorMutationPrepare(ctx context.Context, req brokerapi.ExternalAnchorMutationPrepareRequest) (brokerapi.ExternalAnchorMutationPrepareResponse, error) {
	req.SchemaID = "runecode.protocol.v0.ExternalAnchorMutationPrepareRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("external-anchor-mutation-prepare")
	resp := brokerapi.ExternalAnchorMutationPrepareResponse{}
	return resp, c.invoke(ctx, "external_anchor_mutation_prepare", req, &resp)
}

func (c *rpcBrokerClient) ExternalAnchorMutationGet(ctx context.Context, req brokerapi.ExternalAnchorMutationGetRequest) (brokerapi.ExternalAnchorMutationGetResponse, error) {
	req.SchemaID = "runecode.protocol.v0.ExternalAnchorMutationGetRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("external-anchor-mutation-get")
	resp := brokerapi.ExternalAnchorMutationGetResponse{}
	return resp, c.invoke(ctx, "external_anchor_mutation_get", req, &resp)
}

func (c *rpcBrokerClient) ExternalAnchorMutationExecute(ctx context.Context, req brokerapi.ExternalAnchorMutationExecuteRequest) (brokerapi.ExternalAnchorMutationExecuteResponse, error) {
	req.SchemaID = "runecode.protocol.v0.ExternalAnchorMutationExecuteRequest"
	req.SchemaVersion = localAPISchemaVersion
	req.RequestID = newRequestID("external-anchor-mutation-execute")
	resp := brokerapi.ExternalAnchorMutationExecuteResponse{}
	return resp, c.invoke(ctx, "external_anchor_mutation_execute", req, &resp)
}

func (c *rpcBrokerClient) ProjectSubstratePostureGet(ctx context.Context) (brokerapi.ProjectSubstratePostureGetResponse, error) {
	req := brokerapi.ProjectSubstratePostureGetRequest{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("project-substrate-posture")}
	resp := brokerapi.ProjectSubstratePostureGetResponse{}
	return resp, c.invoke(ctx, "project_substrate_posture_get", req, &resp)
}

func (c *rpcBrokerClient) ProjectSubstrateAdopt(ctx context.Context) (brokerapi.ProjectSubstrateAdoptResponse, error) {
	req := brokerapi.ProjectSubstrateAdoptRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateAdoptRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("project-substrate-adopt")}
	resp := brokerapi.ProjectSubstrateAdoptResponse{}
	return resp, c.invoke(ctx, "project_substrate_adopt", req, &resp)
}

func (c *rpcBrokerClient) ProjectSubstrateInitPreview(ctx context.Context) (brokerapi.ProjectSubstrateInitPreviewResponse, error) {
	req := brokerapi.ProjectSubstrateInitPreviewRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateInitPreviewRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("project-substrate-init-preview")}
	resp := brokerapi.ProjectSubstrateInitPreviewResponse{}
	return resp, c.invoke(ctx, "project_substrate_init_preview", req, &resp)
}

func (c *rpcBrokerClient) ProjectSubstrateInitApply(ctx context.Context, expectedPreviewToken string) (brokerapi.ProjectSubstrateInitApplyResponse, error) {
	req := brokerapi.ProjectSubstrateInitApplyRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateInitApplyRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("project-substrate-init-apply"), ExpectedPreviewToken: strings.TrimSpace(expectedPreviewToken)}
	resp := brokerapi.ProjectSubstrateInitApplyResponse{}
	return resp, c.invoke(ctx, "project_substrate_init_apply", req, &resp)
}

func (c *rpcBrokerClient) ProjectSubstrateUpgradePreview(ctx context.Context) (brokerapi.ProjectSubstrateUpgradePreviewResponse, error) {
	req := brokerapi.ProjectSubstrateUpgradePreviewRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradePreviewRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("project-substrate-upgrade-preview")}
	resp := brokerapi.ProjectSubstrateUpgradePreviewResponse{}
	return resp, c.invoke(ctx, "project_substrate_upgrade_preview", req, &resp)
}

func (c *rpcBrokerClient) ProjectSubstrateUpgradeApply(ctx context.Context, expectedPreviewDigest string) (brokerapi.ProjectSubstrateUpgradeApplyResponse, error) {
	req := brokerapi.ProjectSubstrateUpgradeApplyRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradeApplyRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("project-substrate-upgrade-apply"), ExpectedPreviewDigest: strings.TrimSpace(expectedPreviewDigest)}
	resp := brokerapi.ProjectSubstrateUpgradeApplyResponse{}
	return resp, c.invoke(ctx, "project_substrate_upgrade_apply", req, &resp)
}

func (c *rpcBrokerClient) ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error) {
	req := brokerapi.ReadinessGetRequest{SchemaID: "runecode.protocol.v0.ReadinessGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("readiness")}
	resp := brokerapi.ReadinessGetResponse{}
	return resp, c.invoke(ctx, "readiness_get", req, &resp)
}

func (c *rpcBrokerClient) VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error) {
	req := brokerapi.VersionInfoGetRequest{SchemaID: "runecode.protocol.v0.VersionInfoGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("version")}
	resp := brokerapi.VersionInfoGetResponse{}
	return resp, c.invoke(ctx, "version_info_get", req, &resp)
}

func (c *rpcBrokerClient) ProductLifecyclePostureGet(ctx context.Context) (brokerapi.ProductLifecyclePostureGetResponse, error) {
	req := brokerapi.ProductLifecyclePostureGetRequest{SchemaID: "runecode.protocol.v0.ProductLifecyclePostureGetRequest", SchemaVersion: localAPISchemaVersion, RequestID: newRequestID("product-lifecycle-posture")}
	resp := brokerapi.ProductLifecyclePostureGetResponse{}
	return resp, c.invoke(ctx, "product_lifecycle_posture_get", req, &resp)
}
