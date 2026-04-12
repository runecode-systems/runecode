// Command runecode-tui launches the scaffold terminal UI.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/scaffold"
	"golang.org/x/term"
)

func main() {
	args := os.Args[1:]
	if err := scaffold.ValidateArgs(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if len(args) == 1 && scaffold.IsHelpArg(args[0]) {
		if err := scaffold.WriteHelp(os.Stdout, "runecode-tui"); err != nil {
			fmt.Fprintf(os.Stderr, "runecode-tui failed to write help: %v\n", err)
			os.Exit(1)
		}

		return
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		if err := scaffold.WriteStubMessage(os.Stdout, "runecode-tui"); err != nil {
			fmt.Fprintf(os.Stderr, "runecode-tui failed to write output: %v\n", err)
			os.Exit(1)
		}

		if _, err := fmt.Fprintln(os.Stdout, "Interactive terminal required to launch UI."); err != nil {
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
