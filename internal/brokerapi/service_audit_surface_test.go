package brokerapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestAuditReadinessAndVerificationSurface(t *testing.T) {
	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}

	service, err := NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	readiness, err := service.AuditReadiness()
	if err != nil {
		t.Fatalf("AuditReadiness returned error: %v", err)
	}
	if !readiness.Ready {
		t.Fatal("AuditReadiness.Ready = false, want true")
	}

	surface, err := service.LatestAuditVerificationSurface(10)
	if err != nil {
		t.Fatalf("LatestAuditVerificationSurface returned error: %v", err)
	}
	if !surface.Summary.CryptographicallyValid {
		t.Fatal("summary cryptographically_valid = false, want true")
	}
	if len(surface.Views) != 1 {
		t.Fatalf("views count = %d, want 1", len(surface.Views))
	}
	if surface.Views[0].Event == nil {
		t.Fatal("expected event operational view entry")
	}
}

func TestLatestAuditVerificationSurfaceUsesReportScopedSegment(t *testing.T) {
	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	first, err := buildSeedEventEvidence("session-1")
	if err != nil {
		t.Fatalf("buildSeedEventEvidence(first) returned error: %v", err)
	}
	second, err := buildSeedEventEvidence("session-2")
	if err != nil {
		t.Fatalf("buildSeedEventEvidence(second) returned error: %v", err)
	}
	if err := seedLedgerWithTwoSegmentsAndFirstReport(ledgerRoot, first, second); err != nil {
		t.Fatalf("seedLedgerWithTwoSegmentsAndFirstReport returned error: %v", err)
	}

	service, err := NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	surface, err := service.LatestAuditVerificationSurface(10)
	if err != nil {
		t.Fatalf("LatestAuditVerificationSurface returned error: %v", err)
	}
	if surface.Report.VerificationScope.LastSegmentID != "segment-000001" {
		t.Fatalf("report last_segment_id = %q, want segment-000001", surface.Report.VerificationScope.LastSegmentID)
	}
	if len(surface.Views) != 1 {
		t.Fatalf("views count = %d, want 1", len(surface.Views))
	}
	if surface.Views[0].RecordDigest != first.recordDigest {
		t.Fatalf("view digest = %+v, want %+v", surface.Views[0].RecordDigest, first.recordDigest)
	}
}

func seedLedgerForBrokerSurfaceTest(root string) error {
	if err := prepareLedgerDirs(root); err != nil {
		return err
	}
	evidence, err := buildSeedEventEvidence("session-1")
	if err != nil {
		return err
	}
	if err := writeSeedSegment(root, "segment-000001", evidence.recordDigest, evidence.canonicalEnvelope); err != nil {
		return err
	}
	if _, err := writeSeedSeal(root, "segment-000001", evidence.recordDigest, 0, nil); err != nil {
		return err
	}
	ledger, err := auditd.Open(root)
	if err != nil {
		return err
	}
	if err := configureSeedContractsAndIndex(ledger); err != nil {
		return err
	}
	return persistSeedReport(ledger)
}

func seedLedgerWithTwoSegmentsAndFirstReport(root string, first seedEvidence, second seedEvidence) error {
	if err := prepareLedgerDirs(root); err != nil {
		return err
	}
	if err := writeSeedSegment(root, "segment-000001", first.recordDigest, first.canonicalEnvelope); err != nil {
		return err
	}
	firstSealDigest, err := writeSeedSeal(root, "segment-000001", first.recordDigest, 0, nil)
	if err != nil {
		return err
	}
	if err := writeSeedSegment(root, "segment-000002", second.recordDigest, second.canonicalEnvelope); err != nil {
		return err
	}
	if _, err := writeSeedSeal(root, "segment-000002", second.recordDigest, 1, &firstSealDigest); err != nil {
		return err
	}
	ledger, err := auditd.Open(root)
	if err != nil {
		return err
	}
	if err := configureSeedContractsAndIndex(ledger); err != nil {
		return err
	}
	return persistSeedReport(ledger)
}

type seedEvidence struct {
	recordDigest      trustpolicy.Digest
	canonicalEnvelope []byte
}

type seedRuntimeEvidenceFixture struct {
	facts    launcherbackend.RuntimeFactsSnapshot
	evidence launcherbackend.RuntimeEvidenceSnapshot
}

func prepareLedgerDirs(root string) error {
	for _, path := range []string{filepath.Join(root, "segments"), filepath.Join(root, "sidecar", "segment-seals")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func buildSeedEventEvidence(sessionID string) (seedEvidence, error) {
	fixture, err := buildSeedRuntimeEvidenceFixture(sessionID)
	if err != nil {
		return seedEvidence{}, err
	}
	eventPayload := map[string]any{"session_id": sessionID}
	eventPayloadHash := sha256.Sum256(mustCanonicalJSON(eventPayload))
	event := seedAuditEventEnvelopePayload(sessionID, fixture.evidence, eventPayloadHash)
	envelope := seedSignedEventEnvelope(event)
	canonicalEnvelope := mustCanonicalJSON(envelope)
	sum := sha256.Sum256(canonicalEnvelope)
	return seedEvidence{recordDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, canonicalEnvelope: canonicalEnvelope}, nil
}

func buildSeedRuntimeEvidenceFixture(sessionID string) (seedRuntimeEvidenceFixture, error) {
	runID := "run-1"
	facts := launcherbackend.DefaultRuntimeFacts(runID)
	seedRuntimeFactsLaunchReceipt(&facts, sessionID)
	facts.HardeningPosture = seedRuntimeHardeningPosture()
	evidence, _, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		return seedRuntimeEvidenceFixture{}, err
	}
	return seedRuntimeEvidenceFixture{facts: facts, evidence: evidence}, nil
}

func seedRuntimeFactsLaunchReceipt(facts *launcherbackend.RuntimeFactsSnapshot, sessionID string) {
	facts.LaunchReceipt.BackendKind = "microvm"
	facts.LaunchReceipt.IsolationAssuranceLevel = "isolated"
	facts.LaunchReceipt.ProvisioningPosture = "tofu"
	facts.LaunchReceipt.IsolateID = "isolate-1"
	facts.LaunchReceipt.SessionID = sessionID
	facts.LaunchReceipt.SessionNonce = "seed-nonce-" + sessionID
	facts.LaunchReceipt.LaunchContextDigest = "sha256:" + strings.Repeat("1", 64)
	facts.LaunchReceipt.HandshakeTranscriptHash = "sha256:" + strings.Repeat("2", 64)
	facts.LaunchReceipt.IsolateSessionKeyIDValue = strings.Repeat("a", 64)
	facts.LaunchReceipt.RuntimeImageDescriptorDigest = "sha256:" + strings.Repeat("4", 64)
	facts.LaunchReceipt.RuntimeImageBootProfile = launcherbackend.BootProfileMicroVMLinuxKernelInitrdV1
	facts.LaunchReceipt.BootComponentDigestByName = map[string]string{"kernel": "sha256:" + strings.Repeat("9", 64), "initrd": "sha256:" + strings.Repeat("c", 64)}
	facts.LaunchReceipt.AttestationEvidenceSourceKind = launcherbackend.AttestationSourceKindTrustedRuntime
	facts.LaunchReceipt.AttestationMeasurementProfile = launcherbackend.MeasurementProfileMicroVMBootV1
	facts.LaunchReceipt.AttestationFreshnessMaterial = []string{"nonce:seed"}
	facts.LaunchReceipt.AttestationFreshnessBindingClaims = []string{"launch_evidence_digest", "session_binding"}
	facts.LaunchReceipt.AttestationEvidenceClaimsDigest = "sha256:" + strings.Repeat("6", 64)
	facts.LaunchReceipt.AttestationVerifierPolicyID = "seed-policy"
	facts.LaunchReceipt.AttestationVerifierPolicyDigest = "sha256:" + strings.Repeat("8", 64)
	facts.LaunchReceipt.AttestationVerificationRulesVersion = "v0"
	facts.LaunchReceipt.AttestationVerificationResult = launcherbackend.AttestationVerificationResultValid
	facts.LaunchReceipt.AttestationReplayVerdict = launcherbackend.AttestationReplayVerdictOriginal
	facts.LaunchReceipt.AttestationVerificationTimestamp = "2026-03-13T12:25:00Z"
}

func seedRuntimeHardeningPosture() launcherbackend.AppliedHardeningPosture {
	return launcherbackend.AppliedHardeningPosture{
		Requested:                 launcherbackend.HardeningRequestedHardened,
		Effective:                 launcherbackend.HardeningEffectiveDegraded,
		DegradedReasons:           []string{"seed_test_runtime"},
		ExecutionIdentityPosture:  launcherbackend.HardeningExecutionIdentityUnprivileged,
		RootlessPosture:           launcherbackend.HardeningRootlessEnabled,
		FilesystemExposurePosture: launcherbackend.HardeningFilesystemExposureRestricted,
		WritableLayersPosture:     launcherbackend.HardeningWritableLayersEphemeral,
		NetworkExposurePosture:    launcherbackend.HardeningNetworkExposureRestricted,
		NetworkNamespacePosture:   launcherbackend.HardeningNetworkNamespacePerRole,
		NetworkDefaultPosture:     launcherbackend.HardeningNetworkDefaultLoopbackOnly,
		EgressEnforcementPosture:  launcherbackend.HardeningEgressEnforcementHostLevel,
		SyscallFilteringPosture:   launcherbackend.HardeningSyscallFilteringSeccomp,
		CapabilitiesPosture:       launcherbackend.HardeningCapabilitiesDropped,
		DeviceSurfacePosture:      launcherbackend.HardeningDeviceSurfaceAllowlist,
		ControlChannelKind:        launcherbackend.TransportKindVSock,
		AccelerationKind:          launcherbackend.AccelerationKindNone,
	}
}

func recordSeedRuntimeEvidenceForSession(t testing.TB, service *Service, sessionID string) {
	t.Helper()
	fixture, err := buildSeedRuntimeEvidenceFixture(sessionID)
	if err != nil {
		t.Fatalf("buildSeedRuntimeEvidenceFixture returned error: %v", err)
	}
	if err := service.store.RecordRuntimeEvidenceState(fixture.facts.LaunchReceipt.RunID, fixture.facts, fixture.evidence, launcherbackend.RuntimeLifecycleState{}); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState returned error: %v", err)
	}
}

func seedAuditEventEnvelopePayload(sessionID string, evidence launcherbackend.RuntimeEvidenceSnapshot, eventPayloadHash [32]byte) map[string]any {
	return map[string]any{
		"schema_id":               trustpolicy.AuditEventSchemaID,
		"schema_version":          trustpolicy.AuditEventSchemaVersion,
		"audit_event_type":        "isolate_session_bound",
		"emitter_stream_id":       "auditd-stream-1",
		"seq":                     1,
		"occurred_at":             "2026-03-13T12:15:00Z",
		"principal":               map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"event_payload_schema_id": trustpolicy.IsolateSessionBoundPayloadSchemaID,
		"event_payload": map[string]any{
			"schema_id":                        trustpolicy.IsolateSessionBoundPayloadSchemaID,
			"schema_version":                   trustpolicy.IsolateSessionBoundPayloadSchemaVersion,
			"run_id":                           "run-1",
			"isolate_id":                       "isolate-1",
			"session_id":                       sessionID,
			"backend_kind":                     "microvm",
			"isolation_assurance_level":        "isolated",
			"provisioning_posture":             "tofu",
			"launch_context_digest":            "sha256:" + strings.Repeat("1", 64),
			"handshake_transcript_hash":        "sha256:" + strings.Repeat("2", 64),
			"session_binding_digest":           evidence.Session.EvidenceDigest,
			"runtime_image_descriptor_digest":  evidence.Launch.RuntimeImageDescriptorDigest,
			"applied_hardening_posture_digest": evidence.Hardening.EvidenceDigest,
			"attestation_evidence_digest":      evidence.Attestation.EvidenceDigest,
		},
		"event_payload_hash":            map[string]any{"hash_alg": "sha256", "hash": hex.EncodeToString(eventPayloadHash[:])},
		"protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
		"scope":                         map[string]any{"workspace_id": "workspace-1", "run_id": "run-1", "stage_id": "stage-1"},
		"correlation":                   map[string]any{"session_id": sessionID, "operation_id": "op-1"},
		"subject_ref":                   map[string]any{"object_family": "isolate_binding", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)}, "ref_role": "binding_target"},
		"cause_refs":                    []any{map[string]any{"object_family": "audit_event", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "ref_role": "session_cause"}},
		"related_refs":                  []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)}, "ref_role": "binding"}},
		"signer_evidence_refs":          []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "ref_role": "admissibility"}},
	}
}

func seedSignedEventEnvelope(event map[string]any) trustpolicy.SignedObjectEnvelope {
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditEventSchemaID, PayloadSchemaVersion: trustpolicy.AuditEventSchemaVersion, Payload: mustJSON(event), SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString([]byte("sig"))}}
}

func writeSeedSegment(root string, segmentID string, recordDigest trustpolicy.Digest, canonicalEnvelope []byte) error {
	segment := trustpolicy.AuditSegmentFilePayload{SchemaID: "runecode.protocol.v0.AuditSegmentFile", SchemaVersion: "0.1.0", Header: trustpolicy.AuditSegmentHeader{Format: "audit_segment_framed_v1", SegmentID: segmentID, SegmentState: trustpolicy.AuditSegmentStateSealed, CreatedAt: "2026-03-13T12:00:00Z", Writer: "auditd"}, Frames: []trustpolicy.AuditSegmentRecordFrame{{RecordDigest: recordDigest, ByteLength: int64(len(canonicalEnvelope)), CanonicalSignedEnvelopeBytes: base64.StdEncoding.EncodeToString(canonicalEnvelope)}}, LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateSealed, MarkedAt: "2026-03-13T12:20:00Z"}}
	return writeCanonicalJSON(filepath.Join(root, "segments", segmentID+".json"), segment)
}

func writeSeedSeal(root string, segmentID string, recordDigest trustpolicy.Digest, chainIndex int64, previousSealDigest *trustpolicy.Digest) (trustpolicy.Digest, error) {
	merkleRoot, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot([]trustpolicy.Digest{recordDigest})
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	sealPayload := trustpolicy.AuditSegmentSealPayload{SchemaID: trustpolicy.AuditSegmentSealSchemaID, SchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, SegmentID: segmentID, SealedAfterState: trustpolicy.AuditSegmentStateOpen, SegmentState: trustpolicy.AuditSegmentStateSealed, SegmentCut: trustpolicy.AuditSegmentCutWindowPolicy{OwnershipScope: trustpolicy.AuditSegmentOwnershipScopeInstanceGlobal, MaxSegmentBytes: 2048, CutTrigger: trustpolicy.AuditSegmentCutTriggerSizeWindow}, EventCount: 1, FirstRecordDigest: recordDigest, LastRecordDigest: recordDigest, MerkleProfile: trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1, MerkleRoot: merkleRoot, SegmentFileHashScope: trustpolicy.AuditSegmentFileHashScopeRawFramedV1, SegmentFileHash: recordDigest, SealChainIndex: chainIndex, PreviousSealDigest: previousSealDigest, AnchoringSubject: trustpolicy.AuditSegmentAnchoringSubjectSeal, SealedAt: "2026-03-13T12:20:00Z", ProtocolBundleManifestHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)}, SealReason: "size_threshold"}
	sealEnvelope := trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditSegmentSealSchemaID, PayloadSchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, Payload: mustJSON(sealPayload), SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString([]byte("sig"))}}
	sealDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(sealEnvelope)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	identity, _ := sealDigest.Identity()
	if err := writeCanonicalJSON(filepath.Join(root, "sidecar", "segment-seals", strings.TrimPrefix(identity, "sha256:")+".json"), sealEnvelope); err != nil {
		return trustpolicy.Digest{}, err
	}
	return sealDigest, nil
}

func configureSeedContractsAndIndex(ledger *auditd.Ledger) error {
	if err := ledger.ConfigureVerificationInputs(auditd.VerificationConfiguration{VerifierRecords: []trustpolicy.VerifierRecord{seedVerifierRecord()}, EventContractCatalog: seedEventContractCatalog()}); err != nil {
		return err
	}
	_, err := ledger.BuildIndex()
	return err
}

func persistSeedReport(ledger *auditd.Ledger) error {
	report := trustpolicy.AuditVerificationReportPayload{SchemaID: trustpolicy.AuditVerificationReportSchemaID, SchemaVersion: trustpolicy.AuditVerificationReportSchemaVersion, VerifiedAt: time.Now().UTC().Format(time.RFC3339), VerificationScope: trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: "segment-000001"}, CryptographicallyValid: true, HistoricallyAdmissible: true, CurrentlyDegraded: false, IntegrityStatus: trustpolicy.AuditVerificationStatusOK, AnchoringStatus: trustpolicy.AuditVerificationStatusOK, StoragePostureStatus: trustpolicy.AuditVerificationStatusOK, SegmentLifecycleStatus: trustpolicy.AuditVerificationStatusOK, DegradedReasons: []string{}, HardFailures: []string{}, Findings: []trustpolicy.AuditVerificationFinding{}, Summary: "ok"}
	_, err := ledger.PersistVerificationReport(report)
	return err
}

func seedVerifierRecord() trustpolicy.VerifierRecord {
	publicKey := []byte(strings.Repeat("k", 32))
	keyID := sha256.Sum256(publicKey)
	return trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: hex.EncodeToString(keyID[:]), Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "isolate_session_identity", LogicalScope: "session", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"}, KeyProtectionPosture: "os_keystore", IdentityBindingPosture: "attested", PresenceMode: "os_confirmation", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}
}

func seedEventContractCatalog() trustpolicy.AuditEventContractCatalog {
	return trustpolicy.AuditEventContractCatalog{SchemaID: trustpolicy.AuditEventContractCatalogSchemaID, SchemaVersion: trustpolicy.AuditEventContractCatalogSchemaVersion, CatalogID: "audit_event_contract_v0", Entries: []trustpolicy.AuditEventContractCatalogEntry{{AuditEventType: "isolate_session_bound", AllowedPayloadSchemaIDs: []string{trustpolicy.IsolateSessionBoundPayloadSchemaID}, AllowedSignerPurposes: []string{"isolate_session_identity"}, AllowedSignerScopes: []string{"session"}, RequiredScopeFields: []string{"workspace_id", "run_id", "stage_id"}, RequiredCorrelationFields: []string{"session_id", "operation_id"}, RequireSubjectRef: true, AllowedSubjectRefRoles: []string{"binding_target"}, AllowedCauseRefRoles: []string{"session_cause"}, AllowedRelatedRefRoles: []string{"binding", "evidence", "receipt"}, RequireSignerEvidenceRefs: true, AllowedSignerEvidenceRefRoles: []string{"admissibility", "binding"}}}}
}

func mustJSON(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return b
}

func mustCanonicalJSON(value any) []byte {
	b, err := jsoncanonicalizer.Transform(mustJSON(value))
	if err != nil {
		panic(err)
	}
	return b
}

func writeCanonicalJSON(path string, value any) error {
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
