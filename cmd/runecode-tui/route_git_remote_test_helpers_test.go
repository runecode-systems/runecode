package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (r *recordingBrokerClient) GitRemoteMutationPrepare(ctx context.Context, req brokerapi.GitRemoteMutationPrepareRequest) (brokerapi.GitRemoteMutationPrepareResponse, error) {
	r.record("GitRemoteMutationPrepare")
	return r.base.GitRemoteMutationPrepare(ctx, req)
}

func (r *recordingBrokerClient) GitRemoteMutationGet(ctx context.Context, req brokerapi.GitRemoteMutationGetRequest) (brokerapi.GitRemoteMutationGetResponse, error) {
	r.record("GitRemoteMutationGet")
	return r.base.GitRemoteMutationGet(ctx, req)
}

func (r *recordingBrokerClient) GitRemoteMutationIssueExecuteLease(ctx context.Context, req brokerapi.GitRemoteMutationIssueExecuteLeaseRequest) (brokerapi.GitRemoteMutationIssueExecuteLeaseResponse, error) {
	r.record("GitRemoteMutationIssueExecuteLease")
	return r.base.GitRemoteMutationIssueExecuteLease(ctx, req)
}

func (r *recordingBrokerClient) GitRemoteMutationExecute(ctx context.Context, req brokerapi.GitRemoteMutationExecuteRequest) (brokerapi.GitRemoteMutationExecuteResponse, error) {
	r.record("GitRemoteMutationExecute")
	return r.base.GitRemoteMutationExecute(ctx, req)
}

func (r *recordingBrokerClient) ExternalAnchorMutationPrepare(ctx context.Context, req brokerapi.ExternalAnchorMutationPrepareRequest) (brokerapi.ExternalAnchorMutationPrepareResponse, error) {
	r.record("ExternalAnchorMutationPrepare")
	return r.base.ExternalAnchorMutationPrepare(ctx, req)
}

func (r *recordingBrokerClient) ExternalAnchorMutationGet(ctx context.Context, req brokerapi.ExternalAnchorMutationGetRequest) (brokerapi.ExternalAnchorMutationGetResponse, error) {
	r.record("ExternalAnchorMutationGet")
	return r.base.ExternalAnchorMutationGet(ctx, req)
}

func (r *recordingBrokerClient) ExternalAnchorMutationExecute(ctx context.Context, req brokerapi.ExternalAnchorMutationExecuteRequest) (brokerapi.ExternalAnchorMutationExecuteResponse, error) {
	r.record("ExternalAnchorMutationExecute")
	return r.base.ExternalAnchorMutationExecute(ctx, req)
}

func (f *reloadAwareBrokerClient) GitRemoteMutationPrepare(ctx context.Context, req brokerapi.GitRemoteMutationPrepareRequest) (brokerapi.GitRemoteMutationPrepareResponse, error) {
	return (&fakeBrokerClient{}).GitRemoteMutationPrepare(ctx, req)
}

func (f *reloadAwareBrokerClient) GitRemoteMutationGet(ctx context.Context, req brokerapi.GitRemoteMutationGetRequest) (brokerapi.GitRemoteMutationGetResponse, error) {
	return (&fakeBrokerClient{}).GitRemoteMutationGet(ctx, req)
}

func (f *reloadAwareBrokerClient) GitRemoteMutationIssueExecuteLease(ctx context.Context, req brokerapi.GitRemoteMutationIssueExecuteLeaseRequest) (brokerapi.GitRemoteMutationIssueExecuteLeaseResponse, error) {
	return (&fakeBrokerClient{}).GitRemoteMutationIssueExecuteLease(ctx, req)
}

func (f *reloadAwareBrokerClient) GitRemoteMutationExecute(ctx context.Context, req brokerapi.GitRemoteMutationExecuteRequest) (brokerapi.GitRemoteMutationExecuteResponse, error) {
	return (&fakeBrokerClient{}).GitRemoteMutationExecute(ctx, req)
}

func (f *reloadAwareBrokerClient) ExternalAnchorMutationPrepare(ctx context.Context, req brokerapi.ExternalAnchorMutationPrepareRequest) (brokerapi.ExternalAnchorMutationPrepareResponse, error) {
	return (&fakeBrokerClient{}).ExternalAnchorMutationPrepare(ctx, req)
}

func (f *reloadAwareBrokerClient) ExternalAnchorMutationGet(ctx context.Context, req brokerapi.ExternalAnchorMutationGetRequest) (brokerapi.ExternalAnchorMutationGetResponse, error) {
	return (&fakeBrokerClient{}).ExternalAnchorMutationGet(ctx, req)
}

func (f *reloadAwareBrokerClient) ExternalAnchorMutationExecute(ctx context.Context, req brokerapi.ExternalAnchorMutationExecuteRequest) (brokerapi.ExternalAnchorMutationExecuteResponse, error) {
	return (&fakeBrokerClient{}).ExternalAnchorMutationExecute(ctx, req)
}

func (f *fakeBrokerClient) GitRemoteMutationPrepare(ctx context.Context, req brokerapi.GitRemoteMutationPrepareRequest) (brokerapi.GitRemoteMutationPrepareResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.RunID) == "" {
		return brokerapi.GitRemoteMutationPrepareResponse{}, fmt.Errorf("run id required")
	}
	preparedID := "sha256:" + strings.Repeat("7", 64)
	prepared := fakePreparedGitRemoteMutationState(preparedID)
	return brokerapi.GitRemoteMutationPrepareResponse{
		SchemaID:           "runecode.protocol.v0.GitRemoteMutationPrepareResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-git-remote-prepare",
		PreparedMutationID: preparedID,
		TypedRequestHash:   trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)},
		Prepared:           prepared,
	}, nil
}

func (f *fakeBrokerClient) GitRemoteMutationGet(ctx context.Context, req brokerapi.GitRemoteMutationGetRequest) (brokerapi.GitRemoteMutationGetResponse, error) {
	_ = ctx
	preparedID := strings.TrimSpace(req.PreparedMutationID)
	if preparedID == "" {
		return brokerapi.GitRemoteMutationGetResponse{}, fmt.Errorf("prepared mutation id required")
	}
	return brokerapi.GitRemoteMutationGetResponse{
		SchemaID:      "runecode.protocol.v0.GitRemoteMutationGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-git-remote-get",
		Prepared:      fakePreparedGitRemoteMutationState(preparedID),
	}, nil
}

func (f *fakeBrokerClient) GitRemoteMutationIssueExecuteLease(ctx context.Context, req brokerapi.GitRemoteMutationIssueExecuteLeaseRequest) (brokerapi.GitRemoteMutationIssueExecuteLeaseResponse, error) {
	_ = ctx
	preparedID := strings.TrimSpace(req.PreparedMutationID)
	if preparedID == "" {
		return brokerapi.GitRemoteMutationIssueExecuteLeaseResponse{}, fmt.Errorf("prepared mutation id required")
	}
	leaseID := "lease-git-provider"
	return brokerapi.GitRemoteMutationIssueExecuteLeaseResponse{
		SchemaID:           "runecode.protocol.v0.GitRemoteMutationIssueExecuteLeaseResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-git-remote-issue-execute-lease",
		PreparedMutationID: preparedID,
		Lease: secretsd.Lease{
			LeaseID:      leaseID,
			SecretRef:    "secrets/prod/git/provider-token",
			ConsumerID:   "principal:gateway:git:1",
			RoleKind:     "git-gateway",
			Scope:        "run:run-1",
			DeliveryKind: "git_gateway",
			GitBinding: &secretsd.GitLeaseBinding{
				RepositoryIdentity: "github.com/runecode-ai/runecode",
				AllowedOperations:  []string{"git_ref_update"},
				ActionRequestHash:  "sha256:" + strings.Repeat("2", 64),
				PolicyContextHash:  "sha256:" + strings.Repeat("3", 64),
			},
			Status: "active",
		},
		ProviderAuthLeaseID: leaseID,
	}, nil
}

func (f *fakeBrokerClient) GitRemoteMutationExecute(ctx context.Context, req brokerapi.GitRemoteMutationExecuteRequest) (brokerapi.GitRemoteMutationExecuteResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.PreparedMutationID) == "" {
		return brokerapi.GitRemoteMutationExecuteResponse{}, fmt.Errorf("prepared mutation id required")
	}
	if strings.TrimSpace(req.ApprovalID) == "" {
		return brokerapi.GitRemoteMutationExecuteResponse{}, fmt.Errorf("approval id required")
	}
	if _, err := req.ApprovalRequestHash.Identity(); err != nil {
		return brokerapi.GitRemoteMutationExecuteResponse{}, fmt.Errorf("approval request hash invalid")
	}
	if _, err := req.ApprovalDecisionHash.Identity(); err != nil {
		return brokerapi.GitRemoteMutationExecuteResponse{}, fmt.Errorf("approval decision hash invalid")
	}
	if strings.TrimSpace(req.ProviderAuthLeaseID) == "" {
		return brokerapi.GitRemoteMutationExecuteResponse{}, fmt.Errorf("provider auth lease id required")
	}
	prepared := fakePreparedGitRemoteMutationState(req.PreparedMutationID)
	prepared.LifecycleState = "executed"
	prepared.ExecutionState = "completed"
	prepared.ExecutionReasonCode = ""
	return brokerapi.GitRemoteMutationExecuteResponse{
		SchemaID:           "runecode.protocol.v0.GitRemoteMutationExecuteResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-git-remote-execute",
		PreparedMutationID: req.PreparedMutationID,
		ExecutionState:     "completed",
		Prepared:           prepared,
	}, nil
}

func (f *fakeBrokerClient) ExternalAnchorMutationPrepare(ctx context.Context, req brokerapi.ExternalAnchorMutationPrepareRequest) (brokerapi.ExternalAnchorMutationPrepareResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.RunID) == "" {
		return brokerapi.ExternalAnchorMutationPrepareResponse{}, fmt.Errorf("run id required")
	}
	preparedID := "sha256:" + strings.Repeat("8", 64)
	prepared := fakePreparedExternalAnchorMutationState(preparedID)
	return brokerapi.ExternalAnchorMutationPrepareResponse{
		SchemaID:           "runecode.protocol.v0.ExternalAnchorMutationPrepareResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-external-anchor-prepare",
		PreparedMutationID: preparedID,
		TypedRequestHash:   trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)},
		Prepared:           prepared,
	}, nil
}

func (f *fakeBrokerClient) ExternalAnchorMutationGet(ctx context.Context, req brokerapi.ExternalAnchorMutationGetRequest) (brokerapi.ExternalAnchorMutationGetResponse, error) {
	_ = ctx
	preparedID := strings.TrimSpace(req.PreparedMutationID)
	if preparedID == "" {
		return brokerapi.ExternalAnchorMutationGetResponse{}, fmt.Errorf("prepared mutation id required")
	}
	return brokerapi.ExternalAnchorMutationGetResponse{
		SchemaID:      "runecode.protocol.v0.ExternalAnchorMutationGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-external-anchor-get",
		Prepared:      fakePreparedExternalAnchorMutationState(preparedID),
	}, nil
}

func (f *fakeBrokerClient) ExternalAnchorMutationExecute(ctx context.Context, req brokerapi.ExternalAnchorMutationExecuteRequest) (brokerapi.ExternalAnchorMutationExecuteResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.PreparedMutationID) == "" {
		return brokerapi.ExternalAnchorMutationExecuteResponse{}, fmt.Errorf("prepared mutation id required")
	}
	if strings.TrimSpace(req.ApprovalID) == "" {
		return brokerapi.ExternalAnchorMutationExecuteResponse{}, fmt.Errorf("approval id required")
	}
	if _, err := req.ApprovalRequestHash.Identity(); err != nil {
		return brokerapi.ExternalAnchorMutationExecuteResponse{}, fmt.Errorf("approval request hash invalid")
	}
	if _, err := req.ApprovalDecisionHash.Identity(); err != nil {
		return brokerapi.ExternalAnchorMutationExecuteResponse{}, fmt.Errorf("approval decision hash invalid")
	}
	if strings.TrimSpace(req.TargetAuthLeaseID) == "" {
		return brokerapi.ExternalAnchorMutationExecuteResponse{}, fmt.Errorf("target auth lease id required")
	}
	prepared := fakePreparedExternalAnchorMutationState(req.PreparedMutationID)
	prepared.LifecycleState = "executed"
	prepared.ExecutionState = "completed"
	prepared.ExecutionReasonCode = ""
	return brokerapi.ExternalAnchorMutationExecuteResponse{
		SchemaID:           "runecode.protocol.v0.ExternalAnchorMutationExecuteResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-external-anchor-execute",
		PreparedMutationID: req.PreparedMutationID,
		ExecutionState:     "completed",
		Prepared:           prepared,
	}, nil
}

func fakePreparedGitRemoteMutationState(preparedID string) brokerapi.GitRemoteMutationPreparedState {
	requestHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	actionHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)}
	decisionHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)}
	approvalRequest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("4", 64)}
	approvalDecision := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("5", 64)}
	patchDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}
	expectedTree := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("7", 64)}
	return brokerapi.GitRemoteMutationPreparedState{
		SchemaID:                     "runecode.protocol.v0.GitRemoteMutationPreparedState",
		SchemaVersion:                "0.1.0",
		PreparedMutationID:           preparedID,
		RunID:                        "run-1",
		Provider:                     "github",
		DestinationRef:               "github.com/runecode-ai/runecode",
		RequestKind:                  "git_ref_update",
		TypedRequestSchemaID:         "runecode.protocol.v0.GitRefUpdateRequest",
		TypedRequestSchemaVersion:    "0.1.0",
		TypedRequest:                 map[string]any{"schema_id": "runecode.protocol.v0.GitRefUpdateRequest", "schema_version": "0.1.0", "request_kind": "git_ref_update", "target_ref": "refs/heads/main"},
		TypedRequestHash:             requestHash,
		ActionRequestHash:            actionHash,
		PolicyDecisionHash:           decisionHash,
		RequiredApprovalID:           "sha256:" + strings.Repeat("a", 64),
		RequiredApprovalRequestHash:  &approvalRequest,
		RequiredApprovalDecisionHash: &approvalDecision,
		LifecycleState:               "prepared",
		ExecutionState:               "not_started",
		CreatedAt:                    "2026-01-01T00:00:00Z",
		UpdatedAt:                    "2026-01-01T00:00:00Z",
		DerivedSummary: brokerapi.GitRemoteMutationDerivedSummary{
			SchemaID:                      "runecode.protocol.v0.GitRemoteMutationDerivedSummary",
			SchemaVersion:                 "0.1.0",
			RepositoryIdentity:            "github.com/runecode-ai/runecode",
			TargetRefs:                    []string{"refs/heads/main"},
			ReferencedPatchArtifactHashes: []trustpolicy.Digest{patchDigest},
			ExpectedResultTreeHash:        expectedTree,
			CommitSubject:                 "Apply reviewed patch",
		},
		LastPrepareRequestID: "req-git-remote-prepare",
		LastGetRequestID:     "req-git-remote-get",
	}
}

func fakePreparedExternalAnchorMutationState(preparedID string) brokerapi.ExternalAnchorMutationPreparedState {
	requestHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	actionHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)}
	decisionHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)}
	approvalRequest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("4", 64)}
	approvalDecision := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("5", 64)}
	return brokerapi.ExternalAnchorMutationPreparedState{
		SchemaID:                     "runecode.protocol.v0.ExternalAnchorMutationPreparedState",
		SchemaVersion:                "0.1.0",
		PreparedMutationID:           preparedID,
		RunID:                        "run-1",
		ExecutionPathway:             "non_workspace_gateway",
		AnchorPosture:                "external_configured_not_run",
		DestinationRef:               "sha256/" + strings.Repeat("2", 64),
		RequestKind:                  "external_anchor_submit_v0",
		TypedRequestSchemaID:         "runecode.protocol.v0.ExternalAnchorSubmitRequest",
		TypedRequestSchemaVersion:    "0.1.0",
		TypedRequest:                 map[string]any{"schema_id": "runecode.protocol.v0.ExternalAnchorSubmitRequest", "schema_version": "0.1.0", "request_kind": "external_anchor_submit_v0", "target_kind": "transparency_log"},
		TypedRequestHash:             requestHash,
		ActionRequestHash:            actionHash,
		PolicyDecisionHash:           decisionHash,
		RequiredApprovalID:           "sha256:" + strings.Repeat("a", 64),
		RequiredApprovalRequestHash:  &approvalRequest,
		RequiredApprovalDecisionHash: &approvalDecision,
		LifecycleState:               "prepared",
		ExecutionState:               "not_started",
		CreatedAt:                    "2026-01-01T00:00:00Z",
		UpdatedAt:                    "2026-01-01T00:00:00Z",
		LastPrepareRequestID:         "req-external-anchor-prepare",
		LastGetRequestID:             "req-external-anchor-get",
	}
}
