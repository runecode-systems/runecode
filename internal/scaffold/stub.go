package scaffold

import (
	"fmt"
	"io"
)

func IsHelpArg(arg string) bool {
	switch arg {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

func ValidateArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}

	if len(args) == 1 && IsHelpArg(args[0]) {
		return nil
	}

	return fmt.Errorf("this scaffold stub accepts no arguments")
}

func WriteStubMessage(w io.Writer, binary string) error {
	if _, err := fmt.Fprintf(w, "%s is scaffolded and not yet implemented.\n", binary); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "No network listeners are started in this stub."); err != nil {
		return err
	}

	return nil
}

func WriteHelp(w io.Writer, binary string) error {
	if _, err := fmt.Fprintf(w, "Usage: %s [--help]\n\n", binary); err != nil {
		return err
	}

	return WriteStubMessage(w, binary)
}
