package brokerapi

import (
	"sort"
	"strings"
	"time"
)

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

func (s *Service) StreamSessionTurnExecutionWatchEvents(req SessionTurnExecutionWatchRequest) ([]SessionTurnExecutionWatchEvent, error) {
	defer finalizeSessionTurnExecutionWatchRequest(req)
	executions := s.sessionTurnExecutionWatchStates(req)
	events := sessionTurnExecutionWatchEventsFromStates(req, executions)
	if err := validateSessionTurnExecutionWatchSemantics(events); err != nil {
		return nil, err
	}
	for i := range events {
		if err := s.validateResponse(events[i], sessionTurnExecutionWatchEventSchemaPath); err != nil {
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

func (s *Service) sessionTurnExecutionWatchStates(req SessionTurnExecutionWatchRequest) []SessionTurnExecution {
	states := s.store.SessionDurableStates()
	filtered := filterSessionTurnExecutionWatchStates(states, req)
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].UpdatedAt.Equal(filtered[j].UpdatedAt) {
			if filtered[i].SessionID == filtered[j].SessionID {
				return filtered[i].ExecutionIndex > filtered[j].ExecutionIndex
			}
			return filtered[i].SessionID < filtered[j].SessionID
		}
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})
	out := make([]SessionTurnExecution, 0, len(filtered))
	for _, execution := range filtered {
		out = append(out, buildSessionTurnExecutionFromDurable(execution))
	}
	return out
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

func sessionTurnExecutionWatchEventsFromStates(req SessionTurnExecutionWatchRequest, executions []SessionTurnExecution) []SessionTurnExecutionWatchEvent {
	events := make([]SessionTurnExecutionWatchEvent, 0, 3+len(executions))
	seq := int64(1)
	if req.IncludeSnapshot && len(executions) > 0 {
		events = append(events, sessionTurnExecutionWatchSnapshotEvent(req, seq, executions[0]))
		seq++
	}
	if err := reqContextErr(req.RequestCtx); err != nil {
		events = append(events, sessionTurnExecutionWatchTerminalFromContextErr(req.StreamID, req.RequestID, seq, err))
		return events
	}
	if req.Follow {
		for _, execution := range sessionTurnExecutionFollowCandidates(executions, req.IncludeSnapshot) {
			events = append(events, sessionTurnExecutionWatchUpsertEventForExecution(req, seq, execution))
			seq++
		}
	}
	events = append(events, completedSessionTurnExecutionWatchTerminal(req, seq))
	return events
}

func sessionTurnExecutionFollowCandidates(executions []SessionTurnExecution, includeSnapshot bool) []SessionTurnExecution {
	if len(executions) == 0 {
		return nil
	}
	if !includeSnapshot {
		return executions
	}
	if len(executions) == 1 {
		return nil
	}
	snapshot := executions[0]
	out := make([]SessionTurnExecution, 0, len(executions)-1)
	for idx := 1; idx < len(executions); idx++ {
		candidate := executions[idx]
		if sessionTurnExecutionEquivalentForFollow(candidate, snapshot) {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func sessionTurnExecutionEquivalentForFollow(candidate, snapshot SessionTurnExecution) bool {
	if !sameSessionTurnIdentity(candidate, snapshot) {
		return false
	}
	if candidate.ExecutionIndex != snapshot.ExecutionIndex {
		return false
	}
	if !sameInstant(candidate.UpdatedAt, snapshot.UpdatedAt) {
		return false
	}
	if !sameInstant(candidate.CreatedAt, snapshot.CreatedAt) {
		return false
	}
	if strings.TrimSpace(candidate.ExecutionState) != strings.TrimSpace(snapshot.ExecutionState) {
		return false
	}
	if strings.TrimSpace(candidate.WaitKind) != strings.TrimSpace(snapshot.WaitKind) {
		return false
	}
	if strings.TrimSpace(candidate.WaitState) != strings.TrimSpace(snapshot.WaitState) {
		return false
	}
	if strings.TrimSpace(candidate.BlockedReasonCode) != strings.TrimSpace(snapshot.BlockedReasonCode) {
		return false
	}
	return true
}

func sameSessionTurnIdentity(candidate, snapshot SessionTurnExecution) bool {
	if strings.TrimSpace(candidate.SessionID) != strings.TrimSpace(snapshot.SessionID) {
		return false
	}
	candidateTurnID := strings.TrimSpace(candidate.TurnID)
	snapshotTurnID := strings.TrimSpace(snapshot.TurnID)
	if candidateTurnID == "" || snapshotTurnID == "" {
		return false
	}
	return candidateTurnID == snapshotTurnID
}

func sameInstant(a, b string) bool {
	if strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "" {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	parsedA, errA := time.Parse(time.RFC3339, strings.TrimSpace(a))
	parsedB, errB := time.Parse(time.RFC3339, strings.TrimSpace(b))
	if errA != nil || errB != nil {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	return parsedA.Equal(parsedB)
}
