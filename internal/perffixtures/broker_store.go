package perffixtures

import (
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

const (
	FixtureTUIEmptyV1   = "tui.empty.v1"
	FixtureTUIWaitingV1 = "tui.waiting.v1"
)

type BrokerStoreFixtureResult struct {
	FixtureID string
	SessionID string
	TurnID    string
	RootDir   string
}

func BuildBrokerStoreFixture(rootDir string, fixtureID string) (BrokerStoreFixtureResult, error) {
	switch fixtureID {
	case FixtureTUIEmptyV1:
		if _, err := artifacts.NewStore(rootDir); err != nil {
			return BrokerStoreFixtureResult{}, err
		}
		return BrokerStoreFixtureResult{FixtureID: fixtureID, RootDir: rootDir}, nil
	case FixtureTUIWaitingV1:
		return buildBrokerStoreWaiting(rootDir)
	default:
		return BrokerStoreFixtureResult{}, ErrUnsupportedFixtureID
	}
}

func buildBrokerStoreWaiting(rootDir string) (BrokerStoreFixtureResult, error) {
	store, err := artifacts.NewStore(rootDir)
	if err != nil {
		return BrokerStoreFixtureResult{}, err
	}
	seedTime := time.Date(2026, time.March, 18, 9, 0, 0, 0, time.UTC)
	appendResult, err := store.AppendSessionExecutionTrigger(waitingSessionExecutionTriggerAppendRequest(seedTime))
	if err != nil {
		return BrokerStoreFixtureResult{}, err
	}
	if _, err := store.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{
		SessionID:         "sess-manual-multiwait",
		TurnID:            appendResult.TurnExecution.TurnID,
		ExecutionState:    "waiting",
		WaitKind:          "approval",
		WaitState:         "awaiting_review",
		BlockedReasonCode: "approval_wait",
		OccurredAt:        seedTime.Add(10 * time.Second),
	}); err != nil {
		return BrokerStoreFixtureResult{}, err
	}
	return BrokerStoreFixtureResult{
		FixtureID: FixtureTUIWaitingV1,
		SessionID: "sess-manual-multiwait",
		TurnID:    appendResult.TurnExecution.TurnID,
		RootDir:   rootDir,
	}, nil
}

func waitingSessionExecutionTriggerAppendRequest(seedTime time.Time) artifacts.SessionExecutionTriggerAppendRequest {
	return artifacts.SessionExecutionTriggerAppendRequest{
		SessionID:                   "sess-manual-multiwait",
		WorkspaceID:                 "workspace-local",
		AuthoritativeRepositoryRoot: "/workspace/repo",
		TriggerSource:               "interactive_user",
		RequestedOperation:          "start",
		WorkflowRouting: artifacts.SessionWorkflowPackRoutingDurableState{
			WorkflowFamily:    "runecontext",
			WorkflowOperation: "approved_change_implementation",
		},
		ExecutionState:         "waiting",
		WaitKind:               "approval",
		WaitState:              "awaiting_review",
		BlockedReasonCode:      "approval_wait",
		UserMessageContentText: "WAITING session=sess-manual-multiwait",
		OccurredAt:             seedTime,
	}
}
