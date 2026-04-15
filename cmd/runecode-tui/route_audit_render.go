package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func renderAuditSummary(verify *brokerapi.AuditVerificationGetResponse) string {
	if verify == nil {
		return "Verification posture unavailable"
	}
	s := verify.Summary
	anchorLabel := renderAnchoringPostureLabel(s.AnchoringStatus)
	return fmt.Sprintf("Verification posture: integrity=%s %s anchoring=%s (%s) storage=%s lifecycle=%s degraded=%t %s findings=%d", s.IntegrityStatus, postureBadge(s.IntegrityStatus), s.AnchoringStatus, anchorLabel, s.StoragePostureStatus, s.SegmentLifecycleStatus, s.CurrentlyDegraded, boolBadge("degraded", s.CurrentlyDegraded), s.FindingCount)
}

func renderAuditFinalizeSummary(finalize *brokerapi.AuditFinalizeVerifyResponse) string {
	if finalize == nil {
		return tableHeader("Finalize/verify") + " status=unavailable"
	}
	status := strings.TrimSpace(finalize.ActionStatus)
	if status == "" {
		status = "unknown"
	}
	report := "n/a"
	if finalize.ReportDigest != nil {
		if id, err := finalize.ReportDigest.Identity(); err == nil {
			report = id
		}
	}
	line := fmt.Sprintf("%s status=%s segment=%s report=%s", tableHeader("Finalize/verify"), status, valueOrNA(strings.TrimSpace(finalize.SegmentID)), report)
	if status == "ok" {
		return line
	}
	reason := strings.TrimSpace(finalize.FailureCode)
	if reason == "" {
		reason = strings.TrimSpace(finalize.FailureMessage)
	}
	if reason != "" {
		line += " reason=" + reason
	}
	return line
}

func renderAuditPageSummary(cursor, next string, backDepth, count int) string {
	page := "page=1"
	if cursor != "" {
		page = fmt.Sprintf("page_cursor=%s", cursor)
	}
	nextLabel := "no"
	if next != "" {
		nextLabel = "yes"
	}
	return fmt.Sprintf("Timeline paging: %s entries=%d has_next=%s back_stack=%d", page, count, nextLabel, backDepth)
}

func renderAuditFindings(verify *brokerapi.AuditVerificationGetResponse, presentation contentPresentationMode) string {
	presentation = normalizePresentationMode(presentation)
	if verify == nil {
		return "Verification findings: unavailable"
	}
	if len(verify.Report.Findings) == 0 {
		return "Verification findings: none"
	}
	if presentation == presentationStructured {
		return fmt.Sprintf("Verification findings (structured): total=%d degraded_reasons=%d hard_failures=%d", len(verify.Report.Findings), len(verify.Report.DegradedReasons), len(verify.Report.HardFailures))
	}
	line := "Verification findings (machine-readable):"
	for i, finding := range verify.Report.Findings {
		if presentation == presentationRendered && i >= 4 {
			line += fmt.Sprintf("\n  ... (%d more)", len(verify.Report.Findings)-i)
			break
		}
		if i >= 4 {
			line += fmt.Sprintf("\n  ... (%d more in raw mode)", len(verify.Report.Findings)-i)
			break
		}
		line += fmt.Sprintf("\n  - code=%s severity=%s dimension=%s", finding.Code, finding.Severity, finding.Dimension)
	}
	if len(verify.Report.DegradedReasons) > 0 {
		line += fmt.Sprintf("\n  degraded_reason_codes=%s", joinCSV(verify.Report.DegradedReasons))
	}
	if len(verify.Report.HardFailures) > 0 {
		line += fmt.Sprintf("\n  hard_failure_codes=%s", joinCSV(verify.Report.HardFailures))
	}
	return line
}

func renderAuditTimeline(timeline []brokerapi.AuditTimelineViewEntry, selected int) string {
	if len(timeline) == 0 {
		return "  - no audit entries"
	}
	line := ""
	for i, entry := range timeline {
		marker := " "
		if i == selected {
			marker = ">"
		}
		digest, _ := entry.RecordDigest.Identity()
		posture := "n/a"
		reasonCodes := ""
		if entry.VerificationPosture != nil {
			posture = entry.VerificationPosture.Status
			reasonCodes = joinCSV(entry.VerificationPosture.ReasonCodes)
		}
		line += selectedLine(i == selected, fmt.Sprintf("  %s %s event=%s posture=%s reasons=%s summary=%s", marker, digest, entry.EventType, posture, valueOrNA(reasonCodes), entry.Summary)) + "\n"
	}
	return line
}

func renderAuditInspector(record *brokerapi.AuditRecordGetResponse, presentation contentPresentationMode) string {
	if record == nil {
		return "  Select a timeline record and press enter to load detail."
	}
	presentation = normalizePresentationMode(presentation)
	r := record.Record
	status := "unknown"
	reasons := 0
	if r.VerificationPosture != nil {
		status = r.VerificationPosture.Status
		reasons = len(r.VerificationPosture.ReasonCodes)
	}
	if presentation == presentationStructured {
		return compactLines(
			fmt.Sprintf("  Structured record: family=%s event=%s", r.RecordFamily, r.EventType),
			fmt.Sprintf("  Link counts: references=%d reasons=%d", len(r.LinkedReferences), reasons),
		)
	}
	if presentation == presentationRaw {
		return compactLines(
			"  Raw record (secrets redacted):",
			fmt.Sprintf("  Raw record family=%s event=%s occurred_at=%s", r.RecordFamily, r.EventType, r.OccurredAt),
			fmt.Sprintf("  Raw posture status=%s reasons=%d", status, reasons),
		)
	}
	return compactLines(
		fmt.Sprintf("  Record family: %s event=%s", r.RecordFamily, r.EventType),
		fmt.Sprintf("  Occurred at: %s", r.OccurredAt),
		fmt.Sprintf("  Verification posture: %s (%s) reasons=%d", status, renderAnchoringPostureLabel(status), reasons),
		fmt.Sprintf("  Linked references: %d", len(r.LinkedReferences)),
	)
}

func renderAnchoringPostureLabel(status string) string {
	switch status {
	case "ok":
		return "anchored"
	case "degraded":
		return "unanchored/degraded"
	case "failed":
		return "invalid/failed anchoring"
	default:
		return "unknown anchoring posture"
	}
}

func renderAuditSafetyAlertStrip(verify *brokerapi.AuditVerificationGetResponse) string {
	if verify == nil {
		return tableHeader("Audit safety strip") + " " + dangerBadge("AUDIT_POSTURE_UNAVAILABLE") + " audit verification unavailable"
	}
	s := verify.Summary
	parts := []string{tableHeader("Audit safety strip")}
	if strings.EqualFold(strings.TrimSpace(s.AnchoringStatus), "degraded") || s.CurrentlyDegraded {
		parts = append(parts, auditDegradedBadge("UNANCHORED_OR_DEGRADED_AUDIT"))
	}
	if strings.EqualFold(strings.TrimSpace(s.AnchoringStatus), "failed") || strings.EqualFold(strings.TrimSpace(s.IntegrityStatus), "failed") || strings.EqualFold(strings.TrimSpace(s.IntegrityStatus), "invalid") {
		parts = append(parts, dangerBadge("INVALID_OR_FAILED_ANCHORING"))
	}
	if len(parts) == 1 {
		parts = append(parts, successBadge("AUDIT_ANCHORED_AND_VALID"))
	}
	return strings.Join(parts, " ")
}
