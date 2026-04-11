package policyengine

import "fmt"

func validateWorkspaceRoleCapabilityManifest(roleKind string, capabilityOptIns []string, manifestLabel string) error {
	allowed, known := workspaceRoleAllowedCapabilities()[roleKind]
	if !known {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s role_kind %q missing explicit workspace capability manifest policy (fail-closed)", manifestLabel, roleKind)}
	}
	for _, cap := range capabilityOptIns {
		if _, ok := allowed[cap]; !ok {
			return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s capability %q is not allowed for workspace role_kind %q", manifestLabel, cap, roleKind)}
		}
	}
	return nil
}

func knownWorkspaceRoleKinds() map[string]struct{} {
	return map[string]struct{}{
		"workspace-read": {},
		"workspace-edit": {},
		"workspace-test": {},
	}
}

func workspaceRoleAllowedCapabilities() map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		"workspace-read": {
			"cap_artifact_read": {},
		},
		"workspace-edit": {
			"cap_stage":         {},
			"cap_run":           {},
			"cap_exec":          {},
			"cap_artifact_read": {},
			"cap_backend":       {},
			"promotion":         {},
			"cap_other":         {},
			"always_denied":     {},
		},
		"workspace-test": {
			"cap_stage":         {},
			"cap_run":           {},
			"cap_exec":          {},
			"cap_artifact_read": {},
			"cap_backend":       {},
			"promotion":         {},
			"cap_other":         {},
			"always_denied":     {},
		},
	}
}
