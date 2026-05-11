package main

import (
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
		return renderStateCardSpec(stateCardSpec{
			State:       routeLoadStateLoading,
			Title:       "Dashboard",
			Message:     "Refreshing the executive overview.",
			Reason:      "Dashboard waits for broker-owned status before updating.",
			NextAction:  "Wait for the broker response or press r to retry.",
			ShortcutCue: "r reload",
			RouteCue:    "Action Center",
		})
	}
	if m.errText != "" {
		return renderStateCardSpec(stateCardSpec{
			State:       routeLoadStateError,
			Title:       "Dashboard",
			Message:     "Dashboard is temporarily unavailable.",
			Reason:      m.errText,
			NextAction:  "Press r to retry. Use Status or Action Center if needed.",
			ShortcutCue: "r reload",
			RouteCue:    "Status or Action Center",
		})
	}
	innerWidth := dashboardContentWidth(width)
	cardWidth := innerWidth - 4
	if cardWidth < 1 {
		cardWidth = 1
	}
	snapshot := buildDashboardSnapshot(m.data)
	sections := []string{
		compactLines(
			sectionTitle("Dashboard")+" "+focusBadge(focus),
			renderDashboardHero(snapshot, cardWidth),
		),
		renderProductCard(productCardSpec{Tone: dashboardCardTone(snapshot), Title: "Current work", Lines: []string{
			wrapDashboardLine(renderDashboardCurrentWork(m.data, snapshot), cardWidth),
			wrapDashboardLine(renderDashboardApprovalCue(m.data, snapshot), cardWidth),
		}}),
		renderProductCard(productCardSpec{Tone: visualToneInfo, Title: "At a glance", Lines: []string{
			wrapDashboardLine(dashboardMetricStrip(snapshot), cardWidth),
		}}),
		renderProductCard(productCardSpec{Tone: dashboardCardTone(snapshot), Title: "Next action", Lines: []string{
			wrapDashboardLine(renderDashboardNextActions(m.data, snapshot), cardWidth),
			wrapDashboardLine(renderDashboardDetailCue(snapshot), cardWidth),
		}}),
	}
	return joinDashboardSections(sections...)
}

func joinDashboardSections(sections ...string) string {
	nonEmpty := make([]string, 0, len(sections))
	for _, section := range sections {
		section = strings.Trim(section, "\n")
		if strings.TrimSpace(section) == "" {
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
			Main:   routeSurfaceRegion{Title: "Dashboard", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Bottom: routeSurfaceRegion{Body: keyHint("r reload • 5 Action Center • 7 Audit • 8 Status")},
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
