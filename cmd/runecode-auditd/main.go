// Command runecode-auditd validates audit writer contracts and evidence.
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
		"validate-admission":       handleValidateAdmission,
		"validate-recovery":        handleValidateRecovery,
		"validate-storage-posture": handleValidateStoragePosture,
		"validate-readiness":       handleValidateReadiness,
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
	evidence := trustpolicy.AuditSignerEvidence{}
	if err := loadJSONFile(filePath, &evidence); err != nil {
		return trustpolicy.AuditSignerEvidence{}, err
	}
	return evidence, nil
}

func handleValidateAdmission(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate-admission", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "path to audit admission request JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "validate-admission usage: runecode-auditd validate-admission --file admission.json"}
	}
	if *filePath == "" {
		return &usageError{message: "validate-admission requires --file"}
	}
	request := trustpolicy.AuditAdmissionRequest{}
	if err := loadJSONFile(*filePath, &request); err != nil {
		return err
	}
	if err := trustpolicy.ValidateAuditAdmissionRequest(request); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout, "valid")
	return err
}

func handleValidateRecovery(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate-recovery", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "path to audit segment recovery state JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "validate-recovery usage: runecode-auditd validate-recovery --file recovery.json"}
	}
	if *filePath == "" {
		return &usageError{message: "validate-recovery requires --file"}
	}
	state := trustpolicy.AuditSegmentRecoveryState{}
	if err := loadJSONFile(*filePath, &state); err != nil {
		return err
	}
	decision, err := trustpolicy.EvaluateAuditSegmentRecovery(state)
	if err != nil {
		return err
	}
	return writeJSON(stdout, decision)
}

func handleValidateStoragePosture(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate-storage-posture", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "path to storage posture evidence JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "validate-storage-posture usage: runecode-auditd validate-storage-posture --file posture.json"}
	}
	if *filePath == "" {
		return &usageError{message: "validate-storage-posture requires --file"}
	}
	evidence := trustpolicy.AuditStoragePostureEvidence{}
	if err := loadJSONFile(*filePath, &evidence); err != nil {
		return err
	}
	if err := trustpolicy.ValidateAuditStoragePostureEvidence(evidence); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout, "valid")
	return err
}

func handleValidateReadiness(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate-readiness", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "path to audit readiness JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "validate-readiness usage: runecode-auditd validate-readiness --file readiness.json"}
	}
	if *filePath == "" {
		return &usageError{message: "validate-readiness requires --file"}
	}
	readiness := trustpolicy.AuditdReadiness{}
	if err := loadJSONFile(*filePath, &readiness); err != nil {
		return err
	}
	if err := trustpolicy.ValidateAuditdReadinessContract(readiness); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout, "valid")
	return err
}

func loadJSONFile(filePath string, target any) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, target); err != nil {
		return err
	}
	return nil
}

func writeJSON(w io.Writer, value interface{}) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode-auditd <command> [flags]

Commands:
  validate-signer-evidence --file evidence.json
  validate-admission --file admission.json
  validate-recovery --file recovery.json
  validate-storage-posture --file posture.json
  validate-readiness --file readiness.json`)
	return err
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}
