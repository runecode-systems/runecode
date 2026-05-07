package brokerapi

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionExecutionRunnerCommandUsesAbsoluteNodePathAndPlanRoot(t *testing.T) {
	command, err := sessionExecutionRunnerCommand("/usr/bin/node", "/repo/runner", "/private/runplan/root/runplan.json", "/private/runplan/root/state")
	if err != nil {
		t.Fatalf("sessionExecutionRunnerCommand returned error: %v", err)
	}
	joined := strings.Join(command, " ")
	for _, want := range []string{"/usr/bin/node", "--plan-file", "/private/runplan/root/runplan.json", "--plan-root", "/private/runplan/root", "--state-root", "/private/runplan/root/state", "--broker-transport", "stdio"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("sessionExecutionRunnerCommand = %v, want token %q", command, want)
		}
	}
}

func TestResolveRunnerNodePathRejectsRelativeConfiguredPath(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{RunnerNodePath: "node"})
	_, err := resolveRunnerNodePath(s)
	if err == nil {
		t.Fatal("resolveRunnerNodePath expected error for relative configured path")
	}
	if !strings.Contains(err.Error(), "must be absolute") {
		t.Fatalf("resolveRunnerNodePath error = %q, want absolute path detail", err)
	}
}

func TestResolveRunnerLaunchRootFailsClosedWithoutRepositoryRoot(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	s.projectSubstrate.RepositoryRoot = ""
	s.apiConfig.RepositoryRoot = ""
	_, err := resolveRunnerLaunchRoot(s, sessionExecutionRunnerLaunchSpec{})
	if err == nil {
		t.Fatal("resolveRunnerLaunchRoot expected error when repository root is missing")
	}
	if !strings.Contains(err.Error(), "repository root is required") {
		t.Fatalf("resolveRunnerLaunchRoot error = %q, want repository root required detail", err)
	}
}

func TestWriteSessionExecutionTemporaryFileUsesPrivateParentDirectory(t *testing.T) {
	parent := t.TempDir()
	path, err := writeSessionExecutionTemporaryFile(parent, "runplan-*.json", []byte("{}"))
	if err != nil {
		t.Fatalf("writeSessionExecutionTemporaryFile returned error: %v", err)
	}
	if filepath.Dir(path) != parent {
		t.Fatalf("temporary file dir = %q, want %q", filepath.Dir(path), parent)
	}
}

func TestSummarizeRunnerStderrTruncatesAndNormalizes(t *testing.T) {
	raw := strings.Repeat("x", 600) + "\nsecond-line"
	summary := summarizeRunnerStderr(raw)
	if len(summary) > 512 {
		t.Fatalf("summary len = %d, want <= 512", len(summary))
	}
	if strings.Contains(summary, "\n") {
		t.Fatalf("summary = %q, want newlines normalized", summary)
	}
}
