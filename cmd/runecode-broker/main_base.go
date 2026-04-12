// Command runecode-broker provides a local artifact and policy broker surface.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

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
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		return writeHelp(stdout)
	}
	handler, ok := commandHandlers()[args[0]]
	if !ok {
		_ = writeHelp(stderr)
		return &usageError{message: fmt.Sprintf("unknown command %q", args[0])}
	}
	service, err := brokerServiceFactory()
	if err != nil {
		return fmt.Errorf("runecode-broker failed to initialize store: %w", err)
	}
	return handler(args[1:], service, stdout)
}

var brokerServiceFactory = brokerService
var localIPCListen = brokerapi.ListenLocalIPC

type commandHandler func([]string, *brokerapi.Service, io.Writer) error

func commandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"serve-local":              handleServeLocal,
		"run-list":                 handleRunList,
		"run-get":                  handleRunGet,
		"run-watch":                handleRunWatch,
		"session-list":             handleSessionList,
		"session-get":              handleSessionGet,
		"session-send-message":     handleSessionSendMessage,
		"session-watch":            handleSessionWatch,
		"approval-list":            handleApprovalList,
		"approval-get":             handleApprovalGet,
		"approval-watch":           handleApprovalWatch,
		"list-artifacts":           handleListArtifacts,
		"head-artifact":            handleHeadArtifact,
		"get-artifact":             handleGetArtifact,
		"put-artifact":             handlePutArtifact,
		"check-flow":               handleCheckFlow,
		"promote-excerpt":          handlePromoteExcerpt,
		"revoke-approved-excerpt":  handleRevokeApprovedExcerpt,
		"set-run-status":           handleSetRunStatus,
		"gc":                       handleGC,
		"export-backup":            handleExportBackup,
		"restore-backup":           handleRestoreBackup,
		"show-audit":               handleShowAudit,
		"show-policy":              handleShowPolicy,
		"set-reserved-classes":     handleSetReservedClasses,
		"import-trusted-contract":  handleImportTrustedContract,
		"seed-dev-manual-scenario": handleSeedDevManualScenario,
		"audit-readiness":          handleAuditReadiness,
		"audit-verification":       handleAuditVerification,
		"audit-record-get":         handleAuditRecordGet,
		"version-info":             handleVersionInfo,
		"stream-logs":              handleStreamLogs,
	}
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
	_, err := fmt.Fprintln(w, `Usage: runecode-broker <command> [flags]

Commands:
  serve-local [--runtime-dir dir] [--socket-name broker.sock] [--once]
  run-list [--limit N]
  run-get --run-id id
  run-watch [--stream-id id] [--run-id id] [--workspace-id id] [--lifecycle-state state] [--follow] [--include-snapshot]
  session-list [--limit N]
  session-get --session-id id
  session-send-message --session-id id --content text [--role user|assistant|system|tool] [--idempotency-key key]
  session-watch [--stream-id id] [--session-id id] [--workspace-id id] [--status active|completed|archived] [--last-activity-kind kind] [--follow] [--include-snapshot]
  approval-list [--run-id id] [--status pending|approved|denied|expired|cancelled|superseded|consumed] [--limit N]
  approval-get --approval-id sha256:...
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
  seed-dev-manual-scenario --dev-only [--profile tui-rich-v1]
  audit-readiness
  audit-verification [--limit N]
  audit-record-get --record-digest sha256:...
  version-info
  stream-logs [--stream-id id] [--run-id id] [--role-instance-id id] [--start-cursor cursor] [--follow] [--include-backlog]`)
	return err
}

func brokerService() (*brokerapi.Service, error) {
	return brokerapi.NewService(defaultBrokerStoreRoot(), auditd.DefaultLedgerRoot())
}
