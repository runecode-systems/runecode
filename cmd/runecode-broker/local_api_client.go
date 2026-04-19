package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type brokerLocalAPI interface {
	RunList(context.Context, brokerapi.RunListRequest) (brokerapi.RunListResponse, *brokerapi.ErrorResponse)
	RunGet(context.Context, brokerapi.RunGetRequest) (brokerapi.RunGetResponse, *brokerapi.ErrorResponse)
	RunWatch(context.Context, brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, *brokerapi.ErrorResponse)
	BackendPostureGet(context.Context, brokerapi.BackendPostureGetRequest) (brokerapi.BackendPostureGetResponse, *brokerapi.ErrorResponse)
	BackendPostureChange(context.Context, brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, *brokerapi.ErrorResponse)
	SessionList(context.Context, brokerapi.SessionListRequest) (brokerapi.SessionListResponse, *brokerapi.ErrorResponse)
	SessionGet(context.Context, brokerapi.SessionGetRequest) (brokerapi.SessionGetResponse, *brokerapi.ErrorResponse)
	SessionSendMessage(context.Context, brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, *brokerapi.ErrorResponse)
	SessionWatch(context.Context, brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, *brokerapi.ErrorResponse)
	RunnerCheckpointReport(context.Context, brokerapi.RunnerCheckpointReportRequest) (brokerapi.RunnerCheckpointReportResponse, *brokerapi.ErrorResponse)
	RunnerResultReport(context.Context, brokerapi.RunnerResultReportRequest) (brokerapi.RunnerResultReportResponse, *brokerapi.ErrorResponse)
	ApprovalList(context.Context, brokerapi.ApprovalListRequest) (brokerapi.ApprovalListResponse, *brokerapi.ErrorResponse)
	ApprovalGet(context.Context, brokerapi.ApprovalGetRequest) (brokerapi.ApprovalGetResponse, *brokerapi.ErrorResponse)
	ApprovalResolve(context.Context, brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, *brokerapi.ErrorResponse)
	ApprovalWatch(context.Context, brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, *brokerapi.ErrorResponse)
	ArtifactList(context.Context, brokerapi.LocalArtifactListRequest) (brokerapi.LocalArtifactListResponse, *brokerapi.ErrorResponse)
	ArtifactHead(context.Context, brokerapi.LocalArtifactHeadRequest) (brokerapi.LocalArtifactHeadResponse, *brokerapi.ErrorResponse)
	ArtifactRead(context.Context, brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, *brokerapi.ErrorResponse)
	LogStream(context.Context, brokerapi.LogStreamRequest) ([]brokerapi.LogStreamEvent, *brokerapi.ErrorResponse)
	LLMInvoke(context.Context, brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, *brokerapi.ErrorResponse)
	LLMStream(context.Context, brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, *brokerapi.ErrorResponse)
	ReadinessGet(context.Context, brokerapi.ReadinessGetRequest) (brokerapi.ReadinessGetResponse, *brokerapi.ErrorResponse)
	VersionInfoGet(context.Context, brokerapi.VersionInfoGetRequest) (brokerapi.VersionInfoGetResponse, *brokerapi.ErrorResponse)
	AuditVerificationGet(context.Context, brokerapi.AuditVerificationGetRequest) (brokerapi.AuditVerificationGetResponse, *brokerapi.ErrorResponse)
	AuditFinalizeVerify(context.Context, brokerapi.AuditFinalizeVerifyRequest) (brokerapi.AuditFinalizeVerifyResponse, *brokerapi.ErrorResponse)
	AuditRecordGet(context.Context, brokerapi.AuditRecordGetRequest) (brokerapi.AuditRecordGetResponse, *brokerapi.ErrorResponse)
	AuditAnchorPreflightGet(context.Context, brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, *brokerapi.ErrorResponse)
	AuditAnchorPresenceGet(context.Context, brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, *brokerapi.ErrorResponse)
	AuditAnchorSegment(context.Context, brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, *brokerapi.ErrorResponse)
	GitSetupGet(context.Context, brokerapi.GitSetupGetRequest) (brokerapi.GitSetupGetResponse, *brokerapi.ErrorResponse)
	GitSetupAuthBootstrap(context.Context, brokerapi.GitSetupAuthBootstrapRequest) (brokerapi.GitSetupAuthBootstrapResponse, *brokerapi.ErrorResponse)
	GitSetupIdentityUpsert(context.Context, brokerapi.GitSetupIdentityUpsertRequest) (brokerapi.GitSetupIdentityUpsertResponse, *brokerapi.ErrorResponse)
	ProviderSetupSessionBegin(context.Context, brokerapi.ProviderSetupSessionBeginRequest) (brokerapi.ProviderSetupSessionBeginResponse, *brokerapi.ErrorResponse)
	ProviderSetupSecretIngressPrepare(context.Context, brokerapi.ProviderSetupSecretIngressPrepareRequest) (brokerapi.ProviderSetupSecretIngressPrepareResponse, *brokerapi.ErrorResponse)
	ProviderSetupSecretIngressSubmit(context.Context, brokerapi.ProviderSetupSecretIngressSubmitRequest, []byte) (brokerapi.ProviderSetupSecretIngressSubmitResponse, *brokerapi.ErrorResponse)
	ProviderCredentialLeaseIssue(context.Context, brokerapi.ProviderCredentialLeaseIssueRequest) (brokerapi.ProviderCredentialLeaseIssueResponse, *brokerapi.ErrorResponse)
	ProviderProfileList(context.Context, brokerapi.ProviderProfileListRequest) (brokerapi.ProviderProfileListResponse, *brokerapi.ErrorResponse)
	ProviderProfileGet(context.Context, brokerapi.ProviderProfileGetRequest) (brokerapi.ProviderProfileGetResponse, *brokerapi.ErrorResponse)
	GitRemoteMutationPrepare(context.Context, brokerapi.GitRemoteMutationPrepareRequest) (brokerapi.GitRemoteMutationPrepareResponse, *brokerapi.ErrorResponse)
	GitRemoteMutationGet(context.Context, brokerapi.GitRemoteMutationGetRequest) (brokerapi.GitRemoteMutationGetResponse, *brokerapi.ErrorResponse)
	GitRemoteMutationIssueExecuteLease(context.Context, brokerapi.GitRemoteMutationIssueExecuteLeaseRequest) (brokerapi.GitRemoteMutationIssueExecuteLeaseResponse, *brokerapi.ErrorResponse)
	GitRemoteMutationExecute(context.Context, brokerapi.GitRemoteMutationExecuteRequest) (brokerapi.GitRemoteMutationExecuteResponse, *brokerapi.ErrorResponse)
}

type localRPCInvokeFunc func(ctx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse
type localRPCInvokeSecretFunc func(ctx context.Context, operation string, request any, secret []byte, out any) *brokerapi.ErrorResponse

type localAPIClient struct {
	invoke       localRPCInvokeFunc
	invokeSecret localRPCInvokeSecretFunc
}

var localAPIClientFactory = newInProcessLocalAPIClient

func localAPIForService(service *brokerapi.Service) brokerLocalAPI {
	return localAPIClientFactory(service)
}

func newInProcessLocalAPIClient(service *brokerapi.Service) brokerLocalAPI {
	meta := brokerapi.RequestContext{ClientID: "cli", LaneID: "cli_local_rpc"}
	return &localAPIClient{invoke: newLocalRPCInvoke(service, meta), invokeSecret: newLocalRPCInvokeSecret(service, meta)}
}

func newLocalRPCInvoke(service *brokerapi.Service, meta brokerapi.RequestContext) localRPCInvokeFunc {
	return func(ctx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse {
		wire, errResp := newLocalRPCWire(operation, request, nil)
		if errResp != nil {
			return errResp
		}
		return dispatchLocalRPCRequest(service, meta, ctx, wire, out)
	}
}

func newLocalRPCInvokeSecret(service *brokerapi.Service, meta brokerapi.RequestContext) localRPCInvokeSecretFunc {
	return func(ctx context.Context, operation string, request any, secret []byte, out any) *brokerapi.ErrorResponse {
		wire, errResp := newLocalRPCWire(operation, request, secret)
		if errResp != nil {
			return errResp
		}
		return dispatchLocalRPCRequest(service, meta, ctx, wire, out)
	}
}

func newLocalRPCWire(operation string, request any, secret []byte) (localRPCRequest, *brokerapi.ErrorResponse) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		errResp := brokerapi.ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: "cli-invalid-request", Error: brokerapi.ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "broker_validation_schema_invalid", Category: "validation", Retryable: false, Message: err.Error()}}
		return localRPCRequest{}, &errResp
	}
	wire := localRPCRequest{Operation: operation, Request: json.RawMessage(requestBytes)}
	if len(secret) > 0 {
		wire.SecretIngressPayloadBase64 = base64.StdEncoding.EncodeToString(secret)
	}
	return wire, nil
}

func dispatchLocalRPCRequest(service *brokerapi.Service, meta brokerapi.RequestContext, ctx context.Context, wire localRPCRequest, out any) *brokerapi.ErrorResponse {
	resp := localRPCDispatch(service, normalizedLocalRPCContext(ctx), wire, meta)
	if resp.Error != nil {
		return resp.Error
	}
	if !resp.OK {
		errResp := brokerapi.ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: "cli-local-rpc-failed", Error: brokerapi.ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "gateway_failure", Category: "internal", Retryable: false, Message: "local rpc invocation failed without typed error"}}
		return &errResp
	}
	if out == nil || len(resp.Response) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Response, out); err != nil {
		errResp := brokerapi.ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: "cli-local-rpc-unmarshal", Error: brokerapi.ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "gateway_failure", Category: "internal", Retryable: false, Message: err.Error()}}
		return &errResp
	}
	return nil
}

func normalizedLocalRPCContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func localAPIError(errResp *brokerapi.ErrorResponse) error {
	if errResp == nil {
		return nil
	}
	return fmt.Errorf("%s: %s", errResp.Error.Code, errResp.Error.Message)
}

func (c *localAPIClient) RunList(ctx context.Context, req brokerapi.RunListRequest) (brokerapi.RunListResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.RunListResponse{}
	return resp, c.invoke(ctx, "run_list", req, &resp)
}

func (c *localAPIClient) RunGet(ctx context.Context, req brokerapi.RunGetRequest) (brokerapi.RunGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.RunGetResponse{}
	return resp, c.invoke(ctx, "run_get", req, &resp)
}

func (c *localAPIClient) RunWatch(ctx context.Context, req brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, *brokerapi.ErrorResponse) {
	events := []brokerapi.RunWatchEvent{}
	return events, c.invoke(ctx, "run_watch", req, &events)
}

func (c *localAPIClient) BackendPostureGet(ctx context.Context, req brokerapi.BackendPostureGetRequest) (brokerapi.BackendPostureGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.BackendPostureGetResponse{}
	return resp, c.invoke(ctx, "backend_posture_get", req, &resp)
}

func (c *localAPIClient) BackendPostureChange(ctx context.Context, req brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.BackendPostureChangeResponse{}
	return resp, c.invoke(ctx, "backend_posture_change", req, &resp)
}

func (c *localAPIClient) SessionList(ctx context.Context, req brokerapi.SessionListRequest) (brokerapi.SessionListResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.SessionListResponse{}
	return resp, c.invoke(ctx, "session_list", req, &resp)
}

func (c *localAPIClient) SessionGet(ctx context.Context, req brokerapi.SessionGetRequest) (brokerapi.SessionGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.SessionGetResponse{}
	return resp, c.invoke(ctx, "session_get", req, &resp)
}

func (c *localAPIClient) SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.SessionSendMessageResponse{}
	return resp, c.invoke(ctx, "session_send_message", req, &resp)
}

func (c *localAPIClient) SessionWatch(ctx context.Context, req brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, *brokerapi.ErrorResponse) {
	events := []brokerapi.SessionWatchEvent{}
	return events, c.invoke(ctx, "session_watch", req, &events)
}

func (c *localAPIClient) RunnerCheckpointReport(ctx context.Context, req brokerapi.RunnerCheckpointReportRequest) (brokerapi.RunnerCheckpointReportResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.RunnerCheckpointReportResponse{}
	return resp, c.invoke(ctx, "runner_checkpoint_report", req, &resp)
}

func (c *localAPIClient) RunnerResultReport(ctx context.Context, req brokerapi.RunnerResultReportRequest) (brokerapi.RunnerResultReportResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.RunnerResultReportResponse{}
	return resp, c.invoke(ctx, "runner_result_report", req, &resp)
}

func (c *localAPIClient) ApprovalList(ctx context.Context, req brokerapi.ApprovalListRequest) (brokerapi.ApprovalListResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ApprovalListResponse{}
	return resp, c.invoke(ctx, "approval_list", req, &resp)
}

func (c *localAPIClient) ApprovalGet(ctx context.Context, req brokerapi.ApprovalGetRequest) (brokerapi.ApprovalGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ApprovalGetResponse{}
	return resp, c.invoke(ctx, "approval_get", req, &resp)
}

func (c *localAPIClient) ApprovalResolve(ctx context.Context, req brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ApprovalResolveResponse{}
	return resp, c.invoke(ctx, "approval_resolve", req, &resp)
}

func (c *localAPIClient) ApprovalWatch(ctx context.Context, req brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, *brokerapi.ErrorResponse) {
	events := []brokerapi.ApprovalWatchEvent{}
	return events, c.invoke(ctx, "approval_watch", req, &events)
}

func (c *localAPIClient) ArtifactList(ctx context.Context, req brokerapi.LocalArtifactListRequest) (brokerapi.LocalArtifactListResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.LocalArtifactListResponse{}
	return resp, c.invoke(ctx, "artifact_list", req, &resp)
}

func (c *localAPIClient) ArtifactHead(ctx context.Context, req brokerapi.LocalArtifactHeadRequest) (brokerapi.LocalArtifactHeadResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.LocalArtifactHeadResponse{}
	return resp, c.invoke(ctx, "artifact_head", req, &resp)
}

func (c *localAPIClient) ArtifactRead(ctx context.Context, req brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, *brokerapi.ErrorResponse) {
	events := []brokerapi.ArtifactStreamEvent{}
	return events, c.invoke(ctx, "artifact_read", req, &events)
}

func (c *localAPIClient) LogStream(ctx context.Context, req brokerapi.LogStreamRequest) ([]brokerapi.LogStreamEvent, *brokerapi.ErrorResponse) {
	events := []brokerapi.LogStreamEvent{}
	return events, c.invoke(ctx, "log_stream", req, &events)
}

func (c *localAPIClient) LLMInvoke(ctx context.Context, req brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.LLMInvokeResponse{}
	return resp, c.invoke(ctx, "llm_invoke", req, &resp)
}

func (c *localAPIClient) LLMStream(ctx context.Context, req brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, *brokerapi.ErrorResponse) {
	resp := brokerapi.LLMStreamEnvelope{}
	return resp, c.invoke(ctx, "llm_stream", req, &resp)
}

func (c *localAPIClient) ReadinessGet(ctx context.Context, req brokerapi.ReadinessGetRequest) (brokerapi.ReadinessGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.ReadinessGetResponse{}
	return resp, c.invoke(ctx, "readiness_get", req, &resp)
}

func (c *localAPIClient) VersionInfoGet(ctx context.Context, req brokerapi.VersionInfoGetRequest) (brokerapi.VersionInfoGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.VersionInfoGetResponse{}
	return resp, c.invoke(ctx, "version_info_get", req, &resp)
}

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
