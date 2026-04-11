package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func validateGateAttemptRetryPosture(advisory artifacts.RunnerAdvisoryState, gateAttemptID, gateID, gateKind, gateVersion, planCheckpointCode string, planOrderIndex int, planned compiledRunGatePlan) error {
	if !planned.hasEntries() {
		return nil
	}
	entry, ok := planned.entryFor(gateID, gateKind, gateVersion, planCheckpointCode, planOrderIndex)
	if !ok || gateAttemptID == "" {
		return nil
	}
	if _, exists := advisory.GateAttempts[gateAttemptID]; exists {
		return nil
	}
	if !hasExceededRetryPosture(advisory, entry) {
		return nil
	}
	return fmt.Errorf("gate plan retry posture exceeded: max_attempts=%d for gate %q at %s[%d]", entry.MaxAttempts, entry.GateID, entry.PlanCheckpointCode, entry.PlanOrderIndex)
}

func hasExceededRetryPosture(advisory artifacts.RunnerAdvisoryState, entry runPlannedGateEntry) bool {
	if entry.MaxAttempts <= 0 {
		return false
	}
	seen := map[string]struct{}{}
	for _, existing := range advisory.GateAttempts {
		if existing.GateID != entry.GateID || existing.GateKind != entry.GateKind || existing.GateVersion != entry.GateVersion {
			continue
		}
		if existing.PlanCheckpoint != entry.PlanCheckpointCode || existing.PlanOrderIndex != entry.PlanOrderIndex {
			continue
		}
		seen[existing.GateAttemptID] = struct{}{}
	}
	return len(seen) >= entry.MaxAttempts
}

func validateCheckpointGateAttemptMutation(advisory artifacts.RunnerAdvisoryState, report RunnerCheckpointReport) error {
	if report.GateAttemptID == "" {
		return nil
	}
	existing, ok := advisory.GateAttempts[report.GateAttemptID]
	if !ok {
		return nil
	}
	if existing.Terminal {
		return fmt.Errorf("gate_attempt_id %q already terminal; retries must mint a new gate_attempt_id", report.GateAttemptID)
	}
	if existing.GateID != "" && existing.GateID != report.GateID {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_id", report.GateAttemptID)
	}
	if existing.GateKind != "" && existing.GateKind != report.GateKind {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_kind", report.GateAttemptID)
	}
	if existing.GateVersion != "" && existing.GateVersion != report.GateVersion {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_version", report.GateAttemptID)
	}
	return nil
}

func validateResultGateAttemptMutation(advisory artifacts.RunnerAdvisoryState, report RunnerResultReport) error {
	if report.GateAttemptID == "" {
		return nil
	}
	existing, ok := advisory.GateAttempts[report.GateAttemptID]
	if !ok {
		return nil
	}
	if existing.Terminal {
		return fmt.Errorf("gate_attempt_id %q already has terminal result; retries must mint a new gate_attempt_id", report.GateAttemptID)
	}
	if existing.GateID != "" && existing.GateID != report.GateID {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_id", report.GateAttemptID)
	}
	if existing.GateKind != "" && existing.GateKind != report.GateKind {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_kind", report.GateAttemptID)
	}
	if existing.GateVersion != "" && existing.GateVersion != report.GateVersion {
		return fmt.Errorf("gate_attempt_id %q identity mismatch for gate_version", report.GateAttemptID)
	}
	return nil
}

func validateOverrideReferenceAgainstHistory(advisory artifacts.RunnerAdvisoryState, report RunnerResultReport) error {
	if report.GateLifecycleState != "overridden" || report.OverriddenFailedResultRef == "" {
		return nil
	}
	for _, attempt := range advisory.GateAttempts {
		if attempt.ResultRef != report.OverriddenFailedResultRef {
			continue
		}
		return validateOverrideReferenceMatch(attempt, report)
	}
	return fmt.Errorf("overridden_failed_result_ref does not reference known failed gate result")
}

func validateOverrideReferenceMatch(attempt artifacts.RunnerGateHint, report RunnerResultReport) error {
	if attempt.GateState != "failed" {
		return fmt.Errorf("overridden_failed_result_ref must reference a failed gate result")
	}
	if attempt.GateID != report.GateID || attempt.GateKind != report.GateKind || attempt.GateVersion != report.GateVersion {
		return fmt.Errorf("overridden_failed_result_ref must reference matching gate identity")
	}
	if attempt.GateAttemptID == report.GateAttemptID {
		return fmt.Errorf("overridden_failed_result_ref must reference a prior failed gate attempt")
	}
	return nil
}
