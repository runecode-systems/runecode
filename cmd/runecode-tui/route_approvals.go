package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type approvalsLoadedMsg struct {
	items  []brokerapi.ApprovalSummary
	detail *brokerapi.ApprovalGetResponse
	err    error
	seq    uint64
}

type approvalResolvedMsg struct {
	approvalID string
	result     string
	err        error
}

type approvalsSelectMsg struct {
	ApprovalID string
}

type approvalsRouteModel struct {
	def          routeDefinition
	client       localBrokerClient
	loading      bool
	resolving    bool
	errText      string
	statusText   string
	items        []brokerapi.ApprovalSummary
	selected     int
	active       *brokerapi.ApprovalGetResponse
	presentation contentPresentationMode
	inspectorOn  bool
	loadSeq      uint64
	detailDoc    longFormDocumentState
}

func newApprovalsRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return approvalsRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
}

func (m approvalsRouteModel) ID() routeID { return m.def.ID }

func (m approvalsRouteModel) Title() string { return m.def.Label }

func (m approvalsRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case approvalsSelectMsg:
		return m.handleApprovalsSelect(typed)
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
	case approvalsLoadedMsg:
		return m.handleApprovalsLoaded(typed)
	case approvalResolvedMsg:
		return m.handleApprovalResolved(typed)
	default:
		return m, nil
	}
}

func (m approvalsRouteModel) handleRouteActivated(msg routeActivatedMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	if msg.InspectorSet {
		m.inspectorOn = msg.InspectorVisible
	}
	m.presentation = normalizePresentationMode(msg.PreferredMode)
	return m.reload()
}

func (m approvalsRouteModel) handleApprovalsSelect(msg approvalsSelectMsg) (routeModel, tea.Cmd) {
	approvalID := strings.TrimSpace(msg.ApprovalID)
	if approvalID == "" {
		return m, nil
	}
	m.loading = true
	m.errText = ""
	m.statusText = ""
	m.loadSeq++
	return m, m.loadCmd(approvalID, m.loadSeq)
}

func (m approvalsRouteModel) handleApprovalsLoaded(msg approvalsLoadedMsg) (routeModel, tea.Cmd) {
	if msg.seq != m.loadSeq {
		return m, nil
	}
	m.loading = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		return m, nil
	}
	m.errText = ""
	m.items = msg.items
	if m.selected >= len(m.items) {
		m.selected = 0
	}
	m.active = msg.detail
	m.syncDetailDocument()
	return m, nil
}

func (m approvalsRouteModel) handleApprovalResolved(msg approvalResolvedMsg) (routeModel, tea.Cmd) {
	m.resolving = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		m.statusText = ""
		return m, nil
	}
	m.errText = ""
	m.statusText = fmt.Sprintf("Approval %s resolved via typed ApprovalResolve (%s).", msg.approvalID, valueOrNA(msg.result))
	m.loading = true
	m.loadSeq++
	return m, m.loadCmd("", m.loadSeq)
}

func (m approvalsRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Approvals", "Loading approvals from broker approval contracts...")
	}
	if m.resolving {
		return renderStateCard(routeLoadStateLoading, "Approvals", "Resolving approval via typed broker ApprovalResolve...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Approvals", "Load failed: "+m.errText+" (press r to retry)")
	}
	body := []string{
		sectionTitle("Approvals") + " " + focusBadge(focus),
		renderApprovalSafetyStrip(m.active),
		renderApprovalFlowPath(m.active),
		renderModeSwitchTabs([]string{string(presentationRendered), string(presentationRaw), string(presentationStructured)}, string(normalizePresentationMode(m.presentation))),
		renderDirectory("Approval queue", renderApprovalDirectoryItems(m.items), m.selected),
	}
	if m.statusText != "" {
		body = append(body, "Status: "+m.statusText)
	}
	body = append(body, keyHint("Route keys: j/k move, enter load detail, a resolve current approval where supported, v cycle rendered/raw/structured, i toggle inspector, r reload"))
	return compactLines(body...)
}

func (m approvalsRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	breadcrumbs := []string{"Home", m.def.Label}
	if m.active != nil && strings.TrimSpace(m.active.Approval.ApprovalID) != "" {
		breadcrumbs = append(breadcrumbs, strings.TrimSpace(m.active.Approval.ApprovalID))
	}
	status := strings.TrimSpace(m.statusText)
	if status == "" && strings.TrimSpace(m.errText) != "" {
		status = "Load failed: " + strings.TrimSpace(m.errText)
	}
	inspector := ""
	if m.inspectorOn {
		inspector = renderApprovalInspector(m.active, m.presentation, &m.detailDoc)
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:      routeSurfaceRegion{Title: "Approval workspace", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Inspector: routeSurfaceRegion{Title: "Approval inspector", Body: inspector},
			Bottom:    routeSurfaceRegion{Body: keyHint("Route keys: j/k move, enter load detail, a resolve current approval where supported, v cycle rendered/raw/structured, i toggle inspector, r reload")},
			Status:    routeSurfaceRegion{Body: status},
		},
		Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: m.inspectorOn}},
		Chrome:       routeSurfaceChrome{Breadcrumbs: breadcrumbs},
		Actions: routeSurfaceActions{
			ModeTabs:         []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
			ActiveTab:        string(normalizePresentationMode(m.presentation)),
			CopyActions:      approvalRouteCopyActions(m.active),
			ReferenceActions: approvalInspectorReferenceActions(m.active),
			LocalActions:     approvalInspectorLocalActions(),
		},
	}
}

func (m approvalsRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	switch key.String() {
	case "r":
		return m.reload()
	case "a":
		return m.handleResolveKey()
	case "i":
		m.inspectorOn = !m.inspectorOn
		return m, nil
	case "v":
		m.presentation = nextPresentationMode(m.presentation)
		m.syncDetailDocument()
		return m, nil
	case "j", "down":
		if len(m.items) == 0 {
			return m, nil
		}
		m.selected = (m.selected + 1) % len(m.items)
		return m, nil
	case "k", "up":
		if len(m.items) == 0 {
			return m, nil
		}
		m.selected--
		if m.selected < 0 {
			m.selected = len(m.items) - 1
		}
		return m, nil
	case "enter":
		if len(m.items) == 0 {
			return m, nil
		}
		m.loading = true
		m.errText = ""
		m.statusText = ""
		m.loadSeq++
		return m, m.loadCmd(m.items[m.selected].ApprovalID, m.loadSeq)
	default:
		return m, nil
	}
}

func (m approvalsRouteModel) handleResolveKey() (routeModel, tea.Cmd) {
	if m.active == nil {
		m.statusText = "Load an approval detail first (enter), then press a to resolve."
		return m, nil
	}
	if err := validateApprovalResolveInput(*m.active); err != nil {
		m.resolving = false
		m.errText = ""
		m.statusText = safeUIErrorText(err)
		return m, nil
	}
	m.resolving = true
	m.errText = ""
	m.statusText = ""
	return m, m.resolveCmd(*m.active)
}

func (m approvalsRouteModel) reload() (routeModel, tea.Cmd) {
	m.loading = true
	m.errText = ""
	m.statusText = ""
	m.loadSeq++
	target := ""
	if m.selected >= 0 && m.selected < len(m.items) {
		target = m.items[m.selected].ApprovalID
	}
	return m, m.loadCmd(target, m.loadSeq)
}

func (m approvalsRouteModel) resolveCmd(resp brokerapi.ApprovalGetResponse) tea.Cmd {
	return func() tea.Msg {
		resolveReq, err := approvalResolveRequestFromDetail(resp)
		if err != nil {
			return approvalResolvedMsg{approvalID: strings.TrimSpace(resp.Approval.ApprovalID), err: err}
		}
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resolveResp, err := m.client.ApprovalResolve(ctx, resolveReq)
		if err != nil {
			return approvalResolvedMsg{approvalID: strings.TrimSpace(resp.Approval.ApprovalID), err: err}
		}
		result := strings.TrimSpace(resolveResp.ResolutionReasonCode)
		if result == "" {
			result = strings.TrimSpace(resolveResp.ResolutionStatus)
		}
		if result == "" {
			result = "resolved"
		}
		return approvalResolvedMsg{approvalID: strings.TrimSpace(resp.Approval.ApprovalID), result: result}
	}
}

func (m approvalsRouteModel) loadCmd(approvalID string, seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		listResp, err := m.client.ApprovalList(ctx, 30)
		if err != nil {
			return approvalsLoadedMsg{err: err, seq: seq}
		}
		target := approvalID
		if target != "" && !containsApprovalSummary(listResp.Approvals, target) {
			target = ""
		}
		if target == "" && len(listResp.Approvals) > 0 {
			target = listResp.Approvals[0].ApprovalID
		}
		if target == "" {
			return approvalsLoadedMsg{items: listResp.Approvals, seq: seq}
		}
		getResp, err := m.client.ApprovalGet(ctx, target)
		if err != nil {
			return approvalsLoadedMsg{err: err, seq: seq}
		}
		return approvalsLoadedMsg{items: listResp.Approvals, detail: &getResp, seq: seq}
	}
}

func containsApprovalSummary(items []brokerapi.ApprovalSummary, approvalID string) bool {
	approvalID = strings.TrimSpace(approvalID)
	if approvalID == "" {
		return false
	}
	for _, item := range items {
		if strings.TrimSpace(item.ApprovalID) == approvalID {
			return true
		}
	}
	return false
}

func (m *approvalsRouteModel) syncDetailDocument() {
	if m.active == nil {
		m.detailDoc.SetDocument(workbenchObjectRef{Kind: "approval", ID: "none"}, inspectorContentStructured, "approval details", "")
		return
	}
	summary := m.active.Approval
	detail := m.active.ApprovalDetail
	identity := detail.BoundIdentity
	boundScope := summary.BoundScope
	bindingLabel := approvalBindingLabel(detail.BindingKind)
	lifecycleState := detail.LifecycleDetail.LifecycleState
	lifecycleFlags := renderApprovalLifecycleFlags(detail.LifecycleDetail)
	presentation := normalizePresentationMode(m.presentation)
	content := approvalInspectorContent(summary, detail, identity, boundScope, bindingLabel, lifecycleState, lifecycleFlags, presentation)
	kind := approvalInspectorContentKind(presentation)
	ref := workbenchObjectRef{Kind: "approval", ID: strings.TrimSpace(summary.ApprovalID), WorkspaceID: strings.TrimSpace(boundScope.WorkspaceID)}
	m.detailDoc.SetDocument(ref, kind, "approval details", content)
}
