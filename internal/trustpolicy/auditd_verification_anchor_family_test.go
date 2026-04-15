package trustpolicy

import (
	"encoding/json"
	"testing"
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

func TestVerifyAuditEvidenceRejectsNonMVPAnchorKind(t *testing.T) {
	fixture := newAuditVerificationFixture(t, verifierStatusFixture{status: "active"})
	receipt := fixture.anchorReceiptEnvelope(t, fixture.sealEnvelopeDigest)

	var payload map[string]any
	if err := json.Unmarshal(receipt.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal receipt payload returned error: %v", err)
	}
	receiptPayload := payload["receipt_payload"].(map[string]any)
	receiptPayload["anchor_kind"] = "external_transparency_log_v0"
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

func TestVerifyAuditEvidenceRejectsNonMVPAnchorWitnessKind(t *testing.T) {
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
