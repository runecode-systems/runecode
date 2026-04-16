// Command tlccheck runs bounded TLC model checks for RuneCode formal specs.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	specDirRelative = "formal/tla/security-kernel"
	moduleName      = "SecurityKernelV0"
)

var modelConfigs = []string{
	"SecurityKernelV0.core.cfg",
	"SecurityKernelV0.replay.cfg",
}

var (
	errJavaUnavailable          = errors.New("java unavailable")
	errJavaAvailableNoTLCJar    = errors.New("java available but no tla2tools.jar available")
	errTLCRunnerJavaUnavailable = errors.New("tlc binary not found in PATH and java unavailable; install tlaplus, install java (optionally with TLA2TOOLS_JAR), or use nix develop")
	errTLCRunnerJavaNoJar       = errors.New("tlc binary not found in PATH, java is available but no tla2tools.jar was found, and nix unavailable; set TLA2TOOLS_JAR, vendor third_party/tlaplus/tla2tools.jar, install tlaplus, or use nix develop")
	errTLCRunnerNoJar           = errors.New("tlc binary not found in PATH and no tla2tools.jar available; set TLA2TOOLS_JAR, install tlaplus, or use nix develop")
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "tlc model check failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		return err
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
	return resolveTLCRunnerWithLookPath(repoRoot, exec.LookPath)
}

func resolveTLCRunnerWithLookPath(repoRoot string, lookup func(string) (string, error)) (tlcRunner, error) {
	if runner, ok := resolveNativeTLCRunner(lookup); ok {
		return runner, nil
	}

	jarCandidates, err := tlcJarCandidates(repoRoot)
	if err != nil {
		return tlcRunner{}, err
	}

	javaRunner, javaErr := resolveJavaTLCRunner(jarCandidates, lookup)
	if javaErr == nil {
		return javaRunner, nil
	}

	if runner, ok, err := resolveNixFallbackRunner(repoRoot, lookup); err != nil {
		return tlcRunner{}, err
	} else if ok {
		return runner, nil
	}

	return missingRunnerError(javaErr)
}

func resolveNativeTLCRunner(lookup func(string) (string, error)) (tlcRunner, bool) {
	path, err := lookup("tlc")
	if err != nil {
		return tlcRunner{}, false
	}
	return tlcRunner{program: path, description: "tlc binary"}, true
}

func tlcJarCandidates(repoRoot string) ([]string, error) {
	jarCandidates := []string{}
	if envJar := os.Getenv("TLA2TOOLS_JAR"); envJar != "" {
		if !filepath.IsAbs(envJar) {
			return nil, fmt.Errorf("TLA2TOOLS_JAR must be an absolute path: %q", envJar)
		}
		if !strings.EqualFold(filepath.Ext(envJar), ".jar") {
			return nil, fmt.Errorf("TLA2TOOLS_JAR must point to a .jar file: %q", envJar)
		}
		jarCandidates = append(jarCandidates, envJar)
	}
	jarCandidates = append(jarCandidates, filepath.Join(repoRoot, "third_party", "tlaplus", "tla2tools.jar"))
	return jarCandidates, nil
}

func resolveJavaTLCRunner(jarCandidates []string, lookup func(string) (string, error)) (tlcRunner, error) {
	javaPath, err := lookup("java")
	if err != nil {
		return tlcRunner{}, fmt.Errorf("%w: %v", errJavaUnavailable, err)
	}
	for _, candidate := range jarCandidates {
		if fileExists(candidate) {
			return tlcRunner{
				program:     javaPath,
				argsPrefix:  []string{"-cp", candidate, "tlc2.TLC"},
				description: "java + tla2tools.jar",
			}, nil
		}
	}
	return tlcRunner{}, errJavaAvailableNoTLCJar
}

func resolveNixFallbackRunner(repoRoot string, lookup func(string) (string, error)) (tlcRunner, bool, error) {
	nixPath, err := lookup("nix")
	if err != nil {
		return tlcRunner{}, false, nil
	}
	if !fileExists(filepath.Join(repoRoot, "flake.nix")) {
		return tlcRunner{}, false, fmt.Errorf("nix fallback requires flake.nix at repo root: %s", repoRoot)
	}
	return nixTLCRunner(repoRoot, nixPath), true, nil
}

func missingRunnerError(javaErr error) (tlcRunner, error) {
	if errors.Is(javaErr, errJavaUnavailable) {
		return tlcRunner{}, errTLCRunnerJavaUnavailable
	}
	if errors.Is(javaErr, errJavaAvailableNoTLCJar) {
		return tlcRunner{}, errTLCRunnerJavaNoJar
	}
	return tlcRunner{}, errTLCRunnerNoJar
}

func nixTLCRunner(repoRoot, nixPath string) tlcRunner {
	return tlcRunner{
		program:     nixPath,
		argsPrefix:  []string{"develop", "--no-write-lock-file", "--flake", repoRoot, "-c", "tlc"},
		description: "nix develop tlc",
	}
}

func resolveRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}

	root, ok := findRepoRoot(cwd)
	if !ok {
		return "", fmt.Errorf("resolve repo root from %s: required markers go.mod, justfile, and %s", cwd, specDirRelative)
	}

	return root, nil
}

func findRepoRoot(start string) (string, bool) {
	dir := start
	for {
		if looksLikeRepoRoot(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func looksLikeRepoRoot(path string) bool {
	if !fileExists(filepath.Join(path, "go.mod")) {
		return false
	}
	if !fileExists(filepath.Join(path, "justfile")) {
		return false
	}
	return dirExists(filepath.Join(path, specDirRelative))
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

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
