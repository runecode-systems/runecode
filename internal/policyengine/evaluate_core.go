package policyengine

import (
	"fmt"
	"strings"
	"sync"
)

const (
	policyDecisionSchemaID                    = "runecode.protocol.v0.PolicyDecision"
	policyDecisionSchemaVersion               = "0.3.0"
	policyEvaluationDetailsSchemaID           = "runecode.protocol.details.policy.evaluation.v0"
	requiredApprovalHardFloorSchemaID         = "runecode.protocol.details.policy.required_approval.hard_floor.v0"
	requiredApprovalModerateStageSchemaID     = "runecode.protocol.details.policy.required_approval.stage_sign_off.v0"
	requiredApprovalModerateGatewaySchemaID   = "runecode.protocol.details.policy.required_approval.gateway_egress_scope.v0"
	requiredApprovalModerateGitRemoteSchemaID = "runecode.protocol.details.policy.required_approval.git_remote_ops.v0"
	requiredApprovalModerateBackendSchemaID   = "runecode.protocol.details.policy.required_approval.reduced_assurance_backend.v0"
	requiredApprovalModerateGateSchemaID      = "runecode.protocol.details.policy.required_approval.gate_override.v0"
	requiredApprovalModerateWorkspaceSchemaID = "runecode.protocol.details.policy.required_approval.out_of_workspace_write.v0"
	requiredApprovalModerateSecretSchemaID    = "runecode.protocol.details.policy.required_approval.secret_access.v0"
)

type actionRegistries struct {
	actionKinds          map[string]struct{}
	payloadSchemaIDs     map[string]struct{}
	policyReasonCodes    map[string]struct{}
	approvalTriggerCodes map[string]struct{}
}

var (
	actionRegistriesOnce sync.Once
	actionRegistriesData actionRegistries
	actionRegistriesErr  error
)

func loadActionRegistries() (actionRegistries, error) {
	actionRegistriesOnce.Do(func() {
		actionKinds, err := loadRegistryCodes(actionKindRegistryPath)
		if err != nil {
			actionRegistriesErr = err
			return
		}
		payloadSchemaIDs, err := loadRegistryCodes(actionPayloadRegistryPath)
		if err != nil {
			actionRegistriesErr = err
			return
		}
		actionRegistriesData = actionRegistries{actionKinds: actionKinds, payloadSchemaIDs: payloadSchemaIDs}
		policyReasonCodes, err := loadRegistryCodes(policyReasonRegistryPath)
		if err != nil {
			actionRegistriesErr = err
			return
		}
		approvalTriggerCodes, err := loadRegistryCodes(approvalTriggerRegistryPath)
		if err != nil {
			actionRegistriesErr = err
			return
		}
		actionRegistriesData = actionRegistries{
			actionKinds:          actionKinds,
			payloadSchemaIDs:     payloadSchemaIDs,
			policyReasonCodes:    policyReasonCodes,
			approvalTriggerCodes: approvalTriggerCodes,
		}
	})
	if actionRegistriesErr != nil {
		return actionRegistries{}, actionRegistriesErr
	}
	return actionRegistriesData, nil
}

func ensureKnownPolicyReasonCode(code string) error {
	registries, err := loadActionRegistries()
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("load reason-code registries: %v", err)}
	}
	if _, ok := registries.policyReasonCodes[code]; !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown policy_reason_code %q (fail-closed)", code)}
	}
	return nil
}

func ensureKnownApprovalTriggerCode(code string) error {
	registries, err := loadActionRegistries()
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("load trigger-code registries: %v", err)}
	}
	if _, ok := registries.approvalTriggerCodes[code]; !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown approval_trigger_code %q (fail-closed)", code)}
	}
	return nil
}

func actionRelevantArtifactHashes(action ActionRequest) []string {
	out := make([]string, 0, len(action.RelevantArtifactHashes))
	for _, digest := range action.RelevantArtifactHashes {
		identity, err := digest.Identity()
		if err != nil {
			continue
		}
		out = append(out, identity)
	}
	return out
}

func loadRegistryCodes(path string) (map[string]struct{}, error) {
	abs, err := schemaAbsolutePath(path)
	if err != nil {
		return nil, err
	}
	doc := map[string]any{}
	if err := loadJSONFile(abs, &doc); err != nil {
		return nil, err
	}
	rawCodes, ok := doc["codes"].([]any)
	if !ok {
		return nil, fmt.Errorf("registry %q missing codes array", path)
	}
	codes := make(map[string]struct{}, len(rawCodes))
	for _, raw := range rawCodes {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("registry %q contains non-object code entry", path)
		}
		code, ok := entry["code"].(string)
		if !ok || code == "" {
			return nil, fmt.Errorf("registry %q contains code entry without non-empty code", path)
		}
		codes[code] = struct{}{}
	}
	return codes, nil
}

func Evaluate(compiled *CompiledContext, action ActionRequest) (PolicyDecision, error) {
	if compiled == nil {
		return PolicyDecision{}, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: "compiled context is required"}
	}
	if err := validateActionRequest(action); err != nil {
		return PolicyDecision{}, err
	}
	actionHash, err := canonicalHashValue(action)
	if err != nil {
		return PolicyDecision{}, err
	}

	if decision, matched := evaluatePreRuleDecision(compiled, action, actionHash); matched {
		return finalizePolicyDecision(decision)
	}

	moderateApproval, needsModerateApproval := evaluateModerateProfileApproval(compiled, action, actionHash)
	if decision, matched := evaluateRuleSetWithModerateOverride(compiled, action, actionHash, moderateApproval, needsModerateApproval); matched {
		return finalizePolicyDecision(decision)
	}
	if needsModerateApproval {
		return finalizePolicyDecision(moderateApproval)
	}

	return finalizePolicyDecision(defaultAllowDecision(compiled, action, actionHash))
}

func evaluatePreRuleDecision(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if denyDecision, denied := evaluateInvariantDeny(compiled, action, actionHash); denied {
		return denyDecision, true
	}
	if denyDecision, denied := evaluateAbsentCapabilityDeny(compiled, action, actionHash); denied {
		return denyDecision, true
	}
	if denyDecision, denied := evaluateAllowlistPresenceDeny(compiled, action, actionHash); denied {
		return denyDecision, true
	}
	if hardBoundaryDecision, matched := evaluateHardBoundaryInvariants(compiled, action, actionHash); matched {
		return hardBoundaryDecision, true
	}
	return PolicyDecision{}, false
}

func evaluateRuleSetWithModerateOverride(compiled *CompiledContext, action ActionRequest, actionHash string, moderateApproval PolicyDecision, needsModerateApproval bool) (PolicyDecision, bool) {
	if compiled.RuleSet == nil {
		return PolicyDecision{}, false
	}
	decision, matched := evaluateRuleSet(compiled, action, actionHash, *compiled.RuleSet)
	if !matched {
		return PolicyDecision{}, false
	}
	if decision.DecisionOutcome == DecisionAllow && needsModerateApproval {
		return moderateApproval, true
	}
	return decision, true
}

func defaultAllowDecision(compiled *CompiledContext, action ActionRequest, actionHash string) PolicyDecision {
	return PolicyDecision{
		SchemaID:               policyDecisionSchemaID,
		SchemaVersion:          policyDecisionSchemaVersion,
		DecisionOutcome:        DecisionAllow,
		PolicyReasonCode:       "allow_manifest_opt_in",
		ManifestHash:           compiled.ManifestHash,
		PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: actionRelevantArtifactHashes(action),
		DetailsSchemaID:        policyEvaluationDetailsSchemaID,
		Details: map[string]any{
			"precedence":            "invariants_role_run_stage_allowlists",
			"capability_id":         action.CapabilityID,
			"approval_profile":      string(compiled.Context.ApprovalProfile),
			"active_allowlist_refs": append([]string{}, compiled.Context.ActiveAllowlistRefs...),
			"secondary_factors":     []string{},
		},
	}
}

func finalizePolicyDecision(decision PolicyDecision) (PolicyDecision, error) {
	if err := ensureKnownPolicyReasonCode(decision.PolicyReasonCode); err != nil {
		return PolicyDecision{}, err
	}
	if decision.DecisionOutcome != DecisionRequireHumanApproval {
		return decision, nil
	}
	if decision.RequiredApproval == nil {
		return PolicyDecision{}, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: "required_approval missing for require_human_approval decision"}
	}
	trigger, ok := decision.RequiredApproval["approval_trigger_code"].(string)
	if !ok || strings.TrimSpace(trigger) == "" {
		return PolicyDecision{}, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: "required_approval.approval_trigger_code is required"}
	}
	if err := ensureKnownApprovalTriggerCode(trigger); err != nil {
		return PolicyDecision{}, err
	}
	return decision, nil
}

func denyInvariantDecision(compiled *CompiledContext, action ActionRequest, actionHash string, details map[string]any) PolicyDecision {
	return PolicyDecision{
		SchemaID:               policyDecisionSchemaID,
		SchemaVersion:          policyDecisionSchemaVersion,
		DecisionOutcome:        DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           compiled.ManifestHash,
		PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: actionRelevantArtifactHashes(action),
		DetailsSchemaID:        policyEvaluationDetailsSchemaID,
		Details:                details,
	}
}
