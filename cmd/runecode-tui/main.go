// Command runecode-tui launches the interactive terminal UI.
package main

import (
	"flag"
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
	cfg, err := parseCLIConfig(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if cfg.showHelp {
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

	setCLIIPCConfigOverrides(cfg)

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

type tuiCLIConfig struct {
	showHelp   bool
	runtimeDir string
	socketName string
}

func parseCLIConfig(args []string) (tuiCLIConfig, error) {
	if len(args) == 1 && isHelpArg(args[0]) {
		return tuiCLIConfig{showHelp: true}, nil
	}
	fs := flag.NewFlagSet("runecode-tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runtimeDir := fs.String("runtime-dir", "", "broker local IPC runtime directory override")
	socketName := fs.String("socket-name", "", "broker local IPC socket filename override")
	if err := fs.Parse(args); err != nil {
		return tuiCLIConfig{}, &usageError{message: "runecode-tui usage: runecode-tui [--runtime-dir dir] [--socket-name broker.sock] [--help]"}
	}
	if len(fs.Args()) > 0 {
		return tuiCLIConfig{}, &usageError{message: "runecode-tui accepts no positional arguments; use --help for usage"}
	}
	return tuiCLIConfig{runtimeDir: *runtimeDir, socketName: *socketName}, nil
}

func setCLIIPCConfigOverrides(cfg tuiCLIConfig) {
	if cfg.runtimeDir == "" && cfg.socketName == "" {
		return
	}
	localIPCConfigProvider = localIPCConfigProviderWithOverrides(localIPCConfigProvider, cfg.runtimeDir, cfg.socketName)
}

func writeHelp(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "Usage: runecode-tui [--runtime-dir dir] [--socket-name broker.sock] [--help]"); err != nil {
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
	if _, err := fmt.Fprintln(w, "  runecode-broker serve-local"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Optional IPC override flags are useful for isolated manual/dev workflows:"); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "  runecode-tui --runtime-dir /tmp/runecode-dev/runtime --socket-name broker.dev.sock")
	return err
}

func writeNonInteractiveMessage(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "runecode-tui is an interactive terminal UI for the local broker API."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Interactive terminal required to launch UI."); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "Start local broker first in another terminal: runecode-broker serve-local [--runtime-dir dir] [--socket-name broker.sock]")
	return err
}
