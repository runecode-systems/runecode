package trustpolicy

import (
	"testing"
	"time"
)

func TestValidateAuditVerificationReportRejectsUnknownReasonCode(t *testing.T) {
	report := validVerificationReportForReasonCodeValidation()
	report.DegradedReasons = []string{"unknown_reason_code"}
	if err := ValidateAuditVerificationReportPayload(report); err == nil {
		t.Fatal("ValidateAuditVerificationReportPayload expected error for unknown reason code")
	}
}

func TestValidateAuditVerificationReportRejectsBlankReasonCode(t *testing.T) {
	report := validVerificationReportForReasonCodeValidation()
	report.DegradedReasons = []string{"  "}
	if err := ValidateAuditVerificationReportPayload(report); err == nil {
		t.Fatal("ValidateAuditVerificationReportPayload expected error for blank reason code")
	}
}

func TestValidateAuditVerificationReportRejectsDuplicateReasonCode(t *testing.T) {
	report := validVerificationReportForReasonCodeValidation()
	report.DegradedReasons = []string{AuditVerificationReasonEvidenceExportIncomplete, AuditVerificationReasonEvidenceExportIncomplete}
	if err := ValidateAuditVerificationReportPayload(report); err == nil {
		t.Fatal("ValidateAuditVerificationReportPayload expected error for duplicate reason code")
	}
}

func TestValidateAuditVerificationReportAcceptsExpandedReasonCodes(t *testing.T) {
	report := validVerificationReportForReasonCodeValidation()
	report.DegradedReasons = []string{
		AuditVerificationReasonExternalAnchorDeferredOrUnavailable,
		AuditVerificationReasonMissingRequiredApprovalEvidence,
		AuditVerificationReasonMissingRuntimeAttestationEvidence,
		AuditVerificationReasonNegativeCapabilitySummaryMissing,
		AuditVerificationReasonVerifierIdentityMissingOrUnknown,
		AuditVerificationReasonEvidenceExportIncomplete,
	}
	report.HardFailures = []string{AuditVerificationReasonExternalAnchorInvalid}
	report.CurrentlyDegraded = true
	report.AnchoringStatus = AuditVerificationStatusFailed
	report.Findings = []AuditVerificationFinding{
		{Code: AuditVerificationReasonExternalAnchorValid, Dimension: AuditVerificationDimensionAnchoring, Severity: AuditVerificationSeverityInfo, Message: "validated external anchor"},
		{Code: AuditVerificationReasonExternalAnchorInvalid, Dimension: AuditVerificationDimensionAnchoring, Severity: AuditVerificationSeverityError, Message: "invalid external anchor"},
	}
	report.CryptographicallyValid = false
	if err := ValidateAuditVerificationReportPayload(report); err != nil {
		t.Fatalf("ValidateAuditVerificationReportPayload returned error: %v", err)
	}
}

func validVerificationReportForReasonCodeValidation() AuditVerificationReportPayload {
	return AuditVerificationReportPayload{
		SchemaID:               AuditVerificationReportSchemaID,
		SchemaVersion:          AuditVerificationReportSchemaVersion,
		VerifiedAt:             time.Date(2026, time.March, 13, 13, 0, 0, 0, time.UTC).Format(time.RFC3339),
		VerificationScope:      AuditVerificationScope{ScopeKind: AuditVerificationScopeSegment, LastSegmentID: "segment-0001"},
		CryptographicallyValid: true,
		HistoricallyAdmissible: true,
		CurrentlyDegraded:      false,
		IntegrityStatus:        AuditVerificationStatusOK,
		AnchoringStatus:        AuditVerificationStatusOK,
		AnchoringPosture:       AuditVerificationAnchoringPostureLocalAnchorReceiptOnly,
		StoragePostureStatus:   AuditVerificationStatusOK,
		SegmentLifecycleStatus: AuditVerificationStatusOK,
		VerifierIdentity:       KeyIDProfile + ":" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		TrustRootIdentities:    []string{"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		Findings:               []AuditVerificationFinding{},
		DegradedReasons:        []string{},
		HardFailures:           []string{},
	}
}
