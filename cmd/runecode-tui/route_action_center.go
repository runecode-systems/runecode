package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type actionCenterLoadedMsg struct {
	approvals []brokerapi.ApprovalSummary
	runs      []brokerapi.RunSummary
	project   brokerapi.ProjectSubstratePostureGetResponse
	audit     brokerapi.AuditVerificationGetResponse
	auditErr  string
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
	Title          string
	State          routeLoadState
	Urgency        string
	Reason         string
	Impact         string
	Owner          string
	RequiredAction string
	TargetLabel    string
	EvidenceCue    string
	Target         paletteTarget
}

type actionCenterViewModel struct {
	Families map[actionCenterFamily][]actionCenterItem
	Summary  actionCenterSummary
}

type actionCenterRouteModel struct {
	def         routeDefinition
	client      localBrokerClient
	now         func() time.Time
	loading     bool
	errText     string
	statusText  string
	approvals   []brokerapi.ApprovalSummary
	runs        []brokerapi.RunSummary
	project     brokerapi.ProjectSubstratePostureGetResponse
	audit       *brokerapi.AuditVerificationGetResponse
	auditErr    string
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
		now:         time.Now,
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
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case routeShellPreferencesMsg:
		return m.handleShellPreferences(typed)
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
		m.project = typed.project
		m.audit = &typed.audit
		m.auditErr = typed.auditErr
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

func (m actionCenterRouteModel) handleRouteActivated(msg routeActivatedMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	if msg.InspectorSet {
		m.inspectorOn = msg.InspectorVisible
	}
	m.loading = true
	m.errText = ""
	m.statusText = ""
	m.loadSeq++
	return m, m.loadCmd(m.loadSeq)
}

func (m actionCenterRouteModel) handleShellPreferences(msg routeShellPreferencesMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	m.inspectorOn = msg.InspectorVisible
	return m, nil
}

func (m actionCenterRouteModel) View(width, height int, focus focusArea) string {
	_ = height
	if m.loading {
		return renderStateCardSpec(stateCardSpec{
			State:       routeLoadStateLoading,
			Title:       "Action Center",
			Message:     "Loading broker-known approvals, blocked work, setup posture, degraded cues, and shell watch health.",
			Reason:      "Action Center waits for broker-owned follow-up surfaces before presenting operator triage.",
			NextAction:  "Wait for the route to settle, or press r to retry if loading stalls.",
			ShortcutCue: "r reload",
		})
	}
	if strings.TrimSpace(m.errText) != "" {
		return renderStateCardSpec(stateCardSpec{
			State:       routeLoadStateError,
			Title:       "Action Center",
			Message:     "Action Center is temporarily unavailable.",
			Reason:      m.errText,
			NextAction:  "Press r to retry. If the error continues, use Status for broker posture and Dashboard for a lighter overview.",
			ShortcutCue: "r reload",
			RouteCue:    "Status or Dashboard",
		})
	}
	vm := m.snapshot()
	return compactLines(
		sectionTitle("Action Center")+" "+focusBadge(focus),
			renderStateCardSpec(stateCardSpec{
				State:      vm.Summary.State,
				Title:      vm.Summary.Title,
				Message:    vm.Summary.Message,
				Reason:     vm.Summary.Reason,
				NextAction: vm.Summary.NextAction,
				RouteCue:   "Approvals, Runs, Audit, Status",
			}),
			renderActionCenterQueueStrip(vm, m.family),
			renderActionCenterDirectory("Approvals", vm.Families[actionCenterFamilyApprovals], m.selectedIndex(actionCenterFamilyApprovals, len(vm.Families[actionCenterFamilyApprovals])), width),
			renderActionCenterDirectory("Operational Attention", vm.Families[actionCenterFamilyOps], m.selectedIndex(actionCenterFamilyOps, len(vm.Families[actionCenterFamilyOps])), width),
			renderActionCenterDirectory("Blocked Work", vm.Families[actionCenterFamilyBlocked], m.selectedIndex(actionCenterFamilyBlocked, len(vm.Families[actionCenterFamilyBlocked])), width),
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
	vm := m.snapshot()
	activeItems := vm.Families[m.family]
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

func (m actionCenterRouteModel) snapshot() actionCenterViewModel {
	now := time.Now
	if m.now != nil {
		now = m.now
	}
	current := now().UTC()
	families := m.familyBucketsAt(current)
	return actionCenterViewModel{Families: families, Summary: buildActionCenterSummary(families)}
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
		projectResp, err := m.client.ProjectSubstratePostureGet(ctx)
		if err != nil {
			return actionCenterLoadedMsg{err: err, seq: seq}
		}
		auditResp := degradedDashboardAuditFallback()
		auditErr := ""
		if loadedAudit, err := m.client.AuditVerificationGet(ctx, 40); err == nil {
			auditResp = loadedAudit
		} else {
			auditErr = safeUIErrorText(err)
		}
		return actionCenterLoadedMsg{approvals: approvalResp.Approvals, runs: runResp.Runs, project: projectResp, audit: auditResp, auditErr: auditErr, seq: seq}
	}
}
