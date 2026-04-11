package artifacts

import (
	"fmt"
	"strings"
)

func (s *Store) RecordPolicyDecision(record PolicyDecisionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recordPolicyDecisionLocked(record)
}

func (s *Store) recordPolicyDecisionLocked(record PolicyDecisionRecord) error {
	persisted, err := s.preparePolicyDecisionForPersist(record)
	if err != nil {
		return err
	}
	if persisted.idempotentNoop {
		return nil
	}
	seq, err := s.appendPolicyDecisionAuditEvent(persisted.record, persisted.payload)
	if err != nil {
		return err
	}
	persisted.record.AuditEventType = "policy_decision_recorded"
	persisted.record.AuditEventSeq = seq
	s.persistPolicyDecisionRecord(persisted.record)
	if err := s.maybePersistDerivedApproval(persisted.record, seq); err != nil {
		return err
	}
	return s.saveStateLocked()
}

type preparedPolicyDecision struct {
	record         PolicyDecisionRecord
	payload        map[string]any
	bytes          []byte
	idempotentNoop bool
}

func (s *Store) preparePolicyDecisionForPersist(record PolicyDecisionRecord) (preparedPolicyDecision, error) {
	if err := validatePolicyDecisionRecord(record); err != nil {
		return preparedPolicyDecision{}, err
	}
	payload, payloadBytes, err := canonicalizePolicyDecisionRecord(record)
	if err != nil {
		return preparedPolicyDecision{}, err
	}
	if err := validateObjectPayloadAgainstSchema(payloadBytes, "objects/PolicyDecision.schema.json"); err != nil {
		return preparedPolicyDecision{}, fmt.Errorf("validate persisted policy decision payload: %w", err)
	}
	if err := applyComputedPolicyDecisionDigest(&record, payloadBytes); err != nil {
		return preparedPolicyDecision{}, err
	}
	if record.RecordedAt.IsZero() {
		record.RecordedAt = s.nowFn().UTC()
	}
	if done, err := s.ensureNoConflictingExistingDecision(record, payloadBytes); done {
		if err != nil {
			return preparedPolicyDecision{}, err
		}
		return preparedPolicyDecision{idempotentNoop: true}, nil
	}
	return preparedPolicyDecision{record: record, payload: payload, bytes: payloadBytes}, nil
}

func (s *Store) appendPolicyDecisionAuditEvent(record PolicyDecisionRecord, payload map[string]any) (int64, error) {
	details := policyDecisionAuditDetails(record, payload)
	s.state.LastAuditSequence++
	seq := s.state.LastAuditSequence
	event := newAuditEvent(seq, "policy_decision_recorded", "policyengine", details, s.nowFn)
	if err := s.storeIO.appendAuditEvent(event); err != nil {
		s.state.LastAuditSequence--
		return 0, err
	}
	return seq, nil
}

func (s *Store) persistPolicyDecisionRecord(record PolicyDecisionRecord) {
	s.state.PolicyDecisions[record.Digest] = record
	if record.RunID == "" {
		return
	}
	refs := append([]string{}, s.state.RunPolicyDecisionRefs[record.RunID]...)
	refs = append(refs, record.Digest)
	s.state.RunPolicyDecisionRefs[record.RunID] = uniqueSortedStrings(refs)
}

func (s *Store) maybePersistDerivedApproval(record PolicyDecisionRecord, seq int64) error {
	approval, ok, err := buildApprovalFromPolicyDecision(record, s.nowFn)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	approval.AuditEventType = "policy_decision_recorded"
	approval.AuditEventSeq = seq
	s.state.Approvals[approval.ApprovalID] = approval
	rebuildRunApprovalRefsLocked(&s.state)
	return nil
}

func validatePolicyDecisionRecord(record PolicyDecisionRecord) error {
	if record.SchemaID == "" || record.SchemaVersion == "" {
		return fmt.Errorf("policy decision schema identity is required")
	}
	if record.DetailsSchemaID == "" || record.Details == nil {
		return fmt.Errorf("policy decision details payload is required")
	}
	if record.DecisionOutcome == "require_human_approval" {
		if record.RequiredApprovalSchemaID == "" || record.RequiredApproval == nil {
			return fmt.Errorf("required_approval payload is required for require_human_approval outcome")
		}
	}
	return nil
}

func canonicalizePolicyDecisionRecord(record PolicyDecisionRecord) (map[string]any, []byte, error) {
	payload, err := policyDecisionPayload(record)
	if err != nil {
		return nil, nil, err
	}
	payloadBytes, err := canonicalPolicyDecisionBytes(payload)
	if err != nil {
		return nil, nil, err
	}
	return payload, payloadBytes, nil
}

func applyComputedPolicyDecisionDigest(record *PolicyDecisionRecord, payloadBytes []byte) error {
	computedDigest := digestBytes(payloadBytes)
	if record.Digest == "" {
		record.Digest = computedDigest
		return nil
	}
	if record.Digest != computedDigest {
		return fmt.Errorf("policy decision digest mismatch: provided %q computed %q", record.Digest, computedDigest)
	}
	return nil
}

func (s *Store) ensureNoConflictingExistingDecision(record PolicyDecisionRecord, payloadBytes []byte) (bool, error) {
	existing, exists := s.state.PolicyDecisions[record.Digest]
	if !exists {
		return false, nil
	}
	existingPayload, existingCanonical, err := canonicalizePolicyDecisionRecord(existing)
	if err != nil {
		return true, fmt.Errorf("existing policy decision payload invalid for %q: %w", record.Digest, err)
	}
	_ = existingPayload
	if string(existingCanonical) != string(payloadBytes) {
		return true, fmt.Errorf("policy decision digest %q already recorded with different payload", record.Digest)
	}
	return true, nil
}

func policyDecisionAuditDetails(record PolicyDecisionRecord, payload map[string]any) map[string]interface{} {
	details := map[string]interface{}{
		"policy_decision_digest": record.Digest,
		"decision_outcome":       record.DecisionOutcome,
		"policy_reason_code":     record.PolicyReasonCode,
		"manifest_hash":          record.ManifestHash,
		"action_request_hash":    record.ActionRequestHash,
		"policy_input_hashes":    append([]string{}, record.PolicyInputHashes...),
		"policy_decision":        payload,
	}
	if len(record.RelevantArtifactHashes) > 0 {
		details["relevant_artifact_hashes"] = append([]string{}, record.RelevantArtifactHashes...)
	}
	if record.RunID != "" {
		details["run_id"] = record.RunID
	}
	return details
}

func policyDecisionPayload(record PolicyDecisionRecord) (map[string]any, error) {
	manifest, err := digestObjectFromIdentity(record.ManifestHash)
	if err != nil {
		return nil, fmt.Errorf("manifest_hash: %w", err)
	}
	actionHash, err := digestObjectFromIdentity(record.ActionRequestHash)
	if err != nil {
		return nil, fmt.Errorf("action_request_hash: %w", err)
	}
	policyInputs, err := digestObjectSliceFromIdentities(record.PolicyInputHashes)
	if err != nil {
		return nil, fmt.Errorf("policy_input_hashes: %w", err)
	}
	relevantArtifacts, err := digestObjectSliceFromIdentities(record.RelevantArtifactHashes)
	if err != nil {
		return nil, fmt.Errorf("relevant_artifact_hashes: %w", err)
	}
	payload := map[string]any{
		"schema_id":                record.SchemaID,
		"schema_version":           record.SchemaVersion,
		"decision_outcome":         record.DecisionOutcome,
		"policy_reason_code":       record.PolicyReasonCode,
		"manifest_hash":            manifest,
		"action_request_hash":      actionHash,
		"relevant_artifact_hashes": relevantArtifacts,
		"policy_input_hashes":      policyInputs,
		"details_schema_id":        record.DetailsSchemaID,
		"details":                  record.Details,
	}
	if record.RequiredApprovalSchemaID != "" {
		payload["required_approval_schema_id"] = record.RequiredApprovalSchemaID
	}
	if record.RequiredApproval != nil {
		payload["required_approval"] = record.RequiredApproval
	}
	return payload, nil
}

func digestObjectFromIdentity(identity string) (map[string]any, error) {
	if len(identity) != len("sha256:")+64 || !strings.HasPrefix(identity, "sha256:") {
		return nil, fmt.Errorf("invalid digest identity %q", identity)
	}
	return map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(identity, "sha256:")}, nil
}

func digestObjectSliceFromIdentities(identities []string) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(identities))
	for i := range identities {
		d, err := digestObjectFromIdentity(identities[i])
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		out = append(out, d)
	}
	return out, nil
}

func canonicalPolicyDecisionBytes(payload map[string]any) ([]byte, error) {
	b, err := canonicalizeJSONValue(payload)
	if err != nil {
		return nil, fmt.Errorf("canonicalize policy decision payload: %w", err)
	}
	return b, nil
}

func (s *Store) PolicyDecisionRefsForRun(runID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	refs := append([]string{}, s.state.RunPolicyDecisionRefs[runID]...)
	if len(refs) == 0 {
		return []string{}
	}
	return refs
}

func (s *Store) PolicyDecisionGet(digest string) (PolicyDecisionRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.state.PolicyDecisions[strings.TrimSpace(digest)]
	return rec, ok
}
