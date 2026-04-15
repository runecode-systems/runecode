package brokerapi

import (
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestLatestAnchorPolicyDecisionByActionHashUsesNewestRecordedDecision(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	actionHash := "sha256:" + strings.Repeat("1", 64)
	olderTime := time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC)
	newerTime := time.Date(2026, time.March, 13, 12, 1, 0, 0, time.UTC)
	mustRecordAnchorPolicyDecisionAtTime(t, service, actionHash, policyengine.DecisionAllow, olderTime)
	mustRecordAnchorPolicyDecisionAtTime(t, service, actionHash, policyengine.DecisionRequireHumanApproval, newerTime)
	ref, record, ok := service.latestAnchorPolicyDecisionByActionHash(actionHash)
	if !ok {
		t.Fatal("latestAnchorPolicyDecisionByActionHash found=false, want true")
	}
	if got := strings.TrimSpace(ref); got == "" {
		t.Fatal("policy decision ref empty, want non-empty")
	}
	if got := strings.TrimSpace(record.DecisionOutcome); got != string(policyengine.DecisionRequireHumanApproval) {
		t.Fatalf("decision_outcome = %q, want %q", got, policyengine.DecisionRequireHumanApproval)
	}
	if got := record.RecordedAt.UTC(); !got.Equal(newerTime) {
		t.Fatalf("recorded_at = %s, want %s", got.Format(time.RFC3339), newerTime.Format(time.RFC3339))
	}
}

func TestShouldReplaceAnchorPolicyDecisionUsesStableRefTieBreak(t *testing.T) {
	base := time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC)
	current := mustPolicyDecisionRecordForSelectionTest(base, 10)
	candidate := mustPolicyDecisionRecordForSelectionTest(base, 10)
	currentRef := "sha256:" + strings.Repeat("b", 64)
	candidateRef := "sha256:" + strings.Repeat("a", 64)
	if !shouldReplaceAnchorPolicyDecision(currentRef, current, candidateRef, candidate) {
		t.Fatal("shouldReplaceAnchorPolicyDecision=false, want true for lexicographically smaller ref tie-break")
	}
	if shouldReplaceAnchorPolicyDecision(candidateRef, current, currentRef, candidate) {
		t.Fatal("shouldReplaceAnchorPolicyDecision=true, want false for lexicographically larger ref tie-break")
	}
}

func TestShouldReplaceAnchorPolicyDecisionUnknownOutcomeWinsTieBreakFailClosed(t *testing.T) {
	base := time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC)
	known := artifacts.PolicyDecisionRecord{DecisionOutcome: string(policyengine.DecisionDeny), RecordedAt: base, AuditEventSeq: 10}
	unknown := artifacts.PolicyDecisionRecord{DecisionOutcome: "future_unknown_outcome", RecordedAt: base, AuditEventSeq: 10}
	if !shouldReplaceAnchorPolicyDecision("sha256:aaaa", known, "sha256:bbbb", unknown) {
		t.Fatal("shouldReplaceAnchorPolicyDecision=false, want true when unknown outcome competes with known deny")
	}
}

func mustPolicyDecisionRecordForSelectionTest(when time.Time, seq int64) artifacts.PolicyDecisionRecord {
	return artifacts.PolicyDecisionRecord{RecordedAt: when, AuditEventSeq: seq}
}

func mustRecordAnchorPolicyDecisionAtTime(t *testing.T, service *Service, actionHash string, outcome policyengine.DecisionOutcome, when time.Time) string {
	t.Helper()
	decision := seededAnchorPolicyDecision(actionHash, outcome)
	if outcome == policyengine.DecisionRequireHumanApproval {
		configureSeededAnchorRequiredApproval(&decision, "reauthenticated")
	}
	service.store.SetNowFuncForTests(func() time.Time { return when })
	if err := service.RecordPolicyDecision(anchorApprovalPolicySelectorRunID, "", decision); err != nil {
		t.Fatalf("RecordPolicyDecision(%s) returned error: %v", outcome, err)
	}
	service.store.SetNowFuncForTests(nil)
	return mustFindAnchorPolicyDecisionRefForActionHash(t, service, actionHash)
}
