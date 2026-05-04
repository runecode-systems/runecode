package auditd

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func anchorFailureReasonCode(report trustpolicy.AuditVerificationReportPayload) string {
	if !anchorStatusRequiresReason(report.AnchoringStatus) {
		return ""
	}
	for _, code := range []string{
		firstAnchoringFindingCode(report.Findings),
		firstNonBlankCode(report.HardFailures),
		firstNonBlankCode(report.DegradedReasons),
		firstFindingCode(report.Findings),
	} {
		if code != "" {
			return code
		}
	}
	return ""
}

func anchorFailureReasonMessage(report trustpolicy.AuditVerificationReportPayload, reasonCode string) string {
	if !anchorStatusRequiresReason(report.AnchoringStatus) {
		return ""
	}
	if msg := matchingFindingMessage(report.Findings, reasonCode); msg != "" {
		return msg
	}
	return strings.TrimSpace(report.Summary)
}

func anchorStatusRequiresReason(status string) bool {
	status = strings.TrimSpace(status)
	return status != "" && status != trustpolicy.AuditVerificationStatusOK
}

func firstAnchoringFindingCode(findings []trustpolicy.AuditVerificationFinding) string {
	for idx := range findings {
		if strings.TrimSpace(findings[idx].Dimension) != trustpolicy.AuditVerificationDimensionAnchoring {
			continue
		}
		if code := strings.TrimSpace(findings[idx].Code); code != "" {
			return code
		}
	}
	return ""
}

func firstFindingCode(findings []trustpolicy.AuditVerificationFinding) string {
	for idx := range findings {
		if code := strings.TrimSpace(findings[idx].Code); code != "" {
			return code
		}
	}
	return ""
}

func firstNonBlankCode(codes []string) string {
	for idx := range codes {
		if code := strings.TrimSpace(codes[idx]); code != "" {
			return code
		}
	}
	return ""
}

func matchingFindingMessage(findings []trustpolicy.AuditVerificationFinding, reasonCode string) string {
	reasonCode = strings.TrimSpace(reasonCode)
	if reasonCode == "" {
		return ""
	}
	for idx := range findings {
		if strings.TrimSpace(findings[idx].Code) != reasonCode {
			continue
		}
		if msg := strings.TrimSpace(findings[idx].Message); msg != "" {
			return msg
		}
	}
	return ""
}
