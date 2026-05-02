package auditd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestPersistAuditProofBindingDistinctWhenPayloadChanges(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	recordDigest := mustRecordDigestForTest(t, ledger)

	payload := validAuditProofBindingPayloadFixture(recordDigest, sealResult.SealEnvelopeDigest)
	oneDigest, created, err := ledger.PersistAuditProofBinding(payload)
	if err != nil {
		t.Fatalf("PersistAuditProofBinding(first) returned error: %v", err)
	}
	if !created {
		t.Fatal("PersistAuditProofBinding(first) created=false, want true")
	}

	payload.BindingCommitment = "sha256:" + strings.Repeat("e", 64)
	twoDigest, created, err := ledger.PersistAuditProofBinding(payload)
	if err != nil {
		t.Fatalf("PersistAuditProofBinding(second) returned error: %v", err)
	}
	if !created {
		t.Fatal("PersistAuditProofBinding(second) created=false, want true for changed payload")
	}
	if mustDigestIdentity(twoDigest) == mustDigestIdentity(oneDigest) {
		t.Fatal("PersistAuditProofBinding produced same digest for changed payload")
	}
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, proofBindingsDirName), mustDigestIdentity(oneDigest))
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, proofBindingsDirName), mustDigestIdentity(twoDigest))
}

func TestPersistAuditProofBindingCreatesDistinctRecordsAcrossAdapterIdentity(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	recordDigest := mustRecordDigestForTest(t, ledger)

	one := validAuditProofBindingPayloadFixture(recordDigest, sealResult.SealEnvelopeDigest)
	oneDigest, created, err := ledger.PersistAuditProofBinding(one)
	if err != nil {
		t.Fatalf("PersistAuditProofBinding(first) returned error: %v", err)
	}
	if !created {
		t.Fatal("PersistAuditProofBinding(first) created=false, want true")
	}

	two := one
	two.SchemeAdapterID = "runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v1"
	twoDigest, created, err := ledger.PersistAuditProofBinding(two)
	if err != nil {
		t.Fatalf("PersistAuditProofBinding(second) returned error: %v", err)
	}
	if !created {
		t.Fatal("PersistAuditProofBinding(second) created=false, want true for new adapter identity")
	}
	if mustDigestIdentity(twoDigest) == mustDigestIdentity(oneDigest) {
		t.Fatal("PersistAuditProofBinding produced same digest across different scheme_adapter_id")
	}
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, proofBindingsDirName), mustDigestIdentity(oneDigest))
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, proofBindingsDirName), mustDigestIdentity(twoDigest))
}

func TestPersistZKProofArtifactAndVerificationRecord(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	recordDigest := mustRecordDigestForTest(t, ledger)
	binding := validAuditProofBindingPayloadFixture(recordDigest, sealResult.SealEnvelopeDigest)
	bindingDigest, _, err := ledger.PersistAuditProofBinding(binding)
	if err != nil {
		t.Fatalf("PersistAuditProofBinding returned error: %v", err)
	}

	artifact := validZKProofArtifactPayloadFixture(bindingDigest)
	artifactDigest, err := ledger.PersistZKProofArtifact(artifact)
	if err != nil {
		t.Fatalf("PersistZKProofArtifact returned error: %v", err)
	}
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, proofArtifactsDirName), mustDigestIdentity(artifactDigest))

	record := validZKProofVerificationRecordPayloadFixture(artifactDigest, artifact)
	recordDigestOut, err := ledger.PersistZKProofVerificationRecord(record)
	if err != nil {
		t.Fatalf("PersistZKProofVerificationRecord returned error: %v", err)
	}
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, proofVerificationsDirName), mustDigestIdentity(recordDigestOut))
}

func TestPersistAuditProofBindingFailsClosedOnMalformedExistingSidecar(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	recordDigest := mustRecordDigestForTest(t, ledger)
	payload := validAuditProofBindingPayloadFixture(recordDigest, sealResult.SealEnvelopeDigest)

	badPath := filepath.Join(root, sidecarDirName, proofBindingsDirName, strings.Repeat("f", 64)+".json")
	if err := os.WriteFile(badPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile malformed sidecar returned error: %v", err)
	}

	_, _, err := ledger.PersistAuditProofBinding(payload)
	if err == nil {
		t.Fatal("PersistAuditProofBinding expected fail-closed error for malformed existing sidecar")
	}
}

func mustRecordDigestForTest(t *testing.T, ledger *Ledger) trustpolicy.Digest {
	t.Helper()
	segment, err := ledger.loadSegment("segment-000001")
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	if len(segment.Frames) == 0 {
		t.Fatal("segment has no frames")
	}
	return segment.Frames[0].RecordDigest
}

func validAuditProofBindingPayloadFixture(recordDigest, sealDigest trustpolicy.Digest) trustpolicy.AuditProofBindingPayload {
	return trustpolicy.AuditProofBindingPayload{
		SchemaID:               trustpolicy.AuditProofBindingSchemaID,
		SchemaVersion:          trustpolicy.AuditProofBindingSchemaVersion,
		StatementFamily:        "audit.isolate_session_bound.attested_runtime_membership.v0",
		StatementVersion:       "v0",
		NormalizationProfileID: "runecode.zk.normalize.audit.isolate_session_bound.attested_runtime.v0",
		SchemeAdapterID:        "runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v0",
		AuditRecordDigest:      recordDigest,
		AuditSegmentSealDigest: sealDigest,
		MerkleRoot:             trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		ProtocolBundleManifest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)},
		BindingCommitment:      "sha256:" + strings.Repeat("c", 64),
		ProjectedPublicBindings: trustpolicy.AuditProofBindingProjectedPublicBindings{
			RuntimeImageDescriptorDigest:   "sha256:" + strings.Repeat("1", 64),
			AttestationEvidenceDigest:      "sha256:" + strings.Repeat("2", 64),
			AppliedHardeningPostureDigest:  "sha256:" + strings.Repeat("3", 64),
			SessionBindingDigest:           "sha256:" + strings.Repeat("4", 64),
			ProjectSubstrateSnapshotDigest: "sha256:" + strings.Repeat("5", 64),
		},
		MerklePathVersion: "runecode.zkproof.merkle_authentication_path.ordered_sha256_dse_v1",
		MerkleAuthenticationPath: []trustpolicy.AuditProofBindingMerkleAuthenticationStep{
			{SiblingDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}, SiblingPosition: "right"},
		},
		MerklePathDepth: 1,
		LeafIndex:       0,
		SourceRefs: []trustpolicy.ZKProofSourceRef{
			{SourceFamily: "audit_segment_seal", SourceDigest: sealDigest, SourceRole: "seal"},
			{SourceFamily: "audit_event", SourceDigest: recordDigest, SourceRole: "event"},
		},
	}
}

func validZKProofArtifactPayloadFixture(bindingDigest trustpolicy.Digest) trustpolicy.ZKProofArtifactPayload {
	return trustpolicy.ZKProofArtifactPayload{
		SchemaID:               trustpolicy.ZKProofArtifactSchemaID,
		SchemaVersion:          trustpolicy.ZKProofArtifactSchemaVersion,
		StatementFamily:        "audit.isolate_session_bound.attested_runtime_membership.v0",
		StatementVersion:       "v0",
		SchemeID:               "groth16",
		CurveID:                "bn254",
		CircuitID:              "runecode.zk.circuit.audit.isolate_session_bound.attested_runtime_membership.v0",
		ConstraintSystemDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("7", 64)},
		VerifierKeyDigest:      trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("8", 64)},
		SetupProvenanceDigest:  trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("9", 64)},
		NormalizationProfileID: "runecode.zk.normalize.audit.isolate_session_bound.attested_runtime.v0",
		SchemeAdapterID:        "runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v0",
		PublicInputs: map[string]any{
			"statement_family":   "audit.isolate_session_bound.attested_runtime_membership.v0",
			"binding_commitment": "sha256:" + strings.Repeat("a", 64),
		},
		PublicInputsDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)},
		ProofBytes:         "dGVzdF9wcm9vZl9ieXRlcw==",
		SourceRefs: []trustpolicy.ZKProofSourceRef{
			{SourceFamily: "audit_proof_binding", SourceDigest: bindingDigest, SourceRole: "binding"},
		},
	}
}

func validZKProofVerificationRecordPayloadFixture(proofDigest trustpolicy.Digest, artifact trustpolicy.ZKProofArtifactPayload) trustpolicy.ZKProofVerificationRecordPayload {
	return trustpolicy.ZKProofVerificationRecordPayload{
		SchemaID:                 trustpolicy.ZKProofVerificationRecordSchemaID,
		SchemaVersion:            trustpolicy.ZKProofVerificationRecordSchemaVersion,
		ProofDigest:              proofDigest,
		StatementFamily:          artifact.StatementFamily,
		StatementVersion:         artifact.StatementVersion,
		SchemeID:                 artifact.SchemeID,
		CurveID:                  artifact.CurveID,
		CircuitID:                artifact.CircuitID,
		ConstraintSystemDigest:   artifact.ConstraintSystemDigest,
		VerifierKeyDigest:        artifact.VerifierKeyDigest,
		SetupProvenanceDigest:    artifact.SetupProvenanceDigest,
		NormalizationProfileID:   artifact.NormalizationProfileID,
		SchemeAdapterID:          artifact.SchemeAdapterID,
		PublicInputsDigest:       artifact.PublicInputsDigest,
		VerifierImplementationID: "runecode.trusted.zk.verifier.gnark.v0",
		VerifiedAt:               time.Now().UTC().Format(time.RFC3339),
		VerificationOutcome:      trustpolicy.ProofVerificationOutcomeVerified,
		ReasonCodes:              []string{trustpolicy.ProofVerificationReasonVerified},
		CacheProvenance:          "fresh",
	}
}
