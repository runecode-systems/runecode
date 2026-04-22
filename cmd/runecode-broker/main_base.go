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

type brokerServiceFactoryFunc func(brokerServiceRoots) (*brokerapi.Service, error)

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
	roots, commandArgs, err := parseBrokerGlobalArgs(args)
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
	service, err := brokerServiceFactory(roots)
	if err != nil {
		return fmt.Errorf("runecode-broker failed to initialize store: %w", err)
	}
	return handler(commandArgs[1:], service, stdout)
}

var brokerServiceFactory brokerServiceFactoryFunc = newBrokerService
var localIPCListen = brokerapi.ListenLocalIPC

type commandHandler func([]string, *brokerapi.Service, io.Writer) error

func commandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"serve-local":                             handleServeLocal,
		"run-list":                                handleRunList,
		"run-get":                                 handleRunGet,
		"run-watch":                               handleRunWatch,
		"backend-posture-get":                     handleBackendPostureGet,
		"backend-posture-change":                  handleBackendPostureChange,
		"session-list":                            handleSessionList,
		"session-get":                             handleSessionGet,
		"session-send-message":                    handleSessionSendMessage,
		"session-watch":                           handleSessionWatch,
		"approval-list":                           handleApprovalList,
		"approval-get":                            handleApprovalGet,
		"approval-resolve":                        handleApprovalResolve,
		"approval-watch":                          handleApprovalWatch,
		"list-artifacts":                          handleListArtifacts,
		"head-artifact":                           handleHeadArtifact,
		"get-artifact":                            handleGetArtifact,
		"put-artifact":                            handlePutArtifact,
		"check-flow":                              handleCheckFlow,
		"promote-excerpt":                         handlePromoteExcerpt,
		"revoke-approved-excerpt":                 handleRevokeApprovedExcerpt,
		"set-run-status":                          handleSetRunStatus,
		"gc":                                      handleGC,
		"export-backup":                           handleExportBackup,
		"restore-backup":                          handleRestoreBackup,
		"show-audit":                              handleShowAudit,
		"show-policy":                             handleShowPolicy,
		"set-reserved-classes":                    handleSetReservedClasses,
		"import-trusted-contract":                 handleImportTrustedContract,
		"seed-dev-manual-scenario":                handleSeedDevManualScenario,
		"audit-readiness":                         handleAuditReadiness,
		"audit-verification":                      handleAuditVerification,
		"audit-finalize-verify":                   handleAuditFinalizeVerify,
		"audit-record-get":                        handleAuditRecordGet,
		"audit-anchor-segment":                    handleAuditAnchorSegment,
		"git-setup-get":                           handleGitSetupGet,
		"git-setup-auth-bootstrap":                handleGitSetupAuthBootstrap,
		"git-setup-identity-upsert":               handleGitSetupIdentityUpsert,
		"provider-setup-direct":                   handleProviderSetupDirect,
		"provider-profile-list":                   handleProviderProfileList,
		"provider-profile-get":                    handleProviderProfileGet,
		"project-substrate-get":                   handleProjectSubstrateGet,
		"project-substrate-posture-get":           handleProjectSubstratePostureGet,
		"project-substrate-adopt":                 handleProjectSubstrateAdopt,
		"project-substrate-init-preview":          handleProjectSubstrateInitPreview,
		"project-substrate-init-apply":            handleProjectSubstrateInitApply,
		"project-substrate-upgrade-preview":       handleProjectSubstrateUpgradePreview,
		"project-substrate-upgrade-apply":         handleProjectSubstrateUpgradeApply,
		"git-remote-mutation-prepare":             handleGitRemoteMutationPrepare,
		"git-remote-mutation-get":                 handleGitRemoteMutationGet,
		"git-remote-mutation-issue-execute-lease": handleGitRemoteMutationIssueExecuteLease,
		"git-remote-mutation-execute":             handleGitRemoteMutationExecute,
		"version-info":                            handleVersionInfo,
		"stream-logs":                             handleStreamLogs,
		"llm-invoke":                              handleLLMInvoke,
		"llm-stream":                              handleLLMStream,
	}
}

func parseBrokerGlobalArgs(args []string) (brokerServiceRoots, []string, error) {
	roots := defaultBrokerServiceRoots()
	if len(args) == 0 {
		return roots, nil, nil
	}
	if isHelpArg(args[0]) {
		return roots, args, nil
	}
	fs := flag.NewFlagSet("runecode-broker", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateRoot := fs.String("state-root", roots.stateRoot, "broker state root")
	auditLedgerRoot := fs.String("audit-ledger-root", roots.auditLedgerRoot, "audit ledger root")
	if err := fs.Parse(args); err != nil {
		return brokerServiceRoots{}, nil, &usageError{message: "usage: runecode-broker [--state-root path] [--audit-ledger-root path] <command> [flags]"}
	}
	roots.stateRoot = *stateRoot
	roots.auditLedgerRoot = *auditLedgerRoot
	return roots, fs.Args(), nil
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

const brokerHelpText = `Usage: runecode-broker [--state-root path] [--audit-ledger-root path] <command> [flags]

Global options:
  --state-root path         broker state root (artifact store and broker-owned local state)
  --audit-ledger-root path  audit ledger root

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
  session-watch [--stream-id id] [--session-id id] [--workspace-id id] [--status active|completed|archived] [--last-activity-kind kind] [--follow] [--include-snapshot]
  approval-list [--run-id id] [--status pending|approved|denied|expired|cancelled|superseded|consumed] [--limit N]
  approval-get --approval-id sha256:...
  approval-resolve --approval-request approval-request.json --approval-envelope approval.json [--approval-id sha256:...]
  approval-watch [--stream-id id] [--approval-id sha256:...] [--run-id id] [--workspace-id id] [--status pending|approved|denied|expired|cancelled|superseded|consumed] [--follow] [--include-snapshot]
  list-artifacts
  head-artifact --digest sha256:...
  get-artifact --digest sha256:... --producer role --consumer role [--manifest-opt-in] [--data-class class] --out path
  put-artifact --file path --content-type type --data-class class --provenance-hash sha256:...
  check-flow --producer role --consumer role --data-class class --digest sha256:... [--egress] [--manifest-opt-in]
  promote-excerpt --unapproved-digest sha256:... --approver user --approval-request approval-request.json --approval-envelope approval.json --repo-path path --commit hash --extractor-version v1 --full-content-visible
  revoke-approved-excerpt --digest sha256:... --actor user
  set-run-status --run-id id --status active|retained|closed
  gc
  export-backup --path backup.json
  restore-backup --path backup.json
  show-audit
  show-policy
  set-reserved-classes --enabled=true|false
  import-trusted-contract --kind verifier-record --file verifier.json --evidence import-evidence.json
  seed-dev-manual-scenario --dev-only [--profile tui-rich-v1] (requires dev-seed build tag)
	  audit-readiness
	  audit-verification [--limit N]
	  audit-finalize-verify
	  audit-record-get --record-digest sha256:...
	  audit-anchor-segment --seal-digest sha256:... [--approval-decision-digest sha256:...] [--approval-assurance-level level] [--export-receipt-copy]
	  git-setup-get [--provider github]
	  git-setup-auth-bootstrap [--provider github] --mode browser|device_code
	  git-setup-identity-upsert [--provider github] --profile-id id --display-name name --author-name name --author-email mail --committer-name name --committer-email mail --signoff-name name --signoff-email mail [--default-profile]
	  provider-setup-direct --provider-family family --canonical-host host [--canonical-path-prefix /v1] [--display-label label] [--adapter-kind kind] [--allowlisted-model-ids id1,id2]
	  provider-profile-list
	  provider-profile-get --provider-profile-id id
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
