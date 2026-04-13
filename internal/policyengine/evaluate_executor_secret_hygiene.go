package policyengine

import "strings"

func denyWorkspaceExecutorSecretMaterialLeakage(compiled *CompiledContext, action ActionRequest, actionHash string, payload executorRunPayload, _ typedExecutorContract) (PolicyDecision, bool) {
	if firstLikelySecretArgvToken(payload.Argv) != "" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":          "invariants_first",
			"invariant":           "workspace_executor_contract_matrix",
			"non_approvable":      true,
			"active_role_kind":    compiled.Context.ActiveRoleKind,
			"executor_id":         payload.ExecutorID,
			"reason":              "argv_contains_likely_secret_material",
			"argv_token_redacted": true,
		}), true
	}
	for _, value := range payload.Environment {
		if !looksLikeSecretEnvValue(value) {
			continue
		}
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "workspace_executor_contract_matrix",
			"non_approvable":   true,
			"active_role_kind": compiled.Context.ActiveRoleKind,
			"executor_id":      payload.ExecutorID,
			"reason":           "environment_contains_likely_secret_material",
			"env_key_redacted": true,
		}), true
	}
	return PolicyDecision{}, false
}

func firstLikelySecretArgvToken(argv []string) string {
	for i, token := range argv {
		if isLikelySecretArgvTokenAt(argv, i, token) {
			return strings.TrimSpace(token)
		}
	}
	return ""
}

func isLikelySecretArgvTokenAt(argv []string, index int, token string) bool {
	trimmed := strings.TrimSpace(token)
	if looksLikeSecretArgvToken(trimmed) {
		return true
	}
	normalized := strings.ReplaceAll(trimmed, " ", "")
	if looksLikeSecretArgvToken(normalized) {
		return true
	}
	if !strings.HasPrefix(trimmed, "-") || index+1 >= len(argv) {
		return false
	}
	next := strings.TrimSpace(argv[index+1])
	return looksLikeSecretValueCandidate(next)
}

func looksLikeSecretArgvToken(token string) bool {
	lower := strings.ToLower(token)
	for _, marker := range []string{"--token=", "--password=", "--secret=", "--api-key=", "--api_key=", "--access-key=", "--access_key=", "--secret-key=", "--secret_key="} {
		if strings.HasPrefix(lower, marker) {
			return true
		}
	}
	if strings.Contains(lower, "bearer ") {
		return true
	}
	return false
}

func looksLikeSecretValueCandidate(value string) bool {
	if looksLikeSecretEnvValue(value) {
		return true
	}
	if len(value) < 20 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			hasLetter = true
		case r >= 'A' && r <= 'Z':
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '-' || r == '_' || r == '.' || r == '=' || r == '+' || r == '/':
		default:
			return false
		}
	}
	return hasLetter && hasDigit
}

func looksLikeSecretEnvValue(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	for _, prefix := range []string{"ghp_", "gho_", "github_pat_", "sk-", "rk-", "akia", "asia", "ya29.", "eyj", "xoxb-", "xoxp-"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return strings.Contains(lower, "-----begin") || strings.Contains(lower, "bearer ")
}
