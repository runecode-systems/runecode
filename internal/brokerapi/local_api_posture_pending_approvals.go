package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) recordPendingBackendPostureApproval(decision policyengine.PolicyDecision, _ BackendPostureChangeRequest, _ policyengine.ActionRequest) (string, error) {
	runID := strings.TrimSpace(scopeString(decision.RequiredApproval, "run_id"))
	if err := s.RecordPolicyDecision(runID, "", decision); err != nil {
		return "", err
	}
	policyDigest := latestPolicyDecisionRefForRun(s, runID)
	best := latestMatchingPendingBackendPostureApproval(s.ApprovalList(), decision, policyDigest)
	if best == nil {
		return "", fmt.Errorf("derived pending backend-posture approval not found")
	}
	return best.ApprovalID, nil
}

func latestPolicyDecisionRefForRun(s *Service, runID string) string {
	refs := s.PolicyDecisionRefsForRun(runID)
	if len(refs) == 0 {
		return ""
	}
	return strings.TrimSpace(refs[len(refs)-1])
}

func latestMatchingPendingBackendPostureApproval(records []artifacts.ApprovalRecord, decision policyengine.PolicyDecision, policyDigest string) *artifacts.ApprovalRecord {
	var best *artifacts.ApprovalRecord
	for _, rec := range records {
		if !matchesPendingBackendPostureApproval(rec, decision, policyDigest) {
			continue
		}
		current := rec
		if best == nil || current.RequestedAt.After(best.RequestedAt) {
			best = &current
		}
	}
	return best
}

func matchesPendingBackendPostureApproval(rec artifacts.ApprovalRecord, decision policyengine.PolicyDecision, policyDigest string) bool {
	if rec.Status != "pending" || rec.ActionKind != policyengine.ActionKindBackendPosture {
		return false
	}
	if strings.TrimSpace(rec.ActionRequestHash) != strings.TrimSpace(decision.ActionRequestHash) {
		return false
	}
	if strings.TrimSpace(rec.ManifestHash) != strings.TrimSpace(decision.ManifestHash) {
		return false
	}
	if policyDigest != "" && strings.TrimSpace(rec.PolicyDecisionHash) != policyDigest {
		return false
	}
	return true
}

func scopeString(required map[string]any, key string) string {
	scope, _ := required["scope"].(map[string]any)
	value, _ := scope[key].(string)
	return strings.TrimSpace(value)
}
