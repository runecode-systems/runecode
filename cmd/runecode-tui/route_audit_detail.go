package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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

func (m auditRouteModel) loadFinalizeVerifyCmd(seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		finalizeResp, err := m.client.AuditFinalizeVerify(ctx)
		if err != nil {
			return auditLoadedMsg{err: err, seq: seq}
		}
		return auditLoadedMsg{timeline: m.timeline, verify: m.verify, finalize: &finalizeResp, record: m.active, cursor: m.cursor, next: m.nextCursor, seq: seq}
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

func (m *auditRouteModel) syncDetailDocument() {
	if m.active == nil {
		m.detailDoc.SetDocument(workbenchObjectRef{Kind: "audit", ID: "none"}, inspectorContentLog, "audit record", "")
		return
	}
	r := m.active.Record
	status, reasons, _ := auditInspectorPosture(r)
	presentation := normalizePresentationMode(m.presentation)
	content := auditInspectorContent(r, status, reasons, presentation)
	kind := auditInspectorContentKind(presentation)
	identity, _ := r.RecordDigest.Identity()
	ref := workbenchObjectRef{Kind: "audit", ID: strings.TrimSpace(identity)}
	m.detailDoc.SetDocument(ref, kind, "audit record", content)
}
