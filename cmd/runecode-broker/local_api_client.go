package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type brokerLocalAPI interface {
	RunList(context.Context, brokerapi.RunListRequest) (brokerapi.RunListResponse, *brokerapi.ErrorResponse)
	RunGet(context.Context, brokerapi.RunGetRequest) (brokerapi.RunGetResponse, *brokerapi.ErrorResponse)
	RunWatch(context.Context, brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, *brokerapi.ErrorResponse)
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
	ReadinessGet(context.Context, brokerapi.ReadinessGetRequest) (brokerapi.ReadinessGetResponse, *brokerapi.ErrorResponse)
	VersionInfoGet(context.Context, brokerapi.VersionInfoGetRequest) (brokerapi.VersionInfoGetResponse, *brokerapi.ErrorResponse)
	AuditVerificationGet(context.Context, brokerapi.AuditVerificationGetRequest) (brokerapi.AuditVerificationGetResponse, *brokerapi.ErrorResponse)
	AuditRecordGet(context.Context, brokerapi.AuditRecordGetRequest) (brokerapi.AuditRecordGetResponse, *brokerapi.ErrorResponse)
	AuditAnchorSegment(context.Context, brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, *brokerapi.ErrorResponse)
}

type localRPCInvokeFunc func(ctx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse

type localAPIClient struct{ invoke localRPCInvokeFunc }

var localAPIClientFactory = newInProcessLocalAPIClient

func localAPIForService(service *brokerapi.Service) brokerLocalAPI {
	return localAPIClientFactory(service)
}

func newInProcessLocalAPIClient(service *brokerapi.Service) brokerLocalAPI {
	meta := brokerapi.RequestContext{ClientID: "cli", LaneID: "cli_local_rpc"}
	return &localAPIClient{invoke: func(ctx context.Context, operation string, request any, out any) *brokerapi.ErrorResponse {
		requestCtx := ctx
		if requestCtx == nil {
			requestCtx = context.Background()
		}
		requestBytes, err := json.Marshal(request)
		if err != nil {
			errResp := brokerapi.ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: "cli-invalid-request", Error: brokerapi.ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "broker_validation_schema_invalid", Category: "validation", Retryable: false, Message: err.Error()}}
			return &errResp
		}
		wire := localRPCRequest{Operation: operation, Request: json.RawMessage(requestBytes)}
		resp := localRPCDispatch(service, requestCtx, wire, meta)
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
	}}
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

func (c *localAPIClient) AuditRecordGet(ctx context.Context, req brokerapi.AuditRecordGetRequest) (brokerapi.AuditRecordGetResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.AuditRecordGetResponse{}
	return resp, c.invoke(ctx, "audit_record_get", req, &resp)
}

func (c *localAPIClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, *brokerapi.ErrorResponse) {
	resp := brokerapi.AuditAnchorSegmentResponse{}
	return resp, c.invoke(ctx, "audit_anchor_segment", req, &resp)
}
