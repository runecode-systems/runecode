package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func externalAnchorDeferredAttemptMatches(record artifacts.ExternalAnchorPreparedMutationRecord, attempt externalAnchorPreparedExecutionAttempt) bool {
	return strings.TrimSpace(record.LastExecuteAttemptID) == strings.TrimSpace(attempt.AttemptID) &&
		strings.TrimSpace(record.ExecutionState) == gitRemoteMutationExecutionDeferred &&
		strings.TrimSpace(record.LifecycleState) == gitRemoteMutationLifecyclePrepared
}

func externalAnchorDeferredAttemptClaimedByService(record artifacts.ExternalAnchorPreparedMutationRecord, attempt externalAnchorPreparedExecutionAttempt, claimID string) bool {
	if !externalAnchorDeferredAttemptMatches(record, attempt) {
		return false
	}
	return strings.TrimSpace(record.LastExecuteDeferredClaimID) == strings.TrimSpace(claimID)
}

func deferredExternalAnchorResumableAttempt(record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorPreparedExecutionAttempt, bool) {
	attemptID := strings.TrimSpace(record.LastExecuteAttemptID)
	if attemptID == "" || !externalAnchorDeferredAttemptMatches(record, externalAnchorPreparedExecutionAttempt{PreparedMutationID: record.PreparedMutationID, AttemptID: attemptID}) {
		return externalAnchorPreparedExecutionAttempt{}, false
	}
	input, err := externalAnchorExecutionInputFromRecord(record)
	if err != nil || strings.TrimSpace(input.Mode) != "deferred_poll" {
		return externalAnchorPreparedExecutionAttempt{}, false
	}
	return externalAnchorPreparedExecutionAttempt{PreparedMutationID: strings.TrimSpace(record.PreparedMutationID), AttemptID: attemptID}, true
}

func (s *Service) claimDeferredExternalAnchorAttempt(attempt externalAnchorPreparedExecutionAttempt) bool {
	if s == nil {
		return false
	}
	_, claimed, err := s.ExternalAnchorPreparedClaimDeferredExecution(attempt.PreparedMutationID, attempt.AttemptID, s.deferredExternalAnchorClaimID(), externalAnchorDeferredClaimStaleAfter, s.currentTimestamp())
	if err != nil {
		return false
	}
	return claimed
}

func (s *Service) releaseDeferredExternalAnchorAttemptClaim(attempt externalAnchorPreparedExecutionAttempt) {
	if s == nil {
		return
	}
	_, _ = s.ExternalAnchorPreparedTransitionLifecycle(attempt.PreparedMutationID, gitRemoteMutationLifecyclePrepared, func(current artifacts.ExternalAnchorPreparedMutationRecord) artifacts.ExternalAnchorPreparedMutationRecord {
		if !externalAnchorDeferredAttemptClaimedByService(current, attempt, s.deferredExternalAnchorClaimID()) {
			return current
		}
		current.LastExecuteDeferredClaimID = ""
		current.LastExecuteDeferredClaimedAt = nil
		return current
	})
}

func (s *Service) deferredExternalAnchorClaimID() string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("external-anchor-worker:%s", strings.TrimSpace(s.lifecycleGeneration))
}

func (s *Service) persistDeferredExternalAnchorExecution(attempt externalAnchorPreparedExecutionAttempt, input externalAnchorExecutionInput, outcome externalAnchorExecutionOutcome) bool {
	updated, err := s.ExternalAnchorPreparedTransitionLifecycle(attempt.PreparedMutationID, gitRemoteMutationLifecyclePrepared, func(current artifacts.ExternalAnchorPreparedMutationRecord) artifacts.ExternalAnchorPreparedMutationRecord {
		if !externalAnchorDeferredAttemptClaimedByService(current, attempt, s.deferredExternalAnchorClaimID()) {
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
