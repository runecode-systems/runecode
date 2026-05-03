package brokerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/secretsd"
)

func (s *Service) HandleExternalAnchorMutationIssueExecuteLease(ctx context.Context, req ExternalAnchorMutationIssueExecuteLeaseRequest, meta RequestContext) (ExternalAnchorMutationIssueExecuteLeaseResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.beginGitRemoteMutationRequest(ctx, req, req.RequestID, meta, externalAnchorMutationIssueExecuteLeaseRequestSchemaPath)
	if errResp != nil {
		return ExternalAnchorMutationIssueExecuteLeaseResponse{}, errResp
	}
	defer cleanup()

	preparedMutationID := strings.TrimSpace(req.PreparedMutationID)
	if preparedMutationID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "prepared_mutation_id is required")
		return ExternalAnchorMutationIssueExecuteLeaseResponse{}, &errOut
	}
	record, ok := s.ExternalAnchorPreparedGet(preparedMutationID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_prepared_mutation", "storage", false, fmt.Sprintf("prepared mutation %q not found", preparedMutationID))
		return ExternalAnchorMutationIssueExecuteLeaseResponse{}, &errOut
	}
	if strings.TrimSpace(record.LifecycleState) != gitRemoteMutationLifecyclePrepared {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "prepared mutation must remain prepared before issuing execute lease")
		return ExternalAnchorMutationIssueExecuteLeaseResponse{}, &errOut
	}
	lease, err := s.issueExternalAnchorExecutionLease(record, req.TTLSeconds)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ExternalAnchorMutationIssueExecuteLeaseResponse{}, &errOut
	}
	resp := ExternalAnchorMutationIssueExecuteLeaseResponse{
		SchemaID:           "runecode.protocol.v0.ExternalAnchorMutationIssueExecuteLeaseResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          requestID,
		PreparedMutationID: preparedMutationID,
		Lease:              lease,
		TargetAuthLeaseID:  strings.TrimSpace(lease.LeaseID),
	}
	if err := s.validateResponse(resp, externalAnchorMutationIssueExecuteLeaseResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ExternalAnchorMutationIssueExecuteLeaseResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) issueExternalAnchorExecutionLease(record artifacts.ExternalAnchorPreparedMutationRecord, ttlSeconds int) (secretsd.Lease, error) {
	if s == nil || s.secretsSvc == nil {
		return secretsd.Lease{}, fmt.Errorf("secrets service unavailable for external anchor execution lease issue")
	}
	if strings.TrimSpace(record.DestinationRef) == "" {
		return secretsd.Lease{}, fmt.Errorf("external anchor prepared mutation missing destination_ref")
	}
	if strings.TrimSpace(record.ActionRequestHash) == "" || strings.TrimSpace(record.PolicyDecisionHash) == "" {
		return secretsd.Lease{}, fmt.Errorf("external anchor prepared mutation missing bound action/policy hashes")
	}
	lease, err := s.secretsSvc.IssueLease(secretsd.IssueLeaseRequest{
		SecretRef:    gitRemoteProviderTokenSecretRef,
		ConsumerID:   "principal:gateway:git:1",
		RoleKind:     "git-gateway",
		Scope:        "run:" + strings.TrimSpace(record.RunID),
		DeliveryKind: "git_gateway",
		TTLSeconds:   ttlSeconds,
		GitBinding: &secretsd.GitLeaseBinding{
			RepositoryIdentity: strings.TrimSpace(record.DestinationRef),
			AllowedOperations:  []string{"external_anchor_submit"},
			ActionRequestHash:  strings.TrimSpace(record.ActionRequestHash),
			PolicyContextHash:  strings.TrimSpace(record.PolicyDecisionHash),
		},
	})
	if err != nil {
		return secretsd.Lease{}, err
	}
	return lease, nil
}
