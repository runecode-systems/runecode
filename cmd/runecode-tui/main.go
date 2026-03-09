package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/scaffold"
	"golang.org/x/term"
)

type model struct {
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key := msg.(type) {
	case tea.KeyMsg:
		switch key.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye from runecode-tui.\n"
	}

	return "Runecode TUI scaffold\nPress q or ctrl+c to quit.\n"
}

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

	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "runecode-tui failed: %v\n", err)
		os.Exit(1)
	}
}
