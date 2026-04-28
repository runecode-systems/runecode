package brokerapi

import (
	"fmt"
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

func runPlanAuthorityStepID(planID string) string {
	return runPlanAuthorityStepPrefix + strings.TrimSpace(planID)
}

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
		if strings.TrimSpace(plan.planID) != "" {
			return compiledRunGatePlan{}, fmt.Errorf("trusted run plan %q missing validated project context identity digest", plan.planID)
		}
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
	if err := ensureRunExistsForGatePlan(s.RunStatuses(), runID, false); err != nil {
		return compiledRunGatePlan{}, err
	}
	storedAuthority, ok, err := s.ActiveRunPlanAuthority(runID)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	if !ok {
		return compiledRunGatePlan{projectContextID: projectContextID, projectContextPinned: false}, nil
	}
	selected := runPlanAuthorityRecordFromStored(storedAuthority)
	projectContextID, projectContextPinned, err := validateProjectContextBinding(projectContextID, selected, projectContextID)
	if err != nil {
		return compiledRunGatePlan{}, err
	}
	entries := runPlannedEntriesFromStored(storedAuthority.Entries)
	if len(entries) == 0 {
		return compiledRunGatePlan{projectContextID: projectContextID, projectContextPinned: projectContextPinned, planID: selected.planID, runPlanRef: selected.runPlanRef, workflowDefinitionHash: selected.workflowDefinitionRef, processDefinitionHash: selected.processDefinitionRef, policyContextHash: selected.policyContextHash}, nil
	}
	return buildCompiledGatePlan(entries, selected, projectContextID, projectContextPinned)
}

func validateProjectContextBinding(projectContextID string, selected runPlanAuthorityRecord, originalContextID string) (string, bool, error) {
	projectContextPinned := strings.TrimSpace(selected.projectContextID) != ""
	if !projectContextPinned {
		return "", false, fmt.Errorf("trusted run plan %q missing validated project context identity digest", selected.planID)
	}
	if projectContextID == "" {
		return "", false, fmt.Errorf("trusted run plan %q requires validated project context identity digest", selected.planID)
	}
	if projectContextID != strings.TrimSpace(selected.projectContextID) {
		return "", false, fmt.Errorf("trusted run plan %q project context digest drift: planned %q current %q", selected.planID, strings.TrimSpace(selected.projectContextID), projectContextID)
	}
	return strings.TrimSpace(selected.projectContextID), projectContextPinned, nil
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

func runPlanAuthorityRecordFromStored(rec artifacts.RunPlanAuthorityRecord) runPlanAuthorityRecord {
	return runPlanAuthorityRecord{
		planID:                strings.TrimSpace(rec.PlanID),
		supersedesPlanID:      strings.TrimSpace(rec.SupersedesPlanID),
		runID:                 strings.TrimSpace(rec.RunID),
		workflowDefinitionRef: strings.TrimSpace(rec.WorkflowDefinitionHash),
		processDefinitionRef:  strings.TrimSpace(rec.ProcessDefinitionHash),
		policyContextHash:     strings.TrimSpace(rec.PolicyContextHash),
		projectContextID:      strings.TrimSpace(rec.ProjectContextIdentityDigest),
		runPlanRef:            strings.TrimSpace(rec.RunPlanDigest),
		createdAtUnixNano:     rec.RecordedAt.UTC().UnixNano(),
	}
}

func runPlannedEntriesFromStored(records []artifacts.RunPlanGateEntryRecord) []runPlannedGateEntry {
	if len(records) == 0 {
		return nil
	}
	out := make([]runPlannedGateEntry, 0, len(records))
	for _, record := range records {
		entry := runPlannedGateEntry{
			GateID:               strings.TrimSpace(record.GateID),
			GateKind:             strings.TrimSpace(record.GateKind),
			GateVersion:          strings.TrimSpace(record.GateVersion),
			StageID:              strings.TrimSpace(record.StageID),
			StepID:               strings.TrimSpace(record.StepID),
			RoleInstanceID:       strings.TrimSpace(record.RoleInstanceID),
			PlanCheckpointCode:   strings.TrimSpace(record.PlanCheckpointCode),
			PlanOrderIndex:       record.PlanOrderIndex,
			MaxAttempts:          record.MaxAttempts,
			ExpectedInputDigests: append([]string{}, record.ExpectedInputDigests...),
		}
		if len(record.DependencyCacheHandoffs) > 0 {
			entry.DependencyCacheHandoffs = make([]runPlannedDependencyCacheHandoff, 0, len(record.DependencyCacheHandoffs))
			for _, handoff := range record.DependencyCacheHandoffs {
				entry.DependencyCacheHandoffs = append(entry.DependencyCacheHandoffs, runPlannedDependencyCacheHandoff{
					RequestDigest: strings.TrimSpace(handoff.RequestDigest),
					ConsumerRole:  strings.TrimSpace(handoff.ConsumerRole),
					Required:      handoff.Required,
				})
			}
		}
		out = append(out, entry)
	}
	return out
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
