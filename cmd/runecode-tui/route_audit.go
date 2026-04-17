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
	finalize *brokerapi.AuditFinalizeVerifyResponse
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
	finalize     *brokerapi.AuditFinalizeVerifyResponse
	active       *brokerapi.AuditRecordGetResponse
	cursor       string
	nextCursor   string
	prevCursors  []string
	exportCopy   bool
	presentation contentPresentationMode
	inspectorOn  bool
	loadSeq      uint64
	detailDoc    longFormDocumentState
}

type auditAnchorCompletedMsg struct {
	response   *brokerapi.AuditAnchorSegmentResponse
	err        error
	sealDigest string
}

type auditSelectRecordMsg struct {
	Digest string
}

func newAuditRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return auditRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
}

func (m auditRouteModel) ID() routeID { return m.def.ID }

func (m auditRouteModel) Title() string { return m.def.Label }

func (m auditRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case auditSelectRecordMsg:
		digest := strings.TrimSpace(typed.Digest)
		if digest == "" {
			return m, nil
		}
		m.loading = true
		m.errText = ""
		m.loadSeq++
		return m, m.loadRecordCmd(digest, m.loadSeq)
	case routeViewportScrollMsg:
		if typed.Region == routeRegionInspector {
			m.detailDoc.Scroll(typed.Delta)
		}
		return m, nil
	case routeViewportResizeMsg:
		width, height := longFormViewportSizeForShell(typed.Width, typed.Height)
		m.detailDoc.Resize(width, height)
		return m, nil
	case routeShellPreferencesMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		m.inspectorOn = typed.InspectorVisible
		m.presentation = normalizePresentationMode(typed.PreferredMode)
		m.syncDetailDocument()
		return m, nil
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
	if msg.InspectorSet {
		m.inspectorOn = msg.InspectorVisible
	}
	m.presentation = normalizePresentationMode(msg.PreferredMode)
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
	if msg.finalize != nil {
		m.finalize = msg.finalize
	}
	m.active = msg.record
	m.cursor = msg.cursor
	m.nextCursor = msg.next
	m.syncDetailDocument()
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
		return renderStateCard(routeLoadStateLoading, "Audit", "Loading audit timeline and verification posture...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Audit", "Load failed: "+m.errText+" (press r to retry)")
	}
	body := []string{
		sectionTitle("Audit") + " " + focusBadge(focus),
		renderAuditSafetyAlertStrip(m.verify),
		renderAuditFinalizeSummary(m.finalize),
		renderAuditAnchorActionSummary(m),
		renderAuditPageSummary(m.cursor, m.nextCursor, len(m.prevCursors), len(m.timeline)),
		renderAuditSummary(m.verify),
		renderAuditFindings(m.verify, m.presentation),
		renderModeSwitchTabs([]string{string(presentationRendered), string(presentationRaw), string(presentationStructured)}, string(normalizePresentationMode(m.presentation))),
		renderDirectory("Timeline directory", renderAuditDirectoryItems(m.timeline), m.selected),
		renderAuditTimeline(m.timeline, m.selected),
	}
	if len(m.timeline) == 0 {
		body = append(body, muted("The audit timeline is empty; retry after the broker persists verification posture or sealed timeline records."))
	}
	body = append(body, keyHint("Route keys: j/k move, enter record detail, f finalize+verify sealed segment posture, a anchor selected/latest sealed segment, x toggle anchor export-copy, n next page, p previous page, v cycle rendered/raw/structured, i toggle inspector, r reload"))
	return compactLines(body...)
}

func (m auditRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	breadcrumbs := []string{"Home", m.def.Label}
	if m.active != nil {
		if identity, err := m.active.Record.RecordDigest.Identity(); err == nil && strings.TrimSpace(identity) != "" {
			breadcrumbs = append(breadcrumbs, identity)
		}
	}
	status := strings.TrimSpace(m.statusText)
	if status == "" && strings.TrimSpace(m.errText) != "" {
		status = "Load failed: " + strings.TrimSpace(m.errText)
	}
	inspector := ""
	if m.inspectorOn {
		inspector = renderAuditInspector(m.active, m.presentation, &m.detailDoc)
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:      routeSurfaceRegion{Title: "Audit workspace", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Inspector: routeSurfaceRegion{Title: "Audit inspector", Body: inspector},
			Bottom:    routeSurfaceRegion{Body: keyHint("Route keys: j/k move, enter record detail, f finalize+verify sealed segment posture, a anchor selected/latest sealed segment, x toggle anchor export-copy, n next page, p previous page, v cycle rendered/raw/structured, i toggle inspector, r reload")},
			Status:    routeSurfaceRegion{Body: status},
		},
		Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: m.inspectorOn}},
		Chrome:       routeSurfaceChrome{Breadcrumbs: breadcrumbs},
		Actions: routeSurfaceActions{
			ModeTabs:         []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
			ActiveTab:        string(normalizePresentationMode(m.presentation)),
			CopyActions:      auditRouteCopyActions(m.active),
			ReferenceActions: auditInspectorReferenceActions(m.active),
			LocalActions:     auditInspectorLocalActions(),
		},
	}
}

func (m auditRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	s := key.String()
	if model, cmd, handled := m.handleReloadAndPagingKey(s); handled {
		return model, cmd
	}
	if model, cmd, handled := m.handlePresentationAndInspectorKey(s); handled {
		return model, cmd
	}
	if model, cmd, handled := m.handleAnchorAndFinalizeKey(s); handled {
		return model, cmd
	}
	if model, cmd, handled := m.handleSelectionKey(s); handled {
		return model, cmd
	}
	return m, nil
}

func (m auditRouteModel) handleReloadAndPagingKey(key string) (routeModel, tea.Cmd, bool) {
	switch key {
	case "r":
		model, cmd := m.reload()
		return model, cmd, true
	case "n":
		model, cmd := m.loadNextPage()
		return model, cmd, true
	case "p":
		model, cmd := m.loadPrevPage()
		return model, cmd, true
	default:
		return m, nil, false
	}
}

func (m auditRouteModel) handlePresentationAndInspectorKey(key string) (routeModel, tea.Cmd, bool) {
	switch key {
	case "i":
		m.inspectorOn = !m.inspectorOn
		return m, nil, true
	case "x":
		m.exportCopy = !m.exportCopy
		state := "disabled"
		if m.exportCopy {
			state = "enabled"
		}
		m.statusText = fmt.Sprintf("Anchor receipt export copy %s for next action.", state)
		return m, nil, true
	case "v":
		m.presentation = nextPresentationMode(m.presentation)
		m.syncDetailDocument()
		return m, nil, true
	default:
		return m, nil, false
	}
}

func (m auditRouteModel) handleAnchorAndFinalizeKey(key string) (routeModel, tea.Cmd, bool) {
	switch key {
	case "a":
		if m.anchoring {
			m.statusText = "Anchor action already in progress; wait for completion."
			return m, nil, true
		}
		model, cmd := m.anchorSelectedOrLatestSeal()
		return model, cmd, true
	case "f":
		if m.loading {
			return m, nil, true
		}
		m.loading = true
		m.errText = ""
		m.loadSeq++
		return m, m.loadFinalizeVerifyCmd(m.loadSeq), true
	default:
		return m, nil, false
	}
}

func (m auditRouteModel) handleSelectionKey(key string) (routeModel, tea.Cmd, bool) {
	switch key {
	case "j", "down", "k", "up":
		m.moveTimelineSelection(key == "j" || key == "down")
		return m, nil, true
	case "enter":
		model, cmd := m.loadSelectedRecord()
		return model, cmd, true
	default:
		return m, nil, false
	}
}
