package trustpolicy

import (
	"fmt"
	"strings"
	"time"
)

func ValidateAuditVerificationReportPayload(report AuditVerificationReportPayload) error {
	if err := validateAuditVerificationReportHeader(report); err != nil {
		return err
	}
	if err := validateAuditVerificationReportStatuses(report); err != nil {
		return err
	}
	if err := validateReasonCodeList(report.DegradedReasons, "degraded_reasons"); err != nil {
		return err
	}
	if err := validateReasonCodeList(report.HardFailures, "hard_failures"); err != nil {
		return err
	}
	if err := validateAuditVerificationFindings(report.Findings); err != nil {
		return err
	}
	return validateAuditVerificationReportCrossFieldConstraints(report)
}

func validateAuditVerificationReportHeader(report AuditVerificationReportPayload) error {
	if report.SchemaID != AuditVerificationReportSchemaID {
		return fmt.Errorf("unexpected report schema_id %q", report.SchemaID)
	}
	if report.SchemaVersion != AuditVerificationReportSchemaVersion {
		return fmt.Errorf("unexpected report schema_version %q", report.SchemaVersion)
	}
	if report.VerifiedAt == "" {
		return fmt.Errorf("verified_at is required")
	}
	if _, err := time.Parse(time.RFC3339, report.VerifiedAt); err != nil {
		return fmt.Errorf("invalid verified_at: %w", err)
	}
	if err := validateAuditVerificationScope(report.VerificationScope); err != nil {
		return fmt.Errorf("verification_scope: %w", err)
	}
	return nil
}

func validateAuditVerificationReportStatuses(report AuditVerificationReportPayload) error {
	if err := validateAuditVerificationStatus(report.IntegrityStatus); err != nil {
		return fmt.Errorf("integrity_status: %w", err)
	}
	if err := validateAuditVerificationStatus(report.AnchoringStatus); err != nil {
		return fmt.Errorf("anchoring_status: %w", err)
	}
	if err := validateAuditVerificationStatus(report.StoragePostureStatus); err != nil {
		return fmt.Errorf("storage_posture_status: %w", err)
	}
	if err := validateAuditVerificationStatus(report.SegmentLifecycleStatus); err != nil {
		return fmt.Errorf("segment_lifecycle_status: %w", err)
	}
	return nil
}

func validateReasonCodeList(reasons []string, field string) error {
	seen := map[string]struct{}{}
	for index := range reasons {
		reason := reasons[index]
		if _, ok := auditVerificationAllowedCodes[reason]; !ok {
			return fmt.Errorf("%s[%d] has unknown reason code %q", field, index, reason)
		}
		if _, dup := seen[reason]; dup {
			return fmt.Errorf("%s[%d] duplicates reason code %q", field, index, reason)
		}
		seen[reason] = struct{}{}
	}
	return nil
}

func validateAuditVerificationFindings(findings []AuditVerificationFinding) error {
	for index := range findings {
		if err := validateAuditVerificationFinding(findings[index]); err != nil {
			return fmt.Errorf("findings[%d]: %w", index, err)
		}
	}
	return nil
}

func validateAuditVerificationReportCrossFieldConstraints(report AuditVerificationReportPayload) error {
	if hasFailedStatus(report) && len(report.HardFailures) == 0 {
		return fmt.Errorf("hard_failures must be non-empty when any status is failed")
	}
	if report.CurrentlyDegraded && len(report.DegradedReasons) == 0 {
		return fmt.Errorf("currently_degraded=true requires degraded_reasons")
	}
	if !report.CurrentlyDegraded && len(report.DegradedReasons) > 0 {
		return fmt.Errorf("currently_degraded=false cannot include degraded_reasons")
	}
	if report.CryptographicallyValid && hasCryptographicHardFailure(report.HardFailures) {
		return fmt.Errorf("cryptographically_valid=true cannot include cryptographic hard_failures")
	}
	return nil
}

func hasCryptographicHardFailure(hardFailures []string) bool {
	for _, code := range hardFailures {
		if isCryptographicFailureCode(code) {
			return true
		}
	}
	return false
}

func hasFailedStatus(report AuditVerificationReportPayload) bool {
	return report.IntegrityStatus == AuditVerificationStatusFailed ||
		report.AnchoringStatus == AuditVerificationStatusFailed ||
		report.StoragePostureStatus == AuditVerificationStatusFailed ||
		report.SegmentLifecycleStatus == AuditVerificationStatusFailed
}

func validateAuditVerificationScope(scope AuditVerificationScope) error {
	switch scope.ScopeKind {
	case AuditVerificationScopeInstance:
		return validateInstanceScope(scope)
	case AuditVerificationScopeSegment:
		return validateSegmentScope(scope)
	case AuditVerificationScopeSegmentRange:
		return validateSegmentRangeScope(scope)
	default:
		return fmt.Errorf("unsupported scope_kind %q", scope.ScopeKind)
	}
}

func validateInstanceScope(scope AuditVerificationScope) error {
	if scope.FirstSegmentID != "" || scope.LastSegmentID != "" {
		return fmt.Errorf("instance scope cannot include first_segment_id or last_segment_id")
	}
	return nil
}

func validateSegmentScope(scope AuditVerificationScope) error {
	if scope.LastSegmentID == "" {
		return fmt.Errorf("segment scope requires last_segment_id")
	}
	if scope.FirstSegmentID != "" {
		return fmt.Errorf("segment scope cannot include first_segment_id")
	}
	return nil
}

func validateSegmentRangeScope(scope AuditVerificationScope) error {
	if scope.FirstSegmentID == "" || scope.LastSegmentID == "" {
		return fmt.Errorf("segment_range scope requires first_segment_id and last_segment_id")
	}
	return nil
}

func validateAuditVerificationStatus(status string) error {
	switch status {
	case AuditVerificationStatusOK, AuditVerificationStatusDegraded, AuditVerificationStatusFailed:
		return nil
	default:
		return fmt.Errorf("unsupported status %q", status)
	}
}

func validateAuditVerificationFinding(finding AuditVerificationFinding) error {
	if err := validateAuditVerificationFindingCode(finding.Code); err != nil {
		return err
	}
	if err := validateAuditVerificationFindingDimension(finding.Dimension); err != nil {
		return err
	}
	if err := validateAuditVerificationFindingSeverity(finding.Severity); err != nil {
		return err
	}
	if strings.TrimSpace(finding.Message) == "" {
		return fmt.Errorf("finding message is required")
	}
	if err := validateAuditVerificationFindingSubjectDigest(finding.SubjectRecordDigest); err != nil {
		return err
	}
	return validateAuditVerificationRelatedDigests(finding.RelatedRecordDigests)
}

func validateAuditVerificationFindingSubjectDigest(digest *Digest) error {
	if digest == nil {
		return nil
	}
	if _, err := digest.Identity(); err != nil {
		return fmt.Errorf("subject_record_digest: %w", err)
	}
	return nil
}

func validateAuditVerificationRelatedDigests(digests []Digest) error {
	seenRelated := map[string]struct{}{}
	for index := range digests {
		identity, err := digests[index].Identity()
		if err != nil {
			return fmt.Errorf("related_record_digests[%d]: %w", index, err)
		}
		if _, dup := seenRelated[identity]; dup {
			return fmt.Errorf("related_record_digests[%d] duplicates digest %q", index, identity)
		}
		seenRelated[identity] = struct{}{}
	}
	return nil
}

func validateAuditVerificationFindingCode(code string) error {
	if _, ok := auditVerificationAllowedCodes[code]; !ok {
		return fmt.Errorf("unknown finding code %q", code)
	}
	if !auditVerificationCodePattern.MatchString(code) {
		return fmt.Errorf("finding code %q does not match required pattern", code)
	}
	return nil
}

func validateAuditVerificationFindingDimension(dimension string) error {
	switch dimension {
	case AuditVerificationDimensionIntegrity, AuditVerificationDimensionAnchoring, AuditVerificationDimensionStoragePosture, AuditVerificationDimensionSegmentLifecycle:
		return nil
	default:
		return fmt.Errorf("unsupported finding dimension %q", dimension)
	}
}

func validateAuditVerificationFindingSeverity(severity string) error {
	switch severity {
	case AuditVerificationSeverityInfo, AuditVerificationSeverityWarning, AuditVerificationSeverityError:
		return nil
	default:
		return fmt.Errorf("unsupported finding severity %q", severity)
	}
}
