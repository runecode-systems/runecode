package zkproof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestCompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0Success(t *testing.T) {
	input := validCompileInputFixture(t)
	contract, err := CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0(input)
	if err != nil {
		t.Fatalf("CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0 returned error: %v", err)
	}

	if contract.PublicInputs.StatementFamily != StatementFamilyAuditIsolateSessionBoundAttestedRuntimeMembershipV0 {
		t.Fatalf("statement_family mismatch: got %q", contract.PublicInputs.StatementFamily)
	}
	if contract.PublicInputs.StatementVersion != StatementVersionV0 {
		t.Fatalf("statement_version mismatch: got %q", contract.PublicInputs.StatementVersion)
	}
	if contract.PublicInputs.AttestationEvidenceDigest == "" {
		t.Fatal("attestation_evidence_digest is empty")
	}
	if contract.WitnessInputs.MerkleAuthenticationDepth != 1 {
		t.Fatalf("merkle_authentication_depth mismatch: got %d want 1", contract.WitnessInputs.MerkleAuthenticationDepth)
	}
	if contract.WitnessInputs.PrivateRemainder.RunIDDigest.HashAlg != "sha256" {
		t.Fatalf("run_id_digest hash_alg mismatch: got %q", contract.WitnessInputs.PrivateRemainder.RunIDDigest.HashAlg)
	}
	if contract.WitnessInputs.PrivateRemainder.BackendKindCode == 0 {
		t.Fatal("backend_kind_code is zero")
	}
}

func TestDeriveAndVerifyAuditSegmentMerkleAuthenticationPathV0MatchesAuthoritativeConstruction(t *testing.T) {
	recordDigests := []trustpolicy.Digest{
		deterministicAuditRecordDigestFixtureV0(1),
		deterministicAuditRecordDigestFixtureV0(2),
		deterministicAuditRecordDigestFixtureV0(3),
		deterministicAuditRecordDigestFixtureV0(4),
		deterministicAuditRecordDigestFixtureV0(5),
	}
	root, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot(recordDigests)
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot returned error: %v", err)
	}
	seal := trustpolicy.AuditSegmentSealPayload{
		SchemaID:      trustpolicy.AuditSegmentSealSchemaID,
		SchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion,
		MerkleProfile: trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1,
		MerkleRoot:    root,
	}

	for index := range recordDigests {
		path, err := DeriveAuditSegmentMerkleAuthenticationPathV0(recordDigests, index)
		if err != nil {
			t.Fatalf("DeriveAuditSegmentMerkleAuthenticationPathV0(%d) returned error: %v", index, err)
		}
		if path.PathVersion != MerkleAuthenticationPathFormatV1 {
			t.Fatalf("path_version mismatch: got %q", path.PathVersion)
		}
		if path.LeafIndex != uint64(index) {
			t.Fatalf("leaf_index mismatch: got %d want %d", path.LeafIndex, index)
		}
		if err := VerifyAuditSegmentMerkleAuthenticationPathAgainstSealV0(recordDigests[index], path, seal); err != nil {
			t.Fatalf("VerifyAuditSegmentMerkleAuthenticationPathAgainstSealV0(%d) returned error: %v", index, err)
		}
	}
}

func TestDeriveAuditSegmentMerkleAuthenticationPathV0UsesOrderedSiblingSemanticsAndDuplicateRule(t *testing.T) {
	recordDigests := []trustpolicy.Digest{
		deterministicAuditRecordDigestFixtureV0(10),
		deterministicAuditRecordDigestFixtureV0(11),
		deterministicAuditRecordDigestFixtureV0(12),
	}
	path, err := DeriveAuditSegmentMerkleAuthenticationPathV0(recordDigests, 2)
	if err != nil {
		t.Fatalf("DeriveAuditSegmentMerkleAuthenticationPathV0 returned error: %v", err)
	}
	if len(path.Steps) != 2 {
		t.Fatalf("steps mismatch: got %d want 2", len(path.Steps))
	}
	if path.Steps[0].SiblingPosition != merkleSiblingPositionDuplicate {
		t.Fatalf("first sibling_position mismatch: got %q want %q", path.Steps[0].SiblingPosition, merkleSiblingPositionDuplicate)
	}
	if path.Steps[1].SiblingPosition != merkleSiblingPositionLeft {
		t.Fatalf("second sibling_position mismatch: got %q want %q", path.Steps[1].SiblingPosition, merkleSiblingPositionLeft)
	}
}

func TestDeriveAuditSegmentMerkleAuthenticationPathV0FailsClosedOnDepthBound(t *testing.T) {
	// depth(4097 leaves) = 13, which exceeds MaxMerklePathDepthV0=12
	recordDigests := make([]trustpolicy.Digest, 4097)
	for i := range recordDigests {
		recordDigests[i] = deterministicAuditRecordDigestFixtureV0(uint64(i + 1))
	}
	_, err := DeriveAuditSegmentMerkleAuthenticationPathV0(recordDigests, 4096)
	if err == nil {
		t.Fatal("expected depth-bound failure, got nil")
	}
	ferr, ok := err.(*FeasibilityError)
	if !ok {
		t.Fatalf("expected FeasibilityError, got %T (%v)", err, err)
	}
	if ferr.Code != feasibilityCodeInvalidMerklePath {
		t.Fatalf("code mismatch: got %q want %q", ferr.Code, feasibilityCodeInvalidMerklePath)
	}
	if !strings.Contains(ferr.Message, "exceeds max") {
		t.Fatalf("message mismatch: got %q", ferr.Message)
	}
}

func TestCompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0FailsClosed(t *testing.T) {
	for _, tc := range compileFailsClosedCases(t) {
		t.Run(tc.name, func(t *testing.T) { assertCompileFailsClosed(t, tc) })
	}
}

type compileFailsClosedCase struct {
	name          string
	mutate        func(*CompileAuditIsolateSessionBoundAttestedRuntimeInput)
	errorCode     string
	errorContains string
}

func compileFailsClosedCases(t *testing.T) []compileFailsClosedCase {
	return append(compileFailsClosedCoreCases(t), compileFailsClosedMerkleCases(t)...)
}

func compileFailsClosedCoreCases(t *testing.T) []compileFailsClosedCase {
	return append(compileFailsClosedVerifierCases(), compileFailsClosedEventCases(t)...)
}

func compileFailsClosedVerifierCases() []compileFailsClosedCase {
	return []compileFailsClosedCase{{name: "missing session binding relationship verifier", mutate: func(input *CompileAuditIsolateSessionBoundAttestedRuntimeInput) {
		input.SessionBindingRelationshipVerify = nil
	}, errorCode: feasibilityCodeSessionBindingMismatch, errorContains: "relationship verifier"}, {name: "unsupported commitment deriver", mutate: func(input *CompileAuditIsolateSessionBoundAttestedRuntimeInput) { input.BindingCommitmentDeriver = nil }, errorCode: feasibilityCodeUnsupportedCommitmentDeriver, errorContains: "poseidon-family"}, {name: "non deterministic verification", mutate: func(input *CompileAuditIsolateSessionBoundAttestedRuntimeInput) {
		input.DeterministicVerification = false
	}, errorCode: feasibilityCodeNonDeterministicVerification, errorContains: "deterministic"}}
}

func compileFailsClosedEventCases(t *testing.T) []compileFailsClosedCase {
	return []compileFailsClosedCase{{name: "wrong audit event type", mutate: func(input *CompileAuditIsolateSessionBoundAttestedRuntimeInput) {
		input.VerifiedAuditEvent.AuditEventType = "isolate_session_started"
	}, errorCode: feasibilityCodeIneligibleAuditEvent, errorContains: "audit_event_type"}, {name: "missing attestation evidence digest", mutate: func(input *CompileAuditIsolateSessionBoundAttestedRuntimeInput) {
		clearAttestationEvidenceDigest(t, input)
	}, errorCode: feasibilityCodeMissingBoundedInput, errorContains: "attestation_evidence_digest"}}
}

func clearAttestationEvidenceDigest(t *testing.T, input *CompileAuditIsolateSessionBoundAttestedRuntimeInput) {
	payload := payloadFromEventFixture(t, input.VerifiedAuditEvent)
	payload.AttestationEvidenceDigest = ""
	input.VerifiedAuditEvent.EventPayload = mustJSON(t, payload)
}

func compileFailsClosedMerkleCases(t *testing.T) []compileFailsClosedCase {
	return []compileFailsClosedCase{
		{
			name: "merkle path depth exceeds cap",
			mutate: func(input *CompileAuditIsolateSessionBoundAttestedRuntimeInput) {
				input.MerkleAuthenticationPath.Steps = make([]MerkleAuthenticationStep, MaxMerklePathDepthV0+1)
				for i := range input.MerkleAuthenticationPath.Steps {
					input.MerkleAuthenticationPath.Steps[i] = MerkleAuthenticationStep{SiblingDigest: digestFixture("b"), SiblingPosition: merkleSiblingPositionLeft}
				}
			},
			errorCode:     feasibilityCodeInvalidMerklePath,
			errorContains: "exceeds",
		},
		{
			name: "session binding relationship mismatch",
			mutate: func(input *CompileAuditIsolateSessionBoundAttestedRuntimeInput) {
				input.SessionBindingRelationshipVerify = fakeSessionBindingRelationshipVerifier{err: fmt.Errorf("mismatch")}
			},
			errorCode:     feasibilityCodeSessionBindingMismatch,
			errorContains: "relationship verification failed",
		},
	}
}

func assertCompileFailsClosed(t *testing.T, tc compileFailsClosedCase) {
	t.Helper()
	input := validCompileInputFixture(t)
	tc.mutate(&input)

	_, err := CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ferr, ok := err.(*FeasibilityError)
	if !ok {
		t.Fatalf("expected FeasibilityError, got %T (%v)", err, err)
	}
	if ferr.Code != tc.errorCode {
		t.Fatalf("code mismatch: got %q want %q", ferr.Code, tc.errorCode)
	}
	if !strings.Contains(ferr.Message, tc.errorContains) {
		t.Fatalf("message mismatch: got %q, want contains %q", ferr.Message, tc.errorContains)
	}
}

func TestCanonicalSetupProvenanceDigestV0DeterministicAndConstraintSensitive(t *testing.T) {
	lineage := validSetupLineageFixture()
	oneIdentity, twoIdentity := mustCanonicalSetupDigestPair(t, lineage)
	assertCanonicalSetupDigestsMatch(t, oneIdentity, twoIdentity)
	assertCanonicalSetupDigestChangesOnConstraintDrift(t, lineage, oneIdentity)
}

func validSetupLineageFixture() SetupLineageIdentity {
	return SetupLineageIdentity{Phase1LineageID: "powers-of-tau-bn254-main-v1", Phase1LineageDigest: digestFixture("1"), Phase2TranscriptDigest: digestFixture("2"), FrozenCircuitSourceDig: digestFixture("3"), ConstraintSystemDigest: digestFixture("4"), GnarkModuleVersion: "github.com/consensys/gnark@v0.12.0"}
}

func mustCanonicalSetupDigestPair(t *testing.T, lineage SetupLineageIdentity) (string, string) {
	t.Helper()
	one := mustCanonicalSetupDigestIdentity(t, lineage, "first")
	two := mustCanonicalSetupDigestIdentity(t, lineage, "second")
	return one, two
}

func mustCanonicalSetupDigestIdentity(t *testing.T, lineage SetupLineageIdentity, label string) string {
	t.Helper()
	digest, err := CanonicalSetupProvenanceDigestV0(lineage)
	if err != nil {
		t.Fatalf("CanonicalSetupProvenanceDigestV0(%s) returned error: %v", label, err)
	}
	identity, err := digest.Identity()
	if err != nil {
		t.Fatalf("%s setup digest identity: %v", label, err)
	}
	return identity
}

func assertCanonicalSetupDigestsMatch(t *testing.T, oneIdentity, twoIdentity string) {
	t.Helper()
	if oneIdentity != twoIdentity {
		t.Fatalf("setup_provenance_digest must be deterministic: first=%q second=%q", oneIdentity, twoIdentity)
	}
}

func assertCanonicalSetupDigestChangesOnConstraintDrift(t *testing.T, lineage SetupLineageIdentity, fixedIdentity string) {
	t.Helper()
	mutated := lineage
	mutated.ConstraintSystemDigest = digestFixture("5")
	mutatedIdentity := mustCanonicalSetupDigestIdentity(t, mutated, "mutated")
	if fixedIdentity == mutatedIdentity {
		t.Fatalf("constraint_system_digest drift must change setup provenance digest: fixed=%q mutated=%q", fixedIdentity, mutatedIdentity)
	}
}

func TestVerifySetupIdentityMatchesTrustedPostureV0FailsClosedOnMismatch(t *testing.T) {
	trusted := TrustedVerifierPosture{
		VerifierKeyDigest:      digestFixture("a"),
		ConstraintSystemDigest: digestFixture("b"),
		SetupProvenanceDigest:  digestFixture("c"),
	}
	identity := ProofVerificationIdentity{
		VerifierKeyDigest:      digestFixture("a"),
		ConstraintSystemDigest: digestFixture("b"),
		SetupProvenanceDigest:  digestFixture("c"),
	}
	if err := VerifySetupIdentityMatchesTrustedPostureV0(identity, trusted); err != nil {
		t.Fatalf("expected identity match, got error: %v", err)
	}

	identity.ConstraintSystemDigest = digestFixture("d")
	err := VerifySetupIdentityMatchesTrustedPostureV0(identity, trusted)
	if err == nil {
		t.Fatal("expected setup identity mismatch, got nil")
	}
	ferr, ok := err.(*FeasibilityError)
	if !ok {
		t.Fatalf("expected FeasibilityError, got %T (%v)", err, err)
	}
	if ferr.Code != feasibilityCodeSetupIdentityMismatch {
		t.Fatalf("code mismatch: got %q want %q", ferr.Code, feasibilityCodeSetupIdentityMismatch)
	}
	if !strings.Contains(ferr.Message, "constraint_system_digest") {
		t.Fatalf("message mismatch: got %q", ferr.Message)
	}
}

func TestVerifyProofWithTrustedPostureV0FailClosedWhenBackendUnconfigured(t *testing.T) {
	err := VerifyProofWithTrustedPostureV0(
		nil,
		[]byte{0x1, 0x2},
		AuditIsolateSessionBoundAttestedRuntimePublicInputs{
			StatementFamily:   StatementFamilyAuditIsolateSessionBoundAttestedRuntimeMembershipV0,
			StatementVersion:  StatementVersionV0,
			SchemeAdapterID:   SchemeAdapterGnarkGroth16IsolateSessionBoundV0,
			MerkleRoot:        digestFixture("9"),
			AuditRecordDigest: digestFixture("a"),
			BindingCommitment: digestIdentityFixture("b"),
		},
		ProofVerificationIdentity{
			VerifierKeyDigest:      digestFixture("a"),
			ConstraintSystemDigest: digestFixture("b"),
			SetupProvenanceDigest:  digestFixture("c"),
		},
		TrustedVerifierPosture{
			VerifierKeyDigest:      digestFixture("a"),
			ConstraintSystemDigest: digestFixture("b"),
			SetupProvenanceDigest:  digestFixture("c"),
		},
	)
	if err == nil {
		t.Fatal("expected unconfigured backend failure, got nil")
	}
	ferr, ok := err.(*FeasibilityError)
	if !ok {
		t.Fatalf("expected FeasibilityError, got %T (%v)", err, err)
	}
	if ferr.Code != feasibilityCodeUnconfiguredProofBackend {
		t.Fatalf("code mismatch: got %q want %q", ferr.Code, feasibilityCodeUnconfiguredProofBackend)
	}
}

func TestFrozenCircuitIdentityValidateV0(t *testing.T) {
	identity := FrozenCircuitIdentity{
		SchemeID:               ProofSchemeIDGroth16V0,
		CurveID:                ProofCurveIDBN254V0,
		CircuitID:              "runecode.zk.circuit.audit.isolate_session_bound.attested_runtime_membership.v0",
		ConstraintSystemDigest: digestFixture("a"),
	}
	if err := identity.ValidateV0(); err != nil {
		t.Fatalf("expected valid frozen circuit identity, got error: %v", err)
	}

	identity.CurveID = "bls12_381"
	err := identity.ValidateV0()
	if err == nil {
		t.Fatal("expected unsupported curve error, got nil")
	}
	ferr, ok := err.(*FeasibilityError)
	if !ok {
		t.Fatalf("expected FeasibilityError, got %T (%v)", err, err)
	}
	if ferr.Code != feasibilityCodeUnsupportedProofBackend {
		t.Fatalf("code mismatch: got %q want %q", ferr.Code, feasibilityCodeUnsupportedProofBackend)
	}
}

func validCompileInputFixture(t *testing.T) CompileAuditIsolateSessionBoundAttestedRuntimeInput {
	t.Helper()
	payload := validIsolateSessionBoundPayloadFixture(t)
	return CompileAuditIsolateSessionBoundAttestedRuntimeInput{
		DeterministicVerification: true,
		VerifiedAuditEvent:        validAuditEventFixture(t, payload),
		VerifiedAuditRecordDigest: digestFixture("9"),
		VerifiedAuditSegmentSeal: trustpolicy.AuditSegmentSealPayload{
			SchemaID:      trustpolicy.AuditSegmentSealSchemaID,
			SchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion,
			MerkleProfile: trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1,
			MerkleRoot:    mustMerkleRootForFixture(t, []trustpolicy.Digest{digestFixture("9"), digestFixture("f")}),
		},
		VerifiedAuditSegmentSealDigest:   digestFixture("b"),
		MerkleAuthenticationPath:         mustMerklePathForFixture(t, []trustpolicy.Digest{digestFixture("9"), digestFixture("f")}, 0),
		BindingCommitmentDeriver:         fakeBindingCommitmentDeriver{digest: digestIdentityFixture("d")},
		SessionBindingRelationshipVerify: fakeSessionBindingRelationshipVerifier{},
		ProjectSubstrateSnapshotDigest:   digestIdentityFixture("e"),
	}
}

func validIsolateSessionBoundPayloadFixture(t *testing.T) trustpolicy.IsolateSessionBoundPayload {
	return trustpolicy.IsolateSessionBoundPayload{SchemaID: trustpolicy.IsolateSessionBoundPayloadSchemaID, SchemaVersion: trustpolicy.IsolateSessionBoundPayloadSchemaVersion, RunID: "run-1", IsolateID: "iso-1", SessionID: "sess-1", BackendKind: "microvm", IsolationAssuranceLevel: "isolated", ProvisioningPosture: "attested", LaunchContextDigest: digestIdentityFixture("1"), HandshakeTranscriptHash: digestIdentityFixture("2"), SessionBindingDigest: fixtureSessionBindingDigest(t, "run-1", "iso-1", "sess-1", "microvm", "isolated", "attested", digestIdentityFixture("1"), digestIdentityFixture("2")), RuntimeImageDescriptorDigest: digestIdentityFixture("4"), AppliedHardeningPostureDigest: digestIdentityFixture("5"), AttestationEvidenceDigest: digestIdentityFixture("6")}
}

func validAuditEventFixture(t *testing.T, payload trustpolicy.IsolateSessionBoundPayload) trustpolicy.AuditEventPayload {
	return trustpolicy.AuditEventPayload{SchemaID: trustpolicy.AuditEventSchemaID, SchemaVersion: trustpolicy.AuditEventSchemaVersion, AuditEventType: "isolate_session_bound", EmitterStreamID: "stream-1", Seq: 1, OccurredAt: "2026-03-13T12:20:00Z", EventPayloadSchemaID: trustpolicy.IsolateSessionBoundPayloadSchemaID, EventPayload: mustJSON(t, payload), EventPayloadHash: digestFixture("7"), ProtocolBundleManifestHash: digestFixture("8")}
}

func mustMerkleRootForFixture(t *testing.T, digests []trustpolicy.Digest) trustpolicy.Digest {
	t.Helper()
	root, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot(digests)
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot fixture: %v", err)
	}
	return root
}

func mustMerklePathForFixture(t *testing.T, digests []trustpolicy.Digest, leafIndex int) MerkleAuthenticationPath {
	t.Helper()
	path, err := DeriveAuditSegmentMerkleAuthenticationPathV0(digests, leafIndex)
	if err != nil {
		t.Fatalf("DeriveAuditSegmentMerkleAuthenticationPathV0 fixture: %v", err)
	}
	return path
}

type fakeBindingCommitmentDeriver struct {
	digest string
	err    error
}

func (f fakeBindingCommitmentDeriver) DeriveBindingCommitment(adapterProfileID string, normalized IsolateSessionBoundPrivateRemainder) (string, error) {
	_ = adapterProfileID
	_ = normalized
	if f.err != nil {
		return "", f.err
	}
	return f.digest, nil
}

type fakeSessionBindingRelationshipVerifier struct {
	err error
}

func (f fakeSessionBindingRelationshipVerifier) VerifyNormalizedPrivateRemainderSessionBinding(normalized IsolateSessionBoundPrivateRemainder, sourceSessionBindingDigest string) error {
	_ = normalized
	_ = sourceSessionBindingDigest
	return f.err
}

func fixtureSessionBindingDigest(t *testing.T, runID, isolateID, sessionID, backendKind, assuranceLevel, provisioningPosture, launchContextDigest, handshakeDigest string) string {
	t.Helper()
	normalized, err := normalizePrivateRemainderV0(trustpolicy.IsolateSessionBoundPayload{
		RunID:                   runID,
		IsolateID:               isolateID,
		SessionID:               sessionID,
		BackendKind:             backendKind,
		IsolationAssuranceLevel: assuranceLevel,
		ProvisioningPosture:     provisioningPosture,
		LaunchContextDigest:     launchContextDigest,
		HandshakeTranscriptHash: handshakeDigest,
	})
	if err != nil {
		t.Fatalf("normalizePrivateRemainderV0 fixture: %v", err)
	}
	b, err := json.Marshal(struct {
		RunIDDigest                   string `json:"run_id_digest"`
		IsolateIDDigest               string `json:"isolate_id_digest"`
		SessionIDDigest               string `json:"session_id_digest"`
		BackendKindCode               uint16 `json:"backend_kind_code"`
		IsolationAssuranceLevelCode   uint16 `json:"isolation_assurance_level_code"`
		ProvisioningPostureCode       uint16 `json:"provisioning_posture_code"`
		LaunchContextDigest           string `json:"launch_context_digest"`
		HandshakeTranscriptHashDigest string `json:"handshake_transcript_hash_digest"`
	}{
		RunIDDigest:                   mustDigestIdentity(t, normalized.RunIDDigest),
		IsolateIDDigest:               mustDigestIdentity(t, normalized.IsolateIDDigest),
		SessionIDDigest:               mustDigestIdentity(t, normalized.SessionIDDigest),
		BackendKindCode:               normalized.BackendKindCode,
		IsolationAssuranceLevelCode:   normalized.IsolationAssuranceLevelCode,
		ProvisioningPostureCode:       normalized.ProvisioningPostureCode,
		LaunchContextDigest:           mustDigestIdentity(t, normalized.LaunchContextDigest),
		HandshakeTranscriptHashDigest: mustDigestIdentity(t, normalized.HandshakeTranscriptHashDigest),
	})
	if err != nil {
		t.Fatalf("marshal fixture session binding input: %v", err)
	}
	return digestIdentityForBytes("runecode.zkproof.fixture.session_binding.v0:", b)
}

func mustDigestIdentity(t *testing.T, digest trustpolicy.Digest) string {
	t.Helper()
	identity, err := digest.Identity()
	if err != nil {
		t.Fatalf("digest identity: %v", err)
	}
	return identity
}

func digestIdentityForBytes(prefix string, payload []byte) string {
	sum := sha256.Sum256(append([]byte(prefix), payload...))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func payloadFromEventFixture(t *testing.T, event trustpolicy.AuditEventPayload) trustpolicy.IsolateSessionBoundPayload {
	t.Helper()
	payload := trustpolicy.IsolateSessionBoundPayload{}
	if err := json.Unmarshal(event.EventPayload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return b
}

func digestFixture(char string) trustpolicy.Digest {
	return trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat(char, 64)}
}

func digestIdentityFixture(char string) string {
	return "sha256:" + strings.Repeat(char, 64)
}
