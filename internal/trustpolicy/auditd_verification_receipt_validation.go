package trustpolicy

import (
	"encoding/json"
	"fmt"
	"time"
)

func decodeAndValidateVerifiedReceipt(index int, segmentID string, payload json.RawMessage, verifierRecord VerifierRecord, report *AuditVerificationReportPayload) (auditReceiptPayloadStrict, bool) {
	receipt, err := decodeAuditReceiptPayload(payload)
	if err != nil {
		addHardFailure(report, AuditVerificationReasonReceiptInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] decode failed: %v", index, err), segmentID, nil)
		return auditReceiptPayloadStrict{}, false
	}
	if err := validateAuditReceiptPayload(receipt); err != nil {
		addTypedReceiptInvalidFailure(index, segmentID, receipt, report, fmt.Sprintf("invalid: %v", err))
		return auditReceiptPayloadStrict{}, false
	}
	if err := validateReceiptSignerContract(receipt, verifierRecord); err != nil {
		addTypedReceiptInvalidFailure(index, segmentID, receipt, report, fmt.Sprintf("signer contract invalid: %v", err))
		return auditReceiptPayloadStrict{}, false
	}
	return receipt, true
}

func addTypedReceiptInvalidFailure(index int, segmentID string, receipt auditReceiptPayloadStrict, report *AuditVerificationReportPayload, detail string) {
	if receipt.AuditReceiptKind == "anchor" {
		addHardFailure(report, AuditVerificationReasonAnchorReceiptInvalid, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] %s", index, detail), segmentID, nil)
		return
	}
	addHardFailure(report, AuditVerificationReasonReceiptInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] %s", index, detail), segmentID, nil)
}

func envelopeDeclaresAnchorReceipt(envelope SignedObjectEnvelope) bool {
	if envelope.PayloadSchemaID != AuditReceiptSchemaID {
		return false
	}
	probe := struct {
		AuditReceiptKind string `json:"audit_receipt_kind"`
	}{}
	if err := json.Unmarshal(envelope.Payload, &probe); err != nil {
		return false
	}
	return probe.AuditReceiptKind == "anchor"
}

func addReceiptValidationFailure(report *AuditVerificationReportPayload, segmentID string, index int, anchorReceipt bool, message string) {
	if anchorReceipt {
		addHardFailure(report, AuditVerificationReasonAnchorReceiptInvalid, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] %s", index, message), segmentID, nil)
		return
	}
	addHardFailure(report, AuditVerificationReasonReceiptInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] %s", index, message), segmentID, nil)
}

func decodeAuditReceiptPayload(raw json.RawMessage) (auditReceiptPayloadStrict, error) {
	receipt := auditReceiptPayloadStrict{}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return auditReceiptPayloadStrict{}, fmt.Errorf("decode audit receipt payload: %w", err)
	}
	return receipt, nil
}

func validateAuditReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if err := validateAuditReceiptCoreFields(receipt); err != nil {
		return err
	}
	if err := validateAuditReceiptPayloadPresence(receipt); err != nil {
		return err
	}
	if receipt.AuditReceiptKind == "anchor" {
		return validateAnchorReceiptPayload(receipt)
	}
	if receipt.AuditReceiptKind != "import" && receipt.AuditReceiptKind != "restore" {
		return nil
	}
	return validateImportRestoreReceiptPayload(receipt)
}

func validateAuditReceiptCoreFields(receipt auditReceiptPayloadStrict) error {
	if receipt.SchemaID != AuditReceiptSchemaID {
		return fmt.Errorf("unexpected receipt schema_id %q", receipt.SchemaID)
	}
	if receipt.SchemaVersion != AuditReceiptSchemaVersion {
		return fmt.Errorf("unexpected receipt schema_version %q", receipt.SchemaVersion)
	}
	if _, err := receipt.SubjectDigest.Identity(); err != nil {
		return fmt.Errorf("subject_digest: %w", err)
	}
	if !auditVerificationCodePattern.MatchString(receipt.AuditReceiptKind) {
		return fmt.Errorf("invalid audit_receipt_kind %q", receipt.AuditReceiptKind)
	}
	if _, ok := map[string]struct{}{"anchor": {}, "import": {}, "restore": {}, "reconciliation": {}}[receipt.AuditReceiptKind]; !ok {
		return fmt.Errorf("unsupported audit_receipt_kind %q", receipt.AuditReceiptKind)
	}
	if err := validateReceiptRecorder(receipt.Recorder); err != nil {
		return err
	}
	if receipt.RecordedAt == "" {
		return fmt.Errorf("recorded_at is required")
	}
	if _, err := time.Parse(time.RFC3339, receipt.RecordedAt); err != nil {
		return fmt.Errorf("invalid recorded_at: %w", err)
	}
	return nil
}

func validateReceiptRecorder(recorder json.RawMessage) error {
	if len(recorder) == 0 {
		return fmt.Errorf("recorder is required")
	}
	identity := PrincipalIdentity{}
	if err := json.Unmarshal(recorder, &identity); err != nil {
		return fmt.Errorf("recorder must decode as principal identity: %w", err)
	}
	if identity.SchemaID != "runecode.protocol.v0.PrincipalIdentity" {
		return fmt.Errorf("recorder.schema_id must be runecode.protocol.v0.PrincipalIdentity")
	}
	if identity.SchemaVersion != "0.2.0" {
		return fmt.Errorf("recorder.schema_version must be 0.2.0")
	}
	if identity.ActorKind == "" || identity.PrincipalID == "" || identity.InstanceID == "" {
		return fmt.Errorf("recorder is required")
	}
	return nil
}
