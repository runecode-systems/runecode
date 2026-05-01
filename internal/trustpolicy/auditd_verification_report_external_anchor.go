package trustpolicy

import "fmt"

func validateExternalAnchorFindingConsistency(report AuditVerificationReportPayload) error {
	hasInvalidRequired := hasExternalAnchorFindingForRequiredTarget(report.Findings, AuditVerificationReasonExternalAnchorInvalid)
	hasDeferredRequired := hasExternalAnchorFindingForRequiredTarget(report.Findings, AuditVerificationReasonExternalAnchorDeferredOrUnavailable)
	hasValid := hasFindingWithCode(report.Findings, AuditVerificationReasonExternalAnchorValid)
	if hasInvalidRequired && report.AnchoringStatus != AuditVerificationStatusFailed {
		return fmt.Errorf("external_anchor_invalid finding requires anchoring_status=failed")
	}
	if hasDeferredRequired && report.AnchoringStatus == AuditVerificationStatusOK {
		return fmt.Errorf("external_anchor_deferred_or_unavailable finding cannot coexist with anchoring_status=ok")
	}
	if hasValid && report.AnchoringStatus == AuditVerificationStatusFailed && !hasInvalidRequired {
		return fmt.Errorf("external_anchor_valid finding cannot be sole external posture when anchoring_status=failed")
	}
	return nil
}

func hasExternalAnchorFindingForRequiredTarget(findings []AuditVerificationFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code != code || !externalAnchorFindingAppliesToRequiredTarget(finding) {
			continue
		}
		return true
	}
	return false
}

func externalAnchorFindingAppliesToRequiredTarget(finding AuditVerificationFinding) bool {
	if finding.Severity == AuditVerificationSeverityError {
		return true
	}
	return externalAnchorFindingRequirement(finding) == ExternalAnchorTargetRequirementRequired
}

func externalAnchorFindingRequirement(finding AuditVerificationFinding) string {
	if finding.Details == nil {
		return ExternalAnchorTargetRequirementRequired
	}
	raw, ok := finding.Details["target_requirement"].(string)
	if !ok {
		return ExternalAnchorTargetRequirementRequired
	}
	return NormalizeExternalAnchorTargetRequirement(raw)
}
