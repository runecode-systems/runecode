package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func renderAuditSummary(verify *brokerapi.AuditVerificationGetResponse) string {
	if verify == nil {
		return "Audit health: unavailable"
	}
	s := verify.Summary
	anchorLabel := renderAnchoringPostureLabel(s.AnchoringStatus)
	return fmt.Sprintf("Audit health: integrity=%s %s • receipts=%s (%s) • findings=%d • %s", s.IntegrityStatus, postureBadge(s.IntegrityStatus), s.AnchoringStatus, anchorLabel, s.FindingCount, boolBadge("degraded", s.CurrentlyDegraded))
}

func renderAuditFinalizeSummary(finalize *brokerapi.AuditFinalizeVerifyResponse) string {
	if finalize == nil {
		return "Finalize/verify: unavailable"
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
	line := fmt.Sprintf("Finalize/verify: %s • segment %s • report %s", status, valueOrNA(strings.TrimSpace(finalize.SegmentID)), shortIdentity(report))
	if status == "ok" {
		return line + " • next review findings or anchor/export the latest verified segment"
	}
	reason := sanitizeUIText(strings.TrimSpace(finalize.FailureCode))
	if reason == "" {
		reason = sanitizeUIText(strings.TrimSpace(finalize.FailureMessage))
	}
	if reason != "" {
		line += " • reason " + reason
	}
	return line
}

func renderAuditPageSummary(cursor, next string, backDepth, count int) string {
	page := "page 1"
	if cursor != "" {
		page = fmt.Sprintf("page %s", cursor)
	}
	nextLabel := "no more pages"
	if next != "" {
		nextLabel = "next page available"
	}
	return fmt.Sprintf("Timeline: %s • %d records • %s • back %d", page, count, nextLabel, backDepth)
}

func renderAuditFindings(verify *brokerapi.AuditVerificationGetResponse, presentation contentPresentationMode) string {
	presentation = normalizePresentationMode(presentation)
	if verify == nil {
		return "Verification findings: unavailable"
	}
	if len(verify.Report.Findings) == 0 {
		return "Verification findings: none; the current audit posture does not report degraded or failed reasons."
	}
	if presentation == presentationStructured {
		return fmt.Sprintf("Verification findings (structured): total=%d degraded_reasons=%d hard_failures=%d", len(verify.Report.Findings), len(verify.Report.DegradedReasons), len(verify.Report.HardFailures))
	}
	line := "Findings to review:"
	for i, finding := range verify.Report.Findings {
		if presentation == presentationRendered && i >= 4 {
			line += fmt.Sprintf("\n  ... (%d more)", len(verify.Report.Findings)-i)
			break
		}
		if i >= 4 {
			line += fmt.Sprintf("\n  ... (%d more in raw mode)", len(verify.Report.Findings)-i)
			break
		}
		line += fmt.Sprintf("\n  - %s • %s • %s", finding.Code, finding.Severity, finding.Dimension)
	}
	if len(verify.Report.DegradedReasons) > 0 {
		line += fmt.Sprintf("\n  degraded reasons: %s", joinCSV(verify.Report.DegradedReasons))
	}
	if len(verify.Report.HardFailures) > 0 {
		line += fmt.Sprintf("\n  hard failures: %s", joinCSV(verify.Report.HardFailures))
	}
	return line
}

func renderAuditDirectoryItems(timeline []brokerapi.AuditTimelineViewEntry) []string {
	items := make([]string, 0, len(timeline))
	for _, entry := range timeline {
		items = append(items, auditTimelineDirectoryItem(entry))
	}
	return items
}

func renderAuditSelectedEntryCue(timeline []brokerapi.AuditTimelineViewEntry, selected int) string {
	if len(timeline) == 0 || selected < 0 || selected >= len(timeline) {
		return ""
	}
	entry := timeline[selected]
	label := auditRecordDisplayLabel(entry)
	posture := auditTimelinePostureLabel(entry)
	refs := auditReferenceCue(entry.LinkedReferences)
	parts := []string{fmt.Sprintf("Selected record: %s", label), posture, "press enter for inspector detail"}
	if entry.VerificationPosture != nil && len(entry.VerificationPosture.ReasonCodes) > 0 {
		parts = append(parts, "reasons "+joinCSV(entry.VerificationPosture.ReasonCodes))
	}
	if refs != "" {
		parts = append(parts, refs)
	}
	return strings.Join(parts, " • ")
}

func renderAuditWorkbenchSummary(verify *brokerapi.AuditVerificationGetResponse, finalize *brokerapi.AuditFinalizeVerifyResponse, record *brokerapi.AuditRecordGetResponse, cursor, next string, backDepth, count int, anchoring, exportCopy bool, statusText string) string {
	recordLabel := "latest audit record"
	if record != nil {
		identity, _ := record.Record.RecordDigest.Identity()
		recordLabel = fmt.Sprintf("record %s", shortIdentity(identity))
	}
	health := renderAuditSafetyAlertStrip(verify)
	evidence := renderAuditEvidenceTrail(verify, record)
	finalizeLine := renderAuditFinalizeSummary(finalize)
	anchorLine := renderAuditAnchorWorkbenchLine(anchoring, exportCopy, statusText)
	paging := renderAuditPageSummary(cursor, next, backDepth, count)
	return compactLines(
		tableHeader("Evidence workbench"),
		fmt.Sprintf("Health: %s", health),
		fmt.Sprintf("Record in focus: %s", recordLabel),
		evidence,
		fmt.Sprintf("Next actions: %s", renderAuditNextActionCue(verify, finalize)),
		fmt.Sprintf("Audit summary: %s", renderAuditSummary(verify)),
		finalizeLine,
		anchorLine,
		paging,
	)
}

func renderAuditInspector(record *brokerapi.AuditRecordGetResponse, presentation contentPresentationMode, document *longFormDocumentState) string {
	if record == nil {
		return "  Select a timeline record and press enter to load detail."
	}
	if document == nil {
		fallback := newLongFormDocumentState()
		document = &fallback
	}
	presentation = normalizePresentationMode(presentation)
	r := record.Record
	status, reasons, reasonCodes := auditInspectorPosture(r)
	content := auditInspectorContent(r, status, reasons, presentation)
	identity, _ := r.RecordDigest.Identity()
	referenceItems := auditInspectorReferenceItems(r)
	reasonItems := mapReferenceIDs(reasonCodes, func(string) paletteActionMsg {
		return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}
	})
	contentKind := auditInspectorContentKind(presentation)
	document.SetDocument(workbenchObjectRef{Kind: "audit", ID: strings.TrimSpace(identity)}, contentKind, "audit record", content)
	return renderInspectorShell(inspectorShellSpec{
		Title:        "Audit inspector",
		Summary:      fmt.Sprintf("%s with %d linked reference(s).", auditEventLabel(valueOrNA(r.EventType)), len(r.LinkedReferences)),
		Identity:     fmt.Sprintf("Record %s • %s", shortIdentity(identity), valueOrNA(r.RecordFamily)),
		Status:       fmt.Sprintf("Audit health %s • %d reason code(s)", valueOrNA(status), reasons),
		Badges:       []string{stateBadgeWithLabel("posture", status), appTheme.InspectorHint.Render("record details")},
		References:   []inspectorReference{{Label: "records", Items: referenceItems}, {Label: "reason codes", Items: reasonItems}},
		LocalActions: auditInspectorLocalActions(),
		CopyActions:  auditRouteCopyActions(record),
		ModeTabs:     []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:   string(presentation),
		Document:     document,
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
			"Evidence trail: audit record -> verification posture -> export/offline verification or anchoring where available",
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
		fmt.Sprintf("Primary digest display: %s (copy raw digest below)", auditRecordDigestDetail(record)),
		fmt.Sprintf("Audit health: %s (%s) reasons=%d", status, renderAnchoringPostureLabel(status), reasons),
		fmt.Sprintf("Linked references: %d", len(record.LinkedReferences)),
		"Evidence trail: workflow result -> artifacts -> audit record -> final checks -> offline review or receipts.",
		"Trust posture: linked records stay authoritative; copied digests and raw content are supplemental proof material.",
	)
}

func auditInspectorReferenceItems(record brokerapi.AuditRecordDetail) []inspectorReferenceItem {
	items := make([]inspectorReferenceItem, 0, len(record.LinkedReferences))
	for _, ref := range record.LinkedReferences {
		label := fmt.Sprintf("%s:%s(%s)", valueOrNA(ref.ReferenceKind), valueOrNA(ref.ReferenceID), valueOrNA(ref.Relation))
		kind := strings.ToLower(strings.TrimSpace(ref.ReferenceKind))
		id := strings.TrimSpace(ref.ReferenceID)
		action := paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}
		switch kind {
		case "run":
			action = paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "run", RouteID: routeRuns, RunID: id}}
		case "approval":
			action = paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: id}}
		case "artifact":
			action = paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "artifact", RouteID: routeArtifacts, Digest: id}}
		case "audit", "audit_record":
			action = paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "audit", RouteID: routeAudit, Digest: id}}
		}
		items = append(items, inspectorReferenceItem{Label: label, Action: action})
	}
	return items
}

func auditInspectorLocalActions() []routeActionItem {
	return []routeActionItem{
		{Label: "jump:runs", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeRuns}}},
		{Label: "jump:approvals", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeApprovals}}},
		{Label: "jump:artifacts", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeArtifacts}}},
		{Label: "copy:record_digest"},
	}
}

func auditInspectorReferenceActions(record *brokerapi.AuditRecordGetResponse) []routeActionItem {
	if record == nil {
		return nil
	}
	out := make([]routeActionItem, 0, len(record.Record.LinkedReferences))
	for _, ref := range record.Record.LinkedReferences {
		kind := strings.ToLower(strings.TrimSpace(ref.ReferenceKind))
		id := strings.TrimSpace(ref.ReferenceID)
		if id == "" {
			continue
		}
		switch kind {
		case "run":
			out = append(out, routeActionItem{Label: "run:" + id, Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "run", RouteID: routeRuns, RunID: id}}})
		case "approval":
			out = append(out, routeActionItem{Label: "approval:" + id, Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: id}}})
		case "artifact":
			out = append(out, routeActionItem{Label: "artifact:" + id, Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "artifact", RouteID: routeArtifacts, Digest: id}}})
		case "audit", "audit_record":
			out = append(out, routeActionItem{Label: "record:" + id, Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "audit", RouteID: routeAudit, Digest: id}}})
		}
	}
	return out
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
		return tableHeader("Audit health") + " " + dangerBadge("Audit unavailable") + " audit verification unavailable"
	}
	s := verify.Summary
	parts := []string{tableHeader("Audit health")}
	if strings.EqualFold(strings.TrimSpace(s.AnchoringStatus), "degraded") || s.CurrentlyDegraded {
		parts = append(parts, auditDegradedBadge("Receipts need review"))
	}
	if strings.EqualFold(strings.TrimSpace(s.AnchoringStatus), "failed") || strings.EqualFold(strings.TrimSpace(s.IntegrityStatus), "failed") || strings.EqualFold(strings.TrimSpace(s.IntegrityStatus), "invalid") {
		parts = append(parts, dangerBadge("Audit failed"))
	}
	if len(parts) == 1 {
		parts = append(parts, successBadge("Audit ready"))
	}
	return strings.Join(parts, " ")
}

func renderAuditOverviewCard(verify *brokerapi.AuditVerificationGetResponse, record *brokerapi.AuditRecordGetResponse) string {
	if verify == nil {
		return renderStateCardSpec(stateCardSpec{State: routeLoadStateWaiting, Title: "Verification trail", Message: "Audit verification posture is not loaded yet.", Reason: "The route needs broker verification data before it can explain export, offline verification, or anchoring posture.", NextAction: "Reload this route, then inspect findings and anchoring actions.", ShortcutCue: "r", RouteCue: "Audit"})
	}
	state := routeLoadStateReady
	if verify.Summary.CurrentlyDegraded || strings.EqualFold(strings.TrimSpace(verify.Summary.AnchoringStatus), "degraded") {
		state = routeLoadStateDegraded
	}
	if strings.EqualFold(strings.TrimSpace(verify.Summary.IntegrityStatus), "failed") || strings.EqualFold(strings.TrimSpace(verify.Summary.AnchoringStatus), "failed") {
		state = routeLoadStateBlocked
	}
	recordLabel := "latest verification posture"
	if record != nil {
		identity, _ := record.Record.RecordDigest.Identity()
		recordLabel = "record " + shortIdentity(identity)
	}
	return renderStateCardSpec(stateCardSpec{State: state, Title: "Verification trail", Message: fmt.Sprintf("Audit explains the current health of %s.", recordLabel), Reason: renderAuditSummary(verify), NextAction: "Review findings, then use finalize/verify, export copy, or receipt actions as needed.", ShortcutCue: "f / a / x", RouteCue: "Audit"})
}

func renderAuditEvidenceTrail(verify *brokerapi.AuditVerificationGetResponse, record *brokerapi.AuditRecordGetResponse) string {
	recordLabel := "timeline records"
	if record != nil {
		identity, _ := record.Record.RecordDigest.Identity()
		recordLabel = "record " + shortIdentity(identity)
	}
	if verify == nil {
		return fmt.Sprintf("Evidence trail: workflow result -> artifacts -> audit records (%s) -> audit health unavailable until broker verification data loads", recordLabel)
	}
	return fmt.Sprintf("Evidence trail: workflow result -> artifacts -> audit records (%s) -> final checks -> export/offline review -> receipts", recordLabel)
}
