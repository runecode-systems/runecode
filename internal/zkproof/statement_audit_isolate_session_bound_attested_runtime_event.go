package zkproof

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func deterministicAuditRecordDigestFixtureV0(seed uint64) trustpolicy.Digest {
	seedBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(seedBytes, seed)
	sum := sha256.Sum256(append([]byte("runecode.zkproof.audit_record_digest_fixture.v0:"), seedBytes...))
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}
}

func decodeEligibleIsolateSessionBoundPayload(event trustpolicy.AuditEventPayload) (trustpolicy.IsolateSessionBoundPayload, error) {
	if err := validateEligibleIsolateSessionBoundEvent(event); err != nil {
		return trustpolicy.IsolateSessionBoundPayload{}, err
	}
	payload := trustpolicy.IsolateSessionBoundPayload{}
	if err := json.Unmarshal(event.EventPayload, &payload); err != nil {
		return trustpolicy.IsolateSessionBoundPayload{}, &FeasibilityError{Code: feasibilityCodeIneligibleAuditEvent, Message: fmt.Sprintf("decode isolate_session_bound payload: %v", err)}
	}
	if err := validateEligibleIsolateSessionBoundPayload(payload); err != nil {
		return trustpolicy.IsolateSessionBoundPayload{}, err
	}
	return payload, nil
}

func validateEligibleIsolateSessionBoundEvent(event trustpolicy.AuditEventPayload) error {
	if event.AuditEventType != "isolate_session_bound" {
		return &FeasibilityError{Code: feasibilityCodeIneligibleAuditEvent, Message: "audit_event_type must be isolate_session_bound"}
	}
	if event.EventPayloadSchemaID != trustpolicy.IsolateSessionBoundPayloadSchemaID {
		return &FeasibilityError{Code: feasibilityCodeIneligibleAuditEvent, Message: fmt.Sprintf("event_payload_schema_id must be %q", trustpolicy.IsolateSessionBoundPayloadSchemaID)}
	}
	if _, err := event.ProtocolBundleManifestHash.Identity(); err != nil {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("protocol_bundle_manifest_hash: %v", err)}
	}
	return nil
}

func validateEligibleIsolateSessionBoundPayload(payload trustpolicy.IsolateSessionBoundPayload) error {
	if payload.SchemaID != trustpolicy.IsolateSessionBoundPayloadSchemaID {
		return &FeasibilityError{Code: feasibilityCodeIneligibleAuditEvent, Message: fmt.Sprintf("event payload schema_id must be %q", trustpolicy.IsolateSessionBoundPayloadSchemaID)}
	}
	if payload.SchemaVersion != trustpolicy.IsolateSessionBoundPayloadSchemaVersion {
		return &FeasibilityError{Code: feasibilityCodeIneligibleAuditEvent, Message: fmt.Sprintf("event payload schema_version must be %q", trustpolicy.IsolateSessionBoundPayloadSchemaVersion)}
	}
	return validateEligibleIsolateSessionBoundPayloadDigests(payload)
}

func validateEligibleIsolateSessionBoundPayloadDigests(payload trustpolicy.IsolateSessionBoundPayload) error {
	for _, field := range []struct {
		name  string
		value string
	}{{"runtime_image_descriptor_digest", payload.RuntimeImageDescriptorDigest}, {"attestation_evidence_digest", payload.AttestationEvidenceDigest}, {"applied_hardening_posture_digest", payload.AppliedHardeningPostureDigest}, {"session_binding_digest", payload.SessionBindingDigest}, {"launch_context_digest", payload.LaunchContextDigest}, {"handshake_transcript_hash", payload.HandshakeTranscriptHash}} {
		if err := requireDigestIdentity(field.value, field.name); err != nil {
			return err
		}
	}
	return nil
}

func requireDigestIdentity(value string, fieldName string) error {
	_, err := parseDigestIdentity(value, fieldName)
	return err
}
