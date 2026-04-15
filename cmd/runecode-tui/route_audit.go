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
	anchoring    bool
	errText      string
	statusText   string
	timeline     []brokerapi.AuditTimelineViewEntry
	selected     int
	verify       *brokerapi.AuditVerificationGetResponse
	active       *brokerapi.AuditRecordGetResponse
	cursor       string
	nextCursor   string
	prevCursors  []string
	exportCopy   bool
	presentation contentPresentationMode
	inspectorOn  bool
	loadSeq      uint64
}

type auditAnchorCompletedMsg struct {
	response   *brokerapi.AuditAnchorSegmentResponse
	err        error
	sealDigest string
}

func newAuditRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return auditRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered}
}

func (m auditRouteModel) ID() routeID { return m.def.ID }

func (m auditRouteModel) Title() string { return m.def.Label }

func (m auditRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case auditLoadedMsg:
		return m.handleAuditLoaded(typed)
	case auditAnchorCompletedMsg:
		return m.handleAuditAnchorCompleted(typed)
	default:
		return m, nil
	}
}

func (m auditRouteModel) handleRouteActivated(msg routeActivatedMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	return m.reload()
}

func (m auditRouteModel) handleAuditLoaded(msg auditLoadedMsg) (routeModel, tea.Cmd) {
	if msg.seq != m.loadSeq {
		return m, nil
	}
	m.loading = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		return m, nil
	}
	m.applyAuditNavigation(msg)
	m.errText = ""
	m.timeline = msg.timeline
	if m.selected >= len(m.timeline) {
		m.selected = 0
	}
	m.verify = msg.verify
	m.active = msg.record
	m.cursor = msg.cursor
	m.nextCursor = msg.next
	return m, nil
}

func (m *auditRouteModel) applyAuditNavigation(msg auditLoadedMsg) {
	switch msg.nav {
	case auditPageNavNext:
		m.prevCursors = append(m.prevCursors, msg.from)
	case auditPageNavPrev:
		if len(m.prevCursors) > 0 {
			m.prevCursors = m.prevCursors[:len(m.prevCursors)-1]
		}
	}
}

func (m auditRouteModel) handleAuditAnchorCompleted(msg auditAnchorCompletedMsg) (routeModel, tea.Cmd) {
	m.anchoring = false
	m.errText = ""
	if msg.err != nil {
		m.statusText = fmt.Sprintf("Anchor action: failed seal=%s reason=%s", msg.sealDigest, safeUIErrorText(msg.err))
		return m, nil
	}
	if msg.response == nil {
		m.statusText = fmt.Sprintf("Anchor action: failed seal=%s reason=no response", msg.sealDigest)
		return m, nil
	}
	if strings.TrimSpace(msg.response.AnchoringStatus) != "ok" {
		reason := strings.TrimSpace(msg.response.FailureCode)
		if reason == "" {
			reason = valueOrNA(strings.TrimSpace(msg.response.FailureMessage))
		}
		m.statusText = fmt.Sprintf("Anchor action: failed seal=%s reason=%s", msg.sealDigest, reason)
		return m, nil
	}
	m.statusText = fmt.Sprintf("Anchor action: ok seal=%s receipt=%s export_copy=%s", msg.sealDigest, anchorReceiptIdentity(msg.response), auditExportCopyState(m.exportCopy))
	return m, nil
}

func anchorReceiptIdentity(resp *brokerapi.AuditAnchorSegmentResponse) string {
	if resp == nil || resp.ReceiptDigest == nil {
		return "n/a"
	}
	identity, err := resp.ReceiptDigest.Identity()
	if err != nil {
		return "n/a"
	}
	return identity
}

func auditExportCopyState(exportCopy bool) string {
	if exportCopy {
		return "on"
	}
	return "off"
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
		renderAuditAnchorActionSummary(m),
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
	body = append(body, keyHint("Route keys: j/k move, enter record detail, a anchor selected/latest sealed segment, x toggle anchor export-copy, n next page, p previous page, v cycle rendered/raw/structured, i toggle inspector, r reload"))
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
	if s == "x" {
		m.exportCopy = !m.exportCopy
		state := "disabled"
		if m.exportCopy {
			state = "enabled"
		}
		m.statusText = fmt.Sprintf("Anchor receipt export copy %s for next action.", state)
		return m, nil
	}
	if s == "v" {
		m.presentation = nextPresentationMode(m.presentation)
		return m, nil
	}
	if s == "a" {
		if m.anchoring {
			m.statusText = "Anchor action already in progress; wait for completion."
			return m, nil
		}
		return m.anchorSelectedOrLatestSeal()
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
	m.anchoring = false
	m.errText = ""
	m.statusText = ""
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
