// Command runecode-secretsd validates signing preconditions for trusted key use.
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
		"validate-sign-request": handleValidateSignRequest,
	}
}

func handleValidateSignRequest(args []string, stdout io.Writer) error {
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

func loadSignRequest(filePath string) (trustpolicy.SignRequestPreconditions, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return trustpolicy.SignRequestPreconditions{}, err
	}
	req := trustpolicy.SignRequestPreconditions{}
	if err := json.Unmarshal(b, &req); err != nil {
		return trustpolicy.SignRequestPreconditions{}, err
	}
	return req, nil
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode-secretsd <command> [flags]

Commands:
  validate-sign-request --file request.json`)
	return err
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}
