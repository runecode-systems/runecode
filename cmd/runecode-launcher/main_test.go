package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
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

func TestHelloWorldLaunchSpecValidates(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("microvm/kvm launch spec validation is linux-only in MVP")
	}
	image, err := helloWorldRuntimeImage(t.TempDir())
	if err != nil {
		if err.Error() == "prepare hello-world boot assets: no readable host kernel image found" {
			t.Skip("hello-world runtime image requires readable host kernel image")
		}
		t.Fatalf("helloWorldRuntimeImage returned error: %v", err)
	}
	spec := helloWorldLaunchSpec("run-test", image)
	if err := spec.Validate(); err != nil {
		t.Fatalf("helloWorldLaunchSpec Validate returned error: %v", err)
	}
}

func TestImportRuntimeVerifierAuthorityStateCLI(t *testing.T) {
	workRoot := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "authority-state.json")
	records := []trustpolicy.VerifierRecord{runtimeImageVerifierRecordForCLITests()}
	builtinRevision := uint64(1)
	entryDigest := verifierSetDigestForTests(records)
	state := map[string]any{
		"schema_id":      "runecode.launcher.runtime-verifier-authority-state",
		"schema_version": "0.1.0",
		"generation": map[string]any{
			"revision":          builtinRevision + 1,
			"previous_revision": builtinRevision,
			"changed_at":        time.Now().UTC().Format(time.RFC3339),
		},
		"merge_mode": "replace",
		"authorities_by_kind": map[string]any{
			"runtime_image": []any{map[string]any{
				"verifier_set_ref": entryDigest,
				"records":          records,
				"status":           "active",
				"source":           "imported",
				"changed_at":       time.Now().UTC().Format(time.RFC3339),
			}},
		},
	}
	state["state_digest"] = authorityStateDigestForTests(state)
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal(state) returned error: %v", err)
	}
	if err := os.WriteFile(statePath, raw, 0o600); err != nil {
		t.Fatalf("WriteFile(statePath) returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	if err := run([]string{"import-runtime-verifier-authority-state", "--file", statePath, "--work-root", workRoot}, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("runtime verifier authority state imported\n")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	parts := bytes.Split(bytes.TrimSpace(stdout.Bytes()), []byte("\n"))
	if len(parts) != 2 {
		t.Fatalf("stdout line count = %d, want 2", len(parts))
	}
	receipt := map[string]any{}
	if err := json.Unmarshal(parts[1], &receipt); err != nil {
		t.Fatalf("Unmarshal(receipt) returned error: %v", err)
	}
	if receipt["schema_id"] != "runecode.launcher.runtime-verifier-authority-state-receipt" {
		t.Fatalf("receipt schema_id = %v", receipt["schema_id"])
	}
}

func runtimeImageVerifierRecordForCLITests() trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:      trustpolicy.VerifierSchemaID,
		SchemaVersion: trustpolicy.VerifierSchemaVersion,
		KeyID:         trustpolicy.KeyIDProfile,
		KeyIDValue:    "10ba682c8ad13513971e8b56881aab8bd702bb807796eca81932c735a94d6e6d",
		Alg:           "ed25519",
		PublicKey: trustpolicy.PublicKey{
			Encoding: "base64",
			Value:    "0EqyMnQrtKs6E2i9RhXk5tAiSrcaAWuvhSCjMsl3hzc=",
		},
		LogicalPurpose:         "runtime_image_signing",
		LogicalScope:           "publisher",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.1.0", ActorKind: "service", PrincipalID: "runecode-runtime-image-publisher", InstanceID: "builtin"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-04-29T00:00:00Z",
		Status:                 "active",
	}
}

func TestShowRuntimeVerifierAuthorityStateCLI(t *testing.T) {
	stdout := &bytes.Buffer{}
	if err := run([]string{"show-runtime-verifier-authority-state"}, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	state := map[string]any{}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &state); err != nil {
		t.Fatalf("Unmarshal(state) returned error: %v", err)
	}
	if state["schema_id"] != "runecode.launcher.runtime-verifier-authority-state" {
		t.Fatalf("schema_id = %v", state["schema_id"])
	}
}

func TestExportRuntimeVerifierAuthorityBaselineCLI(t *testing.T) {
	stdout := &bytes.Buffer{}
	if err := run([]string{"export-runtime-verifier-authority-baseline"}, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	state := map[string]any{}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &state); err != nil {
		t.Fatalf("Unmarshal(state) returned error: %v", err)
	}
	if state["schema_id"] != "runecode.launcher.runtime-verifier-authority-state" {
		t.Fatalf("schema_id = %v", state["schema_id"])
	}
}

func verifierSetDigestForTests(records []trustpolicy.VerifierRecord) string {
	b, err := json.Marshal(records)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func authorityStateDigestForTests(state map[string]any) string {
	copy := map[string]any{}
	for k, v := range state {
		copy[k] = v
	}
	copy["state_digest"] = ""
	b, err := json.Marshal(copy)
	if err != nil {
		panic(err)
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}
