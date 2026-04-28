package runplan

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func validateCompileInputPolicyContextHash(policyContextHash string) error {
	trimmed := strings.TrimSpace(policyContextHash)
	if trimmed == "" {
		return fmt.Errorf("policy_context_hash is required")
	}
	if _, err := policyengine.NormalizeHashIdentity(trimmed); err != nil {
		return fmt.Errorf("policy_context_hash invalid: %w", err)
	}
	return nil
}

func validateCompileInputProjectContextIdentityDigest(projectContextIdentityDigest string) error {
	trimmed := strings.TrimSpace(projectContextIdentityDigest)
	if trimmed == "" {
		return nil
	}
	if _, err := policyengine.NormalizeHashIdentity(trimmed); err != nil {
		return fmt.Errorf("project_context_identity_digest invalid: %w", err)
	}
	return nil
}

func validateCompileInputSupersedesPlanID(supersedesPlanID string, planID string) error {
	trimmedSupersedes := strings.TrimSpace(supersedesPlanID)
	if trimmedSupersedes != "" && trimmedSupersedes == strings.TrimSpace(planID) {
		return fmt.Errorf("supersedes_plan_id must differ from plan_id")
	}
	return nil
}
