// Command runecode-broker provides a local artifact and policy broker surface.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/localbootstrap"
)

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

type brokerServiceRoots struct {
	stateRoot       string
	auditLedgerRoot string
}

type brokerGlobalOptions struct {
	roots                 brokerServiceRoots
	stateRootOverridden   bool
	auditLedgerOverridden bool
	liveIPCTarget         brokerLiveIPCTargetOptions
}

type brokerLiveIPCTargetOptions struct {
	runtimeDir string
	socketName string
}

func (o brokerLiveIPCTargetOptions) overridden() bool {
	return o.runtimeDir != "" || o.socketName != ""
}

type brokerServiceFactoryFunc func(brokerServiceRoots) (*brokerapi.Service, error)

type brokerCommandAPIMode string

const (
	brokerCommandAPIModeInProcess brokerCommandAPIMode = "in_process"
	brokerCommandAPIModeLiveIPC   brokerCommandAPIMode = "live_ipc"
)

type brokerCommandSpec struct {
	handler       commandHandler
	requiresStore bool
	apiMode       brokerCommandAPIMode
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if _, errWrite := fmt.Fprintln(os.Stderr, err.Error()); errWrite != nil {
			os.Exit(1)
		}
		if _, ok := err.(*usageError); ok {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	globalOpts, commandArgs, err := parseBrokerGlobalArgs(args)
	if err != nil {
		return err
	}
	if len(commandArgs) == 0 || isHelpArg(commandArgs[0]) {
		return writeHelp(stdout)
	}
	handler, ok := commandHandlers()[commandArgs[0]]
	if !ok {
		_ = writeHelp(stderr)
		return &usageError{message: fmt.Sprintf("unknown command %q", commandArgs[0])}
	}
	return executeCommand(handler, globalOpts, commandArgs, stdout, stderr)
}

func executeCommand(handler brokerCommandSpec, globalOpts brokerGlobalOptions, commandArgs []string, stdout io.Writer, stderr io.Writer) error {
	commandName := commandArgs[0]
	resolvedMode := resolveBrokerCommandAPIMode(commandName, handler.apiMode)
	explicitLiveIPC, err := validateExplicitLiveIPCCommandTarget(commandName, resolvedMode, globalOpts)
	if err != nil {
		return err
	}
	needService := commandNeedsBrokerService(commandName, handler, globalOpts, resolvedMode)
	resolvedMode, needService, err = resolveBrokerCommandExecutionMode(commandName, resolvedMode, globalOpts, needService)
	if err != nil {
		return err
	}
	restoreMode, err := configureBrokerLocalAPIClientMode(resolvedMode, globalOpts.liveIPCTarget, explicitLiveIPC)
	if err != nil {
		return err
	}
	defer restoreMode()
	service, err := buildBrokerServiceIfNeeded(needService, globalOpts.roots)
	if err != nil {
		return err
	}
	return handler.handler(commandArgs[1:], service, stdout)
}

func validateExplicitLiveIPCCommandTarget(commandName string, resolvedMode brokerCommandAPIMode, globalOpts brokerGlobalOptions) (bool, error) {
	explicitLiveIPC := globalOpts.liveIPCTarget.overridden()
	if !explicitLiveIPC {
		return false, nil
	}
	if globalOpts.stateRootOverridden || globalOpts.auditLedgerOverridden {
		return false, &usageError{message: "--runtime-dir/--socket-name cannot be combined with --state-root/--audit-ledger-root"}
	}
	if resolvedMode != brokerCommandAPIModeLiveIPC {
		return false, &usageError{message: fmt.Sprintf("%s does not use live broker IPC; --runtime-dir/--socket-name are only supported for live IPC commands", commandName)}
	}
	return true, nil
}

func commandNeedsBrokerService(commandName string, handler brokerCommandSpec, globalOpts brokerGlobalOptions, resolvedMode brokerCommandAPIMode) bool {
	if commandName == "put-artifact" {
		return globalOpts.stateRootOverridden || globalOpts.auditLedgerOverridden
	}
	return handler.requiresStore || resolvedMode == brokerCommandAPIModeInProcess
}

func resolveBrokerCommandExecutionMode(commandName string, resolvedMode brokerCommandAPIMode, globalOpts brokerGlobalOptions, needService bool) (brokerCommandAPIMode, bool, error) {
	if resolvedMode != brokerCommandAPIModeLiveIPC || needService {
		return resolvedMode, needService, nil
	}
	if !globalOpts.stateRootOverridden && !globalOpts.auditLedgerOverridden {
		return resolvedMode, needService, nil
	}
	if !isArtifactReadLiveIPCCommand(commandName) {
		return resolvedMode, needService, &usageError{message: fmt.Sprintf("%s uses repo-scoped live broker IPC; --state-root and --audit-ledger-root are only supported for local in-process commands", commandName)}
	}
	return brokerCommandAPIModeInProcess, true, nil
}

func isArtifactReadLiveIPCCommand(commandName string) bool {
	switch commandName {
	case "list-artifacts", "head-artifact", "get-artifact":
		return true
	default:
		return false
	}
}

func configureBrokerLocalAPIClientMode(resolvedMode brokerCommandAPIMode, liveIPCTarget brokerLiveIPCTargetOptions, explicitLiveIPC bool) (func(), error) {
	if !explicitLiveIPC {
		return setLocalAPIClientSelection(resolvedMode, nil), nil
	}
	cfg, err := resolveExplicitLiveIPCTargetConfig(liveIPCTarget)
	if err != nil {
		return nil, err
	}
	return setLocalAPIClientSelection(resolvedMode, &cfg), nil
}

func buildBrokerServiceIfNeeded(needService bool, roots brokerServiceRoots) (*brokerapi.Service, error) {
	if !needService {
		return nil, nil
	}
	service, err := brokerServiceFactory(roots)
	if err != nil {
		return nil, fmt.Errorf("runecode-broker failed to initialize store: %w", err)
	}
	return service, nil
}

var brokerServiceFactory brokerServiceFactoryFunc = newBrokerService
var localIPCListen = brokerapi.ListenLocalIPC

type commandHandler func([]string, *brokerapi.Service, io.Writer) error

func commandHandlers() map[string]brokerCommandSpec {
	handlers := baseCommandHandlers()
	addLiveIPCCommandHandlers(handlers)
	return handlers
}

func resolveBrokerCommandAPIMode(command string, mode brokerCommandAPIMode) brokerCommandAPIMode {
	_ = command
	return mode
}

func parseBrokerGlobalArgs(args []string) (brokerGlobalOptions, []string, error) {
	defaults := defaultBrokerServiceRoots()
	globalOpts := brokerGlobalOptions{roots: defaults}
	if len(args) == 0 {
		return globalOpts, nil, nil
	}
	if isHelpArg(args[0]) {
		return globalOpts, args, nil
	}
	fs := flag.NewFlagSet("runecode-broker", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateRoot := fs.String("state-root", defaults.stateRoot, "broker state root")
	auditLedgerRoot := fs.String("audit-ledger-root", defaults.auditLedgerRoot, "audit ledger root")
	runtimeDir := fs.String("runtime-dir", "", "explicit runtime directory for live IPC commands")
	socketName := fs.String("socket-name", "", "explicit socket filename for live IPC commands")
	if err := fs.Parse(args); err != nil {
		return brokerGlobalOptions{}, nil, &usageError{message: "usage: runecode-broker [--state-root path] [--audit-ledger-root path] [--runtime-dir dir] [--socket-name broker.sock] <command> [flags]"}
	}
	globalOpts.roots.stateRoot = *stateRoot
	globalOpts.roots.auditLedgerRoot = *auditLedgerRoot
	globalOpts.liveIPCTarget.runtimeDir = *runtimeDir
	globalOpts.liveIPCTarget.socketName = *socketName
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "state-root":
			globalOpts.stateRootOverridden = true
		case "audit-ledger-root":
			globalOpts.auditLedgerOverridden = true
		}
	})
	return globalOpts, fs.Args(), nil
}

func resolveExplicitLiveIPCTargetConfig(target brokerLiveIPCTargetOptions) (brokerapi.LocalIPCConfig, error) {
	defaults, err := loadDefaultLocalIPCConfig()
	if err != nil {
		return brokerapi.LocalIPCConfig{}, err
	}
	cfg := brokerapi.LocalIPCConfig{
		RuntimeDir:     target.runtimeDir,
		SocketName:     target.socketName,
		RepositoryRoot: defaults.RepositoryRoot,
	}
	if cfg.RuntimeDir == "" {
		cfg.RuntimeDir = defaults.RuntimeDir
	}
	if cfg.SocketName == "" {
		cfg.SocketName = defaults.SocketName
	}
	return cfg, nil
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}

func parseServeLocalArgs(args []string) (brokerapi.LocalIPCConfig, bool, error) {
	defaults, err := brokerapi.DefaultLocalIPCConfig()
	if err != nil {
		return brokerapi.LocalIPCConfig{}, false, err
	}
	fs := flag.NewFlagSet("serve-local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runtimeDir := fs.String("runtime-dir", defaults.RuntimeDir, "runtime directory for local unix socket")
	socketName := fs.String("socket-name", defaults.SocketName, "socket filename")
	once := fs.Bool("once", false, "accept a single connection and exit")
	if err := fs.Parse(args); err != nil {
		return brokerapi.LocalIPCConfig{}, false, &usageError{message: "serve-local usage: runecode-broker serve-local [--runtime-dir dir] [--socket-name broker.sock] [--once]"}
	}
	return brokerapi.LocalIPCConfig{RuntimeDir: *runtimeDir, SocketName: *socketName}, *once, nil
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, brokerHelpText)
	return err
}

const brokerHelpText = `Usage: runecode-broker [--state-root path] [--audit-ledger-root path] [--runtime-dir dir] [--socket-name broker.sock] <command> [flags]

Global options:
  --state-root path         broker state root (artifact store and broker-owned local state)
  --audit-ledger-root path  audit ledger root
  --runtime-dir dir         explicit runtime directory for live IPC commands
  --socket-name name        explicit socket filename for live IPC commands

Commands:
  serve-local [--runtime-dir dir] [--socket-name broker.sock] [--once]
  run-list [--limit N]
  run-get --run-id id
  run-watch [--stream-id id] [--run-id id] [--workspace-id id] [--lifecycle-state state] [--follow] [--include-snapshot]
  backend-posture-get
  backend-posture-change --target-backend-kind microvm|container [--target-instance-id id] [--selection-mode explicit_selection|automatic_fallback_attempt] [--change-kind select_backend] [--assurance-change-kind reduce_assurance|maintain_assurance] [--opt-in-kind exact_action_approval|none] [--reduced-assurance-acknowledged] [--reason text]
  session-list [--limit N]
  session-get --session-id id
  session-send-message --session-id id --content text [--role user|assistant|system|tool] [--idempotency-key key]
  session-execution-trigger --session-id id [--turn-id id] [--trigger-source interactive_user|autonomous_background|resume_follow_up] [--requested-operation start|continue] [--workflow-family runecontext] [--workflow-operation change_draft|spec_draft|draft_promote_apply|approved_change_implementation] [--user-message text] [--idempotency-key key]
  session-watch [--stream-id id] [--session-id id] [--workspace-id id] [--status active|completed|archived] [--last-activity-kind kind] [--follow] [--include-snapshot]
  approval-list [--run-id id] [--status pending|approved|denied|expired|cancelled|superseded|consumed] [--limit N]
  approval-get --approval-id sha256:...
  approval-resolve --approval-request approval-request.json --approval-envelope approval.json [--approval-id sha256:...]
  approval-watch [--stream-id id] [--approval-id sha256:...] [--run-id id] [--workspace-id id] [--status pending|approved|denied|expired|cancelled|superseded|consumed] [--follow] [--include-snapshot]
  list-artifacts
  head-artifact --digest sha256:...
  get-artifact --digest sha256:... --producer role --consumer role [--manifest-opt-in] [--data-class class] --out path
  put-artifact --file path --content-type type --data-class class --provenance-hash sha256:... [--runtime-dir dir] [--socket-name broker.sock]
  check-flow --producer role --consumer role --data-class class --digest sha256:... [--egress] [--manifest-opt-in]
  promote-excerpt --unapproved-digest sha256:... --approver user --approval-request approval-request.json --approval-envelope approval.json --repo-path path --commit hash --extractor-version v1 --full-content-visible
  revoke-approved-excerpt --digest sha256:... --actor user
  set-run-status --run-id id --status active|retained|closed
  gc
  export-backup --path backup.json (artifact/broker state backup; trusted-context audit import links are not made portable by this command)
  restore-backup --path backup.json (restores artifact/broker state only; re-import trusted contracts/evidence in the target environment as needed)
  show-audit
  show-policy
  set-reserved-classes --enabled=true|false
	  import-trusted-contract --kind <kind> --file payload.json --evidence import-evidence.json
  seed-dev-manual-scenario --dev-only [--profile tui-rich-v1] (requires dev-seed build tag; seeds trusted context using the same import/audit semantics as other trusted policy artifacts)
	  audit-readiness
	  audit-verification [--limit N]
	  audit-finalize-verify
	  audit-record-get --record-digest sha256:...
	  audit-anchor-segment --seal-digest sha256:... [--approval-decision-digest sha256:...] [--approval-assurance-level level] [--export-receipt-copy]
	  zk-proof-generate --record-digest sha256:...
	  zk-proof-verify --proof-digest sha256:...
	  git-setup-get [--provider github]
	  git-setup-auth-bootstrap [--provider github] --mode browser|device_code
	  git-setup-identity-upsert [--provider github] --profile-id id --display-name name --author-name name --author-email mail --committer-name name --committer-email mail --signoff-name name --signoff-email mail [--default-profile]
	  provider-setup-direct --provider-family family --canonical-host host [--canonical-path-prefix /v1] [--display-label label] [--adapter-kind kind] [--allowlisted-model-ids id1,id2]
	  provider-credential-lease-issue --provider-profile-id id --run-id run-1 [--ttl-seconds 900]
	  provider-profile-list
	  provider-profile-get --provider-profile-id id
	  dependency-cache-ensure --request-file path
	  dependency-fetch-registry --request-file path
	  dependency-cache-handoff --request-file path
	  project-substrate-get
	  project-substrate-posture-get
	  project-substrate-adopt
	  project-substrate-init-preview
	  project-substrate-init-apply [--expected-preview-token sha256:...]
	  project-substrate-upgrade-preview
	  project-substrate-upgrade-apply --expected-preview-digest sha256:...
	  git-remote-mutation-prepare --request-file path
	  git-remote-mutation-get --request-file path
	  git-remote-mutation-issue-execute-lease --request-file path
	  git-remote-mutation-execute --request-file path
	  external-anchor-mutation-prepare --request-file path
	  external-anchor-mutation-get --request-file path
	  external-anchor-mutation-issue-execute-lease --request-file path
	  external-anchor-mutation-execute --request-file path
  version-info
  stream-logs [--stream-id id] [--run-id id] [--role-instance-id id] [--start-cursor cursor] [--follow] [--include-backlog]
  llm-invoke --run-id id --request-file path [--request-digest sha256:...]
  llm-stream --run-id id --request-file path [--request-digest sha256:...] [--stream-id id] [--follow]`

func defaultBrokerServiceRoots() brokerServiceRoots {
	scope, err := localbootstrap.ResolveRepoScope(localbootstrap.ResolveInput{})
	if err == nil {
		return brokerServiceRoots{stateRoot: scope.StateRoot, auditLedgerRoot: scope.AuditLedgerRoot}
	}
	return brokerServiceRoots{stateRoot: defaultBrokerStoreRoot(), auditLedgerRoot: auditd.DefaultLedgerRoot()}
}

func newBrokerService(roots brokerServiceRoots) (*brokerapi.Service, error) {
	scope, err := localbootstrap.ResolveRepoScope(localbootstrap.ResolveInput{})
	if err != nil {
		return nil, err
	}
	resolved := roots
	if resolved.stateRoot == "" {
		resolved.stateRoot = scope.StateRoot
	}
	if resolved.auditLedgerRoot == "" {
		resolved.auditLedgerRoot = scope.AuditLedgerRoot
	}
	return brokerapi.NewServiceWithConfig(resolved.stateRoot, resolved.auditLedgerRoot, brokerapi.APIConfig{RepositoryRoot: scope.RepositoryRoot})
}
