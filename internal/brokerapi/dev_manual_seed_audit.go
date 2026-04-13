package brokerapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func seedDevManualAuditLedger(root string) (string, error) {
	validatedRoot, err := ensureDevManualLedgerDirs(root)
	if err != nil {
		return "", err
	}
	recordDigest, canonicalEnvelope, err := devManualAuditEnvelopeAndDigest()
	if err != nil {
		return "", err
	}
	if err := writeDevManualSegments(validatedRoot, recordDigest, canonicalEnvelope); err != nil {
		return "", err
	}
	if err := writeDevManualSeal(validatedRoot, recordDigest); err != nil {
		return "", err
	}
	ledger, err := auditd.Open(validatedRoot)
	if err != nil {
		return "", err
	}
	if err := ledger.ConfigureVerificationInputs(auditd.VerificationConfiguration{VerifierRecords: []trustpolicy.VerifierRecord{devManualVerifierRecord()}, EventContractCatalog: devManualEventContractCatalog()}); err != nil {
		return "", err
	}
	if _, err := ledger.BuildIndex(); err != nil {
		return "", err
	}
	if _, err := ledger.PersistVerificationReport(devManualVerificationReport()); err != nil {
		return "", err
	}
	if err := writeDevManualSeedMarker(validatedRoot); err != nil {
		return "", err
	}
	return recordDigestIdentity(recordDigest)
}

func ensureDevManualLedgerDirs(root string) (string, error) {
	validatedRoot, err := ensureDevManualSeedLedgerAllowed(root)
	if err != nil {
		return "", err
	}
	paths := []string{
		filepath.Join(validatedRoot, "segments"),
		filepath.Join(validatedRoot, "sidecar", "segment-seals"),
		filepath.Join(validatedRoot, "sidecar", "verification-reports"),
		filepath.Join(validatedRoot, "contracts"),
	}
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err := os.MkdirAll(path, 0o700); err != nil {
			return "", err
		}
	}
	return validatedRoot, nil
}

func devManualLedgerSeedMarkerPath(root string) string {
	return filepath.Join(root, "contracts", "dev-manual-seed.marker")
}

func writeDevManualSeedMarker(root string) error {
	return os.WriteFile(devManualLedgerSeedMarkerPath(root), []byte(devManualSeedProfile+"\n"), 0o600)
}

func devManualAuditEnvelopeAndDigest() (trustpolicy.Digest, []byte, error) {
	eventPayload := devManualAuditEventPayload()
	eventPayloadHash := sha256.Sum256(mustDevManualCanonicalJSON(eventPayload))
	event := devManualAuditEvent(eventPayload, eventPayloadHash)
	envelope := trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.AuditEventSchemaID,
		PayloadSchemaVersion: trustpolicy.AuditEventSchemaVersion,
		Payload:              mustDevManualJSON(event),
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: strings.Repeat("a", 64),
			Signature:  base64.StdEncoding.EncodeToString([]byte("sig")),
		},
	}
	canonicalEnvelope := mustDevManualCanonicalJSON(envelope)
	recordSum := sha256.Sum256(canonicalEnvelope)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(recordSum[:])}, canonicalEnvelope, nil
}

func devManualAuditEventPayload() map[string]any {
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
	}
}

func devManualAuditEvent(eventPayload map[string]any, eventPayloadHash [32]byte) map[string]any {
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
		"protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
		"scope":                         map[string]any{"workspace_id": devManualSeedWorkspaceID, "run_id": devManualSeedRunID, "stage_id": devManualSeedStageID},
		"correlation":                   map[string]any{"session_id": devManualSeedSessionID, "operation_id": "op-manual-1"},
		"subject_ref":                   map[string]any{"object_family": "isolate_binding", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)}, "ref_role": "binding_target"},
		"cause_refs":                    []any{map[string]any{"object_family": "audit_event", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "ref_role": "session_cause"}},
		"related_refs":                  []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)}, "ref_role": "binding"}},
		"signer_evidence_refs":          []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "ref_role": "admissibility"}},
	}
}

func writeDevManualSegments(root string, recordDigest trustpolicy.Digest, canonicalEnvelope []byte) error {
	sealed := trustpolicy.AuditSegmentFilePayload{
		SchemaID:      "runecode.protocol.v0.AuditSegmentFile",
		SchemaVersion: "0.1.0",
		Header: trustpolicy.AuditSegmentHeader{
			Format:       "audit_segment_framed_v1",
			SegmentID:    "segment-000001",
			SegmentState: trustpolicy.AuditSegmentStateSealed,
			CreatedAt:    "2026-03-13T12:00:00Z",
			Writer:       "auditd",
		},
		Frames:          []trustpolicy.AuditSegmentRecordFrame{{RecordDigest: recordDigest, ByteLength: int64(len(canonicalEnvelope)), CanonicalSignedEnvelopeBytes: base64.StdEncoding.EncodeToString(canonicalEnvelope)}},
		LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateSealed, MarkedAt: "2026-03-13T12:20:00Z"},
	}
	open := trustpolicy.AuditSegmentFilePayload{
		SchemaID:      "runecode.protocol.v0.AuditSegmentFile",
		SchemaVersion: "0.1.0",
		Header: trustpolicy.AuditSegmentHeader{
			Format:       "audit_segment_framed_v1",
			SegmentID:    "segment-000002",
			SegmentState: trustpolicy.AuditSegmentStateOpen,
			CreatedAt:    "2026-03-13T12:21:00Z",
			Writer:       "auditd",
		},
		Frames:          []trustpolicy.AuditSegmentRecordFrame{},
		LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateOpen, MarkedAt: "2026-03-13T12:21:00Z"},
	}
	if err := writeDevManualCanonicalJSON(filepath.Join(root, "segments", "segment-000001.json"), sealed); err != nil {
		return err
	}
	return writeDevManualCanonicalJSON(filepath.Join(root, "segments", "segment-000002.json"), open)
}

func writeDevManualSeal(root string, recordDigest trustpolicy.Digest) error {
	sealEnvelope := devManualSealEnvelope(recordDigest)
	sealDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(sealEnvelope)
	if err != nil {
		return err
	}
	sealID, err := sealDigest.Identity()
	if err != nil {
		return err
	}
	return writeDevManualCanonicalJSON(filepath.Join(root, "sidecar", "segment-seals", strings.TrimPrefix(sealID, "sha256:")+".json"), sealEnvelope)
}

func devManualSealEnvelope(recordDigest trustpolicy.Digest) trustpolicy.SignedObjectEnvelope {
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.AuditSegmentSealSchemaID,
		PayloadSchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion,
		Payload:              mustDevManualJSON(devManualSealPayload(recordDigest)),
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString([]byte("sig"))},
	}
}

func devManualSealPayload(recordDigest trustpolicy.Digest) trustpolicy.AuditSegmentSealPayload {
	return trustpolicy.AuditSegmentSealPayload{
		SchemaID:                   trustpolicy.AuditSegmentSealSchemaID,
		SchemaVersion:              trustpolicy.AuditSegmentSealSchemaVersion,
		SegmentID:                  "segment-000001",
		SealedAfterState:           trustpolicy.AuditSegmentStateOpen,
		SegmentState:               trustpolicy.AuditSegmentStateSealed,
		SegmentCut:                 trustpolicy.AuditSegmentCutWindowPolicy{OwnershipScope: trustpolicy.AuditSegmentOwnershipScopeInstanceGlobal, MaxSegmentBytes: 2048, CutTrigger: trustpolicy.AuditSegmentCutTriggerSizeWindow},
		EventCount:                 1,
		FirstRecordDigest:          recordDigest,
		LastRecordDigest:           recordDigest,
		MerkleProfile:              trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1,
		MerkleRoot:                 recordDigest,
		SegmentFileHashScope:       trustpolicy.AuditSegmentFileHashScopeRawFramedV1,
		SegmentFileHash:            recordDigest,
		SealChainIndex:             0,
		AnchoringSubject:           trustpolicy.AuditSegmentAnchoringSubjectSeal,
		SealedAt:                   "2026-03-13T12:20:00Z",
		ProtocolBundleManifestHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)},
		SealReason:                 "size_threshold",
	}
}

func writeDevManualCanonicalJSON(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return err
	}
	return os.WriteFile(path, canonical, 0o600)
}

func mustDevManualJSON(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return b
}

func mustDevManualCanonicalJSON(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		panic(err)
	}
	return canonical
}

func recordDigestIdentity(digest trustpolicy.Digest) (string, error) {
	identity, err := digest.Identity()
	if err != nil {
		return "", err
	}
	return identity, nil
}
