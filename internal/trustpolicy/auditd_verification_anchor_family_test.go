package trustpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestVerifyAuditEvidenceFailsClosedOnAnchorReceiptWithUnexpectedSubjectFamily(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	payload["subject_family"] = "unexpected_family"
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceFailsClosedOnAnchorReceiptWithoutTypedWitness(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	delete(receiptPayload, "anchor_witness")
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceAcceptsExternalTransparencyLogAnchorKind(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	configureExternalAnchorPayloadFixture(t, receiptPayload, "external_transparency_log_v0")
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.AnchoringStatus != AuditVerificationStatusOK {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusOK)
	}
	if containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want no %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceAcceptsExternalTimestampAuthorityAnchorKind(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	configureExternalAnchorPayloadFixture(t, receiptPayload, "external_timestamp_authority_v0")
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.AnchoringStatus != AuditVerificationStatusOK {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusOK)
	}
	if containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want no %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceAcceptsExternalPublicChainAnchorKind(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	configureExternalAnchorPayloadFixture(t, receiptPayload, "external_public_chain_v0")
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.AnchoringStatus != AuditVerificationStatusOK {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusOK)
	}
	if containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want no %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceRejectsLocalAnchorWithUnexpectedWitnessKind(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	anchorWitness := receiptPayload["anchor_witness"].(map[string]any)
	anchorWitness["witness_kind"] = "external_transparency_log_entry_v0"
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

func TestVerifyAuditEvidenceFailsClosedOnExternalAnchorRuntimeAdapterMismatch(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	configureExternalAnchorPayloadFixture(t, receiptPayload, "external_transparency_log_v0")
	externalAnchor := receiptPayload["external_anchor"].(map[string]any)
	externalAnchor["runtime_adapter"] = "timestamp_authority_v0"
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

func TestVerifyAuditEvidenceFailsClosedOnExternalAnchorTargetDescriptorDigestMismatch(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	configureExternalAnchorPayloadFixture(t, receiptPayload, "external_transparency_log_v0")
	externalAnchor := receiptPayload["external_anchor"].(map[string]any)
	externalAnchor["target_descriptor_digest"] = map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}
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

func configureExternalAnchorPayloadFixture(t *testing.T, receiptPayload map[string]any, anchorKind string) {
	t.Helper()
	delete(receiptPayload, "key_protection_posture")
	delete(receiptPayload, "presence_mode")
	delete(receiptPayload, "anchor_witness")
	receiptPayload["anchor_kind"] = anchorKind

	switch anchorKind {
	case "external_transparency_log_v0":
		descriptor := map[string]any{
			"descriptor_schema_id":   "runecode.protocol.audit.anchor_target.transparency_log.v0",
			"log_id":                 "rekor-public-good",
			"log_public_key_digest":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)},
			"entry_encoding_profile": "rekor_v1",
		}
		receiptPayload["external_anchor"] = buildExternalAnchorFixture(t, "transparency_log", descriptor, "transparency_log_receipt_v0", "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0")
	case "external_timestamp_authority_v0":
		descriptor := map[string]any{
			"descriptor_schema_id":     "runecode.protocol.audit.anchor_target.timestamp_authority.v0",
			"authority_id":             "tsa-1",
			"certificate_chain_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)},
			"timestamp_profile":        "rfc3161",
		}
		receiptPayload["external_anchor"] = buildExternalAnchorFixture(t, "timestamp_authority", descriptor, "timestamp_token_v0", "runecode.protocol.audit.anchor_proof.timestamp_token.v0")
	case "external_public_chain_v0":
		descriptor := map[string]any{
			"descriptor_schema_id":       "runecode.protocol.audit.anchor_target.public_chain.v0",
			"chain_namespace":            "evm",
			"network_id":                 "sepolia",
			"settlement_contract_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)},
		}
		receiptPayload["external_anchor"] = buildExternalAnchorFixture(t, "public_chain", descriptor, "public_chain_tx_receipt_v0", "runecode.protocol.audit.anchor_proof.public_chain_tx_receipt.v0")
	default:
		t.Fatalf("unsupported anchor_kind %q", anchorKind)
	}
}

func buildExternalAnchorFixture(t *testing.T, targetKind string, descriptor map[string]any, proofKind string, proofSchema string) map[string]any {
	t.Helper()
	digest := digestForJSONFixture(t, descriptor)
	derivedExecution := map[string]any{"submit_endpoint_uri": "https://anchor.example/submit"}
	switch targetKind {
	case "timestamp_authority":
		derivedExecution = map[string]any{"tsa_endpoint_uri": "https://tsa.example/timestamp"}
	case "public_chain":
		derivedExecution = map[string]any{"rpc_endpoint_uri": "https://rpc.example"}
	}
	return map[string]any{
		"target_kind":              targetKind,
		"runtime_adapter":          "transparency_log_v0",
		"target_descriptor":        descriptor,
		"target_descriptor_digest": map[string]any{"hash_alg": digest.HashAlg, "hash": digest.Hash},
		"proof": map[string]any{
			"proof_kind":      proofKind,
			"proof_schema_id": proofSchema,
			"proof_digest":    map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)},
		},
		"derived_execution": derivedExecution,
	}
}

func digestForJSONFixture(t *testing.T, value any) Digest {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}
	sum := sha256.Sum256(canonical)
	return Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}
}

func TestVerifyAuditEvidenceFailsClosedOnAnchorReceiptSignerPurposeMismatch(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	fixture.verifierRecords[0].LogicalPurpose = "host_audit"
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceFailsClosedOnAnchorReceiptSignerScopeMismatch(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	fixture.verifierRecords[0].LogicalScope = "user"
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceFailsClosedOnAnchorReceiptSignerPostureMismatch(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	receiptPayload["presence_mode"] = "hardware_touch"
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceMarksPassphrasePresenceAsDegraded(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	fixture.verifierRecords[0].PresenceMode = "passphrase"
	fixture.verifierRecords[0].KeyProtectionPosture = "passphrase_wrapped"
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	receiptPayload["presence_mode"] = "passphrase"
	receiptPayload["key_protection_posture"] = "passphrase_wrapped"
	receipt.Payload = marshalJSONFixture(t, payload)
	receipt = resignEnvelopeFixture(t, fixture.privateKey, receipt)

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.AnchoringStatus != AuditVerificationStatusDegraded {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusDegraded)
	}
	if !containsReasonCode(report.DegradedReasons, AuditVerificationReasonAnchorPassphrasePresenceDegraded) {
		t.Fatalf("degraded_reasons = %v, want %q", report.DegradedReasons, AuditVerificationReasonAnchorPassphrasePresenceDegraded)
	}
}

func TestVerifyAuditEvidenceFailsClosedOnAnchorEnvelopeSignatureInvalid(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)
	receipt.Signature.Signature = "invalid-base64"

	report := mustVerifyAuditEvidenceReport(t, fixture, []SignedObjectEnvelope{receipt})
	if report.AnchoringStatus != AuditVerificationStatusFailed {
		t.Fatalf("anchoring_status = %q, want %q", report.AnchoringStatus, AuditVerificationStatusFailed)
	}
	if !containsReasonCode(report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, want %q", report.HardFailures, AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func TestVerifyAuditEvidenceFailsClosedOnAnchorPayloadWithUnknownField(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	receiptPayload["unexpected_field"] = "unexpected"
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

func TestVerifyAuditEvidenceFailsClosedOnAnchorReceiptApprovalLinkAssuranceNone(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	receiptPayload["approval_assurance_level"] = "none"
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

func TestVerifyAuditEvidenceFailsClosedOnAnchorReceiptHardwareAssuranceWithoutHardwarePosture(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	fixture.verifierRecords[0].PresenceMode = "hardware_touch"
	fixture.verifierRecords[0].KeyProtectionPosture = "hardware_backed"
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	receiptPayload["approval_assurance_level"] = "hardware_backed"
	receiptPayload["presence_mode"] = "hardware_touch"
	receiptPayload["key_protection_posture"] = "os_keystore"
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
