package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (s *Service) ensureSessionExecutionPrimaryRunBinding(requestID, sessionID string, execution artifacts.SessionTurnExecutionDurableState) (artifacts.SessionTurnExecutionDurableState, *ErrorResponse) {
	if strings.TrimSpace(execution.PrimaryRunID) != "" {
		return execution, nil
	}
	runID := sessionExecutionRunID(sessionID, execution.ExecutionIndex)
	updated, errResp := s.updateSessionExecutionRunBinding(requestID, sessionID, execution, runID)
	if errResp != nil {
		return artifacts.SessionTurnExecutionDurableState{}, errResp
	}
	if errResp := s.updateSessionRunBindingState(requestID, sessionID, runID); errResp != nil {
		return artifacts.SessionTurnExecutionDurableState{}, errResp
	}
	if errResp := s.initializeSessionExecutionRunBinding(requestID, sessionID, runID); errResp != nil {
		return artifacts.SessionTurnExecutionDurableState{}, errResp
	}
	return updated, nil
}

func (s *Service) updateSessionExecutionRunBinding(requestID, sessionID string, execution artifacts.SessionTurnExecutionDurableState, runID string) (artifacts.SessionTurnExecutionDurableState, *ErrorResponse) {
	updated, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{
		SessionID:                            sessionID,
		TurnID:                               execution.TurnID,
		ExecutionState:                       execution.ExecutionState,
		WaitKind:                             execution.WaitKind,
		WaitState:                            execution.WaitState,
		OrchestrationScopeID:                 execution.OrchestrationScopeID,
		DependsOnScopeIDs:                    append([]string{}, execution.DependsOnScopeIDs...),
		PrimaryRunID:                         runID,
		LinkedRunIDs:                         uniqueSortedStrings(append(append([]string{}, execution.LinkedRunIDs...), runID)),
		LinkedApprovalIDs:                    append([]string{}, execution.LinkedApprovalIDs...),
		LinkedArtifactDigests:                append([]string{}, execution.LinkedArtifactDigests...),
		LinkedAuditRecordDigests:             append([]string{}, execution.LinkedAuditRecordDigests...),
		BlockedReasonCode:                    execution.BlockedReasonCode,
		TerminalOutcome:                      execution.TerminalOutcome,
		BoundValidatedProjectSubstrateDigest: execution.BoundValidatedProjectSubstrateDigest,
		OccurredAt:                           s.currentTimestamp(),
	})
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return artifacts.SessionTurnExecutionDurableState{}, &errOut
	}
	return updated, nil
}

func (s *Service) updateSessionRunBindingState(requestID, sessionID, runID string) *ErrorResponse {
	if _, err := s.UpdateSessionState(sessionID, func(state artifacts.SessionDurableState) artifacts.SessionDurableState {
		state.LinkedRunIDs = uniqueSortedStrings(append(state.LinkedRunIDs, runID))
		if strings.TrimSpace(state.CreatedByRunID) == "" {
			state.CreatedByRunID = runID
		}
		return state
	}); err != nil {
		errOut := s.errorFromStore(requestID, err)
		return &errOut
	}
	return nil
}

func (s *Service) initializeSessionExecutionRunBinding(requestID, sessionID, runID string) *ErrorResponse {
	if err := s.SetRunStatus(runID, "active"); err != nil {
		errOut := s.errorFromStore(requestID, err)
		return &errOut
	}
	if err := s.RecordRuntimeFacts(runID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: runID, SessionID: sessionID}}); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	return nil
}

func sessionExecutionRunID(sessionID string, executionIndex int) string {
	if executionIndex < 1 {
		executionIndex = 1
	}
	return fmt.Sprintf("%s.run.%06d", strings.TrimSpace(sessionID), executionIndex)
}
