package trustpolicy

import (
	"fmt"
	"sort"
	"strings"
)

type DerivedRunAuditVerificationSummary struct {
	CryptographicallyValid bool     `json:"cryptographically_valid"`
	HistoricallyAdmissible bool     `json:"historically_admissible"`
	CurrentlyDegraded      bool     `json:"currently_degraded"`
	IntegrityStatus        string   `json:"integrity_status"`
	AnchoringStatus        string   `json:"anchoring_status"`
	StoragePostureStatus   string   `json:"storage_posture_status"`
	SegmentLifecycleStatus string   `json:"segment_lifecycle_status"`
	DegradedReasons        []string `json:"degraded_reasons"`
	HardFailures           []string `json:"hard_failures"`
	FindingCount           int      `json:"finding_count"`
	ErrorFindingCount      int      `json:"error_finding_count"`
	WarningFindingCount    int      `json:"warning_finding_count"`
	InfoFindingCount       int      `json:"info_finding_count"`
}

func BuildDerivedRunAuditVerificationSummary(report AuditVerificationReportPayload) (DerivedRunAuditVerificationSummary, error) {
	if err := ValidateAuditVerificationReportPayload(report); err != nil {
		return DerivedRunAuditVerificationSummary{}, err
	}
	summary := DerivedRunAuditVerificationSummary{
		CryptographicallyValid: report.CryptographicallyValid,
		HistoricallyAdmissible: report.HistoricallyAdmissible,
		CurrentlyDegraded:      report.CurrentlyDegraded,
		IntegrityStatus:        report.IntegrityStatus,
		AnchoringStatus:        report.AnchoringStatus,
		StoragePostureStatus:   report.StoragePostureStatus,
		SegmentLifecycleStatus: report.SegmentLifecycleStatus,
		DegradedReasons:        append([]string{}, report.DegradedReasons...),
		HardFailures:           append([]string{}, report.HardFailures...),
		FindingCount:           len(report.Findings),
	}
	for _, finding := range report.Findings {
		switch finding.Severity {
		case AuditVerificationSeverityError:
			summary.ErrorFindingCount++
		case AuditVerificationSeverityWarning:
			summary.WarningFindingCount++
		case AuditVerificationSeverityInfo:
			summary.InfoFindingCount++
		}
	}
	sort.Strings(summary.DegradedReasons)
	sort.Strings(summary.HardFailures)
	return summary, nil
}

func addHardFailure(report *AuditVerificationReportPayload, code string, dimension string, message string, segmentID string, subject *Digest) {
	addFinding(report, AuditVerificationFinding{
		Code:      code,
		Dimension: dimension,
		Severity:  AuditVerificationSeverityError,
		Message:   message,
		SegmentID: segmentID,
	}, subject)
	report.HardFailures = append(report.HardFailures, code)
	markStatus(report, dimension, AuditVerificationStatusFailed)
}

func addDegraded(report *AuditVerificationReportPayload, code string, dimension string, message string, segmentID string, subject *Digest) {
	addFinding(report, AuditVerificationFinding{
		Code:      code,
		Dimension: dimension,
		Severity:  AuditVerificationSeverityWarning,
		Message:   message,
		SegmentID: segmentID,
	}, subject)
	report.DegradedReasons = append(report.DegradedReasons, code)
	markStatus(report, dimension, AuditVerificationStatusDegraded)
}

func addFinding(report *AuditVerificationReportPayload, finding AuditVerificationFinding, subject *Digest) {
	if subject != nil {
		copyDigest := *subject
		finding.SubjectRecordDigest = &copyDigest
	}
	report.Findings = append(report.Findings, finding)
}

func markStatus(report *AuditVerificationReportPayload, dimension string, status string) {
	set := func(current *string, target string) {
		if *current == AuditVerificationStatusFailed {
			return
		}
		if target == AuditVerificationStatusFailed {
			*current = target
			return
		}
		if *current == AuditVerificationStatusOK {
			*current = target
		}
	}
	switch dimension {
	case AuditVerificationDimensionIntegrity:
		set(&report.IntegrityStatus, status)
	case AuditVerificationDimensionAnchoring:
		set(&report.AnchoringStatus, status)
	case AuditVerificationDimensionStoragePosture:
		set(&report.StoragePostureStatus, status)
	case AuditVerificationDimensionSegmentLifecycle:
		set(&report.SegmentLifecycleStatus, status)
	}
}

func finalizeAuditVerificationReport(report AuditVerificationReportPayload) AuditVerificationReportPayload {
	report.Findings = dedupeAndSortFindings(report.Findings)
	report.DegradedReasons = dedupeAndSortReasonCodes(report.DegradedReasons)
	report.HardFailures = dedupeAndSortReasonCodes(report.HardFailures)
	report.CurrentlyDegraded = len(report.DegradedReasons) > 0
	report.CryptographicallyValid = deriveCryptographicValidity(report.HardFailures)
	if report.IntegrityStatus == "" {
		report.IntegrityStatus = AuditVerificationStatusOK
	}
	if report.AnchoringStatus == "" {
		report.AnchoringStatus = AuditVerificationStatusOK
	}
	if report.StoragePostureStatus == "" {
		report.StoragePostureStatus = AuditVerificationStatusOK
	}
	if report.SegmentLifecycleStatus == "" {
		report.SegmentLifecycleStatus = AuditVerificationStatusOK
	}
	if report.Summary == "" {
		report.Summary = buildVerificationSummary(report)
	}
	return report
}

func dedupeAndSortReasonCodes(codes []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	sort.Strings(result)
	return result
}

func dedupeAndSortFindings(findings []AuditVerificationFinding) []AuditVerificationFinding {
	if len(findings) == 0 {
		return findings
	}
	keyed := map[string]AuditVerificationFinding{}
	for _, finding := range findings {
		subjectID := ""
		if finding.SubjectRecordDigest != nil {
			subjectID, _ = finding.SubjectRecordDigest.Identity()
		}
		key := strings.Join([]string{finding.Code, finding.Dimension, finding.Severity, finding.SegmentID, subjectID, finding.Message}, "|")
		if _, exists := keyed[key]; !exists {
			keyed[key] = finding
		}
	}
	keys := make([]string, 0, len(keyed))
	for key := range keyed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]AuditVerificationFinding, 0, len(keys))
	for _, key := range keys {
		result = append(result, keyed[key])
	}
	return result
}

func hasFindingWithCode(findings []AuditVerificationFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func deriveCryptographicValidity(hardFailures []string) bool {
	for _, code := range hardFailures {
		if isCryptographicFailureCode(code) {
			return false
		}
	}
	return true
}

func isCryptographicFailureCode(code string) bool {
	switch code {
	case AuditVerificationReasonDetachedSignatureInvalid,
		AuditVerificationReasonSegmentFrameDigestMismatch,
		AuditVerificationReasonSegmentFileHashMismatch,
		AuditVerificationReasonSegmentSealInvalid:
		return true
	default:
		return false
	}
}

func buildVerificationSummary(report AuditVerificationReportPayload) string {
	if len(report.HardFailures) > 0 {
		return fmt.Sprintf("Audit verification failed with %d hard failure(s) and %d degraded reason(s).", len(report.HardFailures), len(report.DegradedReasons))
	}
	if len(report.DegradedReasons) > 0 {
		return fmt.Sprintf("Audit verification passed with degraded posture (%d reason(s)).", len(report.DegradedReasons))
	}
	return "Audit verification passed with all dimensions in ok posture."
}

func mustDigestIdentity(digest Digest) string {
	identity, _ := digest.Identity()
	return identity
}
