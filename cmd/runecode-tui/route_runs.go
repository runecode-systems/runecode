package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type runsLoadedMsg struct {
	runs   []brokerapi.RunSummary
	detail *brokerapi.RunDetail
	err    error
	seq    uint64
}

type runsRouteModel struct {
	def         routeDefinition
	client      localBrokerClient
	loading     bool
	errText     string
	runs        []brokerapi.RunSummary
	selected    int
	active      *brokerapi.RunDetail
	inspectorOn bool
	loadSeq     uint64
}

func newRunsRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return runsRouteModel{def: def, client: client, inspectorOn: true}
}

func (m runsRouteModel) ID() routeID { return m.def.ID }

func (m runsRouteModel) Title() string { return m.def.Label }

func (m runsRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		return m.reload()
	case tea.KeyMsg:
		return m.handleKey(typed)
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
		return m, nil
	default:
		return m, nil
	}
}

func (m runsRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return "Loading runs from broker run summaries/details..."
	}
	if m.errText != "" {
		return compactLines("Runs", "Load failed: "+m.errText, "Press r to retry.")
	}
	body := []string{
		sectionTitle("Runs") + " " + focusBadge(focus),
		renderRunSafetyStrip(m.activeSummary()),
		tableHeader("Run list"),
		renderRunList(m.runs, m.selected),
	}
	if m.inspectorOn {
		body = append(body, tableHeader("Inspector")+" "+appTheme.InspectorHint.Render("(authoritative vs advisory state shown)"))
		body = append(body, renderRunInspector(m.active))
	}
	body = append(body, keyHint("Route keys: j/k move, enter load detail, i toggle inspector, r reload"))
	return compactLines(body...)
}

func (m runsRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	switch key.String() {
	case "r":
		return m.reload()
	case "i":
		m.inspectorOn = !m.inspectorOn
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
	return m, m.loadCmd("", m.loadSeq)
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

func renderRunInspector(detail *brokerapi.RunDetail) string {
	if detail == nil {
		return "  Select a run and press enter to load detail."
	}
	summary := detail.Summary
	waitingStages := 0
	for _, stage := range detail.StageSummaries {
		if stage.PendingApprovalCount > 0 {
			waitingStages++
		}
	}
	waitingRoles := 0
	for _, role := range detail.RoleSummaries {
		if role.WaitReasonCode != "" {
			waitingRoles++
		}
	}
	return compactLines(
		fmt.Sprintf("  backend_kind=%s", summary.BackendKind),
		"  Runtime isolation assurance (authoritative): "+renderRuntimeIsolationCue(summary.BackendKind, summary.IsolationAssuranceLevel),
		"  Provisioning/binding posture (authoritative): "+renderProvisioningPostureCue(summary.ProvisioningPosture),
		"  Audit posture (authoritative): "+renderAuditPostureCue(summary.AuditIntegrityStatus, summary.AuditAnchoringStatus, summary.AuditCurrentlyDegraded),
		fmt.Sprintf("  Approval profile (authoritative): %s", renderApprovalProfileCue(summary.ApprovalProfile)),
		fmt.Sprintf("  Authoritative broker state (control-plane truth): %d keys", len(detail.AuthoritativeState)),
		fmt.Sprintf("  Advisory state (non-authoritative runner hints): %d keys %s", len(detail.AdvisoryState), renderAdvisoryStateCue(detail.AdvisoryState)),
		fmt.Sprintf("  Coordination summary: blocked=%t wait_reason=%s mode=%s locks=%d conflicts=%d", detail.Coordination.Blocked, detail.Coordination.WaitReasonCode, detail.Coordination.CoordinationMode, detail.Coordination.LockCount, detail.Coordination.ConflictCount),
		fmt.Sprintf("  Blocking cue: %s (reason=%s)", renderBlockingStateCue(detail.Coordination.Blocked, detail.Coordination.WaitReasonCode), valueOrNA(detail.Coordination.WaitReasonCode)),
		fmt.Sprintf("  Lifecycle blocking reason code: %s -> %s", valueOrNA(summary.BlockingReasonCode), renderBlockingStateCue(summary.BlockingReasonCode != "", summary.BlockingReasonCode)),
		fmt.Sprintf("  Stage summaries: %d total, %d with pending approvals", len(detail.StageSummaries), waitingStages),
		fmt.Sprintf("  Role summaries: %d total, %d reporting coordination waits", len(detail.RoleSummaries), waitingRoles),
		fmt.Sprintf("  Pending approvals=%d active manifests=%d policy refs=%d", len(detail.PendingApprovalIDs), len(detail.ActiveManifestHashes), len(detail.LatestPolicyDecisionRefs)),
	)
}

func (m runsRouteModel) activeSummary() brokerapi.RunSummary {
	if m.active != nil {
		return m.active.Summary
	}
	if len(m.runs) > 0 {
		return m.runs[m.selected]
	}
	return brokerapi.RunSummary{}
}
