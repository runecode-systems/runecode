package policyengine

import "strings"

func evaluateBackendSelectionRules(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	payload := backendPosturePayload{}
	if err := decodeActionPayload(action.ActionPayload, &payload); err != nil {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "backend_selection_rules",
			"non_approvable":    true,
			"payload_parse_err": err.Error(),
		}), true
	}
	if payload.BackendClass == "microvm" {
		return evaluateMicroVMBackendSelection(compiled, action, actionHash, payload)
	}
	if payload.BackendClass == "container" {
		return evaluateContainerBackendSelection(compiled, action, actionHash, payload)
	}
	return PolicyDecision{}, false
}

func evaluateMicroVMBackendSelection(compiled *CompiledContext, action ActionRequest, actionHash string, payload backendPosturePayload) (PolicyDecision, bool) {
	if strings.Contains(strings.ToLower(payload.RequestedPosture), "fallback") {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "backend_selection_rules",
			"non_approvable":    true,
			"backend_class":     payload.BackendClass,
			"requested_posture": payload.RequestedPosture,
			"reason":            "automatic_fallback_not_allowed",
			"secondary_factors": []string{"microvm_default_backend_when_available"},
		}), true
	}
	return backendDecision(compiled, action, actionHash, DecisionAllow, "allow_microvm_default", map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "backend_selection_rules",
		"backend_class":     payload.BackendClass,
		"change_kind":       payload.ChangeKind,
		"requested_posture": payload.RequestedPosture,
		"secondary_factors": []string{"microvm_default_backend_when_available"},
	}), true
}

func evaluateContainerBackendSelection(compiled *CompiledContext, action ActionRequest, actionHash string, payload backendPosturePayload) (PolicyDecision, bool) {
	requestedPosture := strings.ToLower(payload.RequestedPosture)
	if strings.Contains(requestedPosture, "fallback") {
		return backendDecision(compiled, action, actionHash, DecisionDeny, "deny_container_automatic_fallback", map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "backend_selection_rules",
			"non_approvable":    true,
			"backend_class":     payload.BackendClass,
			"requested_posture": payload.RequestedPosture,
			"secondary_factors": []string{"no_automatic_microvm_to_container_fallback"},
		}), true
	}
	if !payload.RequiresOptIn {
		return backendDecision(compiled, action, actionHash, DecisionDeny, "deny_container_opt_in_required", map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "backend_selection_rules",
			"non_approvable":    true,
			"backend_class":     payload.BackendClass,
			"requires_opt_in":   payload.RequiresOptIn,
			"secondary_factors": []string{"container_backend_requires_explicit_opt_in"},
		}), true
	}
	reqSchemaID, reqPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
	if reqSchemaID == "" {
		return PolicyDecision{}, false
	}
	return policyApprovalDecision(compiled, action, actionHash, "approval_required", reqSchemaID, reqPayload, map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "backend_selection_rules",
		"backend_class":     payload.BackendClass,
		"change_kind":       payload.ChangeKind,
		"requested_posture": payload.RequestedPosture,
		"approval_profile":  string(compiled.Context.ApprovalProfile),
		"secondary_factors": []string{"container_backend_explicit_opt_in_requires_approval", "approval_must_be_audited"},
	}), true
}

func backendDecision(compiled *CompiledContext, action ActionRequest, actionHash string, outcome DecisionOutcome, reasonCode string, details map[string]any) PolicyDecision {
	return PolicyDecision{
		SchemaID:               policyDecisionSchemaID,
		SchemaVersion:          policyDecisionSchemaVersion,
		DecisionOutcome:        outcome,
		PolicyReasonCode:       reasonCode,
		ManifestHash:           compiled.ManifestHash,
		PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: actionRelevantArtifactHashes(action),
		DetailsSchemaID:        policyEvaluationDetailsSchemaID,
		Details:                details,
	}
}
