package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSignerEvidenceCLI(t *testing.T) {
	evidencePath := filepath.Join(t.TempDir(), "evidence.json")
	if err := os.WriteFile(evidencePath, []byte(`{
  "signer_purpose": "isolate_session_identity",
  "signer_scope": "session",
  "signer_key": {
    "alg": "ed25519",
    "key_id": "key_sha256",
    "key_id_value": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
    "signature": "c2ln"
  },
  "isolate_binding": {
    "run_id": "run-1",
    "isolate_id": "isolate-1",
    "session_id": "session-1",
    "session_nonce": "nonce-1",
    "provisioning_mode": "tofu",
    "image_digest": {"hash_alg": "sha256", "hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
    "active_manifest_hash": {"hash_alg": "sha256", "hash": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
    "handshake_transcript_hash": {"hash_alg": "sha256", "hash": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
    "key_id": "key_sha256",
    "key_id_value": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
    "identity_binding_posture": "tofu"
  }
}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-signer-evidence", "--file", evidencePath}, stdout, stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stdout.String() != "valid\n" {
		t.Fatalf("stdout = %q, want valid", stdout.String())
	}
}

func TestValidateSignerEvidenceUsageError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-signer-evidence"}, stdout, stderr)
	if err == nil {
		t.Fatal("run expected usage error")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestValidateRecoveryCLI(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "recovery.json")
	if err := os.WriteFile(statePath, []byte(`{
  "segment_id": "segment-0001",
  "header_state": "open",
  "lifecycle_marker_state": "open",
  "has_torn_trailing_frame": true,
  "frame_integrity_ok": true,
  "seal_integrity_ok": false
}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-recovery", "--file", statePath}, stdout, stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("truncate_open_torn_trailing_frame")) {
		t.Fatalf("stdout = %q, want truncate decision", stdout.String())
	}
}

func TestValidateStoragePostureCLI(t *testing.T) {
	posturePath := filepath.Join(t.TempDir(), "posture.json")
	if err := os.WriteFile(posturePath, []byte(`{
  "encrypted_at_rest_default": true,
  "encrypted_at_rest_effective": false,
  "dev_plaintext_override_active": true,
  "dev_plaintext_override_reason": "dev_local_filesystem_without_encryption",
  "surfaced_to_operator": true
}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-storage-posture", "--file", posturePath}, stdout, stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stdout.String() != "valid\n" {
		t.Fatalf("stdout = %q, want valid", stdout.String())
	}
}

func TestValidateReadinessCLI(t *testing.T) {
	readinessPath := filepath.Join(t.TempDir(), "readiness.json")
	if err := os.WriteFile(readinessPath, []byte(`{
  "local_only": true,
  "consumption_channel": "broker_local_api",
  "recovery_complete": true,
  "append_position_stable": true,
  "current_segment_writable": true,
  "verifier_material_available": true,
  "derived_index_caught_up": true,
  "ready": true
}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-readiness", "--file", readinessPath}, stdout, stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stdout.String() != "valid\n" {
		t.Fatalf("stdout = %q, want valid", stdout.String())
	}
}
