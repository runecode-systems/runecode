package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func seedLedgerForBrokerCommandTest(root string) error {
	if err := seedLedgerDirectoriesForBrokerTest(root); err != nil {
		return err
	}
	recordDigest, canonicalEnvelope, err := seedEventRecordForBrokerTest()
	if err != nil {
		return err
	}
	if err := seedSegmentsForBrokerTest(root, recordDigest, canonicalEnvelope); err != nil {
		return err
	}
	sealID, err := seedSealForBrokerTest(root, recordDigest)
	if err != nil {
		return err
	}
	reportID, err := seedVerificationReportForBrokerTest(root)
	if err != nil {
		return err
	}
	if err := seedStateForBrokerTest(root, sealID, reportID); err != nil {
		return err
	}
	return seedContractsForBrokerTest(root)
}

func seedLedgerDirectoriesForBrokerTest(root string) error {
	paths := []string{filepath.Join(root, "segments"), filepath.Join(root, "sidecar", "segment-seals"), filepath.Join(root, "sidecar", "verification-reports"), filepath.Join(root, "contracts")}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func seedEventRecordForBrokerTest() (trustpolicy.Digest, []byte, error) {
	eventPayload := map[string]any{
		"schema_id":                        trustpolicy.IsolateSessionBoundPayloadSchemaID,
		"schema_version":                   trustpolicy.IsolateSessionBoundPayloadSchemaVersion,
		"run_id":                           "run-1",
		"isolate_id":                       "isolate-1",
		"session_id":                       "session-1",
		"backend_kind":                     "microvm",
		"isolation_assurance_level":        "isolated",
		"provisioning_posture":             "tofu",
		"launch_context_digest":            "sha256:" + strings.Repeat("1", 64),
		"handshake_transcript_hash":        "sha256:" + strings.Repeat("2", 64),
		"session_binding_digest":           "sha256:" + strings.Repeat("3", 64),
		"runtime_image_descriptor_digest":  "sha256:" + strings.Repeat("4", 64),
		"applied_hardening_posture_digest": "sha256:" + strings.Repeat("5", 64),
	}
	eventPayloadBytes, _ := json.Marshal(eventPayload)
	canonicalEventPayload, _ := jsoncanonicalizer.Transform(eventPayloadBytes)
	eventPayloadHash := sha256.Sum256(canonicalEventPayload)
	event := map[string]any{"schema_id": trustpolicy.AuditEventSchemaID, "schema_version": trustpolicy.AuditEventSchemaVersion, "audit_event_type": "isolate_session_bound", "emitter_stream_id": "auditd-stream-1", "seq": 1, "occurred_at": "2026-03-13T12:15:00Z", "principal": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"}, "event_payload_schema_id": trustpolicy.IsolateSessionBoundPayloadSchemaID, "event_payload": eventPayload, "event_payload_hash": map[string]any{"hash_alg": "sha256", "hash": hex.EncodeToString(eventPayloadHash[:])}, "protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)}, "scope": map[string]any{"workspace_id": "workspace-1", "run_id": "run-1", "stage_id": "stage-1"}, "correlation": map[string]any{"session_id": "session-1", "operation_id": "op-1"}, "subject_ref": map[string]any{"object_family": "isolate_binding", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)}, "ref_role": "binding_target"}, "cause_refs": []any{map[string]any{"object_family": "audit_event", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "ref_role": "session_cause"}}, "related_refs": []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)}, "ref_role": "binding"}}, "signer_evidence_refs": []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "ref_role": "admissibility"}}}
	envelope := trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditEventSchemaID, PayloadSchemaVersion: trustpolicy.AuditEventSchemaVersion, Payload: mustJSONMarshalBrokerTest(event), SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString(make([]byte, 64))}}
	envelopeBytes, _ := json.Marshal(envelope)
	canonicalEnvelope, _ := jsoncanonicalizer.Transform(envelopeBytes)
	recordSum := sha256.Sum256(canonicalEnvelope)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(recordSum[:])}, canonicalEnvelope, nil
}

func seedSegmentsForBrokerTest(root string, recordDigest trustpolicy.Digest, canonicalEnvelope []byte) error {
	sealed := trustpolicy.AuditSegmentFilePayload{SchemaID: "runecode.protocol.v0.AuditSegmentFile", SchemaVersion: "0.1.0", Header: trustpolicy.AuditSegmentHeader{Format: "audit_segment_framed_v1", SegmentID: "segment-000001", SegmentState: trustpolicy.AuditSegmentStateSealed, CreatedAt: "2026-03-13T12:00:00Z", Writer: "auditd"}, Frames: []trustpolicy.AuditSegmentRecordFrame{{RecordDigest: recordDigest, ByteLength: int64(len(canonicalEnvelope)), CanonicalSignedEnvelopeBytes: base64.StdEncoding.EncodeToString(canonicalEnvelope)}}, LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateSealed, MarkedAt: "2026-03-13T12:20:00Z"}}
	open := trustpolicy.AuditSegmentFilePayload{SchemaID: "runecode.protocol.v0.AuditSegmentFile", SchemaVersion: "0.1.0", Header: trustpolicy.AuditSegmentHeader{Format: "audit_segment_framed_v1", SegmentID: "segment-000002", SegmentState: trustpolicy.AuditSegmentStateOpen, CreatedAt: "2026-03-13T12:21:00Z", Writer: "auditd"}, Frames: []trustpolicy.AuditSegmentRecordFrame{}, LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateOpen, MarkedAt: "2026-03-13T12:21:00Z"}}
	if err := writeCanonicalJSONForBrokerTest(filepath.Join(root, "segments", "segment-000001.json"), sealed); err != nil {
		return err
	}
	return writeCanonicalJSONForBrokerTest(filepath.Join(root, "segments", "segment-000002.json"), open)
}

func seedSealForBrokerTest(root string, recordDigest trustpolicy.Digest) (string, error) {
	merkleRoot, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot([]trustpolicy.Digest{recordDigest})
	if err != nil {
		return "", err
	}
	seal := trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditSegmentSealSchemaID, PayloadSchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, Payload: mustJSONMarshalBrokerTest(trustpolicy.AuditSegmentSealPayload{SchemaID: trustpolicy.AuditSegmentSealSchemaID, SchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, SegmentID: "segment-000001", SealedAfterState: trustpolicy.AuditSegmentStateOpen, SegmentState: trustpolicy.AuditSegmentStateSealed, SegmentCut: trustpolicy.AuditSegmentCutWindowPolicy{OwnershipScope: trustpolicy.AuditSegmentOwnershipScopeInstanceGlobal, MaxSegmentBytes: 2048, CutTrigger: trustpolicy.AuditSegmentCutTriggerSizeWindow}, EventCount: 1, FirstRecordDigest: recordDigest, LastRecordDigest: recordDigest, MerkleProfile: trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1, MerkleRoot: merkleRoot, SegmentFileHashScope: trustpolicy.AuditSegmentFileHashScopeRawFramedV1, SegmentFileHash: recordDigest, SealChainIndex: 0, AnchoringSubject: trustpolicy.AuditSegmentAnchoringSubjectSeal, SealedAt: "2026-03-13T12:20:00Z", ProtocolBundleManifestHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)}, SealReason: "size_threshold"}), SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString(make([]byte, 64))}}
	sealDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(seal)
	if err != nil {
		return "", err
	}
	sealID, _ := sealDigest.Identity()
	if err := writeCanonicalJSONForBrokerTest(filepath.Join(root, "sidecar", "segment-seals", strings.TrimPrefix(sealID, "sha256:")+".json"), seal); err != nil {
		return "", err
	}
	return sealID, nil
}

func seedVerificationReportForBrokerTest(root string) (string, error) {
	keyHash := strings.Repeat("c", 64)
	report := trustpolicy.AuditVerificationReportPayload{SchemaID: trustpolicy.AuditVerificationReportSchemaID, SchemaVersion: trustpolicy.AuditVerificationReportSchemaVersion, VerifiedAt: time.Now().UTC().Format(time.RFC3339), VerificationScope: trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: "segment-000001"}, CryptographicallyValid: true, HistoricallyAdmissible: true, CurrentlyDegraded: false, IntegrityStatus: trustpolicy.AuditVerificationStatusOK, AnchoringStatus: trustpolicy.AuditVerificationStatusOK, AnchoringPosture: trustpolicy.AuditVerificationAnchoringPostureLocalAnchorReceiptOnly, StoragePostureStatus: trustpolicy.AuditVerificationStatusOK, SegmentLifecycleStatus: trustpolicy.AuditVerificationStatusOK, VerifierIdentity: trustpolicy.KeyIDProfile + ":" + keyHash, TrustRootIdentities: []string{"sha256:" + keyHash}, DegradedReasons: []string{}, HardFailures: []string{}, Findings: []trustpolicy.AuditVerificationFinding{}, Summary: "ok"}
	reportCanonical, _ := jsoncanonicalizer.Transform(mustJSONMarshalBrokerTest(report))
	reportSum := sha256.Sum256(reportCanonical)
	reportDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(reportSum[:])}
	reportID, _ := reportDigest.Identity()
	if err := writeCanonicalJSONForBrokerTest(filepath.Join(root, "sidecar", "verification-reports", strings.TrimPrefix(reportID, "sha256:")+".json"), report); err != nil {
		return "", err
	}
	return reportID, nil
}

func seedStateForBrokerTest(root, sealID, reportID string) error {
	state := map[string]any{"schema_version": 1, "current_open_segment_id": "segment-000002", "next_segment_number": 3, "open_frame_count": 0, "last_seal_envelope_digest": sealID, "last_sealed_segment_id": "segment-000001", "last_verification_report_digest": reportID, "recovery_complete": true, "last_indexed_record_count": 1}
	return writeCanonicalJSONForBrokerTest(filepath.Join(root, "state.json"), state)
}

func seedContractsForBrokerTest(root string) error {
	publicKey := make([]byte, 32)
	keyID := sha256.Sum256(publicKey)
	verifier := trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: hex.EncodeToString(keyID[:]), Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "isolate_session_identity", LogicalScope: "session", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"}, KeyProtectionPosture: "os_keystore", IdentityBindingPosture: "attested", PresenceMode: "os_confirmation", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}
	if err := writeCanonicalJSONForBrokerTest(filepath.Join(root, "contracts", "verifier-records.json"), []trustpolicy.VerifierRecord{verifier}); err != nil {
		return err
	}
	catalog := trustpolicy.AuditEventContractCatalog{SchemaID: trustpolicy.AuditEventContractCatalogSchemaID, SchemaVersion: trustpolicy.AuditEventContractCatalogSchemaVersion, CatalogID: "audit_event_contract_v0", Entries: []trustpolicy.AuditEventContractCatalogEntry{{AuditEventType: "isolate_session_bound", AllowedPayloadSchemaIDs: []string{trustpolicy.IsolateSessionBoundPayloadSchemaID}, AllowedSignerPurposes: []string{"isolate_session_identity"}, AllowedSignerScopes: []string{"session"}, RequiredScopeFields: []string{"workspace_id", "run_id", "stage_id"}, RequiredCorrelationFields: []string{"session_id", "operation_id"}, RequireSubjectRef: true, AllowedSubjectRefRoles: []string{"binding_target"}, AllowedCauseRefRoles: []string{"session_cause"}, AllowedRelatedRefRoles: []string{"binding", "evidence", "receipt"}, RequireSignerEvidenceRefs: true, AllowedSignerEvidenceRefRoles: []string{"admissibility", "binding"}}}}
	return writeCanonicalJSONForBrokerTest(filepath.Join(root, "contracts", "event-contract-catalog.json"), catalog)
}

func writeCanonicalJSONForBrokerTest(path string, value any) error {
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

func mustJSONMarshalBrokerTest(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return b
}
