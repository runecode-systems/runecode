// Command runecode-broker provides a local artifact and policy broker surface.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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
var resolveBrokerRepoScope = func() (localbootstrap.RepoScope, error) {
	return localbootstrap.ResolveRepoScope(localbootstrap.ResolveInput{})
}

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
		"put-artifact":              {handler: handlePutArtifact, requiresStore: false, apiMode: brokerCommandAPIModeInProcess},
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
		"dependency-cache-ensure":                      {handler: handleDependencyCacheEnsure, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"dependency-fetch-registry":                    {handler: handleDependencyFetchRegistry, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"dependency-cache-handoff":                     {handler: handleDependencyCacheHandoff, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-readiness":                              {handler: handleAuditReadiness, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-verification":                           {handler: handleAuditVerification, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-finalize-verify":                        {handler: handleAuditFinalizeVerify, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-record-get":                             {handler: handleAuditRecordGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-record-inclusion-get":                   {handler: handleAuditRecordInclusionGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-evidence-snapshot-get":                  {handler: handleAuditEvidenceSnapshotGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-evidence-retention-review":              {handler: handleAuditEvidenceRetentionReview, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-evidence-bundle-manifest-get":           {handler: handleAuditEvidenceBundleManifestGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-evidence-bundle-export":                 {handler: handleAuditEvidenceBundleExport, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-evidence-bundle-offline-verify":         {handler: handleAuditEvidenceBundleOfflineVerify, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"audit-anchor-segment":                         {handler: handleAuditAnchorSegment, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-setup-get":                                {handler: handleGitSetupGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-setup-auth-bootstrap":                     {handler: handleGitSetupAuthBootstrap, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-setup-identity-upsert":                    {handler: handleGitSetupIdentityUpsert, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"provider-setup-direct":                        {handler: handleProviderSetupDirect, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"provider-credential-lease-issue":              {handler: handleProviderCredentialLeaseIssue, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"provider-profile-list":                        {handler: handleProviderProfileList, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"provider-profile-get":                         {handler: handleProviderProfileGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-get":                        {handler: handleProjectSubstrateGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-posture-get":                {handler: handleProjectSubstratePostureGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-adopt":                      {handler: handleProjectSubstrateAdopt, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-init-preview":               {handler: handleProjectSubstrateInitPreview, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-init-apply":                 {handler: handleProjectSubstrateInitApply, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-upgrade-preview":            {handler: handleProjectSubstrateUpgradePreview, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"project-substrate-upgrade-apply":              {handler: handleProjectSubstrateUpgradeApply, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-remote-mutation-prepare":                  {handler: handleGitRemoteMutationPrepare, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-remote-mutation-get":                      {handler: handleGitRemoteMutationGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-remote-mutation-issue-execute-lease":      {handler: handleGitRemoteMutationIssueExecuteLease, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"git-remote-mutation-execute":                  {handler: handleGitRemoteMutationExecute, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"external-anchor-mutation-prepare":             {handler: handleExternalAnchorMutationPrepare, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"external-anchor-mutation-get":                 {handler: handleExternalAnchorMutationGet, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"external-anchor-mutation-issue-execute-lease": {handler: handleExternalAnchorMutationIssueExecuteLease, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"external-anchor-mutation-execute":             {handler: handleExternalAnchorMutationExecute, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"version-info":                                 {handler: handleVersionInfo, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"stream-logs":                                  {handler: handleStreamLogs, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"llm-invoke":                                   {handler: handleLLMInvoke, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
		"llm-stream":                                   {handler: handleLLMStream, requiresStore: false, apiMode: brokerCommandAPIModeLiveIPC},
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

func defaultBrokerServiceRoots() brokerServiceRoots {
	scope, err := resolveBrokerRepoScope()
	if err == nil {
		seedRepoRootSchemaEnv(scope.RepositoryRoot)
		return brokerServiceRoots{stateRoot: scope.StateRoot, auditLedgerRoot: scope.AuditLedgerRoot}
	}
	return brokerServiceRoots{stateRoot: defaultBrokerStoreRoot(), auditLedgerRoot: auditd.DefaultLedgerRoot()}
}

func newBrokerService(roots brokerServiceRoots) (*brokerapi.Service, error) {
	scope, err := resolveBrokerRepoScope()
	if err != nil {
		return nil, err
	}
	seedRepoRootSchemaEnv(scope.RepositoryRoot)
	resolved := roots
	if resolved.stateRoot == "" {
		resolved.stateRoot = scope.StateRoot
	}
	if resolved.auditLedgerRoot == "" {
		resolved.auditLedgerRoot = scope.AuditLedgerRoot
	}
	return brokerapi.NewServiceWithConfig(resolved.stateRoot, resolved.auditLedgerRoot, brokerapi.APIConfig{RepositoryRoot: scope.RepositoryRoot})
}

func seedRepoRootSchemaEnv(repoRoot string) {
	if strings.TrimSpace(repoRoot) == "" {
		return
	}
	if strings.TrimSpace(os.Getenv("RUNE_REPO_ROOT")) != "" {
		return
	}
	_ = os.Setenv("RUNE_REPO_ROOT", repoRoot)
}
