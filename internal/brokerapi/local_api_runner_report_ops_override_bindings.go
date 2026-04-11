package brokerapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) resolveOverrideApprovalBindings(runID string, report RunnerResultReport, sanitizedDetails map[string]any) (string, string, error) {
	if report.GateLifecycleState != "overridden" {
		return "", "", nil
	}
	action, err := overrideActionForResult(report, sanitizedDetails)
	if err != nil {
		return "", "", err
	}
	actionHash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		return "", "", fmt.Errorf("canonical override action hash: %w", err)
	}
	latestRef, ok := s.latestGateOverridePolicyDecisionRef(runID, actionHash)
	if !ok {
		return "", "", fmt.Errorf("gate override requires prior policy decision approval for exact override action")
	}
	if err := s.requireValidGateOverrideApproval(runID, latestRef); err != nil {
		return "", "", err
	}
	if !hasPolicyContextDigest(sanitizedDetails, report.NormalizedInputDigests) {
		return "", "", fmt.Errorf("gate override result requires details.policy_context_hash present in normalized_input_digests")
	}
	return actionHash, latestRef, nil
}

func hasPolicyContextDigest(details map[string]any, normalizedInputDigests []string) bool {
	value, _ := details["policy_context_hash"].(string)
	if !isValidDigestIdentity(value) {
		return false
	}
	for _, digest := range normalizedInputDigests {
		if digest == value {
			return true
		}
	}
	return false
}

func (s *Service) latestGateOverridePolicyDecisionRef(runID, actionHash string) (string, bool) {
	latest := ""
	for _, ref := range s.PolicyDecisionRefsForRun(runID) {
		rec, ok := s.PolicyDecisionGet(ref)
		if !ok || !matchesGateOverridePolicyDecision(rec, actionHash) {
			continue
		}
		latest = ref
	}
	if latest == "" {
		return "", false
	}
	return latest, true
}

func matchesGateOverridePolicyDecision(rec artifacts.PolicyDecisionRecord, actionHash string) bool {
	if rec.DecisionOutcome != string(policyengine.DecisionRequireHumanApproval) {
		return false
	}
	if strings.TrimSpace(rec.ActionRequestHash) != actionHash {
		return false
	}
	if strings.TrimSpace(rec.PolicyReasonCode) != "approval_required" {
		return false
	}
	trigger, _ := rec.RequiredApproval["approval_trigger_code"].(string)
	return strings.TrimSpace(trigger) == "gate_override"
}

func (s *Service) requireValidGateOverrideApproval(runID, policyDecisionRef string) error {
	for _, approval := range s.listApprovals() {
		if !isMatchingApprovedGateOverrideApproval(approval, runID, policyDecisionRef) {
			continue
		}
		if err := validateGateOverrideApprovalExpiry(approval.ExpiresAt, s.now().UTC()); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("gate override requires explicit approved approval")
}

func isMatchingApprovedGateOverrideApproval(approval ApprovalSummary, runID, policyDecisionRef string) bool {
	if approval.BoundScope.RunID != runID || approval.BoundScope.ActionKind != policyengine.ActionKindGateOverride {
		return false
	}
	if strings.TrimSpace(approval.PolicyDecisionHash) != policyDecisionRef {
		return false
	}
	return approval.Status == "approved"
}

func validateGateOverrideApprovalExpiry(expiresAtRaw string, now time.Time) error {
	if expiresAtRaw == "" {
		return fmt.Errorf("gate override approval missing expires_at")
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresAtRaw)
	if err != nil {
		return fmt.Errorf("gate override approval has invalid expires_at")
	}
	if now.After(expiresAt.UTC()) {
		return fmt.Errorf("gate override approval expired")
	}
	return nil
}
