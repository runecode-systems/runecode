package brokerapi

import "github.com/runecode-ai/runecode/internal/artifacts"

func buildAdvisoryRunState(advisory artifacts.RunnerAdvisoryState) map[string]any {
	state := map[string]any{
		"source":       "runner_advisory",
		"provenance":   "none_reported",
		"available":    false,
		"bounded_keys": []string{},
	}
	bounded := make([]string, 0, 6)
	pendingByScope := pendingApprovalScopeCounts(advisory.ApprovalWaits)
	if len(pendingByScope) > 0 {
		state["pending_approval_scope_counts"] = pendingByScope
		bounded = append(bounded, "pending_approval_scope_counts")
	}
	if checkpointState := buildAdvisoryLastCheckpointState(advisory.LastCheckpoint); checkpointState != nil {
		state["last_checkpoint"] = checkpointState
		bounded = append(bounded, "last_checkpoint")
	}
	if resultState := buildAdvisoryLastResultState(advisory.LastResult); resultState != nil {
		state["last_result"] = resultState
		bounded = append(bounded, "last_result")
	}
	if lifecycleHint := buildAdvisoryLifecycleHintState(advisory.Lifecycle); lifecycleHint != nil {
		state["lifecycle_hint"] = lifecycleHint
		bounded = append(bounded, "lifecycle_hint")
	}
	if stepAttempts := buildAdvisoryStepAttemptsState(advisory.StepAttempts, pendingByScope); len(stepAttempts) > 0 {
		state["step_attempts"] = stepAttempts
		bounded = append(bounded, "step_attempts")
	}
	if gateAttempts := buildAdvisoryGateAttemptsState(advisory.GateAttempts); len(gateAttempts) > 0 {
		state["gate_attempts"] = gateAttempts
		bounded = append(bounded, "gate_attempts")
	}
	if len(advisory.ApprovalWaits) > 0 {
		state["approval_waits"] = redactedApprovalWaits(advisory.ApprovalWaits)
		bounded = append(bounded, "approval_waits")
	}
	markAdvisoryAvailability(state, bounded)
	return state
}

func pendingApprovalScopeCounts(waits map[string]artifacts.RunnerApproval) map[string]int {
	counts := map[string]int{}
	for _, approval := range waits {
		if approval.Status != "pending" {
			continue
		}
		counts[approvalScopeKey(approval)]++
	}
	return counts
}

func markAdvisoryAvailability(state map[string]any, bounded []string) {
	if len(bounded) == 0 {
		return
	}
	state["available"] = true
	state["provenance"] = "runner_reported"
	state["bounded_keys"] = bounded
}

func redactedApprovalWaits(waits map[string]artifacts.RunnerApproval) map[string]artifacts.RunnerApproval {
	out := make(map[string]artifacts.RunnerApproval, len(waits))
	for approvalID, wait := range waits {
		copyWait := wait
		copyWait.BoundActionHash = ""
		copyWait.BoundStageSummaryHash = ""
		out[approvalID] = copyWait
	}
	return out
}

func approvalScopeKey(approval artifacts.RunnerApproval) string {
	return approval.RunID + "|" + approval.StageID + "|" + approval.StepID + "|" + approval.RoleInstanceID
}
