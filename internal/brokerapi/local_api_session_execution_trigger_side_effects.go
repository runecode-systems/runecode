package brokerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) reconcileSessionExecutionTriggerSideEffects(ctx context.Context, requestID string, session artifacts.SessionDurableState, req SessionExecutionTriggerRequest, resp SessionExecutionTriggerResponse) error {
	if req.RequestedOperation == "start" {
		s.auditSessionExecutionTrigger(requestID, req, resp)
	}
	result, runID, err := s.loadSessionExecutionTriggerResult(req.SessionID, resp.TriggerID)
	if err != nil {
		return err
	}
	if err := s.appendSessionExecutionStartCheckpointIfNeeded(req, resp.TriggerID, runID); err != nil {
		return err
	}
	if !shouldReconcileStartedExecution(req, result) {
		return nil
	}
	authority, err := s.ensureSessionExecutionRunPlanAuthority(result)
	if err != nil {
		return err
	}
	result, err = s.applySessionExecutionWorkflowSideEffects(req.SessionID, result, authority)
	if err != nil {
		return err
	}
	return s.bridgeSessionExecutionTriggerToRun(ctx, requestID, result, authority)
}

func (s *Service) loadSessionExecutionTriggerResult(sessionID, triggerID string) (artifacts.SessionExecutionTriggerAppendResult, string, error) {
	triggerSession, ok := s.SessionState(sessionID)
	if !ok {
		return artifacts.SessionExecutionTriggerAppendResult{}, "", fmt.Errorf("session %q not found", sessionID)
	}
	result, ok := sessionExecutionTriggerAppendResultForID(triggerSession, triggerID)
	if !ok {
		return artifacts.SessionExecutionTriggerAppendResult{}, "", fmt.Errorf("session execution trigger %q not found", triggerID)
	}
	runID := strings.TrimSpace(result.TurnExecution.PrimaryRunID)
	if runID == "" {
		runID = strings.TrimSpace(triggerSession.CreatedByRunID)
	}
	return result, runID, nil
}

func (s *Service) appendSessionExecutionStartCheckpointIfNeeded(req SessionExecutionTriggerRequest, triggerID, runID string) error {
	if req.RequestedOperation != "start" {
		return nil
	}
	return s.appendSessionExecutionStartCheckpoint(req.SessionID, triggerID, runID, req.UserMessageContentText)
}

func shouldReconcileStartedExecution(req SessionExecutionTriggerRequest, result artifacts.SessionExecutionTriggerAppendResult) bool {
	return req.RequestedOperation == "start" && strings.TrimSpace(result.TurnExecution.ExecutionState) == "running"
}

func (s *Service) applySessionExecutionWorkflowSideEffects(sessionID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) (artifacts.SessionExecutionTriggerAppendResult, error) {
	var err error
	result, err = s.applySessionExecutionMutationBearingSideEffects(sessionID, result, authority)
	if err != nil {
		return artifacts.SessionExecutionTriggerAppendResult{}, err
	}
	return s.applySessionExecutionDraftArtifactSideEffects(sessionID, result, authority)
}

func (s *Service) applySessionExecutionMutationBearingSideEffects(sessionID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) (artifacts.SessionExecutionTriggerAppendResult, error) {
	var err error
	result, err = s.applySessionExecutionDraftPromoteSideEffects(sessionID, result, authority)
	if err != nil {
		return artifacts.SessionExecutionTriggerAppendResult{}, err
	}
	return s.applySessionExecutionApprovedImplementationSideEffects(sessionID, result, authority)
}

func (s *Service) applySessionExecutionDraftPromoteSideEffects(sessionID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) (artifacts.SessionExecutionTriggerAppendResult, error) {
	if strings.TrimSpace(authority.workflowOperation) != sessionWorkflowOperationDraftPromoteApply || len(result.TurnExecution.WorkflowRouting.BoundInputArtifacts) == 0 {
		return result, nil
	}
	approvalID, err := s.applySessionExecutionDraftPromote(result, authority)
	if err != nil {
		return artifacts.SessionExecutionTriggerAppendResult{}, err
	}
	if strings.TrimSpace(approvalID) == "" {
		return result, nil
	}
	return s.updateSessionExecutionLinkedApprovals(sessionID, result.TurnExecution.TurnID, append(result.TurnExecution.LinkedApprovalIDs, approvalID))
}

func (s *Service) applySessionExecutionApprovedImplementationSideEffects(sessionID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) (artifacts.SessionExecutionTriggerAppendResult, error) {
	if strings.TrimSpace(authority.workflowOperation) != sessionWorkflowOperationApprovedImplementation {
		return result, nil
	}
	linkedApprovalIDs, linkedArtifactDigests, err := s.applySessionExecutionApprovedImplementation(result, authority)
	if err != nil {
		return artifacts.SessionExecutionTriggerAppendResult{}, err
	}
	if len(linkedApprovalIDs) > 0 {
		result, err = s.updateSessionExecutionLinkedApprovals(sessionID, result.TurnExecution.TurnID, append(result.TurnExecution.LinkedApprovalIDs, linkedApprovalIDs...))
		if err != nil {
			return artifacts.SessionExecutionTriggerAppendResult{}, err
		}
	}
	if len(linkedArtifactDigests) == 0 {
		return result, nil
	}
	return s.updateSessionExecutionLinkedArtifacts(sessionID, result.TurnExecution.TurnID, append(result.TurnExecution.LinkedArtifactDigests, linkedArtifactDigests...))
}

func (s *Service) applySessionExecutionDraftArtifactSideEffects(sessionID string, result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) (artifacts.SessionExecutionTriggerAppendResult, error) {
	linkedArtifactDigests, err := s.materializeSessionExecutionDraftArtifacts(result, authority)
	if err != nil {
		return artifacts.SessionExecutionTriggerAppendResult{}, err
	}
	if len(linkedArtifactDigests) > 0 {
		if result, err = s.updateSessionExecutionLinkedArtifacts(sessionID, result.TurnExecution.TurnID, append(result.TurnExecution.LinkedArtifactDigests, linkedArtifactDigests...)); err != nil {
			return artifacts.SessionExecutionTriggerAppendResult{}, err
		}
	}
	return result, nil
}

func (s *Service) updateSessionExecutionLinkedArtifacts(sessionID, turnID string, digests []string) (artifacts.SessionExecutionTriggerAppendResult, error) {
	updated, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{
		SessionID:             sessionID,
		TurnID:                turnID,
		ExecutionState:        "running",
		LinkedArtifactDigests: uniqueSortedStrings(digests),
		OccurredAt:            s.currentTimestamp(),
	})
	if err != nil {
		return artifacts.SessionExecutionTriggerAppendResult{}, err
	}
	session, ok := s.SessionState(sessionID)
	if !ok {
		return artifacts.SessionExecutionTriggerAppendResult{}, fmt.Errorf("session %q not found", sessionID)
	}
	result, ok := sessionExecutionTriggerAppendResultForID(session, updated.TriggerID)
	if !ok {
		return artifacts.SessionExecutionTriggerAppendResult{}, fmt.Errorf("session execution trigger for turn %q not found", turnID)
	}
	result.TurnExecution = updated
	return result, nil
}

func (s *Service) updateSessionExecutionLinkedApprovals(sessionID, turnID string, approvalIDs []string) (artifacts.SessionExecutionTriggerAppendResult, error) {
	updated, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{
		SessionID:         sessionID,
		TurnID:            turnID,
		ExecutionState:    "running",
		PendingApprovalID: "",
		LinkedApprovalIDs: uniqueSortedStrings(approvalIDs),
		OccurredAt:        s.currentTimestamp(),
	})
	if err != nil {
		return artifacts.SessionExecutionTriggerAppendResult{}, err
	}
	session, ok := s.SessionState(sessionID)
	if !ok {
		return artifacts.SessionExecutionTriggerAppendResult{}, fmt.Errorf("session %q not found", sessionID)
	}
	result, ok := sessionExecutionTriggerAppendResultForID(session, updated.TriggerID)
	if !ok {
		return artifacts.SessionExecutionTriggerAppendResult{}, fmt.Errorf("session execution trigger for turn %q not found", turnID)
	}
	result.TurnExecution = updated
	return result, nil
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
