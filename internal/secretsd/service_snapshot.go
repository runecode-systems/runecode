package secretsd

import "time"

func (s *Service) RuntimeSnapshot() RuntimeSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	snapshot := baseRuntimeSnapshot(s.state.Metrics, len(s.state.Secrets), now)
	latestMutation := latestSecretMutation(s.state.Secrets)
	leaseSummary, leaseMutation := summarizeLeaseSnapshot(s.state.Leases, now)
	snapshot.ActiveLeaseCount = leaseSummary.active
	snapshot.ExpiredLeaseCount = leaseSummary.expired
	snapshot.RevokedLeaseCount = leaseSummary.revoked
	latestMutation = maxTime(latestMutation, leaseMutation)
	if latestMutation.IsZero() {
		latestMutation = now
	}
	snapshot.LastUpdatedAt = latestMutation.UTC()
	return snapshot
}

func baseRuntimeSnapshot(m metrics, secretCount int, now time.Time) RuntimeSnapshot {
	return RuntimeSnapshot{
		LeaseIssueCount:   m.LeaseIssueCount,
		LeaseRenewCount:   m.LeaseRenewCount,
		LeaseRevokeCount:  m.LeaseRevokeCount,
		LeaseDenyCount:    m.LeaseDenyCount,
		SecretRecordCount: secretCount,
		LastRecoveredAt:   now,
	}
}

func latestSecretMutation(records map[string]secretRecord) time.Time {
	latest := time.Time{}
	for _, rec := range records {
		latest = maxTime(latest, rec.UpdatedAt)
	}
	return latest
}

type leaseSnapshot struct {
	active  int
	expired int
	revoked int
}

func summarizeLeaseSnapshot(leases map[string]leaseRecord, now time.Time) (leaseSnapshot, time.Time) {
	summary := leaseSnapshot{}
	latest := time.Time{}
	for _, lease := range leases {
		latest = maxTime(latest, lease.IssuedAt)
		if lease.RevokedAt != nil {
			latest = maxTime(latest, *lease.RevokedAt)
		}
		switch lease.Status {
		case leaseStatusRevoked:
			summary.revoked++
		case leaseStatusActive:
			if lease.ExpiresAt.After(now) {
				summary.active++
			} else {
				summary.expired++
			}
		}
	}
	return summary, latest
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
