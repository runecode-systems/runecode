package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type auditLoadedMsg struct {
	timeline []brokerapi.AuditTimelineViewEntry
	verify   *brokerapi.AuditVerificationGetResponse
	record   *brokerapi.AuditRecordGetResponse
	cursor   string
	next     string
	err      error
	seq      uint64
	nav      auditPageNav
	from     string
}

type auditPageNav string

const (
	auditPageNavNone auditPageNav = "none"
	auditPageNavNext auditPageNav = "next"
	auditPageNavPrev auditPageNav = "prev"
)

type auditRouteModel struct {
	def          routeDefinition
	client       localBrokerClient
	loading      bool
	errText      string
	timeline     []brokerapi.AuditTimelineViewEntry
	selected     int
	verify       *brokerapi.AuditVerificationGetResponse
	active       *brokerapi.AuditRecordGetResponse
	cursor       string
	nextCursor   string
	prevCursors  []string
	presentation contentPresentationMode
	inspectorOn  bool
	loadSeq      uint64
}

func newAuditRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return auditRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered}
}

func (m auditRouteModel) ID() routeID { return m.def.ID }

func (m auditRouteModel) Title() string { return m.def.Label }

func (m auditRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		return m.reload()
	case tea.KeyMsg:
		return m.handleKey(typed)
	case auditLoadedMsg:
		if typed.seq != m.loadSeq {
			return m, nil
		}
		m.loading = false
		if typed.err != nil {
			m.errText = safeUIErrorText(typed.err)
			return m, nil
		}
		switch typed.nav {
		case auditPageNavNext:
			m.prevCursors = append(m.prevCursors, typed.from)
		case auditPageNavPrev:
			if len(m.prevCursors) > 0 {
				m.prevCursors = m.prevCursors[:len(m.prevCursors)-1]
			}
		}
		m.errText = ""
		m.timeline = typed.timeline
		if m.selected >= len(m.timeline) {
			m.selected = 0
		}
		m.verify = typed.verify
		m.active = typed.record
		m.cursor = typed.cursor
		m.nextCursor = typed.next
		return m, nil
	default:
		return m, nil
	}
}

func (m auditRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return "Loading audit timeline and verification posture..."
	}
	if m.errText != "" {
		return compactLines("Audit", "Load failed: "+m.errText, "Press r to retry.")
	}
	body := []string{
		sectionTitle("Audit") + " " + focusBadge(focus),
		renderAuditSafetyAlertStrip(m.verify),
		renderAuditPageSummary(m.cursor, m.nextCursor, len(m.prevCursors), len(m.timeline)),
		renderAuditSummary(m.verify),
		renderAuditFindings(m.verify, m.presentation),
		fmt.Sprintf("Presentation mode=%s", normalizePresentationMode(m.presentation)),
		tableHeader("Timeline"),
		renderAuditTimeline(m.timeline, m.selected),
	}
	if m.inspectorOn {
		body = append(body, tableHeader("Inspector")+" "+appTheme.InspectorHint.Render("(typed record details)"))
		body = append(body, renderAuditInspector(m.active, m.presentation))
	}
	body = append(body, keyHint("Route keys: j/k move, enter record detail, n next page, p previous page, v cycle rendered/raw/structured, i toggle inspector, r reload"))
	return compactLines(body...)
}

func (m auditRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	s := key.String()
	if s == "r" {
		return m.reload()
	}
	if s == "n" {
		return m.loadNextPage()
	}
	if s == "p" {
		return m.loadPrevPage()
	}
	if s == "i" {
		m.inspectorOn = !m.inspectorOn
		return m, nil
	}
	if s == "v" {
		m.presentation = nextPresentationMode(m.presentation)
		return m, nil
	}
	if s == "j" || s == "down" || s == "k" || s == "up" {
		m.moveTimelineSelection(s == "j" || s == "down")
		return m, nil
	}
	if s == "enter" {
		return m.loadSelectedRecord()
	}
	return m, nil
}

func (m auditRouteModel) loadNextPage() (routeModel, tea.Cmd) {
	if m.nextCursor == "" {
		return m, nil
	}
	return m.startPageLoad(m.nextCursor, auditPageNavNext, m.cursor)
}

func (m auditRouteModel) loadPrevPage() (routeModel, tea.Cmd) {
	if len(m.prevCursors) == 0 {
		return m, nil
	}
	prev := m.prevCursors[len(m.prevCursors)-1]
	return m.startPageLoad(prev, auditPageNavPrev, m.cursor)
}

func (m auditRouteModel) startPageLoad(cursor string, nav auditPageNav, from string) (routeModel, tea.Cmd) {
	m.selected = 0
	m.loading = true
	m.errText = ""
	m.loadSeq++
	return m, m.loadPageCmd(cursor, m.loadSeq, nav, from)
}

func (m *auditRouteModel) moveTimelineSelection(forward bool) {
	if len(m.timeline) == 0 {
		return
	}
	if forward {
		m.selected = (m.selected + 1) % len(m.timeline)
		return
	}
	m.selected--
	if m.selected < 0 {
		m.selected = len(m.timeline) - 1
	}
}

func (m auditRouteModel) loadSelectedRecord() (routeModel, tea.Cmd) {
	if len(m.timeline) == 0 {
		return m, nil
	}
	m.loading = true
	m.errText = ""
	digest, _ := m.timeline[m.selected].RecordDigest.Identity()
	m.loadSeq++
	return m, m.loadRecordCmd(digest, m.loadSeq)
}

func (m auditRouteModel) reload() (routeModel, tea.Cmd) {
	m.loading = true
	m.errText = ""
	m.selected = 0
	m.prevCursors = nil
	m.loadSeq++
	return m, m.loadPageCmd("", m.loadSeq, auditPageNavNone, "")
}

func (m auditRouteModel) loadPageCmd(cursor string, seq uint64, nav auditPageNav, from string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		timelineResp, err := m.client.AuditTimeline(ctx, 20, cursor)
		if err != nil {
			return auditLoadedMsg{err: err, seq: seq}
		}
		verifyResp, err := m.client.AuditVerificationGet(ctx, 40)
		if err != nil {
			return auditLoadedMsg{err: err, seq: seq}
		}
		return auditLoadedMsg{timeline: timelineResp.Views, verify: &verifyResp, cursor: cursor, next: timelineResp.NextCursor, seq: seq, nav: nav, from: from}
	}
}

func (m auditRouteModel) loadRecordCmd(recordDigest string, seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		recordResp, err := m.client.AuditRecordGet(ctx, recordDigest)
		if err != nil {
			return auditLoadedMsg{err: err, seq: seq}
		}
		return auditLoadedMsg{timeline: m.timeline, verify: m.verify, record: &recordResp, cursor: m.cursor, next: m.nextCursor, seq: seq}
	}
}

func renderAuditSummary(verify *brokerapi.AuditVerificationGetResponse) string {
	if verify == nil {
		return "Verification posture unavailable"
	}
	s := verify.Summary
	anchorLabel := renderAnchoringPostureLabel(s.AnchoringStatus)
	return fmt.Sprintf("Verification posture: integrity=%s %s anchoring=%s (%s) storage=%s lifecycle=%s degraded=%t %s findings=%d", s.IntegrityStatus, postureBadge(s.IntegrityStatus), s.AnchoringStatus, anchorLabel, s.StoragePostureStatus, s.SegmentLifecycleStatus, s.CurrentlyDegraded, boolBadge("degraded", s.CurrentlyDegraded), s.FindingCount)
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
	if strings.EqualFold(strings.TrimSpace(s.AnchoringStatus), "degraded") || strings.EqualFold(strings.TrimSpace(s.AnchoringStatus), "unanchored") || s.CurrentlyDegraded {
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
