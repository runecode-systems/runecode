package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/runplan"
)

type sessionExecutionPlanAuthority struct {
	runID                        string
	planID                       string
	planCheckpointCode           string
	planOrderIndex               int
	gateID                       string
	gateKind                     string
	gateVersion                  string
	stageID                      string
	stepID                       string
	roleInstanceID               string
	expectedInputDigest          string
	workflowDefinitionRef        string
	processDefinitionRef         string
	workflowDefinitionHash       string
	processDefinitionHash        string
	projectContextIdentityDigest string
	workflowOperation            string
	draftArtifactSchemaID        string
}

func (s *Service) ensureSessionExecutionRunPlanAuthority(result artifacts.SessionExecutionTriggerAppendResult) (sessionExecutionPlanAuthority, error) {
	authorityInputs, err := sessionExecutionAuthorityInputs(result)
	if err != nil {
		return sessionExecutionPlanAuthority{}, err
	}
	workflowRef, processRef, err := s.persistSessionExecutionAuthorityAssets(authorityInputs.runID, authorityInputs.entry.WorkflowID)
	if err != nil {
		return sessionExecutionPlanAuthority{}, err
	}
	projectContextIdentityDigest, err := sessionExecutionProjectContextIdentityDigest(s, result.TurnExecution)
	if err != nil {
		return sessionExecutionPlanAuthority{}, err
	}
	compiled, err := s.compileSessionExecutionRunPlan(authorityInputs.runID, sessionExecutionPlanID(authorityInputs.runID, result.TurnExecution.ExecutionIndex), workflowRef.Digest, processRef.Digest, projectContextIdentityDigest, result)
	if err != nil {
		return sessionExecutionPlanAuthority{}, err
	}
	selectedEntry, err := s.sessionExecutionSelectedPlanEntry(authorityInputs.runID)
	if err != nil {
		return sessionExecutionPlanAuthority{}, err
	}
	return newSessionExecutionPlanAuthority(authorityInputs, compiled, selectedEntry, workflowRef, processRef, projectContextIdentityDigest), nil
}

type sessionExecutionAuthorityInputSet struct {
	runID             string
	workflowOperation string
	entry             runplan.BuiltInWorkflowCatalogEntry
}

func sessionExecutionAuthorityInputs(result artifacts.SessionExecutionTriggerAppendResult) (sessionExecutionAuthorityInputSet, error) {
	runID := strings.TrimSpace(result.TurnExecution.PrimaryRunID)
	if runID == "" {
		return sessionExecutionAuthorityInputSet{}, fmt.Errorf("session execution run binding missing primary run id")
	}
	workflowOperation := strings.TrimSpace(result.TurnExecution.WorkflowRouting.WorkflowOperation)
	entry, err := builtInCatalogEntryForWorkflowOperation(workflowOperation)
	if err != nil {
		return sessionExecutionAuthorityInputSet{}, err
	}
	return sessionExecutionAuthorityInputSet{runID: runID, workflowOperation: workflowOperation, entry: entry}, nil
}

func (s *Service) persistSessionExecutionAuthorityAssets(runID, workflowID string) (artifacts.ArtifactReference, artifacts.ArtifactReference, error) {
	workflowPayload, processPayload, err := builtInWorkflowAssetPayloads(workflowID)
	if err != nil {
		return artifacts.ArtifactReference{}, artifacts.ArtifactReference{}, err
	}
	return s.persistSessionExecutionWorkflowAssets(runID, workflowPayload, processPayload)
}

func sessionExecutionProjectContextIdentityDigest(s *Service, turnExecution artifacts.SessionTurnExecutionDurableState) (string, error) {
	projectContextIdentityDigest := strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
	if projectContextIdentityDigest == "" {
		projectContextIdentityDigest = strings.TrimSpace(turnExecution.BoundValidatedProjectSubstrateDigest)
	}
	if projectContextIdentityDigest == "" {
		return "", fmt.Errorf("validated project context identity digest is required for session execution run plan authority")
	}
	return projectContextIdentityDigest, nil
}

func (s *Service) compileSessionExecutionRunPlan(runID, planID, workflowRef, processRef, projectContextIdentityDigest string, result artifacts.SessionExecutionTriggerAppendResult) (CompileAndPersistRunPlanResult, error) {
	compiled, err := s.CompileAndPersistRunPlan(CompileAndPersistRunPlanRequest{
		RunID:                        runID,
		PlanID:                       planID,
		WorkflowDefinitionRef:        workflowRef,
		ProcessDefinitionRef:         processRef,
		PolicyContextHash:            sessionExecutionPolicyContextHash(result),
		ProjectContextIdentityDigest: projectContextIdentityDigest,
		ApprovedInputSetDigest:       approvedInputSetDigestForSessionExecution(result),
	})
	if err != nil {
		return CompileAndPersistRunPlanResult{}, err
	}
	if strings.TrimSpace(compiled.PlanID) == "" {
		return CompileAndPersistRunPlanResult{}, fmt.Errorf("trusted run plan compilation returned empty plan id")
	}
	return compiled, nil
}

func (s *Service) sessionExecutionSelectedPlanEntry(runID string) (artifacts.RunPlanGateEntryRecord, error) {
	authorityRecord, ok, err := s.ActiveRunPlanAuthority(runID)
	if err != nil {
		return artifacts.RunPlanGateEntryRecord{}, err
	}
	if !ok {
		return artifacts.RunPlanGateEntryRecord{}, fmt.Errorf("trusted run plan authority missing after compile for run %q", runID)
	}
	return selectSessionExecutionPlanEntry(authorityRecord.Entries)
}

func firstExpectedInputDigest(entry artifacts.RunPlanGateEntryRecord) string {
	for _, digest := range entry.ExpectedInputDigests {
		trimmed := strings.TrimSpace(digest)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func selectSessionExecutionPlanEntry(entries []artifacts.RunPlanGateEntryRecord) (artifacts.RunPlanGateEntryRecord, error) {
	if len(entries) == 0 {
		return artifacts.RunPlanGateEntryRecord{}, fmt.Errorf("trusted run plan authority has no gate entries")
	}
	selected := entries[0]
	for _, entry := range entries[1:] {
		if entry.PlanOrderIndex > selected.PlanOrderIndex {
			selected = entry
		}
	}
	if strings.TrimSpace(selected.PlanCheckpointCode) == "" {
		return artifacts.RunPlanGateEntryRecord{}, fmt.Errorf("trusted run plan authority selected entry missing plan_checkpoint_code")
	}
	if strings.TrimSpace(selected.GateID) == "" || strings.TrimSpace(selected.GateKind) == "" || strings.TrimSpace(selected.GateVersion) == "" {
		return artifacts.RunPlanGateEntryRecord{}, fmt.Errorf("trusted run plan authority selected entry missing gate identity")
	}
	return selected, nil
}

func (s *Service) persistSessionExecutionWorkflowAssets(runID string, workflowPayload, processPayload []byte) (artifacts.ArtifactReference, artifacts.ArtifactReference, error) {
	workflowRef, err := s.Put(artifacts.PutRequest{
		Payload:               workflowPayload,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: artifacts.DigestBytes(workflowPayload),
		CreatedByRole:         "brokerapi",
		TrustedSource:         true,
		RunID:                 runID,
		StepID:                "session_execution/workflow_definition",
	})
	if err != nil {
		return artifacts.ArtifactReference{}, artifacts.ArtifactReference{}, fmt.Errorf("persist built-in workflow definition: %w", err)
	}
	processRef, err := s.Put(artifacts.PutRequest{
		Payload:               processPayload,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: artifacts.DigestBytes(processPayload),
		CreatedByRole:         "brokerapi",
		TrustedSource:         true,
		RunID:                 runID,
		StepID:                "session_execution/process_definition",
	})
	if err != nil {
		return artifacts.ArtifactReference{}, artifacts.ArtifactReference{}, fmt.Errorf("persist built-in process definition: %w", err)
	}
	return workflowRef, processRef, nil
}

func sessionExecutionPlanID(runID string, executionIndex int) string {
	if executionIndex < 1 {
		executionIndex = 1
	}
	return fmt.Sprintf("plan_%s_%06d", sessionExecutionIdentifierToken(runID), executionIndex)
}

func sessionExecutionPolicyContextHash(result artifacts.SessionExecutionTriggerAppendResult) string {
	payload := strings.TrimSpace(result.TurnExecution.WorkflowRouting.WorkflowFamily) + "\n" + strings.TrimSpace(result.TurnExecution.WorkflowRouting.WorkflowOperation) + "\n" + strings.TrimSpace(result.TurnExecution.BoundValidatedProjectSubstrateDigest)
	if payload == "\n\n" {
		payload = strings.TrimSpace(result.Trigger.TriggerID)
	}
	return shaDigestIdentity(payload)
}

func approvedInputSetDigestForSessionExecution(result artifacts.SessionExecutionTriggerAppendResult) string {
	if strings.TrimSpace(result.TurnExecution.WorkflowRouting.WorkflowOperation) != sessionWorkflowOperationApprovedImplementation {
		return ""
	}
	for _, binding := range result.TurnExecution.WorkflowRouting.BoundInputArtifacts {
		if strings.TrimSpace(binding.ArtifactRef) == "implementation_input_set" {
			return strings.TrimSpace(binding.ArtifactDigest)
		}
	}
	return ""
}

func newSessionExecutionPlanAuthority(inputs sessionExecutionAuthorityInputSet, compiled CompileAndPersistRunPlanResult, selectedEntry artifacts.RunPlanGateEntryRecord, workflowRef, processRef artifacts.ArtifactReference, projectContextIdentityDigest string) sessionExecutionPlanAuthority {
	return sessionExecutionPlanAuthority{
		runID:                        inputs.runID,
		planID:                       strings.TrimSpace(compiled.PlanID),
		planCheckpointCode:           strings.TrimSpace(selectedEntry.PlanCheckpointCode),
		planOrderIndex:               selectedEntry.PlanOrderIndex,
		gateID:                       strings.TrimSpace(selectedEntry.GateID),
		gateKind:                     strings.TrimSpace(selectedEntry.GateKind),
		gateVersion:                  strings.TrimSpace(selectedEntry.GateVersion),
		stageID:                      strings.TrimSpace(selectedEntry.StageID),
		stepID:                       strings.TrimSpace(selectedEntry.StepID),
		roleInstanceID:               strings.TrimSpace(selectedEntry.RoleInstanceID),
		expectedInputDigest:          firstExpectedInputDigest(selectedEntry),
		workflowDefinitionRef:        workflowRef.Digest,
		processDefinitionRef:         processRef.Digest,
		workflowDefinitionHash:       strings.TrimSpace(inputs.entry.WorkflowDefinitionHash),
		processDefinitionHash:        strings.TrimSpace(inputs.entry.ProcessDefinitionHash),
		projectContextIdentityDigest: projectContextIdentityDigest,
		workflowOperation:            inputs.workflowOperation,
		draftArtifactSchemaID:        strings.TrimSpace(inputs.entry.DraftArtifactSchemaID),
	}
}
