package brokerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) HandleExternalAnchorMutationExecute(ctx context.Context, req ExternalAnchorMutationExecuteRequest, meta RequestContext) (ExternalAnchorMutationExecuteResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.beginGitRemoteMutationRequest(ctx, req, req.RequestID, meta, externalAnchorMutationExecuteRequestSchemaPath)
	if errResp != nil {
		return ExternalAnchorMutationExecuteResponse{}, errResp
	}
	defer cleanup()

	record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, targetAuthLeaseID, errResp := s.resolveExternalAnchorExecuteRequest(req, requestID)
	if errResp != nil {
		return ExternalAnchorMutationExecuteResponse{}, errResp
	}
	attemptBinding, errResp := s.externalAnchorExecuteAttemptBinding(requestID, record)
	if errResp != nil {
		return ExternalAnchorMutationExecuteResponse{}, errResp
	}
	if strings.TrimSpace(record.LastExecuteAttemptID) == strings.TrimSpace(attemptBinding.AttemptID) && strings.TrimSpace(record.ExecutionState) != gitRemoteMutationExecutionNotStarted {
		return s.externalAnchorMutationExecuteResponse(requestID, record.PreparedMutationID)
	}
	snapshot, errResp := s.snapshotExternalAnchorExecutionInputs(requestID, record)
	if errResp != nil {
		return ExternalAnchorMutationExecuteResponse{}, errResp
	}
	record, attemptStarted, errResp := s.beginExternalAnchorPreparedExecution(requestID, record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, targetAuthLeaseID, attemptBinding, snapshot)
	if errResp != nil {
		return ExternalAnchorMutationExecuteResponse{}, errResp
	}
	if !attemptStarted {
		return s.externalAnchorMutationExecuteResponse(requestID, record.PreparedMutationID)
	}
	input, record, errResp := s.executeExternalAnchorPreparedMutation(ctx, requestID, record)
	if errResp != nil {
		return ExternalAnchorMutationExecuteResponse{}, errResp
	}
	if errResp := s.recordExternalAnchorAuditArtifacts(requestID, req.ExportReceiptCopy, record); errResp != nil {
		return ExternalAnchorMutationExecuteResponse{}, errResp
	}
	return s.respondToExternalAnchorExecution(requestID, record, input)
}

func (s *Service) executeExternalAnchorPreparedMutation(ctx context.Context, requestID string, record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorExecutionInput, artifacts.ExternalAnchorPreparedMutationRecord, *ErrorResponse) {
	input, err := externalAnchorExecutionInputFromRecord(record)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("external anchor execute input invalid: %v", err))
		return externalAnchorExecutionInput{}, artifacts.ExternalAnchorPreparedMutationRecord{}, &errOut
	}
	outcome := normalizeExternalAnchorExecutionOutcome(s.externalAnchorRuntime.Execute(ctx, input))
	updated, errResp := s.persistExternalAnchorExecutionOutcome(requestID, record.PreparedMutationID, input.AttemptID, input.PollRemaining, outcome)
	if errResp != nil {
		return externalAnchorExecutionInput{}, artifacts.ExternalAnchorPreparedMutationRecord{}, errResp
	}
	return input, updated, nil
}

func (s *Service) respondToExternalAnchorExecution(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord, input externalAnchorExecutionInput) (ExternalAnchorMutationExecuteResponse, *ErrorResponse) {
	if strings.TrimSpace(record.ExecutionState) != gitRemoteMutationExecutionDeferred {
		return s.externalAnchorMutationExecuteResponse(requestID, record.PreparedMutationID)
	}
	resp, errResp := s.externalAnchorMutationExecuteResponse(requestID, record.PreparedMutationID)
	if errResp != nil {
		return ExternalAnchorMutationExecuteResponse{}, errResp
	}
	if strings.TrimSpace(input.Mode) == "deferred_poll" {
		s.startExternalAnchorBackgroundWorkers()
		s.externalAnchorQueue.enqueue(externalAnchorPreparedExecutionAttempt{PreparedMutationID: record.PreparedMutationID, AttemptID: input.AttemptID})
	}
	return resp, nil
}
