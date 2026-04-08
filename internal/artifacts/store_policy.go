package artifacts

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Store) CheckFlow(req FlowCheckRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateFlowInputs(s.state.Policy, req); err != nil {
		return err
	}
	record, err := s.lookupRecord(req.Digest)
	if err != nil {
		return err
	}
	if err := s.enforceFlowRecordConsistencyLocked(record, req); err != nil {
		return err
	}
	return s.enforceFlowPolicyLocked(req)
}

func (s *Store) RevokeApprovedExcerpt(digest, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.lookupRecord(digest)
	if err != nil {
		return err
	}
	if record.Reference.DataClass != DataClassApprovedFileExcerpts {
		return errUnsupportedRevocationTarget
	}
	s.state.Policy.RevokedApprovedExcerptHashes[digest] = true
	if err := s.appendAuditLocked("artifact_promotion_action", actor, map[string]interface{}{"action": "revoked", "digest": digest}); err != nil {
		return err
	}
	return s.saveStateLocked()
}

func (s *Store) PromoteApprovedExcerpt(req PromotionRequest) (ArtifactReference, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	unapproved, ok := s.state.Artifacts[req.UnapprovedDigest]
	if !ok {
		return ArtifactReference{}, ErrArtifactNotFound
	}
	trustedVerifiers, trustErr := s.trustedVerifierRecordsLocked()
	if trustErr != nil {
		return ArtifactReference{}, errors.Join(ErrTrustedVerifierStateUnavailable, trustErr)
	}
	if err := validatePromotionRequest(s.state.Policy, unapproved, req, trustedVerifiers); err != nil {
		return ArtifactReference{}, err
	}
	if err := s.checkPromotionRateLimitLocked(req.Approver); err != nil {
		return ArtifactReference{}, err
	}
	newRef, record, decisionHash, requestHash, err := s.buildApprovedRecord(unapproved, req)
	if err != nil {
		return ArtifactReference{}, err
	}
	s.state.Artifacts[newRef.Digest] = record
	if err := s.appendAuditLocked("artifact_promotion_action", req.Approver, promotionAuditDetails(req, newRef.Digest, requestHash, decisionHash)); err != nil {
		return ArtifactReference{}, err
	}
	if err := s.saveStateLocked(); err != nil {
		return ArtifactReference{}, err
	}
	return newRef, nil
}

func (s *Store) trustedVerifierRecordsLocked() ([]trustpolicy.VerifierRecord, error) {
	records := []trustpolicy.VerifierRecord{}
	events, eventsErr := s.storeIO.readAuditEvents()
	if eventsErr != nil {
		return nil, eventsErr
	}
	for _, artifactRecord := range s.state.Artifacts {
		if !IsTrustedVerifierArtifact(artifactRecord, events) {
			continue
		}
		blob, err := s.storeIO.readBlob(artifactRecord.BlobPath)
		if err != nil {
			continue
		}
		verifierRecord := trustpolicy.VerifierRecord{}
		if err := json.Unmarshal(blob, &verifierRecord); err != nil {
			continue
		}
		records = append(records, verifierRecord)
	}
	return records, nil
}

func (s *Store) checkPromotionRateLimitLocked(actor string) error {
	if actor == "" {
		return ErrPromotionRequiresApproval
	}
	now := s.nowFn().UTC()
	windowStart := now.Add(-1 * time.Minute)
	entries := s.state.PromotionEventsByActor[actor]
	filtered := recentTimes(entries, windowStart)
	if len(filtered) >= s.state.Policy.MaxPromotionRequestsPerMinute {
		return ErrPromotionRateLimited
	}
	filtered = append(filtered, now)
	s.state.PromotionEventsByActor[actor] = filtered
	return nil
}

func (s *Store) checkQuotasLocked(role, stepID string, nextSize int64) error {
	if err := s.checkRoleQuotaLocked(role, nextSize); err != nil {
		return err
	}
	if err := s.checkStepQuotaLocked(stepID, nextSize); err != nil {
		return err
	}
	return nil
}

func (s *Store) checkRoleQuotaLocked(role string, nextSize int64) error {
	if role == "" {
		return nil
	}
	quota, ok := s.state.Policy.PerRoleQuota[role]
	if !ok {
		return nil
	}
	count, total := s.aggregateForRoleLocked(role)
	if !quotaAllows(quota, count+1, total+nextSize, nextSize) {
		return ErrQuotaExceeded
	}
	return nil
}

func (s *Store) checkStepQuotaLocked(stepID string, nextSize int64) error {
	if stepID == "" {
		return nil
	}
	quota, ok := s.state.Policy.PerStepQuota[stepID]
	if !ok {
		return nil
	}
	count, total := s.aggregateForStepLocked(stepID)
	if !quotaAllows(quota, count+1, total+nextSize, nextSize) {
		return ErrQuotaExceeded
	}
	return nil
}

func quotaAllows(q Quota, count int, total int64, single int64) bool {
	if q.MaxArtifactCount > 0 && count > q.MaxArtifactCount {
		return false
	}
	if q.MaxTotalBytes > 0 && total > q.MaxTotalBytes {
		return false
	}
	if q.MaxSingleArtifactSize > 0 && single > q.MaxSingleArtifactSize {
		return false
	}
	return true
}

func (s *Store) aggregateForRoleLocked(role string) (int, int64) {
	count := 0
	var total int64
	for _, rec := range s.state.Artifacts {
		if rec.CreatedByRole == role {
			count++
			total += rec.Reference.SizeBytes
		}
	}
	return count, total
}

func (s *Store) aggregateForStepLocked(stepID string) (int, int64) {
	count := 0
	var total int64
	for _, rec := range s.state.Artifacts {
		if rec.StepID == stepID {
			count++
			total += rec.Reference.SizeBytes
		}
	}
	return count, total
}
