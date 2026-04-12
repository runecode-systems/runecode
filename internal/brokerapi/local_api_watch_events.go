package brokerapi

import (
	"context"
	"errors"
)

func completedRunWatchTerminal(req RunWatchRequest, seq int64) RunWatchEvent {
	return RunWatchEvent{
		SchemaID:       "runecode.protocol.v0.RunWatchEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       req.StreamID,
		RequestID:      req.RequestID,
		Seq:            seq,
		EventType:      "run_watch_terminal",
		Terminal:       true,
		TerminalStatus: "completed",
	}
}

func completedApprovalWatchTerminal(req ApprovalWatchRequest, seq int64) ApprovalWatchEvent {
	return ApprovalWatchEvent{
		SchemaID:       "runecode.protocol.v0.ApprovalWatchEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       req.StreamID,
		RequestID:      req.RequestID,
		Seq:            seq,
		EventType:      "approval_watch_terminal",
		Terminal:       true,
		TerminalStatus: "completed",
	}
}

func completedSessionWatchTerminal(req SessionWatchRequest, seq int64) SessionWatchEvent {
	return SessionWatchEvent{
		SchemaID:       "runecode.protocol.v0.SessionWatchEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       req.StreamID,
		RequestID:      req.RequestID,
		Seq:            seq,
		EventType:      "session_watch_terminal",
		Terminal:       true,
		TerminalStatus: "completed",
	}
}

func runWatchTerminalFromContextErr(streamID string, requestID string, seq int64, ctxErr error) RunWatchEvent {
	terminal := RunWatchEvent{
		SchemaID:      "runecode.protocol.v0.RunWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      streamID,
		RequestID:     requestID,
		Seq:           seq,
		EventType:     "run_watch_terminal",
		Terminal:      true,
		Error: &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "request_cancelled",
			Category:      "transport",
			Retryable:     true,
			Message:       "run watch stream cancelled",
		},
	}
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		terminal.TerminalStatus = "failed"
		terminal.Error.Code = "broker_timeout_request_deadline_exceeded"
		terminal.Error.Category = "timeout"
		terminal.Error.Message = "run watch stream deadline exceeded"
		return terminal
	}
	terminal.TerminalStatus = "cancelled"
	terminal.Error = nil
	return terminal
}

func approvalWatchTerminalFromContextErr(streamID string, requestID string, seq int64, ctxErr error) ApprovalWatchEvent {
	terminal := ApprovalWatchEvent{
		SchemaID:      "runecode.protocol.v0.ApprovalWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      streamID,
		RequestID:     requestID,
		Seq:           seq,
		EventType:     "approval_watch_terminal",
		Terminal:      true,
		Error: &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "request_cancelled",
			Category:      "transport",
			Retryable:     true,
			Message:       "approval watch stream cancelled",
		},
	}
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		terminal.TerminalStatus = "failed"
		terminal.Error.Code = "broker_timeout_request_deadline_exceeded"
		terminal.Error.Category = "timeout"
		terminal.Error.Message = "approval watch stream deadline exceeded"
		return terminal
	}
	terminal.TerminalStatus = "cancelled"
	terminal.Error = nil
	return terminal
}

func sessionWatchTerminalFromContextErr(streamID string, requestID string, seq int64, ctxErr error) SessionWatchEvent {
	terminal := SessionWatchEvent{
		SchemaID:      "runecode.protocol.v0.SessionWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      streamID,
		RequestID:     requestID,
		Seq:           seq,
		EventType:     "session_watch_terminal",
		Terminal:      true,
		Error: &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "request_cancelled",
			Category:      "transport",
			Retryable:     true,
			Message:       "session watch stream cancelled",
		},
	}
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		terminal.TerminalStatus = "failed"
		terminal.Error.Code = "broker_timeout_request_deadline_exceeded"
		terminal.Error.Category = "timeout"
		terminal.Error.Message = "session watch stream deadline exceeded"
		return terminal
	}
	terminal.TerminalStatus = "cancelled"
	terminal.Error = nil
	return terminal
}

func finalizeRunWatchRequest(req RunWatchRequest) {
	if req.Cancel != nil {
		req.Cancel()
	}
	if req.Release != nil {
		req.Release()
	}
}

func finalizeApprovalWatchRequest(req ApprovalWatchRequest) {
	if req.Cancel != nil {
		req.Cancel()
	}
	if req.Release != nil {
		req.Release()
	}
}

func finalizeSessionWatchRequest(req SessionWatchRequest) {
	if req.Cancel != nil {
		req.Cancel()
	}
	if req.Release != nil {
		req.Release()
	}
}

func ptrRunSummary(value RunSummary) *RunSummary {
	v := value
	return &v
}

func ptrApprovalSummary(value ApprovalSummary) *ApprovalSummary {
	v := value
	return &v
}

func ptrSessionSummary(value SessionSummary) *SessionSummary {
	v := value
	return &v
}
