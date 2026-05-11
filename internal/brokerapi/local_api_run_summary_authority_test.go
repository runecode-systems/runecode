package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/runplan"
)

func TestRunSummaryUsesBuiltInWorkflowAuthorityForSessionExecutionPath(t *testing.T) {
	s, runID, entry := compileRunSummaryAuthorityFixture(t)

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{
		SchemaID:      "runecode.protocol.v0.RunGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-run-summary-authority",
		RunID:         runID,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.WorkflowKind); got != strings.TrimSpace(entry.WorkflowID) {
		t.Fatalf("workflow_kind = %q, want %q", got, entry.WorkflowID)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.WorkflowDefinitionHash); got != strings.TrimSpace(entry.WorkflowDefinitionHash) {
		t.Fatalf("workflow_definition_hash = %q, want %q", got, entry.WorkflowDefinitionHash)
	}
}

func compileRunSummaryAuthorityFixture(t *testing.T) (*Service, string, runplan.BuiltInWorkflowCatalogEntry) {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-summary-authority"
	entry, err := builtInCatalogEntryForWorkflowOperation(sessionWorkflowOperationApprovedImplementation)
	if err != nil {
		t.Fatalf("builtInCatalogEntryForWorkflowOperation returned error: %v", err)
	}
	workflowPayload, processPayload, err := builtInWorkflowAssetPayloads(entry.WorkflowID)
	if err != nil {
		t.Fatalf("builtInWorkflowAssetPayloads returned error: %v", err)
	}
	workflowRef, processRef, err := s.persistSessionExecutionWorkflowAssets(runID, workflowPayload, processPayload)
	if err != nil {
		t.Fatalf("persistSessionExecutionWorkflowAssets returned error: %v", err)
	}
	if err := s.SetRunStatus(runID, "starting"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	if _, err := s.CompileAndPersistRunPlan(CompileAndPersistRunPlanRequest{
		RunID:                 runID,
		PlanID:                "plan-run-summary-authority",
		WorkflowDefinitionRef: workflowRef.Digest,
		ProcessDefinitionRef:  processRef.Digest,
		PolicyContextHash:     artifacts.DigestBytes([]byte("run-summary-authority")),
	}); err != nil {
		t.Fatalf("CompileAndPersistRunPlan returned error: %v", err)
	}
	return s, runID, entry
}
