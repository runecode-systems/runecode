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
	err       error
	seq       uint64
}

type postureChangedMsg struct {
	resp brokerapi.BackendPostureChangeResponse
	err  error
}

type statusRouteModel struct {
	def      routeDefinition
	client   localBrokerClient
	loading  bool
	changing bool
	errText  string
	status   string
	data     statusLoadedMsg
	loadSeq  uint64
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
	switch msg.String() {
	case "r":
		m = m.beginLoad()
		return m, m.loadCmd(m.loadSeq)
	case "c":
		if m.changing || m.loading {
			return m, nil
		}
		m.changing = true
		m.errText = ""
		m.status = ""
		return m, m.changeCmd()
	default:
		return m, nil
	}
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
		return m, tea.Batch(func() tea.Msg { return routeSwitchMsg{RouteID: routeApprovals} }, m.loadCmd(m.loadSeq))
	}
	m.status = fmt.Sprintf("Backend posture outcome=%s reason=%s", msg.resp.Outcome.Outcome, msg.resp.Outcome.OutcomeReasonCode)
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
		return "Loading status route from readiness/version contracts..."
	}
	if m.changing {
		return "Submitting backend posture change through broker local API..."
	}
	if m.errText != "" {
		return compactLines("Status", "Load failed: "+m.errText, "Press r to retry.")
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
		renderBackendPostureLine(m.data.posture),
		m.status,
		keyHint("Route keys: r reload, c request backend posture change"),
	)
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
		postureResp, err := m.client.BackendPostureGet(ctx)
		if err != nil {
			return statusLoadedMsg{err: err, seq: seq}
		}
		return statusLoadedMsg{readiness: readinessResp.Readiness, version: versionResp.VersionInfo, posture: postureResp.Posture, seq: seq}
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
