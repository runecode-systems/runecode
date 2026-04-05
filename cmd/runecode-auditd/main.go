// Command runecode-auditd validates signer evidence for audit events.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var u *usageError
		if errors.As(err, &u) {
			fmt.Fprintln(os.Stderr, u.Error())
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

type commandHandler func([]string, io.Writer) error

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		return writeHelp(stdout)
	}
	handler, ok := commandHandlers()[args[0]]
	if !ok {
		_ = writeHelp(stderr)
		return &usageError{message: fmt.Sprintf("unknown command %q", args[0])}
	}
	return handler(args[1:], stdout)
}

func commandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"validate-signer-evidence": handleValidateSignerEvidence,
	}
}

func handleValidateSignerEvidence(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate-signer-evidence", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "path to audit signer evidence JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "validate-signer-evidence usage: runecode-auditd validate-signer-evidence --file evidence.json"}
	}
	if *filePath == "" {
		return &usageError{message: "validate-signer-evidence requires --file"}
	}
	evidence, err := loadAuditSignerEvidence(*filePath)
	if err != nil {
		return err
	}
	if err := trustpolicy.ValidateAuditSignerEvidence(evidence); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "valid")
	return err
}

func loadAuditSignerEvidence(filePath string) (trustpolicy.AuditSignerEvidence, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return trustpolicy.AuditSignerEvidence{}, err
	}
	evidence := trustpolicy.AuditSignerEvidence{}
	if err := json.Unmarshal(b, &evidence); err != nil {
		return trustpolicy.AuditSignerEvidence{}, err
	}
	return evidence, nil
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode-auditd <command> [flags]

Commands:
  validate-signer-evidence --file evidence.json`)
	return err
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}
