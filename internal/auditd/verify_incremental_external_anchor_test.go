package auditd

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestVerifyCurrentSegmentIncrementalWithPreverifiedSealPersistsReport(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.VerifyCurrentSegmentAndPersist(); err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist baseline returned error: %v", err)
	}
	anchorSigner := newAuditFixtureKey(t)
	receiptEnvelope := fixtureAnchorReceiptEnvelopeForSubject(t, anchorSigner, sealResult.SealEnvelopeDigest)
	receiptDigest, err := ledger.PersistReceiptEnvelope(receiptEnvelope)
	if err != nil {
		t.Fatalf("PersistReceiptEnvelope returned error: %v", err)
	}
	assertDigestSidecarExists(t, root+"/sidecar/receipts", mustDigestIdentity(receiptDigest))

	reportDigest, err := ledger.VerifyCurrentSegmentIncrementalWithPreverifiedSeal(sealResult.SealEnvelopeDigest, fixtureAuditAnchorVerifierRecordForKey(anchorSigner, anchorSigner.keyIDValue))
	if err != nil {
		t.Fatalf("VerifyCurrentSegmentIncrementalWithPreverifiedSeal returned error: %v", err)
	}
	assertDigestSidecarExists(t, root+"/sidecar/verification-reports", mustDigestIdentity(reportDigest))
	report, err := ledger.LatestVerificationReport()
	if err != nil {
		t.Fatalf("LatestVerificationReport returned error: %v", err)
	}
	if report.VerificationScope.ScopeKind != trustpolicy.AuditVerificationScopeSegment {
		t.Fatalf("verification_scope.scope_kind=%q, want segment", report.VerificationScope.ScopeKind)
	}
	if report.VerificationScope.LastSegmentID == "" {
		t.Fatal("verification_scope.last_segment_id empty")
	}
}

func TestVerifyCurrentSegmentIncrementalWithPreverifiedSealRejectsMismatchedSeal(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	_ = mustSealFixtureSegment(t, ledger, fixture)
	anchorSigner := newAuditFixtureKey(t)
	wrong := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)}
	if _, err := ledger.VerifyCurrentSegmentIncrementalWithPreverifiedSeal(wrong, fixtureAuditAnchorVerifierRecordForKey(anchorSigner, anchorSigner.keyIDValue)); err == nil {
		t.Fatal("VerifyCurrentSegmentIncrementalWithPreverifiedSeal expected seal mismatch error")
	}
}

func fixtureAuditAnchorVerifierRecordForKey(f auditFixtureKey, keyIDValue string) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(f.publicKey)},
		LogicalPurpose:         "audit_anchor",
		LogicalScope:           "node",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "secretsd", InstanceID: "secretsd-1"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Status:                 "active",
	}
}

func fixtureAnchorReceiptEnvelopeForSubject(t *testing.T, f auditFixtureKey, subjectDigest trustpolicy.Digest) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	return signEnvelopeWithFixtureKey(t, f.privateKey, f.keyIDValue, trustpolicy.AuditReceiptSchemaID, trustpolicy.AuditReceiptSchemaVersion, map[string]any{
		"schema_id":                 trustpolicy.AuditReceiptSchemaID,
		"schema_version":            trustpolicy.AuditReceiptSchemaVersion,
		"subject_digest":            subjectDigest,
		"audit_receipt_kind":        "anchor",
		"subject_family":            trustpolicy.AuditSegmentAnchoringSubjectSeal,
		"recorder":                  map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"recorded_at":               "2026-03-13T12:25:00Z",
		"receipt_payload_schema_id": "runecode.protocol.audit.receipt.anchor.v0",
		"receipt_payload": map[string]any{
			"anchor_kind":            "local_user_presence_signature",
			"key_protection_posture": "os_keystore",
			"presence_mode":          "os_confirmation",
			"anchor_witness": map[string]any{
				"witness_kind":   "local_user_presence_signature_v0",
				"witness_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
			},
		},
	})
}

func signEnvelopeWithFixtureKey(t *testing.T, privateKey ed25519.PrivateKey, keyID, payloadSchemaID, payloadSchemaVersion string, payload map[string]any) trustpolicy.SignedObjectEnvelope {
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
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: payloadSchemaID, PayloadSchemaVersion: payloadSchemaVersion, Payload: payloadBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyID, Signature: base64.StdEncoding.EncodeToString(signature)}}
}
