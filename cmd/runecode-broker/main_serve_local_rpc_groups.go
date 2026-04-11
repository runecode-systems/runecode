package main

import (
	"context"
	"encoding/json"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func runRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"run_list": {requestSchemaPath: "objects/RunListRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.RunListRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleRunList(ctx, req, meta)
			})
		}},
		"run_get": {requestSchemaPath: "objects/RunGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.RunGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleRunGet(ctx, req, meta)
			})
		}},
		"run_watch": {requestSchemaPath: "objects/RunWatchRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandleRunWatch(service, ctx, raw, meta)
		}},
	}
}

func sessionRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"session_list": {requestSchemaPath: "objects/SessionListRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.SessionListRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleSessionList(ctx, req, meta)
			})
		}},
		"session_get": {requestSchemaPath: "objects/SessionGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.SessionGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleSessionGet(ctx, req, meta)
			})
		}},
		"session_send_message": {requestSchemaPath: "objects/SessionSendMessageRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.SessionSendMessageRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleSessionSendMessage(ctx, req, meta)
			})
		}},
		"session_watch": {requestSchemaPath: "objects/SessionWatchRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandleSessionWatch(service, ctx, raw, meta)
		}},
	}
}

func approvalRunnerRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"runner_checkpoint_report": {requestSchemaPath: "objects/RunnerCheckpointReportRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.RunnerCheckpointReportRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleRunnerCheckpointReport(ctx, req, meta)
			})
		}},
		"runner_result_report": {requestSchemaPath: "objects/RunnerResultReportRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.RunnerResultReportRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleRunnerResultReport(ctx, req, meta)
			})
		}},
		"approval_list": {requestSchemaPath: "objects/ApprovalListRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ApprovalListRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleApprovalList(ctx, req, meta)
			})
		}},
		"approval_get": {requestSchemaPath: "objects/ApprovalGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ApprovalGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleApprovalGet(ctx, req, meta)
			})
		}},
		"approval_resolve": {requestSchemaPath: "objects/ApprovalResolveRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ApprovalResolveRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleApprovalResolve(ctx, req, meta)
			})
		}},
		"approval_watch": {requestSchemaPath: "objects/ApprovalWatchRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandleApprovalWatch(service, ctx, raw, meta)
		}},
	}
}
