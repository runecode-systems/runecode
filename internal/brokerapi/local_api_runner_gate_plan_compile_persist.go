package brokerapi

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/runplan"
)

type CompileAndPersistRunPlanRequest struct {
	RunID                        string
	PlanID                       string
	SupersedesPlanID             string
	WorkflowDefinitionRef        string
	ProcessDefinitionRef         string
	PolicyContextHash            string
	ProjectContextIdentityDigest string
}

type CompileAndPersistRunPlanResult struct {
	RunID             string
	PlanID            string
	RunPlanDigest     string
	CompilationDigest string
}

func (s *Service) CompileAndPersistRunPlan(req CompileAndPersistRunPlanRequest) (CompileAndPersistRunPlanResult, error) {
	planned, workflowRef, processRef, err := s.compileRunPlanForPersistence(req)
	if err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	ref, err := s.persistCompiledRunPlanArtifact(planned)
	if err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	authority, compilation, err := runPlanRecordsFromCompiled(planned, ref.Digest, workflowRef, processRef)
	if err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	if err := s.RecordRunPlanAuthority(authority, compilation); err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	return CompileAndPersistRunPlanResult{
		RunID:             authority.RunID,
		PlanID:            authority.PlanID,
		RunPlanDigest:     authority.RunPlanDigest,
		CompilationDigest: compilation.RecordDigest,
	}, nil
}

func (s *Service) compileRunPlanForPersistence(req CompileAndPersistRunPlanRequest) (runplan.RunPlan, string, string, error) {
	input, err := s.compileRunPlanInputFromArtifacts(req)
	if err != nil {
		return runplan.RunPlan{}, "", "", err
	}
	if err := ensureRunExistsForGatePlan(s.RunStatuses(), input.RunID, false); err != nil {
		return runplan.RunPlan{}, "", "", err
	}
	planned, err := runplan.Compile(input)
	if err != nil {
		return runplan.RunPlan{}, "", "", err
	}
	return planned, strings.TrimSpace(req.WorkflowDefinitionRef), strings.TrimSpace(req.ProcessDefinitionRef), nil
}

func (s *Service) persistCompiledRunPlanArtifact(planned runplan.RunPlan) (artifacts.ArtifactReference, error) {
	payload, err := json.Marshal(planned)
	if err != nil {
		return artifacts.ArtifactReference{}, fmt.Errorf("marshal compiled run plan payload: %w", err)
	}
	return s.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: artifacts.DigestBytes(payload),
		CreatedByRole:         "brokerapi",
		TrustedSource:         true,
		RunID:                 planned.RunID,
		StepID:                runPlanAuthorityStepID(planned.PlanID),
	})
}

func (s *Service) compileRunPlanInputFromArtifacts(req CompileAndPersistRunPlanRequest) (runplan.CompileInput, error) {
	runID := strings.TrimSpace(req.RunID)
	planID := strings.TrimSpace(req.PlanID)
	if runID == "" || planID == "" {
		return runplan.CompileInput{}, fmt.Errorf("run_id and plan_id are required")
	}
	workflowRef := strings.TrimSpace(req.WorkflowDefinitionRef)
	processRef := strings.TrimSpace(req.ProcessDefinitionRef)
	if workflowRef == "" || processRef == "" {
		return runplan.CompileInput{}, fmt.Errorf("workflow_definition_ref and process_definition_ref are required")
	}
	workflowBytes, err := s.readArtifactPayload(workflowRef)
	if err != nil {
		return runplan.CompileInput{}, err
	}
	processBytes, err := s.readArtifactPayload(processRef)
	if err != nil {
		return runplan.CompileInput{}, err
	}
	projectContext := strings.TrimSpace(req.ProjectContextIdentityDigest)
	if projectContext == "" {
		projectContext = strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
	}
	if projectContext == "" {
		return runplan.CompileInput{}, fmt.Errorf("project_context_identity_digest is required for trusted run plan compilation")
	}
	return runplan.CompileInput{
		RunID:                        runID,
		PlanID:                       planID,
		SupersedesPlanID:             strings.TrimSpace(req.SupersedesPlanID),
		WorkflowDefinitionBytes:      workflowBytes,
		ProcessDefinitionBytes:       processBytes,
		ProjectContextIdentityDigest: projectContext,
		PolicyContextHash:            strings.TrimSpace(req.PolicyContextHash),
		ExecutorRegistry:             policyengine.BuildExecutorRegistryProjection(),
	}, nil
}

func (s *Service) readArtifactPayload(digest string) ([]byte, error) {
	r, err := s.Get(digest)
	if err != nil {
		return nil, fmt.Errorf("read artifact %q: %w", digest, err)
	}
	defer r.Close()
	payload, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read artifact payload %q: %w", digest, err)
	}
	return payload, nil
}

func runPlanRecordsFromCompiled(plan runplan.RunPlan, runPlanDigest string, workflowRef string, processRef string) (artifacts.RunPlanAuthorityRecord, artifacts.RunPlanCompilationRecord, error) {
	authority := artifacts.RunPlanAuthorityRecord{
		RunID:                        strings.TrimSpace(plan.RunID),
		PlanID:                       strings.TrimSpace(plan.PlanID),
		SupersedesPlanID:             strings.TrimSpace(plan.SupersedesPlanID),
		RunPlanDigest:                strings.TrimSpace(runPlanDigest),
		WorkflowDefinitionHash:       strings.TrimSpace(plan.WorkflowDefinitionHash),
		ProcessDefinitionHash:        strings.TrimSpace(plan.ProcessDefinitionHash),
		PolicyContextHash:            strings.TrimSpace(plan.PolicyContextHash),
		ProjectContextIdentityDigest: strings.TrimSpace(plan.ProjectContextIdentityDigest),
		CompiledAt:                   parseCompiledAt(plan.CompiledAt),
		RecordedAt:                   parseCompiledAt(plan.CompiledAt),
		Entries:                      runPlanEntryRecords(plan.Entries),
	}
	compilation := artifacts.RunPlanCompilationRecord{
		RunID:                        authority.RunID,
		PlanID:                       authority.PlanID,
		RunPlanDigest:                authority.RunPlanDigest,
		SupersedesPlanID:             authority.SupersedesPlanID,
		WorkflowDefinitionRef:        workflowRef,
		ProcessDefinitionRef:         processRef,
		WorkflowDefinitionHash:       authority.WorkflowDefinitionHash,
		ProcessDefinitionHash:        authority.ProcessDefinitionHash,
		PolicyContextHash:            authority.PolicyContextHash,
		ProjectContextIdentityDigest: authority.ProjectContextIdentityDigest,
		CompiledAt:                   authority.CompiledAt,
		RecordedAt:                   authority.RecordedAt,
	}
	return authority, compilation, nil
}

func runPlanEntryRecords(entries []runplan.Entry) []artifacts.RunPlanGateEntryRecord {
	if len(entries) == 0 {
		return nil
	}
	out := make([]artifacts.RunPlanGateEntryRecord, 0, len(entries))
	for _, entry := range entries {
		out = append(out, runPlanEntryRecord(entry))
	}
	return out
}

func runPlanEntryRecord(entry runplan.Entry) artifacts.RunPlanGateEntryRecord {
	return artifacts.RunPlanGateEntryRecord{
		EntryID:                 strings.TrimSpace(entry.EntryID),
		EntryKind:               strings.TrimSpace(entry.EntryKind),
		PlanCheckpointCode:      strings.TrimSpace(entry.CheckpointCode),
		PlanOrderIndex:          entry.OrderIndex,
		GateID:                  strings.TrimSpace(entry.Gate.GateID),
		GateKind:                strings.TrimSpace(entry.Gate.GateKind),
		GateVersion:             strings.TrimSpace(entry.Gate.GateVersion),
		StageID:                 strings.TrimSpace(entry.StageID),
		StepID:                  strings.TrimSpace(entry.StepID),
		RoleInstanceID:          strings.TrimSpace(entry.RoleInstanceID),
		MaxAttempts:             maxAttemptsFromEntry(entry),
		ExpectedInputDigests:    normalizedInputDigestsFromEntry(entry),
		DependencyCacheHandoffs: runPlanDependencyCacheHandoffs(entry.DependencyCacheHandoffs),
	}
}

func runPlanDependencyCacheHandoffs(handoffs []runplan.DependencyCacheHandoff) []artifacts.RunPlanDependencyCacheHandoffRecord {
	if len(handoffs) == 0 {
		return nil
	}
	out := make([]artifacts.RunPlanDependencyCacheHandoffRecord, 0, len(handoffs))
	for _, handoff := range handoffs {
		requestDigest, err := handoff.RequestDigest.Identity()
		if err != nil {
			continue
		}
		out = append(out, artifacts.RunPlanDependencyCacheHandoffRecord{
			RequestDigest: strings.TrimSpace(requestDigest),
			ConsumerRole:  strings.TrimSpace(handoff.ConsumerRole),
			Required:      handoff.Required,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizedInputDigestsFromEntry(entry runplan.Entry) []string {
	if len(entry.Gate.NormalizedInputs) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(entry.Gate.NormalizedInputs))
	for _, input := range entry.Gate.NormalizedInputs {
		digest, _ := input["input_digest"].(string)
		digest = strings.TrimSpace(digest)
		if digest == "" {
			continue
		}
		if _, exists := seen[digest]; exists {
			continue
		}
		seen[digest] = struct{}{}
		out = append(out, digest)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func maxAttemptsFromEntry(entry runplan.Entry) int {
	retry, ok := entry.Gate.RetrySemantics["max_attempts"].(float64)
	if !ok {
		return 1
	}
	if retry < 1 {
		return 1
	}
	return int(retry)
}

func parseCompiledAt(value string) (parsed time.Time) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
