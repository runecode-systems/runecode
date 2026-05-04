package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

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
	operations := providerLifecycleRPCOperations(service, ctx, meta)
	mergeRPCOperations(operations, projectSubstrateRPCOperations(service, ctx, meta))
	return operations
}

func providerLifecycleRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
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
		"provider_setup_secret_ingress_submit": {requestSchemaPath: "objects/ProviderSetupSecretIngressSubmitRequest.schema.json", handleWire: func(wire localRPCRequest) localRPCResponse {
			return decodeAndHandleProviderSecretIngressSubmit(service, ctx, wire, meta)
		}},
		"provider_validation_begin": {requestSchemaPath: "objects/ProviderValidationBeginRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProviderValidationBeginRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProviderValidationBegin(ctx, req, meta)
			})
		}},
		"provider_validation_commit": {requestSchemaPath: "objects/ProviderValidationCommitRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProviderValidationCommitRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProviderValidationCommit(ctx, req, meta)
			})
		}},
		"provider_credential_lease_issue": {requestSchemaPath: "objects/ProviderCredentialLeaseIssueRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProviderCredentialLeaseIssueRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProviderCredentialLeaseIssue(ctx, req, meta)
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

func projectSubstrateRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"project_substrate_get": {requestSchemaPath: "objects/ProjectSubstrateGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProjectSubstrateGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProjectSubstrateGet(ctx, req, meta)
			})
		}},
		"project_substrate_posture_get": {requestSchemaPath: "objects/ProjectSubstratePostureGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProjectSubstratePostureGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProjectSubstratePostureGet(ctx, req, meta)
			})
		}},
		"project_substrate_adopt": {requestSchemaPath: "objects/ProjectSubstrateAdoptRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProjectSubstrateAdoptRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProjectSubstrateAdopt(ctx, req, meta)
			})
		}},
		"project_substrate_init_preview": {requestSchemaPath: "objects/ProjectSubstrateInitPreviewRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProjectSubstrateInitPreviewRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProjectSubstrateInitPreview(ctx, req, meta)
			})
		}},
		"project_substrate_init_apply": {requestSchemaPath: "objects/ProjectSubstrateInitApplyRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProjectSubstrateInitApplyRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProjectSubstrateInitApply(ctx, req, meta)
			})
		}},
		"project_substrate_upgrade_preview": {requestSchemaPath: "objects/ProjectSubstrateUpgradePreviewRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProjectSubstrateUpgradePreviewRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProjectSubstrateUpgradePreview(ctx, req, meta)
			})
		}},
		"project_substrate_upgrade_apply": {requestSchemaPath: "objects/ProjectSubstrateUpgradeApplyRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ProjectSubstrateUpgradeApplyRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleProjectSubstrateUpgradeApply(ctx, req, meta)
			})
		}},
	}
}

func decodeAndHandleProviderSecretIngressSubmit(service *brokerapi.Service, ctx context.Context, wire localRPCRequest, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.ProviderSetupSecretIngressSubmitRequest{}
	decoder := json.NewDecoder(bytes.NewReader(wire.Request))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(wire.SecretIngressPayloadBase64))
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(req.RequestID, err)}
	}
	resp, errResp := service.HandleProviderSetupSecretIngressSubmit(ctx, req, payload, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	return localRPCOKResponse(resp)
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
		"external_anchor_mutation_prepare": {requestSchemaPath: "objects/ExternalAnchorMutationPrepareRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ExternalAnchorMutationPrepareRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleExternalAnchorMutationPrepare(ctx, req, meta)
			})
		}},
		"external_anchor_mutation_get": {requestSchemaPath: "objects/ExternalAnchorMutationGetRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ExternalAnchorMutationGetRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleExternalAnchorMutationGet(ctx, req, meta)
			})
		}},
		"external_anchor_mutation_issue_execute_lease": {requestSchemaPath: "objects/ExternalAnchorMutationIssueExecuteLeaseRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ExternalAnchorMutationIssueExecuteLeaseRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleExternalAnchorMutationIssueExecuteLease(ctx, req, meta)
			})
		}},
		"external_anchor_mutation_execute": {requestSchemaPath: "objects/ExternalAnchorMutationExecuteRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ExternalAnchorMutationExecuteRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleExternalAnchorMutationExecute(ctx, req, meta)
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
		"artifact_put": {requestSchemaPath: "objects/BrokerArtifactPutRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.ArtifactPutRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleArtifactPut(ctx, req, meta)
			})
		}},
		"artifact_read": {requestSchemaPath: "objects/ArtifactReadRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandleArtifactRead(service, ctx, raw, meta)
		}},
		"dependency_cache_ensure": {requestSchemaPath: "objects/DependencyCacheEnsureRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.DependencyCacheEnsureRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleDependencyCacheEnsure(ctx, req, meta)
			})
		}},
		"dependency_fetch_registry": {requestSchemaPath: "objects/DependencyFetchRegistryRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.DependencyFetchRegistryRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleDependencyFetchRegistry(ctx, req, meta)
			})
		}},
		"dependency_cache_handoff": {requestSchemaPath: "objects/DependencyCacheHandoffRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse {
			return decodeAndHandle(raw, func(req brokerapi.DependencyCacheHandoffRequest) (any, *brokerapi.ErrorResponse) {
				return service.HandleDependencyCacheHandoff(ctx, req, meta)
			})
		}},
		"log_stream": {requestSchemaPath: "objects/LogStreamRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse { return decodeAndHandleLogStream(service, ctx, raw, meta) }},
		"llm_invoke": {requestSchemaPath: "objects/LLMInvokeRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse { return decodeAndHandleLLMInvoke(service, ctx, raw, meta) }},
		"llm_stream": {requestSchemaPath: "objects/LLMStreamRequest.schema.json", handle: func(raw json.RawMessage) localRPCResponse { return decodeAndHandleLLMStream(service, ctx, raw, meta) }},
	}
}

func auditHealthRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	operations := auditEvidenceRPCOperations(service, ctx, meta)
	mergeRPCOperations(operations, auditAnchorRPCOperations(service, ctx, meta))
	mergeRPCOperations(operations, auditStatusRPCOperations(service, ctx, meta))
	return operations
}

func auditEvidenceRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"audit_timeline": auditRPCOperation("objects/AuditTimelineRequest.schema.json", func(req brokerapi.AuditTimelineRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditTimeline(ctx, req, meta)
		}),
		"audit_verification_get": auditRPCOperation("objects/AuditVerificationGetRequest.schema.json", func(req brokerapi.AuditVerificationGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditVerificationGet(ctx, req, meta)
		}),
		"audit_finalize_verify": auditRPCOperation("objects/AuditFinalizeVerifyRequest.schema.json", func(req brokerapi.AuditFinalizeVerifyRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditFinalizeVerify(ctx, req, meta)
		}),
		"audit_record_get": auditRPCOperation("objects/AuditRecordGetRequest.schema.json", func(req brokerapi.AuditRecordGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditRecordGet(ctx, req, meta)
		}),
		"audit_record_inclusion_get": auditRPCOperation("objects/AuditRecordInclusionGetRequest.schema.json", func(req brokerapi.AuditRecordInclusionGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditRecordInclusionGet(ctx, req, meta)
		}),
		"audit_evidence_snapshot_get": auditRPCOperation("objects/AuditEvidenceSnapshotGetRequest.schema.json", func(req brokerapi.AuditEvidenceSnapshotGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditEvidenceSnapshotGet(ctx, req, meta)
		}),
		"audit_evidence_retention_review": auditRPCOperation("objects/AuditEvidenceRetentionReviewRequest.schema.json", func(req brokerapi.AuditEvidenceRetentionReviewRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditEvidenceRetentionReview(ctx, req, meta)
		}),
		"audit_evidence_bundle_manifest_get": auditRPCOperation("objects/AuditEvidenceBundleManifestGetRequest.schema.json", func(req brokerapi.AuditEvidenceBundleManifestGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditEvidenceBundleManifestGet(ctx, req, meta)
		}),
		"audit_evidence_bundle_export": auditRPCOperation("objects/AuditEvidenceBundleExportRequest.schema.json", func(req brokerapi.AuditEvidenceBundleExportRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditEvidenceBundleExport(ctx, req, meta)
		}),
		"audit_evidence_bundle_offline_verify": auditRPCOperation("objects/AuditEvidenceBundleOfflineVerifyRequest.schema.json", func(req brokerapi.AuditEvidenceBundleOfflineVerifyRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditEvidenceBundleOfflineVerify(ctx, req, meta)
		}),
	}
}

func auditAnchorRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"audit_anchor_preflight_get": auditRPCOperation("objects/AuditAnchorPreflightGetRequest.schema.json", func(req brokerapi.AuditAnchorPreflightGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditAnchorPreflightGet(ctx, req, meta)
		}),
		"audit_anchor_presence_get": auditRPCOperation("objects/AuditAnchorPresenceGetRequest.schema.json", func(req brokerapi.AuditAnchorPresenceGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditAnchorPresenceGet(ctx, req, meta)
		}),
		"audit_anchor_segment": auditRPCOperation("objects/AuditAnchorSegmentRequest.schema.json", func(req brokerapi.AuditAnchorSegmentRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleAuditAnchorSegment(ctx, req, meta)
		}),
	}
}

func auditStatusRPCOperations(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) map[string]rpcOperation {
	return map[string]rpcOperation{
		"readiness_get": auditRPCOperation("objects/ReadinessGetRequest.schema.json", func(req brokerapi.ReadinessGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleReadinessGet(ctx, req, meta)
		}),
		"version_info_get": auditRPCOperation("objects/VersionInfoGetRequest.schema.json", func(req brokerapi.VersionInfoGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleVersionInfoGet(ctx, req, meta)
		}),
		"product_lifecycle_posture_get": auditRPCOperation("objects/ProductLifecyclePostureGetRequest.schema.json", func(req brokerapi.ProductLifecyclePostureGetRequest) (any, *brokerapi.ErrorResponse) {
			return service.HandleProductLifecyclePostureGet(ctx, req, meta)
		}),
	}
}

func auditRPCOperation[T any](schemaPath string, handleFn func(T) (any, *brokerapi.ErrorResponse)) rpcOperation {
	return rpcOperation{requestSchemaPath: schemaPath, handle: func(raw json.RawMessage) localRPCResponse {
		return decodeAndHandle(raw, handleFn)
	}}
}
