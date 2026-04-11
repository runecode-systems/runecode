package brokerapi

import "github.com/runecode-ai/runecode/internal/artifacts"

func buildAdvisoryLastCheckpointState(checkpoint *artifacts.RunnerCheckpointAdvisory) map[string]any {
	if checkpoint == nil {
		return nil
	}
	state := map[string]any{
		"lifecycle_state":        checkpoint.LifecycleState,
		"checkpoint_code":        checkpoint.CheckpointCode,
		"occurred_at":            checkpoint.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"idempotency_key":        checkpoint.IdempotencyKey,
		"plan_checkpoint_code":   checkpoint.PlanCheckpoint,
		"plan_order_index":       checkpoint.PlanOrderIndex,
		"gate_id":                checkpoint.GateID,
		"gate_kind":              checkpoint.GateKind,
		"gate_version":           checkpoint.GateVersion,
		"gate_lifecycle_state":   checkpoint.GateState,
		"stage_id":               checkpoint.StageID,
		"step_id":                checkpoint.StepID,
		"role_instance_id":       checkpoint.RoleInstanceID,
		"stage_attempt_id":       checkpoint.StageAttemptID,
		"step_attempt_id":        checkpoint.StepAttemptID,
		"gate_attempt_id":        checkpoint.GateAttemptID,
		"gate_evidence_ref":      checkpoint.GateEvidenceRef,
		"pending_approval_count": checkpoint.PendingApprovals,
	}
	if len(checkpoint.NormalizedInputs) > 0 {
		state["normalized_input_digests"] = append([]string{}, checkpoint.NormalizedInputs...)
	}
	if len(checkpoint.Details) > 0 {
		state["details"] = checkpoint.Details
	}
	return state
}

func buildAdvisoryLastResultState(result *artifacts.RunnerResultAdvisory) map[string]any {
	if result == nil {
		return nil
	}
	state := baseAdvisoryLastResultState(result)
	if len(result.NormalizedInputs) > 0 {
		state["normalized_input_digests"] = append([]string{}, result.NormalizedInputs...)
	}
	applyAdvisoryLastResultOverrides(state, result)
	if len(result.Details) > 0 {
		state["details"] = result.Details
	}
	return state
}

func baseAdvisoryLastResultState(result *artifacts.RunnerResultAdvisory) map[string]any {
	return map[string]any{
		"lifecycle_state":              result.LifecycleState,
		"result_code":                  result.ResultCode,
		"occurred_at":                  result.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"idempotency_key":              result.IdempotencyKey,
		"plan_checkpoint_code":         result.PlanCheckpoint,
		"plan_order_index":             result.PlanOrderIndex,
		"gate_id":                      result.GateID,
		"gate_kind":                    result.GateKind,
		"gate_version":                 result.GateVersion,
		"gate_lifecycle_state":         result.GateState,
		"stage_id":                     result.StageID,
		"step_id":                      result.StepID,
		"role_instance_id":             result.RoleInstanceID,
		"stage_attempt_id":             result.StageAttemptID,
		"step_attempt_id":              result.StepAttemptID,
		"gate_attempt_id":              result.GateAttemptID,
		"gate_evidence_ref":            result.GateEvidenceRef,
		"gate_result_ref":              result.ResultRef,
		"failure_reason_code":          result.FailureReasonCode,
		"override_action_request_hash": result.OverrideActionHash,
		"override_policy_decision_ref": result.OverridePolicyRef,
	}
}

func applyAdvisoryLastResultOverrides(state map[string]any, result *artifacts.RunnerResultAdvisory) {
	if result.OverrideFailedRef != "" {
		state["overridden_failed_result_ref"] = result.OverrideFailedRef
	}
	if result.OverrideActionHash != "" {
		state["override_action_request_hash"] = result.OverrideActionHash
	}
	if result.OverridePolicyRef != "" {
		state["override_policy_decision_ref"] = result.OverridePolicyRef
	}
}

func buildAdvisoryLifecycleHintState(lifecycle *artifacts.RunnerLifecycleHint) map[string]any {
	if lifecycle == nil {
		return nil
	}
	return map[string]any{
		"lifecycle_state":  lifecycle.LifecycleState,
		"occurred_at":      lifecycle.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"stage_id":         lifecycle.StageID,
		"step_id":          lifecycle.StepID,
		"role_instance_id": lifecycle.RoleInstanceID,
		"stage_attempt_id": lifecycle.StageAttemptID,
		"step_attempt_id":  lifecycle.StepAttemptID,
		"gate_attempt_id":  lifecycle.GateAttemptID,
	}
}
