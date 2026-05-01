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
		return requiredApprovalModerateBackendSchemaID, moderateBackendApprovalPayload(base, action)
	case ActionKindGatewayEgress, ActionKindDependencyFetch:
		return requiredApprovalForModerateGateway(base, action)
	case ActionKindWorkspaceWrite:
		return requiredApprovalForModerateWorkspace(base, action)
	case ActionKindSecretAccess:
		return requiredApprovalModerateSecretSchemaID, moderateSecretApprovalPayload(base)
	}

	return "", nil
}

func requiredApprovalForModerateGateway(base map[string]any, action ActionRequest) (string, map[string]any) {
	if !isModerateGatewayCheckpointAction(action) {
		return "", nil
	}
	payload := gatewayModerateApprovalPayload(base, action)
	if payload == nil {
		return "", nil
	}
	if trigger, _ := payload["approval_trigger_code"].(string); trigger == "git_remote_ops" {
		return requiredApprovalModerateGitRemoteSchemaID, payload
	}
	return requiredApprovalModerateGatewaySchemaID, payload
}

func requiredApprovalForModerateWorkspace(base map[string]any, action ActionRequest) (string, map[string]any) {
	targetPath, _ := action.ActionPayload["target_path"].(string)
	if isWorkspaceRelativePath(targetPath) {
		return "", nil
	}
	return requiredApprovalModerateWorkspaceSchemaID, moderateWorkspaceWriteApprovalPayload(base)
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

func moderateBackendApprovalPayload(base map[string]any, action ActionRequest) map[string]any {
	payload := cloneMap(base)
	targetInstanceID, _ := action.ActionPayload["target_instance_id"].(string)
	targetBackendKind, _ := action.ActionPayload["target_backend_kind"].(string)
	selectionMode, _ := action.ActionPayload["selection_mode"].(string)
	changeKind, _ := action.ActionPayload["change_kind"].(string)
	assuranceChangeKind, _ := action.ActionPayload["assurance_change_kind"].(string)
	optInKind, _ := action.ActionPayload["opt_in_kind"].(string)
	reducedAssuranceAcknowledged, _ := action.ActionPayload["reduced_assurance_acknowledged"].(bool)
	payload["approval_trigger_code"] = "reduced_assurance_backend"
	payload["approval_assurance_level"] = string(ApprovalAssuranceReauthenticated)
	payload["why_required"] = "Moderate profile requires explicit approval for reduced-assurance backend opt-ins."
	payload["changes_if_approved"] = "Reduced-assurance backend posture change may be applied."
	payload["security_posture_impact"] = "high"
	payload["future_launches_only"] = true
	payload["existing_isolates_unaffected"] = true
	payload["details"] = map[string]any{
		"target_instance_id":             targetInstanceID,
		"target_backend_kind":            targetBackendKind,
		"selection_mode":                 selectionMode,
		"change_kind":                    changeKind,
		"assurance_change_kind":          assuranceChangeKind,
		"opt_in_kind":                    optInKind,
		"reduced_assurance_acknowledged": reducedAssuranceAcknowledged,
		"requested_posture":              "container_mode_explicit_opt_in",
	}
	return payload
}

func gatewayModerateApprovalPayload(base map[string]any, action ActionRequest) map[string]any {
	payload := cloneMap(base)
	if action.ActionKind == ActionKindDependencyFetch {
		return moderateGatewayScopeApprovalPayload(payload, "dependency_network_fetch")
	}

	operation, _ := action.ActionPayload["operation"].(string)
	if operation == "external_anchor_submit" {
		return moderateExternalAnchorApprovalPayload(payload)
	}
	if isGatewayRemoteMutationOperation(operation) {
		gitPayload, ok := moderateGitRemoteApprovalPayload(payload, action)
		if !ok {
			return nil
		}
		return gitPayload
	}

	return moderateGatewayScopeApprovalPayload(payload, "gateway_egress_scope_change")
}

func moderateGatewayScopeApprovalPayload(payload map[string]any, trigger string) map[string]any {
	payload["approval_trigger_code"] = trigger
	payload["checkpoint_scope"] = "gateway_or_dependency_scope_change"
	payload["why_required"] = "Moderate profile requires checkpoint approval only when enabling or expanding gateway/dependency scope."
	payload["changes_if_approved"] = "Gateway egress action can proceed for the bound request and manifest context."
	payload["security_posture_impact"] = "high"
	return payload
}

func moderateExternalAnchorApprovalPayload(payload map[string]any) map[string]any {
	payload["approval_trigger_code"] = "external_anchor_opt_in"
	payload["approval_assurance_level"] = string(ApprovalAssuranceReauthenticated)
	payload["checkpoint_scope"] = "gateway_remote_state_mutation"
	payload["why_required"] = "External anchoring is disabled by default and requires explicit signed-manifest opt-in plus exact final approval."
	payload["changes_if_approved"] = "One exact external anchor submission may proceed for the bound typed request hash and canonical target descriptor identity."
	payload["security_posture_impact"] = "high"
	payload["required_final_exact_approval"] = true
	payload["stage_sign_off_is_prerequisite_only"] = true
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
	if isGatewayRemoteMutationOperation(operation) {
		return true
	}
	return isGatewayScopeChangeOperation(operation)
}
