package brokerapi

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

func TestSessionExecutionRunIDAvoidsNormalizedSessionTokenCollisions(t *testing.T) {
	first := sessionExecutionRunID("A.B", 1)
	second := sessionExecutionRunID("a_b", 1)

	if first == second {
		t.Fatalf("sessionExecutionRunID(A.B, 1) = %q, sessionExecutionRunID(a_b, 1) = %q; want distinct values", first, second)
	}
}

func TestSessionExecutionRunIDIsDeterministicPortableAndIndexed(t *testing.T) {
	const sessionID = "sess-trigger-create"
	got := sessionExecutionRunID(sessionID, 3)
	wantPattern := regexp.MustCompile(`^run_sess-trigger-create_[0-9a-f]{64}_3$`)
	if !wantPattern.MatchString(got) {
		t.Fatalf("sessionExecutionRunID(%q, 3) = %q, want %s", sessionID, got, wantPattern.String())
	}

	if again := sessionExecutionRunID(sessionID, 3); again != got {
		t.Fatalf("sessionExecutionRunID(%q, 3) = %q on repeat, want %q", sessionID, again, got)
	}

	if firstIndex := sessionExecutionRunID(sessionID, 0); !regexp.MustCompile(`_1$`).MatchString(firstIndex) {
		t.Fatalf("sessionExecutionRunID(%q, 0) = %q, want suffix _1", sessionID, firstIndex)
	}
	if len(got) > 128 {
		t.Fatalf("sessionExecutionRunID(%q, 3) length = %d, want <= 128", sessionID, len(got))
	}

	longSessionID := strings.Repeat("A", 128)
	if got := sessionExecutionRunID(longSessionID, 1); len(got) > 128 {
		t.Fatalf("sessionExecutionRunID(longSessionID, 1) length = %d, want <= 128", len(got))
	}
	if got := sessionExecutionRunID(sessionID, 1234567890); len(got) > 128 {
		t.Fatalf("sessionExecutionRunID(%q, 1234567890) length = %d, want <= 128", sessionID, len(got))
	}
	if !regexp.MustCompile(`_1234567890$`).MatchString(sessionExecutionRunID(sessionID, 1234567890)) {
		t.Fatalf("sessionExecutionRunID(%q, 1234567890) missing full index suffix", sessionID)
	}
}

func TestSessionExecutionDerivedPlanIDIsDeterministicAndBounded(t *testing.T) {
	sourceID := sessionExecutionRunID(strings.Repeat("A", 128), 1234567890)
	got := sessionExecutionDerivedPlanID(sourceID, 1234567890)
	if len(got) > 128 {
		t.Fatalf("sessionExecutionDerivedPlanID length = %d, want <= 128", len(got))
	}
	if !regexp.MustCompile(`^plan_[a-z][a-z0-9_-]*_[0-9a-f]{64}_1234567890$`).MatchString(got) {
		t.Fatalf("sessionExecutionDerivedPlanID = %q, want bounded deterministic plan id", got)
	}
}

func TestSessionExecutionDerivedAttemptIDIsDeterministicAndBounded(t *testing.T) {
	sourceID := sessionExecutionDerivedPlanID(sessionExecutionRunID(strings.Repeat("A", 128), 1234567890), 1234567890)
	got := sessionExecutionDerivedAttemptID("stage_attempt", sourceID, 1234567890)
	if len(got) > 128 {
		t.Fatalf("sessionExecutionDerivedAttemptID length = %d, want <= 128", len(got))
	}
	if !regexp.MustCompile(`^stage_attempt_[a-z][a-z0-9_-]*_[0-9a-f]{64}_1234567890$`).MatchString(got) {
		t.Fatalf("sessionExecutionDerivedAttemptID = %q, want bounded deterministic attempt id", got)
	}
}

func TestEnsureSessionExecutionPrimaryRunBindingRepairsPreviouslyUninitializedRun(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	appendResult, runID := seedUninitializedSessionExecutionRunBinding(t, s)
	if runID == "" {
		t.Fatal("primary_run_id is empty")
	}
	updated, errResp := s.ensureSessionExecutionPrimaryRunBinding("req-session-trigger-repair-ensure", "sess-trigger-repair", appendResult.TurnExecution)
	if errResp != nil {
		t.Fatalf("ensureSessionExecutionPrimaryRunBinding returned error: %+v", errResp)
	}
	assertRecoveredSessionExecutionRunBinding(t, s, updated, runID)
}

func seedUninitializedSessionExecutionRunBinding(t *testing.T, s *Service) (artifacts.SessionExecutionTriggerAppendResult, string) {
	t.Helper()
	appendResult, err := s.AppendSessionExecutionTrigger(artifacts.SessionExecutionTriggerAppendRequest{SessionID: "sess-trigger-repair", WorkspaceID: "workspace-local", AuthoritativeRepositoryRoot: "/repo/root", TriggerSource: "interactive_user", RequestedOperation: "start", ExecutionState: "running", WorkflowRouting: artifacts.SessionWorkflowPackRoutingDurableState{WorkflowFamily: "runecontext", WorkflowOperation: "change_draft"}, OccurredAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("AppendSessionExecutionTrigger returned error: %v", err)
	}
	runID := sessionExecutionRunID("sess-trigger-repair", appendResult.TurnExecution.ExecutionIndex)
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: "sess-trigger-repair", TurnID: appendResult.TurnExecution.TurnID, ExecutionState: appendResult.TurnExecution.ExecutionState, WaitKind: appendResult.TurnExecution.WaitKind, WaitState: appendResult.TurnExecution.WaitState, OrchestrationScopeID: appendResult.TurnExecution.OrchestrationScopeID, DependsOnScopeIDs: append([]string{}, appendResult.TurnExecution.DependsOnScopeIDs...), PrimaryRunID: runID, PendingApprovalID: appendResult.TurnExecution.PendingApprovalID, LinkedRunIDs: uniqueSortedStrings(append(append([]string{}, appendResult.TurnExecution.LinkedRunIDs...), runID)), LinkedApprovalIDs: append([]string{}, appendResult.TurnExecution.LinkedApprovalIDs...), LinkedArtifactDigests: append([]string{}, appendResult.TurnExecution.LinkedArtifactDigests...), LinkedAuditRecordDigests: append([]string{}, appendResult.TurnExecution.LinkedAuditRecordDigests...), BlockedReasonCode: appendResult.TurnExecution.BlockedReasonCode, TerminalOutcome: appendResult.TurnExecution.TerminalOutcome, BoundValidatedProjectSubstrateDigest: appendResult.TurnExecution.BoundValidatedProjectSubstrateDigest, OccurredAt: time.Now().UTC()}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
	if _, err := s.UpdateSessionState("sess-trigger-repair", func(state artifacts.SessionDurableState) artifacts.SessionDurableState {
		state.LinkedRunIDs = uniqueSortedStrings(append(state.LinkedRunIDs, runID))
		state.CreatedByRunID = runID
		return state
	}); err != nil {
		t.Fatalf("UpdateSessionState returned error: %v", err)
	}
	return appendResult, runID
}

func assertRecoveredSessionExecutionRunBinding(t *testing.T, s *Service, updated artifacts.SessionTurnExecutionDurableState, runID string) {
	t.Helper()
	if updated.PrimaryRunID != runID {
		t.Fatalf("updated primary_run_id = %q, want %q", updated.PrimaryRunID, runID)
	}
	if status := s.RunStatuses()[runID]; status != "starting" {
		t.Fatalf("run status = %q, want starting", status)
	}
	facts := s.RuntimeFacts(runID)
	if facts.LaunchReceipt.RunID != runID {
		t.Fatalf("runtime facts run_id = %q, want %q", facts.LaunchReceipt.RunID, runID)
	}
	if facts.LaunchReceipt.SessionID != "sess-trigger-repair" {
		t.Fatalf("runtime facts session_id = %q, want sess-trigger-repair", facts.LaunchReceipt.SessionID)
	}
	if facts.LaunchReceipt.Lifecycle != nil && facts.LaunchReceipt.Lifecycle.CurrentState == "" {
		t.Fatalf("runtime lifecycle = %+v, want empty or valid lifecycle state", facts.LaunchReceipt.Lifecycle)
	}
}
