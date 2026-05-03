package brokerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/secretsd"
)

const gitRemoteProviderTokenSecretRef = "secrets/prod/git/provider-token"

func (s *Service) HandleGitRemoteMutationIssueExecuteLease(ctx context.Context, req GitRemoteMutationIssueExecuteLeaseRequest, meta RequestContext) (GitRemoteMutationIssueExecuteLeaseResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.beginGitRemoteMutationRequest(ctx, req, req.RequestID, meta, gitRemoteMutationIssueExecuteLeaseRequestSchemaPath)
	if errResp != nil {
		return GitRemoteMutationIssueExecuteLeaseResponse{}, errResp
	}
	defer cleanup()

	preparedMutationID := strings.TrimSpace(req.PreparedMutationID)
	if preparedMutationID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "prepared_mutation_id is required")
		return GitRemoteMutationIssueExecuteLeaseResponse{}, &errOut
	}
	record, ok := s.GitRemotePreparedGet(preparedMutationID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_prepared_mutation", "storage", false, fmt.Sprintf("prepared mutation %q not found", preparedMutationID))
		return GitRemoteMutationIssueExecuteLeaseResponse{}, &errOut
	}
	if strings.TrimSpace(record.LifecycleState) != gitRemoteMutationLifecyclePrepared {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "prepared mutation must remain prepared before issuing execute lease")
		return GitRemoteMutationIssueExecuteLeaseResponse{}, &errOut
	}
	lease, err := s.issueGitRemoteExecutionLease(record, req.TTLSeconds)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return GitRemoteMutationIssueExecuteLeaseResponse{}, &errOut
	}
	resp := GitRemoteMutationIssueExecuteLeaseResponse{
		SchemaID:            "runecode.protocol.v0.GitRemoteMutationIssueExecuteLeaseResponse",
		SchemaVersion:       "0.1.0",
		RequestID:           requestID,
		PreparedMutationID:  preparedMutationID,
		Lease:               lease,
		ProviderAuthLeaseID: strings.TrimSpace(lease.LeaseID),
	}
	if err := s.validateResponse(resp, gitRemoteMutationIssueExecuteLeaseResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return GitRemoteMutationIssueExecuteLeaseResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) issueGitRemoteExecutionLease(record artifacts.GitRemotePreparedMutationRecord, ttlSeconds int) (secretsd.Lease, error) {
	if s == nil || s.secretsSvc == nil {
		return secretsd.Lease{}, fmt.Errorf("secrets service unavailable for git execution lease issue")
	}
	repository, err := decodeExecutionRepository(record)
	if err != nil {
		return secretsd.Lease{}, err
	}
	lease, err := s.secretsSvc.IssueLease(secretsd.IssueLeaseRequest{
		SecretRef:    gitRemoteProviderTokenSecretRef,
		ConsumerID:   "principal:gateway:git:1",
		RoleKind:     "git-gateway",
		Scope:        "run:" + strings.TrimSpace(record.RunID),
		DeliveryKind: "git_gateway",
		TTLSeconds:   ttlSeconds,
		GitBinding: &secretsd.GitLeaseBinding{
			RepositoryIdentity: repository.repositoryIdentity,
			AllowedOperations:  []string{gitSecretOperationForRequestKind(record.RequestKind)},
			ActionRequestHash:  strings.TrimSpace(record.ActionRequestHash),
			PolicyContextHash:  strings.TrimSpace(record.PolicyDecisionHash),
		},
	})
	if err != nil {
		return secretsd.Lease{}, err
	}
	return lease, nil
}
