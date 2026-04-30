package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/runecode-ai/runecode/internal/localbootstrap"
)

func writePIDFile(scope localbootstrap.RepoScope, pid int) error {
	return os.WriteFile(pidFilePath(scope), []byte(strconv.Itoa(pid)), 0o600)
}

func pidFilePath(scope localbootstrap.RepoScope) string {
	return filepath.Join(scope.LocalRuntimeDir, "broker.pid")
}

func findSiblingOrPathExecutable(name string) (string, error) {
	exe, err := os.Executable()
	if err == nil {
		sibling := filepath.Join(filepath.Dir(exe), name)
		if info, statErr := os.Stat(sibling); statErr == nil && !info.IsDir() {
			return sibling, nil
		}
	}
	path, lookErr := exec.LookPath(name)
	if lookErr != nil {
		return "", fmt.Errorf("locate %s: %w", name, lookErr)
	}
	return path, nil
}

func isHelpArg(arg string) bool {
	trimmed := strings.TrimSpace(arg)
	return trimmed == "-h" || trimmed == "--help" || trimmed == "help"
}

func joinCSV(values []string) string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = sanitizeCLIField(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	if len(trimmed) == 0 {
		return "none"
	}
	return strings.Join(trimmed, ",")
}

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode [attach|start|status|stop|restart]

Canonical RuneCode product command:
  runecode          same as runecode attach
  runecode attach   ensure repo-scoped broker lifecycle and open TUI
  runecode start    ensure repo-scoped broker lifecycle without opening TUI
  runecode status   non-starting lifecycle status for current repo scope
  runecode stop     stop repo-scoped local broker lifecycle
  runecode restart  stop then start repo-scoped local broker lifecycle`)
	return err
}
