package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestValidateSignRequestCLI(t *testing.T) {
	requestPath := filepath.Join(canonicalTempDir(t), "request.json")
	if err := os.WriteFile(requestPath, []byte(`{
  "logical_purpose": "approval_authority",
  "logical_scope": "user",
  "key_protection_posture": "os_keystore",
  "identity_binding_posture": "attested",
  "presence_mode": "os_confirmation"
}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-sign-request", "--file", requestPath}, strings.NewReader(""), stdout, stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stdout.String() != "valid\n" {
		t.Fatalf("stdout = %q, want valid", stdout.String())
	}
}

func TestImportIssueRenewRevokeRetrieveLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("lease-retrieve output-fd path is non-windows")
	}
	stateRoot := canonicalTempDir(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	secret := "super-secret-material"
	if err := run([]string{"import-secret", "--state-root", stateRoot, "--secret-ref", "secrets/prod/db"}, strings.NewReader(secret), stdout, stderr); err != nil {
		t.Fatalf("import-secret returned error: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"lease-issue", "--state-root", stateRoot, "--secret-ref", "secrets/prod/db", "--consumer-id", "principal:runner:1", "--role-kind", "runner", "--scope", "stage:alpha", "--ttl-seconds", "120"}, strings.NewReader(""), stdout, stderr); err != nil {
		t.Fatalf("lease-issue returned error: %v", err)
	}
	issued := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &issued); err != nil {
		t.Fatalf("lease-issue json unmarshal error: %v", err)
	}
	leaseID, _ := issued["lease_id"].(string)
	if leaseID == "" {
		t.Fatal("lease_id missing")
	}
	stdout.Reset()
	if err := run([]string{"lease-renew", "--state-root", stateRoot, "--lease-id", leaseID, "--consumer-id", "principal:runner:1", "--role-kind", "runner", "--scope", "stage:alpha", "--ttl-seconds", "180"}, strings.NewReader(""), stdout, stderr); err != nil {
		t.Fatalf("lease-renew returned error: %v", err)
	}
	retrieved, err := runLeaseRetrieveToPipe(stateRoot, leaseID, "principal:runner:1", "runner", "stage:alpha", stderr)
	if err != nil {
		t.Fatalf("lease-retrieve returned error: %v", err)
	}
	if retrieved != secret {
		t.Fatalf("retrieve output missing material")
	}
	stdout.Reset()
	if err := run([]string{"lease-revoke", "--state-root", stateRoot, "--lease-id", leaseID, "--consumer-id", "principal:runner:1", "--role-kind", "runner", "--scope", "stage:alpha", "--reason", "operator request"}, strings.NewReader(""), stdout, stderr); err != nil {
		t.Fatalf("lease-revoke returned error: %v", err)
	}
	_, err = runLeaseRetrieveToPipe(stateRoot, leaseID, "principal:runner:1", "runner", "stage:alpha", stderr)
	if err == nil {
		t.Fatal("lease-retrieve expected deny after revoke")
	}
}

func TestRevocationPersistenceRecoveryFailClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("lease-retrieve output-fd path is non-windows")
	}
	stateRoot := canonicalTempDir(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"import-secret", "--state-root", stateRoot, "--secret-ref", "secrets/prod/api"}, strings.NewReader("token-xyz"), stdout, stderr); err != nil {
		t.Fatalf("import-secret returned error: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"lease-issue", "--state-root", stateRoot, "--secret-ref", "secrets/prod/api", "--consumer-id", "principal:runner:2", "--role-kind", "runner", "--scope", "stage:beta"}, strings.NewReader(""), stdout, stderr); err != nil {
		t.Fatalf("lease-issue returned error: %v", err)
	}
	issued := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &issued); err != nil {
		t.Fatalf("json unmarshal returned error: %v", err)
	}
	leaseID, _ := issued["lease_id"].(string)
	stdout.Reset()
	if err := run([]string{"lease-revoke", "--state-root", stateRoot, "--lease-id", leaseID, "--consumer-id", "principal:runner:2", "--role-kind", "runner", "--scope", "stage:beta"}, strings.NewReader(""), stdout, stderr); err != nil {
		t.Fatalf("lease-revoke returned error: %v", err)
	}
	_, err := runLeaseRetrieveToPipe(stateRoot, leaseID, "principal:runner:2", "runner", "stage:beta", stderr)
	if err == nil {
		t.Fatal("expected deny after restart-safe revoke")
	}
	if err := os.WriteFile(filepath.Join(stateRoot, "state.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	err = run([]string{"lease-issue", "--state-root", stateRoot, "--secret-ref", "secrets/prod/api", "--consumer-id", "principal:runner:2", "--role-kind", "runner", "--scope", "stage:beta"}, strings.NewReader(""), &bytes.Buffer{}, stderr)
	if err == nil {
		t.Fatal("expected fail-closed recovery error")
	}
}

func TestOnboardingFDMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fd onboarding is non-windows only")
	}
	stateRoot := canonicalTempDir(t)
	filePath := filepath.Join(t.TempDir(), "secret.bin")
	if err := os.WriteFile(filePath, []byte("fd-secret"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer f.Close()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = run([]string{"import-secret", "--state-root", stateRoot, "--secret-ref", "secrets/prod/fd", "--fd", strconv.Itoa(int(f.Fd()))}, strings.NewReader("unused"), stdout, stderr)
	if err != nil {
		t.Fatalf("fd import returned error: %v", err)
	}
}

func TestValidateSignRequestUsageError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-sign-request"}, strings.NewReader(""), stdout, stderr)
	if err == nil {
		t.Fatal("run expected usage error")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestValidateSignRequestRejectsDirectoryPath(t *testing.T) {
	dirPath := canonicalTempDir(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-sign-request", "--file", dirPath}, strings.NewReader(""), stdout, stderr)
	if err == nil {
		t.Fatal("run expected error for directory --file")
	}
	if err.Error() != "--file path must point to a regular file" {
		t.Fatalf("error = %q, want regular-file message", err.Error())
	}
}

func TestValidateSignRequestRejectsSymlinkPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on typical Windows hosts")
	}
	testRoot := canonicalTempDir(t)
	targetPath := filepath.Join(testRoot, "request.json")
	if err := os.WriteFile(targetPath, []byte(`{
  "logical_purpose": "approval_authority",
  "logical_scope": "user",
  "key_protection_posture": "os_keystore",
  "identity_binding_posture": "attested",
  "presence_mode": "os_confirmation"
}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	linkPath := filepath.Join(testRoot, "request-link.json")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-sign-request", "--file", linkPath}, strings.NewReader(""), stdout, stderr)
	if err == nil {
		t.Fatal("run expected error for symlink --file")
	}
	if err.Error() != "--file path must not be a symlink" {
		t.Fatalf("error = %q, want symlink message", err.Error())
	}
}

func TestValidateSignRequestRejectsSymlinkInParentPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on typical Windows hosts")
	}
	parent := canonicalTempDir(t)
	targetDir := filepath.Join(parent, "target")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	requestPath := filepath.Join(targetDir, "request.json")
	if err := os.WriteFile(requestPath, []byte(`{
  "logical_purpose": "approval_authority",
  "logical_scope": "user",
  "key_protection_posture": "os_keystore",
  "identity_binding_posture": "attested",
  "presence_mode": "os_confirmation"
}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	symlinkParent := filepath.Join(parent, "linked")
	if err := os.Symlink(targetDir, symlinkParent); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-sign-request", "--file", filepath.Join(symlinkParent, "request.json")}, strings.NewReader(""), stdout, stderr)
	if err == nil {
		t.Fatal("run expected error for symlink parent in --file path")
	}
	if err.Error() != "--file path must not contain symlink path components" {
		t.Fatalf("error = %q, want symlink-parent message", err.Error())
	}
}

func TestRetrievalBindingChecks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("lease-retrieve output-fd path is non-windows")
	}
	stateRoot := canonicalTempDir(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"import-secret", "--state-root", stateRoot, "--secret-ref", "secrets/prod/bind"}, strings.NewReader("bind-secret"), stdout, stderr); err != nil {
		t.Fatalf("import-secret returned error: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"lease-issue", "--state-root", stateRoot, "--secret-ref", "secrets/prod/bind", "--consumer-id", "principal:runner:3", "--role-kind", "runner", "--scope", "stage:gamma"}, strings.NewReader(""), stdout, stderr); err != nil {
		t.Fatalf("lease-issue returned error: %v", err)
	}
	issued := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &issued); err != nil {
		t.Fatalf("json unmarshal returned error: %v", err)
	}
	leaseID, _ := issued["lease_id"].(string)
	bad := []struct {
		name string
		id   string
		role string
		scp  string
	}{
		{name: "consumer mismatch", id: "principal:runner:other", role: "runner", scp: "stage:gamma"},
		{name: "role mismatch", id: "principal:runner:3", role: "other", scp: "stage:gamma"},
		{name: "scope mismatch", id: "principal:runner:3", role: "runner", scp: "stage:other"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runLeaseRetrieveToPipe(stateRoot, leaseID, tc.id, tc.role, tc.scp, stderr)
			if err == nil {
				t.Fatalf("expected access deny")
			}
		})
	}
}

func runLeaseRetrieveToPipe(stateRoot, leaseID, consumerID, roleKind, scope string, stderr *bytes.Buffer) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	stdout := &bytes.Buffer{}
	runErr := run([]string{"lease-retrieve", "--state-root", stateRoot, "--lease-id", leaseID, "--consumer-id", consumerID, "--role-kind", roleKind, "--scope", scope, "--output-fd", strconv.Itoa(int(w.Fd()))}, strings.NewReader(""), stdout, stderr)
	_ = w.Close()
	b, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		return "", readErr
	}
	return string(b), runErr
}

func canonicalTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) returned error: %v", dir, err)
	}
	return resolved
}
