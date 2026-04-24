package artifacts

import "strings"

func (s *Store) SessionState(sessionID string) (SessionDurableState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return SessionDurableState{}, false
	}
	state, ok := s.state.Sessions[id]
	if !ok {
		return SessionDurableState{}, false
	}
	return copySessionDurableState(state), true
}

func (s *Store) SessionStates() map[string]SessionDurableState {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]SessionDurableState, len(s.state.Sessions))
	for sessionID, state := range s.state.Sessions {
		out[sessionID] = copySessionDurableState(state)
	}
	return out
}

func (s *Store) SessionDurableStates() []SessionDurableState {
	states := s.SessionStates()
	return SessionSummaryStatesByUpdateDesc(states)
}

func (s *Store) UpdateSessionState(sessionID string, mutate func(SessionDurableState) SessionDurableState) (SessionDurableState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return SessionDurableState{}, ErrSessionTurnExecutionNotFound
	}
	state, ok := s.state.Sessions[id]
	if !ok {
		return SessionDurableState{}, ErrSessionTurnExecutionNotFound
	}
	state = mutate(copySessionDurableState(state))
	s.state.Sessions[id] = state
	if err := s.saveStateLocked(); err != nil {
		return SessionDurableState{}, err
	}
	return copySessionDurableState(state), nil
}
