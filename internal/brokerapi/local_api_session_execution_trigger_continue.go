package brokerapi

import (
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) continueSessionTurnExecution(requestID string, req SessionExecutionTriggerRequest, session artifacts.SessionDurableState) (SessionExecutionTriggerResponse, bool, *ErrorResponse) {
	if replay, ok, errResp := s.replayContinuedSessionExecution(requestID, req, session); ok || errResp != nil {
		return replay, false, errResp
	}
	target, errResp := s.selectSessionExecutionContinueTarget(requestID, req, session)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, false, errResp
	}
	seq, updated, errResp := s.resumeSessionExecutionTarget(requestID, session, target)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, false, errResp
	}
	resp := newContinuedSessionExecutionTriggerResponse(requestID, req, updated, seq)
	if errResp := s.validateContinuedSessionExecutionResponse(requestID, req, resp, updated.TurnID, seq); errResp != nil {
		return SessionExecutionTriggerResponse{}, false, errResp
	}
	return resp, false, nil
}

func (s *Service) selectSessionExecutionContinueTarget(requestID string, req SessionExecutionTriggerRequest, session artifacts.SessionDurableState) (artifacts.SessionTurnExecutionDurableState, *ErrorResponse) {
	target, ok := currentOrResumableTurnExecution(session.TurnExecutions, strings.TrimSpace(req.TurnID))
	if !ok {
		errOut := s.makeError(requestID, "broker_session_execution_continue_missing_execution", "policy", false, artifacts.ErrSessionTurnExecutionNotResumable.Error())
		return artifacts.SessionTurnExecutionDurableState{}, &errOut
	}
	if errResp := s.validateSessionExecutionContinueTarget(requestID, target); errResp != nil {
		return artifacts.SessionTurnExecutionDurableState{}, errResp
	}
	return target, nil
}

func (s *Service) resumeSessionExecutionTarget(requestID string, session artifacts.SessionDurableState, target artifacts.SessionTurnExecutionDurableState) (int64, artifacts.SessionTurnExecutionDurableState, *ErrorResponse) {
	seq, errResp := s.nextSessionInteractionSequence(requestID, session.SessionID)
	if errResp != nil {
		return 0, artifacts.SessionTurnExecutionDurableState{}, errResp
	}
	currentDigest, errResp := s.requireCurrentSessionExecutionDigest(requestID, session, target)
	if errResp != nil {
		return 0, artifacts.SessionTurnExecutionDurableState{}, errResp
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
		return 0, artifacts.SessionTurnExecutionDurableState{}, &errOut
	}
	return seq, updated, nil
}

func (s *Service) validateContinuedSessionExecutionResponse(requestID string, req SessionExecutionTriggerRequest, resp SessionExecutionTriggerResponse, turnID string, seq int64) *ErrorResponse {
	if err := s.validateResponse(resp, sessionExecutionTriggerResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return &errOut
	}
	if errResp := s.storeContinuedSessionExecutionReplay(requestID, req, turnID, seq); errResp != nil {
		return errResp
	}
	return nil
}

func (s *Service) replayContinuedSessionExecution(requestID string, req SessionExecutionTriggerRequest, session artifacts.SessionDurableState) (SessionExecutionTriggerResponse, bool, *ErrorResponse) {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" {
		return SessionExecutionTriggerResponse{}, false, nil
	}
	hash, err := artifacts.SessionExecutionTriggerIdempotencyHash(req.SessionID, req.TriggerSource, req.RequestedOperation, normalizeSessionTriggerApprovalProfile(req.ApprovalProfile), normalizeSessionTriggerAutonomyPosture(req.AutonomyPosture), req.UserMessageContentText)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return SessionExecutionTriggerResponse{}, false, &errOut
	}
	record, ok := session.ExecutionTriggerIdempotencyByKey[key]
	if !ok {
		return SessionExecutionTriggerResponse{}, false, nil
	}
	if record.RequestHash != hash {
		errOut := s.makeError(requestID, "broker_idempotency_key_payload_mismatch", "validation", false, artifacts.ErrSessionExecutionTriggerIdempotencyKeyConflict.Error())
		return SessionExecutionTriggerResponse{}, false, &errOut
	}
	for _, execution := range session.TurnExecutions {
		if strings.TrimSpace(execution.TurnID) != strings.TrimSpace(req.TurnID) {
			continue
		}
		resp := newContinuedSessionExecutionTriggerResponse(requestID, req, execution, record.Seq)
		if err := s.validateResponse(resp, sessionExecutionTriggerResponseSchemaPath); err != nil {
			errOut := s.errorFromValidation(requestID, err)
			return SessionExecutionTriggerResponse{}, false, &errOut
		}
		return resp, true, nil
	}
	errOut := s.makeError(requestID, "broker_session_execution_continue_missing_execution", "policy", false, artifacts.ErrSessionTurnExecutionNotResumable.Error())
	return SessionExecutionTriggerResponse{}, false, &errOut
}

func (s *Service) storeContinuedSessionExecutionReplay(requestID string, req SessionExecutionTriggerRequest, turnID string, seq int64) *ErrorResponse {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" {
		return nil
	}
	hash, err := artifacts.SessionExecutionTriggerIdempotencyHash(req.SessionID, req.TriggerSource, req.RequestedOperation, normalizeSessionTriggerApprovalProfile(req.ApprovalProfile), normalizeSessionTriggerAutonomyPosture(req.AutonomyPosture), req.UserMessageContentText)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	_, err = s.UpdateSessionState(req.SessionID, func(state artifacts.SessionDurableState) artifacts.SessionDurableState {
		if state.ExecutionTriggerIdempotencyByKey == nil {
			state.ExecutionTriggerIdempotencyByKey = map[string]artifacts.SessionExecutionTriggerIdempotencyRecord{}
		}
		state.ExecutionTriggerIdempotencyByKey[key] = artifacts.SessionExecutionTriggerIdempotencyRecord{
			RequestHash: hash,
			TriggerID:   turnID,
			Seq:         seq,
		}
		return state
	})
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return &errOut
	}
	return nil
}

func (s *Service) validateSessionExecutionContinueTarget(requestID string, execution artifacts.SessionTurnExecutionDurableState) *ErrorResponse {
	if strings.TrimSpace(execution.ExecutionState) != "waiting" {
		return nil
	}
	if strings.TrimSpace(execution.WaitKind) != "approval" {
		return nil
	}
	errOut := s.makeError(requestID, "broker_session_execution_continue_waiting_approval", "policy", false, "session turn execution is waiting for approval resolution")
	return &errOut
}

func currentOrResumableTurnExecution(executions []artifacts.SessionTurnExecutionDurableState, turnID string) (artifacts.SessionTurnExecutionDurableState, bool) {
	targetTurnID := strings.TrimSpace(turnID)
	for idx := len(executions) - 1; idx >= 0; idx-- {
		exec := executions[idx]
		if targetTurnID != "" && strings.TrimSpace(exec.TurnID) != targetTurnID {
			continue
		}
		if isResumableSessionTurnExecutionState(exec) {
			return exec, true
		}
	}
	return artifacts.SessionTurnExecutionDurableState{}, false
}

func isResumableSessionTurnExecutionState(execution artifacts.SessionTurnExecutionDurableState) bool {
	switch strings.TrimSpace(execution.ExecutionState) {
	case "queued", "planning", "running", "waiting":
		return true
	case "blocked":
		return strings.TrimSpace(execution.WaitKind) == "project_blocked"
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
