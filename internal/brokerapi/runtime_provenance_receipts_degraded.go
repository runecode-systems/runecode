package brokerapi

import "strings"

func degradedPostureSummaryPayload(runIDs []string, runStatuses map[string]string, approvals []ApprovalSummary) map[string]any {
	payload := defaultDegradedPostureSummaryPayload()
	if len(runIDs) == 0 {
		return payload
	}
	runs := runIDSet(runIDs)
	attachBestRunLifecycle(payload, runIDs, runStatuses)
	if applied := applyOverrideDegradedPosture(payload, runs, approvals); applied {
		return payload
	}
	applyBackendPostureDegradedSummary(payload, runs, approvals)
	return payload
}

func defaultDegradedPostureSummaryPayload() map[string]any {
	return map[string]any{
		"summary_scope_kind":     runtimeSummaryScopeRun,
		"degraded":               false,
		"degradation_cause_code": "none",
		"trust_claim_before":     "standard_assurance",
		"trust_claim_after":      "standard_assurance",
		"changed_trust_claim":    false,
		"user_acknowledged":      false,
		"approval_required":      false,
		"approval_consumed":      false,
		"override_required":      false,
		"override_applied":       false,
	}
}

func runIDSet(runIDs []string) map[string]struct{} {
	runs := map[string]struct{}{}
	for i := range runIDs {
		if runID := strings.TrimSpace(runIDs[i]); runID != "" {
			runs[runID] = struct{}{}
		}
	}
	return runs
}

func attachBestRunLifecycle(payload map[string]any, runIDs []string, runStatuses map[string]string) {
	bestRunID := ""
	for i := range runIDs {
		candidate := strings.TrimSpace(runIDs[i])
		if candidate == "" {
			continue
		}
		if bestRunID == "" || candidate > bestRunID {
			bestRunID = candidate
		}
	}
	if lifecycle := strings.TrimSpace(runStatuses[bestRunID]); lifecycle != "" {
		payload["run_lifecycle_state"] = lifecycle
	}
}

func applyOverrideDegradedPosture(payload map[string]any, runs map[string]struct{}, approvals []ApprovalSummary) bool {
	for _, approval := range approvals {
		if !approvalMatchesRuns(approval, runs) {
			continue
		}
		if strings.TrimSpace(approval.BoundScope.ActionKind) != "action_gate_override" || strings.TrimSpace(approval.Status) != "consumed" {
			continue
		}
		payload["degraded"] = true
		payload["degradation_cause_code"] = "gate_override_applied"
		payload["degradation_reason_codes"] = []string{"gate_override_applied"}
		payload["trust_claim_before"] = "no_override_required"
		payload["trust_claim_after"] = "override_required_or_applied"
		payload["changed_trust_claim"] = true
		payload["user_acknowledged"] = true
		payload["acknowledgment_evidence"] = "approval_consumed"
		payload["approval_required"] = true
		payload["approval_consumed"] = true
		payload["override_required"] = true
		payload["override_applied"] = true
		attachApprovalReferences(payload, approval)
		if actionHash := strings.TrimSpace(approval.ConsumedActionHash); actionHash != "" {
			payload["override_action_request_hash"] = actionHash
		}
		if ref := strings.TrimSpace(approval.PolicyDecisionHash); ref != "" {
			payload["override_policy_decision_ref"] = ref
		}
		return true
	}
	return false
}

func applyBackendPostureDegradedSummary(payload map[string]any, runs map[string]struct{}, approvals []ApprovalSummary) {
	for _, approval := range approvals {
		if !approvalMatchesRuns(approval, runs) {
			continue
		}
		if strings.TrimSpace(approval.BoundScope.ActionKind) != "backend_posture_change" {
			continue
		}
		status := strings.TrimSpace(approval.Status)
		if status != "consumed" && status != "approved" {
			continue
		}
		payload["degraded"] = true
		payload["degradation_cause_code"] = "reduced_assurance_backend_posture"
		payload["degradation_reason_codes"] = []string{"reduced_assurance_backend_posture"}
		payload["trust_claim_before"] = "attested_isolation"
		payload["trust_claim_after"] = "reduced_assurance_container_isolation"
		payload["changed_trust_claim"] = true
		payload["user_acknowledged"] = true
		payload["acknowledgment_evidence"] = "approval_" + status
		payload["approval_required"] = true
		payload["approval_consumed"] = status == "consumed"
		attachApprovalReferences(payload, approval)
		return
	}
}

func approvalMatchesRuns(approval ApprovalSummary, runs map[string]struct{}) bool {
	_, ok := runs[strings.TrimSpace(approval.BoundScope.RunID)]
	return ok
}

func attachApprovalReferences(payload map[string]any, approval ApprovalSummary) {
	if ref := strings.TrimSpace(approval.PolicyDecisionHash); ref != "" {
		payload["approval_policy_decision_ref"] = ref
	}
	if link := strings.TrimSpace(approval.ConsumptionLinkDigest); link != "" {
		payload["approval_consumption_link"] = link
	}
}
