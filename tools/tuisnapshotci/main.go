package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	snapshotBuildTag = "runecode_tui_snapshot"
	snapshotBundle   = "full-audit"
	snapshotViewport = "desktop"
	manifestVersion  = 2
)

var expectedScenarioNames = []string{
	"dashboard-healthy-empty",
	"dashboard-approval-waiting",
	"dashboard-blocked",
	"dashboard-degraded",
	"chat-active-session",
	"runs-active-detail",
	"approvals-pending-detail",
	"action-center-triage",
	"audit-degraded-detail",
	"status-ready-overview",
	"model-providers-credential-needed",
	"git-setup-identity-needed",
	"git-remote-approval-ready",
}

type snapshotManifest struct {
	Version   int                     `json:"version"`
	Bundle    string                  `json:"bundle,omitempty"`
	Viewport  string                  `json:"viewport,omitempty"`
	Width     int                     `json:"width"`
	Height    int                     `json:"height"`
	Coverage  snapshotCoverage        `json:"coverage,omitempty"`
	Scenarios []snapshotManifestEntry `json:"scenarios"`
}

type snapshotCoverage struct {
	Bundle    string   `json:"bundle,omitempty"`
	Routes    []string `json:"routes,omitempty"`
	Viewports []string `json:"viewports,omitempty"`
	Scenarios []string `json:"scenarios,omitempty"`
}

type snapshotManifestEntry struct {
	Name      string                    `json:"name"`
	Viewport  string                    `json:"viewport,omitempty"`
	Width     int                       `json:"width"`
	Height    int                       `json:"height"`
	Route     string                    `json:"route"`
	Artifacts snapshotManifestArtifacts `json:"artifacts"`
}

type snapshotManifestArtifacts struct {
	ANSI snapshotManifestArtifact `json:"ansi"`
	Text snapshotManifestArtifact `json:"text"`
	SVG  snapshotManifestArtifact `json:"svg"`
	PNG  snapshotManifestArtifact `json:"png"`
}

type snapshotManifestArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "tuisnapshotci failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "local-output-dir":
			return runLocalOutputDir(args[1:], stdout, stderr)
		case "cleanup-local-dir":
			return runCleanupLocalDir(args[1:], stderr)
		}
	}

	outputDir, err := parseOutputDirArg(args, stderr)
	if err != nil {
		return err
	}
	return runSnapshotCI(outputDir, stdout)
}

func runSnapshotCI(outputDir string, stdout io.Writer) error {
	workspaceDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}

	workDir, cleanup, err := prepareWorkDir(outputDir)
	if err != nil {
		return err
	}
	defer cleanup()

	binaryPath := filepath.Join(workDir, snapshotBinaryName())
	artifactsDir, err := prepareArtifactsDir(workDir, outputDir)
	if err != nil {
		return err
	}

	if err := buildSnapshotBinary(workspaceDir, binaryPath); err != nil {
		return err
	}
	if err := generateSnapshotArtifacts(workspaceDir, binaryPath, artifactsDir); err != nil {
		return err
	}
	if err := validateSnapshotArtifacts(artifactsDir); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "validated TUI snapshot artifacts in %s\n", artifactsDir)
	return nil
}

func runLocalOutputDir(args []string, stdout io.Writer, stderr io.Writer) error {
	outputDir, err := parseOutputDirArg(args, stderr)
	if err != nil {
		return err
	}
	resolvedOutputDir, err := resolveLocalSnapshotDir(outputDir)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(stdout, resolvedOutputDir)
	return nil
}

func runCleanupLocalDir(args []string, stderr io.Writer) error {
	outputDir, err := parseOutputDirArg(args, stderr)
	if err != nil {
		return err
	}
	if err := cleanupLocalSnapshotDir(outputDir); err != nil {
		return err
	}
	return nil
}

func parseOutputDirArg(args []string, stderr io.Writer) (string, error) {
	fs := flag.NewFlagSet("tuisnapshotci", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outputDir := fs.String("output-dir", "", "directory for generated snapshot artifacts")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if len(fs.Args()) > 0 {
		return "", fmt.Errorf("tuisnapshotci accepts no positional arguments")
	}
	return *outputDir, nil
}

func prepareWorkDir(outputDir string) (string, func(), error) {
	if strings.TrimSpace(outputDir) != "" {
		resolved, err := normalizeTrustedOutputDir(outputDir)
		if err != nil {
			return "", nil, err
		}
		return resolved, func() {}, nil
	}
	tmpDir, err := os.MkdirTemp("", "runecode-tui-snapshot-ci-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp directory: %w", err)
	}
	return tmpDir, func() { _ = os.RemoveAll(tmpDir) }, nil
}

func snapshotBinaryName() string {
	if runtime.GOOS == "windows" {
		return "runecode-tui.exe"
	}
	return "runecode-tui"
}

func buildSnapshotBinary(workspaceDir string, binaryPath string) error {
	if err := runCommand(workspaceDir, "go", "build", "-tags", snapshotBuildTag, "-o", binaryPath, "./cmd/runecode-tui"); err != nil {
		return fmt.Errorf("build snapshot-enabled TUI binary: %w", err)
	}
	return nil
}

func generateSnapshotArtifacts(workspaceDir string, binaryPath string, outputDir string) error {
	if err := runCommand(workspaceDir, binaryPath, "--snapshot-scenario", "all", "--snapshot-viewport", snapshotViewport, "--snapshot-output-dir", outputDir); err != nil {
		return fmt.Errorf("generate snapshot artifacts: %w", err)
	}
	return nil
}

func runCommand(workspaceDir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = workspaceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, trimmed)
	}
	return nil
}
