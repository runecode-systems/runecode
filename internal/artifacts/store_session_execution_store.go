package artifacts

import (
	"fmt"
	"strings"
)

func (s *Store) AppendSessionExecutionTrigger(req SessionExecutionTriggerAppendRequest) (SessionExecutionTriggerAppendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized, err := normalizeSessionExecutionTriggerAppendRequest(req)
	if err != nil {
		return SessionExecutionTriggerAppendResult{}, err
	}
	session := loadSessionForExecutionTriggerAppend(s.state.Sessions, normalized)
	if replay, handled, err := replaySessionExecutionTriggerAppend(session, normalized); handled {
		return replay, err
	}
	if err := enforceSessionExecutionTriggerAdmission(s.state.Sessions, normalized); err != nil {
		return SessionExecutionTriggerAppendResult{}, err
	}
	prior, hadPrior := s.state.Sessions[normalized.SessionID]
	if hadPrior {
		prior = copySessionDurableState(prior)
	}
	appendResult := createSessionExecutionTriggerAppendResult(&session, normalized)
	s.state.Sessions[normalized.SessionID] = session
	if err := s.saveStateLocked(); err != nil {
		if hadPrior {
			s.state.Sessions[normalized.SessionID] = prior
		} else {
			delete(s.state.Sessions, normalized.SessionID)
		}
		return SessionExecutionTriggerAppendResult{}, err
	}
	return appendResult, nil
}

func enforceSessionExecutionTriggerAdmission(states map[string]SessionDurableState, req SessionExecutionTriggerAppendRequest) error {
	if !sessionExecutionTriggerRequiresRepoRootOverlapAdmission(req) {
		return nil
	}
	repoRoot := strings.TrimSpace(req.AuthoritativeRepositoryRoot)
	if repoRoot == "" {
		return fmt.Errorf("%w: authoritative repository root is required", ErrSessionExecutionTriggerOverlapDenied)
	}
	for _, session := range states {
		if execution, ok := activeExecutionForRepoRoot(session, repoRoot); ok {
			return fmt.Errorf("%w: active mutation-bearing shared-workspace run already exists for repository root %q (session=%q turn=%q state=%q)", ErrSessionExecutionTriggerOverlapDenied, repoRoot, execution.SessionID, execution.TurnID, execution.ExecutionState)
		}
	}
	return nil
}

func sessionExecutionTriggerRequiresRepoRootOverlapAdmission(req SessionExecutionTriggerAppendRequest) bool {
	if strings.TrimSpace(req.RequestedOperation) != "start" {
		return false
	}
	return sessionWorkflowRoutingIsMutationBearingSharedWorkspace(req.WorkflowRouting)
}

func sessionExecutionTriggerIsMutationBearingSharedWorkspace(req SessionExecutionTriggerAppendRequest) bool {
	return sessionWorkflowRoutingIsMutationBearingSharedWorkspace(req.WorkflowRouting)
}

func sessionWorkflowRoutingIsMutationBearingSharedWorkspace(routing SessionWorkflowPackRoutingDurableState) bool {
	if strings.TrimSpace(routing.WorkflowFamily) != "runecontext" {
		return true
	}
	switch strings.TrimSpace(routing.WorkflowOperation) {
	case "change_draft", "spec_draft":
		return len(routing.BoundInputArtifacts) != 0
	case "draft_promote_apply", "approved_change_implementation":
		return true
	default:
		return true
	}
}

func activeExecutionForRepoRoot(session SessionDurableState, repoRoot string) (SessionTurnExecutionDurableState, bool) {
	rootByTriggerID := map[string]string{}
	for _, trigger := range session.ExecutionTriggers {
		rootByTriggerID[strings.TrimSpace(trigger.TriggerID)] = strings.TrimSpace(trigger.AuthoritativeRepositoryRoot)
	}
	for _, execution := range session.TurnExecutions {
		if sessionTurnExecutionIsTerminal(execution) {
			continue
		}
		if !sessionTurnExecutionIsMutationBearingSharedWorkspace(execution) {
			continue
		}
		execRoot := rootByTriggerID[strings.TrimSpace(execution.TriggerID)]
		if execRoot == "" {
			return execution, true
		}
		if execRoot != repoRoot {
			continue
		}
		return execution, true
	}
	return SessionTurnExecutionDurableState{}, false
}

func sessionTurnExecutionIsMutationBearingSharedWorkspace(exec SessionTurnExecutionDurableState) bool {
	return sessionWorkflowRoutingIsMutationBearingSharedWorkspace(exec.WorkflowRouting)
}

func sessionTurnExecutionIsTerminal(exec SessionTurnExecutionDurableState) bool {
	switch strings.TrimSpace(exec.ExecutionState) {
	case "completed", "failed":
		return true
	default:
		return false
	}
}

func (s *Store) UpdateSessionTurnExecution(req SessionTurnExecutionUpdateRequest) (SessionTurnExecutionDurableState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized, err := normalizeSessionTurnExecutionUpdateRequest(req)
	if err != nil {
		return SessionTurnExecutionDurableState{}, err
	}
	session, ok := s.state.Sessions[normalized.SessionID]
	if !ok || session.SessionID == "" {
		return SessionTurnExecutionDurableState{}, ErrSessionTurnExecutionNotFound
	}
	prior := copySessionDurableState(session)
	idx := sessionTurnExecutionIndex(session.TurnExecutions, normalized.TurnID)
	if idx == -1 {
		return SessionTurnExecutionDurableState{}, ErrSessionTurnExecutionNotFound
	}
	exec := applySessionTurnExecutionUpdate(session.TurnExecutions[idx], normalized)
	session.TurnExecutions[idx] = exec
	applySessionExecutionSummaryUpdate(&session, exec, normalized.OccurredAt)
	s.state.Sessions[normalized.SessionID] = session
	if err := s.saveStateLocked(); err != nil {
		s.state.Sessions[normalized.SessionID] = prior
		return SessionTurnExecutionDurableState{}, err
	}
	return copySessionTurnExecutionDurableState(exec), nil
}

func replaySessionExecutionTriggerAppend(session SessionDurableState, req SessionExecutionTriggerAppendRequest) (SessionExecutionTriggerAppendResult, bool, error) {
	if req.IdempotencyKey == "" {
		return SessionExecutionTriggerAppendResult{}, false, nil
	}
	if session.ExecutionTriggerIdempotencyByKey == nil {
		session.ExecutionTriggerIdempotencyByKey = map[string]SessionExecutionTriggerIdempotencyRecord{}
	}
	prior, ok := session.ExecutionTriggerIdempotencyByKey[req.IdempotencyKey]
	if !ok {
		return SessionExecutionTriggerAppendResult{}, false, nil
	}
	if prior.RequestHash != req.IdempotencyHash {
		return SessionExecutionTriggerAppendResult{}, true, ErrSessionExecutionTriggerIdempotencyKeyConflict
	}
	trigger, found := sessionReplayExecutionTrigger(session, prior)
	if !found {
		return SessionExecutionTriggerAppendResult{}, true, fmt.Errorf("idempotency replay state missing execution trigger records")
	}
	turnExecution, ok := sessionReplayTurnExecutionByTriggerID(session, trigger.TriggerID)
	if !ok {
		return SessionExecutionTriggerAppendResult{}, true, fmt.Errorf("idempotency replay state missing turn execution records")
	}
	return SessionExecutionTriggerAppendResult{Created: false, Trigger: trigger, TurnExecution: turnExecution, Seq: prior.Seq, IdempotencyHash: prior.RequestHash}, true, nil
}

func createSessionExecutionTriggerAppendResult(session *SessionDurableState, req SessionExecutionTriggerAppendRequest) SessionExecutionTriggerAppendResult {
	trigger, turnExecution, seq := appendSessionExecutionTrigger(session, req)
	storeSessionExecutionTriggerIdempotencyRecord(session, req, trigger, seq)
	return SessionExecutionTriggerAppendResult{
		Created:         true,
		Trigger:         copySessionExecutionTriggerDurableState(trigger),
		TurnExecution:   copySessionTurnExecutionDurableState(turnExecution),
		Seq:             seq,
		IdempotencyHash: req.IdempotencyHash,
	}
}

func storeSessionExecutionTriggerIdempotencyRecord(session *SessionDurableState, req SessionExecutionTriggerAppendRequest, trigger SessionExecutionTriggerDurableState, seq int64) {
	if req.IdempotencyKey == "" {
		return
	}
	if session.ExecutionTriggerIdempotencyByKey == nil {
		session.ExecutionTriggerIdempotencyByKey = map[string]SessionExecutionTriggerIdempotencyRecord{}
	}
	session.ExecutionTriggerIdempotencyByKey[req.IdempotencyKey] = SessionExecutionTriggerIdempotencyRecord{
		RequestHash: req.IdempotencyHash,
		TriggerID:   trigger.TriggerID,
		Seq:         seq,
	}
}

func appendSessionExecutionTrigger(session *SessionDurableState, req SessionExecutionTriggerAppendRequest) (SessionExecutionTriggerDurableState, SessionTurnExecutionDurableState, int64) {
	triggerIndex, triggerID, turnID := nextSessionExecutionIDs(session)
	trigger := newSessionExecutionTriggerState(session.SessionID, triggerID, triggerIndex, req)
	turnExecution := newSessionTurnExecutionState(session.SessionID, turnID, triggerID, triggerIndex, req)
	session.ExecutionTriggers = append(session.ExecutionTriggers, trigger)
	session.TurnExecutions = append(session.TurnExecutions, turnExecution)
	mergeSessionExecutionLinkedRunIDs(session, req)
	applySessionExecutionSummaryUpdate(session, turnExecution, req.OccurredAt)
	session.LastActivityPreview = previewSessionExecutionTrigger(req)
	session.LastInteractionSequence++
	seq := session.LastInteractionSequence
	return trigger, turnExecution, seq
}

func sessionReplayExecutionTrigger(session SessionDurableState, record SessionExecutionTriggerIdempotencyRecord) (SessionExecutionTriggerDurableState, bool) {
	for _, trigger := range session.ExecutionTriggers {
		if trigger.TriggerID == record.TriggerID {
			return copySessionExecutionTriggerDurableState(trigger), true
		}
	}
	return SessionExecutionTriggerDurableState{}, false
}

func sessionReplayTurnExecutionByTriggerID(session SessionDurableState, triggerID string) (SessionTurnExecutionDurableState, bool) {
	for _, execution := range session.TurnExecutions {
		if execution.TriggerID == triggerID {
			return copySessionTurnExecutionDurableState(execution), true
		}
	}
	return SessionTurnExecutionDurableState{}, false
}
