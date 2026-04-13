// Command runecode-secretsd provides trusted local secrets runtime operations.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		var u *usageError
		if errors.As(err, &u) {
			fmt.Fprintln(os.Stderr, u.Error())
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

type commandHandler func([]string, io.Reader, io.Writer) error

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		return writeHelp(stdout)
	}
	handler, ok := commandHandlers()[args[0]]
	if !ok {
		_ = writeHelp(stderr)
		return &usageError{message: fmt.Sprintf("unknown command %q", args[0])}
	}
	return handler(args[1:], stdin, stdout)
}

func commandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"validate-sign-request": handleValidateSignRequest,
		"import-secret":         handleImportSecret,
		"lease-issue":           handleLeaseIssue,
		"lease-renew":           handleLeaseRenew,
		"lease-revoke":          handleLeaseRevoke,
		"lease-retrieve":        handleLeaseRetrieve,
	}
}

func handleValidateSignRequest(args []string, _ io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate-sign-request", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "path to sign-request preconditions JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "validate-sign-request usage: runecode-secretsd validate-sign-request --file request.json"}
	}
	if *filePath == "" {
		return &usageError{message: "validate-sign-request requires --file"}
	}
	req, err := loadSignRequest(*filePath)
	if err != nil {
		return err
	}
	if err := trustpolicy.ValidateSignRequestPreconditions(req); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "valid")
	return err
}

func handleImportSecret(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("import-secret", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateRoot := fs.String("state-root", defaultStateRoot(), "state root")
	secretRef := fs.String("secret-ref", "", "canonical secret reference")
	fd := fs.Int("fd", -1, "file descriptor containing secret material")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "import-secret usage: runecode-secretsd import-secret --secret-ref ref [--state-root path] [--fd N]"}
	}
	if strings.TrimSpace(*secretRef) == "" {
		return &usageError{message: "import-secret requires --secret-ref"}
	}
	resolvedRoot, err := resolveValidatedStateRoot(*stateRoot)
	if err != nil {
		return err
	}
	svc, err := secretsd.Open(resolvedRoot)
	if err != nil {
		return err
	}
	materialReader, closeFn, err := onboardingReader(*fd, stdin)
	if err != nil {
		return err
	}
	if closeFn != nil {
		defer closeFn()
	}
	meta, err := svc.ImportSecret(*secretRef, materialReader)
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{
		"secret_ref":      meta.SecretRef,
		"secret_id":       meta.SecretID,
		"material_digest": meta.MaterialDigest,
		"imported_at":     meta.ImportedAt,
	})
}

func handleLeaseIssue(args []string, _ io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("lease-issue", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateRoot := fs.String("state-root", defaultStateRoot(), "state root")
	secretRef := fs.String("secret-ref", "", "canonical secret reference")
	consumerID := fs.String("consumer-id", "", "consumer principal")
	roleKind := fs.String("role-kind", "", "consumer role kind")
	scope := fs.String("scope", "", "bound scope")
	ttl := fs.Int("ttl-seconds", 0, "requested ttl seconds")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "lease-issue usage: runecode-secretsd lease-issue --secret-ref ref --consumer-id id --role-kind kind --scope scope [--ttl-seconds N] [--state-root path]"}
	}
	resolvedRoot, err := resolveValidatedStateRoot(*stateRoot)
	if err != nil {
		return err
	}
	svc, err := secretsd.Open(resolvedRoot)
	if err != nil {
		return err
	}
	lease, err := svc.IssueLease(secretsd.IssueLeaseRequest{SecretRef: *secretRef, ConsumerID: *consumerID, RoleKind: *roleKind, Scope: *scope, TTLSeconds: *ttl})
	if err != nil {
		return err
	}
	return writeJSON(stdout, lease)
}

func handleLeaseRenew(args []string, _ io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("lease-renew", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateRoot := fs.String("state-root", defaultStateRoot(), "state root")
	leaseID := fs.String("lease-id", "", "lease identifier")
	consumerID := fs.String("consumer-id", "", "consumer principal")
	roleKind := fs.String("role-kind", "", "consumer role kind")
	scope := fs.String("scope", "", "bound scope")
	ttl := fs.Int("ttl-seconds", 0, "requested ttl seconds")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "lease-renew usage: runecode-secretsd lease-renew --lease-id id --consumer-id id --role-kind kind --scope scope [--ttl-seconds N] [--state-root path]"}
	}
	resolvedRoot, err := resolveValidatedStateRoot(*stateRoot)
	if err != nil {
		return err
	}
	svc, err := secretsd.Open(resolvedRoot)
	if err != nil {
		return err
	}
	lease, err := svc.RenewLease(secretsd.RenewLeaseRequest{LeaseID: *leaseID, ConsumerID: *consumerID, RoleKind: *roleKind, Scope: *scope, TTLSeconds: *ttl})
	if err != nil {
		return err
	}
	return writeJSON(stdout, lease)
}

func handleLeaseRevoke(args []string, _ io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("lease-revoke", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateRoot := fs.String("state-root", defaultStateRoot(), "state root")
	leaseID := fs.String("lease-id", "", "lease identifier")
	consumerID := fs.String("consumer-id", "", "consumer principal")
	roleKind := fs.String("role-kind", "", "consumer role kind")
	scope := fs.String("scope", "", "bound scope")
	reason := fs.String("reason", "", "revocation reason")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "lease-revoke usage: runecode-secretsd lease-revoke --lease-id id --consumer-id id --role-kind kind --scope scope [--reason text] [--state-root path]"}
	}
	resolvedRoot, err := resolveValidatedStateRoot(*stateRoot)
	if err != nil {
		return err
	}
	svc, err := secretsd.Open(resolvedRoot)
	if err != nil {
		return err
	}
	lease, err := svc.RevokeLease(secretsd.RevokeLeaseRequest{LeaseID: *leaseID, ConsumerID: *consumerID, RoleKind: *roleKind, Scope: *scope, Reason: *reason})
	if err != nil {
		return err
	}
	return writeJSON(stdout, lease)
}

func handleLeaseRetrieve(args []string, _ io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("lease-retrieve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateRoot := fs.String("state-root", defaultStateRoot(), "state root")
	leaseID := fs.String("lease-id", "", "lease identifier")
	consumerID := fs.String("consumer-id", "", "consumer principal")
	roleKind := fs.String("role-kind", "", "consumer role kind")
	scope := fs.String("scope", "", "bound scope")
	outputFD := fs.Int("output-fd", -1, "file descriptor receiving retrieved secret material")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "lease-retrieve usage: runecode-secretsd lease-retrieve --lease-id id --consumer-id id --role-kind kind --scope scope --output-fd N [--state-root path]"}
	}
	if *outputFD < 0 {
		return &usageError{message: "lease-retrieve requires --output-fd"}
	}
	resolvedRoot, err := resolveValidatedStateRoot(*stateRoot)
	if err != nil {
		return err
	}
	svc, err := secretsd.Open(resolvedRoot)
	if err != nil {
		return err
	}
	material, lease, err := svc.Retrieve(secretsd.RetrieveRequest{LeaseID: *leaseID, ConsumerID: *consumerID, RoleKind: *roleKind, Scope: *scope})
	if err != nil {
		return err
	}
	if err := writeLeaseMaterialToFD(*outputFD, material); err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{"lease_id": lease.LeaseID, "status": lease.Status, "bytes_written": len(material)})
}

func onboardingReader(fd int, stdin io.Reader) (io.Reader, func(), error) {
	if fd >= 0 {
		if runtime.GOOS == "windows" {
			return nil, nil, fmt.Errorf("--fd is not supported on windows")
		}
		file := os.NewFile(uintptr(fd), "secret-fd")
		if file == nil {
			return nil, nil, fmt.Errorf("invalid --fd")
		}
		return file, func() { _ = file.Close() }, nil
	}
	if stdin == nil {
		return nil, nil, fmt.Errorf("stdin is required for secret material")
	}
	return stdin, nil, nil
}

func loadSignRequest(filePath string) (trustpolicy.SignRequestPreconditions, error) {
	f, err := openValidatedSignRequestFile(filePath)
	if err != nil {
		return trustpolicy.SignRequestPreconditions{}, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return trustpolicy.SignRequestPreconditions{}, err
	}
	req := trustpolicy.SignRequestPreconditions{}
	if err := json.Unmarshal(b, &req); err != nil {
		return trustpolicy.SignRequestPreconditions{}, err
	}
	return req, nil
}

func openValidatedSignRequestFile(filePath string) (*os.File, error) {
	trimmed := strings.TrimSpace(filePath)
	if trimmed == "" {
		return nil, fmt.Errorf("--file path is required")
	}
	cleanPath, err := filepath.Abs(filepath.Clean(trimmed))
	if err != nil {
		return nil, err
	}
	if err := rejectLinkedPathComponents(filepath.Dir(cleanPath)); err != nil {
		if errors.Is(err, errLinkedPathComponent) {
			return nil, fmt.Errorf("--file path must not contain symlink path components")
		}
		return nil, err
	}
	initialInfo, err := os.Lstat(cleanPath)
	if err != nil {
		return nil, err
	}
	if initialInfo.IsDir() {
		return nil, fmt.Errorf("--file path must point to a regular file")
	}
	linked, err := pathEntryIsLinkOrReparse(cleanPath, initialInfo)
	if err != nil {
		return nil, err
	}
	if linked {
		return nil, fmt.Errorf("--file path must not be a symlink")
	}
	if !initialInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("--file path must point to a regular file")
	}
	f, err := os.Open(cleanPath)
	if err != nil {
		return nil, err
	}
	openedInfo, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if !os.SameFile(initialInfo, openedInfo) {
		_ = f.Close()
		return nil, fmt.Errorf("--file path changed during validation")
	}
	return f, nil
}

func writeJSON(w io.Writer, value interface{}) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func defaultStateRoot() string {
	if root := strings.TrimSpace(os.Getenv("RUNE_SECRETS_STATE_ROOT")); root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".runecode-secretsd"
	}
	return filepath.Join(home, ".runecode", "secretsd")
}

func resolveValidatedStateRoot(root string) (string, error) {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return "", fmt.Errorf("state root is required")
	}
	cleanPath, err := filepath.Abs(filepath.Clean(trimmed))
	if err != nil {
		return "", err
	}
	if err := rejectLinkedPathComponents(filepath.Dir(cleanPath)); err != nil {
		if errors.Is(err, errLinkedPathComponent) {
			return "", fmt.Errorf("state root must not contain symlink path components")
		}
		return "", err
	}
	info, err := os.Lstat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cleanPath, nil
		}
		return "", err
	}
	linked, err := pathEntryIsLinkOrReparse(cleanPath, info)
	if err != nil {
		return "", err
	}
	if linked {
		return "", fmt.Errorf("state root must not be a symlink")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("state root must be a directory")
	}
	return cleanPath, nil
}

func writeLeaseMaterialToFD(fd int, material []byte) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("--output-fd is not supported on windows")
	}
	f := os.NewFile(uintptr(fd), "lease-output")
	if f == nil {
		return fmt.Errorf("invalid --output-fd")
	}
	defer f.Close()
	if len(material) == 0 {
		return nil
	}
	_, err := f.Write(material)
	return err
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode-secretsd <command> [flags]

Commands:
  validate-sign-request --file request.json
  import-secret --secret-ref ref [--state-root path] [--fd N]
  lease-issue --secret-ref ref --consumer-id id --role-kind kind --scope scope [--ttl-seconds N] [--state-root path]
  lease-renew --lease-id id --consumer-id id --role-kind kind --scope scope [--ttl-seconds N] [--state-root path]
  lease-revoke --lease-id id --consumer-id id --role-kind kind --scope scope [--reason text] [--state-root path]
  lease-retrieve --lease-id id --consumer-id id --role-kind kind --scope scope --output-fd N [--state-root path]

Notes:
  - Secret material import defaults to stdin; --fd is supported on non-Windows platforms.
  - Do not pass secret values via CLI args or environment variables.`)
	return err
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}
