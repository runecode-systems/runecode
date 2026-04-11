package policyengine

import "strings"

func evaluateHardBoundaryInvariants(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if decision, denied := evaluateNoEscalationInPlace(compiled, action, actionHash); denied {
		return decision, true
	}
	if decision, denied := evaluateWorkspaceRoleActionMatrix(compiled, action, actionHash); denied {
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
	case ActionKindWorkspaceWrite:
		decision, matched := evaluateWorkspaceWriteBoundary(compiled, action, actionHash)
		return decision, matched
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

func evaluateWorkspaceWriteBoundary(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if compiled.Context.ActiveRoleFamily != "workspace" {
		return PolicyDecision{}, false
	}
	targetPath, _ := action.ActionPayload["target_path"].(string)
	if compiled.Context.ActiveRoleKind == "workspace-test" && !isWorkspaceTestWritablePath(targetPath) {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "workspace_write_role_boundary",
			"non_approvable":   true,
			"active_role_kind": compiled.Context.ActiveRoleKind,
			"target_path":      targetPath,
			"reason":           "workspace_test_write_outside_build_output",
		}), true
	}
	return PolicyDecision{}, false
}

func isWorkspaceTestWritablePath(targetPath string) bool {
	if !isWorkspaceRelativePath(targetPath) {
		return false
	}
	normalized := strings.ToLower(strings.ReplaceAll(targetPath, "\\", "/"))
	allowedPrefixes := []string{"build-output/", "build/", "dist/", "out/", ".rune/build-output/"}
	for _, prefix := range allowedPrefixes {
		if normalized == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

func evaluateWorkspaceRoleActionMatrix(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if compiled.Context.ActiveRoleFamily != "workspace" {
		return PolicyDecision{}, false
	}
	if !workspaceRoleMatrixActionKind(action.ActionKind) {
		return PolicyDecision{}, false
	}

	allowedActions, knownRoleKind := workspaceAllowedActionsByRole()[compiled.Context.ActiveRoleKind]
	if !knownRoleKind {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "workspace_role_action_matrix",
			"non_approvable":   true,
			"active_role_kind": compiled.Context.ActiveRoleKind,
			"reason":           "unknown_workspace_role_kind_fail_closed",
		}), true
	}

	if _, ok := allowedActions[action.ActionKind]; ok {
		return PolicyDecision{}, false
	}

	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":       "invariants_first",
		"invariant":        "workspace_role_action_matrix",
		"non_approvable":   true,
		"active_role_kind": compiled.Context.ActiveRoleKind,
		"action_kind":      action.ActionKind,
		"reason":           "action_kind_not_allowed_for_workspace_role",
	}), true
}

func workspaceRoleMatrixActionKind(actionKind string) bool {
	switch actionKind {
	case ActionKindWorkspaceWrite, ActionKindArtifactRead, ActionKindExecutorRun:
		return true
	default:
		return false
	}
}

func workspaceAllowedActionsByRole() map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		"workspace-read": {
			ActionKindArtifactRead: {},
		},
		"workspace-edit": {
			ActionKindWorkspaceWrite: {},
			ActionKindArtifactRead:   {},
			ActionKindExecutorRun:    {},
		},
		"workspace-test": {
			ActionKindWorkspaceWrite: {},
			ActionKindArtifactRead:   {},
			ActionKindExecutorRun:    {},
		},
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
