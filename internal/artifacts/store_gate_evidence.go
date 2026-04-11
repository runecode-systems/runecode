package artifacts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GateEvidenceArtifact is the canonical typed evidence object persisted per gate attempt.
// It is intentionally reference-heavy and keeps large outputs in referenced artifacts.
type GateEvidenceArtifact struct {
	SchemaID               string         `json:"schema_id"`
	SchemaVersion          string         `json:"schema_version"`
	GateID                 string         `json:"gate_id"`
	GateKind               string         `json:"gate_kind"`
	GateVersion            string         `json:"gate_version"`
	PlanCheckpointCode     string         `json:"plan_checkpoint_code,omitempty"`
	PlanOrderIndex         int            `json:"plan_order_index,omitempty"`
	RunID                  string         `json:"run_id"`
	StageID                string         `json:"stage_id,omitempty"`
	StepID                 string         `json:"step_id,omitempty"`
	RoleInstanceID         string         `json:"role_instance_id,omitempty"`
	GateAttemptID          string         `json:"gate_attempt_id"`
	StartedAt              string         `json:"started_at"`
	FinishedAt             string         `json:"finished_at"`
	NormalizedInputDigests []string       `json:"normalized_input_digests,omitempty"`
	Runtime                map[string]any `json:"runtime"`
	Outcome                map[string]any `json:"outcome"`
	OutputArtifactDigests  []string       `json:"output_artifact_digests,omitempty"`
	PolicyDecisionRefs     []string       `json:"policy_decision_refs,omitempty"`
	OverrideActionHash     string         `json:"override_action_request_hash,omitempty"`
	OverridePolicyRef      string         `json:"override_policy_decision_ref,omitempty"`
	OverriddenFailedRef    string         `json:"overridden_failed_result_ref,omitempty"`
	FailureReasonCode      string         `json:"failure_reason_code,omitempty"`
}

func (s *Store) PutGateEvidence(runID string, evidence GateEvidenceArtifact) (ArtifactReference, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateGateEvidenceArtifact(evidence, runID); err != nil {
		return ArtifactReference{}, err
	}
	if _, err := time.Parse(time.RFC3339, evidence.StartedAt); err != nil {
		return ArtifactReference{}, fmt.Errorf("gate evidence started_at must be RFC3339")
	}
	finishedAt, err := time.Parse(time.RFC3339, evidence.FinishedAt)
	if err != nil {
		return ArtifactReference{}, fmt.Errorf("gate evidence finished_at must be RFC3339")
	}
	payload, err := canonicalizeJSONValue(evidence)
	if err != nil {
		return ArtifactReference{}, fmt.Errorf("canonicalize gate evidence: %w", err)
	}
	finishedAtUTC := finishedAt.UTC()
	ref, err := s.putLocked(PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             DataClassGateEvidence,
		ProvenanceReceiptHash: evidenceOutputProvenanceHash(evidence),
		CreatedByRole:         "brokerapi",
		TrustedSource:         true,
		RunID:                 strings.TrimSpace(runID),
		StepID:                strings.TrimSpace(evidence.StepID),
	})
	if err != nil {
		return ArtifactReference{}, err
	}
	if err := s.appendAuditLocked("gate_evidence_recorded", "brokerapi", map[string]interface{}{
		"run_id":            strings.TrimSpace(runID),
		"gate_id":           evidence.GateID,
		"gate_kind":         evidence.GateKind,
		"gate_version":      evidence.GateVersion,
		"gate_attempt_id":   evidence.GateAttemptID,
		"finished_at":       finishedAtUTC.Format(time.RFC3339),
		"gate_evidence_ref": ref.Digest,
	}); err != nil {
		return ArtifactReference{}, err
	}
	if err := s.saveStateLocked(); err != nil {
		return ArtifactReference{}, err
	}
	return ref, nil
}

func validateGateEvidenceArtifact(evidence GateEvidenceArtifact, runID string) error {
	if err := validateGateEvidenceContract(evidence, runID); err != nil {
		return err
	}
	if err := validateGateEvidenceTimestamps(evidence); err != nil {
		return err
	}
	if err := validateGateEvidencePayload(evidence); err != nil {
		return err
	}
	if err := validateGateEvidenceDigestSets(evidence); err != nil {
		return err
	}
	return validateGateEvidenceOptionalRefs(evidence)
}

func validateGateEvidenceContract(evidence GateEvidenceArtifact, runID string) error {
	if evidence.SchemaID != "runecode.protocol.v0.GateEvidence" {
		return fmt.Errorf("gate evidence schema_id must be runecode.protocol.v0.GateEvidence")
	}
	if evidence.SchemaVersion != "0.1.0" {
		return fmt.Errorf("gate evidence schema_version must be 0.1.0")
	}
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(evidence.RunID) == "" || strings.TrimSpace(runID) != strings.TrimSpace(evidence.RunID) {
		return fmt.Errorf("gate evidence run id must match run scope")
	}
	if strings.TrimSpace(evidence.GateID) == "" || strings.TrimSpace(evidence.GateKind) == "" || strings.TrimSpace(evidence.GateVersion) == "" || strings.TrimSpace(evidence.GateAttemptID) == "" {
		return fmt.Errorf("gate evidence requires gate_id, gate_kind, gate_version, and gate_attempt_id")
	}
	if !isValidGateKind(evidence.GateKind) {
		return fmt.Errorf("gate evidence has unsupported gate_kind %q", evidence.GateKind)
	}
	if strings.TrimSpace(evidence.PlanCheckpointCode) != "" && evidence.PlanOrderIndex < 0 {
		return fmt.Errorf("gate evidence plan_order_index must be >= 0 when plan_checkpoint_code is set")
	}
	return nil
}

func validateGateEvidenceTimestamps(evidence GateEvidenceArtifact) error {
	startedAt, err := time.Parse(time.RFC3339, evidence.StartedAt)
	if err != nil {
		return fmt.Errorf("gate evidence started_at must be RFC3339")
	}
	finishedAt, err := time.Parse(time.RFC3339, evidence.FinishedAt)
	if err != nil {
		return fmt.Errorf("gate evidence finished_at must be RFC3339")
	}
	if finishedAt.Before(startedAt) {
		return fmt.Errorf("gate evidence finished_at must not be before started_at")
	}
	return nil
}

func validateGateEvidencePayload(evidence GateEvidenceArtifact) error {
	if len(evidence.Runtime) == 0 {
		return fmt.Errorf("gate evidence runtime is required")
	}
	if len(evidence.Outcome) == 0 {
		return fmt.Errorf("gate evidence outcome is required")
	}
	return nil
}

func validateGateEvidenceDigestSets(evidence GateEvidenceArtifact) error {
	if err := ensureDigestIdentities(evidence.NormalizedInputDigests, "normalized_input_digests"); err != nil {
		return err
	}
	if err := ensureDigestIdentities(evidence.OutputArtifactDigests, "output_artifact_digests"); err != nil {
		return err
	}
	return ensureDigestIdentities(evidence.PolicyDecisionRefs, "policy_decision_refs")
}

func validateGateEvidenceOptionalRefs(evidence GateEvidenceArtifact) error {
	if strings.TrimSpace(evidence.OverrideActionHash) != "" && !isValidDigest(strings.TrimSpace(evidence.OverrideActionHash)) {
		return fmt.Errorf("gate evidence override_action_request_hash must be digest identity")
	}
	if strings.TrimSpace(evidence.OverridePolicyRef) != "" && !isValidDigest(strings.TrimSpace(evidence.OverridePolicyRef)) {
		return fmt.Errorf("gate evidence override_policy_decision_ref must be digest identity")
	}
	if strings.TrimSpace(evidence.OverriddenFailedRef) != "" && !isValidDigest(strings.TrimSpace(evidence.OverriddenFailedRef)) {
		return fmt.Errorf("gate evidence overridden_failed_result_ref must be digest identity")
	}
	return nil
}

func isValidGateKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "build", "test", "lint", "format", "secret_scan", "policy":
		return true
	default:
		return false
	}
}

func ensureDigestIdentities(values []string, field string) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if !isValidDigest(trimmed) {
			return fmt.Errorf("%s contains invalid digest %q", field, value)
		}
		if _, ok := seen[trimmed]; ok {
			return fmt.Errorf("%s contains duplicate digest %q", field, trimmed)
		}
		seen[trimmed] = struct{}{}
	}
	return nil
}

func evidenceOutputProvenanceHash(evidence GateEvidenceArtifact) string {
	b, _ := json.Marshal(map[string]any{
		"run_id":          evidence.RunID,
		"gate_id":         evidence.GateID,
		"gate_attempt_id": evidence.GateAttemptID,
		"finished_at":     evidence.FinishedAt,
	})
	return digestBytes(b)
}
