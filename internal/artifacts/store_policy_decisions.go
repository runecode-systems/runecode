package artifacts

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (s *Store) RecordPolicyDecision(record PolicyDecisionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recordPolicyDecisionLocked(record)
}

func (s *Store) recordPolicyDecisionLocked(record PolicyDecisionRecord) error {
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
	payload, err := policyDecisionPayload(record)
	if err != nil {
		return err
	}
	payloadBytes, err := canonicalPolicyDecisionBytes(payload)
	if err != nil {
		return err
	}
	if err := validateObjectPayloadAgainstSchema(payloadBytes, "objects/PolicyDecision.schema.json"); err != nil {
		return fmt.Errorf("validate persisted policy decision payload: %w", err)
	}
	computedDigest := digestBytes(payloadBytes)
	if record.Digest == "" {
		record.Digest = computedDigest
	} else if record.Digest != computedDigest {
		return fmt.Errorf("policy decision digest mismatch: provided %q computed %q", record.Digest, computedDigest)
	}
	if record.RecordedAt.IsZero() {
		record.RecordedAt = s.nowFn().UTC()
	}
	if existing, exists := s.state.PolicyDecisions[record.Digest]; exists {
		existingPayload, payloadErr := policyDecisionPayload(existing)
		if payloadErr != nil {
			return fmt.Errorf("existing policy decision payload invalid for %q: %w", record.Digest, payloadErr)
		}
		existingCanonical, canonicalErr := canonicalPolicyDecisionBytes(existingPayload)
		if canonicalErr != nil {
			return fmt.Errorf("canonicalize existing policy decision payload for %q: %w", record.Digest, canonicalErr)
		}
		if string(existingCanonical) != string(payloadBytes) {
			return fmt.Errorf("policy decision digest %q already recorded with different payload", record.Digest)
		}
		return nil
	}

	details := map[string]interface{}{
		"policy_decision_digest": record.Digest,
		"decision_outcome":       record.DecisionOutcome,
		"policy_reason_code":     record.PolicyReasonCode,
		"manifest_hash":          record.ManifestHash,
		"action_request_hash":    record.ActionRequestHash,
		"policy_input_hashes":    append([]string{}, record.PolicyInputHashes...),
	}
	if len(record.RelevantArtifactHashes) > 0 {
		details["relevant_artifact_hashes"] = append([]string{}, record.RelevantArtifactHashes...)
	}
	if record.RunID != "" {
		details["run_id"] = record.RunID
	}
	details["policy_decision"] = payload

	s.state.LastAuditSequence++
	seq := s.state.LastAuditSequence
	event := newAuditEvent(seq, "policy_decision_recorded", "policyengine", details, s.nowFn)
	if err := s.storeIO.appendAuditEvent(event); err != nil {
		s.state.LastAuditSequence--
		return err
	}
	record.AuditEventType = "policy_decision_recorded"
	record.AuditEventSeq = seq

	s.state.PolicyDecisions[record.Digest] = record
	if record.RunID != "" {
		refs := append([]string{}, s.state.RunPolicyDecisionRefs[record.RunID]...)
		refs = append(refs, record.Digest)
		s.state.RunPolicyDecisionRefs[record.RunID] = uniqueSortedStrings(refs)
	}
	return s.saveStateLocked()
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
	b, err := jsoncanonicalizer.Transform(mustJSON(payload))
	if err != nil {
		return nil, fmt.Errorf("canonicalize policy decision payload: %w", err)
	}
	return b, nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
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

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	clone := append([]string{}, values...)
	sort.Strings(clone)
	out := make([]string, 0, len(clone))
	for _, v := range clone {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}
