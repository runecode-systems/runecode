package brokerapi

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	sessionExecutionRunnerStderrCaptureLimit  = 8 * 1024
	sessionExecutionRunnerStderrSummaryLimit  = 512
	sessionExecutionRunnerStderrSummarySuffix = " [truncated]"
)

func launchSessionExecutionRunnerSubprocess(ctx context.Context, s *Service, spec sessionExecutionRunnerLaunchSpec) error {
	if ctx == nil {
		return fmt.Errorf("runner subprocess context is required")
	}
	if err := sessionExecutionRunnerRequestLifecycleErr(ctx); err != nil {
		return err
	}
	prepared, err := prepareSessionExecutionRunnerLaunch(s, spec)
	if err != nil {
		return err
	}
	defer os.RemoveAll(prepared.stateRoot)
	runnerCtx, stopRunner := sessionExecutionRunnerSubprocessContext(ctx)
	defer stopRunner(nil)
	cmd := exec.CommandContext(runnerCtx, prepared.command[0], prepared.command[1:]...)
	cmd.Dir = prepared.runnerRoot
	cmd.Env = prepared.env
	stdin, stdout, stderr, err := openSessionExecutionRunnerPipes(cmd)
	if err != nil {
		return err
	}
	if err := sessionExecutionRunnerRequestLifecycleErr(ctx); err != nil {
		_ = stdin.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch runner subprocess: %w", err)
	}
	stderrBytes, stderrDone := captureSessionExecutionRunnerStderr(stderr)
	return waitForSessionExecutionRunner(ctx, s, spec, runnerCtx, stopRunner, cmd, stdin, stdout, stderrBytes, stderrDone)
}

type preparedSessionExecutionRunnerLaunch struct {
	runnerRoot string
	stateRoot  string
	command    []string
	env        []string
}

func prepareSessionExecutionRunnerLaunch(s *Service, spec sessionExecutionRunnerLaunchSpec) (preparedSessionExecutionRunnerLaunch, error) {
	runnerRoot, err := resolveRunnerLaunchRoot(s, spec)
	if err != nil {
		return preparedSessionExecutionRunnerLaunch{}, err
	}
	if _, err := os.Stat(filepath.Join(runnerRoot, "package.json")); err != nil {
		return preparedSessionExecutionRunnerLaunch{}, fmt.Errorf("runner launch root missing package.json: %w", err)
	}
	if err := validateSessionExecutionRunnerInstall(runnerRoot); err != nil {
		return preparedSessionExecutionRunnerLaunch{}, err
	}
	stateRoot, err := os.MkdirTemp(filepath.Dir(spec.planPath), "runecode-runner-state-")
	if err != nil {
		return preparedSessionExecutionRunnerLaunch{}, fmt.Errorf("create runner state root: %w", err)
	}
	nodePath, err := resolveRunnerNodePath(s)
	if err != nil {
		return preparedSessionExecutionRunnerLaunch{}, err
	}
	command, err := sessionExecutionRunnerCommand(nodePath, runnerRoot, spec.planPath, stateRoot)
	if err != nil {
		return preparedSessionExecutionRunnerLaunch{}, err
	}
	return preparedSessionExecutionRunnerLaunch{
		runnerRoot: runnerRoot,
		stateRoot:  stateRoot,
		command:    command,
		env:        sessionExecutionRunnerEnv(filepath.Join(filepath.Dir(runnerRoot), "protocol", "schemas"), stateRoot),
	}, nil
}

func openSessionExecutionRunnerPipes(cmd *exec.Cmd) (io.WriteCloser, io.Reader, io.Reader, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open runner stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open runner stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open runner stderr: %w", err)
	}
	return stdin, stdout, stderr, nil
}

func captureSessionExecutionRunnerStderr(stderr io.Reader) (*boundedSessionExecutionRunnerStderrCapture, <-chan struct{}) {
	stderrBytes := newBoundedSessionExecutionRunnerStderrCapture(sessionExecutionRunnerStderrCaptureLimit)
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(stderrBytes, stderr)
		close(stderrDone)
	}()
	return stderrBytes, stderrDone
}

type boundedSessionExecutionRunnerStderrCapture struct {
	builder   strings.Builder
	remaining int
	truncated bool
}

func newBoundedSessionExecutionRunnerStderrCapture(limit int) *boundedSessionExecutionRunnerStderrCapture {
	return &boundedSessionExecutionRunnerStderrCapture{remaining: limit}
}

func (c *boundedSessionExecutionRunnerStderrCapture) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if c.remaining <= 0 {
		c.truncated = true
		return len(p), nil
	}
	keep := len(p)
	if keep > c.remaining {
		keep = c.remaining
		c.truncated = true
	}
	_, _ = c.builder.Write(p[:keep])
	c.remaining -= keep
	if keep < len(p) {
		c.truncated = true
	}
	return len(p), nil
}

func (c *boundedSessionExecutionRunnerStderrCapture) String() string {
	return c.builder.String()
}

func (c *boundedSessionExecutionRunnerStderrCapture) Truncated() bool {
	return c.truncated
}

func resolveRunnerLaunchRoot(s *Service, spec sessionExecutionRunnerLaunchSpec) (string, error) {
	overrideRoot := strings.TrimSpace(spec.runnerRoot)
	if overrideRoot == "" && strings.TrimSpace(s.projectSubstrate.RepositoryRoot) == "" && strings.TrimSpace(s.apiConfig.RepositoryRoot) == "" {
		return "", fmt.Errorf("resolve runner launch root: repository root is required")
	}
	if overrideRoot != "" {
		if runnerRoot, ok := firstRunnerRootCandidate(overrideRoot); ok {
			return runnerRoot, nil
		}
		if runnerRoot, ok := directRunnerRootCandidate(overrideRoot); ok {
			return runnerRoot, nil
		}
	}
	if runnerRoot, ok := firstRunnerRootCandidate(strings.TrimSpace(s.projectSubstrate.RepositoryRoot), strings.TrimSpace(s.apiConfig.RepositoryRoot)); ok {
		return runnerRoot, nil
	}
	return "", fmt.Errorf("resolve runner launch root: repository root missing runner/package.json")
}

func directRunnerRootCandidate(candidate string) (string, bool) {
	root := strings.TrimSpace(candidate)
	if root == "" {
		return "", false
	}
	clean := filepath.Clean(root)
	if _, err := os.Stat(filepath.Join(clean, "package.json")); err == nil {
		return clean, true
	}
	return "", false
}

func firstRunnerRootCandidate(candidates ...string) (string, bool) {
	for _, candidate := range candidates {
		root := strings.TrimSpace(candidate)
		if root == "" {
			continue
		}
		clean := filepath.Clean(root)
		runnerRoot := filepath.Join(clean, "runner")
		if _, err := os.Stat(filepath.Join(runnerRoot, "package.json")); err == nil {
			return runnerRoot, true
		}
	}
	return "", false
}

func validateSessionExecutionRunnerInstall(runnerRoot string) error {
	for _, dependency := range []string{"ajv", "ajv-formats"} {
		if _, err := os.Stat(filepath.Join(runnerRoot, "node_modules", dependency, "package.json")); err != nil {
			return fmt.Errorf("runner launch root missing installed runtime dependency %q; run (cd runner && npm ci): %w", dependency, err)
		}
	}
	return nil
}

func sessionExecutionRunnerCommand(nodePath, repoRoot, planPath, stateRoot string) ([]string, error) {
	cliPath := filepath.Join(repoRoot, "src", "cli.ts")
	planRoot := filepath.Dir(planPath)
	return []string{nodePath, "--experimental-strip-types", cliPath, "--plan-file", planPath, "--plan-root", planRoot, "--state-root", stateRoot, "--broker-transport", "stdio"}, nil
}

func resolveRunnerNodePath(s *Service) (string, error) {
	configured := strings.TrimSpace(s.apiConfig.RunnerNodePath)
	if configured != "" {
		if !filepath.IsAbs(configured) {
			return "", fmt.Errorf("configured runner node path must be absolute")
		}
		return configured, nil
	}
	resolved, err := exec.LookPath("node")
	if err != nil {
		return "", fmt.Errorf("resolve node runtime for runner launch: %w", err)
	}
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("resolved node runtime must be absolute")
	}
	return resolved, nil
}

func summarizeRunnerStderr(raw string, truncated bool) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.ReplaceAll(trimmed, "\n", " | ")
	trimmed = strings.ReplaceAll(trimmed, "\r", "")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		if truncated {
			return strings.TrimSpace(sessionExecutionRunnerStderrSummarySuffix)
		}
		return "none"
	}
	suffix := ""
	if truncated {
		suffix = sessionExecutionRunnerStderrSummarySuffix
	}
	limit := sessionExecutionRunnerStderrSummaryLimit - len(suffix)
	if limit < 0 {
		limit = 0
	}
	if len(trimmed) > limit {
		trimmed = trimmed[:limit]
	}
	return trimmed + suffix
}
