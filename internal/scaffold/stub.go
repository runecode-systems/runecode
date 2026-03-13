// Package scaffold provides shared stub behavior for unimplemented binaries.
package scaffold

import (
	"fmt"
	"io"
)

// IsHelpArg reports whether arg requests scaffold help output.
func IsHelpArg(arg string) bool {
	switch arg {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

// ValidateArgs rejects arguments that scaffold binaries do not support yet.
func ValidateArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}

	if len(args) == 1 && IsHelpArg(args[0]) {
		return nil
	}

	return fmt.Errorf("this scaffold stub accepts no arguments")
}

// WriteStubMessage writes the standard scaffold status message for binary.
func WriteStubMessage(w io.Writer, binary string) error {
	if _, err := fmt.Fprintf(w, "%s is scaffolded and not yet implemented.\n", binary); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "No network listeners are started in this stub."); err != nil {
		return err
	}

	return nil
}

// WriteHelp writes the scaffold usage text and stub status message.
func WriteHelp(w io.Writer, binary string) error {
	if _, err := fmt.Fprintf(w, "Usage: %s [--help]\n\n", binary); err != nil {
		return err
	}

	return WriteStubMessage(w, binary)
}
