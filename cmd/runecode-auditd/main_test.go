package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
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

func TestValidateAdmissionUsageError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"validate-admission"}, stdout, stderr)
	if err == nil {
		t.Fatal("run expected usage error")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestValidateAdmissionCLI(t *testing.T) {
	admissionPath := filepath.Join(t.TempDir(), "admission.json")
	request := validAuditAdmissionRequestFixture(t)
	requestBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if err := os.WriteFile(admissionPath, requestBytes, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = run([]string{"validate-admission", "--file", admissionPath}, stdout, stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stdout.String() != "valid\n" {
		t.Fatalf("stdout = %q, want valid", stdout.String())
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

func TestAppendEventRejectsInvalidRequestWithoutWritingContracts(t *testing.T) {
	root := t.TempDir()
	requestPath := filepath.Join(t.TempDir(), "admission.json")
	if err := os.WriteFile(requestPath, []byte(`{"checks":{}}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"append-event", "--file", requestPath, "--ledger-root", root}, stdout, stderr)
	if err == nil {
		t.Fatal("run returned nil error, want validation error")
	}
	if _, statErr := os.Stat(filepath.Join(root, "contracts")); !os.IsNotExist(statErr) {
		t.Fatalf("contracts directory should not be created for invalid request, stat err = %v", statErr)
	}
}

func validAuditAdmissionRequestFixture(t *testing.T) trustpolicy.AuditAdmissionRequest {
	t.Helper()
	publicKey, privateKey, keyIDValue := generateAuditFixtureKeyMaterial(t)
	payloadBytes := buildAuditAdmissionEventPayloadBytes(t)
	signature := signAuditAdmissionPayload(t, privateKey, payloadBytes)
	return trustpolicy.AuditAdmissionRequest{
		Checks: trustpolicy.AuditAdmissionChecks{
			SchemaValidation:         true,
			EventContractValidation:  true,
			SignerEvidenceValidation: true,
			DetachedSignatureVerify:  true,
		},
		Envelope: trustpolicy.SignedObjectEnvelope{
			SchemaID:             trustpolicy.EnvelopeSchemaID,
			SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
			PayloadSchemaID:      trustpolicy.AuditEventSchemaID,
			PayloadSchemaVersion: trustpolicy.AuditEventSchemaVersion,
			Payload:              payloadBytes,
			SignatureInput:       trustpolicy.SignatureInputProfile,
			Signature: trustpolicy.SignatureBlock{
				Alg:        "ed25519",
				KeyID:      trustpolicy.KeyIDProfile,
				KeyIDValue: keyIDValue,
				Signature:  base64.StdEncoding.EncodeToString(signature),
			},
		},
		VerifierRecords:      []trustpolicy.VerifierRecord{buildAuditAdmissionVerifierRecord(publicKey, keyIDValue)},
		EventContractCatalog: buildAuditEventContractCatalogFixture(),
		SignerEvidence:       []trustpolicy.AuditSignerEvidenceReference{buildSignerEvidenceReferenceFixture(keyIDValue)},
	}
}

func generateAuditFixtureKeyMaterial(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	keyID := sha256.Sum256(publicKey)
	return publicKey, privateKey, hex.EncodeToString(keyID[:])
}

func buildAuditAdmissionEventPayloadBytes(t *testing.T) json.RawMessage {
	t.Helper()
	eventPayload := map[string]any{"session_id": "session-1"}
	eventPayloadHash := hashCanonicalJSONFixture(t, eventPayload)
	payload := map[string]any{
		"schema_id":                     trustpolicy.AuditEventSchemaID,
		"schema_version":                trustpolicy.AuditEventSchemaVersion,
		"audit_event_type":              "isolate_session_bound",
		"emitter_stream_id":             "auditd-stream-1",
		"seq":                           1,
		"occurred_at":                   "2026-03-13T12:15:00Z",
		"principal":                     map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"event_payload_schema_id":       "runecode.protocol.audit.payload.isolate-session-bound.v0",
		"event_payload":                 eventPayload,
		"event_payload_hash":            map[string]any{"hash_alg": "sha256", "hash": eventPayloadHash},
		"protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
		"scope":                         map[string]any{"workspace_id": "workspace-1", "run_id": "run-1", "stage_id": "stage-1"},
		"correlation":                   map[string]any{"session_id": "session-1", "operation_id": "op-1"},
		"subject_ref":                   map[string]any{"object_family": "isolate_binding", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)}, "ref_role": "binding_target"},
		"cause_refs":                    []any{map[string]any{"object_family": "audit_event", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "ref_role": "session_cause"}},
		"related_refs":                  []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)}, "ref_role": "binding"}},
		"signer_evidence_refs":          []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "ref_role": "admissibility"}},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	return payloadBytes
}

func signAuditAdmissionPayload(t *testing.T, privateKey ed25519.PrivateKey, payload json.RawMessage) []byte {
	t.Helper()
	canonicalPayload, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		t.Fatalf("Transform payload returned error: %v", err)
	}
	return ed25519.Sign(privateKey, canonicalPayload)
}

func buildAuditEventContractCatalogFixture() trustpolicy.AuditEventContractCatalog {
	return trustpolicy.AuditEventContractCatalog{
		SchemaID:      trustpolicy.AuditEventContractCatalogSchemaID,
		SchemaVersion: trustpolicy.AuditEventContractCatalogSchemaVersion,
		CatalogID:     "audit_event_contract_v0",
		Entries: []trustpolicy.AuditEventContractCatalogEntry{{
			AuditEventType:                "isolate_session_bound",
			AllowedPayloadSchemaIDs:       []string{"runecode.protocol.audit.payload.isolate-session-bound.v0"},
			AllowedSignerPurposes:         []string{"isolate_session_identity"},
			AllowedSignerScopes:           []string{"session"},
			RequiredScopeFields:           []string{"workspace_id", "run_id", "stage_id"},
			RequiredCorrelationFields:     []string{"session_id", "operation_id"},
			RequireSubjectRef:             true,
			AllowedSubjectRefRoles:        []string{"binding_target"},
			AllowedCauseRefRoles:          []string{"session_cause"},
			AllowedRelatedRefRoles:        []string{"binding", "evidence", "receipt"},
			RequireGatewayContext:         false,
			RequireSignerEvidenceRefs:     true,
			AllowedSignerEvidenceRefRoles: []string{"admissibility", "binding"},
		}},
	}
}

func buildSignerEvidenceReferenceFixture(keyIDValue string) trustpolicy.AuditSignerEvidenceReference {
	return trustpolicy.AuditSignerEvidenceReference{
		Digest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)},
		Evidence: trustpolicy.AuditSignerEvidence{
			SignerPurpose: "isolate_session_identity",
			SignerScope:   "session",
			SignerKey: trustpolicy.SignatureBlock{
				Alg:        "ed25519",
				KeyID:      trustpolicy.KeyIDProfile,
				KeyIDValue: keyIDValue,
				Signature:  "c2ln",
			},
			IsolateBinding: &trustpolicy.IsolateSessionBinding{
				RunID:                   "run-1",
				IsolateID:               "isolate-1",
				SessionID:               "session-1",
				SessionNonce:            "nonce-1",
				ProvisioningMode:        "tofu",
				ImageDigest:             trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)},
				ActiveManifestHash:      trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)},
				HandshakeTranscriptHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)},
				KeyID:                   trustpolicy.KeyIDProfile,
				KeyIDValue:              keyIDValue,
				IdentityBindingPosture:  "tofu",
			},
		},
	}
}

func hashCanonicalJSONFixture(t *testing.T, value any) string {
	t.Helper()
	valueBytes, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal value returned error: %v", err)
	}
	canonicalValue, err := jsoncanonicalizer.Transform(valueBytes)
	if err != nil {
		t.Fatalf("Transform value returned error: %v", err)
	}
	sum := sha256.Sum256(canonicalValue)
	return hex.EncodeToString(sum[:])
}

func buildAuditAdmissionVerifierRecord(publicKey ed25519.PublicKey, keyIDValue string) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "isolate_session_identity",
		LogicalScope:           "session",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 "active",
	}
}
