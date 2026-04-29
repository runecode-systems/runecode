package brokerapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/runplan"
)

type compileIdentity struct {
	RunID                         string   `json:"run_id"`
	PlanID                        string   `json:"plan_id"`
	SupersedesPlanID              string   `json:"supersedes_plan_id,omitempty"`
	WorkflowDefinitionRef         string   `json:"workflow_definition_ref"`
	ProcessDefinitionRef          string   `json:"process_definition_ref"`
	WorkflowDefinitionHash        string   `json:"workflow_definition_hash"`
	ProcessDefinitionHash         string   `json:"process_definition_hash"`
	PolicyContextHash             string   `json:"policy_context_hash"`
	ProjectContextIdentityDigest  string   `json:"project_context_identity_digest"`
	ApprovalProfile               string   `json:"approval_profile"`
	AutonomyPosture               string   `json:"autonomy_posture"`
	ExpectedInputDigests          []string `json:"expected_input_digests,omitempty"`
	DependencyCacheRequestDigests []string `json:"dependency_cache_request_digests,omitempty"`
	ApprovedInputSetDigest        string   `json:"approved_input_set_digest,omitempty"`
}

func compileIdentityFromInput(workflowRef, processRef, approvedInputSetDigest string, input runplan.CompileInput) (compileIdentity, string, error) {
	workflow, workflowHash, err := decodeWorkflowCompileIdentity(input.WorkflowDefinitionBytes)
	if err != nil {
		return compileIdentity{}, "", err
	}
	process, processHash, err := decodeProcessCompileIdentity(input.ProcessDefinitionBytes)
	if err != nil {
		return compileIdentity{}, "", err
	}
	identity := compileIdentity{
		RunID:                         strings.TrimSpace(input.RunID),
		PlanID:                        strings.TrimSpace(input.PlanID),
		SupersedesPlanID:              strings.TrimSpace(input.SupersedesPlanID),
		WorkflowDefinitionRef:         strings.TrimSpace(workflowRef),
		ProcessDefinitionRef:          strings.TrimSpace(processRef),
		WorkflowDefinitionHash:        strings.TrimSpace(workflowHash),
		ProcessDefinitionHash:         strings.TrimSpace(processHash),
		PolicyContextHash:             strings.TrimSpace(input.PolicyContextHash),
		ProjectContextIdentityDigest:  strings.TrimSpace(input.ProjectContextIdentityDigest),
		ApprovalProfile:               strings.TrimSpace(workflow.ApprovalProfile),
		AutonomyPosture:               strings.TrimSpace(workflow.AutonomyPosture),
		ExpectedInputDigests:          expectedInputDigestsForGates(process.GateDefinitions),
		DependencyCacheRequestDigests: dependencyCacheRequestDigestsForGates(process.GateDefinitions),
		ApprovedInputSetDigest:        strings.TrimSpace(approvedInputSetDigest),
	}
	cacheKey, err := compileIdentityCacheKey(identity)
	if err != nil {
		return compileIdentity{}, "", err
	}
	return identity, cacheKey, nil
}

func decodeWorkflowCompileIdentity(payload []byte) (runplan.WorkflowDefinition, string, error) {
	canonical, err := artifacts.CanonicalizeJSONBytes(payload)
	if err != nil {
		return runplan.WorkflowDefinition{}, "", fmt.Errorf("canonicalize workflow definition: %w", err)
	}
	workflow := runplan.WorkflowDefinition{}
	if err := json.Unmarshal(canonical, &workflow); err != nil {
		return runplan.WorkflowDefinition{}, "", fmt.Errorf("decode workflow definition: %w", err)
	}
	return workflow, artifacts.DigestBytes(canonical), nil
}

func decodeProcessCompileIdentity(payload []byte) (runplan.ProcessDefinition, string, error) {
	canonical, err := artifacts.CanonicalizeJSONBytes(payload)
	if err != nil {
		return runplan.ProcessDefinition{}, "", fmt.Errorf("canonicalize process definition: %w", err)
	}
	process := runplan.ProcessDefinition{}
	if err := json.Unmarshal(canonical, &process); err != nil {
		return runplan.ProcessDefinition{}, "", fmt.Errorf("decode process definition: %w", err)
	}
	return process, artifacts.DigestBytes(canonical), nil
}

func compileIdentityCacheKey(identity compileIdentity) (string, error) {
	encoded, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(encoded)
	if err != nil {
		return "", err
	}
	return artifacts.DigestBytes(canonical), nil
}

func expectedInputDigestsForGates(gates []runplan.GateDefinition) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, gate := range gates {
		for _, input := range gate.Gate.NormalizedInputs {
			digest, _ := input["input_digest"].(string)
			digest = strings.TrimSpace(digest)
			if digest == "" {
				continue
			}
			if _, ok := seen[digest]; ok {
				continue
			}
			seen[digest] = struct{}{}
			out = append(out, digest)
		}
	}
	sort.Strings(out)
	return out
}

func dependencyCacheRequestDigestsForGates(gates []runplan.GateDefinition) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, gate := range gates {
		for _, handoff := range gate.DependencyCacheHandoffs {
			digest, err := handoff.RequestDigest.Identity()
			if err != nil {
				continue
			}
			out = appendUniqueDigest(out, seen, digest)
		}
	}
	sort.Strings(out)
	return out
}

func appendUniqueDigest(out []string, seen map[string]struct{}, digest string) []string {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return out
	}
	if _, ok := seen[digest]; ok {
		return out
	}
	seen[digest] = struct{}{}
	return append(out, digest)
}
