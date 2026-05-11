package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type statusLoadedMsg struct {
	readiness brokerapi.BrokerReadiness
	version   brokerapi.BrokerVersionInfo
	lifecycle brokerapi.BrokerProductLifecyclePosture
	posture   brokerapi.BackendPostureState
	project   brokerapi.ProjectSubstratePostureGetResponse
	err       error
	seq       uint64
}

type postureChangedMsg struct {
	resp brokerapi.BackendPostureChangeResponse
	err  error
}

type projectSubstrateActionResultMsg struct {
	status         string
	err            error
	adoption       *brokerapi.ProjectSubstrateAdoptResponse
	initPreview    *brokerapi.ProjectSubstrateInitPreviewResponse
	upgradePreview *brokerapi.ProjectSubstrateUpgradePreviewResponse
	actionCard     *stateCardSpec
	reload         bool
}

const statusRouteKeyHintText = "Keys: r reload • c backend • a adopt • i/I init • u/U upgrade"

const projectSubstrateHandleAcquiredText = "<acquired>"

type statusRouteModel struct {
	def                        routeDefinition
	client                     localBrokerClient
	loading                    bool
	changing                   bool
	actioning                  bool
	validatingProjectSubstrate bool
	actionMsg                  string
	errText                    string
	status                     string
	projectSubstrateActionCard *stateCardSpec
	data                       statusLoadedMsg
	loadSeq                    uint64
}

func newStatusRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return statusRouteModel{def: def, client: client}
}

func (m statusRouteModel) ID() routeID { return m.def.ID }

func (m statusRouteModel) Title() string { return m.def.Label }

func (m statusRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKeyMsg(typed)
	case statusLoadedMsg:
		return m.handleStatusLoaded(typed)
	case postureChangedMsg:
		return m.handlePostureChanged(typed)
	case projectSubstrateActionResultMsg:
		return m.handleProjectSubstrateActionResult(typed)
	default:
		return m, nil
	}
}

func (m statusRouteModel) handleRouteActivated(msg routeActivatedMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	m.validatingProjectSubstrate = false
	m = m.beginLoad()
	return m, m.loadCmd(m.loadSeq)
}

func (m statusRouteModel) handleKeyMsg(msg tea.KeyMsg) (routeModel, tea.Cmd) {
	key := msg.String()
	if key == "r" {
		m.validatingProjectSubstrate = false
		m = m.beginLoad()
		return m, m.loadCmd(m.loadSeq)
	}
	if key == "c" {
		return m.beginBackendPostureChange()
	}
	return m.beginProjectSubstrateAction(key)
}

func (m statusRouteModel) beginBackendPostureChange() (routeModel, tea.Cmd) {
	if m.changing || m.loading || m.actioning {
		return m, nil
	}
	m.changing = true
	m.errText = ""
	m.status = ""
	return m, m.changeCmd()
}

func (m statusRouteModel) handleStatusLoaded(msg statusLoadedMsg) (routeModel, tea.Cmd) {
	if msg.seq != m.loadSeq {
		return m, nil
	}
	wasValidating := m.validatingProjectSubstrate
	m.loading = false
	m.validatingProjectSubstrate = false
	if msg.err != nil {
		if wasValidating {
			m.projectSubstrateActionCard = projectSubstrateValidationFailureCard(m.data.project, safeUIErrorText(msg.err))
			m.errText = ""
			m.status = "Project setup validation could not be refreshed. Review the broker guidance below."
			return m, nil
		}
		m.errText = safeUIErrorText(msg.err)
		return m, nil
	}
	m.data = msg
	m.errText = ""
	m.projectSubstrateActionCard = nil
	if wasValidating {
		m.status = "Project setup validation refreshed. Review the updated managed-operation and setup posture below."
		return m, nil
	}
	if m.status == "" {
		m.status = "Status refreshed. Review managed-operation and project setup guidance below."
	}
	return m, nil
}

func (m statusRouteModel) handlePostureChanged(msg postureChangedMsg) (routeModel, tea.Cmd) {
	m.changing = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		m.status = ""
		return m, nil
	}
	m.errText = ""
	m = m.beginLoad()
	if msg.resp.Outcome.Outcome == "approval_required" {
		m.status = "Backend selection needs approval; opening Approvals."
		return m, tea.Batch(func() tea.Msg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeApprovals}}
		}, m.loadCmd(m.loadSeq))
	}
	m.status = fmt.Sprintf("Backend selection result: outcome=%s reason=%s", msg.resp.Outcome.Outcome, msg.resp.Outcome.OutcomeReasonCode)
	return m, m.loadCmd(m.loadSeq)
}

func (m statusRouteModel) handleProjectSubstrateActionResult(msg projectSubstrateActionResultMsg) (routeModel, tea.Cmd) {
	m.actioning = false
	m.actionMsg = ""
	if msg.actionCard != nil {
		m.projectSubstrateActionCard = msg.actionCard
	}
	if msg.err != nil {
		m.validatingProjectSubstrate = false
		if msg.actionCard != nil {
			m.errText = ""
			m.status = strings.TrimSpace(msg.status)
			return m, nil
		}
		m.errText = safeUIErrorText(msg.err)
		m.status = strings.TrimSpace(msg.status)
		return m, nil
	}
	m.errText = ""
	if msg.adoption != nil {
		m.data.project.Adoption = msg.adoption.Adoption
	}
	if msg.initPreview != nil {
		m.data.project.InitPreview = msg.initPreview.Preview
	}
	if msg.upgradePreview != nil {
		m.data.project.UpgradePreview = msg.upgradePreview.Preview
	}
	m.status = msg.status
	if !msg.reload {
		m.validatingProjectSubstrate = false
		return m, nil
	}
	m.validatingProjectSubstrate = true
	m = m.beginLoad()
	return m, m.loadCmd(m.loadSeq)
}

func (m statusRouteModel) beginLoad() statusRouteModel {
	m.loading = true
	m.errText = ""
	m.loadSeq++
	return m
}

func (m statusRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if card := m.renderTransientStatusCard(); card != "" {
		return card
	}
	return m.renderReadyStatusView(focus)
}

func (m statusRouteModel) renderTransientStatusCard() string {
	if m.loading {
		if m.validatingProjectSubstrate {
			return renderStateCardSpec(stateCardSpec{
				State:      routeLoadStateLoading,
				Title:      "Project setup",
				Message:    "Validating broker-owned project setup after apply.",
				Reason:     valueOrNA(strings.TrimSpace(m.status)),
				NextAction: "Wait for refreshed managed-operation and project setup status before continuing normal work.",
			})
		}
		return renderStateCardSpec(stateCardSpec{
			State:       routeLoadStateLoading,
			Title:       "Status",
			Message:     "Refreshing broker-owned readiness, lifecycle, and project setup posture.",
			Reason:      "RuneCode waits for authoritative broker status before updating this route.",
			ShortcutCue: "r reload",
		})
	}
	if m.changing {
		return renderStateCardSpec(stateCardSpec{
			State:      routeLoadStateLoading,
			Title:      "Backend selection",
			Message:    "Submitting backend selection through the broker-owned flow.",
			Reason:     "RuneCode waits for broker approval or completion before updating local status.",
			NextAction: "Wait for the broker response or review Approvals if one is required.",
			RouteCue:   "Approvals",
		})
	}
	if m.actioning {
		return renderStateCardSpec(stateCardSpec{
			State:      routeLoadStateLoading,
			Title:      "Project setup",
			Message:    valueOrNA(strings.TrimSpace(m.actionMsg)),
			Reason:     "Setup and remediation stay broker-owned; the TUI only reflects the resulting posture.",
			NextAction: "Wait for broker status, then review the refreshed setup guidance below.",
		})
	}
	if m.errText != "" {
		return renderStateCardSpec(stateCardSpec{
			State:       routeLoadStateError,
			Title:       "Status",
			Message:     "Status is temporarily unavailable.",
			Reason:      m.errText,
			NextAction:  "Press r to retry. If a setup apply failed, rerun preview before applying again.",
			ShortcutCue: "r reload",
		})
	}
	return ""
}

func (m statusRouteModel) renderReadyStatusView(focus focusArea) string {
	r := m.data.readiness
	v := m.data.version
	return compactLines(
		sectionTitle("Status")+" "+focusBadge(focus),
		renderStatusSafetyStrip(r),
		renderStatusOverviewLine(r, m.data.lifecycle, m.data.project),
		renderStatusSetupLine(m.data.project),
		renderReadinessDiagnostics(r),
		renderVersionPostureLine(v),
		renderLifecycleOperationCard(m.data.lifecycle),
		renderLifecycleStatusLine(m.data.lifecycle),
		renderLifecycleBlockedReasonLine(m.data.lifecycle),
		renderLifecycleDegradedReasonLine(m.data.lifecycle),
		renderProjectSubstrateStatusCard(m.data.project),
		renderProjectSubstrateActionCard(m.projectSubstrateActionCard),
		renderProjectSubstrateStatusLine(m.data.project),
		renderProjectSubstrateGuidance(m.data.project),
		renderBackendPostureLine(m.data.posture),
		m.status,
		keyHint(statusRouteKeyHintText),
	)
}

func (m statusRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	status := strings.TrimSpace(m.status)
	if status == "" && strings.TrimSpace(m.errText) != "" {
		status = "Load failed: " + strings.TrimSpace(m.errText)
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:   routeSurfaceRegion{Title: "System status", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Bottom: routeSurfaceRegion{Body: keyHint(statusRouteKeyHintText)},
			Status: routeSurfaceRegion{Body: status},
		},
		Capabilities: routeSurfaceCapabilities{},
		Chrome:       routeSurfaceChrome{Breadcrumbs: []string{"Home", m.def.Label}},
	}
}

func (m statusRouteModel) loadCmd(seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		readinessResp, err := m.client.ReadinessGet(ctx)
		if err != nil {
			return statusLoadedMsg{err: err, seq: seq}
		}
		versionResp, err := m.client.VersionInfoGet(ctx)
		if err != nil {
			return statusLoadedMsg{err: err, seq: seq}
		}
		projectResp, err := m.client.ProjectSubstratePostureGet(ctx)
		if err != nil {
			return statusLoadedMsg{err: err, seq: seq}
		}
		lifecycleResp, err := m.client.ProductLifecyclePostureGet(ctx)
		if err != nil {
			return statusLoadedMsg{err: err, seq: seq}
		}
		postureResp, err := m.client.BackendPostureGet(ctx)
		if err != nil {
			return statusLoadedMsg{err: err, seq: seq}
		}
		return statusLoadedMsg{readiness: readinessResp.Readiness, version: versionResp.VersionInfo, lifecycle: lifecycleResp.ProductLifecycle, posture: postureResp.Posture, project: projectResp, seq: seq}
	}
}

func (m statusRouteModel) changeCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		current := m.data.posture
		targetBackend := "container"
		if strings.EqualFold(strings.TrimSpace(current.BackendKind), "container") {
			targetBackend = "microvm"
		}
		resp, err := m.client.BackendPostureChange(ctx, brokerapi.BackendPostureChangeRequest{
			TargetInstanceID:             strings.TrimSpace(current.InstanceID),
			TargetBackendKind:            targetBackend,
			SelectionMode:                "explicit_selection",
			ChangeKind:                   "select_backend",
			AssuranceChangeKind:          "reduce_assurance",
			OptInKind:                    "exact_action_approval",
			ReducedAssuranceAcknowledged: true,
			Reason:                       "operator_requested_reduced_assurance_backend_opt_in",
		})
		if err != nil {
			return postureChangedMsg{err: err}
		}
		return postureChangedMsg{resp: resp}
	}
}
