package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

type sessionDraftPromoteResolvedInput struct {
	draftArtifactDigest string
	draftTextDigest     string
	draftSchemaID       string
	draftIdentity       string
	draftText           []byte
	targetRelativePath  string
	targetAbsolutePath  string
	appliedFileDigest   string
	sourcePromptDigest  string
	projectDigest       string
}

func (s *Service) applySessionExecutionDraftPromote(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority) (string, error) {
	resolved, err := s.resolveSessionDraftPromoteApplyInput(result, authority)
	if err != nil {
		return "", err
	}
	prepared, err := prepareDraftPromoteMutation(resolved)
	if err != nil {
		return "", err
	}
	approvalID := sessionDraftPromoteApplyApprovalIdentity(resolved)
	if err := finalizeBrokerOwnedMutationWrites(prepared, func() error {
		if err := s.recordSessionExecutionDraftPromoteApproval(result, authority, resolved, approvalID); err != nil {
			return err
		}
		if err := s.appendDraftPromoteAuditEvent(result, authority, resolved, approvalID); err != nil {
			return fmt.Errorf("append draft promote/apply audit event: %w", err)
		}
		return nil
	}); err != nil {
		return "", err
	}
	return approvalID, nil
}

func prepareDraftPromoteMutation(resolved sessionDraftPromoteResolvedInput) ([]brokerOwnedPreparedMutationWrite, error) {
	return prepareBrokerOwnedMutationWrites([]brokerOwnedMutationWriteIntent{{
		targetAbsolutePath: resolved.targetAbsolutePath,
		targetRelativePath: resolved.targetRelativePath,
		contents:           resolved.draftText,
		expectedDigest:     resolved.appliedFileDigest,
		mode:               0o644,
	}})
}

func (s *Service) appendDraftPromoteAuditEvent(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority, resolved sessionDraftPromoteResolvedInput, approvalID string) error {
	return s.AppendTrustedAuditEvent("runecontext_draft_promote_apply", "brokerapi", map[string]interface{}{
		"run_id":                strings.TrimSpace(authority.runID),
		"plan_id":               strings.TrimSpace(authority.planID),
		"workflow_operation":    strings.TrimSpace(authority.workflowOperation),
		"draft_artifact_digest": strings.TrimSpace(resolved.draftArtifactDigest),
		"draft_text_digest":     strings.TrimSpace(resolved.draftTextDigest),
		"draft_schema_id":       strings.TrimSpace(resolved.draftSchemaID),
		"draft_identity":        strings.TrimSpace(resolved.draftIdentity),
		"source_prompt_digest":  strings.TrimSpace(resolved.sourcePromptDigest),
		"project_digest":        strings.TrimSpace(resolved.projectDigest),
		"target_relative_path":  strings.TrimSpace(resolved.targetRelativePath),
		"applied_file_digest":   strings.TrimSpace(resolved.appliedFileDigest),
		"approval_id":           strings.TrimSpace(approvalID),
		"trigger_id":            strings.TrimSpace(result.Trigger.TriggerID),
		"turn_id":               strings.TrimSpace(result.TurnExecution.TurnID),
	})
}

func sessionDraftPromoteApplyApprovalIdentity(resolved sessionDraftPromoteResolvedInput) string {
	return shaDigestIdentity(strings.TrimSpace(resolved.draftArtifactDigest) + "\n" + strings.TrimSpace(resolved.targetRelativePath))
}

func (s *Service) recordSessionExecutionDraftPromoteApproval(result artifacts.SessionExecutionTriggerAppendResult, authority sessionExecutionPlanAuthority, resolved sessionDraftPromoteResolvedInput, approvalID string) error {
	actionHash := sessionDraftPromoteActionHash(resolved)
	priorRefs := stringSetFromSlice(s.PolicyDecisionRefsForRun(strings.TrimSpace(authority.runID)))
	if err := s.RecordPolicyDecision(strings.TrimSpace(authority.runID), "", draftPromotePolicyDecision(authority, resolved, actionHash)); err != nil {
		return fmt.Errorf("record draft promote/apply policy decision: %w", err)
	}
	decisionHash, err := recordedPolicyDecisionHashForRunAndAction(s, strings.TrimSpace(authority.runID), actionHash, priorRefs)
	if err != nil {
		return fmt.Errorf("locate draft promote/apply policy decision hash: %w", err)
	}
	record := draftPromoteApprovalRecord(authority, resolved, approvalID, actionHash, decisionHash, s.currentTimestamp())
	if err := s.RecordApproval(record); err != nil {
		return fmt.Errorf("record draft promote/apply approval: %w", err)
	}
	return nil
}

func sessionDraftPromoteActionHash(resolved sessionDraftPromoteResolvedInput) string {
	return shaDigestIdentity(strings.TrimSpace(resolved.targetRelativePath) + "\n" + strings.TrimSpace(resolved.appliedFileDigest) + "\n" + strings.TrimSpace(resolved.draftArtifactDigest))
}

func recordedPolicyDecisionHashForRunAndAction(s *Service, runID, actionHash string, priorRefs map[string]struct{}) (string, error) {
	newMatches, existingMatches := policyDecisionMatchesForRunAndAction(s, runID, actionHash, priorRefs)
	return selectRecordedPolicyDecisionHash(newMatches, existingMatches)
}

func policyDecisionMatchesForRunAndAction(s *Service, runID, actionHash string, priorRefs map[string]struct{}) ([]string, []string) {
	newMatches := []string{}
	existingMatches := []string{}
	for _, digest := range s.PolicyDecisionRefsForRun(strings.TrimSpace(runID)) {
		record, ok := s.PolicyDecisionGet(strings.TrimSpace(digest))
		if !ok {
			continue
		}
		if strings.TrimSpace(record.ActionRequestHash) != strings.TrimSpace(actionHash) {
			continue
		}
		trimmed := strings.TrimSpace(record.Digest)
		if _, existed := priorRefs[trimmed]; existed {
			existingMatches = append(existingMatches, trimmed)
			continue
		}
		newMatches = append(newMatches, trimmed)
	}
	return newMatches, existingMatches
}

func selectRecordedPolicyDecisionHash(newMatches, existingMatches []string) (string, error) {
	if len(newMatches) == 1 {
		return newMatches[0], nil
	}
	if len(newMatches) > 1 {
		return "", fmt.Errorf("multiple newly recorded decisions matched action hash")
	}
	if len(existingMatches) == 1 {
		return existingMatches[0], nil
	}
	if len(existingMatches) > 1 {
		return "", fmt.Errorf("multiple existing decisions matched action hash")
	}
	return "", fmt.Errorf("decision not found")
}

func stringSetFromSlice(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out[trimmed] = struct{}{}
		}
	}
	return out
}
