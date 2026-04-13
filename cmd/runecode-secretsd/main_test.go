package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateSignRequestCLI(t *testing.T) {
	requestPath := filepath.Join(t.TempDir(), "request.json")
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
	err := run([]string{"validate-sign-request", "--file", requestPath}, stdout, stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stdout.String() != "valid\n" {
		t.Fatalf("stdout = %q, want valid", stdout.String())
	}
}

func TestValidateSignRequestUsageError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-sign-request"}, stdout, stderr)
	if err == nil {
		t.Fatal("run expected usage error")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestValidateSignRequestRejectsDirectoryPath(t *testing.T) {
	dirPath := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-sign-request", "--file", dirPath}, stdout, stderr)
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
	targetPath := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(targetPath, []byte(`{
  "logical_purpose": "approval_authority",
  "logical_scope": "user",
  "key_protection_posture": "os_keystore",
  "identity_binding_posture": "attested",
  "presence_mode": "os_confirmation"
}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	linkPath := filepath.Join(t.TempDir(), "request-link.json")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-sign-request", "--file", linkPath}, stdout, stderr)
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
	parent := t.TempDir()
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
	err := run([]string{"validate-sign-request", "--file", filepath.Join(symlinkParent, "request.json")}, stdout, stderr)
	if err == nil {
		t.Fatal("run expected error for symlink parent in --file path")
	}
	if err.Error() != "--file path must not contain symlink path components" {
		t.Fatalf("error = %q, want symlink-parent message", err.Error())
	}
}
