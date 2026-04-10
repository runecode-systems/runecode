package trustpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func decodeAuditEventPayload(payload json.RawMessage) (AuditEventPayload, error) {
	event := AuditEventPayload{}
	if err := json.Unmarshal(payload, &event); err != nil {
		return AuditEventPayload{}, fmt.Errorf("decode audit event payload: %w", err)
	}
	return event, nil
}

func validateAuditEventPayloadShape(event AuditEventPayload) error {
	if err := validateAuditEventPayloadRequiredFields(event); err != nil {
		return err
	}
	if err := validateAuditEventPayloadDigestFields(event); err != nil {
		return err
	}
	if err := validateTypedAuditEventPayload(event); err != nil {
		return err
	}
	if event.OccurredAt == "" {
		return fmt.Errorf("occurred_at is required")
	}
	if _, err := time.Parse(time.RFC3339, event.OccurredAt); err != nil {
		return fmt.Errorf("invalid occurred_at: %w", err)
	}
	return nil
}

func validateTypedAuditEventPayload(event AuditEventPayload) error {
	switch event.EventPayloadSchemaID {
	case IsolateSessionStartedPayloadSchemaID:
		payload := IsolateSessionStartedPayload{}
		if err := json.Unmarshal(event.EventPayload, &payload); err != nil {
			return fmt.Errorf("decode isolate session started payload: %w", err)
		}
		if err := validateIsolateSessionStartedPayload(payload); err != nil {
			return err
		}
	case IsolateSessionBoundPayloadSchemaID:
		payload := IsolateSessionBoundPayload{}
		if err := json.Unmarshal(event.EventPayload, &payload); err != nil {
			return fmt.Errorf("decode isolate session bound payload: %w", err)
		}
		if err := validateIsolateSessionBoundPayload(payload); err != nil {
			return err
		}
	}
	return nil
}

func validateIsolateSessionStartedPayload(payload IsolateSessionStartedPayload) error {
	if err := validateIsolateSessionPayloadCommon(
		payload.SchemaID,
		IsolateSessionStartedPayloadSchemaID,
		payload.SchemaVersion,
		IsolateSessionStartedPayloadSchemaVersion,
		payload.RunID,
		payload.IsolateID,
		payload.SessionID,
		"isolate session started payload",
		payload.BackendKind,
		payload.IsolationAssuranceLevel,
		payload.ProvisioningPosture,
	); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.LaunchContextDigest, "launch_context_digest"); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.HandshakeTranscriptHash, "handshake_transcript_hash"); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.LaunchReceiptDigest, "launch_receipt_digest"); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.RuntimeImageDescriptorDigest, "runtime_image_descriptor_digest"); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.AppliedHardeningPostureDigest, "applied_hardening_posture_digest"); err != nil {
		return err
	}
	return validateOptionalAttestationEvidenceDigest(payload.AttestationEvidenceDigest)
}

func validateIsolateSessionBoundPayload(payload IsolateSessionBoundPayload) error {
	if err := validateIsolateSessionPayloadCommon(
		payload.SchemaID,
		IsolateSessionBoundPayloadSchemaID,
		payload.SchemaVersion,
		IsolateSessionBoundPayloadSchemaVersion,
		payload.RunID,
		payload.IsolateID,
		payload.SessionID,
		"isolate session bound payload",
		payload.BackendKind,
		payload.IsolationAssuranceLevel,
		payload.ProvisioningPosture,
	); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.LaunchContextDigest, "launch_context_digest"); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.HandshakeTranscriptHash, "handshake_transcript_hash"); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.SessionBindingDigest, "session_binding_digest"); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.RuntimeImageDescriptorDigest, "runtime_image_descriptor_digest"); err != nil {
		return err
	}
	if err := requireDigestIdentityString(payload.AppliedHardeningPostureDigest, "applied_hardening_posture_digest"); err != nil {
		return err
	}
	return validateOptionalAttestationEvidenceDigest(payload.AttestationEvidenceDigest)
}

func validateIsolateSessionPayloadCommon(schemaID string, wantSchemaID string, schemaVersion string, wantSchemaVersion string, runID string, isolateID string, sessionID string, payloadName string, backendKind string, isolationAssuranceLevel string, provisioningPosture string) error {
	if schemaID != wantSchemaID {
		return fmt.Errorf("%s schema_id must be %q", payloadName, wantSchemaID)
	}
	if schemaVersion != wantSchemaVersion {
		return fmt.Errorf("%s schema_version must be %q", payloadName, wantSchemaVersion)
	}
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(isolateID) == "" || strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("%s requires run_id, isolate_id, and session_id", payloadName)
	}
	if !containsString([]string{"microvm", "container", "unknown"}, backendKind) {
		return fmt.Errorf("%s backend_kind %q is unsupported", payloadName, backendKind)
	}
	if !containsString([]string{"isolated", "degraded", "unknown"}, isolationAssuranceLevel) {
		return fmt.Errorf("%s isolation_assurance_level %q is unsupported", payloadName, isolationAssuranceLevel)
	}
	if !containsString([]string{"tofu", "attested", "unknown"}, provisioningPosture) {
		return fmt.Errorf("%s provisioning_posture %q is unsupported", payloadName, provisioningPosture)
	}
	return nil
}

func validateOptionalAttestationEvidenceDigest(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return requireDigestIdentityString(value, "attestation_evidence_digest")
}

func requireDigestIdentityString(value string, fieldName string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("%s must be digest identity sha256:<64 lowercase hex>", fieldName)
	}
	if _, err := (Digest{HashAlg: parts[0], Hash: parts[1]}).Identity(); err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}
	return nil
}

func validateAuditEventPayloadRequiredFields(event AuditEventPayload) error {
	if event.SchemaID != AuditEventSchemaID {
		return fmt.Errorf("unexpected audit event schema_id %q", event.SchemaID)
	}
	if event.SchemaVersion != AuditEventSchemaVersion {
		return fmt.Errorf("unexpected audit event schema_version %q", event.SchemaVersion)
	}
	if event.AuditEventType == "" || event.EmitterStreamID == "" || event.Seq < 1 {
		return fmt.Errorf("audit event requires audit_event_type, emitter_stream_id, and seq >= 1")
	}
	if event.EventPayloadSchemaID == "" {
		return fmt.Errorf("audit event requires event_payload_schema_id")
	}
	if len(event.EventPayload) == 0 {
		return fmt.Errorf("audit event requires event_payload")
	}
	return nil
}

func validateAuditEventPayloadDigestFields(event AuditEventPayload) error {
	if _, err := event.EventPayloadHash.Identity(); err != nil {
		return fmt.Errorf("event_payload_hash: %w", err)
	}
	if _, err := event.ProtocolBundleManifestHash.Identity(); err != nil {
		return fmt.Errorf("protocol_bundle_manifest_hash: %w", err)
	}
	if err := validateOptionalDigestField(event.PreviousEventHash, "previous_event_hash"); err != nil {
		return err
	}
	if err := validateOptionalDigestField(event.ActiveRoleManifestHash, "active_role_manifest_hash"); err != nil {
		return err
	}
	if err := validateOptionalDigestField(event.ActiveCapabilityManifestHash, "active_capability_manifest_hash"); err != nil {
		return err
	}
	return nil
}

func validateOptionalDigestField(value *Digest, fieldName string) error {
	if value == nil {
		return nil
	}
	if _, err := value.Identity(); err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}
	return nil
}

func validateAuditEventPayloadHash(event AuditEventPayload) error {
	canonicalPayload, err := jsoncanonicalizer.Transform(event.EventPayload)
	if err != nil {
		return fmt.Errorf("canonicalize event_payload: %w", err)
	}
	sum := sha256.Sum256(canonicalPayload)
	computed := Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}
	computedIdentity, _ := computed.Identity()
	expectedIdentity, err := event.EventPayloadHash.Identity()
	if err != nil {
		return fmt.Errorf("event_payload_hash: %w", err)
	}
	if computedIdentity != expectedIdentity {
		return fmt.Errorf("event_payload_hash mismatch: got %q want %q", computedIdentity, expectedIdentity)
	}
	return nil
}
