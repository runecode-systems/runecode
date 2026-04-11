package brokerapi

type sessionInteractionState struct {
	NextSeq         int64
	NextTurnIndex   int
	IdempotentByKey map[string]SessionSendMessageResponse
}

func (s *Service) nextSessionInteractionSeq(sessionID string) int64 {
	s.sessionInteractionMu.Lock()
	defer s.sessionInteractionMu.Unlock()
	state := s.sessionInteractionState[sessionID]
	state.NextSeq++
	s.sessionInteractionState[sessionID] = state
	return state.NextSeq
}

func (s *Service) nextSessionInteractionTurnIndex(sessionID string, baseline int) int {
	s.sessionInteractionMu.Lock()
	defer s.sessionInteractionMu.Unlock()
	state := s.sessionInteractionState[sessionID]
	if state.NextTurnIndex < baseline {
		state.NextTurnIndex = baseline
	}
	state.NextTurnIndex++
	s.sessionInteractionState[sessionID] = state
	return state.NextTurnIndex
}

func (s *Service) sessionIdempotentInteractionResponse(sessionID, idempotencyKey string) (SessionSendMessageResponse, bool) {
	if idempotencyKey == "" {
		return SessionSendMessageResponse{}, false
	}
	s.sessionInteractionMu.Lock()
	defer s.sessionInteractionMu.Unlock()
	state, ok := s.sessionInteractionState[sessionID]
	if !ok || len(state.IdempotentByKey) == 0 {
		return SessionSendMessageResponse{}, false
	}
	resp, exists := state.IdempotentByKey[idempotencyKey]
	if !exists {
		return SessionSendMessageResponse{}, false
	}
	return resp, true
}

func (s *Service) storeSessionIdempotentInteractionResponse(sessionID, idempotencyKey string, resp SessionSendMessageResponse) {
	if idempotencyKey == "" {
		return
	}
	s.sessionInteractionMu.Lock()
	defer s.sessionInteractionMu.Unlock()
	state := s.sessionInteractionState[sessionID]
	if state.IdempotentByKey == nil {
		state.IdempotentByKey = map[string]SessionSendMessageResponse{}
	}
	state.IdempotentByKey[idempotencyKey] = resp
	s.sessionInteractionState[sessionID] = state
}
