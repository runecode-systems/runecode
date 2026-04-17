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
}

func newRunsRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return runsRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered}
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
	case runsSelectRunMsg:
		runID := strings.TrimSpace(typed.RunID)
		if runID == "" {
			return m, nil
		}
		m.loading = true
		m.errText = ""
		m.loadSeq++
		return m, m.loadCmd(runID, m.loadSeq)
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
		return renderStateCard(routeLoadStateLoading, "Runs", "Loading runs from broker run summaries/details...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Runs", "Load failed: "+m.errText+" (press r to retry)")
	}
	body := []string{
		sectionTitle("Runs") + " " + focusBadge(focus),
		renderRunSafetyStrip(m.activeSummary()),
		renderModeSwitchTabs([]string{string(presentationRendered), string(presentationRaw), string(presentationStructured)}, string(normalizePresentationMode(m.presentation))),
		renderDirectory("Run directory", renderRunDirectoryItems(m.runs), m.selected),
	}
	if m.inspectorOn {
		body = append(body, renderRunInspector(m.active, m.presentation))
	}
	body = append(body, keyHint("Route keys: j/k move, enter load detail, i toggle inspector, v cycle rendered/raw/structured, r reload"))
	return compactLines(body...)
}

func (m runsRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	breadcrumbs := []string{"Home", m.def.Label}
	if m.active != nil && m.active.Summary.RunID != "" {
		breadcrumbs = append(breadcrumbs, m.active.Summary.RunID)
	}
	status := ""
	if m.errText != "" {
		status = "Load failed: " + m.errText
	}
	return routeSurface{
		Main:           m.View(ctx.Width, ctx.Height, ctx.Focus),
		Inspector:      renderRunInspector(m.active, m.presentation),
		BottomStrip:    keyHint("Route keys: j/k move, enter load detail, i toggle inspector, v cycle rendered/raw/structured, r reload"),
		Status:         status,
		Breadcrumbs:    breadcrumbs,
		MainTitle:      "Run workbench",
		InspectorTitle: "Run inspector",
		ModeTabs:       []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveTab:      string(normalizePresentationMode(m.presentation)),
		CopyActions:    runRouteCopyActions(m.active),
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

func renderRunDirectoryItems(runs []brokerapi.RunSummary) []string {
	items := make([]string, 0, len(runs))
	for _, run := range runs {
		items = append(items, fmt.Sprintf("%s %s approvals=%d", run.RunID, stateBadgeWithLabel("state", run.LifecycleState), run.PendingApprovalCount))
	}
	return items
}

func renderRunInspector(detail *brokerapi.RunDetail, presentation contentPresentationMode) string {
	if detail == nil {
		return "  Select a run and press enter to load detail."
	}
	summary := detail.Summary
	presentation = normalizePresentationMode(presentation)
	waitingStages := pendingApprovalStageCount(detail.StageSummaries)
	waitingRoles := waitingRoleCount(detail.RoleSummaries)
	content := runInspectorContent(summary, detail, waitingStages, waitingRoles, presentation)
	contentKind := runInspectorContentKind(presentation)
	return renderInspectorShell(inspectorShellSpec{
		Title:          "Run inspector",
		Summary:        fmt.Sprintf("run=%s lifecycle=%s pending_approvals=%d", summary.RunID, valueOrNA(summary.LifecycleState), summary.PendingApprovalCount),
		Identity:       fmt.Sprintf("run=%s backend=%s", summary.RunID, valueOrNA(summary.BackendKind)),
		Status:         fmt.Sprintf("lifecycle=%s blocked=%t", valueOrNA(summary.LifecycleState), detail.Coordination.Blocked),
		Badges:         []string{stateBadgeWithLabel("state", summary.LifecycleState), appTheme.InspectorHint.Render("authoritative vs advisory state shown")},
		References:     []inspectorReference{{Label: "approvals", Items: detail.PendingApprovalIDs}},
		LocalActions:   []string{"jump:approvals", "jump:artifacts", "jump:audit", "copy:run_id"},
		CopyActions:    runRouteCopyActions(detail),
		ModeTabs:       []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:     string(presentation),
		ContentKind:    contentKind,
		ContentLabel:   "run details",
		Content:        content,
		ViewportWidth:  96,
		ViewportHeight: 14,
	})
}

func pendingApprovalStageCount(stages []brokerapi.RunStageSummary) int {
	count := 0
	for _, stage := range stages {
		if stage.PendingApprovalCount > 0 {
			count++
		}
	}
	return count
}

func waitingRoleCount(roles []brokerapi.RunRoleSummary) int {
	count := 0
	for _, role := range roles {
		if role.WaitReasonCode != "" {
			count++
		}
	}
	return count
}

func runInspectorContent(summary brokerapi.RunSummary, detail *brokerapi.RunDetail, waitingStages int, waitingRoles int, presentation contentPresentationMode) string {
	if presentation == presentationStructured {
		return compactLines(
			fmt.Sprintf("structured run detail: run=%s", summary.RunID),
			fmt.Sprintf("counts: authoritative=%d advisory=%d stages=%d roles=%d pending_approvals=%d", len(detail.AuthoritativeState), len(detail.AdvisoryState), len(detail.StageSummaries), len(detail.RoleSummaries), len(detail.PendingApprovalIDs)),
		)
	}
	if presentation == presentationRaw {
		return compactLines(
			fmt.Sprintf("raw summary run_id=%s lifecycle=%s backend_kind=%s", summary.RunID, summary.LifecycleState, summary.BackendKind),
			fmt.Sprintf("raw coordination blocked=%t wait_reason=%s mode=%s", detail.Coordination.Blocked, valueOrNA(detail.Coordination.WaitReasonCode), valueOrNA(detail.Coordination.CoordinationMode)),
			fmt.Sprintf("raw maps authoritative_keys=%d advisory_keys=%d", len(detail.AuthoritativeState), len(detail.AdvisoryState)),
		)
	}
	return compactLines(
		fmt.Sprintf("backend_kind=%s", summary.BackendKind),
		"Runtime isolation assurance (authoritative): "+renderRuntimeIsolationCue(summary.BackendKind, summary.IsolationAssuranceLevel),
		fmt.Sprintf("Runtime posture degraded (authoritative): %t %s", summary.RuntimePostureDegraded, renderRuntimePostureDegradedBadge(summary.RuntimePostureDegraded)),
		"Provisioning/binding posture (authoritative): "+renderProvisioningPostureCue(summary.ProvisioningPosture),
		"Audit posture (authoritative): "+renderAuditPostureCue(summary.AuditIntegrityStatus, summary.AuditAnchoringStatus, summary.AuditCurrentlyDegraded),
		fmt.Sprintf("Approval profile (authoritative): %s", renderApprovalProfileCue(summary.ApprovalProfile)),
		fmt.Sprintf("Authoritative broker state (control-plane truth): %d keys", len(detail.AuthoritativeState)),
		fmt.Sprintf("Advisory state (non-authoritative runner hints): %d keys %s", len(detail.AdvisoryState), renderAdvisoryStateCue(detail.AdvisoryState)),
		fmt.Sprintf("Coordination summary: blocked=%t wait_reason=%s mode=%s locks=%d conflicts=%d", detail.Coordination.Blocked, detail.Coordination.WaitReasonCode, detail.Coordination.CoordinationMode, detail.Coordination.LockCount, detail.Coordination.ConflictCount),
		fmt.Sprintf("Blocking cue: %s (reason=%s)", renderBlockingStateCue(detail.Coordination.Blocked, detail.Coordination.WaitReasonCode), valueOrNA(detail.Coordination.WaitReasonCode)),
		fmt.Sprintf("Lifecycle blocking reason code: %s -> %s", valueOrNA(summary.BlockingReasonCode), renderBlockingStateCue(summary.BlockingReasonCode != "", summary.BlockingReasonCode)),
		fmt.Sprintf("Stage summaries: %d total, %d with pending approvals", len(detail.StageSummaries), waitingStages),
		fmt.Sprintf("Role summaries: %d total, %d reporting coordination waits", len(detail.RoleSummaries), waitingRoles),
		fmt.Sprintf("Pending approvals=%d active manifests=%d policy refs=%d", len(detail.PendingApprovalIDs), len(detail.ActiveManifestHashes), len(detail.LatestPolicyDecisionRefs)),
	)
}

func runInspectorContentKind(presentation contentPresentationMode) inspectorContentKind {
	if presentation == presentationRaw {
		return inspectorContentRaw
	}
	return inspectorContentStructured
}

func runRouteCopyActions(detail *brokerapi.RunDetail) []routeCopyAction {
	if detail == nil {
		return nil
	}
	summary := detail.Summary
	raw := compactLines(
		fmt.Sprintf("run_id=%s", summary.RunID),
		fmt.Sprintf("workspace_id=%s", summary.WorkspaceID),
		fmt.Sprintf("lifecycle=%s", summary.LifecycleState),
		fmt.Sprintf("backend_kind=%s", summary.BackendKind),
	)
	return compactCopyActions([]routeCopyAction{
		{ID: "run_id", Label: "run id", Text: summary.RunID},
		{ID: "workspace_id", Label: "workspace id", Text: summary.WorkspaceID},
		{ID: "raw_block", Label: "raw block", Text: raw},
	})
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
