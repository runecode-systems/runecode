package perffixtures

import (
	"os"
	"testing"
)

func TestBuildWorkflowFixture(t *testing.T) {
	for _, fixtureID := range []string{FixtureWorkflowFirstPartyMinimal, FixtureWorkflowCHG050Compile} {
		fixtureID := fixtureID
		t.Run(fixtureID, func(t *testing.T) {
			result, err := BuildWorkflowFixture(t.TempDir(), fixtureID)
			if err != nil {
				t.Fatalf("BuildWorkflowFixture returned error: %v", err)
			}
			if _, err := os.Stat(result.RunPlan); err != nil {
				t.Fatalf("runplan missing: %v", err)
			}
			if _, err := os.Stat(result.Workspace); err != nil {
				t.Fatalf("workspace missing: %v", err)
			}
		})
	}
}
