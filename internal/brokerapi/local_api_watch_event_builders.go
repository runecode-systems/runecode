package brokerapi

func runWatchSnapshotEvent(req RunWatchRequest, seq int64, summary RunSummary) RunWatchEvent {
	return RunWatchEvent{
		SchemaID:      "runecode.protocol.v0.RunWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "run_watch_snapshot",
		Run:           ptrRunSummary(summary),
	}
}

func runWatchUpsertEvent(req RunWatchRequest, seq int64, runs []RunSummary) RunWatchEvent {
	upsert := watchUpsertIndex(req.IncludeSnapshot, len(runs))
	return RunWatchEvent{
		SchemaID:      "runecode.protocol.v0.RunWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "run_watch_upsert",
		Run:           ptrRunSummary(runs[upsert]),
	}
}

func approvalWatchSnapshotEvent(req ApprovalWatchRequest, seq int64, summary ApprovalSummary) ApprovalWatchEvent {
	return ApprovalWatchEvent{
		SchemaID:      "runecode.protocol.v0.ApprovalWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "approval_watch_snapshot",
		Approval:      ptrApprovalSummary(summary),
	}
}

func approvalWatchUpsertEvent(req ApprovalWatchRequest, seq int64, approvals []ApprovalSummary) ApprovalWatchEvent {
	upsert := watchUpsertIndex(req.IncludeSnapshot, len(approvals))
	return ApprovalWatchEvent{
		SchemaID:      "runecode.protocol.v0.ApprovalWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "approval_watch_upsert",
		Approval:      ptrApprovalSummary(approvals[upsert]),
	}
}

func sessionWatchSnapshotEvent(req SessionWatchRequest, seq int64, summary SessionSummary) SessionWatchEvent {
	return SessionWatchEvent{
		SchemaID:      "runecode.protocol.v0.SessionWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "session_watch_snapshot",
		Session:       ptrSessionSummary(summary),
	}
}

func sessionWatchUpsertEvent(req SessionWatchRequest, seq int64, sessions []SessionSummary) SessionWatchEvent {
	upsert := watchUpsertIndex(req.IncludeSnapshot, len(sessions))
	return SessionWatchEvent{
		SchemaID:      "runecode.protocol.v0.SessionWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "session_watch_upsert",
		Session:       ptrSessionSummary(sessions[upsert]),
	}
}

func sessionTurnExecutionWatchSnapshotEvent(req SessionTurnExecutionWatchRequest, seq int64, execution SessionTurnExecution) SessionTurnExecutionWatchEvent {
	return SessionTurnExecutionWatchEvent{
		SchemaID:      "runecode.protocol.v0.SessionTurnExecutionWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "session_turn_execution_watch_snapshot",
		TurnExecution: ptrSessionTurnExecution(execution),
	}
}

func sessionTurnExecutionWatchUpsertEvent(req SessionTurnExecutionWatchRequest, seq int64, executions []SessionTurnExecution) SessionTurnExecutionWatchEvent {
	upsert := watchUpsertIndex(req.IncludeSnapshot, len(executions))
	return SessionTurnExecutionWatchEvent{
		SchemaID:      "runecode.protocol.v0.SessionTurnExecutionWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "session_turn_execution_watch_upsert",
		TurnExecution: ptrSessionTurnExecution(executions[upsert]),
	}
}

func sessionTurnExecutionWatchUpsertEventForExecution(req SessionTurnExecutionWatchRequest, seq int64, execution SessionTurnExecution) SessionTurnExecutionWatchEvent {
	return SessionTurnExecutionWatchEvent{
		SchemaID:      "runecode.protocol.v0.SessionTurnExecutionWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "session_turn_execution_watch_upsert",
		TurnExecution: ptrSessionTurnExecution(execution),
	}
}

func watchUpsertIndex(includeSnapshot bool, total int) int {
	if includeSnapshot && total > 1 {
		return 1
	}
	return 0
}
