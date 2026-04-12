package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type statusLoadedMsg struct {
	readiness brokerapi.BrokerReadiness
	version   brokerapi.BrokerVersionInfo
	err       error
	seq       uint64
}

type statusRouteModel struct {
	def     routeDefinition
	client  localBrokerClient
	loading bool
	errText string
	data    statusLoadedMsg
	loadSeq uint64
}

func newStatusRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return statusRouteModel{def: def, client: client}
}

func (m statusRouteModel) ID() routeID { return m.def.ID }

func (m statusRouteModel) Title() string { return m.def.Label }

func (m statusRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
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
	case statusLoadedMsg:
		if typed.seq != m.loadSeq {
			return m, nil
		}
		m.loading = false
		if typed.err != nil {
			m.errText = safeUIErrorText(typed.err)
			return m, nil
		}
		m.data = typed
		m.errText = ""
		return m, nil
	default:
		return m, nil
	}
}

func (m statusRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return "Loading status route from readiness/version contracts..."
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
		keyHint("Route keys: r reload"),
	)
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
		return statusLoadedMsg{readiness: readinessResp.Readiness, version: versionResp.VersionInfo, seq: seq}
	}
}
