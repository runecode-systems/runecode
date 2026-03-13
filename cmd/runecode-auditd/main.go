// Command runecode-auditd prints scaffold output for the audit daemon.
package main

import (
	"fmt"
	"os"

	"github.com/runecode-ai/runecode/internal/scaffold"
)

func main() {
	args := os.Args[1:]
	if err := scaffold.ValidateArgs(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if len(args) == 1 && scaffold.IsHelpArg(args[0]) {
		if err := scaffold.WriteHelp(os.Stdout, "runecode-auditd"); err != nil {
			fmt.Fprintf(os.Stderr, "runecode-auditd failed to write help: %v\n", err)
			os.Exit(1)
		}

		return
	}

	if err := scaffold.WriteStubMessage(os.Stdout, "runecode-auditd"); err != nil {
		fmt.Fprintf(os.Stderr, "runecode-auditd failed to write output: %v\n", err)
		os.Exit(1)
	}
}
