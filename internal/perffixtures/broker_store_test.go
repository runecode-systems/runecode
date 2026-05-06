package perffixtures

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestBuildBrokerStoreFixtureEmptyAndWaiting(t *testing.T) {
	t.Run(FixtureTUIEmptyV1, testEmptyBrokerStoreFixture)
	t.Run(FixtureTUIWaitingV1, testWaitingBrokerStoreFixture)
}

func testEmptyBrokerStoreFixture(t *testing.T) {
	t.Helper()
	store, _, err := buildFixtureStore(t, FixtureTUIEmptyV1)
	if err != nil {
		t.Fatalf("buildFixtureStore returned error: %v", err)
	}
	if states := store.SessionDurableStates(); len(states) != 0 {
		t.Fatalf("empty fixture sessions = %d, want 0", len(states))
	}
}

func testWaitingBrokerStoreFixture(t *testing.T) {
	t.Helper()
	store, result, err := buildFixtureStore(t, FixtureTUIWaitingV1)
	if err != nil {
		t.Fatalf("buildFixtureStore returned error: %v", err)
	}
	state, ok := store.SessionState(result.SessionID)
	if !ok {
		t.Fatalf("SessionState(%q) ok=false", result.SessionID)
	}
	if state.WorkPosture != "waiting" {
		t.Fatalf("work_posture = %q, want waiting", state.WorkPosture)
	}
	if len(state.TurnExecutions) != 1 {
		t.Fatalf("turn_executions len = %d, want 1", len(state.TurnExecutions))
	}
	if got := state.TurnExecutions[0].ExecutionState; got != "waiting" {
		t.Fatalf("execution_state = %q, want waiting", got)
	}
}

func buildFixtureStore(t *testing.T, fixtureID string) (*artifacts.Store, BrokerStoreFixtureResult, error) {
	t.Helper()
	root := t.TempDir()
	result, err := BuildBrokerStoreFixture(root, fixtureID)
	if err != nil {
		return nil, BrokerStoreFixtureResult{}, err
	}
	store, err := artifacts.NewStore(root)
	if err != nil {
		return nil, BrokerStoreFixtureResult{}, err
	}
	return store, result, nil
}

func TestBuildBrokerStoreFixtureRejectsUnknown(t *testing.T) {
	if _, err := BuildBrokerStoreFixture(t.TempDir(), "unknown"); err == nil {
		t.Fatal("BuildBrokerStoreFixture error = nil, want unsupported fixture")
	}
}
