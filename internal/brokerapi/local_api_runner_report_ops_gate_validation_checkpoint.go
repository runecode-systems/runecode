package brokerapi

import "fmt"

func validateGateCheckpointFields(report RunnerCheckpointReport) error {
	if err := validatePlanOrderIndex(report.PlanCheckpointCode, report.PlanOrderIndex); err != nil {
		return err
	}
	if err := validateOptionalGateEvidenceRef(report.GateEvidenceRef, hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests)); err != nil {
		return err
	}
	if !hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) {
		return nil
	}
	if err := validateRequiredGateCheckpointIdentity(report); err != nil {
		return err
	}
	if expected, ok := gateStateForCheckpointCode(report.CheckpointCode); ok && expected != report.GateLifecycleState {
		return fmt.Errorf("checkpoint_code %q requires gate_lifecycle_state %q, got %q", report.CheckpointCode, expected, report.GateLifecycleState)
	}
	return validateNormalizedInputDigests(report.NormalizedInputDigests)
}

func validateGateCheckpointPlanBinding(report RunnerCheckpointReport, planned compiledRunGatePlan) (runPlannedGateEntry, error) {
	if !hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) {
		if err := validateNoPlanBindingWithoutGate(report.PlanCheckpointCode, report.PlanOrderIndex); err != nil {
			return runPlannedGateEntry{}, err
		}
		return runPlannedGateEntry{}, nil
	}
	if !planned.hasEntries() {
		if report.PlanCheckpointCode != "" || report.PlanOrderIndex != 0 {
			return runPlannedGateEntry{}, fmt.Errorf("gate-scoped checkpoint requires trusted run plan gate entries")
		}
		return runPlannedGateEntry{}, nil
	}
	if report.PlanCheckpointCode == "" {
		return runPlannedGateEntry{}, fmt.Errorf("gate-scoped checkpoint requires plan_checkpoint_code")
	}
	entry, ok := planned.entryFor(report.GateID, report.GateKind, report.GateVersion, report.PlanCheckpointCode, report.PlanOrderIndex)
	if !ok {
		return runPlannedGateEntry{}, fmt.Errorf("gate-scoped checkpoint does not match trusted run plan placement")
	}
	if err := validatePlannedInputDigestHooks(entry.ExpectedInputDigests, report.NormalizedInputDigests); err != nil {
		return runPlannedGateEntry{}, err
	}
	return entry, nil
}

func validateRequiredGateCheckpointIdentity(report RunnerCheckpointReport) error {
	if report.GateID == "" || report.GateKind == "" || report.GateVersion == "" || report.GateAttemptID == "" || report.GateLifecycleState == "" {
		return fmt.Errorf("gate-scoped checkpoint requires gate_id, gate_kind, gate_version, gate_attempt_id, and gate_lifecycle_state")
	}
	if !isGateKind(report.GateKind) {
		return fmt.Errorf("unsupported gate_kind %q", report.GateKind)
	}
	if !isGateCheckpointState(report.GateLifecycleState) {
		return fmt.Errorf("unsupported gate_lifecycle_state %q for checkpoint", report.GateLifecycleState)
	}
	return nil
}
