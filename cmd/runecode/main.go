// Command runecode provides canonical product lifecycle flows.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/localbootstrap"
)

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

const localAPISchemaVersion = "0.1.0"

var (
	errNoLiveBroker             = errors.New("no live broker reachable")
	errBrokerProcessUnreachable = errors.New("repo-scoped broker process is alive but unreachable")

	resolveRepoScope = func() (localbootstrap.RepoScope, error) {
		return localbootstrap.ResolveRepoScope(localbootstrap.ResolveInput{})
	}

	resolveRepoBrokerProcess = queryRepoBrokerProcess

	queryProductLifecyclePosture = func(ctx context.Context, cfg brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
		posture, _, err := resolveRepoBrokerProcess(ctx, cfg)
		return posture, err
	}

	queryProjectSubstratePosture = queryRepoProjectSubstratePosture

	startRepoBroker    = startRepoBrokerProcess
	launchTUI          = launchTUIProcess
	stopRepoBroker     = stopRepoBrokerProcess
	writeBrokerPID     = writePIDFile
	brokerProcessAlive = processIsAlive

	interruptBrokerProcess = func(pid int) error {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return proc.Signal(os.Interrupt)
	}

	killBrokerProcess = func(pid int) error {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return proc.Kill()
	}

	cleanupStartedBrokerProcess = func(cmd *exec.Cmd) error {
		if cmd == nil || cmd.Process == nil {
			return nil
		}
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
		_ = cmd.Wait()
		return nil
	}

	windowsTasklistOutput = func(pid int) (string, error) {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
		out, err := cmd.Output()
		return string(out), err
	}
)

func queryRepoBrokerProcess(ctx context.Context, cfg brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, int, error) {
	client, err := brokerapi.DialLocalRPC(ctx, cfg)
	if err != nil {
		return brokerapi.BrokerProductLifecyclePosture{}, 0, errNoLiveBroker
	}
	defer client.Close()
	peer, err := client.PeerCredentials()
	if err != nil {
		return brokerapi.BrokerProductLifecyclePosture{}, 0, err
	}
	resp := brokerapi.ProductLifecyclePostureGetResponse{}
	errResp := client.Invoke(ctx, "product_lifecycle_posture_get", brokerapi.ProductLifecyclePostureGetRequest{
		SchemaID:      "runecode.protocol.v0.ProductLifecyclePostureGetRequest",
		SchemaVersion: localAPISchemaVersion,
		RequestID:     "runecode-cli-product-lifecycle-posture",
	}, &resp)
	if errResp != nil {
		return brokerapi.BrokerProductLifecyclePosture{}, 0, fmt.Errorf("%s: %s", errResp.Error.Code, strings.TrimSpace(errResp.Error.Message))
	}
	return resp.ProductLifecycle, peer.PID, nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		var u *usageError
		if errors.As(err, &u) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, _ io.Writer) error {
	if len(args) == 0 {
		return runAttach(stdout)
	}
	if isHelpArg(args[0]) {
		return writeHelp(stdout)
	}
	if len(args) > 1 {
		return &usageError{message: "usage: runecode [attach|start|status|stop|restart]"}
	}
	switch strings.TrimSpace(args[0]) {
	case "attach":
		return runAttach(stdout)
	case "start":
		return runStart(stdout)
	case "status":
		return runStatus(stdout)
	case "stop":
		return runStop(stdout)
	case "restart":
		return runRestart(stdout)
	default:
		return &usageError{message: fmt.Sprintf("unknown command %q", args[0])}
	}
}

func runAttach(stdout io.Writer) error {
	scope, posture, err := ensureRepoLifecycle()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "attach mode=%s posture=%s instance=%s\n", sanitizeCLIField(posture.AttachMode), sanitizeCLIField(posture.LifecyclePosture), sanitizeCLIField(posture.ProductInstanceID)); err != nil {
		return err
	}
	return launchTUI(scope)
}

func runStart(stdout io.Writer) error {
	_, posture, err := ensureRepoLifecycle()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "runecode started mode=%s posture=%s instance=%s\n", sanitizeCLIField(posture.AttachMode), sanitizeCLIField(posture.LifecyclePosture), sanitizeCLIField(posture.ProductInstanceID)); err != nil {
		return err
	}
	return nil
}

func runStatus(stdout io.Writer) error {
	scope, err := resolveRepoScope()
	if err != nil {
		return err
	}
	cfg := brokerapi.LocalIPCConfig{RuntimeDir: scope.LocalRuntimeDir, SocketName: scope.LocalSocketName, RepositoryRoot: scope.RepositoryRoot}
	posture, err := queryProductLifecyclePosture(context.Background(), cfg)
	if err != nil {
		if errors.Is(err, errNoLiveBroker) {
			_, writeErr := fmt.Fprintln(stdout, "runecode status: no live product instance reachable")
			return writeErr
		}
		return err
	}
	if err := validatePostureIdentity(scope, posture); err != nil {
		return err
	}
	projectSubstrate, err := queryProjectSubstratePosture(context.Background(), cfg)
	if err != nil {
		if errors.Is(err, errNoLiveBroker) {
			_, writeErr := fmt.Fprintln(stdout, "runecode status: no live product instance reachable")
			return writeErr
		}
		return err
	}
	if err := validateProjectSubstrateIdentity(scope, projectSubstrate); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "runecode status: instance=%s generation=%s mode=%s posture=%s attachable=%t normal_operation_allowed=%t blocked_reasons=%s degraded_reasons=%s project_substrate_posture=%s project_substrate_validation_state=%s project_substrate_normal_operation_allowed=%t project_substrate_blocked_reasons=%s project_substrate_remediation_guidance=%s\n",
		sanitizeCLIField(posture.ProductInstanceID),
		sanitizeCLIField(posture.LifecycleGeneration),
		sanitizeCLIField(posture.AttachMode),
		sanitizeCLIField(posture.LifecyclePosture),
		posture.Attachable,
		posture.NormalOperationAllowed,
		joinCSV(posture.BlockedReasonCodes),
		joinCSV(posture.DegradedReasonCodes),
		sanitizeCLIField(projectSubstrate.PostureSummary.CompatibilityPosture),
		sanitizeCLIField(projectSubstrate.PostureSummary.ValidationState),
		projectSubstrate.PostureSummary.NormalOperationAllowed,
		joinCSV(projectSubstrate.PostureSummary.BlockedReasonCodes),
		joinCSV(projectSubstrate.RemediationGuidance),
	)
	return err
}

func runStop(stdout io.Writer) error {
	scope, err := resolveRepoScope()
	if err != nil {
		return err
	}
	if err := stopRepoBroker(scope); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "runecode stopped instance=%s\n", scope.ProductInstance)
	return err
}

func runRestart(stdout io.Writer) error {
	scope, err := resolveRepoScope()
	if err != nil {
		return err
	}
	if err := stopRepoBroker(scope); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_, posture, err := ensureRepoLifecycle()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "runecode restarted mode=%s posture=%s instance=%s\n", sanitizeCLIField(posture.AttachMode), sanitizeCLIField(posture.LifecyclePosture), sanitizeCLIField(posture.ProductInstanceID))
	return err
}

func ensureRepoLifecycle() (localbootstrap.RepoScope, brokerapi.BrokerProductLifecyclePosture, error) {
	scope, err := resolveRepoScope()
	if err != nil {
		return localbootstrap.RepoScope{}, brokerapi.BrokerProductLifecyclePosture{}, err
	}
	cfg := brokerapi.LocalIPCConfig{RuntimeDir: scope.LocalRuntimeDir, SocketName: scope.LocalSocketName, RepositoryRoot: scope.RepositoryRoot}
	posture, err := currentRepoLifecycle(scope, cfg)
	if err == nil {
		return scope, posture, nil
	}
	if !errors.Is(err, errNoLiveBroker) {
		return localbootstrap.RepoScope{}, brokerapi.BrokerProductLifecyclePosture{}, err
	}
	posture, reachable, err := recoverStaleRepoRuntimeArtifacts(scope, cfg)
	if err != nil {
		return localbootstrap.RepoScope{}, brokerapi.BrokerProductLifecyclePosture{}, err
	}
	if reachable {
		return scope, posture, nil
	}
	if err := startRepoBroker(scope); err != nil {
		return localbootstrap.RepoScope{}, brokerapi.BrokerProductLifecyclePosture{}, err
	}
	posture, err = waitForAttachableRepoLifecycle(scope, cfg, 6*time.Second)
	if err != nil {
		return localbootstrap.RepoScope{}, brokerapi.BrokerProductLifecyclePosture{}, err
	}
	return scope, posture, nil
}

func currentRepoLifecycle(scope localbootstrap.RepoScope, cfg brokerapi.LocalIPCConfig) (brokerapi.BrokerProductLifecyclePosture, error) {
	posture, err := queryProductLifecyclePosture(context.Background(), cfg)
	if err != nil {
		return brokerapi.BrokerProductLifecyclePosture{}, err
	}
	if err := validatePostureIdentity(scope, posture); err != nil {
		return brokerapi.BrokerProductLifecyclePosture{}, err
	}
	return posture, nil
}

func waitForAttachableRepoLifecycle(scope localbootstrap.RepoScope, cfg brokerapi.LocalIPCConfig, timeout time.Duration) (brokerapi.BrokerProductLifecyclePosture, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		posture, err := currentRepoLifecycle(scope, cfg)
		if err != nil {
			time.Sleep(150 * time.Millisecond)
			continue
		}
		if !posture.Attachable {
			return brokerapi.BrokerProductLifecyclePosture{}, fmt.Errorf("broker lifecycle posture is not attachable")
		}
		return posture, nil
	}
	return brokerapi.BrokerProductLifecyclePosture{}, fmt.Errorf("timed out waiting for repo-scoped broker to become attachable")
}

func validatePostureIdentity(scope localbootstrap.RepoScope, posture brokerapi.BrokerProductLifecyclePosture) error {
	if comparableRepoRoot(posture.RepositoryRoot) != comparableRepoRoot(scope.RepositoryRoot) {
		return fmt.Errorf("reachable broker repository root does not match authoritative repository root")
	}
	if strings.TrimSpace(posture.ProductInstanceID) != strings.TrimSpace(scope.ProductInstance) {
		return fmt.Errorf("reachable broker product instance does not match authoritative repository scope")
	}
	return nil
}

func startRepoBrokerProcess(scope localbootstrap.RepoScope) error {
	binary, err := findSiblingOrPathExecutable("runecode-broker")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(scope.LocalRuntimeDir, 0o700); err != nil {
		return err
	}
	cmd := exec.Command(binary, "serve-local", "--runtime-dir", scope.LocalRuntimeDir, "--socket-name", scope.LocalSocketName)
	cmd.Dir = scope.RepositoryRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := writeBrokerPID(scope, cmd.Process.Pid); err != nil {
		if cleanupErr := cleanupStartedBrokerProcess(cmd); cleanupErr != nil {
			return fmt.Errorf("persist broker pid: %w; cleanup failed: %v", err, cleanupErr)
		}
		return fmt.Errorf("persist broker pid: %w", err)
	}
	return nil
}

func launchTUIProcess(scope localbootstrap.RepoScope) error {
	binary, err := findSiblingOrPathExecutable("runecode-tui")
	if err != nil {
		return err
	}
	cmd := exec.Command(binary, "--runtime-dir", scope.LocalRuntimeDir, "--socket-name", scope.LocalSocketName)
	cmd.Dir = scope.RepositoryRoot
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopRepoBrokerProcess(scope localbootstrap.RepoScope) error {
	cfg := brokerapi.LocalIPCConfig{RuntimeDir: scope.LocalRuntimeDir, SocketName: scope.LocalSocketName, RepositoryRoot: scope.RepositoryRoot}
	posture, pid, err := resolveRepoBrokerProcess(context.Background(), cfg)
	if err != nil {
		if errors.Is(err, errNoLiveBroker) {
			return os.ErrNotExist
		}
		return err
	}
	if err := validatePostureIdentity(scope, posture); err != nil {
		return err
	}
	if pid <= 0 {
		return fmt.Errorf("live broker peer pid unavailable")
	}
	if err := interruptBrokerProcess(pid); err != nil {
		return err
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		_, probeErr := queryProductLifecyclePosture(context.Background(), cfg)
		if errors.Is(probeErr, errNoLiveBroker) {
			_ = os.Remove(pidFilePath(scope))
			return nil
		}
		time.Sleep(120 * time.Millisecond)
	}
	if err := killBrokerProcess(pid); err != nil {
		return err
	}
	if _, probeErr := queryProductLifecyclePosture(context.Background(), cfg); !errors.Is(probeErr, errNoLiveBroker) {
		if probeErr == nil {
			return fmt.Errorf("broker remained reachable after forced stop")
		}
		return probeErr
	}
	_ = os.Remove(pidFilePath(scope))
	return nil
}

func writePIDFile(scope localbootstrap.RepoScope, pid int) error {
	return os.WriteFile(pidFilePath(scope), []byte(strconv.Itoa(pid)), 0o600)
}

func pidFilePath(scope localbootstrap.RepoScope) string {
	return filepath.Join(scope.LocalRuntimeDir, "broker.pid")
}

func findSiblingOrPathExecutable(name string) (string, error) {
	exe, err := os.Executable()
	if err == nil {
		sibling := filepath.Join(filepath.Dir(exe), name)
		if info, statErr := os.Stat(sibling); statErr == nil && !info.IsDir() {
			return sibling, nil
		}
	}
	path, lookErr := exec.LookPath(name)
	if lookErr != nil {
		return "", fmt.Errorf("locate %s: %w", name, lookErr)
	}
	return path, nil
}

func isHelpArg(arg string) bool {
	trimmed := strings.TrimSpace(arg)
	return trimmed == "-h" || trimmed == "--help" || trimmed == "help"
}

func joinCSV(values []string) string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = sanitizeCLIField(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	if len(trimmed) == 0 {
		return "none"
	}
	return strings.Join(trimmed, ",")
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode [attach|start|status|stop|restart]

Canonical RuneCode product command:
  runecode          same as runecode attach
  runecode attach   ensure repo-scoped broker lifecycle and open TUI
  runecode start    ensure repo-scoped broker lifecycle without opening TUI
  runecode status   non-starting lifecycle status for current repo scope
  runecode stop     stop repo-scoped local broker lifecycle
  runecode restart  stop then start repo-scoped local broker lifecycle`)
	return err
}
