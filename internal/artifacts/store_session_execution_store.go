package artifacts

import (
	"fmt"
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
	appendResult := createSessionExecutionTriggerAppendResult(&session, normalized)
	s.state.Sessions[normalized.SessionID] = session
	if err := s.saveStateLocked(); err != nil {
		return SessionExecutionTriggerAppendResult{}, err
	}
	return appendResult, nil
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
	idx := sessionTurnExecutionIndex(session.TurnExecutions, normalized.TurnID)
	if idx == -1 {
		return SessionTurnExecutionDurableState{}, ErrSessionTurnExecutionNotFound
	}
	exec := applySessionTurnExecutionUpdate(session.TurnExecutions[idx], normalized)
	session.TurnExecutions[idx] = exec
	applySessionExecutionSummaryUpdate(&session, exec, normalized.OccurredAt)
	s.state.Sessions[normalized.SessionID] = session
	if err := s.saveStateLocked(); err != nil {
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
