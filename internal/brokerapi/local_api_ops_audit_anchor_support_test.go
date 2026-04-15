package brokerapi

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

type anchorImmutabilityBaseline struct {
	segmentPath   string
	segmentBefore []byte
	sealPath      string
	sealBefore    []byte
	sealDigest    trustpolicy.Digest
	sealFiles     []string
}

func captureAnchorImmutabilityBaseline(t *testing.T, service *Service, ledgerRoot string) anchorImmutabilityBaseline {
	t.Helper()
	segmentID := mustLatestSegmentIDForAnchorTest(t, service)
	segmentPath := filepath.Join(ledgerRoot, "segments", segmentID+".json")
	segmentBefore, err := os.ReadFile(segmentPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) before anchor returned error: %v", segmentPath, err)
	}
	sealDigest := mustSealDigestForSegment(t, ledgerRoot, segmentID)
	sealPath := sidecarPathForDigest(ledgerRoot, "segment-seals", sealDigest)
	sealBefore, err := os.ReadFile(sealPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) before anchor returned error: %v", sealPath, err)
	}
	return anchorImmutabilityBaseline{
		segmentPath:   segmentPath,
		segmentBefore: segmentBefore,
		sealPath:      sealPath,
		sealBefore:    sealBefore,
		sealDigest:    sealDigest,
		sealFiles:     mustListSealSidecars(t, ledgerRoot),
	}
}

func assertAnchorImmutabilityAfterAnchor(t *testing.T, service *Service, baseline anchorImmutabilityBaseline) {
	t.Helper()
	segmentAfter, err := os.ReadFile(baseline.segmentPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) after anchor returned error: %v", baseline.segmentPath, err)
	}
	if !bytes.Equal(baseline.segmentBefore, segmentAfter) {
		t.Fatal("sealed segment bytes changed after anchoring")
	}
	sealAfter, err := os.ReadFile(baseline.sealPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) after anchor returned error: %v", baseline.sealPath, err)
	}
	if !bytes.Equal(baseline.sealBefore, sealAfter) {
		t.Fatal("original segment seal envelope changed after anchoring")
	}
	sealFilesAfter := mustListSealSidecars(t, service.auditRoot)
	if strings.Join(baseline.sealFiles, ",") != strings.Join(sealFilesAfter, ",") {
		t.Fatalf("segment seal sidecar set changed after anchoring: before=%v after=%v", baseline.sealFiles, sealFilesAfter)
	}
	if got := mustLatestSealDigestForAnchorTest(t, service); mustDigestIdentityForAnchorTest(got) != mustDigestIdentityForAnchorTest(baseline.sealDigest) {
		t.Fatalf("latest seal digest changed after anchoring: got=%q want=%q", mustDigestIdentityForAnchorTest(got), mustDigestIdentityForAnchorTest(baseline.sealDigest))
	}
}

func newAuditAnchorTestService(t *testing.T) (*Service, string) {
	t.Helper()
	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	if err := seedLedgerForAuditAnchorTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	secretsRoot := filepath.Join(t.TempDir(), "secrets")
	t.Setenv("RUNE_SECRETS_STATE_ROOT", secretsRoot)
	service, err := NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	service.secretsSvc = mustOpenSecretsService(t, secretsRoot)
	return service, ledgerRoot
}

func seedLedgerForAuditAnchorTest(root string) error {
	if err := prepareLedgerDirs(root); err != nil {
		return err
	}
	evidence, err := buildSeedEventEvidence("session-anchor")
	if err != nil {
		return err
	}
	if err := writeSeedSegment(root, "segment-000001", evidence.recordDigest, evidence.canonicalEnvelope); err != nil {
		return err
	}
	anchorPublic, anchorPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	anchorKeyID := sha256.Sum256(anchorPublic)
	anchorKeyIDHex := hex.EncodeToString(anchorKeyID[:])
	if err := writeSeedSealSignedByKey(root, "segment-000001", evidence.recordDigest, 0, anchorPrivate, anchorKeyIDHex); err != nil {
		return err
	}
	ledger, err := openSeededAnchorLedger(root)
	if err != nil {
		return err
	}
	if err := configureSeedContractsAndIndex(ledger); err != nil {
		return err
	}
	if err := addAuditAnchorVerifierRecord(ledger, anchorPublic, anchorKeyIDHex); err != nil {
		return err
	}
	return persistSeedReport(ledger)
}

func openSeededAnchorLedger(root string) (*auditd.Ledger, error) {
	return auditd.Open(root)
}

func addAuditAnchorVerifierRecord(ledger *auditd.Ledger, anchorPublic ed25519.PublicKey, anchorKeyIDHex string) error {
	baseCatalog := seedEventContractCatalog()
	baseVerifier := seedVerifierRecord()
	anchorVerifier := trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             anchorKeyIDHex,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(anchorPublic)},
		LogicalPurpose:         "audit_anchor",
		LogicalScope:           "node",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "secretsd", InstanceID: "secretsd-1"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 "active",
	}
	return ledger.ConfigureVerificationInputs(auditd.VerificationConfiguration{
		VerifierRecords:      []trustpolicy.VerifierRecord{baseVerifier, anchorVerifier},
		EventContractCatalog: baseCatalog,
	})
}

func writeSeedSealSignedByKey(root string, segmentID string, recordDigest trustpolicy.Digest, chainIndex int64, privateKey ed25519.PrivateKey, keyIDValue string) error {
	sealingRaw, err := rawFramedBytesFromSeedSegment(root, segmentID)
	if err != nil {
		return err
	}
	segmentHash, err := trustpolicy.ComputeSegmentFileHash(sealingRaw)
	if err != nil {
		return err
	}
	merkleRoot, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot([]trustpolicy.Digest{recordDigest})
	if err != nil {
		return err
	}
	sealingPayload := seedSealPayload(segmentID, recordDigest, merkleRoot, segmentHash, chainIndex)
	payloadBytes, err := json.Marshal(sealingPayload)
	if err != nil {
		return err
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(payloadBytes)
	if err != nil {
		return err
	}
	sealingEnvelope := sealEnvelopeForPayloadBytes(payloadBytes, canonicalPayload, privateKey, keyIDValue)
	sealDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(sealingEnvelope)
	if err != nil {
		return err
	}
	identity, _ := sealDigest.Identity()
	return writeCanonicalJSON(filepath.Join(root, "sidecar", "segment-seals", strings.TrimPrefix(identity, "sha256:")+".json"), sealingEnvelope)
}

func sealEnvelopeForPayloadBytes(payloadBytes, canonicalPayload []byte, privateKey ed25519.PrivateKey, keyIDValue string) trustpolicy.SignedObjectEnvelope {
	signature := ed25519.Sign(privateKey, canonicalPayload)
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.AuditSegmentSealSchemaID,
		PayloadSchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion,
		Payload:              payloadBytes,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(signature),
		},
	}
}

func seedSealPayload(segmentID string, recordDigest trustpolicy.Digest, merkleRoot trustpolicy.Digest, segmentHash trustpolicy.Digest, chainIndex int64) map[string]any {
	return map[string]any{
		"schema_id":                     trustpolicy.AuditSegmentSealSchemaID,
		"schema_version":                trustpolicy.AuditSegmentSealSchemaVersion,
		"segment_id":                    segmentID,
		"sealed_after_state":            trustpolicy.AuditSegmentStateOpen,
		"segment_state":                 trustpolicy.AuditSegmentStateSealed,
		"segment_cut":                   map[string]any{"ownership_scope": trustpolicy.AuditSegmentOwnershipScopeInstanceGlobal, "max_segment_bytes": 2048, "cut_trigger": trustpolicy.AuditSegmentCutTriggerSizeWindow},
		"event_count":                   1,
		"first_record_digest":           map[string]any{"hash_alg": recordDigest.HashAlg, "hash": recordDigest.Hash},
		"last_record_digest":            map[string]any{"hash_alg": recordDigest.HashAlg, "hash": recordDigest.Hash},
		"merkle_profile":                trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1,
		"merkle_root":                   map[string]any{"hash_alg": merkleRoot.HashAlg, "hash": merkleRoot.Hash},
		"segment_file_hash_scope":       trustpolicy.AuditSegmentFileHashScopeRawFramedV1,
		"segment_file_hash":             map[string]any{"hash_alg": segmentHash.HashAlg, "hash": segmentHash.Hash},
		"seal_chain_index":              chainIndex,
		"anchoring_subject":             trustpolicy.AuditSegmentAnchoringSubjectSeal,
		"sealed_at":                     "2026-03-13T12:20:00Z",
		"protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
		"seal_reason":                   "size_threshold",
	}
}

func rawFramedBytesFromSeedSegment(root string, segmentID string) ([]byte, error) {
	segmentPath := filepath.Join(root, "segments", segmentID+".json")
	segment := trustpolicy.AuditSegmentFilePayload{}
	if err := readJSONFile(pathToSegment(segmentPath), &segment); err != nil {
		return nil, err
	}
	raw := make([]byte, 0, 256)
	for idx, frame := range segment.Frames {
		envelopeBytes, err := base64.StdEncoding.DecodeString(frame.CanonicalSignedEnvelopeBytes)
		if err != nil {
			return nil, fmt.Errorf("frame %d decode: %w", idx, err)
		}
		if int64(len(envelopeBytes)) != frame.ByteLength {
			return nil, fmt.Errorf("frame %d byte_length mismatch", idx)
		}
		raw = append(raw, envelopeBytes...)
		raw = append(raw, '\n')
	}
	return raw, nil
}

func pathToSegment(path string) string { return path }

func mustOpenSecretsService(t *testing.T, root string) *secretsd.Service {
	t.Helper()
	svc, err := secretsd.Open(root)
	if err != nil {
		t.Fatalf("secretsd.Open returned error: %v", err)
	}
	return svc
}

func mustLatestSealDigestForAnchorTest(t *testing.T, service *Service) trustpolicy.Digest {
	t.Helper()
	report, err := service.auditLedger.LatestVerificationReport()
	if err != nil {
		t.Fatalf("LatestVerificationReport returned error: %v", err)
	}
	segmentID := strings.TrimSpace(report.VerificationScope.LastSegmentID)
	if segmentID == "" {
		t.Fatal("latest verification report missing last_segment_id")
	}
	return mustSealDigestForSegment(t, service.auditRoot, segmentID)
}

func mustLatestSegmentIDForAnchorTest(t *testing.T, service *Service) string {
	t.Helper()
	report, err := service.auditLedger.LatestVerificationReport()
	if err != nil {
		t.Fatalf("LatestVerificationReport returned error: %v", err)
	}
	segmentID := strings.TrimSpace(report.VerificationScope.LastSegmentID)
	if segmentID == "" {
		t.Fatal("latest verification report missing last_segment_id")
	}
	return segmentID
}

func mustSealDigestForSegment(t *testing.T, ledgerRoot string, segmentID string) trustpolicy.Digest {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(ledgerRoot, "sidecar", "segment-seals"))
	if err != nil {
		t.Fatalf("ReadDir seals returned error: %v", err)
	}
	for _, entry := range entries {
		digest, matches, entryErr := sealDigestFromEntryForSegment(ledgerRoot, entry.Name(), segmentID)
		if entryErr != nil {
			t.Fatalf("resolve seal digest from %q returned error: %v", entry.Name(), entryErr)
		}
		if matches {
			return digest
		}
	}
	t.Fatalf("seal digest not found for segment %q", segmentID)
	return trustpolicy.Digest{}
}

func sealDigestFromEntryForSegment(ledgerRoot, entryName, segmentID string) (trustpolicy.Digest, bool, error) {
	if !strings.HasSuffix(entryName, ".json") {
		return trustpolicy.Digest{}, false, nil
	}
	path := filepath.Join(ledgerRoot, "sidecar", "segment-seals", entryName)
	var envelope trustpolicy.SignedObjectEnvelope
	if err := readJSONFile(path, &envelope); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	var payload trustpolicy.AuditSegmentSealPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if payload.SegmentID != segmentID {
		return trustpolicy.Digest{}, false, nil
	}
	return trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimSuffix(entryName, ".json")}, true, nil
}

func readJSONFile(path string, into any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, into)
}

func assertAnchorReceiptSidecarExists(t *testing.T, ledgerRoot string, digest trustpolicy.Digest) {
	t.Helper()
	id, _ := digest.Identity()
	path := filepath.Join(ledgerRoot, "sidecar", "receipts", strings.TrimPrefix(id, "sha256:")+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("receipt sidecar not found at %q: %v", path, err)
	}
}

func assertAnchorVerificationReportSidecarExists(t *testing.T, ledgerRoot string, digest trustpolicy.Digest) {
	t.Helper()
	id, _ := digest.Identity()
	path := filepath.Join(ledgerRoot, "sidecar", "verification-reports", strings.TrimPrefix(id, "sha256:")+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("verification report sidecar not found at %q: %v", path, err)
	}
}

func sidecarPathForDigest(ledgerRoot string, sidecarDir string, digest trustpolicy.Digest) string {
	id, _ := digest.Identity()
	return filepath.Join(ledgerRoot, "sidecar", sidecarDir, strings.TrimPrefix(id, "sha256:")+".json")
}

func mustListSealSidecars(t *testing.T, ledgerRoot string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(ledgerRoot, "sidecar", "segment-seals"))
	if err != nil {
		t.Fatalf("ReadDir segment-seals returned error: %v", err)
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		out = append(out, entry.Name())
	}
	sort.Strings(out)
	return out
}

func mustSeedPendingApprovalForAnchorTest(t *testing.T, service *Service) trustpolicy.Digest {
	t.Helper()
	requestEnv, _, verifiers := signedApprovalArtifactsForBrokerTestsWithOutcome(t, "human", "sha256:"+strings.Repeat("d", 64), "approve")
	for _, verifier := range verifiers {
		if err := putTrustedVerifierRecordForService(service, verifier); err != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
		}
	}
	seedPendingApprovalForSignedRequest(t, service, "run-anchor", "step-anchor", "sha256:"+strings.Repeat("d", 64), *requestEnv)
	approvalID, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	decisionDigest, err := digestFromIdentity(approvalID)
	if err != nil {
		t.Fatalf("digestFromIdentity returned error: %v", err)
	}
	return decisionDigest
}

func mustSeedConsumedApprovalForAnchorTest(t *testing.T, service *Service) trustpolicy.Digest {
	t.Helper()
	requestEnv, decisionEnv, verifiers := signedApprovalArtifactsForBrokerTestsWithOutcome(t, "human", "sha256:"+strings.Repeat("d", 64), "approve")
	for _, verifier := range verifiers {
		if err := putTrustedVerifierRecordForService(service, verifier); err != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
		}
	}
	seedPendingApprovalForSignedRequest(t, service, "run-anchor", "step-anchor", "sha256:"+strings.Repeat("d", 64), *requestEnv)
	approvalID, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	stored, ok := service.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	decisionDigestID, err := signedEnvelopeDigest(*decisionEnv)
	if err != nil {
		t.Fatalf("signedEnvelopeDigest returned error: %v", err)
	}
	now := time.Now().UTC()
	stored.Status = "consumed"
	stored.DecisionEnvelope = decisionEnv
	stored.DecisionDigest = decisionDigestID
	stored.DecidedAt = &now
	stored.ConsumedAt = &now
	if err := service.RecordApproval(stored); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	decisionDigest, err := digestFromIdentity(decisionDigestID)
	if err != nil {
		t.Fatalf("digestFromIdentity returned error: %v", err)
	}
	return decisionDigest
}

func mustReadAnchorReceiptSidecar(t *testing.T, ledgerRoot string, digest trustpolicy.Digest) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	id, _ := digest.Identity()
	path := filepath.Join(ledgerRoot, "sidecar", "receipts", strings.TrimPrefix(id, "sha256:")+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", path, err)
	}
	env := trustpolicy.SignedObjectEnvelope{}
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatalf("Unmarshal anchor receipt envelope returned error: %v", err)
	}
	return env
}

func mustAnchorReceiptPayload(t *testing.T, envelope trustpolicy.SignedObjectEnvelope) anchorReceiptPayloadForTest {
	t.Helper()
	receipt := map[string]any{}
	if err := json.Unmarshal(envelope.Payload, &receipt); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	rawPayload, ok := receipt["receipt_payload"]
	if !ok {
		t.Fatal("receipt_payload missing")
	}
	payloadBytes, err := json.Marshal(rawPayload)
	if err != nil {
		t.Fatalf("Marshal receipt_payload returned error: %v", err)
	}
	payload := anchorReceiptPayloadForTest{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("Unmarshal typed receipt_payload returned error: %v", err)
	}
	return payload
}

func mustDigestIdentityForAnchorTest(d trustpolicy.Digest) string {
	id, _ := d.Identity()
	return id
}

func mustAuditAnchorPresenceAttestation(t *testing.T, mode string, sealDigest trustpolicy.Digest) *AuditAnchorPresenceAttestation {
	t.Helper()
	challenge := "presence-challenge-" + strings.Repeat("a", 16)
	token, err := auditAnchorPresenceTokenForBrokerTest(mode, sealDigest, challenge)
	if err != nil {
		t.Fatalf("auditAnchorPresenceTokenForBrokerTest returned error: %v", err)
	}
	return &AuditAnchorPresenceAttestation{Challenge: challenge, AcknowledgmentToken: token}
}

func auditAnchorPresenceTokenForBrokerTest(mode string, sealDigest trustpolicy.Digest, challenge string) (string, error) {
	return secretsd.ComputeAuditAnchorPresenceAcknowledgmentToken(mode, sealDigest, challenge)
}
