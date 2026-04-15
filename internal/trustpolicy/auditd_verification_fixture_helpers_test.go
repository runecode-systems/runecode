package trustpolicy

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func newAuditVerificationFixture(t *testing.T, verifierStatus verifierStatusFixture) auditVerificationFixture {
	t.Helper()
	request := validAuditAdmissionRequestFixture(t)
	publicKey, privateKey, keyID := generateAuditVerificationFixtureKeys(t)
	eventPayload := mustUnmarshalAuditEventPayload(t, request.Envelope.Payload)
	verificationBundle := buildVerificationFixtureSignedArtifacts(t, privateKey, keyID, eventPayload)
	verifierRecord := buildVerificationFixtureVerifierRecord(publicKey, keyID, verifierStatus)
	return auditVerificationFixture{segment: verificationBundle.segment, rawSegmentBytes: verificationBundle.rawSegmentBytes, sealEnvelope: verificationBundle.sealEnvelope, sealEnvelopeDigest: verificationBundle.sealEnvelopeDigest, verifierRecords: []VerifierRecord{verifierRecord}, eventContractCatalog: request.EventContractCatalog, signerEvidence: buildVerificationFixtureSignerEvidence(keyID), privateKey: privateKey, keyID: keyID}
}

func mustVerifyAuditEvidenceReport(t *testing.T, fixture auditVerificationFixture, receipts []SignedObjectEnvelope) AuditVerificationReportPayload {
	t.Helper()
	report, err := VerifyAuditEvidence(AuditVerificationInput{Scope: AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID}, Segment: fixture.segment, RawFramedSegmentBytes: fixture.rawSegmentBytes, SegmentSealEnvelope: fixture.sealEnvelope, ReceiptEnvelopes: receipts, VerifierRecords: fixture.verifierRecords, EventContractCatalog: fixture.eventContractCatalog, SignerEvidence: fixture.signerEvidence, Now: time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if err := ValidateAuditVerificationReportPayload(report); err != nil {
		t.Fatalf("ValidateAuditVerificationReportPayload returned error: %v", err)
	}
	return report
}

func assertMissingAnchorDegradesReport(t *testing.T, report AuditVerificationReportPayload) {
	t.Helper()
	if report.IntegrityStatus != AuditVerificationStatusOK {
		t.Fatalf("integrity_status = %q, want %q", report.IntegrityStatus, AuditVerificationStatusOK)
	}
	if report.AnchoringStatus != AuditVerificationStatusDegraded {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusDegraded)
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonAnchorReceiptMissing) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonAnchorReceiptMissing)
	}
	if len(report.HardFailures) != 0 {
		t.Fatalf("hard_failures = %v, want empty", report.HardFailures)
	}
}

func assertDerivedSummaryDegraded(t *testing.T, report AuditVerificationReportPayload) {
	t.Helper()
	summary, err := BuildDerivedRunAuditVerificationSummary(report)
	if err != nil {
		t.Fatalf("BuildDerivedRunAuditVerificationSummary returned error: %v", err)
	}
	if !summary.CurrentlyDegraded {
		t.Fatal("derived summary expected currently_degraded=true")
	}
	if summary.WarningFindingCount == 0 {
		t.Fatal("derived summary expected at least one warning finding")
	}
}

func mustUnmarshalAuditEventPayload(t *testing.T, payload json.RawMessage) map[string]any {
	t.Helper()
	var eventPayload map[string]any
	if err := json.Unmarshal(payload, &eventPayload); err != nil {
		t.Fatalf("Unmarshal event payload returned error: %v", err)
	}
	return eventPayload
}

func generateAuditVerificationFixtureKeys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	keySum := sha256.Sum256(publicKey)
	return publicKey, privateKey, hex.EncodeToString(keySum[:])
}

func buildVerificationFixtureSignedArtifacts(t *testing.T, privateKey ed25519.PrivateKey, keyID string, eventPayload map[string]any) verificationFixtureSignedArtifacts {
	t.Helper()
	_, eventCanonicalBytes, eventDigest := buildVerificationFixtureEventFrame(t, privateKey, keyID, eventPayload)
	segment, rawSegmentBytes, segmentHash, merkleRoot := buildVerificationFixtureSegment(t, eventDigest, eventCanonicalBytes)
	sealEnvelope, sealEnvelopeDigest := buildVerificationFixtureSealEnvelope(t, privateKey, keyID, segment.Header.SegmentID, eventDigest, segmentHash, merkleRoot, eventPayload)
	return verificationFixtureSignedArtifacts{segment: segment, rawSegmentBytes: rawSegmentBytes, sealEnvelope: sealEnvelope, sealEnvelopeDigest: sealEnvelopeDigest}
}

func buildVerificationFixtureEventFrame(t *testing.T, privateKey ed25519.PrivateKey, keyID string, eventPayload map[string]any) (SignedObjectEnvelope, []byte, Digest) {
	t.Helper()
	eventEnvelope := signEnvelopeFixture(t, privateKey, keyID, AuditEventSchemaID, AuditEventSchemaVersion, eventPayload)
	eventCanonicalBytes := canonicalEnvelopeBytesFixture(t, eventEnvelope)
	return eventEnvelope, eventCanonicalBytes, digestForBytesFixture(eventCanonicalBytes)
}

func buildVerificationFixtureSegment(t *testing.T, eventDigest Digest, eventCanonicalBytes []byte) (AuditSegmentFilePayload, []byte, Digest, Digest) {
	t.Helper()
	rawSegmentBytes := []byte("audit-segment-raw-bytes-0001")
	segmentHash, err := ComputeSegmentFileHash(rawSegmentBytes)
	if err != nil {
		t.Fatalf("ComputeSegmentFileHash returned error: %v", err)
	}
	merkleRoot, err := ComputeOrderedAuditSegmentMerkleRoot([]Digest{eventDigest})
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot returned error: %v", err)
	}
	segment := AuditSegmentFilePayload{SchemaID: "runecode.protocol.v0.AuditSegmentFile", SchemaVersion: "0.1.0", Header: AuditSegmentHeader{Format: "audit_segment_framed_v1", SegmentID: "segment-0001", SegmentState: "sealed", CreatedAt: "2026-03-13T12:10:00Z", Writer: "auditd"}, Frames: []AuditSegmentRecordFrame{mapEventFrameFixture(eventDigest, eventCanonicalBytes)}, LifecycleMarker: AuditSegmentLifecycleMarker{State: "sealed", MarkedAt: "2026-03-13T12:20:00Z", Reason: "size_threshold"}}
	return segment, rawSegmentBytes, segmentHash, merkleRoot
}

func buildVerificationFixtureSealEnvelope(t *testing.T, privateKey ed25519.PrivateKey, keyID, segmentID string, eventDigest, segmentHash, merkleRoot Digest, eventPayload map[string]any) (SignedObjectEnvelope, Digest) {
	t.Helper()
	sealPayload := map[string]any{"schema_id": AuditSegmentSealSchemaID, "schema_version": AuditSegmentSealSchemaVersion, "segment_id": segmentID, "sealed_after_state": AuditSegmentStateOpen, "segment_state": AuditSegmentStateSealed, "segment_cut": map[string]any{"ownership_scope": AuditSegmentOwnershipScopeInstanceGlobal, "max_segment_bytes": 1024, "cut_trigger": AuditSegmentCutTriggerSizeWindow}, "event_count": 1, "first_record_digest": eventDigest, "last_record_digest": eventDigest, "merkle_profile": AuditSegmentMerkleProfileOrderedDSEv1, "merkle_root": merkleRoot, "segment_file_hash_scope": AuditSegmentFileHashScopeRawFramedV1, "segment_file_hash": segmentHash, "seal_chain_index": 0, "anchoring_subject": AuditSegmentAnchoringSubjectSeal, "sealed_at": "2026-03-13T12:20:00Z", "protocol_bundle_manifest_hash": eventPayload["protocol_bundle_manifest_hash"]}
	sealEnvelope := signEnvelopeFixture(t, privateKey, keyID, AuditSegmentSealSchemaID, AuditSegmentSealSchemaVersion, sealPayload)
	sealEnvelopeDigest, err := ComputeSignedEnvelopeAuditRecordDigest(sealEnvelope)
	if err != nil {
		t.Fatalf("ComputeSignedEnvelopeAuditRecordDigest returned error: %v", err)
	}
	return sealEnvelope, sealEnvelopeDigest
}

func buildVerificationFixtureVerifierRecord(publicKey ed25519.PublicKey, keyID string, verifierStatus verifierStatusFixture) VerifierRecord {
	status := verifierStatus.status
	if status == "" {
		status = "active"
	}
	return VerifierRecord{SchemaID: VerifierSchemaID, SchemaVersion: VerifierSchemaVersion, KeyID: KeyIDProfile, KeyIDValue: keyID, Alg: "ed25519", PublicKey: PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "audit_anchor", LogicalScope: "node", OwnerPrincipal: PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"}, KeyProtectionPosture: "os_keystore", IdentityBindingPosture: "attested", PresenceMode: "os_confirmation", CreatedAt: "2026-03-13T12:00:00Z", Status: status, StatusChangedAt: verifierStatus.statusChangedAt}
}

func buildVerificationFixtureSignerEvidence(keyID string) []AuditSignerEvidenceReference {
	return []AuditSignerEvidenceReference{{Digest: Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)}, Evidence: AuditSignerEvidence{SignerPurpose: "isolate_session_identity", SignerScope: "session", SignerKey: SignatureBlock{Alg: "ed25519", KeyID: KeyIDProfile, KeyIDValue: keyID, Signature: "c2ln"}, IsolateBinding: &IsolateSessionBinding{RunID: "run-1", IsolateID: "isolate-1", SessionID: "session-1", SessionNonce: "nonce-0123456789abcd", ProvisioningMode: "tofu", ImageDigest: Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}, ActiveManifestHash: Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)}, HandshakeTranscriptHash: Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)}, KeyID: KeyIDProfile, KeyIDValue: keyID, IdentityBindingPosture: "tofu"}}}}
}

func (f auditVerificationFixture) anchorReceiptEnvelope(t *testing.T, subjectDigest Digest) SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{"schema_id": AuditReceiptSchemaID, "schema_version": AuditReceiptSchemaVersion, "subject_digest": subjectDigest, "audit_receipt_kind": "anchor", "subject_family": "audit_segment_seal", "recorder": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"}, "recorded_at": "2026-03-13T12:25:00Z", "receipt_payload_schema_id": "runecode.protocol.audit.receipt.anchor.v0", "receipt_payload": map[string]any{"anchor_kind": "local_user_presence_signature", "key_protection_posture": "os_keystore", "presence_mode": "os_confirmation", "approval_assurance_level": "session_authenticated", "approval_decision_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("a", 64)}, "anchor_witness": map[string]any{"witness_kind": "local_user_presence_signature_v0", "witness_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)}}}})
}

func (f auditVerificationFixture) importRestoreReceiptEnvelope(t *testing.T, subjectDigest Digest, sealEnvelope SignedObjectEnvelope) SignedObjectEnvelope {
	t.Helper()
	sealing := mustDecodeSealPayloadFixture(t, sealEnvelope.Payload)
	matchingHash := map[string]any{"hash_alg": sealing.SegmentFileHash.HashAlg, "hash": sealing.SegmentFileHash.Hash}
	matchingRoot := map[string]any{"hash_alg": sealing.MerkleRoot.HashAlg, "hash": sealing.MerkleRoot.Hash}
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{"schema_id": AuditReceiptSchemaID, "schema_version": AuditReceiptSchemaVersion, "subject_digest": subjectDigest, "audit_receipt_kind": "import", "subject_family": "audit_segment_seal", "recorded_at": "2026-03-13T12:25:00Z", "recorder": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"}, "receipt_payload_schema_id": "runecode.protocol.audit.receipt.import_restore_provenance.v0", "receipt_payload": map[string]any{"provenance_action": "import", "segment_file_hash_scope": "raw_framed_segment_bytes_v1", "imported_segments": []any{map[string]any{"imported_segment_seal_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("8", 64)}, "imported_segment_root": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("7", 64)}, "source_segment_file_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("6", 64)}, "local_segment_file_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("6", 64)}, "byte_identity_verified": true}, map[string]any{"imported_segment_seal_digest": map[string]any{"hash_alg": subjectDigest.HashAlg, "hash": subjectDigest.Hash}, "imported_segment_root": matchingRoot, "source_segment_file_hash": matchingHash, "local_segment_file_hash": matchingHash, "byte_identity_verified": true}}, "source_manifest_digests": []any{map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("4", 64)}}, "authority_context": map[string]any{"authority_kind": "operator", "authority_id": "operator-1"}}})
}

func mustDecodeSealPayloadFixture(t *testing.T, payload json.RawMessage) AuditSegmentSealPayload {
	t.Helper()
	sealing := AuditSegmentSealPayload{}
	if err := json.Unmarshal(payload, &sealing); err != nil {
		t.Fatalf("Unmarshal seal payload returned error: %v", err)
	}
	return sealing
}

func signEnvelopeFixture(t *testing.T, privateKey ed25519.PrivateKey, keyID, payloadSchemaID, payloadSchemaVersion string, payload map[string]any) SignedObjectEnvelope {
	t.Helper()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(payloadBytes)
	if err != nil {
		t.Fatalf("canonicalize payload returned error: %v", err)
	}
	signature := ed25519.Sign(privateKey, canonicalPayload)
	return SignedObjectEnvelope{SchemaID: EnvelopeSchemaID, SchemaVersion: EnvelopeSchemaVersion, PayloadSchemaID: payloadSchemaID, PayloadSchemaVersion: payloadSchemaVersion, Payload: payloadBytes, SignatureInput: SignatureInputProfile, Signature: SignatureBlock{Alg: "ed25519", KeyID: KeyIDProfile, KeyIDValue: keyID, Signature: base64.StdEncoding.EncodeToString(signature)}}
}

func canonicalEnvelopeBytesFixture(t *testing.T, envelope SignedObjectEnvelope) []byte {
	t.Helper()
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal envelope returned error: %v", err)
	}
	canonicalEnvelope, err := jsoncanonicalizer.Transform(envelopeBytes)
	if err != nil {
		t.Fatalf("canonicalize envelope returned error: %v", err)
	}
	return canonicalEnvelope
}

func digestForBytesFixture(bytes []byte) Digest {
	sum := sha256.Sum256(bytes)
	return Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}
}

func marshalJSONFixture(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	return b
}

func resignEnvelopeFixture(t *testing.T, privateKey ed25519.PrivateKey, envelope SignedObjectEnvelope) SignedObjectEnvelope {
	t.Helper()
	canonicalPayload, err := jsoncanonicalizer.Transform(envelope.Payload)
	if err != nil {
		t.Fatalf("Transform payload returned error: %v", err)
	}
	envelope.Signature.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, canonicalPayload))
	return envelope
}

func mapEventFrameFixture(digest Digest, canonicalEnvelopeBytes []byte) AuditSegmentRecordFrame {
	return AuditSegmentRecordFrame{RecordDigest: digest, ByteLength: int64(len(canonicalEnvelopeBytes)), CanonicalSignedEnvelopeBytes: base64.StdEncoding.EncodeToString(canonicalEnvelopeBytes)}
}

func containsReasonCode(values []string, code string) bool {
	for _, value := range values {
		if value == code {
			return true
		}
	}
	return false
}
