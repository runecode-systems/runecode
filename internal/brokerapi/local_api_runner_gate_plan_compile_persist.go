package brokerapi

import (
	"encoding/json"
	"fmt"
	"io"
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
	ApprovedInputSetDigest       string
}

type CompileAndPersistRunPlanResult struct {
	RunID             string
	PlanID            string
	RunPlanDigest     string
	CompilationDigest string
}

func (s *Service) CompileAndPersistRunPlan(req CompileAndPersistRunPlanRequest) (CompileAndPersistRunPlanResult, error) {
	input, workflowRef, processRef, err := s.compileRunPlanInputFromArtifacts(req)
	if err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	if err := ensureRunExistsForGatePlan(s.RunStatuses(), input.RunID, false); err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	identity, cacheKey, err := compileIdentityFromInput(workflowRef, processRef, req.ApprovedInputSetDigest, input)
	if err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	if cached, ok, err := s.lookupCompileCache(cacheKey, identity); err != nil {
		return CompileAndPersistRunPlanResult{}, err
	} else if ok {
		return cached, nil
	}
	inFlight, owner := s.compileCoordinator.startOrJoin(cacheKey)
	if !owner {
		<-inFlight.ready
		return inFlight.res, inFlight.err
	}
	release := s.compileCoordinator.acquire()
	defer release()
	res, compileErr := s.compileAndPersistRunPlanUncached(req, input, workflowRef, processRef, cacheKey)
	s.compileCoordinator.complete(cacheKey, inFlight, res, compileErr)
	return res, compileErr
}

func (s *Service) compileAndPersistRunPlanUncached(req CompileAndPersistRunPlanRequest, input runplan.CompileInput, workflowRef, processRef, cacheKey string) (CompileAndPersistRunPlanResult, error) {
	planned, err := s.compileRunPlanForPersistence(input)
	if err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	ref, err := s.persistCompiledRunPlanArtifact(planned)
	if err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	authority, compilation, err := runPlanRecordsFromCompiled(planned, ref.Digest, workflowRef, processRef, cacheKey)
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

func (s *Service) compileRunPlanForPersistence(input runplan.CompileInput) (runplan.RunPlan, error) {
	planned, err := runplan.Compile(input)
	if err != nil {
		return runplan.RunPlan{}, err
	}
	return planned, nil
}

func (s *Service) lookupCompileCache(cacheKey string, _ compileIdentity) (CompileAndPersistRunPlanResult, bool, error) {
	compilation, ok := s.RunPlanCompilationRecordByCacheKey(cacheKey)
	if !ok {
		return CompileAndPersistRunPlanResult{}, false, nil
	}
	if strings.TrimSpace(compilation.CompileCacheKey) != strings.TrimSpace(cacheKey) {
		return CompileAndPersistRunPlanResult{}, false, fmt.Errorf("compile cache key drift for run_id=%q plan_id=%q", compilation.RunID, compilation.PlanID)
	}
	return CompileAndPersistRunPlanResult{RunID: compilation.RunID, PlanID: compilation.PlanID, RunPlanDigest: compilation.RunPlanDigest, CompilationDigest: compilation.RecordDigest}, true, nil
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

func (s *Service) compileRunPlanInputFromArtifacts(req CompileAndPersistRunPlanRequest) (runplan.CompileInput, string, string, error) {
	runID := strings.TrimSpace(req.RunID)
	planID := strings.TrimSpace(req.PlanID)
	if runID == "" || planID == "" {
		return runplan.CompileInput{}, "", "", fmt.Errorf("run_id and plan_id are required")
	}
	workflowRef := strings.TrimSpace(req.WorkflowDefinitionRef)
	processRef := strings.TrimSpace(req.ProcessDefinitionRef)
	if workflowRef == "" || processRef == "" {
		return runplan.CompileInput{}, "", "", fmt.Errorf("workflow_definition_ref and process_definition_ref are required")
	}
	workflowBytes, err := s.readArtifactPayload(workflowRef)
	if err != nil {
		return runplan.CompileInput{}, "", "", err
	}
	processBytes, err := s.readArtifactPayload(processRef)
	if err != nil {
		return runplan.CompileInput{}, "", "", err
	}
	projectContext := strings.TrimSpace(req.ProjectContextIdentityDigest)
	if projectContext == "" {
		projectContext = strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
	}
	if projectContext == "" {
		return runplan.CompileInput{}, "", "", fmt.Errorf("project_context_identity_digest is required for trusted run plan compilation")
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
	}, workflowRef, processRef, nil
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

func runPlanRecordsFromCompiled(plan runplan.RunPlan, runPlanDigest string, workflowRef string, processRef string, compileCacheKey string) (artifacts.RunPlanAuthorityRecord, artifacts.RunPlanCompilationRecord, error) {
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
		CompileCacheKey:              strings.TrimSpace(compileCacheKey),
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
