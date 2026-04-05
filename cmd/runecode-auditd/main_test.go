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
