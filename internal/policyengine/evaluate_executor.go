package policyengine

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
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
		denyWorkspaceExecutorEnvironmentMismatch,
	}
	for _, validate := range validators {
		if decision, blocked := validate(compiled, action, actionHash, payload, contract); blocked {
			return decision, true
		}
	}
	return PolicyDecision{}, false
}

type workspaceExecutorContractValidator func(*CompiledContext, ActionRequest, string, executorRunPayload, typedExecutorContract) (PolicyDecision, bool)

func resolveWorkspaceExecutorContract(compiled *CompiledContext, action ActionRequest, actionHash string, executorID string) (typedExecutorContract, PolicyDecision, bool) {
	contract, known := workspaceExecutorContractsByID()[executorID]
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
	if !contract.RequireWorkspace || payload.WorkingDir == "" || isWorkspaceRelativePath(payload.WorkingDir) {
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

func workspaceExecutorContractsByID() map[string]typedExecutorContract {
	return map[string]typedExecutorContract{
		"workspace-runner": {
			ID:               "workspace-runner",
			AllowedRoles:     roleSet("workspace-edit", "workspace-test"),
			AllowedClass:     "workspace_ordinary",
			AllowedNetwork:   roleSet("none"),
			AllowedEnvKeys:   roleSet("CI", "HOME", "LANG", "LC_ALL", "PATH", "PWD", "TMP", "TMPDIR", "TEMP"),
			AllowEmptyEnv:    true,
			RequireWorkspace: true,
			AllowEnvWrapper:  true,
			AllowedArgvHeads: [][]string{{"go", "test"}, {"go", "build"}, {"go", "vet"}, {"go", "fmt"}, {"go", "list"}, {"python"}, {"node", "--test"}, {"npm", "test"}, {"just", "test"}, {"just", "lint"}, {"just", "fmt"}, {"just", "ci"}},
			MaxArgvItems:     64,
			MaxTimeoutSecs:   3600,
		},
		"python": {
			ID:               "python",
			AllowedRoles:     roleSet("workspace-edit", "workspace-test"),
			AllowedClass:     "workspace_ordinary",
			AllowedNetwork:   roleSet("none"),
			AllowedEnvKeys:   roleSet("PYTHONPATH", "PYTHONWARNINGS", "CI", "HOME", "LANG", "LC_ALL", "PATH", "PWD", "TMP", "TMPDIR", "TEMP"),
			AllowEmptyEnv:    true,
			RequireWorkspace: true,
			AllowEnvWrapper:  false,
			AllowedArgvHeads: [][]string{{"python"}},
			MaxArgvItems:     64,
			MaxTimeoutSecs:   3600,
		},
	}
}

func validateExecutorArgvShape(argv []string, contract typedExecutorContract) error {
	if len(argv) == 0 {
		return fmt.Errorf("argv must not be empty")
	}
	for _, token := range argv {
		if strings.EqualFold(strings.TrimSpace(token), "sudo") {
			return fmt.Errorf("executor contract forbids privilege-escalation launcher")
		}
	}
	base := argv
	if contract.AllowEnvWrapper {
		base = unwrapLauncherArgv(argv)
		if len(base) == 0 {
			return fmt.Errorf("argv wrapper chain does not resolve to concrete executable")
		}
	}
	invoked := strings.ToLower(filepath.Base(base[0]))
	expected := strings.ToLower(filepath.Base(contract.ID))
	if expected != "workspace-runner" && invoked != expected {
		return fmt.Errorf("argv executable %q does not match executor_id %q", invoked, contract.ID)
	}
	if err := validateExecutorArgvHead(base, contract); err != nil {
		return err
	}
	if isRawShellInvocation(executorRunPayload{ExecutorID: contract.ID, Argv: argv}) {
		return fmt.Errorf("executor contract forbids raw shell passthrough")
	}
	if hasCommandStringPassthrough(base) {
		return fmt.Errorf("executor contract forbids command-string passthrough")
	}
	return nil
}

func validateExecutorArgvHead(argv []string, contract typedExecutorContract) error {
	if len(contract.AllowedArgvHeads) == 0 {
		return nil
	}
	for _, head := range contract.AllowedArgvHeads {
		if len(argv) < len(head) {
			continue
		}
		matches := true
		for i := range head {
			if strings.ToLower(argv[i]) != strings.ToLower(head[i]) {
				matches = false
				break
			}
		}
		if matches {
			return nil
		}
	}
	return fmt.Errorf("argv does not match reviewed operation heads for executor_id %q", contract.ID)
}

func validateExecutorEnvironmentShape(environment map[string]string, contract typedExecutorContract) error {
	if len(environment) == 0 {
		if contract.AllowEmptyEnv {
			return nil
		}
		return fmt.Errorf("environment must not be empty")
	}
	for key, value := range environment {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("environment key must not be empty")
		}
		if strings.Contains(key, "=") {
			return fmt.Errorf("environment key %q must not include '='", key)
		}
		if strings.Contains(value, "\x00") {
			return fmt.Errorf("environment value for %q contains NUL byte", key)
		}
		if _, ok := contract.AllowedEnvKeys[key]; !ok {
			return fmt.Errorf("environment key %q is not allowed", key)
		}
	}
	return nil
}

func hasCommandStringPassthrough(argv []string) bool {
	if len(argv) < 2 {
		return false
	}
	exec := strings.ToLower(filepath.Base(argv[0]))
	tokenSet := map[string]struct{}{}
	for i := 1; i < len(argv); i++ {
		tokenSet[strings.ToLower(strings.TrimSpace(argv[i]))] = struct{}{}
	}
	if exec == "python" || exec == "python3" {
		_, short := tokenSet["-c"]
		return short
	}
	if exec == "node" {
		_, short := tokenSet["-e"]
		_, long := tokenSet["--eval"]
		return short || long
	}
	if exec == "powershell" || exec == "pwsh" {
		_, alias := tokenSet["-c"]
		_, short := tokenSet["-command"]
		_, long := tokenSet["--command"]
		return alias || short || long
	}
	if exec == "cmd" || exec == "cmd.exe" {
		_, short := tokenSet["/c"]
		return short
	}
	return false
}

func roleSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
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

func unwrapLauncherArgv(argv []string) []string {
	idx := 0
	for idx < len(argv) {
		tok := strings.ToLower(filepath.Base(argv[idx]))
		switch tok {
		case "env":
			idx++
			for idx < len(argv) && isEnvAssignmentToken(argv[idx]) {
				idx++
			}
			continue
		case "command", "nohup":
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
	for _, token := range argv {
		if tokenIsSystemModifying(token) {
			return true
		}
	}
	return false
}

func tokenIsSystemModifying(token string) bool {
	lower := strings.ToLower(strings.TrimSpace(token))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "/etc/") || strings.Contains(lower, "c:\\windows") {
		return true
	}
	base := strings.ToLower(filepath.Base(lower))
	if _, ok := systemModifyingExecutableNames[base]; ok {
		return true
	}
	for _, candidate := range splitCommandLikeToken(lower) {
		candidateBase := strings.ToLower(filepath.Base(candidate))
		if _, ok := systemModifyingExecutableNames[candidateBase]; ok {
			return true
		}
	}
	return false
}

func splitCommandLikeToken(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("'\"`;|&(){}[]<>,", r)
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, strings.Trim(trimmed, "`"))
	}
	return out
}

func destinationRefMatches(descriptor DestinationDescriptor, destinationRef string) bool {
	if strings.TrimSpace(destinationRef) == "" {
		return false
	}

	host, port, path := parseDestinationRef(destinationRef)
	if host == "" || strings.ToLower(host) != strings.ToLower(descriptor.CanonicalHost) {
		return false
	}

	if descriptor.CanonicalPort != nil {
		if port == nil || *port != *descriptor.CanonicalPort {
			return false
		}
	} else {
		expectedPort := 443
		if port != nil && *port != expectedPort {
			return false
		}
	}

	if descriptor.CanonicalPathPrefix != "" {
		normalizedPath := normalizeDestinationPath(path)
		normalizedPrefix := normalizeDestinationPath(descriptor.CanonicalPathPrefix)
		if !strings.HasPrefix(normalizedPath, normalizedPrefix) {
			return false
		}
	}

	return true
}

func normalizeDestinationPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	normalized := path.Clean(trimmed)
	if normalized == "." {
		return "/"
	}
	return normalized
}

func isEnvAssignmentToken(token string) bool {
	t := strings.TrimSpace(token)
	if t == "" {
		return false
	}
	eq := strings.IndexByte(t, '=')
	if eq <= 0 {
		return false
	}
	return isEnvAssignmentName(t[:eq])
}

func isEnvAssignmentName(name string) bool {
	for i, r := range name {
		if i == 0 {
			if !isEnvAssignmentStartRune(r) {
				return false
			}
			continue
		}
		if !isEnvAssignmentContinueRune(r) {
			return false
		}
	}
	return true
}

func isEnvAssignmentStartRune(r rune) bool {
	return r == '_' || isASCIIAlpha(r)
}

func isEnvAssignmentContinueRune(r rune) bool {
	return isEnvAssignmentStartRune(r) || isASCIIDigit(r)
}

func isASCIIAlpha(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
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
