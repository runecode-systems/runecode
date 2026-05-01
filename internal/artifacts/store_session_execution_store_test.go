package artifacts

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAppendSessionExecutionTriggerDeniesMutationBearingRepoRootOverlap(t *testing.T) {
	store := newTestStore(t)
	occurredAt := time.Now().UTC()

	_, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-mutation-a",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "draft_promote_apply",
		},
		OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatalf("first AppendSessionExecutionTrigger returned error: %v", err)
	}

	_, err = store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-mutation-b",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "approved_change_implementation",
		},
		OccurredAt: occurredAt.Add(time.Second),
	})
	if !errors.Is(err, ErrSessionExecutionTriggerOverlapDenied) {
		t.Fatalf("second AppendSessionExecutionTrigger error = %v, want %v", err, ErrSessionExecutionTriggerOverlapDenied)
	}
}

func TestAppendSessionExecutionTriggerAllowsArtifactOnlyRepoRootOverlap(t *testing.T) {
	store := newTestStore(t)
	occurredAt := time.Now().UTC()

	_, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-artifact-a",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "change_draft",
		},
		OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatalf("first AppendSessionExecutionTrigger returned error: %v", err)
	}

	res, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-artifact-b",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "spec_draft",
		},
		OccurredAt: occurredAt.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("second AppendSessionExecutionTrigger returned error: %v", err)
	}
	if !res.Created {
		t.Fatal("second AppendSessionExecutionTrigger Created = false, want true")
	}
}

func TestAppendSessionExecutionTriggerAllowsMutationWhenExistingExecutionIsNonMutation(t *testing.T) {
	store := newTestStore(t)
	occurredAt := time.Now().UTC()

	_, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-non-mutation-a",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "change_draft",
		},
		OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatalf("first AppendSessionExecutionTrigger returned error: %v", err)
	}

	res, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-mutation-b",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "approved_change_implementation",
		},
		OccurredAt: occurredAt.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("second AppendSessionExecutionTrigger returned error: %v", err)
	}
	if !res.Created {
		t.Fatal("second AppendSessionExecutionTrigger Created = false, want true")
	}
}

func TestAppendSessionExecutionTriggerDeniesMutationWhenExistingExecutionRoutingIsUnknownOrMissing(t *testing.T) {
	for _, tc := range []struct {
		name    string
		routing SessionWorkflowPackRoutingDurableState
	}{
		{name: "unknown workflow family", routing: SessionWorkflowPackRoutingDurableState{WorkflowFamily: "custom", WorkflowOperation: "something"}},
		{name: "missing routing", routing: SessionWorkflowPackRoutingDurableState{}},
	} {
		t.Run(tc.name, func(t *testing.T) { assertExistingExecutionBlocksMutationStart(t, tc.routing) })
	}
}

func TestAppendSessionExecutionTriggerDeniesMutationWhenExistingExecutionLosesRepoRootLinkage(t *testing.T) {
	store := newTestStore(t)
	occurredAt := time.Now().UTC()

	first, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-missing-root-a",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "draft_promote_apply",
		},
		OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatalf("first AppendSessionExecutionTrigger returned error: %v", err)
	}
	clearExecutionTriggerRepoRoot(store, "sess-missing-root-a", first.Trigger.TriggerID)

	_, err = store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-missing-root-b",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "approved_change_implementation",
		},
		OccurredAt: occurredAt.Add(time.Second),
	})
	if !errors.Is(err, ErrSessionExecutionTriggerOverlapDenied) {
		t.Fatalf("second AppendSessionExecutionTrigger error = %v, want %v", err, ErrSessionExecutionTriggerOverlapDenied)
	}
}

func TestAppendSessionExecutionTriggerDeniesMutationWhenExistingExecutionWasContinued(t *testing.T) {
	store := newTestStore(t)
	occurredAt := time.Now().UTC()
	first, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{SessionID: "sess-continued-a", AuthoritativeRepositoryRoot: "/repo/root", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: SessionWorkflowPackRoutingDurableState{WorkflowFamily: "runecontext", WorkflowOperation: "draft_promote_apply"}, OccurredAt: occurredAt})
	if err != nil {
		t.Fatalf("first AppendSessionExecutionTrigger returned error: %v", err)
	}
	if _, err := store.UpdateSessionTurnExecution(SessionTurnExecutionUpdateRequest{SessionID: "sess-continued-a", TurnID: first.TurnExecution.TurnID, ExecutionState: "waiting", OccurredAt: occurredAt.Add(time.Second)}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
	if _, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{SessionID: "sess-continued-a", AuthoritativeRepositoryRoot: "/repo/root", TriggerSource: "resume_follow_up", RequestedOperation: "continue", WorkflowRouting: SessionWorkflowPackRoutingDurableState{WorkflowFamily: "runecontext", WorkflowOperation: "draft_promote_apply"}, OccurredAt: occurredAt.Add(2 * time.Second)}); err != nil {
		t.Fatalf("continue AppendSessionExecutionTrigger returned error: %v", err)
	}
	assertMutationStartDenied(t, store, "sess-continued-b", occurredAt.Add(3*time.Second))
}

func TestAppendSessionExecutionTriggerDeniesArtifactOnlyLabelWithBoundArtifactsOverlap(t *testing.T) {
	store := newTestStore(t)
	occurredAt := time.Now().UTC()

	_, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-labeled-a",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:      "runecontext",
			WorkflowOperation:   "change_draft",
			BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifactDurableState{{ArtifactRef: "change_draft_artifact", ArtifactDigest: "sha256:" + strings.Repeat("a", 64)}},
		},
		OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatalf("first AppendSessionExecutionTrigger returned error: %v", err)
	}

	_, err = store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-labeled-b",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:      "runecontext",
			WorkflowOperation:   "spec_draft",
			BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifactDurableState{{ArtifactRef: "spec_draft_artifact", ArtifactDigest: "sha256:" + strings.Repeat("b", 64)}},
		},
		OccurredAt: occurredAt.Add(time.Second),
	})
	if !errors.Is(err, ErrSessionExecutionTriggerOverlapDenied) {
		t.Fatalf("second AppendSessionExecutionTrigger error = %v, want %v", err, ErrSessionExecutionTriggerOverlapDenied)
	}
}

func TestAppendSessionExecutionTriggerSaveFailureRollsBackInMemoryState(t *testing.T) {
	store := newTestStore(t)
	store.storeIO.statePath = brokenStateSavePath(store.rootDir)
	req := SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-save-fail",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "draft_promote_apply",
		},
		IdempotencyKey:  "idem-save-fail",
		IdempotencyHash: "hash-save-fail",
		OccurredAt:      time.Now().UTC(),
	}
	if _, err := store.AppendSessionExecutionTrigger(req); err == nil {
		t.Fatal("AppendSessionExecutionTrigger expected save failure")
	}
	if _, ok := store.state.Sessions[req.SessionID]; ok {
		t.Fatal("session persisted in memory after failed save")
	}
	if _, err := store.AppendSessionExecutionTrigger(req); err == nil {
		t.Fatal("retry AppendSessionExecutionTrigger expected save failure, not in-memory replay")
	}
}

func TestAppendSessionExecutionTriggerSaveFailureRestoresExistingSessionState(t *testing.T) {
	store := newTestStore(t)
	base, before := seedExistingSessionForSaveFailureTest(t, store)
	var err error
	store.storeIO.statePath = brokenStateSavePath(store.rootDir)
	_, err = store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-existing-save-fail",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "draft_promote_apply",
		},
		IdempotencyKey:  "idem-existing-next",
		IdempotencyHash: "hash-existing-next",
		OccurredAt:      time.Now().UTC().Add(time.Second),
	})
	if err == nil {
		t.Fatal("AppendSessionExecutionTrigger expected save failure for existing session")
	}
	after := store.state.Sessions["sess-existing-save-fail"]
	if len(after.ExecutionTriggers) != len(before.ExecutionTriggers) || len(after.TurnExecutions) != len(before.TurnExecutions) {
		t.Fatalf("existing session counts changed after failed save: before triggers=%d turns=%d after triggers=%d turns=%d", len(before.ExecutionTriggers), len(before.TurnExecutions), len(after.ExecutionTriggers), len(after.TurnExecutions))
	}
	if len(after.ExecutionTriggerIdempotencyByKey) != len(before.ExecutionTriggerIdempotencyByKey) {
		t.Fatalf("existing session idempotency changed after failed save: before=%d after=%d", len(before.ExecutionTriggerIdempotencyByKey), len(after.ExecutionTriggerIdempotencyByKey))
	}
	if after.ExecutionTriggers[0].TriggerID != base.Trigger.TriggerID {
		t.Fatalf("existing session trigger head changed after failed save: got %q want %q", after.ExecutionTriggers[0].TriggerID, base.Trigger.TriggerID)
	}
}

func assertExistingExecutionBlocksMutationStart(t *testing.T, routing SessionWorkflowPackRoutingDurableState) {
	t.Helper()
	store := newTestStore(t)
	occurredAt := time.Now().UTC()
	_, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{SessionID: "sess-unknown-a", AuthoritativeRepositoryRoot: "/repo/root", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: routing, OccurredAt: occurredAt})
	if err != nil {
		t.Fatalf("first AppendSessionExecutionTrigger returned error: %v", err)
	}
	assertMutationStartDenied(t, store, "sess-mutation-b", occurredAt.Add(time.Second))
}

func assertMutationStartDenied(t *testing.T, store *Store, sessionID string, occurredAt time.Time) {
	t.Helper()
	_, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{SessionID: sessionID, AuthoritativeRepositoryRoot: "/repo/root", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: SessionWorkflowPackRoutingDurableState{WorkflowFamily: "runecontext", WorkflowOperation: "approved_change_implementation"}, OccurredAt: occurredAt})
	if !errors.Is(err, ErrSessionExecutionTriggerOverlapDenied) {
		t.Fatalf("second AppendSessionExecutionTrigger error = %v, want %v", err, ErrSessionExecutionTriggerOverlapDenied)
	}
}

func seedExistingSessionForSaveFailureTest(t *testing.T, store *Store) (SessionExecutionTriggerAppendResult, SessionDurableState) {
	t.Helper()
	base, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{SessionID: "sess-existing-save-fail", AuthoritativeRepositoryRoot: "/repo/root", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: SessionWorkflowPackRoutingDurableState{WorkflowFamily: "runecontext", WorkflowOperation: "draft_promote_apply"}, IdempotencyKey: "idem-existing-base", IdempotencyHash: "hash-existing-base", OccurredAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("base AppendSessionExecutionTrigger returned error: %v", err)
	}
	return base, copySessionDurableState(store.state.Sessions["sess-existing-save-fail"])
}

func clearExecutionTriggerRepoRoot(store *Store, sessionID, triggerID string) {
	session := store.state.Sessions[sessionID]
	for i := range session.ExecutionTriggers {
		if session.ExecutionTriggers[i].TriggerID == triggerID {
			session.ExecutionTriggers[i].AuthoritativeRepositoryRoot = ""
		}
	}
	store.state.Sessions[sessionID] = session
}
