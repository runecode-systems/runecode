package brokerapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

const (
	sessionExecutionBlockedReasonProjectSubstratePosture = "project_substrate_posture_blocked"
	sessionExecutionBlockedReasonProjectSubstrateDrift   = "project_substrate_digest_drift"
)

func (s *Service) HandleSessionExecutionTrigger(ctx context.Context, req SessionExecutionTriggerRequest, meta RequestContext) (SessionExecutionTriggerResponse, *ErrorResponse) {
	requestID, errResp := s.prepareSessionExecutionTriggerRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return SessionExecutionTriggerResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return SessionExecutionTriggerResponse{}, &errOut
	}
	if errResp := s.validateSessionExecutionTriggerRequest(requestID, req); errResp != nil {
		return SessionExecutionTriggerResponse{}, errResp
	}
	session, errResp := s.sessionStateForSendMessage(requestID, req.SessionID)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, errResp
	}
	resp, created, errResp := s.newSessionExecutionTriggerResponse(requestID, req, session)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, errResp
	}
	if created {
		s.auditSessionExecutionTrigger(requestID, req, resp)
	}
	return resp, nil
}

func (s *Service) prepareSessionExecutionTriggerRequest(reqID, fallbackReqID string, admissionErr error, req SessionExecutionTriggerRequest) (string, *ErrorResponse) {
	requestID := strings.TrimSpace(resolveRequestID(reqID, fallbackReqID))
	if admissionErr != nil {
		errID := requestID
		if errID == "" {
			errID = defaultRequestIDFallback
		}
		err := s.makeError(errID, "broker_api_auth_admission_denied", "auth", false, admissionErr.Error())
		return "", &err
	}
	if requestID == "" {
		err := s.makeError(defaultRequestIDFallback, "broker_validation_request_id_missing", "validation", false, "request_id is required")
		return "", &err
	}
	if err := s.validateRequest(req, sessionExecutionTriggerRequestSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return "", &errOut
	}
	return requestID, nil
}

func (s *Service) newSessionExecutionTriggerResponse(requestID string, req SessionExecutionTriggerRequest, session artifacts.SessionDurableState) (SessionExecutionTriggerResponse, bool, *ErrorResponse) {
	if req.RequestedOperation == "continue" {
		return s.continueSessionTurnExecution(requestID, req, session)
	}
	appendReq, errResp := s.buildSessionExecutionAppendRequest(requestID, req, session)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, false, errResp
	}
	appendResult, errResp := s.appendSessionExecutionTriggerResult(requestID, appendReq)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, false, errResp
	}
	resp := buildSessionExecutionTriggerAckResponse(requestID, req.SessionID, appendResult.Trigger, appendResult.TurnExecution, appendResult.Seq)
	if err := s.validateResponse(resp, sessionExecutionTriggerResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionExecutionTriggerResponse{}, false, &errOut
	}
	return resp, appendResult.Created, nil
}

func (s *Service) buildSessionExecutionAppendRequest(requestID string, req SessionExecutionTriggerRequest, session artifacts.SessionDurableState) (artifacts.SessionExecutionTriggerAppendRequest, *ErrorResponse) {
	project, errResp := s.requireSupportedProjectSubstrateForSessionExecution(requestID)
	if errResp != nil {
		return artifacts.SessionExecutionTriggerAppendRequest{}, errResp
	}
	approvalProfile := normalizeSessionTriggerApprovalProfile(req.ApprovalProfile)
	autonomyPosture := normalizeSessionTriggerAutonomyPosture(req.AutonomyPosture)
	occurredAt := s.currentTimestamp()
	idempotencyHash, err := artifacts.SessionExecutionTriggerIdempotencyHash(req.SessionID, req.TriggerSource, req.RequestedOperation, approvalProfile, autonomyPosture, req.UserMessageContentText)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return artifacts.SessionExecutionTriggerAppendRequest{}, &errOut
	}
	links := sessionExecutionLinksFromSessionState(session)
	boundDigest := strings.TrimSpace(project.Snapshot.ValidatedSnapshotDigest)
	if boundDigest == "" {
		boundDigest = strings.TrimSpace(project.Snapshot.ProjectContextIdentityDigest)
	}
	return artifacts.SessionExecutionTriggerAppendRequest{
		SessionID:                            req.SessionID,
		WorkspaceID:                          session.WorkspaceID,
		CreatedByRunID:                       session.CreatedByRunID,
		TriggerSource:                        req.TriggerSource,
		RequestedOperation:                   req.RequestedOperation,
		ApprovalProfile:                      approvalProfile,
		AutonomyPosture:                      autonomyPosture,
		PrimaryRunID:                         strings.TrimSpace(session.CreatedByRunID),
		LinkedRunIDs:                         links.runIDs,
		LinkedApprovalIDs:                    links.approvalIDs,
		LinkedArtifactDigests:                links.artifactDigests,
		LinkedAuditRecordDigests:             links.auditRecordDigests,
		BoundValidatedProjectSubstrateDigest: boundDigest,
		ExecutionState:                       "running",
		UserMessageContentText:               strings.TrimSpace(req.UserMessageContentText),
		IdempotencyKey:                       strings.TrimSpace(req.IdempotencyKey),
		IdempotencyHash:                      idempotencyHash,
		OccurredAt:                           occurredAt,
	}, nil
}

func (s *Service) appendSessionExecutionTriggerResult(requestID string, appendReq artifacts.SessionExecutionTriggerAppendRequest) (artifacts.SessionExecutionTriggerAppendResult, *ErrorResponse) {
	appendResult, err := s.AppendSessionExecutionTrigger(appendReq)
	if err != nil {
		if errors.Is(err, artifacts.ErrSessionActiveTurnExecutionExists) {
			errOut := s.makeError(requestID, "broker_session_execution_active_turn_exists", "policy", false, err.Error())
			return artifacts.SessionExecutionTriggerAppendResult{}, &errOut
		}
		if errors.Is(err, artifacts.ErrSessionExecutionTriggerIdempotencyKeyConflict) {
			errOut := s.makeError(requestID, "broker_idempotency_key_payload_mismatch", "validation", false, err.Error())
			return artifacts.SessionExecutionTriggerAppendResult{}, &errOut
		}
		errOut := s.errorFromStore(requestID, err)
		return artifacts.SessionExecutionTriggerAppendResult{}, &errOut
	}
	return appendResult, nil
}

func (s *Service) continueSessionTurnExecution(requestID string, req SessionExecutionTriggerRequest, session artifacts.SessionDurableState) (SessionExecutionTriggerResponse, bool, *ErrorResponse) {
	target, ok := currentOrResumableTurnExecution(session.TurnExecutions)
	if !ok {
		errOut := s.makeError(requestID, "broker_session_execution_continue_missing_execution", "policy", false, artifacts.ErrSessionTurnExecutionNotResumable.Error())
		return SessionExecutionTriggerResponse{}, false, &errOut
	}
	currentDigest, errResp := s.requireCurrentSessionExecutionDigest(requestID, session, target)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, false, errResp
	}
	updated, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{
		SessionID:                            session.SessionID,
		TurnID:                               target.TurnID,
		ExecutionState:                       "running",
		WaitKind:                             "",
		WaitState:                            "",
		BlockedReasonCode:                    "",
		BoundValidatedProjectSubstrateDigest: currentDigest,
		OccurredAt:                           s.currentTimestamp(),
	})
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return SessionExecutionTriggerResponse{}, false, &errOut
	}
	resp := newContinuedSessionExecutionTriggerResponse(requestID, req, updated)
	if err := s.validateResponse(resp, sessionExecutionTriggerResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionExecutionTriggerResponse{}, false, &errOut
	}
	return resp, false, nil
}

func currentOrResumableTurnExecution(executions []artifacts.SessionTurnExecutionDurableState) (artifacts.SessionTurnExecutionDurableState, bool) {
	for idx := len(executions) - 1; idx >= 0; idx-- {
		exec := executions[idx]
		if isResumableSessionTurnExecutionState(exec.ExecutionState) {
			return exec, true
		}
	}
	return artifacts.SessionTurnExecutionDurableState{}, false
}

func isResumableSessionTurnExecutionState(state string) bool {
	switch strings.TrimSpace(state) {
	case "queued", "planning", "running", "waiting":
		return true
	default:
		return false
	}
}

func (s *Service) markTurnExecutionProjectBlocked(sessionID string, execution artifacts.SessionTurnExecutionDurableState, reason string, occurredAt time.Time) error {
	blockedReason := strings.TrimSpace(reason)
	if blockedReason == "" {
		blockedReason = sessionExecutionBlockedReasonProjectSubstratePosture
	}
	_, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{
		SessionID:                            sessionID,
		TurnID:                               execution.TurnID,
		ExecutionState:                       "blocked",
		WaitKind:                             "project_blocked",
		WaitState:                            "waiting_project_blocked",
		BlockedReasonCode:                    blockedReason,
		BoundValidatedProjectSubstrateDigest: strings.TrimSpace(execution.BoundValidatedProjectSubstrateDigest),
		OccurredAt:                           occurredAt,
	})
	return err
}

func (s *Service) auditSessionExecutionTrigger(requestID string, req SessionExecutionTriggerRequest, resp SessionExecutionTriggerResponse) {
	_ = s.AppendTrustedAuditEvent("session_execution_trigger_submitted", "brokerapi", map[string]interface{}{
		"session_id":          req.SessionID,
		"trigger_id":          resp.TriggerID,
		"turn_id":             resp.TurnID,
		"stream_id":           resp.StreamID,
		"seq":                 resp.Seq,
		"request_id":          requestID,
		"trigger_source":      req.TriggerSource,
		"approval_profile":    resp.ApprovalProfile,
		"autonomy_posture":    resp.AutonomyPosture,
		"execution_state":     resp.ExecutionState,
		"requested_operation": req.RequestedOperation,
	})
}

func validateSessionSendMessageRoleForTranscriptOnly(role string) error {
	if role != "user" && role != "assistant" && role != "system" && role != "tool" {
		return fmt.Errorf("role is invalid")
	}
	return nil
}
