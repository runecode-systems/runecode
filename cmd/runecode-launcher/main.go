// Command runecode-launcher provides trusted isolate-session validation helpers.
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
		"validate-isolate-binding": handleValidateIsolateBinding,
	}
}

func handleValidateIsolateBinding(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate-isolate-binding", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "path to isolate session binding JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "validate-isolate-binding usage: runecode-launcher validate-isolate-binding --file binding.json"}
	}
	if *filePath == "" {
		return &usageError{message: "validate-isolate-binding requires --file"}
	}
	binding, err := loadIsolateSessionBinding(*filePath)
	if err != nil {
		return err
	}
	if err := trustpolicy.ValidateIsolateSessionBinding(binding); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "valid")
	return err
}

func loadIsolateSessionBinding(filePath string) (trustpolicy.IsolateSessionBinding, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return trustpolicy.IsolateSessionBinding{}, err
	}
	binding := trustpolicy.IsolateSessionBinding{}
	if err := json.Unmarshal(b, &binding); err != nil {
		return trustpolicy.IsolateSessionBinding{}, err
	}
	return binding, nil
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode-launcher <command> [flags]

Commands:
  validate-isolate-binding --file binding.json`)
	return err
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}
