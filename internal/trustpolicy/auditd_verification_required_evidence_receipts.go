package trustpolicy

import "strings"

type receiptEvidenceSummary struct {
	hasRuntimeSummary            bool
	hasNegativeCapabilitySummary bool
	hasNegativeCapabilityLimited bool
	hasEvidenceBundleExport      bool
	hasMetaAuditEvidence         bool
	hasBoundaryAuthorization     bool
	hasApprovalEvidence          bool
	hasApprovalConsumption       bool
	hasPublicationEvidence       bool
	hasOverrideEvidence          bool
}

func (s *receiptEvidenceSummary) observeReceipt(receipt auditReceiptPayloadStrict) {
	s.observeReceiptKind(receipt.AuditReceiptKind)
	s.observeAnchorApprovalEvidence(receipt)
	s.observeNegativeCapabilitySupport(receipt)
}

func (s *receiptEvidenceSummary) observeReceiptKind(kind string) {
	switch kind {
	case auditReceiptKindRuntimeSummary:
		s.hasRuntimeSummary = true
	case auditReceiptKindNegativeCapabilitySummary:
		s.hasNegativeCapabilitySummary = true
	case auditReceiptKindEvidenceBundleExport:
		s.hasEvidenceBundleExport = true
		s.hasMetaAuditEvidence = true
	case auditReceiptKindEvidenceImport,
		auditReceiptKindEvidenceRestore,
		auditReceiptKindRetentionPolicyChanged,
		auditReceiptKindArchivalOperation,
		auditReceiptKindVerifierConfigurationChanged,
		auditReceiptKindTrustRootUpdated,
		auditReceiptKindSensitiveEvidenceView:
		s.hasMetaAuditEvidence = true
	case auditReceiptKindProviderInvocationAuthorized, auditReceiptKindProviderInvocationDenied:
		s.hasBoundaryAuthorization = true
	case auditReceiptKindApprovalResolution:
		s.hasApprovalEvidence = true
	case auditReceiptKindApprovalConsumption:
		s.hasApprovalEvidence = true
		s.hasApprovalConsumption = true
	case auditReceiptKindArtifactPublished:
		s.hasPublicationEvidence = true
	case auditReceiptKindOverrideOrBreakGlass:
		s.hasOverrideEvidence = true
	}
}

func (s *receiptEvidenceSummary) observeAnchorApprovalEvidence(receipt auditReceiptPayloadStrict) {
	if receipt.AuditReceiptKind != "anchor" {
		return
	}
	payload := anchorReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err == nil && payload.ApprovalDecision != nil {
		s.hasApprovalEvidence = true
	}
}

func (s *receiptEvidenceSummary) observeNegativeCapabilitySupport(receipt auditReceiptPayloadStrict) {
	if receipt.AuditReceiptKind != auditReceiptKindNegativeCapabilitySummary {
		return
	}
	payload := negativeCapabilitySummaryReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return
	}
	for _, support := range []string{
		payload.SecretLeaseEvidenceSupport,
		payload.NetworkEgressEvidenceSupport,
		payload.ApprovalConsumptionEvidenceSupport,
		payload.BoundaryCrossingEvidenceSupport,
	} {
		if strings.TrimSpace(support) != "explicit" {
			s.hasNegativeCapabilityLimited = true
			return
		}
	}
}

func reportReceiptEvidenceSummary(report *AuditVerificationReportPayload, segmentID string, summary receiptEvidenceSummary) {
	applyMissingNegativeCapabilitySummary(report, segmentID, summary)
	applyEvidenceExportCompleteness(report, segmentID, summary)
	applyRequiredApprovalEvidence(report, segmentID, summary)
	applyRequiredOverrideEvidence(report, segmentID, summary)
	applyNegativeCapabilitySupportPosture(report, segmentID, summary)
	applyAnchoringPostureFromSummary(report)
}

func applyMissingNegativeCapabilitySummary(report *AuditVerificationReportPayload, segmentID string, summary receiptEvidenceSummary) {
	if summary.hasRuntimeSummary && !summary.hasNegativeCapabilitySummary {
		addDegraded(report, AuditVerificationReasonNegativeCapabilitySummaryMissing, AuditVerificationDimensionIntegrity, "runtime summary evidence is present but negative capability summary evidence is missing", segmentID, nil)
	}
}

func applyEvidenceExportCompleteness(report *AuditVerificationReportPayload, segmentID string, summary receiptEvidenceSummary) {
	if summary.hasMetaAuditEvidence && !summary.hasEvidenceBundleExport {
		addDegraded(report, AuditVerificationReasonEvidenceExportIncomplete, AuditVerificationDimensionIntegrity, "evidence export receipt is missing from verification evidence set", segmentID, nil)
	}
}

func applyRequiredApprovalEvidence(report *AuditVerificationReportPayload, segmentID string, summary receiptEvidenceSummary) {
	if summary.hasBoundaryAuthorization && !summary.hasApprovalEvidence {
		addHardFailure(report, AuditVerificationReasonMissingRequiredApprovalEvidence, AuditVerificationDimensionIntegrity, "boundary authorization evidence exists without approval evidence linkage", segmentID, nil)
	}
	if summary.hasPublicationEvidence && !summary.hasApprovalConsumption && !summary.hasOverrideEvidence {
		addHardFailure(report, AuditVerificationReasonMissingRequiredApprovalEvidence, AuditVerificationDimensionIntegrity, "publication evidence exists without approval consumption linkage", segmentID, nil)
	}
}

func applyRequiredOverrideEvidence(report *AuditVerificationReportPayload, segmentID string, summary receiptEvidenceSummary) {
	if summary.hasOverrideEvidence && !summary.hasApprovalEvidence && !summary.hasPublicationEvidence {
		addDegraded(report, AuditVerificationReasonMissingRequiredApprovalEvidence, AuditVerificationDimensionIntegrity, "override evidence exists without linked approval resolution evidence", segmentID, nil)
	}
}

func applyNegativeCapabilitySupportPosture(report *AuditVerificationReportPayload, segmentID string, summary receiptEvidenceSummary) {
	if summary.hasNegativeCapabilitySummary && summary.hasNegativeCapabilityLimited {
		addDegraded(report, AuditVerificationReasonNegativeCapabilitySupportLimitedOrUnknown, AuditVerificationDimensionIntegrity, "negative-capability summary includes limited evidence support", segmentID, nil)
	}
}

func applyAnchoringPostureFromSummary(report *AuditVerificationReportPayload) {
	if report.AnchoringStatus == AuditVerificationStatusFailed || reasonCodePresent(report.HardFailures, AuditVerificationReasonExternalAnchorInvalid) {
		report.AnchoringPosture = AuditVerificationAnchoringPostureExternalAnchorInvalid
		return
	}
	if report.AnchoringStatus != AuditVerificationStatusDegraded {
		if hasFindingWithCode(report.Findings, AuditVerificationReasonExternalAnchorValid) {
			report.AnchoringPosture = AuditVerificationAnchoringPostureExternalAnchorValidated
		}
		return
	}
	if reasonCodePresent(report.DegradedReasons, AuditVerificationReasonExternalAnchorDeferredOrUnavailable) {
		report.AnchoringPosture = AuditVerificationAnchoringPostureExternalAnchorDeferredOrUnknown
		return
	}
	if reasonCodePresent(report.DegradedReasons, AuditVerificationReasonAnchorReceiptMissing) {
		report.AnchoringPosture = AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound
	}
}

func reasonCodePresent(codes []string, want string) bool {
	for i := range codes {
		if strings.TrimSpace(codes[i]) == want {
			return true
		}
	}
	return false
}
