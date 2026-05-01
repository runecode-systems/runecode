//go:build linux

package launcherdaemon

import (
	"os"
	"path/filepath"
	"sort"
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

func setHelloWorldGoBinaryCandidatesForTests(t *testing.T, candidates []string) {
	t.Helper()
	previous := helloWorldGoBinaryCandidates
	helloWorldGoBinaryCandidates = append([]string(nil), candidates...)
	t.Cleanup(func() {
		helloWorldGoBinaryCandidates = previous
	})
}
