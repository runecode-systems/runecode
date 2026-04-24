package brokerapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	session, errResp := s.sessionStateForExecutionTrigger(requestID, req)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, errResp
	}
	resp, created, errResp := s.newSessionExecutionTriggerResponse(requestID, req, session)
	if errResp != nil {
		return SessionExecutionTriggerResponse{}, errResp
	}
	if created || req.RequestedOperation == "continue" {
		if err := s.reconcileSessionExecutionTriggerSideEffects(requestID, session, req, resp); err != nil {
			errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
			return SessionExecutionTriggerResponse{}, &errOut
		}
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

func (s *Service) sessionStateForExecutionTrigger(requestID string, req SessionExecutionTriggerRequest) (artifacts.SessionDurableState, *ErrorResponse) {
	sessionID := strings.TrimSpace(req.SessionID)
	session, ok := s.SessionState(sessionID)
	if ok {
		return session, nil
	}
	if strings.TrimSpace(req.RequestedOperation) == "continue" {
		errOut := s.makeError(requestID, "broker_not_found_session", "storage", false, fmt.Sprintf("session %q not found", sessionID))
		return artifacts.SessionDurableState{}, &errOut
	}
	return artifacts.SessionDurableState{SessionID: sessionID, WorkspaceID: "workspace-local"}, nil
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
	if updated, errResp := s.ensureSessionExecutionPrimaryRunBinding(requestID, req.SessionID, appendResult.TurnExecution); errResp != nil {
		return SessionExecutionTriggerResponse{}, false, errResp
	} else {
		appendResult.TurnExecution = updated
	}
	resp := buildSessionExecutionTriggerAckResponse(requestID, req.SessionID, appendResult.Trigger, appendResult.TurnExecution, appendResult.Seq)
	if err := s.validateResponse(resp, sessionExecutionTriggerResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionExecutionTriggerResponse{}, false, &errOut
	}
	return resp, appendResult.Created, nil
}

func (s *Service) appendSessionExecutionTriggerResult(requestID string, appendReq artifacts.SessionExecutionTriggerAppendRequest) (artifacts.SessionExecutionTriggerAppendResult, *ErrorResponse) {
	appendResult, err := s.AppendSessionExecutionTrigger(appendReq)
	if err != nil {
		if errors.Is(err, artifacts.ErrSessionExecutionTriggerIdempotencyKeyConflict) {
			errOut := s.makeError(requestID, "broker_idempotency_key_payload_mismatch", "validation", false, err.Error())
			return artifacts.SessionExecutionTriggerAppendResult{}, &errOut
		}
		errOut := s.errorFromStore(requestID, err)
		return artifacts.SessionExecutionTriggerAppendResult{}, &errOut
	}
	return appendResult, nil
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
