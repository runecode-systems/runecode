package perffixtures

import (
	"fmt"
	"os"
	"path/filepath"
)

const FixtureRunnerBoundaryMinimal = "runner.boundary.minimal.v1"

type RunnerFixtureResult struct {
	FixtureID        string
	RootDir          string
	WorkflowFilePath string
	WorkspaceDir     string
}

func BuildRunnerFixture(rootDir string, fixtureID string) (RunnerFixtureResult, error) {
	if fixtureID != FixtureRunnerBoundaryMinimal {
		return RunnerFixtureResult{}, fmt.Errorf("%w: %s", ErrUnsupportedFixtureID, fixtureID)
	}
	workspace := filepath.Join(rootDir, "runner-workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return RunnerFixtureResult{}, err
	}
	workflowPath := filepath.Join(rootDir, "workflow.json")
	workflow := `{"schema_id":"runecode.protocol.runner.workflow.v1","name":"minimal","steps":[{"id":"step-1","kind":"noop","inputs":{}}]}`
	if err := os.WriteFile(workflowPath, []byte(workflow), 0o644); err != nil {
		return RunnerFixtureResult{}, err
	}
	return RunnerFixtureResult{FixtureID: fixtureID, RootDir: rootDir, WorkflowFilePath: workflowPath, WorkspaceDir: workspace}, nil
}
