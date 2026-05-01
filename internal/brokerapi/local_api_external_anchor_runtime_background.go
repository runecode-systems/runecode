package brokerapi

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type externalAnchorPreparedExecutionAttempt struct {
	PreparedMutationID string
	AttemptID          string
}

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
	const maxTransitions = 6
	for i := 0; i < maxTransitions; i++ {
		input, ok := s.deferredExternalAnchorExecutionInput(attempt)
		if !ok {
			return
		}
		outcome := normalizeExternalAnchorExecutionOutcome(s.externalAnchorRuntime.Execute(context.Background(), input))
		if !s.persistDeferredExternalAnchorExecution(attempt, input, outcome) || input.PollRemaining <= 0 {
			return
		}
		time.Sleep(externalAnchorDeferredBackoff(input.PollRemaining + 1))
	}
}

func (s *Service) deferredExternalAnchorExecutionInput(attempt externalAnchorPreparedExecutionAttempt) (externalAnchorExecutionInput, bool) {
	record, ok := s.ExternalAnchorPreparedGet(attempt.PreparedMutationID)
	if !ok || !externalAnchorDeferredAttemptMatches(record, attempt) {
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

func externalAnchorDeferredAttemptMatches(record artifacts.ExternalAnchorPreparedMutationRecord, attempt externalAnchorPreparedExecutionAttempt) bool {
	return strings.TrimSpace(record.LastExecuteAttemptID) == strings.TrimSpace(attempt.AttemptID) &&
		strings.TrimSpace(record.ExecutionState) == gitRemoteMutationExecutionDeferred &&
		strings.TrimSpace(record.LifecycleState) == gitRemoteMutationLifecyclePrepared
}

func (s *Service) persistDeferredExternalAnchorExecution(attempt externalAnchorPreparedExecutionAttempt, input externalAnchorExecutionInput, outcome externalAnchorExecutionOutcome) bool {
	updated, err := s.ExternalAnchorPreparedTransitionLifecycle(attempt.PreparedMutationID, gitRemoteMutationLifecyclePrepared, func(current artifacts.ExternalAnchorPreparedMutationRecord) artifacts.ExternalAnchorPreparedMutationRecord {
		if !externalAnchorDeferredAttemptMatches(current, attempt) {
			return current
		}
		setExternalAnchorExecutionOutcome(&current, outcome)
		if strings.TrimSpace(current.ExecutionState) == gitRemoteMutationExecutionDeferred {
			setExternalAnchorDeferredPollRemaining(&current, input.PollRemaining)
		}
		return current
	})
	if err != nil {
		return false
	}
	return strings.TrimSpace(updated.ExecutionState) == gitRemoteMutationExecutionDeferred
}
