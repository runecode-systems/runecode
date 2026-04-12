package brokerapi

func (s *Service) StreamRunWatchEvents(req RunWatchRequest) ([]RunWatchEvent, error) {
	defer finalizeRunWatchRequest(req)
	runs, err := s.runWatchSummaries(req)
	if err != nil {
		return nil, err
	}
	events := runWatchEventsFromSummaries(req, runs)
	if err := validateRunWatchSemantics(events); err != nil {
		return nil, err
	}
	for i := range events {
		if err := s.validateResponse(events[i], runWatchEventSchemaPath); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (s *Service) StreamApprovalWatchEvents(req ApprovalWatchRequest) ([]ApprovalWatchEvent, error) {
	defer finalizeApprovalWatchRequest(req)
	approvals := s.approvalWatchSummaries(req)
	events := approvalWatchEventsFromSummaries(req, approvals)
	if err := validateApprovalWatchSemantics(events); err != nil {
		return nil, err
	}
	for i := range events {
		if err := s.validateResponse(events[i], approvalWatchEventSchemaPath); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (s *Service) StreamSessionWatchEvents(req SessionWatchRequest) ([]SessionWatchEvent, error) {
	defer finalizeSessionWatchRequest(req)
	sessions, err := s.sessionWatchSummaries(req)
	if err != nil {
		return nil, err
	}
	events := sessionWatchEventsFromSummaries(req, sessions)
	if err := validateSessionWatchSemantics(events); err != nil {
		return nil, err
	}
	for i := range events {
		if err := s.validateResponse(events[i], sessionWatchEventSchemaPath); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (s *Service) runWatchSummaries(req RunWatchRequest) ([]RunSummary, error) {
	allRuns, err := s.runSummaries("updated_at_desc")
	if err != nil {
		return nil, err
	}
	return filterRunWatchSummaries(allRuns, req), nil
}

func (s *Service) approvalWatchSummaries(req ApprovalWatchRequest) []ApprovalSummary {
	approvals := s.listApprovals()
	approvals = filterApprovalWatchSummaries(approvals, req)
	sortApprovals(approvals)
	return approvals
}

func (s *Service) sessionWatchSummaries(req SessionWatchRequest) ([]SessionSummary, error) {
	summaries, err := s.sessionSummaries("updated_at_desc")
	if err != nil {
		return nil, err
	}
	return filterSessionWatchSummaries(summaries, req), nil
}

func runWatchEventsFromSummaries(req RunWatchRequest, runs []RunSummary) []RunWatchEvent {
	events := make([]RunWatchEvent, 0, 3)
	seq := int64(1)
	if req.IncludeSnapshot && len(runs) > 0 {
		events = append(events, runWatchSnapshotEvent(req, seq, runs[0]))
		seq++
	}
	if err := reqContextErr(req.RequestCtx); err != nil {
		events = append(events, runWatchTerminalFromContextErr(req.StreamID, req.RequestID, seq, err))
		return events
	}
	if req.Follow && len(runs) > 0 {
		events = append(events, runWatchUpsertEvent(req, seq, runs))
		seq++
	}
	events = append(events, completedRunWatchTerminal(req, seq))
	return events
}

func approvalWatchEventsFromSummaries(req ApprovalWatchRequest, approvals []ApprovalSummary) []ApprovalWatchEvent {
	events := make([]ApprovalWatchEvent, 0, 3)
	seq := int64(1)
	if req.IncludeSnapshot && len(approvals) > 0 {
		events = append(events, approvalWatchSnapshotEvent(req, seq, approvals[0]))
		seq++
	}
	if err := reqContextErr(req.RequestCtx); err != nil {
		events = append(events, approvalWatchTerminalFromContextErr(req.StreamID, req.RequestID, seq, err))
		return events
	}
	if req.Follow && len(approvals) > 0 {
		events = append(events, approvalWatchUpsertEvent(req, seq, approvals))
		seq++
	}
	events = append(events, completedApprovalWatchTerminal(req, seq))
	return events
}

func sessionWatchEventsFromSummaries(req SessionWatchRequest, sessions []SessionSummary) []SessionWatchEvent {
	events := make([]SessionWatchEvent, 0, 3)
	seq := int64(1)
	if req.IncludeSnapshot && len(sessions) > 0 {
		events = append(events, sessionWatchSnapshotEvent(req, seq, sessions[0]))
		seq++
	}
	if err := reqContextErr(req.RequestCtx); err != nil {
		events = append(events, sessionWatchTerminalFromContextErr(req.StreamID, req.RequestID, seq, err))
		return events
	}
	if req.Follow && len(sessions) > 0 {
		events = append(events, sessionWatchUpsertEvent(req, seq, sessions))
		seq++
	}
	events = append(events, completedSessionWatchTerminal(req, seq))
	return events
}

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
	upsert := runs[0]
	if len(runs) > 1 {
		upsert = runs[1]
	}
	return RunWatchEvent{
		SchemaID:      "runecode.protocol.v0.RunWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "run_watch_upsert",
		Run:           ptrRunSummary(upsert),
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
	upsert := approvals[0]
	if len(approvals) > 1 {
		upsert = approvals[1]
	}
	return ApprovalWatchEvent{
		SchemaID:      "runecode.protocol.v0.ApprovalWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "approval_watch_upsert",
		Approval:      ptrApprovalSummary(upsert),
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
	upsert := sessions[0]
	if len(sessions) > 1 {
		upsert = sessions[1]
	}
	return SessionWatchEvent{
		SchemaID:      "runecode.protocol.v0.SessionWatchEvent",
		SchemaVersion: "0.1.0",
		StreamID:      req.StreamID,
		RequestID:     req.RequestID,
		Seq:           seq,
		EventType:     "session_watch_upsert",
		Session:       ptrSessionSummary(upsert),
	}
}
