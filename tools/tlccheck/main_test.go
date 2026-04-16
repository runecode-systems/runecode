package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNixTLCRunnerIncludesRepoRootFlake(t *testing.T) {
	runner := nixTLCRunner("/repo/root", "/usr/bin/nix")
	if runner.program != "/usr/bin/nix" {
		t.Fatalf("runner.program = %q, want /usr/bin/nix", runner.program)
	}
	wantPrefix := []string{"develop", "--no-write-lock-file", "--flake", "/repo/root", "-c", "tlc"}
	if !reflect.DeepEqual(runner.argsPrefix, wantPrefix) {
		t.Fatalf("runner.argsPrefix = %#v, want %#v", runner.argsPrefix, wantPrefix)
	}
}

func TestResolveTLCRunnerUsesNixFallbackWithRepoRootFlake(t *testing.T) {
	_, err := resolveTLCRunnerWithLookPath("/workspace/runecode", lookPathStub(map[string]string{"nix": "/nix/store/bin/nix"}))
	if err == nil || !strings.Contains(err.Error(), "requires flake.nix") {
		t.Fatalf("resolveTLCRunnerWithLookPath() error = %v, want flake validation error", err)
	}
}

func TestResolveTLCRunnerUsesNixFallbackWithRepoRootFlakeWhenFlakeExists(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "flake.nix"), "{}")

	runner, err := resolveTLCRunnerWithLookPath(repoRoot, lookPathStub(map[string]string{"nix": "/nix/store/bin/nix"}))
	if err != nil {
		t.Fatalf("resolveTLCRunnerWithLookPath() error = %v, want nil", err)
	}
	if runner.program != "/nix/store/bin/nix" {
		t.Fatalf("runner.program = %q, want /nix/store/bin/nix", runner.program)
	}
	wantPrefix := []string{"develop", "--no-write-lock-file", "--flake", repoRoot, "-c", "tlc"}
	if !reflect.DeepEqual(runner.argsPrefix, wantPrefix) {
		t.Fatalf("runner.argsPrefix = %#v, want %#v", runner.argsPrefix, wantPrefix)
	}
}

func TestResolveTLCRunnerPrefersTLCBinaryOverNixFallback(t *testing.T) {
	runner, err := resolveTLCRunnerWithLookPath("/workspace/runecode", lookPathStub(map[string]string{
		"tlc": "/usr/local/bin/tlc",
		"nix": "/nix/store/bin/nix",
	}))
	if err != nil {
		t.Fatalf("resolveTLCRunnerWithLookPath() error = %v, want nil", err)
	}
	if runner.program != "/usr/local/bin/tlc" {
		t.Fatalf("runner.program = %q, want /usr/local/bin/tlc", runner.program)
	}
	if len(runner.argsPrefix) != 0 {
		t.Fatalf("runner.argsPrefix = %#v, want empty", runner.argsPrefix)
	}
}

func TestResolveTLCRunnerPrefersJavaAndJarOverNix(t *testing.T) {
	repoRoot := t.TempDir()
	jarPath := filepath.Join(repoRoot, "custom", "tla2tools.jar")
	writeFile(t, jarPath, "jar-bytes")
	t.Setenv("TLA2TOOLS_JAR", jarPath)

	runner, err := resolveTLCRunnerWithLookPath(repoRoot, lookPathStub(map[string]string{
		"java": "/usr/bin/java",
		"nix":  "/nix/store/bin/nix",
	}))
	if err != nil {
		t.Fatalf("resolveTLCRunnerWithLookPath() error = %v, want nil", err)
	}
	if runner.program != "/usr/bin/java" {
		t.Fatalf("runner.program = %q, want /usr/bin/java", runner.program)
	}
	wantPrefix := []string{"-cp", jarPath, "tlc2.TLC"}
	if !reflect.DeepEqual(runner.argsPrefix, wantPrefix) {
		t.Fatalf("runner.argsPrefix = %#v, want %#v", runner.argsPrefix, wantPrefix)
	}
}

func TestResolveTLCRunnerRejectsRelativeTLA2TOOLSJar(t *testing.T) {
	t.Setenv("TLA2TOOLS_JAR", "relative/path/tla2tools.jar")

	_, err := resolveTLCRunnerWithLookPath("/workspace/runecode", lookPathStub(map[string]string{"java": "/usr/bin/java"}))
	if err == nil || !strings.Contains(err.Error(), "must be an absolute path") {
		t.Fatalf("resolveTLCRunnerWithLookPath() error = %v, want absolute-path validation error", err)
	}
}

func TestResolveTLCRunnerRejectsNonJarTLA2TOOLSPath(t *testing.T) {
	badPath := filepath.Join(t.TempDir(), "not-a-jar.txt")
	t.Setenv("TLA2TOOLS_JAR", badPath)

	_, err := resolveTLCRunnerWithLookPath("/workspace/runecode", lookPathStub(map[string]string{"java": "/usr/bin/java"}))
	if err == nil || !strings.Contains(err.Error(), "must point to a .jar file") {
		t.Fatalf("resolveTLCRunnerWithLookPath() error = %v, want extension validation error", err)
	}
}

func TestResolveTLCRunnerMissingRunnerErrorMentionsJavaUnavailable(t *testing.T) {
	t.Setenv("TLA2TOOLS_JAR", "")

	_, err := resolveTLCRunnerWithLookPath("/workspace/runecode", lookPathStub(map[string]string{}))
	if err == nil {
		t.Fatal("resolveTLCRunnerWithLookPath() error = nil, want runner resolution error")
	}
	if !strings.Contains(err.Error(), "java unavailable") {
		t.Fatalf("resolveTLCRunnerWithLookPath() error = %v, want java unavailable message", err)
	}
}

func TestResolveTLCRunnerMissingRunnerErrorMentionsJarMissingWhenJavaExists(t *testing.T) {
	t.Setenv("TLA2TOOLS_JAR", "")

	_, err := resolveTLCRunnerWithLookPath("/workspace/runecode", lookPathStub(map[string]string{"java": "/usr/bin/java"}))
	if err == nil {
		t.Fatal("resolveTLCRunnerWithLookPath() error = nil, want runner resolution error")
	}
	if !strings.Contains(err.Error(), "java is available but no tla2tools.jar") {
		t.Fatalf("resolveTLCRunnerWithLookPath() error = %v, want java-available jar-missing message", err)
	}
}

func TestFindRepoRootWalksUpToRepoMarkers(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "go.mod"), "module github.com/runecode-ai/runecode\n")
	writeFile(t, filepath.Join(repoRoot, "justfile"), "default:\n\t@true\n")
	if err := os.MkdirAll(filepath.Join(repoRoot, specDirRelative), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	nested := filepath.Join(repoRoot, "tools", "tlccheck")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	got, ok := findRepoRoot(nested)
	if !ok {
		t.Fatal("findRepoRoot() = false, want true")
	}
	if got != repoRoot {
		t.Fatalf("findRepoRoot() = %q, want %q", got, repoRoot)
	}
}

func lookPathStub(entries map[string]string) func(string) (string, error) {
	return func(file string) (string, error) {
		if path, ok := entries[file]; ok {
			return path, nil
		}
		return "", errors.New("not found")
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
