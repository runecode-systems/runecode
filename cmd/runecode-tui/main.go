// Command runecode-tui launches the interactive terminal UI.
package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

func main() {
	args := os.Args[1:]
	if err := validateArgs(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if len(args) == 1 && isHelpArg(args[0]) {
		if err := writeHelp(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "runecode-tui failed to write help: %v\n", err)
			os.Exit(1)
		}

		return
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		if err := writeNonInteractiveMessage(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "runecode-tui failed to write output: %v\n", err)
			os.Exit(1)
		}

		return
	}

	p := tea.NewProgram(newShellModel(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "runecode-tui failed: %v\n", err)
		os.Exit(1)
	}
}

func isHelpArg(arg string) bool {
	switch arg {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

func validateArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}
	if len(args) == 1 && isHelpArg(args[0]) {
		return nil
	}
	return &usageError{message: "runecode-tui accepts no arguments; use --help for usage"}
}

func writeHelp(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "Usage: runecode-tui [--help]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Interactive terminal UI for the local RuneCode broker API."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Requires a local broker API listener started in another terminal:"); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "  runecode-broker serve-local")
	return err
}

func writeNonInteractiveMessage(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "runecode-tui is an interactive terminal UI for the local broker API."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Interactive terminal required to launch UI."); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "Start local broker first in another terminal: runecode-broker serve-local")
	return err
}
