package artifacts

import (
	"fmt"
	"strings"
	"time"
)

func (s *Store) ExternalAnchorPreparedClaimDeferredExecution(preparedMutationID, expectedAttemptID, claimID string, staleAfter time.Duration, claimedAt time.Time) (ExternalAnchorPreparedMutationRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	preparedMutationID, expectedAttemptID, claimID, claimedAt, err := normalizeExternalAnchorPreparedClaimInput(preparedMutationID, expectedAttemptID, claimID, claimedAt, s.nowFn)
	if err != nil {
		return ExternalAnchorPreparedMutationRecord{}, false, err
	}
	rec, ok := s.state.ExternalAnchorPrepared[preparedMutationID]
	if !ok {
		return ExternalAnchorPreparedMutationRecord{}, false, fmt.Errorf("prepared mutation %q not found", preparedMutationID)
	}
	if !externalAnchorPreparedRecordIsDeferredResumable(rec, expectedAttemptID) || externalAnchorPreparedDeferredClaimUnavailable(rec, claimID, staleAfter, claimedAt) {
		return rec, false, nil
	}
	rec.LastExecuteDeferredClaimID = claimID
	rec.LastExecuteDeferredClaimedAt = &claimedAt
	if err := validateExternalAnchorPreparedRecord(rec); err != nil {
		return ExternalAnchorPreparedMutationRecord{}, false, err
	}
	return s.persistClaimedExternalAnchorPreparedRecord(preparedMutationID, rec)
}

func normalizeExternalAnchorPreparedClaimInput(preparedMutationID, expectedAttemptID, claimID string, claimedAt time.Time, nowFn func() time.Time) (string, string, string, time.Time, error) {
	preparedMutationID = strings.TrimSpace(preparedMutationID)
	expectedAttemptID = strings.TrimSpace(expectedAttemptID)
	claimID = strings.TrimSpace(claimID)
	if preparedMutationID == "" {
		return "", "", "", time.Time{}, fmt.Errorf("prepared mutation id is required")
	}
	if expectedAttemptID == "" {
		return "", "", "", time.Time{}, fmt.Errorf("expected attempt id is required")
	}
	if claimID == "" {
		return "", "", "", time.Time{}, fmt.Errorf("claim id is required")
	}
	if claimedAt.IsZero() {
		claimedAt = nowFn().UTC()
	} else {
		claimedAt = claimedAt.UTC()
	}
	return preparedMutationID, expectedAttemptID, claimID, claimedAt, nil
}

func externalAnchorPreparedDeferredClaimUnavailable(record ExternalAnchorPreparedMutationRecord, claimID string, staleAfter time.Duration, claimedAt time.Time) bool {
	existingClaimID := strings.TrimSpace(record.LastExecuteDeferredClaimID)
	if existingClaimID == "" || existingClaimID == claimID {
		return false
	}
	if staleAfter <= 0 {
		return true
	}
	claimedAtValue := externalAnchorPreparedDeferredClaimedAt(record)
	return claimedAtValue.IsZero() || claimedAtValue.Add(staleAfter).After(claimedAt)
}

func externalAnchorPreparedDeferredClaimedAt(record ExternalAnchorPreparedMutationRecord) time.Time {
	if record.LastExecuteDeferredClaimedAt == nil {
		return time.Time{}
	}
	return record.LastExecuteDeferredClaimedAt.UTC()
}

func externalAnchorPreparedRecordIsDeferredResumable(record ExternalAnchorPreparedMutationRecord, expectedAttemptID string) bool {
	return strings.TrimSpace(record.LifecycleState) == "prepared" &&
		strings.TrimSpace(record.ExecutionState) == "deferred" &&
		strings.TrimSpace(record.LastExecuteAttemptID) == strings.TrimSpace(expectedAttemptID)
}

func (s *Store) persistClaimedExternalAnchorPreparedRecord(preparedMutationID string, rec ExternalAnchorPreparedMutationRecord) (ExternalAnchorPreparedMutationRecord, bool, error) {
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = s.nowFn().UTC()
	}
	rec.UpdatedAt = s.nowFn().UTC()
	s.state.ExternalAnchorPrepared[preparedMutationID] = rec
	rebuildRunExternalAnchorPreparedRefsLocked(&s.state)
	if err := s.saveStateLocked(); err != nil {
		return ExternalAnchorPreparedMutationRecord{}, false, err
	}
	return rec, true, nil
}
