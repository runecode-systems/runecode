package brokerapi

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestEnsureSessionExecutionRunPlanAuthorityCompilesBuiltInPlan(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-session-plan-authority"
	if err := s.SetRunStatus(runID, "starting"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	result := sessionExecutionPlanAuthorityAppendResult(s, runID)
	authority, err := s.ensureSessionExecutionRunPlanAuthority(result)
	if err != nil {
		t.Fatalf("ensureSessionExecutionRunPlanAuthority returned error: %v", err)
	}
	assertSessionExecutionPlanAuthorityFields(t, authority)
	stored, ok, err := s.ActiveRunPlanAuthority(runID)
	if err != nil {
		t.Fatalf("ActiveRunPlanAuthority returned error: %v", err)
	}
	if !ok {
		t.Fatal("active run plan authority missing")
	}
	if stored.PlanID != authority.planID {
		t.Fatalf("stored plan_id = %q, want %q", stored.PlanID, authority.planID)
	}
	if err := s.bridgeSessionExecutionTriggerToRun("req-session-plan-authority", result, authority); err != nil {
		t.Fatalf("bridgeSessionExecutionTriggerToRun returned error: %v", err)
	}
	runnerAdvisory, ok := s.RunnerAdvisory(runID)
	if !ok {
		t.Fatal("runner advisory missing after bridged checkpoint")
	}
	if runnerAdvisory.Lifecycle == nil || strings.TrimSpace(runnerAdvisory.Lifecycle.LifecycleState) != "completed" {
		t.Fatalf("runner advisory lifecycle = %+v, want completed", runnerAdvisory.Lifecycle)
	}
}

func sessionExecutionPlanAuthorityAppendResult(s *Service, runID string) artifacts.SessionExecutionTriggerAppendResult {
	return artifacts.SessionExecutionTriggerAppendResult{
		Trigger: artifacts.SessionExecutionTriggerDurableState{SessionID: "sess-session-plan-authority", TriggerID: "trigger-session-plan-authority"},
		TurnExecution: artifacts.SessionTurnExecutionDurableState{
			ExecutionIndex:                       1,
			PrimaryRunID:                         runID,
			BoundValidatedProjectSubstrateDigest: s.projectSubstrate.Snapshot.ValidatedSnapshotDigest,
			WorkflowRouting: artifacts.SessionWorkflowPackRoutingDurableState{
				WorkflowFamily:    "runecontext",
				WorkflowOperation: sessionWorkflowOperationChangeDraft,
			},
		},
	}
}

func assertSessionExecutionPlanAuthorityFields(t *testing.T, authority sessionExecutionPlanAuthority) {
	t.Helper()
	if authority.planID == "" {
		t.Fatal("plan_id is empty")
	}
	if authority.planCheckpointCode == "" {
		t.Fatal("plan_checkpoint_code is empty")
	}
	if authority.gateID == "" {
		t.Fatal("gate_id is empty")
	}
	if authority.draftArtifactSchemaID != "runecode.protocol.v0.RuneContextChangeDraftArtifact" {
		t.Fatalf("draft_artifact_schema_id = %q", authority.draftArtifactSchemaID)
	}
}
