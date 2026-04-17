package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type actionCenterLoadedMsg struct {
	approvals []brokerapi.ApprovalSummary
	runs      []brokerapi.RunSummary
	audit     brokerapi.AuditVerificationGetResponse
	err       error
	seq       uint64
}

type actionCenterFamily string

const (
	actionCenterFamilyApprovals  actionCenterFamily = "approvals"
	actionCenterFamilyOps        actionCenterFamily = "operational_attention"
	actionCenterFamilyBlocked    actionCenterFamily = "blocked_work_impact"
	actionCenterExpirySoonWindow                    = 2 * time.Hour
)

type actionCenterItem struct {
	Title     string
	Detail    string
	Urgency   string
	ExpiryCue string
	StaleCue  string
	Impact    string
	Target    paletteTarget
}

type actionCenterRouteModel struct {
	def         routeDefinition
	client      localBrokerClient
	loading     bool
	errText     string
	statusText  string
	approvals   []brokerapi.ApprovalSummary
	runs        []brokerapi.RunSummary
	audit       *brokerapi.AuditVerificationGetResponse
	watch       dashboardLiveActivity
	watchHealth shellSyncHealth
	inspectorOn bool
	family      actionCenterFamily
	selected    map[actionCenterFamily]int
	loadSeq     uint64
}

func newActionCenterRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return actionCenterRouteModel{
		def:         def,
		client:      client,
		inspectorOn: true,
		family:      actionCenterFamilyApprovals,
		selected: map[actionCenterFamily]int{
			actionCenterFamilyApprovals: 0,
			actionCenterFamilyOps:       0,
			actionCenterFamilyBlocked:   0,
		},
		watchHealth: shellSyncHealth{State: shellSyncStateLoading},
		watch: dashboardLiveActivity{
			runWatch:      watchFamilySummary{family: "run_watch", lastStatus: "ok"},
			approvalWatch: watchFamilySummary{family: "approval_watch", lastStatus: "ok"},
			sessionWatch:  watchFamilySummary{family: "session_watch", lastStatus: "ok"},
		},
	}
}

func (m actionCenterRouteModel) ID() routeID { return m.def.ID }

func (m actionCenterRouteModel) Title() string { return m.def.Label }

func (m actionCenterRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		if typed.InspectorSet {
			m.inspectorOn = typed.InspectorVisible
		}
		m.loading = true
		m.errText = ""
		m.statusText = ""
		m.loadSeq++
		return m, m.loadCmd(m.loadSeq)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case routeShellPreferencesMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		m.inspectorOn = typed.InspectorVisible
		return m, nil
	case actionCenterLoadedMsg:
		if typed.seq != m.loadSeq {
			return m, nil
		}
		m.loading = false
		if typed.err != nil {
			m.errText = safeUIErrorText(typed.err)
			return m, nil
		}
		m.errText = ""
		m.approvals = typed.approvals
		m.runs = typed.runs
		m.audit = &typed.audit
		m.normalizeSelection()
		return m, nil
	case shellLiveActivityUpdatedMsg:
		m.watch = typed.Live
		m.watch.feed = append([]shellLiveActivityEntry(nil), typed.Feed...)
		m.watchHealth = typed.Health
		m.normalizeSelection()
		return m, nil
	default:
		return m, nil
	}
}

func (m actionCenterRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Action Center", "Loading approvals, run posture, audit posture, and shell watch health...")
	}
	if strings.TrimSpace(m.errText) != "" {
		return renderStateCard(routeLoadStateError, "Action Center", "Load failed: "+m.errText+" (press r to retry)")
	}
	families := m.familyBuckets()
	return compactLines(
		sectionTitle("Action Center")+" "+focusBadge(focus),
		fmt.Sprintf("Queue families: %s=%d %s=%d %s=%d", infoBadge("approvals"), len(families[actionCenterFamilyApprovals]), warnBadge("operational_attention"), len(families[actionCenterFamilyOps]), dangerBadge("blocked_work_impact"), len(families[actionCenterFamilyBlocked])),
		fmt.Sprintf("Active triage family: %s", stateBadgeWithLabel("family", string(m.family))),
		"Question/answer queues are reserved for future canonical broker models and are intentionally not implemented locally.",
		renderDirectory("Approvals queue (canonical)", renderActionCenterItems(families[actionCenterFamilyApprovals]), m.selectedIndex(actionCenterFamilyApprovals, len(families[actionCenterFamilyApprovals]))),
		renderDirectory("Operational attention", renderActionCenterItems(families[actionCenterFamilyOps]), m.selectedIndex(actionCenterFamilyOps, len(families[actionCenterFamilyOps]))),
		renderDirectory("Blocked-work impact", renderActionCenterItems(families[actionCenterFamilyBlocked]), m.selectedIndex(actionCenterFamilyBlocked, len(families[actionCenterFamilyBlocked]))),
		muted("If every bucket is empty, the control plane is currently waiting on new canonical work or operator intervention."),
		keyHint("Route keys: [/] change family, j/k move, enter drill-down, i toggle inspector, r reload"),
	)
}

func (m actionCenterRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	status := strings.TrimSpace(m.statusText)
	if status == "" && strings.TrimSpace(m.errText) != "" {
		status = "Load failed: " + strings.TrimSpace(m.errText)
	}
	families := m.familyBuckets()
	activeItems := families[m.family]
	selected := m.selectedIndex(m.family, len(activeItems))
	inspector := ""
	if m.inspectorOn {
		inspector = renderActionCenterInspector(m.family, activeItems, selected)
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:      routeSurfaceRegion{Title: "Action Center", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Inspector: routeSurfaceRegion{Title: "Action Center inspector", Body: inspector},
			Bottom:    routeSurfaceRegion{Body: keyHint("Route keys: [/] change family, j/k move, enter drill-down, i toggle inspector, r reload")},
			Status:    routeSurfaceRegion{Body: status},
		},
		Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: m.inspectorOn}},
		Chrome:       routeSurfaceChrome{Breadcrumbs: []string{"Home", m.def.Label}},
	}
}

func (m actionCenterRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	switch key.String() {
	case "r":
		m.loading = true
		m.errText = ""
		m.statusText = ""
		m.loadSeq++
		return m, m.loadCmd(m.loadSeq)
	case "j", "down":
		m.moveSelection(1)
		return m, nil
	case "k", "up":
		m.moveSelection(-1)
		return m, nil
	case "i":
		m.inspectorOn = !m.inspectorOn
		return m, nil
	case "]":
		m.family = m.nextFamily()
		m.normalizeSelection()
		return m, nil
	case "[":
		m.family = m.prevFamily()
		m.normalizeSelection()
		return m, nil
	case "enter":
		item, ok := m.selectedItem()
		if !ok {
			return m, nil
		}
		m.statusText = "Drill-down: " + item.Title
		if strings.TrimSpace(item.Target.Kind) == "" {
			return m, nil
		}
		return m, func() tea.Msg { return paletteActionMsg{Verb: verbJump, Target: item.Target} }
	default:
		return m, nil
	}
}

func (m actionCenterRouteModel) loadCmd(seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		approvalResp, err := m.client.ApprovalList(ctx, 40)
		if err != nil {
			return actionCenterLoadedMsg{err: err, seq: seq}
		}
		runResp, err := m.client.RunList(ctx, 40)
		if err != nil {
			return actionCenterLoadedMsg{err: err, seq: seq}
		}
		auditResp, err := m.client.AuditVerificationGet(ctx, 40)
		if err != nil {
			return actionCenterLoadedMsg{err: err, seq: seq}
		}
		return actionCenterLoadedMsg{approvals: approvalResp.Approvals, runs: runResp.Runs, audit: auditResp, seq: seq}
	}
}
