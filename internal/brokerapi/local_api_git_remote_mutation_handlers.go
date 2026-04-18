package brokerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
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
	record, errResp = s.beginGitRemotePreparedExecution(requestID, record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity)
	if errResp != nil {
		return GitRemoteMutationExecuteResponse{}, errResp
	}

	proof, execErr := s.executePreparedGitRemoteMutation(requestCtx, record, req.ProviderAuthLeaseID)
	if execErr != nil {
		record = failedGitRemoteMutationExecutionRecord(record, execErr)
		if errResp := s.persistGitRemotePreparedRecord(requestID, record); errResp != nil {
			return GitRemoteMutationExecuteResponse{}, errResp
		}
		if err := s.appendGitRemoteExecutionAudit(record, proof, "failed"); err != nil {
			errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("git remote failure audit emit failed: %v", err))
			return GitRemoteMutationExecuteResponse{}, &errOut
		}
		errOut := s.makeError(requestID, execErr.code, execErr.category, execErr.retryable, execErr.message)
		return GitRemoteMutationExecuteResponse{}, &errOut
	}
	record = completedGitRemoteMutationExecutionRecord(record)
	if errResp := s.persistGitRemotePreparedRecord(requestID, record); errResp != nil {
		return GitRemoteMutationExecuteResponse{}, errResp
	}
	if err := s.appendGitRemoteExecutionAudit(record, proof, "succeeded"); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("git remote success audit emit failed: %v", err))
		return GitRemoteMutationExecuteResponse{}, &errOut
	}
	return s.gitRemoteMutationExecuteResponse(requestID, record.PreparedMutationID)
}

func (s *Service) beginGitRemoteMutationRequest(ctx context.Context, req any, requestID string, meta RequestContext, schemaPath string) (string, context.Context, func(), *ErrorResponse) {
	resolvedRequestID, errResp := s.prepareLocalRequest(requestID, meta.RequestID, meta.AdmissionErr, req, schemaPath)
	if errResp != nil {
		return "", nil, nil, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(resolvedRequestID, err)
		return "", nil, nil, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	cleanup := func() {
		cancel()
		release()
	}
	if err := requestCtx.Err(); err != nil {
		cleanup()
		errOut := s.errorFromContext(resolvedRequestID, err)
		return "", nil, nil, &errOut
	}
	return resolvedRequestID, requestCtx, cleanup, nil
}

func (s *Service) beginGitRemotePreparedExecution(requestID string, record artifacts.GitRemotePreparedMutationRecord, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity string) (artifacts.GitRemotePreparedMutationRecord, *ErrorResponse) {
	updated, err := s.GitRemotePreparedTransitionLifecycle(record.PreparedMutationID, gitRemoteMutationLifecyclePrepared, func(current artifacts.GitRemotePreparedMutationRecord) artifacts.GitRemotePreparedMutationRecord {
		return prepareGitRemoteMutationForExecution(current, requestID, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity)
	})
	if err == nil {
		return updated, nil
	}
	if current, ok := s.GitRemotePreparedGet(record.PreparedMutationID); ok && strings.TrimSpace(current.LifecycleState) != gitRemoteMutationLifecyclePrepared {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "prepared mutation is not in executable prepared state")
		return artifacts.GitRemotePreparedMutationRecord{}, &errOut
	}
	errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
	return artifacts.GitRemotePreparedMutationRecord{}, &errOut
}

func (s *Service) persistPreparedGitRemoteMutationResponse(requestID string, record artifacts.GitRemotePreparedMutationRecord, typedRequestHash trustpolicy.Digest) (GitRemoteMutationPrepareResponse, *ErrorResponse) {
	if errResp := s.persistGitRemotePreparedRecord(requestID, record); errResp != nil {
		return GitRemoteMutationPrepareResponse{}, errResp
	}
	preparedState, errResp := s.readPreparedGitRemoteMutationState(requestID, record.PreparedMutationID)
	if errResp != nil {
		return GitRemoteMutationPrepareResponse{}, errResp
	}
	resp := GitRemoteMutationPrepareResponse{
		SchemaID:           "runecode.protocol.v0.GitRemoteMutationPrepareResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          requestID,
		PreparedMutationID: record.PreparedMutationID,
		TypedRequestHash:   typedRequestHash,
		Prepared:           preparedState,
	}
	if err := s.validateResponse(resp, gitRemoteMutationPrepareResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return GitRemoteMutationPrepareResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) gitRemoteMutationExecuteResponse(requestID, preparedMutationID string) (GitRemoteMutationExecuteResponse, *ErrorResponse) {
	preparedState, errResp := s.readPreparedGitRemoteMutationState(requestID, preparedMutationID)
	if errResp != nil {
		return GitRemoteMutationExecuteResponse{}, errResp
	}
	resp := GitRemoteMutationExecuteResponse{
		SchemaID:           "runecode.protocol.v0.GitRemoteMutationExecuteResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          requestID,
		PreparedMutationID: preparedMutationID,
		ExecutionState:     preparedState.ExecutionState,
		Prepared:           preparedState,
	}
	if err := s.validateResponse(resp, gitRemoteMutationExecuteResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return GitRemoteMutationExecuteResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) persistGitRemotePreparedRecord(requestID string, record artifacts.GitRemotePreparedMutationRecord) *ErrorResponse {
	if err := s.GitRemotePreparedUpsert(record); err != nil {
		errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return &errOut
	}
	return nil
}

func (s *Service) readPreparedGitRemoteMutationState(requestID, preparedMutationID string) (GitRemoteMutationPreparedState, *ErrorResponse) {
	record, ok := s.GitRemotePreparedGet(preparedMutationID)
	if !ok {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "prepared mutation unavailable after persistence")
		return GitRemoteMutationPreparedState{}, &errOut
	}
	preparedState, err := gitPreparedStateFromRecord(record)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return GitRemoteMutationPreparedState{}, &errOut
	}
	return preparedState, nil
}

func prepareGitRemoteMutationForExecution(record artifacts.GitRemotePreparedMutationRecord, requestID, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity string) artifacts.GitRemotePreparedMutationRecord {
	record.LifecycleState = gitRemoteMutationLifecycleExecuting
	record.ExecutionState = gitRemoteMutationExecutionNotStarted
	record.ExecutionReasonCode = ""
	record.RequiredApprovalReqHash = approvalRequestHashIdentity
	record.RequiredApprovalDecHash = approvalDecisionHashIdentity
	record.LastExecuteApprovalID = approvalID
	record.LastExecuteApprovalReqID = approvalRequestHashIdentity
	record.LastExecuteApprovalDecID = approvalDecisionHashIdentity
	record.LastExecuteRequestID = requestID
	return record
}

func failedGitRemoteMutationExecutionRecord(record artifacts.GitRemotePreparedMutationRecord, execErr *gitRemoteExecutionError) artifacts.GitRemotePreparedMutationRecord {
	record.ExecutionState = execErr.executionState
	record.ExecutionReasonCode = execErr.reasonCode
	record.LifecycleState = gitRemoteMutationLifecycleFailed
	record.LifecycleReasonCode = execErr.reasonCode
	return record
}

func completedGitRemoteMutationExecutionRecord(record artifacts.GitRemotePreparedMutationRecord) artifacts.GitRemotePreparedMutationRecord {
	record.ExecutionState = gitRemoteMutationExecutionCompleted
	record.ExecutionReasonCode = ""
	record.LifecycleState = gitRemoteMutationLifecycleExecuted
	record.LifecycleReasonCode = ""
	return record
}
