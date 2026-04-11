package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func buildRunnerCheckpointAdvisory(report RunnerCheckpointReport, occurred time.Time, details map[string]any) artifacts.RunnerCheckpointAdvisory {
	return artifacts.RunnerCheckpointAdvisory{
		LifecycleState:   report.LifecycleState,
		CheckpointCode:   report.CheckpointCode,
		OccurredAt:       occurred.UTC(),
		IdempotencyKey:   report.IdempotencyKey,
		PlanCheckpoint:   report.PlanCheckpointCode,
		PlanOrderIndex:   report.PlanOrderIndex,
		GateID:           report.GateID,
		GateKind:         report.GateKind,
		GateVersion:      report.GateVersion,
		GateState:        report.GateLifecycleState,
		StageID:          report.StageID,
		StepID:           report.StepID,
		RoleInstanceID:   report.RoleInstanceID,
		StageAttemptID:   report.StageAttemptID,
		StepAttemptID:    report.StepAttemptID,
		GateAttemptID:    report.GateAttemptID,
		GateEvidenceRef:  strings.TrimSpace(report.GateEvidenceRef),
		NormalizedInputs: append([]string{}, report.NormalizedInputDigests...),
		PendingApprovals: report.PendingApprovalCount,
		Details:          details,
	}
}

func buildRunnerResultAdvisory(report RunnerResultReport, occurred time.Time, details map[string]any, gateEvidenceRef string, gateResultRef string, overrideActionHash string, overridePolicyRef string) artifacts.RunnerResultAdvisory {
	return artifacts.RunnerResultAdvisory{
		LifecycleState:     report.LifecycleState,
		ResultCode:         report.ResultCode,
		OccurredAt:         occurred.UTC(),
		IdempotencyKey:     report.IdempotencyKey,
		PlanCheckpoint:     report.PlanCheckpointCode,
		PlanOrderIndex:     report.PlanOrderIndex,
		GateID:             report.GateID,
		GateKind:           report.GateKind,
		GateVersion:        report.GateVersion,
		GateState:          report.GateLifecycleState,
		StageID:            report.StageID,
		StepID:             report.StepID,
		RoleInstanceID:     report.RoleInstanceID,
		StageAttemptID:     report.StageAttemptID,
		StepAttemptID:      report.StepAttemptID,
		GateAttemptID:      report.GateAttemptID,
		NormalizedInputs:   append([]string{}, report.NormalizedInputDigests...),
		FailureReasonCode:  report.FailureReasonCode,
		OverrideFailedRef:  report.OverriddenFailedResultRef,
		OverrideActionHash: overrideActionHash,
		OverridePolicyRef:  overridePolicyRef,
		ResultRef:          gateResultRef,
		GateEvidenceRef:    gateEvidenceRef,
		Details:            details,
	}
}

func canonicalGateResultRef(runID string, report RunnerResultReport, gateEvidenceRef string) (string, error) {
	if !hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) {
		return "", nil
	}
	payload := gateResultRefPayload(strings.TrimSpace(runID), report, strings.TrimSpace(gateEvidenceRef))
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal gate result ref payload: %w", err)
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "", fmt.Errorf("canonicalize gate result ref payload: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func gateResultRefPayload(runID string, report RunnerResultReport, gateEvidenceRef string) map[string]any {
	payload := map[string]any{
		"schema_id":                    "runecode.protocol.v0.GateResultReport",
		"schema_version":               "0.1.0",
		"run_id":                       runID,
		"gate_id":                      strings.TrimSpace(report.GateID),
		"gate_kind":                    strings.TrimSpace(report.GateKind),
		"gate_version":                 strings.TrimSpace(report.GateVersion),
		"gate_attempt_id":              strings.TrimSpace(report.GateAttemptID),
		"lifecycle_state":              strings.TrimSpace(report.GateLifecycleState),
		"result_code":                  strings.TrimSpace(report.ResultCode),
		"occurred_at":                  strings.TrimSpace(report.OccurredAt),
		"idempotency_key":              strings.TrimSpace(report.IdempotencyKey),
		"stage_id":                     strings.TrimSpace(report.StageID),
		"step_id":                      strings.TrimSpace(report.StepID),
		"role_instance_id":             strings.TrimSpace(report.RoleInstanceID),
		"stage_attempt_id":             strings.TrimSpace(report.StageAttemptID),
		"step_attempt_id":              strings.TrimSpace(report.StepAttemptID),
		"failure_reason_code":          strings.TrimSpace(report.FailureReasonCode),
		"overridden_failed_result_ref": strings.TrimSpace(report.OverriddenFailedResultRef),
		"gate_evidence_ref":            gateEvidenceRef,
	}
	if strings.TrimSpace(report.PlanCheckpointCode) != "" {
		payload["plan_checkpoint_code"] = strings.TrimSpace(report.PlanCheckpointCode)
		payload["plan_order_index"] = report.PlanOrderIndex
	}
	if len(report.NormalizedInputDigests) > 0 {
		payload["normalized_input_digests"] = append([]string{}, report.NormalizedInputDigests...)
	}
	return payload
}
