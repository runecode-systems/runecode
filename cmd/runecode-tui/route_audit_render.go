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

func renderAuditDirectoryItems(timeline []brokerapi.AuditTimelineViewEntry) []string {
	items := make([]string, 0, len(timeline))
	for _, entry := range timeline {
		digest, _ := entry.RecordDigest.Identity()
		items = append(items, fmt.Sprintf("%s event=%s", valueOrNA(digest), valueOrNA(entry.EventType)))
	}
	return items
}

func renderAuditInspector(record *brokerapi.AuditRecordGetResponse, presentation contentPresentationMode) string {
	if record == nil {
		return "  Select a timeline record and press enter to load detail."
	}
	presentation = normalizePresentationMode(presentation)
	r := record.Record
	status, reasons, reasonCodes := auditInspectorPosture(r)
	content := auditInspectorContent(r, status, reasons, presentation)
	referenceItems := auditInspectorReferenceItems(r)
	contentKind := auditInspectorContentKind(presentation)
	return renderInspectorShell(inspectorShellSpec{
		Title:          "Audit inspector",
		Summary:        fmt.Sprintf("event=%s family=%s linked_refs=%d", valueOrNA(r.EventType), valueOrNA(r.RecordFamily), len(r.LinkedReferences)),
		Identity:       fmt.Sprintf("record_family=%s event_type=%s", valueOrNA(r.RecordFamily), valueOrNA(r.EventType)),
		Status:         fmt.Sprintf("verification=%s reasons=%d", valueOrNA(status), reasons),
		Badges:         []string{stateBadgeWithLabel("posture", status), appTheme.InspectorHint.Render("typed record details")},
		References:     []inspectorReference{{Label: "records", Items: referenceItems}, {Label: "reason codes", Items: reasonCodes}},
		LocalActions:   []string{"jump:runs", "jump:approvals", "jump:artifacts", "copy:record_digest"},
		CopyActions:    auditRouteCopyActions(record),
		ModeTabs:       []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:     string(presentation),
		ContentKind:    contentKind,
		ContentLabel:   "audit record",
		Content:        content,
		ViewportWidth:  96,
		ViewportHeight: 12,
	})
}

func auditInspectorPosture(record brokerapi.AuditRecordDetail) (string, int, []string) {
	status := "unknown"
	reasons := 0
	reasonCodes := []string{}
	if record.VerificationPosture != nil {
		status = record.VerificationPosture.Status
		reasons = len(record.VerificationPosture.ReasonCodes)
		reasonCodes = record.VerificationPosture.ReasonCodes
	}
	return status, reasons, reasonCodes
}

func auditInspectorContent(record brokerapi.AuditRecordDetail, status string, reasons int, presentation contentPresentationMode) string {
	if presentation == presentationStructured {
		return compactLines(
			fmt.Sprintf("Structured record: family=%s event=%s", record.RecordFamily, record.EventType),
			fmt.Sprintf("Link counts: references=%d reasons=%d", len(record.LinkedReferences), reasons),
		)
	}
	if presentation == presentationRaw {
		return compactLines(
			"Raw record (secrets redacted):",
			fmt.Sprintf("Raw record family=%s event=%s occurred_at=%s", record.RecordFamily, record.EventType, record.OccurredAt),
			fmt.Sprintf("Raw posture status=%s reasons=%d", status, reasons),
		)
	}
	return compactLines(
		fmt.Sprintf("Record family: %s event=%s", record.RecordFamily, record.EventType),
		fmt.Sprintf("Occurred at: %s", record.OccurredAt),
		fmt.Sprintf("Verification posture: %s (%s) reasons=%d", status, renderAnchoringPostureLabel(status), reasons),
		fmt.Sprintf("Linked references: %d", len(record.LinkedReferences)),
	)
}

func auditInspectorReferenceItems(record brokerapi.AuditRecordDetail) []string {
	items := make([]string, 0, len(record.LinkedReferences))
	for _, ref := range record.LinkedReferences {
		items = append(items, fmt.Sprintf("%s:%s(%s)", valueOrNA(ref.ReferenceKind), valueOrNA(ref.ReferenceID), valueOrNA(ref.Relation)))
	}
	return items
}

func auditInspectorContentKind(presentation contentPresentationMode) inspectorContentKind {
	if presentation == presentationRaw {
		return inspectorContentRaw
	}
	if presentation == presentationRendered {
		return inspectorContentLog
	}
	return inspectorContentStructured
}

func auditRouteCopyActions(record *brokerapi.AuditRecordGetResponse) []routeCopyAction {
	if record == nil {
		return nil
	}
	r := record.Record
	digest, _ := r.RecordDigest.Identity()
	linked := make([]string, 0, len(r.LinkedReferences))
	for _, ref := range r.LinkedReferences {
		linked = append(linked, fmt.Sprintf("%s:%s(%s)", valueOrNA(ref.ReferenceKind), valueOrNA(ref.ReferenceID), valueOrNA(ref.Relation)))
	}
	raw := compactLines(
		fmt.Sprintf("record_digest=%s", valueOrNA(digest)),
		fmt.Sprintf("record_family=%s", valueOrNA(r.RecordFamily)),
		fmt.Sprintf("event_type=%s", valueOrNA(r.EventType)),
		fmt.Sprintf("occurred_at=%s", valueOrNA(r.OccurredAt)),
	)
	return compactCopyActions([]routeCopyAction{
		{ID: "record_digest", Label: "record digest", Text: digest},
		{ID: "linked_references", Label: "linked references", Text: strings.Join(linked, "\n")},
		{ID: "raw_block", Label: "raw block", Text: raw},
	})
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
