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
	allowedFields := map[string]struct{}{
		"schema_id":                 {},
		"schema_version":            {},
		"subject_digest":            {},
		"audit_receipt_kind":        {},
		"subject_family":            {},
		"recorder":                  {},
		"recorded_at":               {},
		"receipt_payload_schema_id": {},
		"receipt_payload":           {},
	}
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return auditReceiptPayloadStrict{}, fmt.Errorf("decode audit receipt payload object: %w", err)
	}
	maxAuditReceiptProperties := len(allowedFields)
	if len(obj) > maxAuditReceiptProperties {
		return auditReceiptPayloadStrict{}, fmt.Errorf("audit receipt has too many properties: got %d, max %d", len(obj), maxAuditReceiptProperties)
	}
	for key := range obj {
		if _, ok := allowedFields[key]; !ok {
			return auditReceiptPayloadStrict{}, fmt.Errorf("unsupported audit receipt field %q", key)
		}
	}
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
	validator := auditReceiptPayloadValidator(receipt.AuditReceiptKind)
	if validator == nil {
		return fmt.Errorf("unsupported audit_receipt_kind %q for payload validation", receipt.AuditReceiptKind)
	}
	return validator(receipt)
}

func auditReceiptPayloadValidator(kind string) func(auditReceiptPayloadStrict) error {
	switch kind {
	case "anchor":
		return validateAnchorReceiptPayload
	case "import", "restore", "reconciliation":
		return validateImportRestoreReceiptPayload
	case auditReceiptKindProviderInvocationAuthorized, auditReceiptKindProviderInvocationDenied:
		return validateProviderInvocationReceiptPayload
	case auditReceiptKindApprovalResolution, auditReceiptKindApprovalConsumption:
		return validateApprovalEvidenceReceiptPayload
	case auditReceiptKindArtifactPublished:
		return validatePublicationEvidenceReceiptPayload
	case auditReceiptKindOverrideOrBreakGlass:
		return validateOverrideEvidenceReceiptPayload
	case auditReceiptKindSecretLeaseIssued, auditReceiptKindSecretLeaseRevoked:
		return validateSecretLeaseReceiptPayload
	case auditReceiptKindRuntimeSummary:
		return validateRuntimeSummaryReceiptPayload
	case auditReceiptKindDegradedPostureSummary:
		return validateDegradedPostureSummaryReceiptPayload
	case auditReceiptKindNegativeCapabilitySummary:
		return validateNegativeCapabilitySummaryReceiptPayload
	case auditReceiptKindEvidenceBundleExport,
		auditReceiptKindEvidenceImport,
		auditReceiptKindEvidenceRestore,
		auditReceiptKindRetentionPolicyChanged,
		auditReceiptKindArchivalOperation,
		auditReceiptKindVerifierConfigurationChanged,
		auditReceiptKindTrustRootUpdated,
		auditReceiptKindSensitiveEvidenceView:
		return validateMetaAuditActionReceiptPayload
	default:
		return nil
	}
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
	if err := validateAuditReceiptKind(receipt.AuditReceiptKind); err != nil {
		return err
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

func validateAuditReceiptKind(kind string) error {
	if !auditVerificationCodePattern.MatchString(kind) {
		return fmt.Errorf("invalid audit_receipt_kind %q", kind)
	}
	if _, ok := supportedAuditReceiptKinds()[kind]; !ok {
		return fmt.Errorf("unsupported audit_receipt_kind %q", kind)
	}
	return nil
}

func supportedAuditReceiptKinds() map[string]struct{} {
	return map[string]struct{}{
		"anchor":         {},
		"import":         {},
		"restore":        {},
		"reconciliation": {},
		auditReceiptKindProviderInvocationAuthorized: {},
		auditReceiptKindProviderInvocationDenied:     {},
		auditReceiptKindApprovalResolution:           {},
		auditReceiptKindApprovalConsumption:          {},
		auditReceiptKindArtifactPublished:            {},
		auditReceiptKindOverrideOrBreakGlass:         {},
		auditReceiptKindSecretLeaseIssued:            {},
		auditReceiptKindSecretLeaseRevoked:           {},
		auditReceiptKindRuntimeSummary:               {},
		auditReceiptKindDegradedPostureSummary:       {},
		auditReceiptKindNegativeCapabilitySummary:    {},
		auditReceiptKindEvidenceBundleExport:         {},
		auditReceiptKindEvidenceImport:               {},
		auditReceiptKindEvidenceRestore:              {},
		auditReceiptKindRetentionPolicyChanged:       {},
		auditReceiptKindArchivalOperation:            {},
		auditReceiptKindVerifierConfigurationChanged: {},
		auditReceiptKindTrustRootUpdated:             {},
		auditReceiptKindSensitiveEvidenceView:        {},
	}
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
