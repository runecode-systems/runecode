package auditd

import (
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
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestAppendReloadRecoveryAndIndex(t *testing.T) {
	root, ledger, result := appendFixtureAndBuildIndex(t)
	if result.FrameCount != 1 {
		t.Fatalf("FrameCount = %d, want 1", result.FrameCount)
	}
	index := mustBuildIndex(t, ledger)
	if index.TotalRecords != 1 {
		t.Fatalf("TotalRecords = %d, want 1", index.TotalRecords)
	}
	assertRecoveredOpenState(t, root, 1)
}

func TestSidecarEvidencePersistenceByDigest(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	sealID, _ := sealResult.SealEnvelopeDigest.Identity()
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, sealsDirName), sealID)
	receiptEnvelope := buildAnchorReceiptEnvelope(t, fixture, sealResult.SealEnvelopeDigest)
	receiptDigest := mustPersistReceipt(t, ledger, receiptEnvelope)
	receiptID, _ := receiptDigest.Identity()
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, receiptsDirName), receiptID)

	report := validReportFixture("segment-000001")
	reportDigest := mustPersistReport(t, ledger, report)
	reportID, _ := reportDigest.Identity()
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, verificationReportsDirName), reportID)
}

func TestReadinessSemantics(t *testing.T) {
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	readiness, err := ledger.Readiness()
	if err != nil {
		t.Fatalf("Readiness returned error: %v", err)
	}
	if readiness.Ready {
		t.Fatal("Readiness.Ready = true, want false before verification inputs")
	}
	if readiness.VerifierMaterialAvailable {
		t.Fatal("VerifierMaterialAvailable = true, want false")
	}

	fixture := newAuditFixtureKey(t)
	request := validAdmissionRequestForLedger(t, fixture)
	if err := ledger.ConfigureVerificationInputs(VerificationConfiguration{VerifierRecords: request.VerifierRecords, EventContractCatalog: request.EventContractCatalog, SignerEvidence: request.SignerEvidence}); err != nil {
		t.Fatalf("ConfigureVerificationInputs returned error: %v", err)
	}
	if _, err := ledger.AppendAdmittedEvent(request); err != nil {
		t.Fatalf("AppendAdmittedEvent returned error: %v", err)
	}
	if _, err := ledger.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	readiness, err = ledger.Readiness()
	if err != nil {
		t.Fatalf("Readiness(after append) returned error: %v", err)
	}
	if !readiness.Ready {
		t.Fatal("Readiness.Ready = false, want true")
	}
}

func TestReadinessFailsClosedWhenVerificationInputsMalformed(t *testing.T) {
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contractsDir, "event-contract-catalog.json"), []byte(`{"bad":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contractsDir, "verifier-records.json"), []byte(`[]`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	readiness, err := ledger.Readiness()
	if err != nil {
		t.Fatalf("Readiness returned error: %v", err)
	}
	if readiness.VerifierMaterialAvailable {
		t.Fatal("VerifierMaterialAvailable = true, want false for malformed contracts")
	}
	if readiness.Ready {
		t.Fatal("Readiness.Ready = true, want false for malformed contracts")
	}
}

func TestLatestVerificationReportRecoversFromStatePointerLoss(t *testing.T) {
	root, ledger, _ := setupLedgerWithAdmissionFixture(t)
	report := validReportFixture("segment-000001")
	digest := mustPersistReport(t, ledger, report)
	statePath := filepath.Join(root, stateFileName)
	state := ledgerState{}
	if err := readJSONFile(statePath, &state); err != nil {
		t.Fatalf("readJSONFile returned error: %v", err)
	}
	state.LastVerificationReportDigest = ""
	if err := writeCanonicalJSONFile(statePath, state); err != nil {
		t.Fatalf("writeCanonicalJSONFile returned error: %v", err)
	}
	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(reopened) returned error: %v", err)
	}
	loaded, err := reopened.LatestVerificationReport()
	if err != nil {
		t.Fatalf("LatestVerificationReport returned error: %v", err)
	}
	loadedDigest, err := canonicalDigest(loaded)
	if err != nil {
		t.Fatalf("canonicalDigest returned error: %v", err)
	}
	loadedID, _ := loadedDigest.Identity()
	expectedID, _ := digest.Identity()
	if loadedID != expectedID {
		t.Fatalf("loaded report digest = %q, want %q", loadedID, expectedID)
	}
}

func TestWriteCanonicalJSONFileReplacesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"stale":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	state := ledgerState{SchemaVersion: stateSchemaVersion, LastSealedSegmentID: "segment-000123"}
	if err := writeCanonicalJSONFile(path, state); err != nil {
		t.Fatalf("writeCanonicalJSONFile returned error: %v", err)
	}
	loaded := ledgerState{}
	if err := readJSONFile(path, &loaded); err != nil {
		t.Fatalf("readJSONFile returned error: %v", err)
	}
	if loaded.LastSealedSegmentID != "segment-000123" {
		t.Fatalf("LastSealedSegmentID = %q, want segment-000123", loaded.LastSealedSegmentID)
	}
}

func TestConfigureVerificationInputsClearsOmittedOptionalFiles(t *testing.T) {
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	fixture := newAuditFixtureKey(t)
	request := validAdmissionRequestForLedger(t, fixture)
	if err := ledger.ConfigureVerificationInputs(VerificationConfiguration{
		VerifierRecords:      request.VerifierRecords,
		EventContractCatalog: request.EventContractCatalog,
		SignerEvidence:       request.SignerEvidence,
		StoragePosture:       fullStoragePostureFixture(),
	}); err != nil {
		t.Fatalf("ConfigureVerificationInputs(initial) returned error: %v", err)
	}
	contractsDir := filepath.Join(root, "contracts")
	assertPathPresent(t, filepath.Join(contractsDir, "signer-evidence.json"), "signer-evidence.json missing after initial configure")
	assertPathPresent(t, filepath.Join(contractsDir, "storage-posture.json"), "storage-posture.json missing after initial configure")

	if err := ledger.ConfigureVerificationInputs(VerificationConfiguration{
		VerifierRecords:      request.VerifierRecords,
		EventContractCatalog: request.EventContractCatalog,
	}); err != nil {
		t.Fatalf("ConfigureVerificationInputs(update) returned error: %v", err)
	}
	assertPathMissing(t, filepath.Join(contractsDir, "signer-evidence.json"), "signer-evidence.json should be removed when omitted")
	assertPathMissing(t, filepath.Join(contractsDir, "storage-posture.json"), "storage-posture.json should be removed when omitted")
}

func fullStoragePostureFixture() *trustpolicy.AuditStoragePostureEvidence {
	return &trustpolicy.AuditStoragePostureEvidence{EncryptedAtRestDefault: true, EncryptedAtRestEffective: true, DevPlaintextOverrideActive: false, SurfacedToOperator: true}
}

func assertPathPresent(t *testing.T, path string, msg string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

func assertPathMissing(t *testing.T, path string, msg string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("%s, stat err = %v", msg, err)
	}
}

type auditFixtureKey struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	keyIDValue string
}

func newAuditFixtureKey(t *testing.T) auditFixtureKey {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	id := sha256.Sum256(publicKey)
	return auditFixtureKey{publicKey: publicKey, privateKey: privateKey, keyIDValue: hex.EncodeToString(id[:])}
}

func validAdmissionRequestForLedger(t *testing.T, f auditFixtureKey) trustpolicy.AuditAdmissionRequest {
	t.Helper()
	eventPayload := map[string]any{"session_id": "session-1"}
	eventPayloadHash := sha256.Sum256(mustCanonicalJSON(t, eventPayload))
	payload := auditEventPayloadFixture(eventPayload, eventPayloadHash)
	payloadBytes := mustJSON(t, payload)
	signature := ed25519.Sign(f.privateKey, mustCanonicalBytes(t, payloadBytes))
	return trustpolicy.AuditAdmissionRequest{Checks: trustpolicy.AuditAdmissionChecks{SchemaValidation: true, EventContractValidation: true, SignerEvidenceValidation: true, DetachedSignatureVerify: true}, Envelope: signedEventEnvelope(payloadBytes, f.keyIDValue, signature), VerifierRecords: []trustpolicy.VerifierRecord{buildVerifierRecord(f)}, EventContractCatalog: eventContractCatalogFixture(), SignerEvidence: []trustpolicy.AuditSignerEvidenceReference{signerEvidenceFixture(f.keyIDValue)}}
}

func auditEventPayloadFixture(eventPayload map[string]any, eventPayloadHash [32]byte) map[string]any {
	return map[string]any{
		"schema_id":                     trustpolicy.AuditEventSchemaID,
		"schema_version":                trustpolicy.AuditEventSchemaVersion,
		"audit_event_type":              "isolate_session_bound",
		"emitter_stream_id":             "auditd-stream-1",
		"seq":                           1,
		"occurred_at":                   "2026-03-13T12:15:00Z",
		"principal":                     map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"event_payload_schema_id":       "runecode.protocol.audit.payload.isolate-session-bound.v0",
		"event_payload":                 eventPayload,
		"event_payload_hash":            map[string]any{"hash_alg": "sha256", "hash": hex.EncodeToString(eventPayloadHash[:])},
		"protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
		"scope":                         map[string]any{"workspace_id": "workspace-1", "run_id": "run-1", "stage_id": "stage-1"},
		"correlation":                   map[string]any{"session_id": "session-1", "operation_id": "op-1"},
		"subject_ref":                   map[string]any{"object_family": "isolate_binding", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)}, "ref_role": "binding_target"},
		"cause_refs":                    []any{map[string]any{"object_family": "audit_event", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "ref_role": "session_cause"}},
		"related_refs":                  []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)}, "ref_role": "binding"}},
		"signer_evidence_refs":          []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "ref_role": "admissibility"}},
	}
}

func signedEventEnvelope(payloadBytes []byte, keyID string, signature []byte) trustpolicy.SignedObjectEnvelope {
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditEventSchemaID, PayloadSchemaVersion: trustpolicy.AuditEventSchemaVersion, Payload: payloadBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyID, Signature: base64.StdEncoding.EncodeToString(signature)}}
}

func eventContractCatalogFixture() trustpolicy.AuditEventContractCatalog {
	return trustpolicy.AuditEventContractCatalog{SchemaID: trustpolicy.AuditEventContractCatalogSchemaID, SchemaVersion: trustpolicy.AuditEventContractCatalogSchemaVersion, CatalogID: "audit_event_contract_v0", Entries: []trustpolicy.AuditEventContractCatalogEntry{{AuditEventType: "isolate_session_bound", AllowedPayloadSchemaIDs: []string{"runecode.protocol.audit.payload.isolate-session-bound.v0"}, AllowedSignerPurposes: []string{"isolate_session_identity"}, AllowedSignerScopes: []string{"session"}, RequiredScopeFields: []string{"workspace_id", "run_id", "stage_id"}, RequiredCorrelationFields: []string{"session_id", "operation_id"}, RequireSubjectRef: true, AllowedSubjectRefRoles: []string{"binding_target"}, AllowedCauseRefRoles: []string{"session_cause"}, AllowedRelatedRefRoles: []string{"binding", "evidence", "receipt"}, RequireGatewayContext: false, RequireSignerEvidenceRefs: true, AllowedSignerEvidenceRefRoles: []string{"admissibility", "binding"}}}}
}

func signerEvidenceFixture(keyIDValue string) trustpolicy.AuditSignerEvidenceReference {
	return trustpolicy.AuditSignerEvidenceReference{Digest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)}, Evidence: trustpolicy.AuditSignerEvidence{SignerPurpose: "isolate_session_identity", SignerScope: "session", SignerKey: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: "c2ln"}, IsolateBinding: &trustpolicy.IsolateSessionBinding{RunID: "run-1", IsolateID: "isolate-1", SessionID: "session-1", SessionNonce: "nonce-1", ProvisioningMode: "tofu", ImageDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}, ActiveManifestHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)}, HandshakeTranscriptHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)}, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, IdentityBindingPosture: "tofu"}}}
}

func validReportFixture(segmentID string) trustpolicy.AuditVerificationReportPayload {
	return trustpolicy.AuditVerificationReportPayload{SchemaID: trustpolicy.AuditVerificationReportSchemaID, SchemaVersion: trustpolicy.AuditVerificationReportSchemaVersion, VerifiedAt: time.Now().UTC().Format(time.RFC3339), VerificationScope: trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: segmentID}, CryptographicallyValid: true, HistoricallyAdmissible: true, CurrentlyDegraded: false, IntegrityStatus: trustpolicy.AuditVerificationStatusOK, AnchoringStatus: trustpolicy.AuditVerificationStatusOK, StoragePostureStatus: trustpolicy.AuditVerificationStatusOK, SegmentLifecycleStatus: trustpolicy.AuditVerificationStatusOK, DegradedReasons: []string{}, HardFailures: []string{}, Findings: []trustpolicy.AuditVerificationFinding{}, Summary: "ok"}
}

func assertDigestSidecarExists(t *testing.T, dir string, digestID string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, strings.TrimPrefix(digestID, "sha256:")+".json")); err != nil {
		t.Fatalf("sidecar missing: %v", err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	return b
}

func mustCanonicalBytes(t *testing.T, value []byte) []byte {
	t.Helper()
	b, err := jsoncanonicalizer.Transform(value)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}
	return b
}

func mustCanonicalJSON(t *testing.T, value any) []byte {
	t.Helper()
	return mustCanonicalBytes(t, mustJSON(t, value))
}

func buildVerifierRecord(f auditFixtureKey) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             f.keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(f.publicKey)},
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

func buildSealEnvelopeForSegment(t *testing.T, f auditFixtureKey, ledger *Ledger, segment trustpolicy.AuditSegmentFilePayload, previous *trustpolicy.Digest, chainIndex int64) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	payloadBytes := mustJSON(t, sealPayloadFixture(t, ledger, segment, previous, chainIndex))
	sig := ed25519.Sign(f.privateKey, mustCanonicalBytes(t, payloadBytes))
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditSegmentSealSchemaID, PayloadSchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, Payload: payloadBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: f.keyIDValue, Signature: base64.StdEncoding.EncodeToString(sig)}}
}

func sealPayloadFixture(t *testing.T, ledger *Ledger, segment trustpolicy.AuditSegmentFilePayload, previous *trustpolicy.Digest, chainIndex int64) trustpolicy.AuditSegmentSealPayload {
	t.Helper()
	digests := make([]trustpolicy.Digest, 0, len(segment.Frames))
	for _, frame := range segment.Frames {
		digests = append(digests, frame.RecordDigest)
	}
	root, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot(digests)
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot returned error: %v", err)
	}
	raw, err := ledger.rawSegmentFramedBytes(segment)
	if err != nil {
		t.Fatalf("rawSegmentFramedBytes returned error: %v", err)
	}
	fileHash, err := trustpolicy.ComputeSegmentFileHash(raw)
	if err != nil {
		t.Fatalf("ComputeSegmentFileHash returned error: %v", err)
	}
	return trustpolicy.AuditSegmentSealPayload{SchemaID: trustpolicy.AuditSegmentSealSchemaID, SchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, SegmentID: segment.Header.SegmentID, SealedAfterState: trustpolicy.AuditSegmentStateOpen, SegmentState: trustpolicy.AuditSegmentStateSealed, SegmentCut: trustpolicy.AuditSegmentCutWindowPolicy{OwnershipScope: trustpolicy.AuditSegmentOwnershipScopeInstanceGlobal, MaxSegmentBytes: 1024, CutTrigger: trustpolicy.AuditSegmentCutTriggerSizeWindow}, EventCount: int64(len(segment.Frames)), FirstRecordDigest: segment.Frames[0].RecordDigest, LastRecordDigest: segment.Frames[len(segment.Frames)-1].RecordDigest, MerkleProfile: trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1, MerkleRoot: root, SegmentFileHashScope: trustpolicy.AuditSegmentFileHashScopeRawFramedV1, SegmentFileHash: fileHash, SealChainIndex: chainIndex, PreviousSealDigest: previous, AnchoringSubject: trustpolicy.AuditSegmentAnchoringSubjectSeal, SealedAt: "2026-03-13T12:30:00Z", ProtocolBundleManifestHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)}, SealReason: "size_threshold"}
}

func setupLedgerWithAdmissionFixture(t *testing.T) (string, *Ledger, auditFixtureKey) {
	t.Helper()
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	fixture := newAuditFixtureKey(t)
	request := validAdmissionRequestForLedger(t, fixture)
	if err := ledger.ConfigureVerificationInputs(VerificationConfiguration{VerifierRecords: request.VerifierRecords, EventContractCatalog: request.EventContractCatalog, SignerEvidence: request.SignerEvidence}); err != nil {
		t.Fatalf("ConfigureVerificationInputs returned error: %v", err)
	}
	if _, err := ledger.AppendAdmittedEvent(request); err != nil {
		t.Fatalf("AppendAdmittedEvent returned error: %v", err)
	}
	return root, ledger, fixture
}

func appendFixtureAndBuildIndex(t *testing.T) (string, *Ledger, AppendResult) {
	t.Helper()
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	fixture := newAuditFixtureKey(t)
	request := validAdmissionRequestForLedger(t, fixture)
	if err := ledger.ConfigureVerificationInputs(VerificationConfiguration{VerifierRecords: request.VerifierRecords, EventContractCatalog: request.EventContractCatalog, SignerEvidence: request.SignerEvidence}); err != nil {
		t.Fatalf("ConfigureVerificationInputs returned error: %v", err)
	}
	result, err := ledger.AppendAdmittedEvent(request)
	if err != nil {
		t.Fatalf("AppendAdmittedEvent returned error: %v", err)
	}
	_, _ = ledger.BuildIndex()
	return root, ledger, result
}

func mustBuildIndex(t *testing.T, ledger *Ledger) derivedIndex {
	t.Helper()
	index, err := ledger.BuildIndex()
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}
	return index
}

func assertRecoveredOpenState(t *testing.T, root string, expectedOpenCount int) {
	t.Helper()
	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(reload) returned error: %v", err)
	}
	state, err := reopened.loadState()
	if err != nil {
		t.Fatalf("loadState returned error: %v", err)
	}
	if state.OpenFrameCount != expectedOpenCount {
		t.Fatalf("OpenFrameCount = %d, want %d", state.OpenFrameCount, expectedOpenCount)
	}
	if state.CurrentOpenSegmentID == "" || !state.RecoveryComplete {
		t.Fatalf("unexpected recovered state: %+v", state)
	}
}

func mustSealFixtureSegment(t *testing.T, ledger *Ledger, fixture auditFixtureKey) SealResult {
	t.Helper()
	segment, err := ledger.loadSegment("segment-000001")
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	sealEnvelope := buildSealEnvelopeForSegment(t, fixture, ledger, segment, nil, 0)
	sealResult, err := ledger.SealCurrentSegment(sealEnvelope)
	if err != nil {
		t.Fatalf("SealCurrentSegment returned error: %v", err)
	}
	return sealResult
}

func mustPersistReceipt(t *testing.T, ledger *Ledger, envelope trustpolicy.SignedObjectEnvelope) trustpolicy.Digest {
	t.Helper()
	digest, err := ledger.PersistReceiptEnvelope(envelope)
	if err != nil {
		t.Fatalf("PersistReceiptEnvelope returned error: %v", err)
	}
	return digest
}

func mustPersistReport(t *testing.T, ledger *Ledger, report trustpolicy.AuditVerificationReportPayload) trustpolicy.Digest {
	t.Helper()
	digest, err := ledger.PersistVerificationReport(report)
	if err != nil {
		t.Fatalf("PersistVerificationReport returned error: %v", err)
	}
	return digest
}

func buildAnchorReceiptEnvelope(t *testing.T, f auditFixtureKey, sealDigest trustpolicy.Digest) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	payload := map[string]any{
		"schema_id":          trustpolicy.AuditReceiptSchemaID,
		"schema_version":     trustpolicy.AuditReceiptSchemaVersion,
		"subject_digest":     map[string]any{"hash_alg": sealDigest.HashAlg, "hash": sealDigest.Hash},
		"audit_receipt_kind": "anchor",
		"subject_family":     "audit_segment_seal",
		"recorder":           map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"recorded_at":        "2026-03-13T12:35:00Z",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	canonical, err := jsoncanonicalizer.Transform(payloadBytes)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}
	sig := ed25519.Sign(f.privateKey, canonical)
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.AuditReceiptSchemaID,
		PayloadSchemaVersion: trustpolicy.AuditReceiptSchemaVersion,
		Payload:              payloadBytes,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: f.keyIDValue, Signature: base64.StdEncoding.EncodeToString(sig)},
	}
}
