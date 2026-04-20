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
	status string
	err    error
}

const statusRouteKeyHintText = "Route keys: r reload, c request backend posture change, a adopt substrate, i init preview, I init apply, u upgrade preview, U upgrade apply"

const projectSubstrateHandleAcquiredText = "<acquired>"

type statusRouteModel struct {
	def       routeDefinition
	client    localBrokerClient
	loading   bool
	changing  bool
	actioning bool
	actionMsg string
	errText   string
	status    string
	data      statusLoadedMsg
	loadSeq   uint64
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
	m = m.beginLoad()
	return m, m.loadCmd(m.loadSeq)
}

func (m statusRouteModel) handleKeyMsg(msg tea.KeyMsg) (routeModel, tea.Cmd) {
	key := msg.String()
	if key == "r" {
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
	m.loading = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		return m, nil
	}
	m.data = msg
	m.errText = ""
	if m.status == "" {
		m.status = "Press c to request backend posture change (microvm/container)."
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
		m.status = "Backend posture change requires approval; opening shared Approvals route."
		return m, tea.Batch(func() tea.Msg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeApprovals}}
		}, m.loadCmd(m.loadSeq))
	}
	m.status = fmt.Sprintf("Backend posture outcome=%s reason=%s", msg.resp.Outcome.Outcome, msg.resp.Outcome.OutcomeReasonCode)
	return m, m.loadCmd(m.loadSeq)
}

func (m statusRouteModel) handleProjectSubstrateActionResult(msg projectSubstrateActionResultMsg) (routeModel, tea.Cmd) {
	m.actioning = false
	m.actionMsg = ""
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		m.status = ""
		return m, nil
	}
	m.errText = ""
	m.status = msg.status
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
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Status", "Loading status route from readiness/version contracts...")
	}
	if m.changing {
		return renderStateCard(routeLoadStateLoading, "Status", "Submitting backend posture change through broker local API...")
	}
	if m.actioning {
		return renderStateCard(routeLoadStateLoading, "Status", valueOrNA(strings.TrimSpace(m.actionMsg)))
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Status", "Load failed: "+m.errText+" (press r to retry)")
	}
	r := m.data.readiness
	v := m.data.version
	diagnostics := renderReadinessDiagnostics(r)
	return compactLines(
		sectionTitle("Status")+" "+focusBadge(focus),
		renderStatusSafetyStrip(r),
		fmt.Sprintf("Broker ready=%t local_only=%t channel=%s", r.Ready, r.LocalOnly, r.ConsumptionChannel),
		fmt.Sprintf("Broker cues: %s %s", boolBadge("ready", r.Ready), boolBadge("local_only", r.LocalOnly)),
		fmt.Sprintf("Subsystem posture: recovery=%t append=%t writable=%t verifier_material=%t derived_index=%t", r.RecoveryComplete, r.AppendPositionStable, r.CurrentSegmentWritable, r.VerifierMaterialAvailable, r.DerivedIndexCaughtUp),
		diagnostics,
		fmt.Sprintf("Version posture: product=%s revision=%s build=%s", v.ProductVersion, v.BuildRevision, v.BuildTime),
		fmt.Sprintf("Protocol posture: bundle=%s manifest=%s api=%s/%s", v.ProtocolBundleVersion, v.ProtocolBundleManifestHash, v.APIFamily, v.APIVersion),
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

func renderBackendPostureLine(posture brokerapi.BackendPostureState) string {
	if strings.TrimSpace(posture.InstanceID) == "" {
		return "Backend posture: unavailable"
	}
	return fmt.Sprintf("Backend posture: instance=%s backend=%s reduced_assurance=%t pending_approval=%t", valueOrNA(posture.InstanceID), valueOrNA(posture.BackendKind), posture.ReducedAssuranceActive, posture.PendingApproval)
}

func renderStatusSafetyStrip(r brokerapi.BrokerReadiness) string {
	parts := []string{tableHeader("Runtime/audit readiness strip")}
	if !r.VerifierMaterialAvailable {
		parts = append(parts, dangerBadge("RUNTIME_POSTURE_AUTH_UNAVAILABLE"))
	} else {
		parts = append(parts, successBadge("RUNTIME_POSTURE_AUTH_AVAILABLE"))
	}
	if !r.RecoveryComplete || !r.AppendPositionStable || !r.CurrentSegmentWritable {
		parts = append(parts, auditDegradedBadge("AUDIT_POSTURE_DEGRADED_OR_UNAVAILABLE"))
	} else {
		parts = append(parts, successBadge("AUDIT_STORAGE_NOMINAL"))
	}
	if !r.Ready {
		parts = append(parts, systemFailureBadge("SYSTEM_FAILURE_BROKER_NOT_READY"))
	}
	return compactLines(parts...)
}

func renderReadinessDiagnostics(r brokerapi.BrokerReadiness) string {
	issues := []string{}
	if !r.RecoveryComplete {
		issues = append(issues, "recovery=incomplete")
	}
	if !r.AppendPositionStable {
		issues = append(issues, "ledger_append=unstable")
	}
	if !r.CurrentSegmentWritable {
		issues = append(issues, "current_segment=read_only_or_unavailable")
	}
	if !r.VerifierMaterialAvailable {
		issues = append(issues, "verifier_material=missing")
	}
	if !r.DerivedIndexCaughtUp {
		issues = append(issues, "derived_index=lagging")
	}
	if len(issues) == 0 {
		return "Diagnostics: all readiness subsystems report nominal posture."
	}
	return fmt.Sprintf("Diagnostics: degraded subsystems=%s", joinCSV(issues))
}

func renderProjectSubstrateStatusLine(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	summary := posture.PostureSummary
	if strings.TrimSpace(summary.SchemaID) == "" {
		return "Project substrate posture: unavailable"
	}
	return fmt.Sprintf(
		"Project substrate posture: state=%s compatibility=%s normal_operation_allowed=%t",
		valueOrNA(summary.ValidationState),
		valueOrNA(summary.CompatibilityPosture),
		summary.NormalOperationAllowed,
	)
}

func renderProjectSubstrateGuidance(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	parts := []string{}
	if strings.TrimSpace(posture.BlockedExplanation) != "" {
		parts = append(parts, "Project substrate block: "+sanitizeUIText(posture.BlockedExplanation))
	}
	if len(posture.RemediationGuidance) > 0 {
		parts = append(parts, "Project substrate remediation: "+joinCSV(posture.RemediationGuidance))
	}
	if strings.TrimSpace(posture.InitPreview.Status) != "" {
		parts = append(parts, fmt.Sprintf("Project substrate init: status=%s", posture.InitPreview.Status))
	}
	if strings.TrimSpace(posture.UpgradePreview.Status) != "" {
		parts = append(parts, fmt.Sprintf("Project substrate upgrade: status=%s", posture.UpgradePreview.Status))
	}
	if len(parts) == 0 {
		return "Project substrate guidance: none"
	}
	return compactLines(parts...)
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
		postureResp, err := m.client.BackendPostureGet(ctx)
		if err != nil {
			return statusLoadedMsg{err: err, seq: seq}
		}
		return statusLoadedMsg{readiness: readinessResp.Readiness, version: versionResp.VersionInfo, posture: postureResp.Posture, project: projectResp, seq: seq}
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
