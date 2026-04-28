package artifacts

import (
	"errors"
	"os"
	"path/filepath"
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

func TestAppendSessionExecutionTriggerDeniesArtifactOnlyLabelWithBoundArtifactsOverlap(t *testing.T) {
	store := newTestStore(t)
	occurredAt := time.Now().UTC()

	_, err := store.AppendSessionExecutionTrigger(SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-labeled-a",
		AuthoritativeRepositoryRoot: "/repo/root",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "change_draft",
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
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "spec_draft",
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
	store.storeIO.statePath = filepath.Join(t.TempDir(), "state-dir")
	if err := os.MkdirAll(store.storeIO.statePath, 0o755); err != nil {
		t.Fatalf("mkdir failing state path: %v", err)
	}
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
