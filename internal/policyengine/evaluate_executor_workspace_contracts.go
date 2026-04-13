package policyengine

type typedExecutorContract struct {
	ID               string
	AllowedRoles     map[string]struct{}
	AllowedClass     string
	AllowedNetwork   map[string]struct{}
	AllowedEnvKeys   map[string]struct{}
	AllowEmptyEnv    bool
	RequireWorkspace bool
	AllowEnvWrapper  bool
	AllowedArgvHeads [][]string
	MaxArgvItems     int
	MaxTimeoutSecs   int
}

type workspaceExecutorContractValidator func(*CompiledContext, ActionRequest, string, executorRunPayload, typedExecutorContract) (PolicyDecision, bool)

func evaluateWorkspaceRoleExecutorMatrix(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload) (PolicyDecision, bool) {
	if compiled.Context.ActiveRoleFamily != "workspace" {
		return PolicyDecision{}, false
	}
	contract, decision, known := resolveWorkspaceExecutorContract(compiled, action, actionHash, payload.ExecutorID)
	if !known {
		return decision, true
	}
	validators := []workspaceExecutorContractValidator{
		denyWorkspaceExecutorRoleMismatch,
		denyWorkspaceExecutorClassMismatch,
		denyWorkspaceExecutorNetworkMismatch,
		denyWorkspaceExecutorWorkingDirMismatch,
		denyWorkspaceExecutorArgvLengthMismatch,
		denyWorkspaceExecutorTimeoutMismatch,
		denyWorkspaceExecutorArgvShapeMismatch,
		denyWorkspaceExecutorSecretMaterialLeakage,
		denyWorkspaceExecutorEnvironmentMismatch,
	}
	for _, validate := range validators {
		if decision, blocked := validate(compiled, action, actionHash, payload, contract); blocked {
			return decision, true
		}
	}
	return PolicyDecision{}, false
}

func resolveWorkspaceExecutorContract(compiled *CompiledContext, action ActionRequest, actionHash string, executorID string) (typedExecutorContract, PolicyDecision, bool) {
	contract, known := workspaceExecutorContractByID(executorID)
	if known {
		return contract, PolicyDecision{}, true
	}
	return typedExecutorContract{}, denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":       "invariants_first",
		"invariant":        "workspace_executor_contract_matrix",
		"non_approvable":   true,
		"active_role_kind": compiled.Context.ActiveRoleKind,
		"executor_id":      executorID,
		"reason":           "unknown_executor_id_fail_closed",
	}), false
}

func denyWorkspaceExecutorRoleMismatch(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload, contract typedExecutorContract) (PolicyDecision, bool) {
	if _, ok := contract.AllowedRoles[compiled.Context.ActiveRoleKind]; ok {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":       "invariants_first",
		"invariant":        "workspace_executor_contract_matrix",
		"non_approvable":   true,
		"active_role_kind": compiled.Context.ActiveRoleKind,
		"executor_id":      payload.ExecutorID,
		"reason":           "executor_id_not_allowed_for_workspace_role",
	}), true
}

func denyWorkspaceExecutorClassMismatch(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload, contract typedExecutorContract) (PolicyDecision, bool) {
	if payload.ExecutorClass == contract.AllowedClass {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":       "invariants_first",
		"invariant":        "workspace_executor_contract_matrix",
		"non_approvable":   true,
		"active_role_kind": compiled.Context.ActiveRoleKind,
		"executor_id":      payload.ExecutorID,
		"executor_class":   payload.ExecutorClass,
		"required_class":   contract.AllowedClass,
		"reason":           "executor_class_not_allowed_for_executor_id",
	}), true
}

func denyWorkspaceExecutorNetworkMismatch(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload, contract typedExecutorContract) (PolicyDecision, bool) {
	_, networkAllowed := contract.AllowedNetwork[payload.NetworkAccess]
	_, emptyMapsToNone := contract.AllowedNetwork["none"]
	if networkAllowed || (payload.NetworkAccess == "" && emptyMapsToNone) {
		return PolicyDecision{}, false
	}
	details := map[string]any{
		"precedence":                    "invariants_first",
		"invariant":                     "workspace_executor_contract_matrix",
		"non_approvable":                true,
		"active_role_kind":              compiled.Context.ActiveRoleKind,
		"executor_id":                   payload.ExecutorID,
		"network_access":                payload.NetworkAccess,
		"reason":                        "network_access_not_allowed_for_executor_id",
		"workspace_offline_only":        true,
		"required_cross_boundary_route": "artifact_io",
		"artifact_route_actions":        []string{ActionKindArtifactRead},
	}
	return denyInvariantDecision(compiled, action, actionHash, details), true
}

func denyWorkspaceExecutorWorkingDirMismatch(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload, contract typedExecutorContract) (PolicyDecision, bool) {
	if !contract.RequireWorkspace {
		return PolicyDecision{}, false
	}
	if payload.WorkingDir == "" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "workspace_executor_contract_matrix",
			"non_approvable":   true,
			"active_role_kind": compiled.Context.ActiveRoleKind,
			"executor_id":      payload.ExecutorID,
			"reason":           "working_directory_required_but_missing",
		}), true
	}
	if isWorkspaceRelativePath(payload.WorkingDir) {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "workspace_executor_contract_matrix",
		"non_approvable":    true,
		"active_role_kind":  compiled.Context.ActiveRoleKind,
		"executor_id":       payload.ExecutorID,
		"working_directory": payload.WorkingDir,
		"reason":            "working_directory_not_workspace_scoped",
	}), true
}

func denyWorkspaceExecutorArgvLengthMismatch(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload, contract typedExecutorContract) (PolicyDecision, bool) {
	if contract.MaxArgvItems <= 0 || len(payload.Argv) <= contract.MaxArgvItems {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":       "invariants_first",
		"invariant":        "workspace_executor_contract_matrix",
		"non_approvable":   true,
		"active_role_kind": compiled.Context.ActiveRoleKind,
		"executor_id":      payload.ExecutorID,
		"argv_len":         len(payload.Argv),
		"reason":           "argv_too_long_for_executor_contract",
	}), true
}

func denyWorkspaceExecutorTimeoutMismatch(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload, contract typedExecutorContract) (PolicyDecision, bool) {
	if contract.MaxTimeoutSecs <= 0 || payload.TimeoutSeconds == nil || *payload.TimeoutSeconds <= contract.MaxTimeoutSecs {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":       "invariants_first",
		"invariant":        "workspace_executor_contract_matrix",
		"non_approvable":   true,
		"active_role_kind": compiled.Context.ActiveRoleKind,
		"executor_id":      payload.ExecutorID,
		"timeout_seconds":  *payload.TimeoutSeconds,
		"reason":           "timeout_exceeds_executor_contract",
	}), true
}

func denyWorkspaceExecutorArgvShapeMismatch(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload, contract typedExecutorContract) (PolicyDecision, bool) {
	if err := validateExecutorArgvShape(payload.Argv, contract); err != nil {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "workspace_executor_contract_matrix",
			"non_approvable":   true,
			"active_role_kind": compiled.Context.ActiveRoleKind,
			"executor_id":      payload.ExecutorID,
			"reason":           "argv_shape_invalid_for_executor_contract",
			"argv_error":       err.Error(),
		}), true
	}
	return PolicyDecision{}, false
}

func denyWorkspaceExecutorEnvironmentMismatch(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload, contract typedExecutorContract) (PolicyDecision, bool) {
	if err := validateExecutorEnvironmentShape(payload.Environment, contract); err != nil {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "workspace_executor_contract_matrix",
			"non_approvable":   true,
			"active_role_kind": compiled.Context.ActiveRoleKind,
			"executor_id":      payload.ExecutorID,
			"reason":           "environment_shape_invalid_for_executor_contract",
			"env_error":        err.Error(),
		}), true
	}
	return PolicyDecision{}, false
}
