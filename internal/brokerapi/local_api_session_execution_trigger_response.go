package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func buildSessionExecutionTriggerAckResponse(requestID, sessionID string, trigger artifacts.SessionExecutionTriggerDurableState, execution artifacts.SessionTurnExecutionDurableState, seq int64) SessionExecutionTriggerResponse {
	return SessionExecutionTriggerResponse{
		SchemaID:               "runecode.protocol.v0.SessionExecutionTriggerResponse",
		SchemaVersion:          "0.1.0",
		RequestID:              requestID,
		SessionID:              sessionID,
		TriggerID:              trigger.TriggerID,
		TurnID:                 execution.TurnID,
		TriggerSource:          trigger.TriggerSource,
		RequestedOperation:     trigger.RequestedOperation,
		ApprovalProfile:        execution.ApprovalProfile,
		AutonomyPosture:        execution.AutonomyPosture,
		ExecutionState:         execution.ExecutionState,
		UserMessageContentText: trigger.UserMessageContentText,
		EventType:              "session_execution_trigger_ack",
		StreamID:               sessionInteractionStreamID(sessionID),
		Seq:                    seq,
	}
}

func newContinuedSessionExecutionTriggerResponse(requestID string, req SessionExecutionTriggerRequest, updated artifacts.SessionTurnExecutionDurableState, seq int64) SessionExecutionTriggerResponse {
	return SessionExecutionTriggerResponse{
		SchemaID:               "runecode.protocol.v0.SessionExecutionTriggerResponse",
		SchemaVersion:          "0.1.0",
		RequestID:              requestID,
		SessionID:              req.SessionID,
		TriggerID:              updated.TriggerID,
		TurnID:                 updated.TurnID,
		TriggerSource:          req.TriggerSource,
		RequestedOperation:     req.RequestedOperation,
		ApprovalProfile:        updated.ApprovalProfile,
		AutonomyPosture:        updated.AutonomyPosture,
		ExecutionState:         updated.ExecutionState,
		UserMessageContentText: strings.TrimSpace(req.UserMessageContentText),
		EventType:              "session_execution_trigger_ack",
		StreamID:               sessionInteractionStreamID(req.SessionID),
		Seq:                    seq,
	}
}
