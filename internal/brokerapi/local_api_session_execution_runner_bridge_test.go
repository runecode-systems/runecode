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

func TestResolveRunnerLaunchRootAcceptsDirectRunnerRootOverride(t *testing.T) {
	repoRoot := repositoryRootForProjectSubstrateTests(t)
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	s.projectSubstrate.RepositoryRoot = ""
	s.apiConfig.RepositoryRoot = ""
	runnerRoot, err := resolveRunnerLaunchRoot(s, sessionExecutionRunnerLaunchSpec{runnerRoot: filepath.Join(repoRoot, "runner")})
	if err != nil {
		t.Fatalf("resolveRunnerLaunchRoot returned error: %v", err)
	}
	if want := filepath.Join(repoRoot, "runner"); runnerRoot != want {
		t.Fatalf("runnerRoot = %q, want %q", runnerRoot, want)
	}
}

func TestResolveRunnerLaunchRootAcceptsRepositoryRootOverride(t *testing.T) {
	repoRoot := repositoryRootForProjectSubstrateTests(t)
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	s.projectSubstrate.RepositoryRoot = ""
	s.apiConfig.RepositoryRoot = ""
	runnerRoot, err := resolveRunnerLaunchRoot(s, sessionExecutionRunnerLaunchSpec{runnerRoot: repoRoot})
	if err != nil {
		t.Fatalf("resolveRunnerLaunchRoot returned error: %v", err)
	}
	if want := filepath.Join(repoRoot, "runner"); runnerRoot != want {
		t.Fatalf("runnerRoot = %q, want %q", runnerRoot, want)
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
	summary := summarizeRunnerStderr(raw, false)
	if len(summary) > 512 {
		t.Fatalf("summary len = %d, want <= 512", len(summary))
	}
	if strings.Contains(summary, "\n") {
		t.Fatalf("summary = %q, want newlines normalized", summary)
	}
}

func TestCaptureSessionExecutionRunnerStderrBoundsBufferedSize(t *testing.T) {
	stderr := strings.NewReader(strings.Repeat("runner stderr line\n", 2000))
	capture, done := captureSessionExecutionRunnerStderr(stderr)
	<-done
	if got := len(capture.String()); got != sessionExecutionRunnerStderrCaptureLimit {
		t.Fatalf("captured stderr len = %d, want %d", got, sessionExecutionRunnerStderrCaptureLimit)
	}
	if !capture.Truncated() {
		t.Fatal("capture.Truncated() = false, want true")
	}
	if capture.String() == "" {
		t.Fatal("captured stderr unexpectedly empty")
	}
}

func TestSummarizeRunnerStderrAppendsTruncationSuffixDeterministically(t *testing.T) {
	raw := "  first line\nsecond line\r\n"
	summary := summarizeRunnerStderr(raw, true)
	want := "first line | second line [truncated]"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
	if len(summary) > sessionExecutionRunnerStderrSummaryLimit {
		t.Fatalf("summary len = %d, want <= %d", len(summary), sessionExecutionRunnerStderrSummaryLimit)
	}
}

func TestSessionExecutionRunnerEnvSanitizesInheritedEnvironment(t *testing.T) {
	t.Setenv("PATH", "/custom/bin")
	t.Setenv("HOME", "/tmp/home")
	t.Setenv("TMPDIR", "/tmp/runtime")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	env := sessionExecutionRunnerEnv("/repo/protocol/schemas", "/isolated/state-root")
	joined := strings.Join(env, "\n")
	for _, want := range []string{
		"PATH=/custom/bin",
		"HOME=/isolated/state-root",
		"TMPDIR=/isolated/state-root",
		"TEMP=/isolated/state-root",
		"TMP=/isolated/state-root",
		"RUNECODE_PROTOCOL_SCHEMAS_ROOT=/repo/protocol/schemas",
		"LANG=C",
		"LC_ALL=C",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("sessionExecutionRunnerEnv missing %q in %q", want, joined)
		}
	}
	if strings.Contains(joined, "AWS_SECRET_ACCESS_KEY=") {
		t.Fatalf("sessionExecutionRunnerEnv unexpectedly leaked secret env: %q", joined)
	}
}

func TestSessionExecutionRunnerBaseEnvWindowsKeepsRequiredMinimalVariables(t *testing.T) {
	lookup := func(key string) (string, bool) {
		values := map[string]string{
			"Path":         `C:\\node;C:\\Windows\\System32`,
			"USERPROFILE":  `C:\\Users\\runner`,
			"TEMP":         `C:\\Temp`,
			"LOCALAPPDATA": `C:\\Users\\runner\\AppData\\Local`,
			"SYSTEMROOT":   `C:\\Windows`,
			"COMSPEC":      `C:\\Windows\\System32\\cmd.exe`,
			"PATHEXT":      `.COM;.EXE;.BAT;.CMD`,
			"SECRET_TOKEN": "should-not-leak",
		}
		value, ok := values[key]
		return value, ok
	}
	env := sessionExecutionRunnerBaseEnv("windows", `C:\isolated\runner-state`, lookup)
	joined := strings.Join(env, "\n")
	for _, want := range []string{
		`PATH=C:\\node;C:\\Windows\\System32`,
		`HOME=C:\isolated\runner-state`,
		`USERPROFILE=C:\isolated\runner-state`,
		`TEMP=C:\isolated\runner-state`,
		`TMP=C:\isolated\runner-state`,
		`APPDATA=C:\isolated\runner-state\AppData\Roaming`,
		`LOCALAPPDATA=C:\isolated\runner-state\AppData\Local`,
		`SystemRoot=C:\\Windows`,
		`ComSpec=C:\\Windows\\System32\\cmd.exe`,
		`PATHEXT=.COM;.EXE;.BAT;.CMD`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("sessionExecutionRunnerBaseEnv missing %q in %q", want, joined)
		}
	}
	if strings.Contains(joined, "SECRET_TOKEN=") {
		t.Fatalf("sessionExecutionRunnerBaseEnv unexpectedly leaked secret env: %q", joined)
	}
}
