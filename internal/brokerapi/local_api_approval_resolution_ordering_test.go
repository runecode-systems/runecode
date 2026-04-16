package brokerapi

import (
	"strconv"
	"strings"
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

func TestPrefersStageBindingCandidatePlanBindingPrecedence(t *testing.T) {
	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	noPlan := latestStageBinding{approvalID: "sha256:111", requestedAt: baseTime, revision: 1, hasRevision: true}
	withPlan := latestStageBinding{approvalID: "sha256:222", requestedAt: baseTime, planID: "plan-a", revision: 1, hasRevision: true}
	if !prefersStageBindingCandidate(withPlan, noPlan) {
		t.Fatal("expected candidate with plan_id to win over unscoped plan binding")
	}
	if prefersStageBindingCandidate(noPlan, withPlan) {
		t.Fatal("expected unscoped candidate not to win over plan-scoped candidate")
	}

	planALater := latestStageBinding{approvalID: "sha256:333", requestedAt: baseTime.Add(time.Minute), planID: "plan-a", revision: 1, hasRevision: true}
	planBEarlier := latestStageBinding{approvalID: "sha256:444", requestedAt: baseTime, planID: "plan-b", revision: 1, hasRevision: true}
	if !prefersStageBindingCandidate(planALater, planBEarlier) {
		t.Fatal("expected later requested_at to win when plan bindings differ")
	}
}

func TestParseNonNegativeSummaryRevisionAcceptsIntAtMaxSafeInteger(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("int max cannot represent max safe integer on 32-bit")
	}
	maxSafe := int64(9007199254740991)
	revision, err := parseNonNegativeSummaryRevision(int(maxSafe))
	if err != nil {
		t.Fatalf("parseNonNegativeSummaryRevision() error = %v, want nil", err)
	}
	if revision != maxSafe {
		t.Fatalf("parseNonNegativeSummaryRevision() = %d, want %d", revision, maxSafe)
	}
}

func TestParseNonNegativeSummaryRevisionRejectsIntBeyondMaxSafeInteger(t *testing.T) {
	_, err := parseNonNegativeSummaryRevision(float64(9007199254740992))
	if err == nil || !strings.Contains(err.Error(), "must be a non-negative integer") {
		t.Fatalf("parseNonNegativeSummaryRevision() error = %v, want non-negative integer error", err)
	}
}

func TestParseNonNegativeSummaryRevisionAcceptsZero(t *testing.T) {
	revision, err := parseNonNegativeSummaryRevision(0)
	if err != nil {
		t.Fatalf("parseNonNegativeSummaryRevision() error = %v, want nil", err)
	}
	if revision != 0 {
		t.Fatalf("parseNonNegativeSummaryRevision() = %d, want 0", revision)
	}
}
