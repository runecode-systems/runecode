package brokerapi

import (
	"context"
	"strings"
	"testing"
)

func TestApprovalSummaryPolicyDecisionHashUsesPersistedPolicyDecisionDigest(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	unapproved := putUnapprovedExcerptArtifactForApprovalTest(t, s, "run-policy-hash", "step-1", "d")
	createPendingApprovalFromPolicyDecision(t, s, "run-policy-hash", "step-1", unapproved.Digest)
	approval := requireSingleApprovalSummaryForRun(t, s, "run-policy-hash")
	assertApprovalUsesPersistedPolicyDecisionHash(t, s, "run-policy-hash", approval)
}

func requireSingleApprovalSummaryForRun(t *testing.T, s *Service, runID string) ApprovalSummary {
	t.Helper()
	listResp, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-approval-list-policy-hash",
		RunID:         runID,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalList error response: %+v", errResp)
	}
	if len(listResp.Approvals) != 1 {
		t.Fatalf("approval count = %d, want 1", len(listResp.Approvals))
	}
	return listResp.Approvals[0]
}

func assertApprovalUsesPersistedPolicyDecisionHash(t *testing.T, s *Service, runID string, approval ApprovalSummary) {
	t.Helper()
	if strings.TrimSpace(approval.PolicyDecisionHash) == "" {
		t.Fatal("approval policy_decision_hash should be populated from persisted policy decision digest")
	}
	if approval.PolicyDecisionHash != approval.BoundScope.PolicyDecisionHash {
		t.Fatalf("summary policy_decision_hash = %q, bound_scope policy_decision_hash = %q, want equal", approval.PolicyDecisionHash, approval.BoundScope.PolicyDecisionHash)
	}
	if approval.DecisionDigest != "" {
		t.Fatalf("pending approval decision_digest = %q, want empty", approval.DecisionDigest)
	}
	refs := s.PolicyDecisionRefsForRun(runID)
	if len(refs) == 0 {
		t.Fatal("expected persisted policy decision refs for run")
	}
	for _, ref := range refs {
		if ref == approval.PolicyDecisionHash {
			return
		}
	}
	t.Fatalf("approval policy_decision_hash %q not found in persisted policy decision refs %v", approval.PolicyDecisionHash, refs)
}
