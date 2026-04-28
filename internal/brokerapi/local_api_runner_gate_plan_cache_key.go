package brokerapi

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/runplan"
)

func compileCacheKeyForPlan(plan runplan.RunPlan, workflowRef string, processRef string) string {
	payload := map[string]any{
		"workflow_definition_ref":         strings.TrimSpace(workflowRef),
		"process_definition_ref":          strings.TrimSpace(processRef),
		"workflow_definition_hash":        strings.TrimSpace(plan.WorkflowDefinitionHash),
		"process_definition_hash":         strings.TrimSpace(plan.ProcessDefinitionHash),
		"policy_context_hash":             strings.TrimSpace(plan.PolicyContextHash),
		"project_context_identity_digest": strings.TrimSpace(plan.ProjectContextIdentityDigest),
		"approval_profile":                strings.TrimSpace(plan.ApprovalProfile),
		"autonomy_posture":                strings.TrimSpace(plan.AutonomyPosture),
		"expected_input_digests":          expectedInputDigestsForPlan(plan),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(encoded)
	if err != nil {
		return ""
	}
	return artifacts.DigestBytes(canonical)
}

func expectedInputDigestsForPlan(plan runplan.RunPlan) []string {
	inputDigests := make([]string, 0)
	seen := map[string]struct{}{}
	for _, entry := range plan.Entries {
		for _, input := range entry.Gate.NormalizedInputs {
			digest, _ := input["input_digest"].(string)
			digest = strings.TrimSpace(digest)
			if digest == "" {
				continue
			}
			if _, ok := seen[digest]; ok {
				continue
			}
			seen[digest] = struct{}{}
			inputDigests = append(inputDigests, digest)
		}
	}
	sort.Strings(inputDigests)
	return inputDigests
}
