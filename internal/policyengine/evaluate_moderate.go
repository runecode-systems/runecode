package policyengine

func evaluateModerateProfileApproval(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if compiled.Context.ApprovalProfile != ApprovalProfileModerate {
		return PolicyDecision{}, false
	}

	if details, ok := moderateCheckpointDetails(action.ActionKind); ok {
		reqSchemaID, reqPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
		if reqSchemaID == "" {
			return PolicyDecision{}, false
		}
		return policyApprovalDecision(compiled, action, actionHash, "approval_required", reqSchemaID, reqPayload, details), true
	}

	return PolicyDecision{}, false
}

func moderateCheckpointDetails(actionKind string) (map[string]any, bool) {
	base := map[string]any{
		"precedence":       "approval_profile_moderate",
		"checkpoint_model": "",
	}
	switch actionKind {
	case ActionKindStageSummarySign:
		base["checkpoint_model"] = "stage_sign_off"
		return base, true
	case ActionKindGatewayEgress, ActionKindDependencyFetch:
		base["checkpoint_model"] = "scope_checkpoint"
		return base, true
	case ActionKindSecretAccess:
		base["checkpoint_model"] = "secret_checkpoint"
		return base, true
	case ActionKindGateOverride:
		base["checkpoint_model"] = "gate_override_checkpoint"
		return base, true
	case ActionKindBackendPosture:
		base["checkpoint_model"] = "backend_posture_checkpoint"
		return base, true
	case ActionKindWorkspaceWrite:
		delete(base, "checkpoint_model")
		return base, true
	default:
		return nil, false
	}
}

func policyApprovalDecision(compiled *CompiledContext, action ActionRequest, actionHash, reasonCode, requiredSchemaID string, requiredPayload map[string]any, details map[string]any) PolicyDecision {
	outDetails := map[string]any{
		"approval_profile": string(compiled.Context.ApprovalProfile),
	}
	for k, v := range details {
		outDetails[k] = v
	}
	return PolicyDecision{
		SchemaID:                 policyDecisionSchemaID,
		SchemaVersion:            policyDecisionSchemaVersion,
		DecisionOutcome:          DecisionRequireHumanApproval,
		PolicyReasonCode:         reasonCode,
		ManifestHash:             compiled.ManifestHash,
		PolicyInputHashes:        append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:        actionHash,
		RelevantArtifactHashes:   actionRelevantArtifactHashes(action),
		DetailsSchemaID:          policyEvaluationDetailsSchemaID,
		Details:                  outDetails,
		RequiredApprovalSchemaID: requiredSchemaID,
		RequiredApproval:         requiredPayload,
	}
}

func requiredApprovalForModerateProfile(compiled *CompiledContext, action ActionRequest, actionHash string) (string, map[string]any) {
	base := baseModerateApprovalPayload(compiled, action, actionHash)
	switch action.ActionKind {
	case ActionKindStageSummarySign:
		return requiredApprovalModerateStageSchemaID, moderateStageApprovalPayload(base)
	case ActionKindGateOverride:
		return requiredApprovalModerateGateSchemaID, moderateGateOverrideApprovalPayload(base)
	case ActionKindBackendPosture:
		return requiredApprovalModerateBackendSchemaID, moderateBackendApprovalPayload(base)
	case ActionKindGatewayEgress, ActionKindDependencyFetch:
		if !isModerateGatewayCheckpointAction(action) {
			return "", nil
		}
		return requiredApprovalModerateGatewaySchemaID, moderateGatewayApprovalPayload(base, action.ActionKind)
	case ActionKindWorkspaceWrite:
		if targetPath, ok := action.ActionPayload["target_path"].(string); ok {
			if !isWorkspaceRelativePath(targetPath) {
				return requiredApprovalModerateWorkspaceSchemaID, moderateWorkspaceWriteApprovalPayload(base)
			}
		}
	case ActionKindSecretAccess:
		return requiredApprovalModerateSecretSchemaID, moderateSecretApprovalPayload(base)
	}

	return "", nil
}

func baseModerateApprovalPayload(compiled *CompiledContext, action ActionRequest, actionHash string) map[string]any {
	return map[string]any{
		"approval_assurance_level":          string(ApprovalAssuranceSessionAuthenticated),
		"scope":                             approvalScopeForAction(action),
		"effects_if_denied_or_deferred":     "Action remains blocked until an approval decision is provided.",
		"blocked_work":                      []string{"action_execution"},
		"approval_ttl_seconds":              1800,
		"approval_assertion_hash_supported": true,
		"related_hashes": map[string]any{
			"manifest_hash":            compiled.ManifestHash,
			"action_request_hash":      actionHash,
			"policy_input_hashes":      append([]string{}, compiled.PolicyInputHashes...),
			"relevant_artifact_hashes": actionRelevantArtifactHashes(action),
		},
	}
}

func moderateStageApprovalPayload(base map[string]any) map[string]any {
	payload := cloneMap(base)
	payload["approval_trigger_code"] = "stage_sign_off"
	payload["why_required"] = "Moderate profile requires stage checkpoint sign-off before proceeding."
	payload["changes_if_approved"] = "Stage summary is signed off for this exact summary hash."
	payload["security_posture_impact"] = "moderate"
	payload["stage_summary_staleness_posture"] = "invalidate_on_bound_input_change"
	return payload
}

func moderateGateOverrideApprovalPayload(base map[string]any) map[string]any {
	payload := cloneMap(base)
	payload["approval_trigger_code"] = "gate_override"
	payload["approval_assurance_level"] = string(ApprovalAssuranceReauthenticated)
	payload["why_required"] = "Moderate profile requires explicit approval for gate overrides."
	payload["changes_if_approved"] = "Gate override can be consumed once for this exact action request hash."
	payload["security_posture_impact"] = "high"
	return payload
}

func moderateBackendApprovalPayload(base map[string]any) map[string]any {
	payload := cloneMap(base)
	payload["approval_trigger_code"] = "reduced_assurance_backend"
	payload["approval_assurance_level"] = string(ApprovalAssuranceReauthenticated)
	payload["why_required"] = "Moderate profile requires explicit approval for reduced-assurance backend opt-ins."
	payload["changes_if_approved"] = "Reduced-assurance backend posture change may be applied."
	payload["security_posture_impact"] = "high"
	return payload
}

func moderateGatewayApprovalPayload(base map[string]any, kind string) map[string]any {
	payload := cloneMap(base)
	payload["approval_trigger_code"] = "gateway_egress_scope_change"
	if kind == ActionKindDependencyFetch {
		payload["approval_trigger_code"] = "dependency_network_fetch"
	}
	payload["checkpoint_scope"] = "gateway_or_dependency_scope_change"
	payload["why_required"] = "Moderate profile requires checkpoint approval only when enabling or expanding gateway/dependency scope."
	payload["changes_if_approved"] = "Gateway egress action can proceed for the bound request and manifest context."
	payload["security_posture_impact"] = "high"
	return payload
}

func moderateWorkspaceWriteApprovalPayload(base map[string]any) map[string]any {
	payload := cloneMap(base)
	payload["approval_trigger_code"] = "out_of_workspace_write"
	payload["why_required"] = "Moderate profile requires approval for writes outside workspace allowlist."
	payload["changes_if_approved"] = "Out-of-workspace write can proceed for this exact action request hash."
	payload["security_posture_impact"] = "high"
	return payload
}

func moderateSecretApprovalPayload(base map[string]any) map[string]any {
	payload := cloneMap(base)
	payload["approval_trigger_code"] = "secret_access_lease"
	payload["approval_assurance_level"] = string(ApprovalAssuranceReauthenticated)
	payload["why_required"] = "Moderate profile requires explicit approval for secret lease issue, renew, and revoke operations."
	payload["changes_if_approved"] = "Secret lease operation can proceed once for this exact action request hash."
	payload["security_posture_impact"] = "high"
	return payload
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func isModerateGatewayCheckpointAction(action ActionRequest) bool {
	operation, _ := action.ActionPayload["operation"].(string)
	scopeCheckpointOps := map[string]struct{}{
		"enable_gateway":          {},
		"expand_scope":            {},
		"change_allowlist":        {},
		"enable_dependency_fetch": {},
	}
	_, ok := scopeCheckpointOps[operation]
	return ok
}
