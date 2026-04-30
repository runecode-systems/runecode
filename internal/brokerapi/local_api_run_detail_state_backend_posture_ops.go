package brokerapi

import (
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func projectBackendPostureSelectionEvidenceState(state map[string]any, instanceID string, runID string, policyRefs []string, approvals []ApprovalSummary) {
	reducedAssurance := state["backend_kind"] == launcherbackend.BackendKindContainer || state["runtime_posture_degraded"] == true
	if !reducedAssurance {
		return
	}
	evidence := map[string]any{}
	approvalEvidence := backendPostureApprovalEvidence(instanceID, runID, approvals)
	backendPolicyRefs := backendPostureSelectionPolicyRefs(instanceID, runID, policyRefs, approvals, approvalEvidence)
	if len(backendPolicyRefs) > 0 {
		evidence["policy_decision_refs"] = backendPolicyRefs
	}
	if len(approvalEvidence) > 0 {
		evidence["approval"] = approvalEvidence
	}
	if len(evidence) > 0 {
		state["backend_posture_selection_evidence"] = evidence
	}
}

func backendPostureApprovalEvidence(instanceID, runID string, approvals []ApprovalSummary) map[string]any {
	best, ok := bestBackendPostureApproval(instanceID, runID, approvals)
	if !ok {
		best, ok = bestLegacyRunScopedBackendPostureApproval(runID, approvals)
	}
	if !ok {
		return nil
	}
	approval := best
	evidence := map[string]any{"approval_id": approval.ApprovalID}
	if approval.RequestDigest != "" {
		evidence["approval_request_digest"] = approval.RequestDigest
	}
	if approval.DecisionDigest != "" {
		evidence["approval_decision_digest"] = approval.DecisionDigest
	}
	if approval.PolicyDecisionHash != "" {
		evidence["policy_decision_hash"] = approval.PolicyDecisionHash
	}
	if approval.Status != "" {
		evidence["status"] = approval.Status
	}
	return evidence
}

func bestBackendPostureApproval(instanceID, runID string, approvals []ApprovalSummary) (ApprovalSummary, bool) {
	var best ApprovalSummary
	found := false
	for _, approval := range approvals {
		if !isBackendPostureApproval(instanceID, runID, approval) {
			continue
		}
		if !found || approvalEvidencePrecedes(approval, best) {
			best = approval
			found = true
		}
	}
	return best, found
}

func bestLegacyRunScopedBackendPostureApproval(runID string, approvals []ApprovalSummary) (ApprovalSummary, bool) {
	var best ApprovalSummary
	found := false
	for _, approval := range approvals {
		if !isLegacyRunScopedBackendPostureApproval(runID, approval) {
			continue
		}
		if !found || approvalEvidencePrecedes(approval, best) {
			best = approval
			found = true
		}
	}
	return best, found
}

func isBackendPostureApproval(instanceID, runID string, approval ApprovalSummary) bool {
	if approval.BoundScope.ActionKind != policyengine.ActionKindBackendPosture {
		return false
	}
	targetInstanceID := strings.TrimSpace(instanceID)
	approvalInstanceID := strings.TrimSpace(approval.BoundScope.InstanceID)
	if targetInstanceID != "" && approvalInstanceID != "" && approvalInstanceID != targetInstanceID {
		return false
	}
	if targetInstanceID == "" {
		targetInstanceID = approvalInstanceID
	}
	expectedSelectorRunID := instanceControlRunIDForInstanceID(targetInstanceID)
	boundRunID := approval.BoundScope.RunID
	if boundRunID == "" {
		return false
	}
	if strings.HasPrefix(boundRunID, "instance-control:") {
		if expectedSelectorRunID == "" {
			return false
		}
		return boundRunID == expectedSelectorRunID
	}
	return false
}

func isLegacyRunScopedBackendPostureApproval(runID string, approval ApprovalSummary) bool {
	if strings.TrimSpace(runID) == "" {
		return false
	}
	if approval.BoundScope.ActionKind != policyengine.ActionKindBackendPosture {
		return false
	}
	if strings.TrimSpace(approval.BoundScope.InstanceID) != "" {
		return false
	}
	return approval.BoundScope.RunID == runID
}

func backendPostureSelectionPolicyRefs(instanceID, runID string, policyRefs []string, approvals []ApprovalSummary, approvalEvidence map[string]any) []string {
	relevant := backendPostureRelevantPolicyRefs(instanceID, runID, approvals)
	appendApprovalPolicyHash(relevant, approvalEvidence)
	return orderedBackendPosturePolicyRefs(policyRefs, relevant)
}

func backendPostureRelevantPolicyRefs(instanceID, runID string, approvals []ApprovalSummary) map[string]struct{} {
	relevant := map[string]struct{}{}
	for _, approval := range approvals {
		if !isBackendPostureApproval(instanceID, runID, approval) {
			continue
		}
		addPolicyRef(relevant, approval.PolicyDecisionHash)
		addPolicyRef(relevant, approval.BoundScope.PolicyDecisionHash)
	}
	return relevant
}

func appendApprovalPolicyHash(relevant map[string]struct{}, approvalEvidence map[string]any) {
	approvalPolicyHash, _ := approvalEvidence["policy_decision_hash"].(string)
	addPolicyRef(relevant, approvalPolicyHash)
}

func orderedBackendPosturePolicyRefs(policyRefs []string, relevant map[string]struct{}) []string {
	if len(relevant) == 0 {
		return nil
	}
	selected := make([]string, 0, len(relevant))
	for _, ref := range policyRefs {
		trimmed := strings.TrimSpace(ref)
		if _, ok := relevant[trimmed]; !ok {
			continue
		}
		selected = append(selected, trimmed)
		delete(relevant, trimmed)
	}
	for ref := range relevant {
		selected = append(selected, ref)
	}
	sort.Strings(selected)
	return selected
}

func addPolicyRef(values map[string]struct{}, ref string) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return
	}
	values[trimmed] = struct{}{}
}

func approvalEvidencePrecedes(candidate ApprovalSummary, existing ApprovalSummary) bool {
	if approvalEvidenceStatusRank(candidate.Status) != approvalEvidenceStatusRank(existing.Status) {
		return approvalEvidenceStatusRank(candidate.Status) < approvalEvidenceStatusRank(existing.Status)
	}
	if candidate.RequestedAt != existing.RequestedAt {
		return candidate.RequestedAt > existing.RequestedAt
	}
	return candidate.ApprovalID > existing.ApprovalID
}

func approvalEvidenceStatusRank(status string) int {
	switch status {
	case "consumed":
		return 0
	case "approved":
		return 1
	case "pending":
		return 2
	case "superseded":
		return 3
	case "denied":
		return 4
	case "expired":
		return 5
	case "cancelled":
		return 6
	default:
		return 7
	}
}

func approvalEvidenceSatisfiesReducedAssurance(status string) bool {
	trimmedStatus := strings.TrimSpace(status)
	return trimmedStatus == "consumed" || trimmedStatus == "approved"
}
