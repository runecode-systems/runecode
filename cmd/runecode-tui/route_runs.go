package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type runsLoadedMsg struct {
	runs   []brokerapi.RunSummary
	detail *brokerapi.RunDetail
	err    error
	seq    uint64
}

type runsSelectRunMsg struct {
	RunID string
}

type runsRouteModel struct {
	def          routeDefinition
	client       localBrokerClient
	loading      bool
	errText      string
	runs         []brokerapi.RunSummary
	selected     int
	active       *brokerapi.RunDetail
	presentation contentPresentationMode
	inspectorOn  bool
	loadSeq      uint64
	detailDoc    longFormDocumentState
}

func newRunsRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return runsRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
}

func (m runsRouteModel) ID() routeID { return m.def.ID }

func (m runsRouteModel) Title() string { return m.def.Label }

func (m runsRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case runsSelectRunMsg:
		return m.handleRunSelect(typed)
	case routeViewportScrollMsg:
		return m.handleViewportScroll(typed)
	case routeViewportResizeMsg:
		return m.handleViewportResize(typed)
	case routeShellPreferencesMsg:
		return m.handleShellPreferences(typed)
	case runsLoadedMsg:
		if typed.seq != m.loadSeq {
			return m, nil
		}
		m.loading = false
		if typed.err != nil {
			m.errText = safeUIErrorText(typed.err)
			return m, nil
		}
		m.errText = ""
		m.runs = typed.runs
		if m.selected >= len(m.runs) {
			m.selected = 0
		}
		m.active = typed.detail
		if typed.detail != nil {
			m.selected = selectedRunIndex(m.runs, typed.detail.Summary.RunID)
		}
		m.syncDetailDocument()
		return m, nil
	default:
		return m, nil
	}
}

func (m runsRouteModel) handleRouteActivated(msg routeActivatedMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	if msg.InspectorSet {
		m.inspectorOn = msg.InspectorVisible
	}
	m.presentation = normalizePresentationMode(msg.PreferredMode)
	return m.reload()
}

func (m runsRouteModel) handleRunSelect(msg runsSelectRunMsg) (routeModel, tea.Cmd) {
	runID := strings.TrimSpace(msg.RunID)
	if runID == "" {
		return m, nil
	}
	m.loading = true
	m.errText = ""
	m.loadSeq++
	return m, m.loadCmd(runID, m.loadSeq)
}

func selectedRunIndex(runs []brokerapi.RunSummary, runID string) int {
	runID = strings.TrimSpace(runID)
	for i, run := range runs {
		if strings.TrimSpace(run.RunID) == runID {
			return i
		}
	}
	return 0
}

func (m runsRouteModel) handleViewportScroll(msg routeViewportScrollMsg) (routeModel, tea.Cmd) {
	if msg.Region == routeRegionInspector {
		m.detailDoc.Scroll(msg.Delta)
	}
	return m, nil
}

func (m runsRouteModel) handleViewportResize(msg routeViewportResizeMsg) (routeModel, tea.Cmd) {
	width, height := longFormViewportSizeForShell(msg.Width, msg.Height)
	m.detailDoc.Resize(width, height)
	return m, nil
}

func (m runsRouteModel) handleShellPreferences(msg routeShellPreferencesMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	m.inspectorOn = msg.InspectorVisible
	m.presentation = normalizePresentationMode(msg.PreferredMode)
	m.syncDetailDocument()
	return m, nil
}

func (m runsRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Runs", "Loading runs from broker run summaries/details...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Runs", "Load failed: "+m.errText+" (press r to retry)")
	}
	body := []string{
		sectionTitle("Runs") + " " + focusBadge(focus),
		renderRunSafetyStrip(m.activeSummary(), width-4),
		renderModeSwitchTabs([]string{string(presentationRendered), string(presentationRaw), string(presentationStructured)}, string(normalizePresentationMode(m.presentation))),
		renderDirectory("Run directory", renderRunDirectoryItems(m.runs), m.selected),
	}
	if len(m.runs) == 0 {
		body = append(body, muted("No runs are available yet; reload after the broker reports canonical run activity."))
	}
	body = append(body, keyHint("Route keys: j/k move, enter load detail, i toggle inspector, v cycle rendered/raw/structured, r reload"))
	return compactLines(body...)
}

func (m runsRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	breadcrumbs := []string{"Home", m.def.Label}
	if m.active != nil && m.active.Summary.RunID != "" {
		breadcrumbs = append(breadcrumbs, m.active.Summary.RunID)
	}
	status := ""
	if m.errText != "" {
		status = "Load failed: " + m.errText
	}
	inspector := ""
	if m.inspectorOn {
		inspector = renderRunInspector(m.active, m.presentation, &m.detailDoc)
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:      routeSurfaceRegion{Title: "Run workbench", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Inspector: routeSurfaceRegion{Title: "Run inspector", Body: inspector},
			Bottom:    routeSurfaceRegion{Body: keyHint("Route keys: j/k move, enter load detail, i toggle inspector, v cycle rendered/raw/structured, r reload")},
			Status:    routeSurfaceRegion{Body: status},
		},
		Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: m.inspectorOn}},
		Chrome:       routeSurfaceChrome{Breadcrumbs: breadcrumbs},
		Actions: routeSurfaceActions{
			ModeTabs:         []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
			ActiveTab:        string(normalizePresentationMode(m.presentation)),
			CopyActions:      runRouteCopyActions(m.active),
			ReferenceActions: runInspectorReferenceActions(m.active),
			LocalActions:     runInspectorLocalActions(),
		},
	}
}

func (m runsRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	switch key.String() {
	case "r":
		return m.reload()
	case "i":
		m.inspectorOn = !m.inspectorOn
		return m, nil
	case "v":
		m.presentation = nextPresentationMode(m.presentation)
		m.syncDetailDocument()
		return m, nil
	case "j", "down":
		if len(m.runs) == 0 {
			return m, nil
		}
		m.selected = (m.selected + 1) % len(m.runs)
		return m, nil
	case "k", "up":
		if len(m.runs) == 0 {
			return m, nil
		}
		m.selected--
		if m.selected < 0 {
			m.selected = len(m.runs) - 1
		}
		return m, nil
	case "enter":
		if len(m.runs) == 0 {
			return m, nil
		}
		m.loading = true
		m.errText = ""
		m.loadSeq++
		return m, m.loadCmd(m.runs[m.selected].RunID, m.loadSeq)
	default:
		return m, nil
	}
}

func (m runsRouteModel) reload() (routeModel, tea.Cmd) {
	m.loading = true
	m.errText = ""
	m.loadSeq++
	target := ""
	if m.selected >= 0 && m.selected < len(m.runs) {
		target = m.runs[m.selected].RunID
	}
	return m, m.loadCmd(target, m.loadSeq)
}

func (m runsRouteModel) loadCmd(runID string, seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		listResp, err := m.client.RunList(ctx, 30)
		if err != nil {
			return runsLoadedMsg{err: err, seq: seq}
		}
		target := runID
		if target != "" && !containsRunSummary(listResp.Runs, target) {
			target = ""
		}
		if target == "" && len(listResp.Runs) > 0 {
			target = listResp.Runs[0].RunID
		}
		if target == "" {
			return runsLoadedMsg{runs: listResp.Runs, seq: seq}
		}
		getResp, err := m.client.RunGet(ctx, target)
		if err != nil {
			return runsLoadedMsg{err: err, seq: seq}
		}
		return runsLoadedMsg{runs: listResp.Runs, detail: &getResp.Run, seq: seq}
	}
}

func containsRunSummary(items []brokerapi.RunSummary, runID string) bool {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return false
	}
	for _, item := range items {
		if strings.TrimSpace(item.RunID) == runID {
			return true
		}
	}
	return false
}

func renderRunList(runs []brokerapi.RunSummary, selected int) string {
	if len(runs) == 0 {
		return "  - no runs"
	}
	line := ""
	for i, run := range runs {
		marker := " "
		if i == selected {
			marker = ">"
		}
		line += selectedLine(i == selected, fmt.Sprintf("  %s %s %s approvals=%d", marker, run.RunID, stateBadgeWithLabel("state", run.LifecycleState), run.PendingApprovalCount)) + "\n"
		line += fmt.Sprintf("      %s | %s | %s | %s | approval_profile=%s\n", fmt.Sprintf("backend_kind=%s", valueOrNA(run.BackendKind)), renderRuntimeIsolationCue(run.BackendKind, run.IsolationAssuranceLevel), renderProvisioningPostureCue(run.ProvisioningPosture), renderAuditPostureCue(run.AuditIntegrityStatus, run.AuditAnchoringStatus, run.AuditCurrentlyDegraded), valueOrNA(run.ApprovalProfile))
	}
	return line
}

func renderRunDirectoryItems(runs []brokerapi.RunSummary) []string {
	items := make([]string, 0, len(runs))
	for _, run := range runs {
		items = append(items, fmt.Sprintf("%s %s approvals=%d", run.RunID, stateBadgeWithLabel("state", run.LifecycleState), run.PendingApprovalCount))
	}
	return items
}
