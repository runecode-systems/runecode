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

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestValidateAuditAdmissionRequestAcceptsValidEnvelopeAndContracts(t *testing.T) {
	request := validAuditAdmissionRequestFixture(t)
	if err := ValidateAuditAdmissionRequest(request); err != nil {
		t.Fatalf("ValidateAuditAdmissionRequest returned error: %v", err)
	}
}

func TestValidateIsolateSessionBoundPayloadAcceptsNotApplicableProvisioningPosture(t *testing.T) {
	payload := IsolateSessionBoundPayload{
		SchemaID:                      IsolateSessionBoundPayloadSchemaID,
		SchemaVersion:                 IsolateSessionBoundPayloadSchemaVersion,
		RunID:                         "run-1",
		IsolateID:                     "isolate-1",
		SessionID:                     "session-1",
		BackendKind:                   "container",
		IsolationAssuranceLevel:       "degraded",
		ProvisioningPosture:           "not_applicable",
		LaunchContextDigest:           "sha256:" + strings.Repeat("1", 64),
		HandshakeTranscriptHash:       "sha256:" + strings.Repeat("2", 64),
		SessionBindingDigest:          "sha256:" + strings.Repeat("3", 64),
		RuntimeImageDescriptorDigest:  "sha256:" + strings.Repeat("4", 64),
		AppliedHardeningPostureDigest: "sha256:" + strings.Repeat("5", 64),
	}
	if err := validateIsolateSessionBoundPayload(payload); err != nil {
		t.Fatalf("validateIsolateSessionBoundPayload returned error: %v", err)
	}
}

func TestValidateAuditAdmissionRequestFailsClosedOnMissingSignerEvidence(t *testing.T) {
	request := validAuditAdmissionRequestFixture(t)
	request.SignerEvidence = nil
	if err := ValidateAuditAdmissionRequest(request); err == nil {
		t.Fatal("ValidateAuditAdmissionRequest expected signer evidence failure")
	}
}

func TestValidateAuditAdmissionRequestFailsClosedOnCatalogMismatch(t *testing.T) {
	request := validAuditAdmissionRequestFixture(t)
	for index := range request.EventContractCatalog.Entries {
		if request.EventContractCatalog.Entries[index].AuditEventType == "isolate_session_bound" {
			request.EventContractCatalog.Entries[index].AllowedPayloadSchemaIDs = []string{"runecode.protocol.audit.payload.other.v0"}
			break
		}
	}
	if err := ValidateAuditAdmissionRequest(request); err == nil {
		t.Fatal("ValidateAuditAdmissionRequest expected event contract mismatch")
	}
}

func TestValidateAuditAdmissionRequestChecksEnvelopeSignerWhenSignerEvidenceRefsAreEmpty(t *testing.T) {
	request := validAuditAdmissionRequestFixture(t)
	request.Envelope.Payload = payloadWithoutSignerEvidenceRefs(t, request.Envelope.Payload)
	request.SignerEvidence[0].Evidence.SignerPurpose = "approval_authority"
	if err := ValidateAuditAdmissionRequest(request); err == nil {
		t.Fatal("ValidateAuditAdmissionRequest expected signer admissibility failure")
	}
}

func TestValidateAuditAdmissionRequestRejectsSignerEvidenceRefBoundToDifferentSigner(t *testing.T) {
	request := validAuditAdmissionRequestFixture(t)
	_, _, otherKeyID := generateAuditFixtureKeyMaterial(t)
	request.SignerEvidence = append(request.SignerEvidence, buildSignerEvidenceReferenceFixtureWithDigest(otherKeyID, strings.Repeat("9", 64)))
	request.Envelope.Payload = payloadWithSignerEvidenceRefDigest(t, request.Envelope.Payload, strings.Repeat("9", 64))
	if err := ValidateAuditAdmissionRequest(request); err == nil {
		t.Fatal("ValidateAuditAdmissionRequest expected signer evidence binding failure")
	}
}

func TestEvaluateAuditSegmentRecoveryRules(t *testing.T) {
	sealedBad := AuditSegmentRecoveryState{
		SegmentID:            "segment-0001",
		HeaderState:          "sealed",
		LifecycleMarkerState: "sealed",
		FrameIntegrityOK:     false,
		SealIntegrityOK:      true,
	}
	decision, err := EvaluateAuditSegmentRecovery(sealedBad)
	if err != nil {
		t.Fatalf("EvaluateAuditSegmentRecovery returned error: %v", err)
	}
	if !decision.Quarantine || !decision.FailClosed {
		t.Fatalf("sealed mismatch decision = %+v, want quarantine + fail_closed", decision)
	}

	openTorn := AuditSegmentRecoveryState{
		SegmentID:            "segment-0002",
		HeaderState:          "open",
		LifecycleMarkerState: "open",
		HasTornTrailingFrame: true,
		FrameIntegrityOK:     true,
		SealIntegrityOK:      false,
	}
	decision, err = EvaluateAuditSegmentRecovery(openTorn)
	if err != nil {
		t.Fatalf("EvaluateAuditSegmentRecovery returned error: %v", err)
	}
	if !decision.TruncateTrailingFrame || decision.FailClosed {
		t.Fatalf("open torn decision = %+v, want truncate without fail_closed", decision)
	}
}

func TestValidateAuditStoragePostureEvidenceFailsOnSilentPlaintextFallback(t *testing.T) {
	evidence := AuditStoragePostureEvidence{
		EncryptedAtRestDefault:   true,
		EncryptedAtRestEffective: false,
		SurfacedToOperator:       true,
	}
	if err := ValidateAuditStoragePostureEvidence(evidence); err == nil {
		t.Fatal("ValidateAuditStoragePostureEvidence expected forbidden plaintext fallback")
	}
}

func TestValidateAuditStoragePostureEvidenceAllowsExplicitDevDegradedPosture(t *testing.T) {
	evidence := AuditStoragePostureEvidence{
		EncryptedAtRestDefault:     true,
		EncryptedAtRestEffective:   false,
		DevPlaintextOverrideActive: true,
		DevPlaintextOverrideReason: "dev_local_filesystem_without_encryption",
		SurfacedToOperator:         true,
	}
	if err := ValidateAuditStoragePostureEvidence(evidence); err != nil {
		t.Fatalf("ValidateAuditStoragePostureEvidence returned error: %v", err)
	}
}

func TestValidateAuditdReadinessContractRequiresAllDimensions(t *testing.T) {
	readiness := AuditdReadiness{
		LocalOnly:                 true,
		ConsumptionChannel:        "broker_local_api",
		RecoveryComplete:          true,
		AppendPositionStable:      true,
		CurrentSegmentWritable:    true,
		VerifierMaterialAvailable: false,
		DerivedIndexCaughtUp:      true,
		Ready:                     true,
	}
	if err := ValidateAuditdReadinessContract(readiness); err == nil {
		t.Fatal("ValidateAuditdReadinessContract expected ready mismatch error")
	}
}

func TestValidateAuditSegmentSealPayloadAcceptsExplicitWindowedSeal(t *testing.T) {
	seal := validAuditSegmentSealPayloadFixture()
	if err := ValidateAuditSegmentSealPayload(seal); err != nil {
		t.Fatalf("ValidateAuditSegmentSealPayload returned error: %v", err)
	}
}

func TestValidateAuditSegmentSealPayloadFailsClosedOnPerRunOwnership(t *testing.T) {
	seal := validAuditSegmentSealPayloadFixture()
	seal.SegmentCut.OwnershipScope = "per_run"
	if err := ValidateAuditSegmentSealPayload(seal); err == nil {
		t.Fatal("ValidateAuditSegmentSealPayload expected ownership_scope failure")
	}
}

func TestValidateAuditSegmentSealChainLinkRequiresMatchingPreviousDigest(t *testing.T) {
	first := validAuditSegmentSealPayloadFixture()
	first.SealChainIndex = 0
	first.PreviousSealDigest = nil
	if err := ValidateAuditSegmentSealChainLink(first, nil); err != nil {
		t.Fatalf("ValidateAuditSegmentSealChainLink returned error for genesis seal: %v", err)
	}

	previous := testDigestFromByte('a')
	next := validAuditSegmentSealPayloadFixture()
	next.SealChainIndex = 1
	next.PreviousSealDigest = &previous
	if err := ValidateAuditSegmentSealChainLink(next, &previous); err != nil {
		t.Fatalf("ValidateAuditSegmentSealChainLink returned error for matching chain link: %v", err)
	}

	other := testDigestFromByte('b')
	if err := ValidateAuditSegmentSealChainLink(next, &other); err == nil {
		t.Fatal("ValidateAuditSegmentSealChainLink expected mismatch failure")
	}
}

func TestComputeOrderedAuditSegmentMerkleRootIsDeterministicAndOrderSensitive(t *testing.T) {
	digests := []Digest{testDigestFromByte('1'), testDigestFromByte('2'), testDigestFromByte('3')}
	rootA, err := ComputeOrderedAuditSegmentMerkleRoot(digests)
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot returned error: %v", err)
	}
	rootB, err := ComputeOrderedAuditSegmentMerkleRoot(digests)
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot returned error: %v", err)
	}
	if rootA != rootB {
		t.Fatalf("merkle root should be deterministic, got %v and %v", rootA, rootB)
	}

	reversed := []Digest{digests[2], digests[1], digests[0]}
	reversedRoot, err := ComputeOrderedAuditSegmentMerkleRoot(reversed)
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot returned error for reversed input: %v", err)
	}
	if rootA == reversedRoot {
		t.Fatalf("merkle root should be order-sensitive, got same root %v", rootA)
	}

	if err := VerifyOrderedAuditSegmentMerkleRoot(digests, rootA); err != nil {
		t.Fatalf("VerifyOrderedAuditSegmentMerkleRoot returned error: %v", err)
	}
	if err := VerifyOrderedAuditSegmentMerkleRoot(reversed, rootA); err == nil {
		t.Fatal("VerifyOrderedAuditSegmentMerkleRoot expected mismatch for reordered leaves")
	}
}

func TestComputeAndVerifySegmentFileHashUsesExactRawBytes(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03, 0xff, 0x00, 0x10}
	hash, err := ComputeSegmentFileHash(raw)
	if err != nil {
		t.Fatalf("ComputeSegmentFileHash returned error: %v", err)
	}
	if err := VerifySegmentFileHash(raw, hash); err != nil {
		t.Fatalf("VerifySegmentFileHash returned error: %v", err)
	}

	mutated := append([]byte{}, raw...)
	mutated[2] = 0x09
	if err := VerifySegmentFileHash(mutated, hash); err == nil {
		t.Fatal("VerifySegmentFileHash expected mismatch for mutated bytes")
	}
}

func validAuditSegmentSealPayloadFixture() AuditSegmentSealPayload {
	return AuditSegmentSealPayload{
		SchemaID:                   AuditSegmentSealSchemaID,
		SchemaVersion:              AuditSegmentSealSchemaVersion,
		SegmentID:                  "segment-0001",
		SealedAfterState:           AuditSegmentStateOpen,
		SegmentState:               AuditSegmentStateSealed,
		SegmentCut:                 AuditSegmentCutWindowPolicy{OwnershipScope: AuditSegmentOwnershipScopeInstanceGlobal, MaxSegmentBytes: 1024, CutTrigger: AuditSegmentCutTriggerSizeWindow},
		EventCount:                 3,
		FirstRecordDigest:          testDigestFromByte('1'),
		LastRecordDigest:           testDigestFromByte('3'),
		MerkleProfile:              AuditSegmentMerkleProfileOrderedDSEv1,
		MerkleRoot:                 testDigestFromByte('4'),
		SegmentFileHashScope:       AuditSegmentFileHashScopeRawFramedV1,
		SegmentFileHash:            testDigestFromByte('5'),
		SealChainIndex:             0,
		AnchoringSubject:           AuditSegmentAnchoringSubjectSeal,
		SealedAt:                   "2026-03-13T12:20:00Z",
		ProtocolBundleManifestHash: testDigestFromByte('6'),
		SealReason:                 "size_threshold",
	}
}

func testDigestFromByte(ch byte) Digest {
	return Digest{HashAlg: "sha256", Hash: strings.Repeat(string(ch), 64)}
}

func validAuditAdmissionRequestFixture(t *testing.T) AuditAdmissionRequest {
	t.Helper()
	publicKey, privateKey, keyIDValue := generateAuditFixtureKeyMaterial(t)
	payloadBytes := buildAuditAdmissionEventPayloadBytes(t)
	signature := signAuditAdmissionPayload(t, privateKey, payloadBytes)
	return AuditAdmissionRequest{
		Checks: AuditAdmissionChecks{
			SchemaValidation:         true,
			EventContractValidation:  true,
			SignerEvidenceValidation: true,
			DetachedSignatureVerify:  true,
		},
		Envelope: SignedObjectEnvelope{
			SchemaID:             EnvelopeSchemaID,
			SchemaVersion:        EnvelopeSchemaVersion,
			PayloadSchemaID:      AuditEventSchemaID,
			PayloadSchemaVersion: AuditEventSchemaVersion,
			Payload:              payloadBytes,
			SignatureInput:       SignatureInputProfile,
			Signature: SignatureBlock{
				Alg:        "ed25519",
				KeyID:      KeyIDProfile,
				KeyIDValue: keyIDValue,
				Signature:  base64.StdEncoding.EncodeToString(signature),
			},
		},
		VerifierRecords:      []VerifierRecord{buildAuditAdmissionVerifierRecord(publicKey, keyIDValue)},
		EventContractCatalog: buildAuditEventContractCatalogFixture(),
		SignerEvidence:       []AuditSignerEvidenceReference{buildSignerEvidenceReferenceFixture(keyIDValue)},
	}
}

func generateAuditFixtureKeyMaterial(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	keyID := sha256.Sum256(publicKey)
	return publicKey, privateKey, hex.EncodeToString(keyID[:])
}

func buildAuditAdmissionEventPayloadBytes(t *testing.T) json.RawMessage {
	t.Helper()
	eventPayload := baseIsolateSessionBoundPayloadFixture()
	eventPayloadHash := hashCanonicalJSONFixture(t, eventPayload)
	payload := buildAuditAdmissionEnvelopePayload(eventPayload, eventPayloadHash)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	return payloadBytes
}

func baseIsolateSessionBoundPayloadFixture() map[string]any {
	return map[string]any{
		"schema_id":                        IsolateSessionBoundPayloadSchemaID,
		"schema_version":                   IsolateSessionBoundPayloadSchemaVersion,
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
}

func buildAuditAdmissionEnvelopePayload(eventPayload map[string]any, eventPayloadHash string) map[string]any {
	return map[string]any{
		"schema_id":                     AuditEventSchemaID,
		"schema_version":                AuditEventSchemaVersion,
		"audit_event_type":              "isolate_session_bound",
		"emitter_stream_id":             "auditd-stream-1",
		"seq":                           1,
		"occurred_at":                   "2026-03-13T12:15:00Z",
		"principal":                     map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"event_payload_schema_id":       IsolateSessionBoundPayloadSchemaID,
		"event_payload":                 eventPayload,
		"event_payload_hash":            map[string]any{"hash_alg": "sha256", "hash": eventPayloadHash},
		"protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
		"scope":                         map[string]any{"workspace_id": "workspace-1", "run_id": "run-1", "stage_id": "stage-1"},
		"correlation":                   map[string]any{"session_id": "session-1", "operation_id": "op-1"},
		"subject_ref":                   map[string]any{"object_family": "isolate_binding", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)}, "ref_role": "binding_target"},
		"cause_refs":                    []any{map[string]any{"object_family": "audit_event", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "ref_role": "session_cause"}},
		"related_refs":                  []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)}, "ref_role": "binding"}},
		"signer_evidence_refs":          []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "ref_role": "admissibility"}},
	}
}

func hashCanonicalJSONFixture(t *testing.T, value any) string {
	t.Helper()
	valueBytes, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal value returned error: %v", err)
	}
	canonicalValue, err := jsoncanonicalizer.Transform(valueBytes)
	if err != nil {
		t.Fatalf("Transform value returned error: %v", err)
	}
	sum := sha256.Sum256(canonicalValue)
	return hex.EncodeToString(sum[:])
}

func signAuditAdmissionPayload(t *testing.T, privateKey ed25519.PrivateKey, payload json.RawMessage) []byte {
	t.Helper()
	canonicalPayload, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		t.Fatalf("Transform payload returned error: %v", err)
	}
	return ed25519.Sign(privateKey, canonicalPayload)
}

func buildAuditAdmissionVerifierRecord(publicKey ed25519.PublicKey, keyIDValue string) VerifierRecord {
	return VerifierRecord{
		SchemaID:               VerifierSchemaID,
		SchemaVersion:          VerifierSchemaVersion,
		KeyID:                  KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "approval_authority",
		LogicalScope:           "deployment",
		OwnerPrincipal:         PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 "active",
	}
}

func buildAuditEventContractCatalogFixture() AuditEventContractCatalog {
	return AuditEventContractCatalog{
		SchemaID:      AuditEventContractCatalogSchemaID,
		SchemaVersion: AuditEventContractCatalogSchemaVersion,
		CatalogID:     "audit_event_contract_v0",
		Entries:       []AuditEventContractCatalogEntry{auditEventContractStartedEntry(), auditEventContractBoundEntry()},
	}
}

func auditEventContractStartedEntry() AuditEventContractCatalogEntry {
	return AuditEventContractCatalogEntry{
		AuditEventType:                 "isolate_session_started",
		AllowedPayloadSchemaIDs:        []string{IsolateSessionStartedPayloadSchemaID},
		AllowedSignerPurposes:          []string{"isolate_session_identity"},
		AllowedSignerScopes:            []string{"session"},
		RequiredScopeFields:            []string{"workspace_id", "run_id"},
		RequiredCorrelationFields:      []string{"session_id", "operation_id"},
		RequireSubjectRef:              false,
		AllowedSubjectRefRoles:         []string{},
		AllowedCauseRefRoles:           []string{},
		AllowedRelatedRefRoles:         []string{"binding", "evidence", "receipt"},
		RequireGatewayContext:          false,
		AllowedGatewayEgressCategories: []string{},
		RequireSignerEvidenceRefs:      false,
		AllowedSignerEvidenceRefRoles:  []string{"admissibility", "binding"},
	}
}

func auditEventContractBoundEntry() AuditEventContractCatalogEntry {
	return AuditEventContractCatalogEntry{
		AuditEventType:                 "isolate_session_bound",
		AllowedPayloadSchemaIDs:        []string{IsolateSessionBoundPayloadSchemaID},
		AllowedSignerPurposes:          []string{"isolate_session_identity"},
		AllowedSignerScopes:            []string{"session"},
		RequiredScopeFields:            []string{"workspace_id", "run_id", "stage_id"},
		RequiredCorrelationFields:      []string{"session_id", "operation_id"},
		RequireSubjectRef:              true,
		AllowedSubjectRefRoles:         []string{"binding_target"},
		AllowedCauseRefRoles:           []string{"session_cause"},
		AllowedRelatedRefRoles:         []string{"binding", "evidence", "receipt"},
		RequireGatewayContext:          false,
		AllowedGatewayEgressCategories: []string{},
		RequireSignerEvidenceRefs:      true,
		AllowedSignerEvidenceRefRoles:  []string{"admissibility", "binding"},
	}
}

func buildSignerEvidenceReferenceFixture(keyIDValue string) AuditSignerEvidenceReference {
	return buildSignerEvidenceReferenceFixtureWithDigest(keyIDValue, strings.Repeat("f", 64))
}

func buildSignerEvidenceReferenceFixtureWithDigest(keyIDValue string, digestHash string) AuditSignerEvidenceReference {
	return AuditSignerEvidenceReference{
		Digest: Digest{HashAlg: "sha256", Hash: digestHash},
		Evidence: AuditSignerEvidence{
			SignerPurpose: "isolate_session_identity",
			SignerScope:   "session",
			SignerKey: SignatureBlock{
				Alg:        "ed25519",
				KeyID:      KeyIDProfile,
				KeyIDValue: keyIDValue,
				Signature:  "c2ln",
			},
			IsolateBinding: &IsolateSessionBinding{
				RunID:                   "run-1",
				IsolateID:               "isolate-1",
				SessionID:               "session-1",
				SessionNonce:            "nonce-0123456789abcd",
				ProvisioningMode:        "tofu",
				ImageDigest:             Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)},
				ActiveManifestHash:      Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)},
				HandshakeTranscriptHash: Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)},
				KeyID:                   KeyIDProfile,
				KeyIDValue:              keyIDValue,
				IdentityBindingPosture:  "tofu",
			},
		},
	}
}

func payloadWithSignerEvidenceRefDigest(t *testing.T, payload json.RawMessage, digestHash string) json.RawMessage {
	t.Helper()
	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("Unmarshal payload returned error: %v", err)
	}
	event["signer_evidence_refs"] = []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": digestHash}, "ref_role": "admissibility"}}
	updatedPayload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	return updatedPayload
}

func payloadWithoutSignerEvidenceRefs(t *testing.T, payload json.RawMessage) json.RawMessage {
	t.Helper()
	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("Unmarshal payload returned error: %v", err)
	}
	delete(event, "signer_evidence_refs")
	updatedPayload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	return updatedPayload
}
