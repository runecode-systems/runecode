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

func TestVerifyAuditEvidenceMissingAnchorIsDegradedByDefault(t *testing.T) {
	report := mustVerifyAuditEvidenceReport(t, newAuditVerificationFixture(t, verifierStatusFixture{status: "active"}), nil)
	assertMissingAnchorDegradesReport(t, report)
	assertDerivedSummaryDegraded(t, report)
}

func TestVerifyAuditEvidenceInvalidAnchorFailsClosed(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	invalidAnchor := fixture.anchorReceiptEnvelope(t, testDigestFromByte('9'))

	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                 AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:               fixture.segment,
		RawFramedSegmentBytes: fixture.rawSegmentBytes,
		SegmentSealEnvelope:   fixture.sealEnvelope,
		ReceiptEnvelopes:      []SignedObjectEnvelope{invalidAnchor},
		VerifierRecords:       fixture.verifierRecords,
		EventContractCatalog:  fixture.eventContractCatalog,
		SignerEvidence:        fixture.signerEvidence,
		Now:                   time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if report.AnchoringStatus != AuditVerificationStatusFailed {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusFailed)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceDistinguishesHistoricalAdmissibilityFromCurrentDegradedPosture(t *testing.T) {
	statusChangedAt := "2026-03-13T12:30:00Z"
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "revoked", statusChangedAt: statusChangedAt})
	anchor := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                 AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:               fixture.segment,
		RawFramedSegmentBytes: fixture.rawSegmentBytes,
		SegmentSealEnvelope:   fixture.sealEnvelope,
		ReceiptEnvelopes:      []SignedObjectEnvelope{anchor},
		VerifierRecords:       fixture.verifierRecords,
		EventContractCatalog:  fixture.eventContractCatalog,
		SignerEvidence:        fixture.signerEvidence,
		Now:                   time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("VerifyAuditEvidence returned error: %v", err)
	}
	if !report.CryptographicallyValid {
		t.Fatal("cryptographically_valid = false, want true")
	}
	if !report.HistoricallyAdmissible {
		t.Fatal("historically_admissible = false, want true")
	}
	if !report.CurrentlyDegraded {
		t.Fatal("currently_degraded = false, want true")
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonSignerCurrentlyRevokedOrCompromised) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonSignerCurrentlyRevokedOrCompromised)
	}
	if report.AnchoringStatus != AuditVerificationStatusOK {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusOK)
	}
}

func TestVerifyAuditEvidenceFailsClosedOnInvalidReceiptRecorder(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	payload["recorder"] = map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity"}
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.AnchoringStatus != AuditVerificationStatusFailed {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusFailed)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceKeepsCryptographicValidityForAnchoringOnlyFailure(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	invalidAnchor := fixture.anchorReceiptEnvelope(t, testDigestFromByte('9'))
	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{invalidAnchor})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
	if !report.CryptographicallyValid {
		t.Fatal("cryptographically_valid = false, want true for non-cryptographic hard failure")
	}
}

func TestVerifyAuditEvidenceMarksCryptographicValidityFalseForDigestFailure(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	broken := fixture
	broken.segment.Frames[0].RecordDigest = testDigestFromByte('9')
	report := mustVerifyAuditEvidenceReport(t, broken, nil)
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonSegmentFrameDigestMismatch) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonSegmentFrameDigestMismatch)
	}
	if report.CryptographicallyValid {
		t.Fatal("cryptographically_valid = true, want false for digest mismatch")
	}
}

type verifierStatusFixture struct {
	status          string
	statusChangedAt string
}

type auditVerificationFixture struct {
	segment              AuditSegmentFilePayload
	rawSegmentBytes      []byte
	sealEnvelope         SignedObjectEnvelope
	sealEnvelopeDigest   Digest
	verifierRecords      []VerifierRecord
	eventContractCatalog AuditEventContractCatalog
	signerEvidence       []AuditSignerEvidenceReference
	privateKey           ed25519.PrivateKey
	keyID                string
}

func newAuditVerificationFixture(t *testing.T, verifierStatus verifierStatusFixture) auditVerificationFixture {
	t.Helper()
	request := validAuditAdmissionRequestFixture(t)
	publicKey, privateKey, keyID := generateAuditVerificationFixtureKeys(t)
	eventPayload := mustUnmarshalAuditEventPayload(t, request.Envelope.Payload)
	verificationBundle := buildVerificationFixtureSignedArtifacts(t, privateKey, keyID, eventPayload)
	verifierRecord := buildVerificationFixtureVerifierRecord(publicKey, keyID, verifierStatus)

	return auditVerificationFixture{
		segment:              verificationBundle.segment,
		rawSegmentBytes:      verificationBundle.rawSegmentBytes,
		sealEnvelope:         verificationBundle.sealEnvelope,
		sealEnvelopeDigest:   verificationBundle.sealEnvelopeDigest,
		verifierRecords:      []VerifierRecord{verifierRecord},
		eventContractCatalog: request.EventContractCatalog,
		signerEvidence:       buildVerificationFixtureSignerEvidence(keyID),
		privateKey:           privateKey,
		keyID:                keyID,
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

func buildVerificationFixtureSignerEvidence(keyID string) []AuditSignerEvidenceReference {
	return []AuditSignerEvidenceReference{{
		Digest: Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)},
		Evidence: AuditSignerEvidence{
			SignerPurpose: "isolate_session_identity",
			SignerScope:   "session",
			SignerKey:     SignatureBlock{Alg: "ed25519", KeyID: KeyIDProfile, KeyIDValue: keyID, Signature: "c2ln"},
			IsolateBinding: &IsolateSessionBinding{
				RunID:                   "run-1",
				IsolateID:               "isolate-1",
				SessionID:               "session-1",
				SessionNonce:            "nonce-1",
				ProvisioningMode:        "tofu",
				ImageDigest:             Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)},
				ActiveManifestHash:      Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)},
				HandshakeTranscriptHash: Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)},
				KeyID:                   KeyIDProfile,
				KeyIDValue:              keyID,
				IdentityBindingPosture:  "tofu",
			},
		},
	}}
}

type verificationFixtureSignedArtifacts struct {
	segment            AuditSegmentFilePayload
	rawSegmentBytes    []byte
	sealEnvelope       SignedObjectEnvelope
	sealEnvelopeDigest Digest
}

func buildVerificationFixtureSignedArtifacts(t *testing.T, privateKey ed25519.PrivateKey, keyID string, eventPayload map[string]any) verificationFixtureSignedArtifacts {
	t.Helper()
	_, eventCanonicalBytes, eventDigest := buildVerificationFixtureEventFrame(t, privateKey, keyID, eventPayload)
	segment, rawSegmentBytes, segmentHash, merkleRoot := buildVerificationFixtureSegment(t, eventDigest, eventCanonicalBytes)
	sealEnvelope, sealEnvelopeDigest := buildVerificationFixtureSealEnvelope(t, privateKey, keyID, segment.Header.SegmentID, eventDigest, segmentHash, merkleRoot, eventPayload)
	return verificationFixtureSignedArtifacts{
		segment:            segment,
		rawSegmentBytes:    rawSegmentBytes,
		sealEnvelope:       sealEnvelope,
		sealEnvelopeDigest: sealEnvelopeDigest,
	}
}

func mustVerifyAuditEvidenceReport(t *testing.T, fixture auditVerificationFixture, receipts []SignedObjectEnvelope) AuditVerificationReportPayload {
	t.Helper()
	report, err := VerifyAuditEvidence(AuditVerificationInput{
		Scope:                 AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: fixture.segment.Header.SegmentID},
		Segment:               fixture.segment,
		RawFramedSegmentBytes: fixture.rawSegmentBytes,
		SegmentSealEnvelope:   fixture.sealEnvelope,
		ReceiptEnvelopes:      receipts,
		VerifierRecords:       fixture.verifierRecords,
		EventContractCatalog:  fixture.eventContractCatalog,
		SignerEvidence:        fixture.signerEvidence,
		Now:                   time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC),
	})
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

func generateAuditVerificationFixtureKeys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	keySum := sha256.Sum256(publicKey)
	return publicKey, privateKey, hex.EncodeToString(keySum[:])
}

func buildVerificationFixtureEventFrame(t *testing.T, privateKey ed25519.PrivateKey, keyID string, eventPayload map[string]any) (SignedObjectEnvelope, []byte, Digest) {
	t.Helper()
	eventEnvelope := signEnvelopeFixture(t, privateKey, keyID, AuditEventSchemaID, AuditEventSchemaVersion, eventPayload)
	eventCanonicalBytes := canonicalEnvelopeBytesFixture(t, eventEnvelope)
	eventDigest := digestForBytesFixture(eventCanonicalBytes)
	return eventEnvelope, eventCanonicalBytes, eventDigest
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
	segment := AuditSegmentFilePayload{
		SchemaID:      "runecode.protocol.v0.AuditSegmentFile",
		SchemaVersion: "0.1.0",
		Header: AuditSegmentHeader{
			Format:       "audit_segment_framed_v1",
			SegmentID:    "segment-0001",
			SegmentState: "sealed",
			CreatedAt:    "2026-03-13T12:10:00Z",
			Writer:       "auditd",
		},
		Frames: []AuditSegmentRecordFrame{mapEventFrameFixture(eventDigest, eventCanonicalBytes)},
		LifecycleMarker: AuditSegmentLifecycleMarker{
			State:    "sealed",
			MarkedAt: "2026-03-13T12:20:00Z",
			Reason:   "size_threshold",
		},
	}
	return segment, rawSegmentBytes, segmentHash, merkleRoot
}

func buildVerificationFixtureSealEnvelope(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	keyID string,
	segmentID string,
	eventDigest Digest,
	segmentHash Digest,
	merkleRoot Digest,
	eventPayload map[string]any,
) (SignedObjectEnvelope, Digest) {
	t.Helper()
	sealPayload := map[string]any{
		"schema_id":                     AuditSegmentSealSchemaID,
		"schema_version":                AuditSegmentSealSchemaVersion,
		"segment_id":                    segmentID,
		"sealed_after_state":            AuditSegmentStateOpen,
		"segment_state":                 AuditSegmentStateSealed,
		"segment_cut":                   map[string]any{"ownership_scope": AuditSegmentOwnershipScopeInstanceGlobal, "max_segment_bytes": 1024, "cut_trigger": AuditSegmentCutTriggerSizeWindow},
		"event_count":                   1,
		"first_record_digest":           eventDigest,
		"last_record_digest":            eventDigest,
		"merkle_profile":                AuditSegmentMerkleProfileOrderedDSEv1,
		"merkle_root":                   merkleRoot,
		"segment_file_hash_scope":       AuditSegmentFileHashScopeRawFramedV1,
		"segment_file_hash":             segmentHash,
		"seal_chain_index":              0,
		"anchoring_subject":             AuditSegmentAnchoringSubjectSeal,
		"sealed_at":                     "2026-03-13T12:20:00Z",
		"protocol_bundle_manifest_hash": eventPayload["protocol_bundle_manifest_hash"],
	}
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
	return VerifierRecord{
		SchemaID:               VerifierSchemaID,
		SchemaVersion:          VerifierSchemaVersion,
		KeyID:                  KeyIDProfile,
		KeyIDValue:             keyID,
		Alg:                    "ed25519",
		PublicKey:              PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "host_audit",
		LogicalScope:           "node",
		OwnerPrincipal:         PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 status,
		StatusChangedAt:        verifierStatus.statusChangedAt,
	}
}

func (f auditVerificationFixture) anchorReceiptEnvelope(t *testing.T, subjectDigest Digest) SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeFixture(t, f.privateKey, f.keyID, AuditReceiptSchemaID, AuditReceiptSchemaVersion, map[string]any{
		"schema_id":          AuditReceiptSchemaID,
		"schema_version":     AuditReceiptSchemaVersion,
		"subject_digest":     subjectDigest,
		"audit_receipt_kind": "anchor",
		"subject_family":     "audit_segment_seal",
		"recorder": map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "daemon",
			"principal_id":   "auditd",
			"instance_id":    "auditd-1",
		},
		"recorded_at": "2026-03-13T12:25:00Z",
	})
}

func mapEventFrameFixture(digest Digest, canonicalEnvelopeBytes []byte) AuditSegmentRecordFrame {
	return AuditSegmentRecordFrame{
		RecordDigest:                 digest,
		ByteLength:                   int64(len(canonicalEnvelopeBytes)),
		CanonicalSignedEnvelopeBytes: base64.StdEncoding.EncodeToString(canonicalEnvelopeBytes),
	}
}

func signEnvelopeFixture(t *testing.T, privateKey ed25519.PrivateKey, keyID string, payloadSchemaID string, payloadSchemaVersion string, payload map[string]any) SignedObjectEnvelope {
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
	return SignedObjectEnvelope{
		SchemaID:             EnvelopeSchemaID,
		SchemaVersion:        EnvelopeSchemaVersion,
		PayloadSchemaID:      payloadSchemaID,
		PayloadSchemaVersion: payloadSchemaVersion,
		Payload:              payloadBytes,
		SignatureInput:       SignatureInputProfile,
		Signature: SignatureBlock{
			Alg:        "ed25519",
			KeyID:      KeyIDProfile,
			KeyIDValue: keyID,
			Signature:  base64.StdEncoding.EncodeToString(signature),
		},
	}
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

func containsReasonCode(values []string, code string) bool {
	for _, value := range values {
		if value == code {
			return true
		}
	}
	return false
}
