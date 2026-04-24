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
	resolvedMode := resolveBrokerCommandAPIMode(commandArgs[0], handler.apiMode)
	if resolvedMode == brokerCommandAPIModeLiveIPC && (globalOpts.stateRootOverridden || globalOpts.auditLedgerOverridden) {
		return &usageError{message: fmt.Sprintf("%s uses repo-scoped live broker IPC; --state-root and --audit-ledger-root are only supported for local in-process commands", commandArgs[0])}
	}
	restoreMode := setLocalAPIClientMode(resolvedMode)
	defer restoreMode()
	var service *brokerapi.Service
	if handler.requiresStore || resolvedMode == brokerCommandAPIModeInProcess {
		service, err = brokerServiceFactory(globalOpts.roots)
		if err != nil {
			return fmt.Errorf("runecode-broker failed to initialize store: %w", err)
		}
	}
	return handler.handler(commandArgs[1:], service, stdout)
}

var brokerServiceFactory brokerServiceFactoryFunc = newBrokerService
var localIPCListen = brokerapi.ListenLocalIPC

type commandHandler func([]string, *brokerapi.Service, io.Writer) error

func commandHandlers() map[string]brokerCommandSpec {
	handlers := map[string]brokerCommandSpec{
		"serve-local":               {handler: handleServeLocal, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"run-list":                  {handler: handleRunList, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"run-get":                   {handler: handleRunGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"run-watch":                 {handler: handleRunWatch, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"backend-posture-get":       {handler: handleBackendPostureGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"backend-posture-change":    {handler: handleBackendPostureChange, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"session-list":              {handler: handleSessionList, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"session-get":               {handler: handleSessionGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"session-send-message":      {handler: handleSessionSendMessage, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"session-execution-trigger": {handler: handleSessionExecutionTrigger, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"session-watch":             {handler: handleSessionWatch, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"approval-list":             {handler: handleApprovalList, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"approval-get":              {handler: handleApprovalGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"approval-resolve":          {handler: handleApprovalResolve, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"approval-watch":            {handler: handleApprovalWatch, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"list-artifacts":            {handler: handleListArtifacts, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"head-artifact":             {handler: handleHeadArtifact, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"get-artifact":              {handler: handleGetArtifact, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"put-artifact":              {handler: handlePutArtifact, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"check-flow":                {handler: handleCheckFlow, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"promote-excerpt":           {handler: handlePromoteExcerpt, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"revoke-approved-excerpt":   {handler: handleRevokeApprovedExcerpt, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"set-run-status":            {handler: handleSetRunStatus, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"gc":                        {handler: handleGC, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"export-backup":             {handler: handleExportBackup, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"restore-backup":            {handler: handleRestoreBackup, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"show-audit":                {handler: handleShowAudit, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"show-policy":               {handler: handleShowPolicy, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"set-reserved-classes":      {handler: handleSetReservedClasses, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"import-trusted-contract":   {handler: handleImportTrustedContract, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
		"seed-dev-manual-scenario":  {handler: handleSeedDevManualScenario, requiresStore: true, apiMode: brokerCommandAPIModeInProcess},
	}
	addLiveIPCCommandHandlers(handlers)
	return handlers
}

func addLiveIPCCommandHandlers(handlers map[string]brokerCommandSpec) {
	for command, spec := range map[string]brokerCommandSpec{
		"audit-readiness":                         {handler: handleAuditReadiness, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-verification":                      {handler: handleAuditVerification, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-finalize-verify":                   {handler: handleAuditFinalizeVerify, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-record-get":                        {handler: handleAuditRecordGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-anchor-segment":                    {handler: handleAuditAnchorSegment, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-setup-get":                           {handler: handleGitSetupGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-setup-auth-bootstrap":                {handler: handleGitSetupAuthBootstrap, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-setup-identity-upsert":               {handler: handleGitSetupIdentityUpsert, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"provider-setup-direct":                   {handler: handleProviderSetupDirect, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"provider-credential-lease-issue":         {handler: handleProviderCredentialLeaseIssue, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"provider-profile-list":                   {handler: handleProviderProfileList, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"provider-profile-get":                    {handler: handleProviderProfileGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-get":                   {handler: handleProjectSubstrateGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-posture-get":           {handler: handleProjectSubstratePostureGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-adopt":                 {handler: handleProjectSubstrateAdopt, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-init-preview":          {handler: handleProjectSubstrateInitPreview, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-init-apply":            {handler: handleProjectSubstrateInitApply, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-upgrade-preview":       {handler: handleProjectSubstrateUpgradePreview, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-upgrade-apply":         {handler: handleProjectSubstrateUpgradeApply, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-remote-mutation-prepare":             {handler: handleGitRemoteMutationPrepare, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-remote-mutation-get":                 {handler: handleGitRemoteMutationGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-remote-mutation-issue-execute-lease": {handler: handleGitRemoteMutationIssueExecuteLease, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-remote-mutation-execute":             {handler: handleGitRemoteMutationExecute, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"version-info":                            {handler: handleVersionInfo, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"stream-logs":                             {handler: handleStreamLogs, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"llm-invoke":                              {handler: handleLLMInvoke, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"llm-stream":                              {handler: handleLLMStream, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
	} {
		handlers[command] = spec
	}
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
	if err := fs.Parse(args); err != nil {
		return brokerGlobalOptions{}, nil, &usageError{message: "usage: runecode-broker [--state-root path] [--audit-ledger-root path] <command> [flags]"}
	}
	globalOpts.roots.stateRoot = *stateRoot
	globalOpts.roots.auditLedgerRoot = *auditLedgerRoot
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
  session-execution-trigger --session-id id [--trigger-source interactive_user|autonomous_background|resume_follow_up] [--requested-operation start|continue] [--user-message text] [--idempotency-key key]
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
	  provider-credential-lease-issue --provider-profile-id id --run-id run-1 [--ttl-seconds 900]
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
