package brokerapi

import (
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func newSessionDetail(summary SessionSummary, projectedTurns []SessionTranscriptTurn, runs, approvals, artifactsByDigest, auditRecordDigests map[string]struct{}) SessionDetail {
	return SessionDetail{
		SchemaID:                 "runecode.protocol.v0.SessionDetail",
		SchemaVersion:            "0.1.0",
		Summary:                  summary,
		TranscriptTurns:          projectedTurns,
		CurrentTurnExecution:     nil,
		LatestTurnExecution:      nil,
		LinkedRunIDs:             boundedSortedKeys(runs, 256),
		LinkedApprovalIDs:        boundedSortedKeys(approvals, 512),
		LinkedArtifactDigests:    boundedSortedKeys(artifactsByDigest, 1024),
		LinkedAuditRecordDigests: boundedSortedKeys(auditRecordDigests, 1024),
	}
}

func buildSessionTurnExecutionFromDurable(in artifacts.SessionTurnExecutionDurableState) SessionTurnExecution {
	return SessionTurnExecution{
		SchemaID:                             "runecode.protocol.v0.SessionTurnExecution",
		SchemaVersion:                        "0.1.0",
		TurnID:                               in.TurnID,
		SessionID:                            in.SessionID,
		ExecutionIndex:                       in.ExecutionIndex,
		TriggerID:                            in.TriggerID,
		TriggerSource:                        in.TriggerSource,
		RequestedOperation:                   in.RequestedOperation,
		ExecutionState:                       in.ExecutionState,
		WaitKind:                             in.WaitKind,
		WaitState:                            in.WaitState,
		ApprovalProfile:                      in.ApprovalProfile,
		AutonomyPosture:                      in.AutonomyPosture,
		PrimaryRunID:                         in.PrimaryRunID,
		PendingApprovalID:                    in.PendingApprovalID,
		LinkedRunIDs:                         boundedStrings(append([]string{}, in.LinkedRunIDs...), 256),
		LinkedApprovalIDs:                    boundedStrings(append([]string{}, in.LinkedApprovalIDs...), 512),
		LinkedArtifactDigests:                boundedStrings(append([]string{}, in.LinkedArtifactDigests...), 1024),
		LinkedAuditRecordDigests:             boundedStrings(append([]string{}, in.LinkedAuditRecordDigests...), 1024),
		BoundValidatedProjectSubstrateDigest: in.BoundValidatedProjectSubstrateDigest,
		BlockedReasonCode:                    in.BlockedReasonCode,
		TerminalOutcome:                      in.TerminalOutcome,
		CreatedAt:                            in.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                            in.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func currentAndLatestSessionTurnExecution(executions []artifacts.SessionTurnExecutionDurableState) (*SessionTurnExecution, *SessionTurnExecution) {
	if len(executions) == 0 {
		return nil, nil
	}
	latest := buildSessionTurnExecutionFromDurable(executions[len(executions)-1])
	current := currentSessionTurnExecution(executions)
	return current, &latest
}

func currentSessionTurnExecution(executions []artifacts.SessionTurnExecutionDurableState) *SessionTurnExecution {
	for idx := len(executions) - 1; idx >= 0; idx-- {
		if !isSessionTurnExecutionActiveState(executions[idx].ExecutionState) {
			continue
		}
		active := buildSessionTurnExecutionFromDurable(executions[idx])
		return &active
	}
	return nil
}

func isSessionTurnExecutionActiveState(state string) bool {
	switch state {
	case "queued", "planning", "running", "waiting", "blocked":
		return true
	default:
		return false
	}
}
