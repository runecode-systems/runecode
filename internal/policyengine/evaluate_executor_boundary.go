package policyengine

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

var systemModifyingExecutableNames = map[string]struct{}{
	"apt": {}, "apt-get": {}, "yum": {}, "dnf": {}, "apk": {}, "pacman": {}, "brew": {},
	"systemctl": {}, "service": {}, "modprobe": {}, "sysctl": {}, "mount": {}, "umount": {},
	"iptables": {}, "ufw": {}, "nft": {}, "netsh": {}, "sc": {},
	"docker": {}, "podman": {}, "kubectl": {}, "helm": {},
}

func evaluateExecutorBoundary(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	payload, decision, blocked := decodeExecutorPayload(compiled, action, actionHash)
	if blocked {
		return decision, true
	}
	if decision, blocked := evaluateWorkspaceRoleExecutorMatrix(compiled, action, actionHash, payload); blocked {
		return decision, true
	}
	if decision, blocked := denyExecutorForNetworkAccess(compiled, action, actionHash, payload); blocked {
		return decision, true
	}
	if payload.ExecutorClass == "workspace_ordinary" {
		return evaluateWorkspaceOrdinaryExecutor(compiled, action, actionHash, payload)
	}
	return evaluateExecutorHardFloor(compiled, action, actionHash, payload)
}

func decodeExecutorPayload(compiled *CompiledContext, action ActionRequest, actionHash string) (executorRunPayload, PolicyDecision, bool) {
	payload := executorRunPayload{}
	if err := decodeActionPayload(action.ActionPayload, &payload); err != nil {
		return executorRunPayload{}, denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "deny_by_default_shell",
			"non_approvable":    true,
			"payload_parse_err": err.Error(),
		}), true
	}
	return payload, PolicyDecision{}, false
}

func denyExecutorForNetworkAccess(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload) (PolicyDecision, bool) {
	if payload.NetworkAccess == "" || payload.NetworkAccess == "none" {
		return PolicyDecision{}, false
	}
	details := map[string]any{
		"precedence":       "invariants_first",
		"invariant":        "network_egress_hard_boundary",
		"non_approvable":   true,
		"network_access":   payload.NetworkAccess,
		"required_network": "none",
		"action_kind":      ActionKindExecutorRun,
	}
	if compiled.Context.ActiveRoleFamily == "workspace" {
		details["workspace_offline_only"] = true
		details["required_cross_boundary_route"] = "artifact_io"
		details["artifact_route_actions"] = []string{ActionKindArtifactRead}
	}
	return denyInvariantDecision(compiled, action, actionHash, details), true
}

func evaluateWorkspaceOrdinaryExecutor(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload) (PolicyDecision, bool) {
	if isSystemModifyingArgv(payload.Argv) {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":     "invariants_first",
			"invariant":      "ordinary_workspace_executor_constraints",
			"non_approvable": true,
			"reason":         "system_modifying_execution_not_ordinary",
		}), true
	}
	if decision, blocked := denyExecutorForOutOfWorkspaceDir(compiled, action, actionHash, payload); blocked {
		return decision, true
	}
	if decision, blocked := denyExecutorForRawShell(compiled, action, actionHash, payload); blocked {
		return decision, true
	}
	return PolicyDecision{}, false
}

func denyExecutorForOutOfWorkspaceDir(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload) (PolicyDecision, bool) {
	if payload.WorkingDir == "" || isWorkspaceRelativePath(payload.WorkingDir) {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "ordinary_workspace_executor_constraints",
		"non_approvable":    true,
		"working_directory": payload.WorkingDir,
		"workspace_scoped":  false,
	}), true
}

func denyExecutorForRawShell(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload) (PolicyDecision, bool) {
	if !isRawShellInvocation(payload) {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":     "invariants_first",
		"invariant":      "ordinary_workspace_executor_constraints",
		"non_approvable": true,
		"reason":         "raw_shell_not_implicitly_ordinary",
		"executor_id":    payload.ExecutorID,
	}), true
}

func evaluateExecutorHardFloor(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload) (PolicyDecision, bool) {
	classes, floor := classifyHardFloorOperation(action, &payload)
	if len(classes) == 0 {
		return PolicyDecision{}, false
	}
	return hardFloorApprovalDecision(compiled, action, actionHash, classes, floor), true
}

func decodeActionPayload(payload map[string]any, target any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func isRawShellInvocation(payload executorRunPayload) bool {
	rawShellNames := map[string]struct{}{
		"sh": {}, "bash": {}, "zsh": {}, "fish": {}, "pwsh": {}, "powershell": {}, "cmd": {}, "cmd.exe": {},
	}
	if _, ok := rawShellNames[strings.ToLower(payload.ExecutorID)]; ok {
		return true
	}
	if len(payload.Argv) == 0 {
		return false
	}
	argv := unwrapLauncherArgv(payload.Argv)
	if len(argv) == 0 {
		return false
	}
	base := strings.ToLower(filepath.Base(argv[0]))
	_, ok := rawShellNames[base]
	return ok
}
