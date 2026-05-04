package auditd

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestAnchorCurrentSegmentPersistsVerifierRecordForRestartVerification(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	request := mustAnchorRequestForRestartVerification(t, sealResult.SealEnvelopeDigest)

	if _, err := ledger.AnchorCurrentSegment(request); err != nil {
		t.Fatalf("AnchorCurrentSegment returned error: %v", err)
	}

	records := []trustpolicy.VerifierRecord{}
	if err := readJSONFile(filepath.Join(root, "contracts", "verifier-records.json"), &records); err != nil {
		t.Fatalf("readJSONFile(verifier-records) returned error: %v", err)
	}
	if !containsVerifierRecordWithKeyID(records, request.SignerKeyIDValue) {
		t.Fatalf("anchor verifier key_id_value %q was not persisted", request.SignerKeyIDValue)
	}
	assertRestartVerificationStaysAnchored(t, root)
}

func mustAnchorRequestForRestartVerification(t *testing.T, sealDigest trustpolicy.Digest) AnchorSegmentRequest {
	t.Helper()
	anchorPublic, anchorPrivate, signerKeyID := mustGenerateAnchorSigner(t)
	request := AnchorSegmentRequest{
		SealDigest:           sealDigest,
		AnchorKind:           "local_user_presence_signature",
		KeyProtectionPosture: "os_keystore",
		PresenceMode:         "os_confirmation",
		AnchorWitnessKind:    "local_user_presence_signature_v0",
		AnchorWitnessDigest:  trustpolicy.Digest{HashAlg: "sha256", Hash: "6fb33062f2e64a2f95b6627f64c70ed0f8c05ef4984dd67d24196fef5a2588f3"},
		Recorder: trustpolicy.PrincipalIdentity{
			SchemaID:      "runecode.protocol.v0.PrincipalIdentity",
			SchemaVersion: "0.2.0",
			ActorKind:     "daemon",
			PrincipalID:   "secretsd",
			InstanceID:    "secretsd-1",
		},
		SignerPublicKeyBase64: base64.StdEncoding.EncodeToString(anchorPublic),
		SignerKeyIDValue:      signerKeyID,
		SignerLogicalScope:    "node",
		SignerInstanceID:      "secretsd-1",
		RecordedAtRFC3339:     "2026-03-13T12:40:00Z",
	}
	request.Signature = mustSignAnchorRequest(t, request, anchorPrivate, signerKeyID)
	return request
}

func mustGenerateAnchorSigner(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	anchorPublic, anchorPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	anchorKeyID := sha256.Sum256(anchorPublic)
	return anchorPublic, anchorPrivate, hex.EncodeToString(anchorKeyID[:])
}

func mustSignAnchorRequest(t *testing.T, request AnchorSegmentRequest, privateKey ed25519.PrivateKey, signerKeyID string) trustpolicy.SignatureBlock {
	t.Helper()
	payloadJSON, err := marshalAnchorReceiptPayload(request)
	if err != nil {
		t.Fatalf("marshalAnchorReceiptPayload returned error: %v", err)
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(payloadJSON)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}
	signature := ed25519.Sign(privateKey, canonicalPayload)
	return trustpolicy.SignatureBlock{
		Alg:        "ed25519",
		KeyID:      trustpolicy.KeyIDProfile,
		KeyIDValue: signerKeyID,
		Signature:  base64.StdEncoding.EncodeToString(signature),
	}
}

func assertRestartVerificationStaysAnchored(t *testing.T, root string) {
	t.Helper()
	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(restart) returned error: %v", err)
	}
	verification, err := reopened.VerifyCurrentSegmentAndPersist()
	if err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist(restart) returned error: %v", err)
	}
	if verification.Report.AnchoringStatus != trustpolicy.AuditVerificationStatusOK {
		t.Fatalf("anchoring_status = %q, want %q", verification.Report.AnchoringStatus, trustpolicy.AuditVerificationStatusOK)
	}
	if containsReasonCodeForAuditdTest(verification.Report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, did not want %q", verification.Report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func containsVerifierRecordWithKeyID(records []trustpolicy.VerifierRecord, keyID string) bool {
	for index := range records {
		if records[index].KeyIDValue == keyID {
			return true
		}
	}
	return false
}

func containsReasonCodeForAuditdTest(codes []string, code string) bool {
	for index := range codes {
		if codes[index] == code {
			return true
		}
	}
	return false
}

func TestAnchorFailureReasonCodePrefersAnchoringFinding(t *testing.T) {
	report := trustpolicy.AuditVerificationReportPayload{
		AnchoringStatus: trustpolicy.AuditVerificationStatusFailed,
		Findings: []trustpolicy.AuditVerificationFinding{
			{Code: trustpolicy.AuditVerificationReasonExternalAnchorInvalid, Dimension: trustpolicy.AuditVerificationDimensionAnchoring},
			{Code: trustpolicy.AuditVerificationReasonReceiptInvalid, Dimension: trustpolicy.AuditVerificationDimensionIntegrity},
		},
		HardFailures: []string{trustpolicy.AuditVerificationReasonAnchorReceiptInvalid},
		DegradedReasons: []string{
			trustpolicy.AuditVerificationReasonExternalAnchorDeferredOrUnavailable,
		},
	}
	if got := anchorFailureReasonCode(report); got != trustpolicy.AuditVerificationReasonExternalAnchorInvalid {
		t.Fatalf("anchorFailureReasonCode() = %q, want %q", got, trustpolicy.AuditVerificationReasonExternalAnchorInvalid)
	}
}

func TestAnchorFailureReasonCodeFallsBackToHardFailure(t *testing.T) {
	report := trustpolicy.AuditVerificationReportPayload{
		AnchoringStatus: trustpolicy.AuditVerificationStatusFailed,
		HardFailures:    []string{trustpolicy.AuditVerificationReasonAnchorReceiptInvalid},
	}
	if got := anchorFailureReasonCode(report); got != trustpolicy.AuditVerificationReasonAnchorReceiptInvalid {
		t.Fatalf("anchorFailureReasonCode() = %q, want %q", got, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestAnchorFailureReasonCodeFallsBackToDegradedReason(t *testing.T) {
	report := trustpolicy.AuditVerificationReportPayload{
		AnchoringStatus: trustpolicy.AuditVerificationStatusDegraded,
		DegradedReasons: []string{trustpolicy.AuditVerificationReasonExternalAnchorDeferredOrUnavailable},
	}
	if got := anchorFailureReasonCode(report); got != trustpolicy.AuditVerificationReasonExternalAnchorDeferredOrUnavailable {
		t.Fatalf("anchorFailureReasonCode() = %q, want %q", got, trustpolicy.AuditVerificationReasonExternalAnchorDeferredOrUnavailable)
	}
}

func TestAnchorFailureReasonMessagePrefersMatchingFinding(t *testing.T) {
	report := trustpolicy.AuditVerificationReportPayload{
		AnchoringStatus: trustpolicy.AuditVerificationStatusFailed,
		Findings: []trustpolicy.AuditVerificationFinding{
			{Code: trustpolicy.AuditVerificationReasonAnchorReceiptInvalid, Message: "anchor receipt signature does not verify"},
		},
		Summary: "fallback summary",
	}
	if got := anchorFailureReasonMessage(report, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid); got != "anchor receipt signature does not verify" {
		t.Fatalf("anchorFailureReasonMessage() = %q, want %q", got, "anchor receipt signature does not verify")
	}
}
