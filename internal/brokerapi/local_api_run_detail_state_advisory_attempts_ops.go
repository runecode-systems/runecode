package brokerapi

import "github.com/runecode-ai/runecode/internal/artifacts"

func buildAdvisoryStepAttemptsState(stepAttempts map[string]artifacts.RunnerStepHint, pendingByScope map[string]int) map[string]any {
	if len(stepAttempts) == 0 {
		return nil
	}
	out := map[string]any{}
	for attemptID, hint := range stepAttempts {
		out[attemptID] = buildAdvisoryStepAttemptEntry(hint, pendingByScope)
	}
	return out
}

func buildAdvisoryStepAttemptEntry(hint artifacts.RunnerStepHint, pendingByScope map[string]int) map[string]any {
	entry := map[string]any{
		"step_attempt_id":      hint.StepAttemptID,
		"run_id":               hint.RunID,
		"gate_id":              hint.GateID,
		"gate_kind":            hint.GateKind,
		"gate_version":         hint.GateVersion,
		"gate_lifecycle_state": hint.GateState,
		"stage_id":             hint.StageID,
		"step_id":              hint.StepID,
		"role_instance_id":     hint.RoleInstanceID,
		"stage_attempt_id":     hint.StageAttemptID,
		"gate_attempt_id":      hint.GateAttemptID,
		"gate_evidence_ref":    hint.GateEvidenceRef,
		"status":               hint.Status,
		"last_updated_at":      hint.LastUpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if !hint.StartedAt.IsZero() {
		entry["started_at"] = hint.StartedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if !hint.FinishedAt.IsZero() {
		entry["finished_at"] = hint.FinishedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if hint.CurrentPhase != "" {
		entry["current_phase"] = hint.CurrentPhase
	}
	if hint.PhaseStatus != "" {
		entry["phase_status"] = hint.PhaseStatus
	}
	scopeKey := approvalScopeKey(artifacts.RunnerApproval{RunID: hint.RunID, StageID: hint.StageID, StepID: hint.StepID, RoleInstanceID: hint.RoleInstanceID})
	if pending := pendingByScope[scopeKey]; pending > 0 {
		entry["blocked_on_scope_pending_approval"] = true
		entry["pending_approval_scope_count"] = pending
		return entry
	}
	entry["blocked_on_scope_pending_approval"] = false
	return entry
}

func buildAdvisoryGateAttemptsState(gateAttempts map[string]artifacts.RunnerGateHint) map[string]any {
	if len(gateAttempts) == 0 {
		return nil
	}
	out := map[string]any{}
	for attemptID, hint := range gateAttempts {
		entry := map[string]any{
			"gate_attempt_id":              hint.GateAttemptID,
			"run_id":                       hint.RunID,
			"plan_checkpoint_code":         hint.PlanCheckpoint,
			"plan_order_index":             hint.PlanOrderIndex,
			"gate_id":                      hint.GateID,
			"gate_kind":                    hint.GateKind,
			"gate_version":                 hint.GateVersion,
			"gate_lifecycle_state":         hint.GateState,
			"stage_id":                     hint.StageID,
			"step_id":                      hint.StepID,
			"role_instance_id":             hint.RoleInstanceID,
			"stage_attempt_id":             hint.StageAttemptID,
			"step_attempt_id":              hint.StepAttemptID,
			"gate_evidence_ref":            hint.GateEvidenceRef,
			"gate_result_ref":              hint.ResultRef,
			"failure_reason_code":          hint.FailureReasonCode,
			"overridden_failed_result_ref": hint.OverrideFailedRef,
			"override_action_request_hash": hint.OverrideActionHash,
			"override_policy_decision_ref": hint.OverridePolicyRef,
			"result_code":                  hint.ResultCode,
			"terminal":                     hint.Terminal,
			"last_updated_at":              hint.LastUpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
		if !hint.StartedAt.IsZero() {
			entry["started_at"] = hint.StartedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		if !hint.FinishedAt.IsZero() {
			entry["finished_at"] = hint.FinishedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		out[attemptID] = entry
	}
	return out
}
