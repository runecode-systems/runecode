package artifacts

import (
	"fmt"
	"strings"
)

func validateApprovalRecord(record ApprovalRecord) error {
	if err := validateApprovalRecordRequiredFields(record); err != nil {
		return err
	}
	if err := validateApprovalRecordBindings(record); err != nil {
		return err
	}
	return validateApprovalRecordOptionalDigests(record)
}

func validateApprovalRecordRequiredFields(record ApprovalRecord) error {
	if !isValidDigest(record.ApprovalID) {
		return fmt.Errorf("approval_id: %w", ErrInvalidDigest)
	}
	if strings.TrimSpace(record.Status) == "" {
		return fmt.Errorf("approval status is required")
	}
	if strings.TrimSpace(record.ActionKind) == "" {
		return fmt.Errorf("action kind is required")
	}
	if record.RequestedAt.IsZero() {
		return fmt.Errorf("requested_at is required")
	}
	if strings.TrimSpace(record.ApprovalTriggerCode) == "" {
		return fmt.Errorf("approval_trigger_code is required")
	}
	if strings.TrimSpace(record.ChangesIfApproved) == "" {
		return fmt.Errorf("changes_if_approved is required")
	}
	if strings.TrimSpace(record.ApprovalAssuranceLevel) == "" {
		return fmt.Errorf("approval_assurance_level is required")
	}
	if strings.TrimSpace(record.PresenceMode) == "" {
		return fmt.Errorf("presence_mode is required")
	}
	return nil
}

func validateApprovalRecordBindings(record ApprovalRecord) error {
	if !isValidDigest(record.ManifestHash) {
		return fmt.Errorf("manifest_hash: %w", ErrInvalidDigest)
	}
	if !isValidDigest(record.ActionRequestHash) {
		return fmt.Errorf("action_request_hash: %w", ErrInvalidDigest)
	}
	for i := range record.RelevantArtifactHashes {
		if !isValidDigest(record.RelevantArtifactHashes[i]) {
			return fmt.Errorf("relevant_artifact_hashes[%d]: %w", i, ErrInvalidDigest)
		}
	}
	return nil
}

func validateApprovalRecordOptionalDigests(record ApprovalRecord) error {
	if record.RequestDigest != "" && !isValidDigest(record.RequestDigest) {
		return fmt.Errorf("request_digest: %w", ErrInvalidDigest)
	}
	if record.DecisionDigest != "" && !isValidDigest(record.DecisionDigest) {
		return fmt.Errorf("decision_digest: %w", ErrInvalidDigest)
	}
	if record.PolicyDecisionHash != "" && !isValidDigest(record.PolicyDecisionHash) {
		return fmt.Errorf("policy_decision_hash: %w", ErrInvalidDigest)
	}
	if record.SourceDigest != "" && !isValidDigest(record.SourceDigest) {
		return fmt.Errorf("source_digest: %w", ErrInvalidDigest)
	}
	return nil
}
