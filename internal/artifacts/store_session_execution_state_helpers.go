package artifacts

import (
	"fmt"
	"strings"
	"time"
)

func nextSessionExecutionIDs(session *SessionDurableState) (int, string, string) {
	triggerIndex := len(session.ExecutionTriggers) + 1
	triggerID := fmt.Sprintf("%s.trigger.%06d", session.SessionID, triggerIndex)
	turnID := fmt.Sprintf("%s.exec.%06d", session.SessionID, triggerIndex)
	return triggerIndex, triggerID, turnID
}

func newSessionExecutionTriggerState(sessionID, triggerID string, triggerIndex int, req SessionExecutionTriggerAppendRequest) SessionExecutionTriggerDurableState {
	return SessionExecutionTriggerDurableState{
		TriggerID:              triggerID,
		SessionID:              sessionID,
		TriggerIndex:           triggerIndex,
		TriggerSource:          req.TriggerSource,
		RequestedOperation:     req.RequestedOperation,
		UserMessageContentText: req.UserMessageContentText,
		CreatedAt:              req.OccurredAt,
	}
}

func newSessionTurnExecutionState(sessionID, turnID, triggerID string, triggerIndex int, req SessionExecutionTriggerAppendRequest) SessionTurnExecutionDurableState {
	return SessionTurnExecutionDurableState{
		TurnID:                               turnID,
		SessionID:                            sessionID,
		ExecutionIndex:                       triggerIndex,
		TriggerID:                            triggerID,
		TriggerSource:                        req.TriggerSource,
		RequestedOperation:                   req.RequestedOperation,
		ExecutionState:                       req.ExecutionState,
		WaitKind:                             req.WaitKind,
		WaitState:                            req.WaitState,
		ApprovalProfile:                      req.ApprovalProfile,
		AutonomyPosture:                      req.AutonomyPosture,
		PrimaryRunID:                         req.PrimaryRunID,
		PendingApprovalID:                    req.PendingApprovalID,
		LinkedRunIDs:                         append([]string{}, req.LinkedRunIDs...),
		LinkedApprovalIDs:                    append([]string{}, req.LinkedApprovalIDs...),
		LinkedArtifactDigests:                append([]string{}, req.LinkedArtifactDigests...),
		LinkedAuditRecordDigests:             append([]string{}, req.LinkedAuditRecordDigests...),
		BoundValidatedProjectSubstrateDigest: req.BoundValidatedProjectSubstrateDigest,
		BlockedReasonCode:                    req.BlockedReasonCode,
		TerminalOutcome:                      req.TerminalOutcome,
		CreatedAt:                            req.OccurredAt,
		UpdatedAt:                            req.OccurredAt,
	}
}

func mergeSessionExecutionLinkedRunIDs(session *SessionDurableState, req SessionExecutionTriggerAppendRequest) {
	session.LinkedRunIDs = mergeSessionLinkedRunIDs(session.LinkedRunIDs, req.LinkedRunIDs)
	if req.PrimaryRunID != "" {
		session.LinkedRunIDs = mergeSessionLinkedRunIDs(session.LinkedRunIDs, []string{req.PrimaryRunID})
	}
}

func applySessionExecutionSummaryUpdate(session *SessionDurableState, exec SessionTurnExecutionDurableState, occurredAt time.Time) {
	session.UpdatedAt = occurredAt
	session.LastActivityAt = occurredAt
	session.LastActivityKind = "execution_trigger_submitted"
	session.LastActivityPreview = previewSessionTurnExecutionState(exec)
	session.HasIncompleteTurn = false
	session.Status = "active"
	session.WorkPosture = sessionWorkPostureFromExecutionState(exec.ExecutionState)
	session.WorkPostureReason = exec.BlockedReasonCode
}

func sessionTurnExecutionIndex(executions []SessionTurnExecutionDurableState, turnID string) int {
	for i := range executions {
		if executions[i].TurnID == turnID {
			return i
		}
	}
	return -1
}

func applySessionTurnExecutionUpdate(exec SessionTurnExecutionDurableState, req SessionTurnExecutionUpdateRequest) SessionTurnExecutionDurableState {
	exec.ExecutionState = req.ExecutionState
	exec.WaitKind = req.WaitKind
	exec.WaitState = req.WaitState
	if req.PrimaryRunID != "" {
		exec.PrimaryRunID = req.PrimaryRunID
	}
	if req.PendingApprovalID != "" {
		exec.PendingApprovalID = req.PendingApprovalID
	}
	if len(req.LinkedRunIDs) > 0 {
		exec.LinkedRunIDs = append([]string{}, req.LinkedRunIDs...)
	}
	if len(req.LinkedApprovalIDs) > 0 {
		exec.LinkedApprovalIDs = append([]string{}, req.LinkedApprovalIDs...)
	}
	if len(req.LinkedArtifactDigests) > 0 {
		exec.LinkedArtifactDigests = append([]string{}, req.LinkedArtifactDigests...)
	}
	if len(req.LinkedAuditRecordDigests) > 0 {
		exec.LinkedAuditRecordDigests = append([]string{}, req.LinkedAuditRecordDigests...)
	}
	exec.BlockedReasonCode = req.BlockedReasonCode
	exec.TerminalOutcome = req.TerminalOutcome
	if req.BoundValidatedProjectSubstrateDigest != "" {
		exec.BoundValidatedProjectSubstrateDigest = req.BoundValidatedProjectSubstrateDigest
	}
	exec.UpdatedAt = req.OccurredAt
	return exec
}

func hasActiveSessionTurnExecution(session SessionDurableState) bool {
	for _, exec := range session.TurnExecutions {
		if isSessionTurnExecutionActive(exec.ExecutionState) {
			return true
		}
	}
	return false
}

func isSessionTurnExecutionActive(state string) bool {
	switch strings.TrimSpace(state) {
	case "queued", "planning", "running", "waiting", "blocked":
		return true
	default:
		return false
	}
}

func sessionWorkPostureFromExecutionState(executionState string) string {
	switch executionState {
	case "waiting":
		return "waiting"
	case "blocked":
		return "blocked"
	case "failed":
		return "failed"
	case "completed":
		return "idle"
	default:
		return "running"
	}
}

func previewSessionTurnExecutionState(execution SessionTurnExecutionDurableState) string {
	if strings.TrimSpace(execution.BlockedReasonCode) != "" {
		return execution.ExecutionState + ":" + execution.BlockedReasonCode
	}
	if strings.TrimSpace(execution.WaitState) != "" {
		return execution.WaitState
	}
	return execution.ExecutionState
}

func previewSessionExecutionTrigger(req SessionExecutionTriggerAppendRequest) string {
	if req.UserMessageContentText != "" {
		return req.UserMessageContentText
	}
	return req.RequestedOperation
}
