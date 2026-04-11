package brokerapi

import (
	"testing"
	"time"
)

func TestParseRequestedAtNormalizesRFC3339Offsets(t *testing.T) {
	parsed := parseRequestedAt("2026-04-10T12:00:00+00:00")
	want := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if !parsed.Equal(want) {
		t.Fatalf("parseRequestedAt mismatch: got %s want %s", parsed.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestPrefersStageBindingCandidateUsesTimestampThenApprovalID(t *testing.T) {
	earliest := latestStageBinding{
		approvalID:  "sha256:111",
		requestedAt: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		revision:    2,
		hasRevision: true,
	}
	later := latestStageBinding{
		approvalID:  "sha256:222",
		requestedAt: time.Date(2026, 4, 10, 12, 1, 0, 0, time.UTC),
		revision:    2,
		hasRevision: true,
	}
	if !prefersStageBindingCandidate(later, earliest) {
		t.Fatal("expected later requested_at candidate to be preferred")
	}

	tieLow := latestStageBinding{
		approvalID:  "sha256:aaa",
		requestedAt: time.Date(2026, 4, 10, 12, 2, 0, 0, time.UTC),
		revision:    3,
		hasRevision: true,
	}
	tieHigh := latestStageBinding{
		approvalID:  "sha256:bbb",
		requestedAt: time.Date(2026, 4, 10, 12, 2, 0, 0, time.UTC),
		revision:    3,
		hasRevision: true,
	}
	if !prefersStageBindingCandidate(tieHigh, tieLow) {
		t.Fatal("expected higher approval_id to win timestamp tie")
	}
}
