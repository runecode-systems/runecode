package artifacts

import (
	"fmt"
	"strings"
	"time"
)

func computeRunPlanCompilationDigests(rec RunPlanCompilationRecord) (string, string, error) {
	bindingPayload := map[string]any{
		"run_id":                          rec.RunID,
		"plan_id":                         rec.PlanID,
		"run_plan_digest":                 rec.RunPlanDigest,
		"supersedes_plan_id":              rec.SupersedesPlanID,
		"workflow_definition_ref":         rec.WorkflowDefinitionRef,
		"process_definition_ref":          rec.ProcessDefinitionRef,
		"workflow_definition_hash":        rec.WorkflowDefinitionHash,
		"process_definition_hash":         rec.ProcessDefinitionHash,
		"policy_context_hash":             rec.PolicyContextHash,
		"project_context_identity_digest": rec.ProjectContextIdentityDigest,
	}
	if strings.TrimSpace(rec.CompileCacheKey) != "" {
		bindingPayload["compile_cache_key"] = strings.TrimSpace(rec.CompileCacheKey)
	}
	bindingCanonical, err := canonicalizeJSONValue(bindingPayload)
	if err != nil {
		return "", "", fmt.Errorf("canonicalize run plan compilation binding: %w", err)
	}
	bindingDigest := digestBytes(bindingCanonical)
	recordPayload := map[string]any{
		"binding_digest": bindingDigest,
		"compiled_at":    rec.CompiledAt.UTC().Format(time.RFC3339Nano),
	}
	recordCanonical, err := canonicalizeJSONValue(recordPayload)
	if err != nil {
		return "", "", fmt.Errorf("canonicalize run plan compilation record: %w", err)
	}
	return bindingDigest, digestBytes(recordCanonical), nil
}
