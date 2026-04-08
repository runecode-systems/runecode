package policyengine

import "fmt"

func evaluateRuleSet(compiled *CompiledContext, action ActionRequest, actionHash string, ruleSet PolicyRuleSet) (PolicyDecision, bool) {
	matchedRule, selected, found := selectHighestPrecedenceRule(ruleSet.Rules, action)
	if !found {
		return PolicyDecision{}, false
	}
	decision := PolicyDecision{
		SchemaID:               policyDecisionSchemaID,
		SchemaVersion:          policyDecisionSchemaVersion,
		DecisionOutcome:        selected,
		PolicyReasonCode:       matchedRule.ReasonCode,
		ManifestHash:           compiled.ManifestHash,
		PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: actionRelevantArtifactHashes(action),
		DetailsSchemaID:        matchedRule.DetailsSchemaID,
		Details: map[string]any{
			"precedence": "deny_gt_require_human_approval_gt_allow",
			"source":     policyRuleSetSchemaID,
			"rule_id":    matchedRule.RuleID,
		},
	}
	if selected == DecisionRequireHumanApproval {
		decision.RequiredApprovalSchemaID, decision.RequiredApproval = ruleSetRequiredApproval(compiled, action, actionHash)
	}
	return decision, true
}

func ruleSetRequiredApproval(compiled *CompiledContext, action ActionRequest, actionHash string) (string, map[string]any) {
	requiredSchemaID, requiredPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
	if requiredSchemaID != "" {
		return requiredSchemaID, requiredPayload
	}
	return defaultRuleSetRequiredApproval(compiled, action, actionHash)
}

func defaultRuleSetRequiredApproval(compiled *CompiledContext, action ActionRequest, actionHash string) (string, map[string]any) {
	return "runecode.protocol.details.policy.required_approval.rule.v0", map[string]any{
		"approval_trigger_code":         "stage_sign_off",
		"approval_assurance_level":      string(ApprovalAssuranceSessionAuthenticated),
		"scope":                         approvalScopeForAction(action),
		"why_required":                  "Rule-set selected require_human_approval effect.",
		"changes_if_approved":           "Action may proceed once approval is granted.",
		"effects_if_denied_or_deferred": "Action remains blocked.",
		"security_posture_impact":       "moderate",
		"blocked_work":                  []string{"action_execution"},
		"approval_ttl_seconds":          1800,
		"related_hashes": map[string]any{
			"manifest_hash":            compiled.ManifestHash,
			"action_request_hash":      actionHash,
			"policy_input_hashes":      append([]string{}, compiled.PolicyInputHashes...),
			"relevant_artifact_hashes": actionRelevantArtifactHashes(action),
		},
	}
}

func selectHighestPrecedenceRule(rules []PolicyRule, action ActionRequest) (PolicyRule, DecisionOutcome, bool) {
	state := ruleSelectionState{}
	for _, rule := range rules {
		if !ruleMatchesAction(rule, action) {
			continue
		}
		state.record(rule)
	}
	if state.hasDeny {
		return state.denyRule, DecisionDeny, true
	}
	if state.hasApproval {
		return state.approvalRule, DecisionRequireHumanApproval, true
	}
	if state.hasAllow {
		return state.allowRule, DecisionAllow, true
	}
	return PolicyRule{}, "", false
}

type ruleSelectionState struct {
	hasAllow     bool
	hasApproval  bool
	hasDeny      bool
	allowRule    PolicyRule
	approvalRule PolicyRule
	denyRule     PolicyRule
}

func ruleMatchesAction(rule PolicyRule, action ActionRequest) bool {
	if rule.ActionKind != action.ActionKind {
		return false
	}
	return rule.CapabilityID == "" || rule.CapabilityID == action.CapabilityID
}

func (s *ruleSelectionState) record(rule PolicyRule) {
	switch rule.Effect {
	case string(DecisionDeny):
		s.hasDeny = true
		s.denyRule = rule
	case string(DecisionRequireHumanApproval):
		s.hasApproval = true
		s.approvalRule = rule
	case string(DecisionAllow):
		s.hasAllow = true
		s.allowRule = rule
	}
}

func matchedReasonCode(outcome DecisionOutcome) string {
	switch outcome {
	case DecisionDeny:
		return "deny_by_default"
	case DecisionRequireHumanApproval:
		return "approval_required"
	case DecisionAllow:
		return "allow_manifest_opt_in"
	default:
		panic(fmt.Sprintf("unknown outcome %q", outcome))
	}
}
