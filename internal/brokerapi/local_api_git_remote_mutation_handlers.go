package brokerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

const (
	gitRemoteMutationLifecyclePrepared       = "prepared"
	gitRemoteMutationLifecycleExecuting      = "executing"
	gitRemoteMutationLifecycleExecuted       = "executed"
	gitRemoteMutationLifecycleFailed         = "failed"
	gitRemoteMutationExecutionNotStarted     = "not_started"
	gitRemoteMutationExecutionDeferred       = "deferred"
	gitRemoteMutationExecutionNotImplemented = "execution_not_implemented"
	gitRemoteMutationExecutionCompleted      = "completed"
	gitRemoteMutationExecutionBlocked        = "blocked"
	gitRemoteMutationExecutionFailed         = "failed"
	gitRemoteMutationLifecycleDeferredReason = "execution_deferred"
	gitRemoteMutationZeroObjectID            = "0000000000000000000000000000000000000000"
)

func (s *Service) HandleGitRemoteMutationPrepare(ctx context.Context, req GitRemoteMutationPrepareRequest, meta RequestContext) (GitRemoteMutationPrepareResponse, *ErrorResponse) {
	requestID, requestCtx, cleanup, errResp := s.beginGitRemoteMutationRequest(ctx, req, req.RequestID, meta, gitRemoteMutationPrepareRequestSchemaPath)
	if errResp != nil {
		return GitRemoteMutationPrepareResponse{}, errResp
	}
	defer cleanup()

	resolved, errResp := s.resolveGitRemotePrepareInput(req, requestID)
	if errResp != nil {
		return GitRemoteMutationPrepareResponse{}, errResp
	}
	decision, approvalBinding, policyDecisionHash, errResp := s.evaluatePreparedGitRemoteMutation(requestCtx, requestID, resolved, req.TypedRequest)
	if errResp != nil {
		return GitRemoteMutationPrepareResponse{}, errResp
	}
	record, errResp := s.buildPreparedGitRemoteMutationRecord(req, requestID, resolved, decision, approvalBinding, policyDecisionHash)
	if errResp != nil {
		return GitRemoteMutationPrepareResponse{}, errResp
	}
	return s.persistPreparedGitRemoteMutationResponse(requestID, record, resolved.typedRequestHash)
}

func (s *Service) HandleGitRemoteMutationGet(ctx context.Context, req GitRemoteMutationGetRequest, meta RequestContext) (GitRemoteMutationGetResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.beginGitRemoteMutationRequest(ctx, req, req.RequestID, meta, gitRemoteMutationGetRequestSchemaPath)
	if errResp != nil {
		return GitRemoteMutationGetResponse{}, errResp
	}
	defer cleanup()

	preparedMutationID := strings.TrimSpace(req.PreparedMutationID)
	if preparedMutationID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "prepared_mutation_id is required")
		return GitRemoteMutationGetResponse{}, &errOut
	}
	record, ok := s.GitRemotePreparedGet(preparedMutationID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_prepared_mutation", "storage", false, fmt.Sprintf("prepared mutation %q not found", preparedMutationID))
		return GitRemoteMutationGetResponse{}, &errOut
	}
	record.LastGetRequestID = requestID
	if err := s.GitRemotePreparedUpsert(record); err != nil {
		errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return GitRemoteMutationGetResponse{}, &errOut
	}
	preparedState, errResp := s.readPreparedGitRemoteMutationState(requestID, preparedMutationID)
	if errResp != nil {
		return GitRemoteMutationGetResponse{}, errResp
	}
	resp := GitRemoteMutationGetResponse{
		SchemaID:      "runecode.protocol.v0.GitRemoteMutationGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Prepared:      preparedState,
	}
	if err := s.validateResponse(resp, gitRemoteMutationGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return GitRemoteMutationGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleGitRemoteMutationExecute(ctx context.Context, req GitRemoteMutationExecuteRequest, meta RequestContext) (GitRemoteMutationExecuteResponse, *ErrorResponse) {
	requestID, requestCtx, cleanup, errResp := s.beginGitRemoteMutationRequest(ctx, req, req.RequestID, meta, gitRemoteMutationExecuteRequestSchemaPath)
	if errResp != nil {
		return GitRemoteMutationExecuteResponse{}, errResp
	}
	defer cleanup()

	record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, errResp := s.resolveGitRemoteExecuteRequest(req, requestID)
	if errResp != nil {
		return GitRemoteMutationExecuteResponse{}, errResp
	}
	attemptBinding, errResp := s.gitRemoteExecuteAttemptBinding(requestID, record)
	if errResp != nil {
		return GitRemoteMutationExecuteResponse{}, errResp
	}
	snapshot, errResp := s.snapshotGitRemoteExecutionInputs(requestID)
	if errResp != nil {
		return GitRemoteMutationExecuteResponse{}, errResp
	}
	record, errResp = s.beginGitRemotePreparedExecution(requestID, record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, req.ProviderAuthLeaseID, attemptBinding, snapshot)
	if errResp != nil {
		return GitRemoteMutationExecuteResponse{}, errResp
	}
	if errResp := s.executeAndPersistGitRemoteMutation(requestCtx, requestID, req.ProviderAuthLeaseID, &record); errResp != nil {
		return GitRemoteMutationExecuteResponse{}, errResp
	}
	return s.gitRemoteMutationExecuteResponse(requestID, record.PreparedMutationID)
}

func (s *Service) executeAndPersistGitRemoteMutation(ctx context.Context, requestID, providerAuthLeaseID string, record *artifacts.GitRemotePreparedMutationRecord) *ErrorResponse {
	proof, execErr := s.executePreparedGitRemoteMutation(ctx, *record, providerAuthLeaseID)
	if execErr != nil {
		return s.handleGitRemoteExecutionFailure(requestID, record, proof, execErr)
	}
	return s.handleGitRemoteExecutionSuccess(requestID, record, proof)
}

func (s *Service) handleGitRemoteExecutionFailure(requestID string, record *artifacts.GitRemotePreparedMutationRecord, proof gitRuntimeProofPayload, execErr *gitRemoteExecutionError) *ErrorResponse {
	*record = failedGitRemoteMutationExecutionRecord(*record, execErr)
	if errResp := s.persistGitRemotePreparedRecord(requestID, *record); errResp != nil {
		return errResp
	}
	if err := s.appendGitRemoteExecutionAudit(*record, proof, "failed"); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("git remote failure audit emit failed: %v", err))
		return &errOut
	}
	errOut := s.makeError(requestID, execErr.code, execErr.category, execErr.retryable, execErr.message)
	return &errOut
}

func (s *Service) handleGitRemoteExecutionSuccess(requestID string, record *artifacts.GitRemotePreparedMutationRecord, proof gitRuntimeProofPayload) *ErrorResponse {
	*record = completedGitRemoteMutationExecutionRecord(*record)
	if errResp := s.persistGitRemotePreparedRecord(requestID, *record); errResp != nil {
		return errResp
	}
	if err := s.appendGitRemoteExecutionAudit(*record, proof, "succeeded"); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("git remote success audit emit failed: %v", err))
		return &errOut
	}
	return nil
}
