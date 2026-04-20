package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type dashboardLoadedMsg struct {
	data dashboardData
	err  error
	seq  uint64
}

type dashboardData struct {
	readiness brokerapi.BrokerReadiness
	version   brokerapi.BrokerVersionInfo
	project   brokerapi.ProjectSubstratePostureGetResponse
	runs      []brokerapi.RunSummary
	approvals []brokerapi.ApprovalSummary
	audit     brokerapi.AuditVerificationGetResponse
	auditErr  string
	live      dashboardLiveActivity
}

type dashboardLiveActivity struct {
	runWatch      watchFamilySummary
	approvalWatch watchFamilySummary
	sessionWatch  watchFamilySummary
	feed          []shellLiveActivityEntry
}

type watchFamilySummary struct {
	family        string
	eventCount    int
	snapshotCount int
	upsertCount   int
	terminalCount int
	errorCount    int
	lastEventType string
	lastSubject   string
	lastStatus    string
}

type dashboardRouteModel struct {
	def     routeDefinition
	client  localBrokerClient
	loading bool
	errText string
	data    dashboardData
	loadSeq uint64
}

func newDashboardRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return dashboardRouteModel{def: def, client: client, data: dashboardData{live: dashboardLiveActivity{
		runWatch:      watchFamilySummary{family: "run_watch", lastStatus: "ok"},
		approvalWatch: watchFamilySummary{family: "approval_watch", lastStatus: "ok"},
		sessionWatch:  watchFamilySummary{family: "session_watch", lastStatus: "ok"},
	}}}
}

func (m dashboardRouteModel) ID() routeID { return m.def.ID }

func (m dashboardRouteModel) Title() string { return m.def.Label }

func (m dashboardRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		m.loading = true
		m.errText = ""
		m.loadSeq++
		return m, m.loadCmd(m.loadSeq)
	case tea.KeyMsg:
		if typed.String() != "r" {
			return m, nil
		}
		m.loading = true
		m.errText = ""
		m.loadSeq++
		return m, m.loadCmd(m.loadSeq)
	case dashboardLoadedMsg:
		if typed.seq != m.loadSeq {
			return m, nil
		}
		m.loading = false
		if typed.err != nil {
			m.errText = safeUIErrorText(typed.err)
			return m, nil
		}
		live := m.data.live
		m.data = typed.data
		m.data.live = live
		m.errText = ""
		return m, nil
	case shellLiveActivityUpdatedMsg:
		m.data.live = typed.Live
		m.data.live.feed = append([]shellLiveActivityEntry(nil), typed.Feed...)
		return m, nil
	default:
		return m, nil
	}
}

func (m dashboardRouteModel) View(width, height int, focus focusArea) string {
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Dashboard", "Loading dashboard from broker API...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Dashboard", "Load failed: "+m.errText+" (press r to retry)")
	}
	focusLabel := "inactive"
	if focus == focusContent {
		focusLabel = "active"
	}
	primaryRun := primaryDashboardRun(m.data.runs)
	innerWidth := dashboardContentWidth(width)
	sections := []string{
		compactLines(sectionTitle("Dashboard")+" "+focusBadge(focus)+" "+navStateBadge(focusLabel == "active"), tableHeader("Now")+" "+renderDashboardNowBar(primaryRun, len(m.data.approvals), focusLabel == "active", innerWidth)),
		compactLines(tableHeader("Safety Summary"), renderRunSafetyStrip(primaryDashboardRun(m.data.runs), innerWidth), wrapDashboardLine(renderDashboardSafetyAlerts(m.data), innerWidth)),
		m.controlPlaneSection(innerWidth),
		compactLines(tableHeader("Live Activity"), wrapDashboardLine("Live activity (typed watch families; logs are supplemental inspection only):", innerWidth), wrapDashboardLine(muted("Live activity uses semantic watch families with explicit event types."), innerWidth), renderWatchFamilySummary(m.data.live.runWatch), renderWatchFamilySummary(m.data.live.approvalWatch), renderWatchFamilySummary(m.data.live.sessionWatch), renderLiveActivityFeed(m.data.live.feed)),
		compactLines(tableHeader("Highlights"), wrapDashboardLine(renderRunHighlights(m.data.runs), innerWidth), wrapDashboardLine(renderApprovalHighlights(m.data.approvals), innerWidth)),
		compactLines(tableHeader("Actions")+" "+keyHint("r reload")+" "+muted("tab moves focus • : opens command surface"), keyHint("Route keys: r reload")),
	}
	return joinDashboardSections(sections...)
}

func (m dashboardRouteModel) controlPlaneSection(width int) string {
	parts := []string{
		tableHeader("Control Plane"),
		wrapPartsByWidth([]string{tableHeader("Readiness"), boolBadge("ready", m.data.readiness.Ready), boolBadge("local_only", m.data.readiness.LocalOnly), boolBadge("recovery_complete", m.data.readiness.RecoveryComplete)}, " ", width),
		wrapPartsByWidth([]string{tableHeader("Safety posture"), stateBadgeWithLabel("integrity", m.data.audit.Summary.IntegrityStatus), stateBadgeWithLabel("anchoring", m.data.audit.Summary.AnchoringStatus), boolBadge("degraded", m.data.audit.Summary.CurrentlyDegraded)}, " ", width),
	}
	if notice := strings.TrimSpace(renderDashboardAuditFallbackNotice(m.data.auditErr)); notice != "" {
		parts = append(parts, wrapDashboardLine(notice, width))
	}
	parts = append(parts,
		wrapDashboardLine(renderDashboardProjectSubstrateLine(m.data.project), width),
		wrapDashboardLine(renderDashboardProjectSubstrateGuidance(m.data.project), width),
		wrapDashboardLine(fmt.Sprintf("Workflow posture: runs=%d pending_approvals=%d", len(m.data.runs), pendingApprovalCount(m.data.runs, m.data.approvals)), width),
		wrapDashboardLine(fmt.Sprintf("Version: %s (%s) protocol bundle=%s", m.data.version.ProductVersion, m.data.version.BuildRevision, m.data.version.ProtocolBundleVersion), width),
	)
	return compactLines(parts...)
}

func joinDashboardSections(sections ...string) string {
	nonEmpty := make([]string, 0, len(sections))
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		nonEmpty = append(nonEmpty, section)
	}
	return strings.Join(nonEmpty, "\n\n")
}

func dashboardContentWidth(width int) int {
	if width <= 0 {
		return 1
	}
	contentWidth := width - 4
	if contentWidth < 1 {
		return 1
	}
	return contentWidth
}

func wrapDashboardLine(line string, width int) string {
	line = strings.TrimSpace(line)
	if line == "" || width <= 0 || lipgloss.Width(line) <= width {
		return line
	}
	if strings.Contains(line, " | ") {
		wrapped := wrapPartsByWidth(strings.Split(line, " | "), " | ", width)
		if !hasOverwideDashboardLine(wrapped, width) {
			return wrapped
		}
		line = strings.ReplaceAll(line, " | ", " ")
	}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return line
	}
	lines := []string{parts[0]}
	for _, part := range parts[1:] {
		candidate := lines[len(lines)-1] + " " + part
		if lipgloss.Width(candidate) <= width {
			lines[len(lines)-1] = candidate
			continue
		}
		lines = append(lines, part)
	}
	return strings.Join(lines, "\n")
}

func hasOverwideDashboardLine(content string, width int) bool {
	for _, line := range strings.Split(content, "\n") {
		if lipgloss.Width(line) > width {
			return true
		}
	}
	return false
}

func (m dashboardRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main: routeSurfaceRegion{Title: "Dashboard", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
		},
		Capabilities: routeSurfaceCapabilities{},
		Chrome:       routeSurfaceChrome{Breadcrumbs: []string{"Home", m.def.Label}},
	}
}

func (m dashboardRouteModel) loadCmd(seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		readinessResp, err := m.client.ReadinessGet(ctx)
		if err != nil {
			return dashboardLoadedMsg{err: err, seq: seq}
		}
		versionResp, err := m.client.VersionInfoGet(ctx)
		if err != nil {
			return dashboardLoadedMsg{err: err, seq: seq}
		}
		projectResp, err := m.client.ProjectSubstratePostureGet(ctx)
		if err != nil {
			return dashboardLoadedMsg{err: err, seq: seq}
		}
		runResp, err := m.client.RunList(ctx, 5)
		if err != nil {
			return dashboardLoadedMsg{err: err, seq: seq}
		}
		approvalResp, err := m.client.ApprovalList(ctx, 5)
		if err != nil {
			return dashboardLoadedMsg{err: err, seq: seq}
		}
		auditResp := degradedDashboardAuditFallback()
		auditErr := ""
		if loadedAudit, err := m.client.AuditVerificationGet(ctx, 20); err == nil {
			auditResp = loadedAudit
		} else {
			auditErr = safeUIErrorText(err)
		}
		return dashboardLoadedMsg{data: dashboardData{readiness: readinessResp.Readiness, version: versionResp.VersionInfo, project: projectResp, runs: runResp.Runs, approvals: approvalResp.Approvals, audit: auditResp, auditErr: auditErr}, seq: seq}
	}
}
