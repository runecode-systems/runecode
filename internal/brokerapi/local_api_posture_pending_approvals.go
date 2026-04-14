package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) recordPendingBackendPostureApproval(decision policyengine.PolicyDecision, _ BackendPostureChangeRequest, action policyengine.ActionRequest) (string, error) {
	selectorRunID := instanceControlSelectorRunIDForDecision(action, decision)
	if selectorRunID == "" {
		return "", fmt.Errorf("instance-control selector run_id is required for backend posture approvals")
	}
	policyDigest := strings.TrimSpace(decisionDigestIdentity(decision))
	if policyDigest == "" {
		return "", fmt.Errorf("backend posture policy decision hash unavailable")
	}
	best := latestMatchingPendingBackendPostureApproval(s.ApprovalList(), decision, policyDigest, selectorRunID)
	if best == nil {
		return "", fmt.Errorf("derived pending backend-posture approval not found")
	}
	return best.ApprovalID, nil
}

func latestMatchingPendingBackendPostureApproval(records []artifacts.ApprovalRecord, decision policyengine.PolicyDecision, policyDigest string, selectorRunID string) *artifacts.ApprovalRecord {
	var best *artifacts.ApprovalRecord
	for _, rec := range records {
		if !matchesPendingBackendPostureApproval(rec, decision, policyDigest, selectorRunID) {
			continue
		}
		current := rec
		if best == nil || current.RequestedAt.After(best.RequestedAt) {
			best = &current
		}
	}
	return best
}

func matchesPendingBackendPostureApproval(rec artifacts.ApprovalRecord, decision policyengine.PolicyDecision, policyDigest string, selectorRunID string) bool {
	if rec.Status != "pending" || rec.ActionKind != policyengine.ActionKindBackendPosture {
		return false
	}
	if strings.TrimSpace(rec.RunID) != strings.TrimSpace(selectorRunID) {
		return false
	}
	if strings.TrimSpace(rec.ActionRequestHash) != strings.TrimSpace(decision.ActionRequestHash) {
		return false
	}
	if strings.TrimSpace(rec.ManifestHash) != strings.TrimSpace(decision.ManifestHash) {
		return false
	}
	if strings.TrimSpace(rec.PolicyDecisionHash) != strings.TrimSpace(policyDigest) {
		return false
	}
	instanceID := requiredApprovalScopeString(decision.RequiredApproval, "instance_id")
	if instanceID != "" && strings.TrimSpace(rec.InstanceID) != instanceID {
		return false
	}
	return true
}
