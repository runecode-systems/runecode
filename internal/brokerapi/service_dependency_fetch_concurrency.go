package brokerapi

import "context"

func (s *dependencyFetchService) acquireFlight(requestHash string) (*dependencyFetchFlight, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.inflight[requestHash]; ok {
		current.waiters++
		return current, false
	}
	flight := &dependencyFetchFlight{done: make(chan struct{}), waiters: 1}
	s.inflight[requestHash] = flight
	return flight, true
}

func (s *dependencyFetchService) releaseFlightWaiter(requestHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	flight, ok := s.inflight[requestHash]
	if !ok {
		return
	}
	flight.waiters--
	if flight.waiters <= 0 {
		delete(s.inflight, requestHash)
	}
}

func (s *dependencyFetchService) acquireFetchToken(ctx context.Context) error {
	select {
	case s.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *dependencyFetchService) releaseFetchToken() {
	select {
	case <-s.sem:
	default:
	}
}
