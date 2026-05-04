//go:build linux

package launcherdaemon

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestHelloWorldGoBuildEnvMinimal(t *testing.T) {
	workDir := t.TempDir()
	env := helloWorldGoBuildEnv(workDir)
	got := append([]string(nil), env...)
	sort.Strings(got)
	want := []string{
		"CGO_ENABLED=0",
		"GOARCH=amd64",
		"GOCACHE=" + filepath.Join(workDir, "gocache"),
		"GOOS=linux",
		"GOTOOLCHAIN=local",
	}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("env length = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("env[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveHelloWorldGoBinaryUsesApprovedCandidates(t *testing.T) {
	nonExecPath := filepath.Join(t.TempDir(), "go")
	if err := os.WriteFile(nonExecPath, []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(nonExecPath) returned error: %v", err)
	}
	execPath := filepath.Join(t.TempDir(), "go")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(execPath) returned error: %v", err)
	}
	setHelloWorldGoBinaryCandidatesForTests(t, []string{nonExecPath, execPath})

	got, err := resolveHelloWorldGoBinary()
	if err != nil {
		t.Fatalf("resolveHelloWorldGoBinary returned error: %v", err)
	}
	if got != execPath {
		t.Fatalf("resolveHelloWorldGoBinary path = %q, want %q", got, execPath)
	}
}

func TestResolveHelloWorldGoBinaryFailsClosedWhenUnavailable(t *testing.T) {
	setHelloWorldGoBinaryCandidatesForTests(t, []string{filepath.Join(t.TempDir(), "missing-go")})
	_, err := resolveHelloWorldGoBinary()
	if err == nil {
		t.Fatal("resolveHelloWorldGoBinary expected failure")
	}
	if got := err.Error(); got != "host go compiler not found in approved paths" {
		t.Fatalf("resolveHelloWorldGoBinary error = %q", got)
	}
}

func TestBuildHelloInitBinaryUsesStagingDir(t *testing.T) {
	workDir := t.TempDir()
	logPath := filepath.Join(workDir, "pwd.log")
	goBin := filepath.Join(workDir, "go")
	script := "#!/bin/sh\npwd > \"" + logPath + "\"\n"
	if err := os.WriteFile(goBin, []byte(script), 0o700); err != nil {
		t.Fatalf("WriteFile(fake go binary) returned error: %v", err)
	}
	src := filepath.Join(workDir, "init.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(init.go) returned error: %v", err)
	}
	if err := buildHelloInitBinary(context.Background(), goBin, filepath.Join(workDir, "init"), src); err != nil {
		t.Fatalf("buildHelloInitBinary returned error: %v", err)
	}
	loggedDir, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(pwd.log) returned error: %v", err)
	}
	if got := strings.TrimSpace(string(loggedDir)); got != workDir {
		t.Fatalf("buildHelloInitBinary ran in %q, want %q", got, workDir)
	}
}

func TestHelloInitProgramEmitsRuntimePostHandshakeMaterialBeforeHello(t *testing.T) {
	program := helloInitProgram()
	runtimeLineIndex := strings.Index(program, "/proc/cmdline")
	helloIndex := strings.Index(program, helloWorldToken)
	if runtimeLineIndex < 0 {
		t.Fatal("hello init program must read runtime post-handshake material line from guest cmdline")
	}
	if helloIndex < 0 {
		t.Fatal("hello init program must emit hello token")
	}
	if runtimeLineIndex > helloIndex {
		t.Fatal("hello init program must emit runtime post-handshake material before hello token")
	}
}

func TestQEMUGuestRuntimeMaterialKernelArg(t *testing.T) {
	if got := qemuGuestRuntimeMaterialKernelArg(""); got != "" {
		t.Fatalf("guest material arg for empty input = %q, want empty", got)
	}
	if got := qemuGuestRuntimeMaterialKernelArg("RUNE_POST_HANDSHAKE_MATERIAL=abc"); got != "RUNE_POST_HANDSHAKE_MATERIAL_LINE=RUNE_POST_HANDSHAKE_MATERIAL=abc" {
		t.Fatalf("guest material arg = %q", got)
	}
}

func setHelloWorldGoBinaryCandidatesForTests(t *testing.T, candidates []string) {
	t.Helper()
	previous := helloWorldGoBinaryCandidates
	helloWorldGoBinaryCandidates = append([]string(nil), candidates...)
	t.Cleanup(func() {
		helloWorldGoBinaryCandidates = previous
	})
}
