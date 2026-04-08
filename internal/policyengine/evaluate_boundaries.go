package policyengine

import "strings"

type executorRunPayload struct {
	SchemaID       string   `json:"schema_id"`
	SchemaVersion  string   `json:"schema_version"`
	ExecutorClass  string   `json:"executor_class"`
	ExecutorID     string   `json:"executor_id"`
	Argv           []string `json:"argv"`
	WorkingDir     string   `json:"working_directory,omitempty"`
	NetworkAccess  string   `json:"network_access,omitempty"`
	TimeoutSeconds *int     `json:"timeout_seconds,omitempty"`
}

type gatewayEgressPayload struct {
	SchemaID        string `json:"schema_id"`
	SchemaVersion   string `json:"schema_version"`
	GatewayRoleKind string `json:"gateway_role_kind"`
	DestinationKind string `json:"destination_kind"`
	DestinationRef  string `json:"destination_ref"`
	EgressDataClass string `json:"egress_data_class"`
	Operation       string `json:"operation,omitempty"`
	PayloadHash     string `json:"payload_hash,omitempty"`
}

type backendPosturePayload struct {
	SchemaID         string `json:"schema_id"`
	SchemaVersion    string `json:"schema_version"`
	BackendClass     string `json:"backend_class"`
	ChangeKind       string `json:"change_kind"`
	RequestedPosture string `json:"requested_posture"`
	RequiresOptIn    bool   `json:"requires_opt_in"`
}

type promotionPayload struct {
	SchemaID            string `json:"schema_id"`
	SchemaVersion       string `json:"schema_version"`
	PromotionKind       string `json:"promotion_kind"`
	TargetDataClass     string `json:"target_data_class"`
	AuthoritativeImport bool   `json:"authoritative_import,omitempty"`
}

func evaluateHardBoundaryInvariants(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if decision, denied := evaluateNoEscalationInPlace(compiled, action, actionHash); denied {
		return decision, true
	}

	if decision, matched := evaluateHardFloorApprovalRequirement(compiled, action, actionHash); matched {
		return decision, true
	}

	if compiled.Context.ActiveRoleFamily == "gateway" {
		if decision, denied := evaluateGatewayNoWorkspaceAccess(compiled, action, actionHash); denied {
			return decision, true
		}
	}

	switch action.ActionKind {
	case ActionKindGatewayEgress, ActionKindDependencyFetch:
		decision, matched := evaluateGatewayBoundary(compiled, action, actionHash)
		return decision, matched
	case ActionKindExecutorRun:
		decision, matched := evaluateExecutorBoundary(compiled, action, actionHash)
		return decision, matched
	case ActionKindBackendPosture:
		decision, matched := evaluateBackendSelectionRules(compiled, action, actionHash)
		return decision, matched
	default:
		return PolicyDecision{}, false
	}
}

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

func evaluateHardFloorApprovalRequirement(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	classes, floor := classifyHardFloorOperation(action, nil)
	if len(classes) == 0 {
		return PolicyDecision{}, false
	}
	return hardFloorApprovalDecision(compiled, action, actionHash, classes, floor), true
}

func evaluateNoEscalationInPlace(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if action.RoleFamily != "" && action.RoleFamily != compiled.Context.ActiveRoleFamily {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":                 "invariants_first",
			"invariant":                  "no_escalation_in_place",
			"non_approvable":             true,
			"requested_role_family":      action.RoleFamily,
			"active_context_role_family": compiled.Context.ActiveRoleFamily,
		}), true
	}
	if action.RoleKind != "" && action.RoleKind != compiled.Context.ActiveRoleKind {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":               "invariants_first",
			"invariant":                "no_escalation_in_place",
			"non_approvable":           true,
			"requested_role_kind":      action.RoleKind,
			"active_context_role_kind": compiled.Context.ActiveRoleKind,
		}), true
	}
	return PolicyDecision{}, false
}

func evaluateGatewayNoWorkspaceAccess(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	switch action.ActionKind {
	case ActionKindWorkspaceWrite, ActionKindExecutorRun:
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "gateway_no_workspace_access",
			"non_approvable":   true,
			"active_role_kind": compiled.Context.ActiveRoleKind,
			"action_kind":      action.ActionKind,
		}), true
	default:
		return PolicyDecision{}, false
	}
}
