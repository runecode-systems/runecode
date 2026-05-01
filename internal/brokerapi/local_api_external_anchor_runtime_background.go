package brokerapi

import (
	"context"
	"strings"
	"sync"
	"time"
)

type externalAnchorPreparedExecutionAttempt struct {
	PreparedMutationID string
	AttemptID          string
}

const externalAnchorDeferredClaimStaleAfter = 30 * time.Second

type externalAnchorBackgroundQueue struct {
	mu         sync.Mutex
	pending    []externalAnchorPreparedExecutionAttempt
	inFlight   map[string]bool
	workerWake chan struct{}
	stop       chan struct{}
	running    bool
}

func newExternalAnchorBackgroundQueue() *externalAnchorBackgroundQueue {
	return &externalAnchorBackgroundQueue{
		pending:    []externalAnchorPreparedExecutionAttempt{},
		inFlight:   map[string]bool{},
		workerWake: make(chan struct{}, 1),
		stop:       make(chan struct{}),
	}
}

func (q *externalAnchorBackgroundQueue) markRunning() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.running {
		return false
	}
	q.running = true
	return true
}

func (q *externalAnchorBackgroundQueue) enqueue(attempt externalAnchorPreparedExecutionAttempt) {
	key := strings.TrimSpace(attempt.PreparedMutationID) + ":" + strings.TrimSpace(attempt.AttemptID)
	if key == ":" {
		return
	}
	q.mu.Lock()
	if q.inFlight[key] || externalAnchorAttemptAlreadyPending(q.pending, attempt) {
		q.mu.Unlock()
		return
	}
	q.pending = append(q.pending, attempt)
	q.inFlight[key] = true
	q.mu.Unlock()
	select {
	case q.workerWake <- struct{}{}:
	default:
	}
}

func externalAnchorAttemptAlreadyPending(pending []externalAnchorPreparedExecutionAttempt, attempt externalAnchorPreparedExecutionAttempt) bool {
	for _, next := range pending {
		if strings.TrimSpace(next.PreparedMutationID) == strings.TrimSpace(attempt.PreparedMutationID) && strings.TrimSpace(next.AttemptID) == strings.TrimSpace(attempt.AttemptID) {
			return true
		}
	}
	return false
}

func (q *externalAnchorBackgroundQueue) next() (externalAnchorPreparedExecutionAttempt, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) == 0 {
		return externalAnchorPreparedExecutionAttempt{}, false
	}
	next := q.pending[0]
	q.pending = q.pending[1:]
	return next, true
}

func (q *externalAnchorBackgroundQueue) complete(attempt externalAnchorPreparedExecutionAttempt) {
	key := strings.TrimSpace(attempt.PreparedMutationID) + ":" + strings.TrimSpace(attempt.AttemptID)
	if key == ":" {
		return
	}
	q.mu.Lock()
	delete(q.inFlight, key)
	q.mu.Unlock()
}

func (q *externalAnchorBackgroundQueue) close() {
	close(q.stop)
}

func (s *Service) startExternalAnchorBackgroundWorkers() {
	if s == nil || s.externalAnchorQueue == nil || !s.externalAnchorQueue.markRunning() {
		return
	}
	workers := s.apiConfig.ExternalAnchor.MaxParallelExecutions
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		go s.externalAnchorBackgroundWorker()
	}
}

func (s *Service) resumeDeferredExternalAnchorExecutionsFromDurableState() {
	if s == nil {
		return
	}
	resumable := make([]externalAnchorPreparedExecutionAttempt, 0)
	for _, preparedMutationID := range s.ExternalAnchorPreparedIDs() {
		record, ok := s.ExternalAnchorPreparedGet(preparedMutationID)
		if !ok {
			continue
		}
		attempt, ok := deferredExternalAnchorResumableAttempt(record)
		if !ok {
			continue
		}
		resumable = append(resumable, attempt)
	}
	if len(resumable) == 0 {
		return
	}
	s.startExternalAnchorBackgroundWorkers()
	for _, attempt := range resumable {
		s.externalAnchorQueue.enqueue(attempt)
	}
}

func (s *Service) enqueueDeferredExternalAnchorExecution(preparedMutationID string) {
	if s == nil || s.externalAnchorQueue == nil {
		return
	}
	record, ok := s.ExternalAnchorPreparedGet(preparedMutationID)
	if !ok {
		return
	}
	attempt, ok := deferredExternalAnchorResumableAttempt(record)
	if !ok {
		return
	}
	s.startExternalAnchorBackgroundWorkers()
	s.externalAnchorQueue.enqueue(attempt)
}

func (s *Service) externalAnchorBackgroundWorker() {
	for {
		select {
		case <-s.externalAnchorQueue.stop:
			return
		case <-s.externalAnchorQueue.workerWake:
		}
		for {
			attempt, ok := s.externalAnchorQueue.next()
			if !ok {
				break
			}
			s.processDeferredExternalAnchorAttempt(attempt)
			s.externalAnchorQueue.complete(attempt)
		}
	}
}

func (s *Service) processDeferredExternalAnchorAttempt(attempt externalAnchorPreparedExecutionAttempt) {
	if !s.claimDeferredExternalAnchorAttempt(attempt) {
		return
	}
	const maxTransitions = 6
	for i := 0; i < maxTransitions; i++ {
		input, ok := s.deferredExternalAnchorExecutionInput(attempt)
		if !ok {
			s.releaseDeferredExternalAnchorAttemptClaim(attempt)
			return
		}
		outcome := normalizeExternalAnchorExecutionOutcome(s.externalAnchorRuntime.Execute(context.Background(), input))
		stillDeferred := s.persistDeferredExternalAnchorExecution(attempt, input, outcome)
		if !stillDeferred {
			return
		}
		if input.PollRemaining <= 0 {
			s.releaseDeferredExternalAnchorAttemptClaim(attempt)
			s.externalAnchorQueue.enqueue(attempt)
			return
		}
		time.Sleep(externalAnchorDeferredBackoff(input.PollRemaining + 1))
	}
	s.releaseDeferredExternalAnchorAttemptClaim(attempt)
	s.externalAnchorQueue.enqueue(attempt)
}

func (s *Service) deferredExternalAnchorExecutionInput(attempt externalAnchorPreparedExecutionAttempt) (externalAnchorExecutionInput, bool) {
	record, ok := s.ExternalAnchorPreparedGet(attempt.PreparedMutationID)
	if !ok || !externalAnchorDeferredAttemptClaimedByService(record, attempt, s.deferredExternalAnchorClaimID()) {
		return externalAnchorExecutionInput{}, false
	}
	input, err := externalAnchorExecutionInputFromRecord(record)
	if err != nil || strings.TrimSpace(input.Mode) != "deferred_poll" {
		return externalAnchorExecutionInput{}, false
	}
	if input.PollRemaining > 0 {
		input.PollRemaining--
	}
	return input, true
}
