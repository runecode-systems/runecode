package main

import (
	"context"
	"encoding/json"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func localRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	operations := map[string]rpcOperation{}
	mergeRPCOperations(operations, runApprovalRPCOperations(service, ctx, meta))
	mergeRPCOperations(operations, gitSetupRPCOperations(service, ctx, meta))
	mergeRPCOperations(operations, providerSetupRPCOperations(service, ctx, meta))
	mergeRPCOperations(operations, artifactRPCOperations(service, ctx, meta))
	mergeRPCOperations(operations, auditHealthRPCOperations(service, ctx, meta))
	return operations
}

func providerSetupRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"provider_setup_session_begin": {requestSchemaPath: "objects/ProviderSetupSessionBeginRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProviderSetupSessionBeginRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProviderSetupSessionBegin(ctx, req, meta)
			})
		}},
		"provider_setup_secret_ingress_prepare": {requestSchemaPath: "objects/ProviderSetupSecretIngressPrepareRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProviderSetupSecretIngressPrepareRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProviderSetupSecretIngressPrepare(ctx, req, meta)
			})
		}},
		"provider_profile_list": {requestSchemaPath: "objects/ProviderProfileListRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProviderProfileListRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProviderProfileList(ctx, req, meta)
			})
		}},
		"provider_profile_get": {requestSchemaPath: "objects/ProviderProfileGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProviderProfileGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProviderProfileGet(ctx, req, meta)
			})
		}},
	}
}

func mergeRPCOperations(dst, src map[string]rpcOperation) {
	for key, op := range src {
		dst[key] = op
	}
}

func runApprovalRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	operations := map[string]rpcOperation{}
	mergeRPCOperations(operations, runRPCOperations(service, ctx, meta))
	mergeRPCOperations(operations, sessionRPCOperations(service, ctx, meta))
	mergeRPCOperations(operations, approvalRunnerRPCOperations(service, ctx, meta))
	return operations
}

func gitSetupRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"git_setup_get": {requestSchemaPath: "objects/GitSetupGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.GitSetupGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleGitSetupGet(ctx, req, meta)
			})
		}},
		"git_setup_auth_bootstrap": {requestSchemaPath: "objects/GitSetupAuthBootstrapRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.GitSetupAuthBootstrapRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleGitSetupAuthBootstrap(ctx, req, meta)
			})
		}},
		"git_setup_identity_upsert": {requestSchemaPath: "objects/GitSetupIdentityUpsertRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.GitSetupIdentityUpsertRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleGitSetupIdentityUpsert(ctx, req, meta)
			})
		}},
		"git_remote_mutation_prepare": {requestSchemaPath: "objects/GitRemoteMutationPrepareRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.GitRemoteMutationPrepareRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleGitRemoteMutationPrepare(ctx, req, meta)
			})
		}},
		"git_remote_mutation_get": {requestSchemaPath: "objects/GitRemoteMutationGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.GitRemoteMutationGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleGitRemoteMutationGet(ctx, req, meta)
			})
		}},
		"git_remote_mutation_issue_execute_lease": {requestSchemaPath: "objects/GitRemoteMutationIssueExecuteLeaseRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.GitRemoteMutationIssueExecuteLeaseRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleGitRemoteMutationIssueExecuteLease(ctx, req, meta)
			})
		}},
		"git_remote_mutation_execute": {requestSchemaPath: "objects/GitRemoteMutationExecuteRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.GitRemoteMutationExecuteRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleGitRemoteMutationExecute(ctx, req, meta)
			})
		}},
	}
}

func artifactRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"artifact_list": {requestSchemaPath: "objects/ArtifactListRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.LocalArtifactListRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleArtifactListV0(ctx, req, meta)
			})
		}},
		"artifact_head": {requestSchemaPath: "objects/ArtifactHeadRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.LocalArtifactHeadRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleArtifactHeadV0(ctx, req, meta)
			})
		}},
		"artifact_read": {requestSchemaPath: "objects/ArtifactReadRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandleArtifactRead(service, ctx, raw, meta)
		}},
		"log_stream": {requestSchemaPath: "objects/LogStreamRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse { return decodeAndHandleLogStream(service, ctx, raw, meta) }},
		"llm_invoke": {requestSchemaPath: "objects/LLMInvokeRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse { return decodeAndHandleLLMInvoke(service, ctx, raw, meta) }},
		"llm_stream": {requestSchemaPath: "objects/LLMStreamRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse { return decodeAndHandleLLMStream(service, ctx, raw, meta) }},
	}
}

func auditHealthRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"audit_timeline": {requestSchemaPath: "objects/AuditTimelineRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.AuditTimelineRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleAuditTimeline(ctx, req, meta)
			})
		}},
		"audit_verification_get": {requestSchemaPath: "objects/AuditVerificationGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.AuditVerificationGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleAuditVerificationGet(ctx, req, meta)
			})
		}},
		"audit_finalize_verify": {requestSchemaPath: "objects/AuditFinalizeVerifyRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.AuditFinalizeVerifyRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleAuditFinalizeVerify(ctx, req, meta)
			})
		}},
		"audit_record_get": {requestSchemaPath: "objects/AuditRecordGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.AuditRecordGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleAuditRecordGet(ctx, req, meta)
			})
		}},
		"audit_anchor_preflight_get": {requestSchemaPath: "objects/AuditAnchorPreflightGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.AuditAnchorPreflightGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleAuditAnchorPreflightGet(ctx, req, meta)
			})
		}},
		"audit_anchor_presence_get": {requestSchemaPath: "objects/AuditAnchorPresenceGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.AuditAnchorPresenceGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleAuditAnchorPresenceGet(ctx, req, meta)
			})
		}},
		"audit_anchor_segment": {requestSchemaPath: "objects/AuditAnchorSegmentRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.AuditAnchorSegmentRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleAuditAnchorSegment(ctx, req, meta)
			})
		}},
		"readiness_get": {requestSchemaPath: "objects/ReadinessGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ReadinessGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleReadinessGet(ctx, req, meta)
			})
		}},
		"version_info_get": {requestSchemaPath: "objects/VersionInfoGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.VersionInfoGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleVersionInfoGet(ctx, req, meta)
			})
		}},
	}
}
