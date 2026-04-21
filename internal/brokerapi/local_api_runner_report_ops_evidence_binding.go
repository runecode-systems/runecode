package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) resolveGateEvidenceRef(runID string, report RunnerResultReport, planned runPlannedGateEntry) (string, error) {
	providedRef := report.GateEvidenceRef
	if report.GateEvidence == nil {
		return providedRef, nil
	}
	evidence := report.GateEvidence
	runtimeSummary, err := sanitizeRunnerDetails(evidence.Runtime)
	if err != nil {
		return "", fmt.Errorf("gate_evidence.runtime: %w", err)
	}
	outcomeSummary, err := sanitizeRunnerDetails(evidence.Outcome)
	if err != nil {
		return "", fmt.Errorf("gate_evidence.outcome: %w", err)
	}
	if err := validateGateEvidenceReportBinding(runID, report, evidence); err != nil {
		return "", err
	}
	evidenceRecord := buildGateEvidenceRecord(report, evidence, runtimeSummary, outcomeSummary, planned)
	canonicalEvidence, err := canonicalGateEvidenceDigest(evidenceRecord)
	if err != nil {
		return "", err
	}
	if providedRef != "" && providedRef != canonicalEvidence {
		return "", fmt.Errorf("gate_evidence_ref does not match canonical evidence digest")
	}
	ref, err := s.PutGateEvidence(runID, evidenceRecord)
	if err != nil {
		return "", err
	}
	return ref.Digest, nil
}

func validateGateEvidenceReportBinding(runID string, report RunnerResultReport, evidence *GateEvidence) error {
	if evidence.RunID != runID {
		return fmt.Errorf("gate_evidence.run_id must match run_id")
	}
	if evidence.GateID != report.GateID || evidence.GateKind != report.GateKind || evidence.GateVersion != report.GateVersion || evidence.GateAttemptID != report.GateAttemptID {
		return fmt.Errorf("gate_evidence identity must match gate report binding")
	}
	if evidence.PlanCheckpointCode != "" && evidence.PlanCheckpointCode != report.PlanCheckpointCode {
		return fmt.Errorf("gate_evidence.plan_checkpoint_code must match gate report binding")
	}
	if evidence.PlanCheckpointCode != "" && evidence.PlanOrderIndex != report.PlanOrderIndex {
		return fmt.Errorf("gate_evidence.plan_order_index must match gate report binding")
	}
	return nil
}

func buildGateEvidenceRecord(report RunnerResultReport, evidence *GateEvidence, runtimeSummary map[string]any, outcomeSummary map[string]any, planned runPlannedGateEntry) artifacts.GateEvidenceArtifact {
	evidenceRecord := gateEvidenceRecordBase(evidence, runtimeSummary, outcomeSummary)
	if report.PlanCheckpointCode != "" {
		evidenceRecord.PlanCheckpointCode = report.PlanCheckpointCode
		evidenceRecord.PlanOrderIndex = report.PlanOrderIndex
	}
	if planned.MaxAttempts > 0 {
		evidenceRecord.Runtime["planned_retry_max_attempts"] = planned.MaxAttempts
	}
	applyGateEvidenceProjectContext(&evidenceRecord, runtimeSummary, planned)
	return evidenceRecord
}

func gateEvidenceRecordBase(evidence *GateEvidence, runtimeSummary map[string]any, outcomeSummary map[string]any) artifacts.GateEvidenceArtifact {
	return artifacts.GateEvidenceArtifact{
		SchemaID:               evidence.SchemaID,
		SchemaVersion:          evidence.SchemaVersion,
		GateID:                 evidence.GateID,
		GateKind:               evidence.GateKind,
		GateVersion:            evidence.GateVersion,
		ProjectContextID:       evidence.ProjectContextID,
		PlanCheckpointCode:     evidence.PlanCheckpointCode,
		PlanOrderIndex:         evidence.PlanOrderIndex,
		RunID:                  evidence.RunID,
		StageID:                evidence.StageID,
		StepID:                 evidence.StepID,
		RoleInstanceID:         evidence.RoleInstanceID,
		GateAttemptID:          evidence.GateAttemptID,
		StartedAt:              evidence.StartedAt,
		FinishedAt:             evidence.FinishedAt,
		NormalizedInputDigests: append([]string{}, evidence.NormalizedInputDigests...),
		Runtime:                runtimeSummary,
		Outcome:                outcomeSummary,
		OutputArtifactDigests:  append([]string{}, evidence.OutputArtifactDigests...),
		PolicyDecisionRefs:     append([]string{}, evidence.PolicyDecisionRefs...),
		OverrideActionHash:     evidence.OverrideActionRequestHash,
		OverridePolicyRef:      evidence.OverridePolicyDecisionRef,
		OverriddenFailedRef:    evidence.OverriddenFailedResultRef,
		FailureReasonCode:      evidence.FailureReasonCode,
	}
}

func applyGateEvidenceProjectContext(evidenceRecord *artifacts.GateEvidenceArtifact, runtimeSummary map[string]any, planned runPlannedGateEntry) {
	if strings.TrimSpace(evidenceRecord.ProjectContextID) == "" {
		evidenceRecord.ProjectContextID = strings.TrimSpace(planned.ProjectContextID)
	}
	if strings.TrimSpace(evidenceRecord.ProjectContextID) == "" {
		if value, ok := runtimeSummary["project_context_identity_digest"].(string); ok {
			evidenceRecord.ProjectContextID = strings.TrimSpace(value)
		}
	}
	if strings.TrimSpace(evidenceRecord.ProjectContextID) != "" {
		evidenceRecord.Runtime["project_context_identity_digest"] = strings.TrimSpace(evidenceRecord.ProjectContextID)
	}
}

func canonicalGateEvidenceDigest(evidence artifacts.GateEvidenceArtifact) (string, error) {
	payload, err := json.Marshal(evidence)
	if err != nil {
		return "", fmt.Errorf("marshal gate evidence: %w", err)
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(payload)
	if err != nil {
		return "", fmt.Errorf("canonicalize gate evidence: %w", err)
	}
	return artifacts.DigestBytes(canonical), nil
}
