package trustpolicy

import (
	"strings"
	"testing"
	"time"
)

func TestValidateAuditProofBindingPayload(t *testing.T) {
	payload := validAuditProofBindingPayloadForTest()
	if err := ValidateAuditProofBindingPayload(payload); err != nil {
		t.Fatalf("ValidateAuditProofBindingPayload returned error: %v", err)
	}

	payload.MerklePathDepth = 2
	err := ValidateAuditProofBindingPayload(payload)
	if err == nil || !strings.Contains(err.Error(), "merkle_authentication_path length") {
		t.Fatalf("ValidateAuditProofBindingPayload error=%v, want path-depth mismatch", err)
	}
}

func TestValidateZKProofArtifactPayload(t *testing.T) {
	payload := validZKProofArtifactPayloadForTest()
	if err := ValidateZKProofArtifactPayload(payload); err != nil {
		t.Fatalf("ValidateZKProofArtifactPayload returned error: %v", err)
	}

	payload.SourceRefs = nil
	err := ValidateZKProofArtifactPayload(payload)
	if err == nil || !strings.Contains(err.Error(), "source_refs is required") {
		t.Fatalf("ValidateZKProofArtifactPayload error=%v, want missing source refs", err)
	}
}

func TestValidateZKProofVerificationRecordPayloadAndSetupIdentity(t *testing.T) {
	artifact := validZKProofArtifactPayloadForTest()
	record := validZKProofVerificationRecordPayloadForTest(artifact)
	if err := ValidateZKProofVerificationRecordPayload(record); err != nil {
		t.Fatalf("ValidateZKProofVerificationRecordPayload returned error: %v", err)
	}

	matched, code, err := EvaluateZKProofSetupIdentityMatch(artifact, ZKProofTrustedVerifierPosture{
		ConstraintSystemDigest: artifact.ConstraintSystemDigest,
		VerifierKeyDigest:      artifact.VerifierKeyDigest,
		SetupProvenanceDigest:  artifact.SetupProvenanceDigest,
	})
	if err != nil {
		t.Fatalf("EvaluateZKProofSetupIdentityMatch returned error: %v", err)
	}
	if !matched || code != ProofVerificationReasonVerified {
		t.Fatalf("setup identity match=(%v,%q), want (true,%q)", matched, code, ProofVerificationReasonVerified)
	}

	mismatched, code, err := EvaluateZKProofSetupIdentityMatch(artifact, ZKProofTrustedVerifierPosture{
		ConstraintSystemDigest: Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)},
		VerifierKeyDigest:      artifact.VerifierKeyDigest,
		SetupProvenanceDigest:  artifact.SetupProvenanceDigest,
	})
	if err != nil {
		t.Fatalf("EvaluateZKProofSetupIdentityMatch(mismatch) returned error: %v", err)
	}
	if mismatched || code != ProofVerificationReasonSetupIdentityMismatch {
		t.Fatalf("setup identity mismatch=(%v,%q), want (false,%q)", mismatched, code, ProofVerificationReasonSetupIdentityMismatch)
	}
}

func validAuditProofBindingPayloadForTest() AuditProofBindingPayload {
	return AuditProofBindingPayload{
		SchemaID:               AuditProofBindingSchemaID,
		SchemaVersion:          AuditProofBindingSchemaVersion,
		StatementFamily:        "audit.isolate_session_bound.attested_runtime_membership.v0",
		StatementVersion:       "v0",
		NormalizationProfileID: "runecode.zk.normalize.audit.isolate_session_bound.attested_runtime.v0",
		SchemeAdapterID:        "runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v0",
		AuditRecordDigest:      Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)},
		AuditSegmentSealDigest: Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)},
		MerkleRoot:             Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)},
		ProtocolBundleManifest: Digest{HashAlg: "sha256", Hash: strings.Repeat("4", 64)},
		BindingCommitment:      "sha256:" + strings.Repeat("5", 64),
		ProjectedPublicBindings: AuditProofBindingProjectedPublicBindings{
			RuntimeImageDescriptorDigest:   "sha256:" + strings.Repeat("6", 64),
			AttestationEvidenceDigest:      "sha256:" + strings.Repeat("7", 64),
			AppliedHardeningPostureDigest:  "sha256:" + strings.Repeat("8", 64),
			SessionBindingDigest:           "sha256:" + strings.Repeat("9", 64),
			ProjectSubstrateSnapshotDigest: "sha256:" + strings.Repeat("a", 64),
		},
		MerklePathVersion: "runecode.zkproof.merkle_authentication_path.ordered_sha256_dse_v1",
		MerkleAuthenticationPath: []AuditProofBindingMerkleAuthenticationStep{
			{SiblingDigest: Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)}, SiblingPosition: "left"},
		},
		MerklePathDepth: 1,
		LeafIndex:       0,
		SourceRefs: []ZKProofSourceRef{
			{SourceFamily: "audit_event", SourceDigest: Digest{HashAlg: "sha256", Hash: strings.Repeat("c", 64)}, SourceRole: "event"},
		},
	}
}

func validZKProofArtifactPayloadForTest() ZKProofArtifactPayload {
	return ZKProofArtifactPayload{
		SchemaID:               ZKProofArtifactSchemaID,
		SchemaVersion:          ZKProofArtifactSchemaVersion,
		StatementFamily:        "audit.isolate_session_bound.attested_runtime_membership.v0",
		StatementVersion:       "v0",
		SchemeID:               "groth16",
		CurveID:                "bn254",
		CircuitID:              "runecode.zk.circuit.audit.isolate_session_bound.attested_runtime_membership.v0",
		ConstraintSystemDigest: Digest{HashAlg: "sha256", Hash: strings.Repeat("d", 64)},
		VerifierKeyDigest:      Digest{HashAlg: "sha256", Hash: strings.Repeat("e", 64)},
		SetupProvenanceDigest:  Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)},
		NormalizationProfileID: "runecode.zk.normalize.audit.isolate_session_bound.attested_runtime.v0",
		SchemeAdapterID:        "runecode.zk.adapter.gnark.groth16.isolate_session_bound_attested_runtime.v0",
		PublicInputs: map[string]any{
			"statement_family": "audit.isolate_session_bound.attested_runtime_membership.v0",
		},
		PublicInputsDigest: Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		ProofBytes:         "dGVzdA==",
		SourceRefs: []ZKProofSourceRef{
			{SourceFamily: "audit_proof_binding", SourceDigest: Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)}, SourceRole: "binding"},
		},
	}
}

func validZKProofVerificationRecordPayloadForTest(artifact ZKProofArtifactPayload) ZKProofVerificationRecordPayload {
	return ZKProofVerificationRecordPayload{
		SchemaID:                 ZKProofVerificationRecordSchemaID,
		SchemaVersion:            ZKProofVerificationRecordSchemaVersion,
		ProofDigest:              Digest{HashAlg: "sha256", Hash: strings.Repeat("c", 64)},
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
		VerificationOutcome:      ProofVerificationOutcomeVerified,
		ReasonCodes:              []string{ProofVerificationReasonVerified},
		CacheProvenance:          "fresh",
	}
}
