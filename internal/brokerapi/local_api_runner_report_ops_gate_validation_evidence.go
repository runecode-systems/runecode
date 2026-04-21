package brokerapi

import (
	"fmt"
	"time"
)

func validateGateEvidenceBindingFields(report RunnerResultReport) error {
	if err := validateGateEvidencePayloadShape(report.GateEvidence); err != nil {
		return err
	}
	if report.GateEvidenceRef != "" && !isValidDigestIdentity(report.GateEvidenceRef) {
		return fmt.Errorf("gate_evidence_ref must be digest identity")
	}
	if !hasGateBinding(report.GateID, report.GateKind, report.GateVersion, report.GateAttemptID, report.GateLifecycleState, report.NormalizedInputDigests) && report.OverriddenFailedResultRef == "" {
		if report.GateEvidence != nil || report.GateEvidenceRef != "" {
			return fmt.Errorf("gate evidence requires gate identity binding")
		}
	}
	return nil
}

func validateGateEvidencePayloadShape(evidence *GateEvidence) error {
	if evidence == nil {
		return nil
	}
	if err := validateGateEvidenceCoreFields(evidence); err != nil {
		return err
	}
	if err := validateGateEvidenceDigestFields(evidence); err != nil {
		return err
	}
	return validateGateEvidenceOverrideRefs(evidence)
}

func validateGateEvidenceCoreFields(evidence *GateEvidence) error {
	if evidence.SchemaID != "runecode.protocol.v0.GateEvidence" {
		return fmt.Errorf("gate_evidence.schema_id must be runecode.protocol.v0.GateEvidence")
	}
	if evidence.SchemaVersion != "0.1.0" {
		return fmt.Errorf("gate_evidence.schema_version must be 0.1.0")
	}
	if err := validateGateEvidenceIdentity(evidence); err != nil {
		return err
	}
	if evidence.PlanCheckpointCode != "" && evidence.PlanOrderIndex < 0 {
		return fmt.Errorf("gate_evidence.plan_order_index must be >= 0 when plan_checkpoint_code is set")
	}
	if !isGateKind(evidence.GateKind) {
		return fmt.Errorf("gate_evidence has unsupported gate_kind %q", evidence.GateKind)
	}
	if evidence.ProjectContextID != "" && !isValidDigestIdentity(evidence.ProjectContextID) {
		return fmt.Errorf("gate_evidence.project_context_identity_digest must be digest identity")
	}
	if err := validateGateEvidenceTimes(evidence); err != nil {
		return err
	}
	if err := validateGateEvidencePayloadRequiredMaps(evidence); err != nil {
		return err
	}
	return nil
}

func validateGateEvidenceIdentity(evidence *GateEvidence) error {
	if evidence.GateID == "" || evidence.GateKind == "" || evidence.GateVersion == "" || evidence.GateAttemptID == "" {
		return fmt.Errorf("gate_evidence requires gate_id, gate_kind, gate_version, and gate_attempt_id")
	}
	return nil
}

func validateGateEvidenceTimes(evidence *GateEvidence) error {
	if _, err := time.Parse(time.RFC3339, evidence.StartedAt); err != nil {
		return fmt.Errorf("gate_evidence.started_at must be RFC3339")
	}
	if _, err := time.Parse(time.RFC3339, evidence.FinishedAt); err != nil {
		return fmt.Errorf("gate_evidence.finished_at must be RFC3339")
	}
	return nil
}

func validateGateEvidencePayloadRequiredMaps(evidence *GateEvidence) error {
	if len(evidence.Runtime) == 0 {
		return fmt.Errorf("gate_evidence.runtime is required")
	}
	if len(evidence.Outcome) == 0 {
		return fmt.Errorf("gate_evidence.outcome is required")
	}
	return nil
}

func validateGateEvidenceDigestFields(evidence *GateEvidence) error {
	if err := validateNormalizedInputDigests(evidence.NormalizedInputDigests); err != nil {
		return fmt.Errorf("gate_evidence.%w", err)
	}
	if err := validateNormalizedInputDigests(evidence.OutputArtifactDigests); err != nil {
		return fmt.Errorf("gate_evidence output_artifact_digests: %w", err)
	}
	if err := validateNormalizedInputDigests(evidence.PolicyDecisionRefs); err != nil {
		return fmt.Errorf("gate_evidence policy_decision_refs: %w", err)
	}
	return nil
}

func validateGateEvidenceOverrideRefs(evidence *GateEvidence) error {
	if evidence.OverrideActionRequestHash != "" && !isValidDigestIdentity(evidence.OverrideActionRequestHash) {
		return fmt.Errorf("gate_evidence.override_action_request_hash must be digest identity")
	}
	if evidence.OverridePolicyDecisionRef != "" && !isValidDigestIdentity(evidence.OverridePolicyDecisionRef) {
		return fmt.Errorf("gate_evidence.override_policy_decision_ref must be digest identity")
	}
	if evidence.OverriddenFailedResultRef != "" && !isValidDigestIdentity(evidence.OverriddenFailedResultRef) {
		return fmt.Errorf("gate_evidence.overridden_failed_result_ref must be digest identity")
	}
	return nil
}
