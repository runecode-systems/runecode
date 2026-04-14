package policyengine

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
	if payload.TargetBackendKind == "microvm" {
		return evaluateMicroVMBackendSelection(compiled, action, actionHash, payload)
	}
	if payload.TargetBackendKind == "container" {
		return evaluateContainerBackendSelection(compiled, action, actionHash, payload)
	}
	return PolicyDecision{}, false
}

func evaluateMicroVMBackendSelection(compiled *CompiledContext, action ActionRequest, actionHash string, payload backendPosturePayload) (PolicyDecision, bool) {
	if payload.SelectionMode == "automatic_fallback_attempt" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":          "invariants_first",
			"invariant":           "backend_selection_rules",
			"non_approvable":      true,
			"target_backend_kind": payload.TargetBackendKind,
			"selection_mode":      payload.SelectionMode,
			"reason":              "automatic_fallback_not_allowed",
			"secondary_factors":   []string{"microvm_default_backend_when_available"},
		}), true
	}
	return backendDecision(compiled, action, actionHash, DecisionAllow, "allow_microvm_default", map[string]any{
		"precedence":            "invariants_first",
		"invariant":             "backend_selection_rules",
		"target_backend_kind":   payload.TargetBackendKind,
		"selection_mode":        payload.SelectionMode,
		"change_kind":           payload.ChangeKind,
		"assurance_change_kind": payload.AssuranceChangeKind,
		"opt_in_kind":           payload.OptInKind,
		"secondary_factors":     []string{"microvm_default_backend_when_available"},
	}), true
}

func evaluateContainerBackendSelection(compiled *CompiledContext, action ActionRequest, actionHash string, payload backendPosturePayload) (PolicyDecision, bool) {
	if payload.SelectionMode == "automatic_fallback_attempt" {
		return denyContainerSelectionDecision(compiled, action, actionHash, payload, "deny_container_automatic_fallback", map[string]any{
			"selection_mode":    payload.SelectionMode,
			"secondary_factors": []string{"no_automatic_microvm_to_container_fallback"},
		}), true
	}
	if payload.OptInKind != "exact_action_approval" {
		return denyContainerSelectionDecision(compiled, action, actionHash, payload, "deny_container_opt_in_required", map[string]any{
			"opt_in_kind":       payload.OptInKind,
			"secondary_factors": []string{"container_backend_requires_explicit_opt_in"},
		}), true
	}
	if !payload.ReducedAssuranceAcknowledged {
		return denyContainerSelectionDecision(compiled, action, actionHash, payload, "deny_container_acknowledgment_required", map[string]any{
			"reduced_assurance_acknowledged": payload.ReducedAssuranceAcknowledged,
			"secondary_factors":              []string{"container_backend_requires_explicit_reduced_assurance_acknowledgment"},
		}), true
	}
	reqSchemaID, reqPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
	if reqSchemaID == "" {
		return denyContainerSelectionDecision(compiled, action, actionHash, payload, "deny_container_opt_in_required", map[string]any{
			"selection_mode":   payload.SelectionMode,
			"approval_profile": string(compiled.Context.ApprovalProfile),
			"reason":           "approval_profile_missing_backend_opt_in_path",
		}), true
	}
	return policyApprovalDecision(compiled, action, actionHash, "approval_required", reqSchemaID, reqPayload, map[string]any{
		"precedence":                     "invariants_first",
		"invariant":                      "backend_selection_rules",
		"target_backend_kind":            payload.TargetBackendKind,
		"selection_mode":                 payload.SelectionMode,
		"change_kind":                    payload.ChangeKind,
		"assurance_change_kind":          payload.AssuranceChangeKind,
		"opt_in_kind":                    payload.OptInKind,
		"reduced_assurance_acknowledged": payload.ReducedAssuranceAcknowledged,
		"approval_profile":               string(compiled.Context.ApprovalProfile),
		"secondary_factors":              []string{"container_backend_explicit_opt_in_requires_approval", "approval_must_be_audited"},
	}), true
}

func denyContainerSelectionDecision(compiled *CompiledContext, action ActionRequest, actionHash string, payload backendPosturePayload, reasonCode string, extra map[string]any) PolicyDecision {
	details := map[string]any{
		"precedence":          "invariants_first",
		"invariant":           "backend_selection_rules",
		"non_approvable":      true,
		"target_backend_kind": payload.TargetBackendKind,
	}
	for key, value := range extra {
		details[key] = value
	}
	return backendDecision(compiled, action, actionHash, DecisionDeny, reasonCode, details)
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
