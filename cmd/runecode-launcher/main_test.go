package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateIsolateBindingCLI(t *testing.T) {
	bindingPath := filepath.Join(t.TempDir(), "binding.json")
	if err := os.WriteFile(bindingPath, []byte(`{
  "run_id": "run-1",
  "isolate_id": "isolate-1",
  "session_id": "session-1",
  "session_nonce": "nonce-0123456789abcdef",
  "provisioning_mode": "tofu",
  "image_digest": {"hash_alg": "sha256", "hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
  "active_manifest_hash": {"hash_alg": "sha256", "hash": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "handshake_transcript_hash": {"hash_alg": "sha256", "hash": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
  "key_id": "key_sha256",
  "key_id_value": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
  "identity_binding_posture": "tofu"
}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-isolate-binding", "--file", bindingPath}, stdout, stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stdout.String() != "valid\n" {
		t.Fatalf("stdout = %q, want valid", stdout.String())
	}
}

func TestValidateIsolateBindingUsageError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-isolate-binding"}, stdout, stderr)
	if err == nil {
		t.Fatal("run expected usage error")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestServeOnceCLI(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"serve", "--once"}, stdout, stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stdout.String() != "launcher service started and stopped\n" {
		t.Fatalf("stdout = %q, want launcher service started and stopped", stdout.String())
	}
}

func TestServeUsageError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"serve", "--bad-flag"}, stdout, stderr)
	if err == nil {
		t.Fatal("run expected usage error")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}
