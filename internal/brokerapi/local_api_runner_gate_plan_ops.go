package brokerapi

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type runPlannedGateEntry struct {
	GateID                  string
	GateKind                string
	GateVersion             string
	StageID                 string
	StepID                  string
	RoleInstanceID          string
	PlanID                  string
	RunPlanRef              string
	PlanCheckpointCode      string
	PlanOrderIndex          int
	ProjectContextID        string
	WorkflowDefinitionHash  string
	ProcessDefinitionHash   string
	PolicyContextHash       string
	MaxAttempts             int
	ExpectedInputDigests    []string
	DependencyCacheHandoffs []runPlannedDependencyCacheHandoff
}

type runPlannedDependencyCacheHandoff struct {
	RequestDigest string
	ConsumerRole  string
	Required      bool
}

type compiledRunGatePlan struct {
	entries                []runPlannedGateEntry
	byGate                 map[string]runPlannedGateEntry
	projectContextID       string
	projectContextPinned   bool
	planID                 string
	runPlanRef             string
	workflowDefinitionHash string
	processDefinitionHash  string
	policyContextHash      string
}

type runPlanAuthorityRecord struct {
	planID                string
	supersedesPlanID      string
	runID                 string
	workflowDefinitionRef string
	processDefinitionRef  string
	policyContextHash     string
	projectContextID      string
	runPlanRef            string
	createdAtUnixNano     int64
	definition            map[string]any
}

const runPlanAuthorityStepPrefix = "compiled_run_plan/"

func (s *Service) compileRunGatePlan(runID string) (compiledRunGatePlan, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return compiledRunGatePlan{}, nil
	}
	if cached, ok := s.runGatePlanCache.get(runID); ok {
		resolved, err := s.resolveCachedRunGatePlan(cached)
		if err != nil {
			return compiledRunGatePlan{}, err
		}
		return resolved, nil
	}
	compiled, err := s.compileRunGatePlanUncached(runID)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	s.runGatePlanCache.setPlan(runID, compiled)
	return compiled, nil
}

func (s *Service) resolveCachedRunGatePlan(plan compiledRunGatePlan) (compiledRunGatePlan, error) {
	if !plan.projectContextPinned {
		currentProjectContextID := strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
		return plan.withProjectContextID(currentProjectContextID), nil
	}
	currentProjectContextID := strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
	if currentProjectContextID == "" {
		return compiledRunGatePlan{}, fmt.Errorf("trusted run plan %q requires validated project context identity digest", plan.planID)
	}
	if currentProjectContextID != strings.TrimSpace(plan.projectContextID) {
		return compiledRunGatePlan{}, fmt.Errorf("trusted run plan %q project context digest drift: planned %q current %q", plan.planID, strings.TrimSpace(plan.projectContextID), currentProjectContextID)
	}
	return plan, nil
}

func (s *Service) compileRunGatePlanUncached(runID string) (compiledRunGatePlan, error) {
	projectContextID := strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
	records, foundRun := runScopedPlanRecords(s.List(), runID)
	if err := ensureRunExistsForGatePlan(s.RunStatuses(), runID, foundRun); err != nil {
		return compiledRunGatePlan{}, err
	}
	authorities, err := s.collectRunPlanAuthorities(runID, records)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	selected, ok, err := selectActiveRunPlanAuthority(authorities)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	if !ok {
		return compiledRunGatePlan{projectContextID: projectContextID, projectContextPinned: false}, nil
	}
	projectContextID, projectContextPinned, err := validateProjectContextBinding(projectContextID, selected, projectContextID)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	entries, err := extractGateDefinitionsForRunPlan(selected.definition)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	if len(entries) == 0 {
		return compiledRunGatePlan{projectContextID: projectContextID, projectContextPinned: projectContextPinned, planID: selected.planID, runPlanRef: selected.runPlanRef, workflowDefinitionHash: selected.workflowDefinitionRef, processDefinitionHash: selected.processDefinitionRef, policyContextHash: selected.policyContextHash}, nil
	}
	return buildCompiledGatePlan(entries, selected, projectContextID, projectContextPinned)
}

func validateProjectContextBinding(projectContextID string, selected runPlanAuthorityRecord, originalContextID string) (string, bool, error) {
	projectContextPinned := strings.TrimSpace(selected.projectContextID) != ""
	if strings.TrimSpace(selected.projectContextID) != "" {
		if projectContextID == "" {
			return "", false, fmt.Errorf("trusted run plan %q requires validated project context identity digest", selected.planID)
		}
		if projectContextID != strings.TrimSpace(selected.projectContextID) {
			return "", false, fmt.Errorf("trusted run plan %q project context digest drift: planned %q current %q", selected.planID, strings.TrimSpace(selected.projectContextID), projectContextID)
		}
		return strings.TrimSpace(selected.projectContextID), projectContextPinned, nil
	}
	return originalContextID, projectContextPinned, nil
}

func buildCompiledGatePlan(entries []runPlannedGateEntry, selected runPlanAuthorityRecord, projectContextID string, projectContextPinned bool) (compiledRunGatePlan, error) {
	sortRunPlanEntries(entries)
	byGate, err := buildRunPlanEntryIndex(entries)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	for i := range entries {
		entries[i].PlanID = selected.planID
		entries[i].RunPlanRef = selected.runPlanRef
		entries[i].ProjectContextID = projectContextID
		entries[i].WorkflowDefinitionHash = selected.workflowDefinitionRef
		entries[i].ProcessDefinitionHash = selected.processDefinitionRef
		entries[i].PolicyContextHash = selected.policyContextHash
		key := runPlanGateKey(entries[i].GateID, entries[i].GateKind, entries[i].GateVersion, entries[i].PlanCheckpointCode, entries[i].PlanOrderIndex)
		entry := byGate[key]
		entry.PlanID = selected.planID
		entry.RunPlanRef = selected.runPlanRef
		entry.ProjectContextID = projectContextID
		entry.WorkflowDefinitionHash = selected.workflowDefinitionRef
		entry.ProcessDefinitionHash = selected.processDefinitionRef
		entry.PolicyContextHash = selected.policyContextHash
		byGate[key] = entry
	}
	return compiledRunGatePlan{entries: entries, byGate: byGate, projectContextID: projectContextID, projectContextPinned: projectContextPinned, planID: selected.planID, runPlanRef: selected.runPlanRef, workflowDefinitionHash: selected.workflowDefinitionRef, processDefinitionHash: selected.processDefinitionRef, policyContextHash: selected.policyContextHash}, nil
}

func (p compiledRunGatePlan) withProjectContextID(projectContextID string) compiledRunGatePlan {
	if p.projectContextPinned {
		return p
	}
	trimmed := strings.TrimSpace(projectContextID)
	if strings.TrimSpace(p.projectContextID) == trimmed {
		return p
	}
	out := p
	out.projectContextID = trimmed
	if len(p.entries) > 0 {
		out.entries = append([]runPlannedGateEntry(nil), p.entries...)
		for i := range out.entries {
			out.entries[i].ProjectContextID = trimmed
		}
	}
	if len(p.byGate) > 0 {
		out.byGate = make(map[string]runPlannedGateEntry, len(p.byGate))
		for key, entry := range p.byGate {
			entry.ProjectContextID = trimmed
			out.byGate[key] = entry
		}
	}
	return out
}

func (s *Service) collectRunPlanAuthorities(runID string, records []artifacts.ArtifactRecord) ([]runPlanAuthorityRecord, error) {
	authorities := make([]runPlanAuthorityRecord, 0, len(records))
	for _, record := range records {
		authority, ok, err := s.runPlanAuthorityFromRecord(runID, record)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		authorities = append(authorities, authority)
	}
	return authorities, nil
}

func (s *Service) runPlanAuthorityFromRecord(runID string, record artifacts.ArtifactRecord) (runPlanAuthorityRecord, bool, error) {
	obj, ok, err := s.readExecutableRunPlan(record.Reference.Digest)
	if err != nil {
		return runPlanAuthorityRecord{}, false, err
	}
	if !ok {
		return runPlanAuthorityRecord{}, false, nil
	}
	authority, err := runPlanAuthorityFromDefinition(runID, record, obj)
	if err != nil {
		return runPlanAuthorityRecord{}, false, err
	}
	return authority, true, nil
}

func (s *Service) readExecutableRunPlan(digest string) (map[string]any, bool, error) {
	r, err := s.Get(digest)
	if err != nil {
		return nil, false, fmt.Errorf("read trusted run plan artifact %q: %w", digest, err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, false, fmt.Errorf("read trusted run plan artifact body %q: %w", digest, err)
	}
	obj := map[string]any{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, false, nil
	}
	schemaID, _ := obj["schema_id"].(string)
	if strings.TrimSpace(schemaID) == "runecode.protocol.v0.RunPlan" {
		return obj, true, nil
	}
	return nil, false, nil
}
