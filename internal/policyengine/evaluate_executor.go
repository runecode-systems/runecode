package policyengine

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
)

func evaluateExecutorBoundary(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	payload, decision, blocked := decodeExecutorPayload(compiled, action, actionHash)
	if blocked {
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
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":       "invariants_first",
		"invariant":        "network_egress_hard_boundary",
		"non_approvable":   true,
		"network_access":   payload.NetworkAccess,
		"required_network": "none",
		"action_kind":      ActionKindExecutorRun,
	}), true
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

func unwrapLauncherArgv(argv []string) []string {
	idx := 0
	for idx < len(argv) {
		tok := strings.ToLower(filepath.Base(argv[idx]))
		switch tok {
		case "env":
			idx++
			for idx < len(argv) && strings.Contains(argv[idx], "=") {
				idx++
			}
			continue
		case "command", "nohup", "sudo":
			idx++
			continue
		default:
			return argv[idx:]
		}
	}
	return argv[idx:]
}

func isWorkspaceRelativePath(raw string) bool {
	path := strings.TrimSpace(raw)
	if path == "" {
		return false
	}
	if isCrossPlatformAbsolutePath(path) {
		return false
	}

	clean := filepath.Clean(path)
	normalized := strings.ReplaceAll(clean, "\\", "/")
	return normalized != ".." && !strings.HasPrefix(normalized, "../")
}

func isCrossPlatformAbsolutePath(path string) bool {
	if filepath.IsAbs(path) {
		return true
	}
	if strings.HasPrefix(path, "\\\\") || strings.HasPrefix(path, "\\") {
		return true
	}
	if len(path) >= 2 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' {
		return true
	}
	return false
}

func isSystemModifyingArgv(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	first := strings.ToLower(filepath.Base(argv[0]))
	systemTools := map[string]struct{}{
		"apt": {}, "apt-get": {}, "yum": {}, "dnf": {}, "apk": {}, "pacman": {}, "brew": {},
		"systemctl": {}, "service": {}, "modprobe": {}, "sysctl": {}, "mount": {}, "umount": {},
		"iptables": {}, "ufw": {}, "nft": {}, "netsh": {}, "sc": {},
		"docker": {}, "podman": {}, "kubectl": {}, "helm": {},
	}
	if _, ok := systemTools[first]; ok {
		return true
	}
	for _, arg := range argv {
		lower := strings.ToLower(arg)
		if strings.Contains(lower, "/etc/") || strings.Contains(lower, "c:\\windows") {
			return true
		}
	}
	return false
}

func destinationRefMatches(descriptor DestinationDescriptor, destinationRef string) bool {
	if strings.TrimSpace(destinationRef) == "" {
		return false
	}

	host, port, path := parseDestinationRef(destinationRef)
	if host == "" || host != descriptor.CanonicalHost {
		return false
	}

	expectedPort := 443
	if descriptor.CanonicalPort != nil {
		expectedPort = *descriptor.CanonicalPort
	}
	if port != nil && *port != expectedPort {
		return false
	}

	if descriptor.CanonicalPathPrefix != "" {
		if !strings.HasPrefix(path, descriptor.CanonicalPathPrefix) {
			return false
		}
	}

	return true
}

func parseDestinationRef(ref string) (string, *int, string) {
	value := strings.TrimSpace(ref)
	if value == "" {
		return "", nil, ""
	}

	hostPort := value
	path := ""
	if slash := strings.Index(hostPort, "/"); slash >= 0 {
		path = hostPort[slash:]
		hostPort = hostPort[:slash]
	}

	host := hostPort
	var port *int
	if colon := strings.LastIndex(hostPort, ":"); colon > 0 && colon < len(hostPort)-1 {
		if parsed, err := strconv.Atoi(hostPort[colon+1:]); err == nil && parsed > 0 && parsed <= 65535 {
			h := hostPort[:colon]
			host = h
			port = &parsed
		}
	}

	if host == "" {
		return "", nil, ""
	}
	if path == "" {
		path = "/"
	}

	return host, port, path
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
