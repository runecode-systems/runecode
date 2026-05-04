//go:build runecode_devseed

package brokerapi

import (
	"encoding/hex"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func devManualAuditEventPayload(material devManualVerificationMaterial, profile string) map[string]any {
	return map[string]any{
		"schema_id":                        trustpolicy.IsolateSessionBoundPayloadSchemaID,
		"schema_version":                   trustpolicy.IsolateSessionBoundPayloadSchemaVersion,
		"run_id":                           devManualSeedRunID,
		"isolate_id":                       "isolate-manual-001",
		"session_id":                       devManualSeedSessionID,
		"backend_kind":                     "microvm",
		"isolation_assurance_level":        "isolated",
		"provisioning_posture":             "tofu",
		"launch_context_digest":            digestWithByte("1"),
		"handshake_transcript_hash":        digestWithByte("2"),
		"session_binding_digest":           digestWithByte("3"),
		"runtime_image_descriptor_digest":  digestWithByte("4"),
		"applied_hardening_posture_digest": digestWithByte("5"),
		"seed_profile":                     profile,
		"seed_verifier":                    material.keyIDValue,
	}
}

func devManualAuditEvent(eventPayload map[string]any, eventPayloadHash [32]byte, profile string) map[string]any {
	return map[string]any{
		"schema_id":                     trustpolicy.AuditEventSchemaID,
		"schema_version":                trustpolicy.AuditEventSchemaVersion,
		"audit_event_type":              "isolate_session_bound",
		"emitter_stream_id":             "auditd-stream-manual",
		"seq":                           1,
		"occurred_at":                   devManualSeedRecordedAtRFC3339,
		"principal":                     map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-manual"},
		"event_payload_schema_id":       trustpolicy.IsolateSessionBoundPayloadSchemaID,
		"event_payload":                 eventPayload,
		"event_payload_hash":            map[string]any{"hash_alg": "sha256", "hash": hex.EncodeToString(eventPayloadHash[:])},
		"protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": stringsRepeat("b")},
		"scope":                         map[string]any{"workspace_id": devManualSeedWorkspaceID, "run_id": devManualSeedRunID, "stage_id": devManualSeedStageID},
		"correlation":                   map[string]any{"session_id": devManualSeedSessionID, "operation_id": "op-manual-1"},
		"subject_ref":                   map[string]any{"object_family": "isolate_binding", "digest": map[string]any{"hash_alg": "sha256", "hash": stringsRepeat("c")}, "ref_role": "binding_target"},
		"cause_refs":                    []any{map[string]any{"object_family": "audit_event", "digest": map[string]any{"hash_alg": "sha256", "hash": stringsRepeat("d")}, "ref_role": "session_cause"}},
		"related_refs":                  []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": stringsRepeat("e")}, "ref_role": "binding"}},
		"signer_evidence_refs":          []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": stringsRepeat("f")}, "ref_role": "admissibility"}},
		"seed_profile":                  profile,
	}
}
