package artifacts

import (
	"fmt"
	"strings"
	"time"
)

func reconcileConsumedGateOverrideApprovalsLocked(state *StoreState) bool {
	if state == nil || len(state.RunnerAdvisoryByRun) == 0 || len(state.Approvals) == 0 {
		return false
	}
	changed := false
	for runID, advisory := range state.RunnerAdvisoryByRun {
		for _, gateAttempt := range advisory.GateAttempts {
			if reconcileConsumedGateOverrideApprovalForAttempt(state, runID, gateAttempt) {
				changed = true
			}
		}
	}
	if changed {
		rebuildRunApprovalRefsLocked(state)
	}
	return changed
}

func reconcileConsumedGateOverrideApprovalForAttempt(state *StoreState, runID string, gateAttempt RunnerGateHint) bool {
	policyRef := strings.TrimSpace(gateAttempt.OverridePolicyRef)
	if policyRef == "" || strings.TrimSpace(gateAttempt.GateState) != "overridden" {
		return false
	}
	for approvalID, approval := range state.Approvals {
		if !matchesApprovedGateOverrideApprovalRecord(approval, runID, policyRef) {
			continue
		}
		state.Approvals[approvalID] = consumedGateOverrideApproval(approval, gateAttemptFinishedAt(gateAttempt))
		return true
	}
	return false
}

func (s *Store) consumeGateOverrideApprovalLocked(runID, policyDecisionRef string, result RunnerResultAdvisory) (string, ApprovalRecord, bool, error) {
	if strings.TrimSpace(policyDecisionRef) == "" {
		return "", ApprovalRecord{}, false, nil
	}
	for approvalID, approval := range s.state.Approvals {
		if !matchesApprovedGateOverrideApprovalRecord(approval, runID, policyDecisionRef) {
			continue
		}
		prior := approval
		now := result.OccurredAt.UTC()
		approval.Status = "consumed"
		approval.DecidedAt = &now
		approval.ConsumedAt = &now
		s.state.Approvals[approvalID] = approval
		rebuildRunApprovalRefsLocked(&s.state)
		return approvalID, prior, true, nil
	}
	return "", ApprovalRecord{}, false, fmt.Errorf("gate override requires explicit approved approval")
}

func matchesApprovedGateOverrideApprovalRecord(approval ApprovalRecord, runID, policyDecisionRef string) bool {
	if strings.TrimSpace(approval.RunID) != strings.TrimSpace(runID) {
		return false
	}
	if strings.TrimSpace(approval.ActionKind) != "action_gate_override" {
		return false
	}
	if strings.TrimSpace(approval.PolicyDecisionHash) != strings.TrimSpace(policyDecisionRef) {
		return false
	}
	return strings.TrimSpace(approval.Status) == "approved"
}

func gateAttemptFinishedAt(gateAttempt RunnerGateHint) time.Time {
	when := gateAttempt.FinishedAt.UTC()
	if when.IsZero() {
		when = gateAttempt.LastUpdatedAt.UTC()
	}
	return when
}

func consumedGateOverrideApproval(approval ApprovalRecord, when time.Time) ApprovalRecord {
	approval.Status = "consumed"
	approval.DecidedAt = &when
	approval.ConsumedAt = &when
	return approval
}

func applyApprovalWait(state *RunnerAdvisoryState, approval RunnerApproval) {
	ensureApprovalWaitMap(state)
	approvalID := strings.TrimSpace(approval.ApprovalID)
	approval.Status = strings.TrimSpace(approval.Status)
	if approval.Status == "pending" {
		supersedePendingApprovals(state, approvalID, approval)
	}
	state.ApprovalWaits[approvalID] = approval
}

func ensureApprovalWaitMap(state *RunnerAdvisoryState) {
	if state.ApprovalWaits == nil {
		state.ApprovalWaits = map[string]RunnerApproval{}
	}
}

func supersedePendingApprovals(state *RunnerAdvisoryState, incomingApprovalID string, incoming RunnerApproval) {
	for existingID, existing := range state.ApprovalWaits {
		if !shouldSupersedeApproval(existingID, incomingApprovalID, existing, incoming) {
			continue
		}
		state.ApprovalWaits[existingID] = supersededApproval(existing, incomingApprovalID, incoming.OccurredAt.UTC())
	}
}

func shouldSupersedeApproval(existingID, incomingApprovalID string, existing, incoming RunnerApproval) bool {
	if existingID == incomingApprovalID || existing.Status != "pending" {
		return false
	}
	return runnerApprovalSupersedesByIdentity(existing, incoming)
}

func supersededApproval(approval RunnerApproval, supersededBy string, when time.Time) RunnerApproval {
	approval.Status = "superseded"
	approval.SupersededByApproval = supersededBy
	approval.ResolvedAt = &when
	return approval
}

func runnerApprovalSupersedesByIdentity(current, incoming RunnerApproval) bool {
	if current.ApprovalType != incoming.ApprovalType {
		return false
	}
	if current.RunID != incoming.RunID || current.StageID != incoming.StageID || current.StepID != incoming.StepID || current.RoleInstanceID != incoming.RoleInstanceID {
		return false
	}
	if incoming.ApprovalType == "exact_action" {
		return strings.TrimSpace(current.BoundActionHash) != "" && current.BoundActionHash == incoming.BoundActionHash
	}
	if incoming.ApprovalType == "stage_sign_off" {
		return strings.TrimSpace(current.BoundStageSummaryHash) != "" && current.BoundStageSummaryHash == incoming.BoundStageSummaryHash
	}
	return false
}
