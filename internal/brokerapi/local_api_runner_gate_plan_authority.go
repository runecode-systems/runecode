package brokerapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func runPlanAuthorityStepID(planID string) string {
	return runPlanAuthorityStepPrefix + strings.TrimSpace(planID)
}

func runPlanAuthorityFromDefinition(runID string, record artifacts.ArtifactRecord, definition map[string]any) (runPlanAuthorityRecord, error) {
	planID, err := extractPlanIDAndVerifyStepID(definition, record)
	if err != nil {
		return runPlanAuthorityRecord{}, err
	}
	authorityRunID, err := extractAndValidateRunID(definition, record, runID)
	if err != nil {
		return runPlanAuthorityRecord{}, err
	}
	workflowDefinitionHash := extractHashField(definition, "workflow_definition_hash")
	if !isValidDigestIdentity(workflowDefinitionHash) {
		return runPlanAuthorityRecord{}, fmt.Errorf("trusted run plan artifact %q has invalid workflow_definition_hash", record.Reference.Digest)
	}
	processDefinitionHash := extractHashField(definition, "process_definition_hash")
	if !isValidDigestIdentity(processDefinitionHash) {
		return runPlanAuthorityRecord{}, fmt.Errorf("trusted run plan artifact %q has invalid process_definition_hash", record.Reference.Digest)
	}
	policyContextHash := extractHashField(definition, "policy_context_hash")
	if !isValidDigestIdentity(policyContextHash) {
		return runPlanAuthorityRecord{}, fmt.Errorf("trusted run plan artifact %q has invalid policy_context_hash", record.Reference.Digest)
	}
	supersedesPlanID := extractAndValidateSupersedesPlanID(definition, planID)
	projectContextID := extractProjectContextID(definition)
	if err := validateProjectContextID(projectContextID, record); err != nil {
		return runPlanAuthorityRecord{}, err
	}
	return runPlanAuthorityRecord{
		planID:                planID,
		supersedesPlanID:      supersedesPlanID,
		runID:                 authorityRunID,
		workflowDefinitionRef: workflowDefinitionHash,
		processDefinitionRef:  processDefinitionHash,
		policyContextHash:     policyContextHash,
		projectContextID:      projectContextID,
		runPlanRef:            strings.TrimSpace(record.Reference.Digest),
		createdAtUnixNano:     record.CreatedAt.UTC().UnixNano(),
		definition:            definition,
	}, nil
}

func extractPlanIDAndVerifyStepID(definition map[string]any, record artifacts.ArtifactRecord) (string, error) {
	planID, _ := definition["plan_id"].(string)
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return "", fmt.Errorf("trusted run plan artifact %q missing plan_id", record.Reference.Digest)
	}
	if expectedStepID := runPlanAuthorityStepID(planID); strings.TrimSpace(record.StepID) != expectedStepID {
		return "", fmt.Errorf("trusted run plan artifact %q must use step_id %q", record.Reference.Digest, expectedStepID)
	}
	return planID, nil
}

func extractAndValidateRunID(definition map[string]any, record artifacts.ArtifactRecord, runID string) (string, error) {
	authorityRunID, _ := definition["run_id"].(string)
	authorityRunID = strings.TrimSpace(authorityRunID)
	if authorityRunID == "" {
		return "", fmt.Errorf("trusted run plan artifact %q missing run_id", record.Reference.Digest)
	}
	if authorityRunID != strings.TrimSpace(runID) {
		return "", fmt.Errorf("trusted run plan artifact %q run_id %q does not match %q", record.Reference.Digest, authorityRunID, runID)
	}
	return authorityRunID, nil
}

func extractHashField(definition map[string]any, field string) string {
	hash, _ := definition[field].(string)
	return strings.TrimSpace(hash)
}

func extractAndValidateSupersedesPlanID(definition map[string]any, planID string) string {
	supersedesPlanID, _ := definition["supersedes_plan_id"].(string)
	supersedesPlanID = strings.TrimSpace(supersedesPlanID)
	if supersedesPlanID == planID {
		return ""
	}
	return supersedesPlanID
}

func extractProjectContextID(definition map[string]any) string {
	projectContextID, _ := definition["project_context_identity_digest"].(string)
	return strings.TrimSpace(projectContextID)
}

func validateProjectContextID(projectContextID string, record artifacts.ArtifactRecord) error {
	if projectContextID != "" && !isValidDigestIdentity(projectContextID) {
		return fmt.Errorf("trusted run plan artifact %q has invalid project_context_identity_digest", record.Reference.Digest)
	}
	return nil
}

func selectActiveRunPlanAuthority(authorities []runPlanAuthorityRecord) (runPlanAuthorityRecord, bool, error) {
	if len(authorities) == 0 {
		return runPlanAuthorityRecord{}, false, nil
	}
	byPlanID, err := collectAuthoritiesByPlanID(authorities)
	if err != nil {
		return runPlanAuthorityRecord{}, false, err
	}
	active, err := resolveActiveAuthorities(byPlanID)
	if err != nil {
		return runPlanAuthorityRecord{}, false, err
	}
	if len(active) != 1 {
		rec, err := formatAuthorityError(active)
		return rec, false, err
	}
	return active[0], true, nil
}

func collectAuthoritiesByPlanID(authorities []runPlanAuthorityRecord) (map[string]runPlanAuthorityRecord, error) {
	byPlanID := map[string]runPlanAuthorityRecord{}
	for _, authority := range authorities {
		if existing, ok := byPlanID[authority.planID]; ok {
			if existing.runPlanRef != authority.runPlanRef {
				return nil, fmt.Errorf("run has conflicting trusted run plan artifacts for plan_id %q", authority.planID)
			}
			if authority.createdAtUnixNano > existing.createdAtUnixNano {
				byPlanID[authority.planID] = authority
			}
			continue
		}
		byPlanID[authority.planID] = authority
	}
	return byPlanID, nil
}

func resolveActiveAuthorities(byPlanID map[string]runPlanAuthorityRecord) ([]runPlanAuthorityRecord, error) {
	superseded := collectSupersededPlanIDs(byPlanID)
	active := filterActiveAuthorities(byPlanID, superseded)
	return active, nil
}

func collectSupersededPlanIDs(byPlanID map[string]runPlanAuthorityRecord) map[string]struct{} {
	superseded := map[string]struct{}{}
	for _, authority := range byPlanID {
		if authority.supersedesPlanID != "" {
			superseded[authority.supersedesPlanID] = struct{}{}
		}
	}
	return superseded
}

func filterActiveAuthorities(byPlanID map[string]runPlanAuthorityRecord, superseded map[string]struct{}) []runPlanAuthorityRecord {
	active := make([]runPlanAuthorityRecord, 0, len(byPlanID))
	for planID, authority := range byPlanID {
		if _, ok := superseded[planID]; !ok {
			active = append(active, authority)
		}
	}
	return active
}

func formatAuthorityError(active []runPlanAuthorityRecord) (runPlanAuthorityRecord, error) {
	planIDs := make([]string, 0, len(active))
	for _, authority := range active {
		planIDs = append(planIDs, authority.planID)
	}
	sort.Strings(planIDs)
	if len(active) == 0 {
		return runPlanAuthorityRecord{}, fmt.Errorf("run has no active trusted run plan authority")
	}
	return runPlanAuthorityRecord{}, fmt.Errorf("run has ambiguous active trusted run plan authority: %s", strings.Join(planIDs, ","))
}
