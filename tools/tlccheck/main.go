// Command tlccheck runs bounded TLC model checks for RuneCode formal specs.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	specDirRelative = "formal/tla/security-kernel"
	moduleName      = "SecurityKernelV0"
)

var modelConfigs = []string{
	"SecurityKernelV0.core.cfg",
	"SecurityKernelV0.replay.cfg",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "tlc model check failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve workspace root: %w", err)
	}

	specDir := filepath.Join(repoRoot, specDirRelative)
	if err := ensureDir(specDir); err != nil {
		return err
	}

	runner, err := resolveTLCRunner(repoRoot)
	if err != nil {
		return err
	}

	for _, cfg := range modelConfigs {
		cfgPath := filepath.Join(specDir, cfg)
		if err := ensureFile(cfgPath); err != nil {
			return err
		}

		if err := runModelCheck(specDir, moduleName, cfg, runner); err != nil {
			return err
		}
	}

	return nil
}

func runModelCheck(specDir, moduleName, configName string, runner tlcRunner) error {
	metaDir, err := os.MkdirTemp("", "runecode-tlc-")
	if err != nil {
		return fmt.Errorf("create temp tlc metadata dir: %w", err)
	}
	defer os.RemoveAll(metaDir)

	args := []string{
		"-workers",
		"1",
		"-cleanup",
		"-metadir",
		metaDir,
		"-config",
		configName,
		moduleName,
	}

	cmd := exec.Command(runner.program, append(runner.argsPrefix, args...)...)
	cmd.Dir = specDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[tlc] running %s with %s\n", configName, runner.description)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", configName, err)
	}

	return nil
}

type tlcRunner struct {
	program     string
	argsPrefix  []string
	description string
}

func resolveTLCRunner(repoRoot string) (tlcRunner, error) {
	if path, err := exec.LookPath("tlc"); err == nil {
		return tlcRunner{program: path, description: "tlc binary"}, nil
	}

	jarCandidates := []string{}
	if envJar := os.Getenv("TLA2TOOLS_JAR"); envJar != "" {
		jarCandidates = append(jarCandidates, envJar)
	}
	jarCandidates = append(jarCandidates, filepath.Join(repoRoot, "third_party", "tlaplus", "tla2tools.jar"))

	javaPath, javaErr := exec.LookPath("java")
	if javaErr == nil {
		for _, candidate := range jarCandidates {
			if fileExists(candidate) {
				return tlcRunner{
					program:     javaPath,
					argsPrefix:  []string{"-cp", candidate, "tlc2.TLC"},
					description: "java + tla2tools.jar",
				}, nil
			}
		}
	}

	if nixPath, err := exec.LookPath("nix"); err == nil {
		return tlcRunner{
			program:     nixPath,
			argsPrefix:  []string{"develop", "--no-write-lock-file", "-c", "tlc"},
			description: "nix develop tlc",
		}, nil
	}

	if javaErr != nil {
		return tlcRunner{}, errors.New("tlc binary not found in PATH, no tla2tools.jar configured, and java unavailable; install tlaplus, set TLA2TOOLS_JAR with java on PATH, or use nix develop")
	}

	return tlcRunner{}, errors.New("tlc binary not found in PATH and no tla2tools.jar available; set TLA2TOOLS_JAR, install tlaplus, or use nix develop")
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat dir %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory: %s", path)
	}

	return nil
}

func ensureFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file, got directory: %s", path)
	}

	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
