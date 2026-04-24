package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) reconcileSessionExecutionTriggerSideEffects(requestID string, session artifacts.SessionDurableState, req SessionExecutionTriggerRequest, resp SessionExecutionTriggerResponse) error {
	if req.RequestedOperation == "start" {
		s.auditSessionExecutionTrigger(requestID, req, resp)
	}
	triggerSession, ok := s.SessionState(req.SessionID)
	if !ok {
		return fmt.Errorf("session %q not found", req.SessionID)
	}
	result, ok := sessionExecutionTriggerAppendResultForID(triggerSession, resp.TriggerID)
	if !ok {
		return fmt.Errorf("session execution trigger %q not found", resp.TriggerID)
	}
	runID := strings.TrimSpace(result.TurnExecution.PrimaryRunID)
	if runID == "" {
		runID = strings.TrimSpace(triggerSession.CreatedByRunID)
	}
	if req.RequestedOperation == "start" {
		if err := s.appendSessionExecutionStartCheckpoint(req.SessionID, resp.TriggerID, runID, req.UserMessageContentText); err != nil {
			return err
		}
	}
	if err := s.bridgeSessionExecutionTriggerToRun(runID, result); err != nil {
		return err
	}
	return nil
}

func (s *Service) nextSessionInteractionSequence(requestID, sessionID string) (int64, *ErrorResponse) {
	updated, err := s.UpdateSessionState(sessionID, func(state artifacts.SessionDurableState) artifacts.SessionDurableState {
		state.LastInteractionSequence++
		return state
	})
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return 0, &errOut
	}
	return updated.LastInteractionSequence, nil
}

func sessionExecutionTriggerAppendResultForID(session artifacts.SessionDurableState, triggerID string) (artifacts.SessionExecutionTriggerAppendResult, bool) {
	triggerID = strings.TrimSpace(triggerID)
	if triggerID == "" {
		return artifacts.SessionExecutionTriggerAppendResult{}, false
	}
	for _, trigger := range session.ExecutionTriggers {
		if trigger.TriggerID != triggerID {
			continue
		}
		for _, execution := range session.TurnExecutions {
			if execution.TriggerID != triggerID {
				continue
			}
			return artifacts.SessionExecutionTriggerAppendResult{Created: false, Trigger: trigger, TurnExecution: execution, Seq: sessionInteractionSequenceForTrigger(session, triggerID)}, true
		}
	}
	return artifacts.SessionExecutionTriggerAppendResult{}, false
}

func sessionInteractionSequenceForTrigger(session artifacts.SessionDurableState, triggerID string) int64 {
	for _, record := range session.ExecutionTriggerIdempotencyByKey {
		if record.TriggerID == triggerID {
			return record.Seq
		}
	}
	return 0
}
