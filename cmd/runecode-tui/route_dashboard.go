package main

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type dashboardLoadedMsg struct {
	data dashboardData
	err  error
	seq  uint64
}

type dashboardData struct {
	readiness brokerapi.BrokerReadiness
	version   brokerapi.BrokerVersionInfo
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
	return dashboardRouteModel{def: def, client: client}
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
		m.data = typed.data
		m.errText = ""
		return m, nil
	default:
		return m, nil
	}
}

func (m dashboardRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return "Loading dashboard from broker API..."
	}
	if m.errText != "" {
		return compactLines(
			"Dashboard overview",
			"Load failed: "+m.errText,
			"Press r to retry.",
		)
	}
	focusLabel := "inactive"
	if focus == focusContent {
		focusLabel = "active"
	}
	primaryRun := primaryDashboardRun(m.data.runs)
	return compactLines(
		sectionTitle("Dashboard")+" "+focusBadge(focus)+" "+navStateBadge(focusLabel == "active"),
		tableHeader("Now")+" "+renderDashboardNowBar(primaryRun, len(m.data.approvals), focusLabel == "active"),
		"",
		tableHeader("Safety Summary"),
		renderRunSafetyStrip(primaryDashboardRun(m.data.runs)),
		renderDashboardSafetyAlerts(m.data),
		"",
		tableHeader("Control Plane"),
		tableHeader("Readiness")+" "+boolBadge("ready", m.data.readiness.Ready)+" "+boolBadge("local_only", m.data.readiness.LocalOnly)+" "+boolBadge("recovery_complete", m.data.readiness.RecoveryComplete),
		tableHeader("Safety posture")+" "+stateBadgeWithLabel("integrity", m.data.audit.Summary.IntegrityStatus)+" "+stateBadgeWithLabel("anchoring", m.data.audit.Summary.AnchoringStatus)+" "+boolBadge("degraded", m.data.audit.Summary.CurrentlyDegraded),
		renderDashboardAuditFallbackNotice(m.data.auditErr),
		fmt.Sprintf("Workflow posture: runs=%d pending_approvals=%d", len(m.data.runs), pendingApprovalCount(m.data.runs, m.data.approvals)),
		fmt.Sprintf("Version: %s (%s) protocol bundle=%s", m.data.version.ProductVersion, m.data.version.BuildRevision, m.data.version.ProtocolBundleVersion),
		"",
		tableHeader("Live Activity"),
		"Live activity (typed watch families; logs are supplemental inspection only):",
		muted("Live activity uses semantic watch families with explicit event types."),
		renderWatchFamilySummary(m.data.live.runWatch),
		renderWatchFamilySummary(m.data.live.approvalWatch),
		renderWatchFamilySummary(m.data.live.sessionWatch),
		"",
		tableHeader("Highlights"),
		renderRunHighlights(m.data.runs),
		renderApprovalHighlights(m.data.approvals),
		"",
		tableHeader("Actions")+" "+keyHint("r reload")+" "+muted("tab moves focus • : opens command surface"),
		keyHint("Route keys: r reload"),
	)
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
		live := loadLiveActivity(ctx, m.client)
		return dashboardLoadedMsg{data: dashboardData{readiness: readinessResp.Readiness, version: versionResp.VersionInfo, runs: runResp.Runs, approvals: approvalResp.Approvals, audit: auditResp, auditErr: auditErr, live: live}, seq: seq}
	}
}

func degradedDashboardAuditFallback() brokerapi.AuditVerificationGetResponse {
	return brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{
		CryptographicallyValid: false,
		HistoricallyAdmissible: false,
		CurrentlyDegraded:      true,
		IntegrityStatus:        "failed",
		AnchoringStatus:        "failed",
		StoragePostureStatus:   "failed",
		SegmentLifecycleStatus: "failed",
		HardFailures:           []string{"audit_surface_unavailable"},
	}}
}

func renderDashboardAuditFallbackNotice(auditErr string) string {
	if strings.TrimSpace(auditErr) == "" {
		return ""
	}
	return dangerBadge("AUDIT_VERIFICATION_UNAVAILABLE") + " audit verification unavailable; showing degraded fallback posture (" + auditErr + ")"
}

func renderDashboardNowBar(run brokerapi.RunSummary, approvalCount int, focusActive bool) string {
	parts := []string{}
	if focusActive {
		parts = append(parts, successBadge("CONTENT_READY"))
	} else {
		parts = append(parts, neutralBadge("CONTENT_IDLE"))
	}
	if strings.TrimSpace(run.RunID) != "" {
		parts = append(parts, fmt.Sprintf("run=%s", run.RunID))
		parts = append(parts, stateBadgeWithLabel("state", run.LifecycleState))
		parts = append(parts, stateBadgeWithLabel("backend", run.BackendKind))
	}
	if approvalCount > 0 {
		parts = append(parts, approvalRequiredBadge(fmt.Sprintf("PENDING_APPROVALS=%d", approvalCount)))
	} else {
		parts = append(parts, successBadge("PENDING_APPROVALS=0"))
	}
	return strings.Join(parts, " ")
}

func loadLiveActivity(ctx context.Context, client localBrokerClient) dashboardLiveActivity {
	runSummary := watchFamilySummary{family: "run_watch", lastStatus: "ok"}
	runEvents, err := client.RunWatch(ctx, brokerapi.RunWatchRequest{StreamID: newRequestID("run-watch-stream"), IncludeSnapshot: true, Follow: false})
	if err != nil {
		runSummary.lastStatus = "watch_error"
		runSummary.lastSubject = "ipc_watch_error"
	} else {
		runSummary = summarizeRunWatchEvents(runEvents)
	}

	approvalSummary := watchFamilySummary{family: "approval_watch", lastStatus: "ok"}
	approvalEvents, err := client.ApprovalWatch(ctx, brokerapi.ApprovalWatchRequest{StreamID: newRequestID("approval-watch-stream"), IncludeSnapshot: true, Follow: false})
	if err != nil {
		approvalSummary.lastStatus = "watch_error"
		approvalSummary.lastSubject = "ipc_watch_error"
	} else {
		approvalSummary = summarizeApprovalWatchEvents(approvalEvents)
	}

	sessionSummary := watchFamilySummary{family: "session_watch", lastStatus: "ok"}
	sessionEvents, err := client.SessionWatch(ctx, brokerapi.SessionWatchRequest{StreamID: newRequestID("session-watch-stream"), IncludeSnapshot: true, Follow: false})
	if err != nil {
		sessionSummary.lastStatus = "watch_error"
		sessionSummary.lastSubject = "ipc_watch_error"
	} else {
		sessionSummary = summarizeSessionWatchEvents(sessionEvents)
	}

	return dashboardLiveActivity{runWatch: runSummary, approvalWatch: approvalSummary, sessionWatch: sessionSummary}
}

func summarizeRunWatchEvents(events []brokerapi.RunWatchEvent) watchFamilySummary {
	s := watchFamilySummary{family: "run_watch", eventCount: len(events), lastStatus: "ok"}
	for _, event := range events {
		s.lastEventType = event.EventType
		if event.Run != nil {
			s.lastSubject = event.Run.RunID
		}
		if event.Terminal {
			s.terminalCount++
			s.lastStatus = event.TerminalStatus
		}
		if event.Error != nil {
			s.errorCount++
			s.lastStatus = "error"
		}
		switch event.EventType {
		case "run_watch_snapshot":
			s.snapshotCount++
		case "run_watch_upsert":
			s.upsertCount++
		}
	}
	return s
}

func summarizeApprovalWatchEvents(events []brokerapi.ApprovalWatchEvent) watchFamilySummary {
	s := watchFamilySummary{family: "approval_watch", eventCount: len(events), lastStatus: "ok"}
	for _, event := range events {
		s.lastEventType = event.EventType
		if event.Approval != nil {
			s.lastSubject = event.Approval.ApprovalID
		}
		if event.Terminal {
			s.terminalCount++
			s.lastStatus = event.TerminalStatus
		}
		if event.Error != nil {
			s.errorCount++
			s.lastStatus = "error"
		}
		switch event.EventType {
		case "approval_watch_snapshot":
			s.snapshotCount++
		case "approval_watch_upsert":
			s.upsertCount++
		}
	}
	return s
}

func summarizeSessionWatchEvents(events []brokerapi.SessionWatchEvent) watchFamilySummary {
	s := watchFamilySummary{family: "session_watch", eventCount: len(events), lastStatus: "ok"}
	for _, event := range events {
		s.lastEventType = event.EventType
		if event.Session != nil {
			s.lastSubject = event.Session.Identity.SessionID
		}
		if event.Terminal {
			s.terminalCount++
			s.lastStatus = event.TerminalStatus
		}
		if event.Error != nil {
			s.errorCount++
			s.lastStatus = "error"
		}
		switch event.EventType {
		case "session_watch_snapshot":
			s.snapshotCount++
		case "session_watch_upsert":
			s.upsertCount++
		}
	}
	return s
}

func renderWatchFamilySummary(summary watchFamilySummary) string {
	return fmt.Sprintf(
		"  %s\n    totals events=%d snapshot=%d upsert=%d terminal=%d errors=%d\n    last_event=%s subject=%s status=%s",
		infoBadge(summary.family),
		summary.eventCount,
		summary.snapshotCount,
		summary.upsertCount,
		summary.terminalCount,
		summary.errorCount,
		valueOrNA(summary.lastEventType),
		valueOrNA(summary.lastSubject),
		valueOrNA(summary.lastStatus),
	)
}

func pendingApprovalCount(runs []brokerapi.RunSummary, approvals []brokerapi.ApprovalSummary) int {
	total := 0
	for _, run := range runs {
		total += run.PendingApprovalCount
	}
	if total > 0 {
		return total
	}
	return len(approvals)
}

func renderRunHighlights(runs []brokerapi.RunSummary) string {
	if len(runs) == 0 {
		return "  No runs returned"
	}
	first := runs[0]
	return fmt.Sprintf("  Latest run %s %s backend=%s isolation=%s approvals=%d", first.RunID, stateBadgeWithLabel("state", first.LifecycleState), first.BackendKind, first.IsolationAssuranceLevel, first.PendingApprovalCount)
}

func renderApprovalHighlights(approvals []brokerapi.ApprovalSummary) string {
	if len(approvals) == 0 {
		return "  No pending approvals"
	}
	first := approvals[0]
	return fmt.Sprintf("  Approval %s %s trigger=%s run=%s", first.ApprovalID, stateBadgeWithLabel("status", first.Status), first.ApprovalTriggerCode, valueOrNA(first.BoundScope.RunID))
}

func primaryDashboardRun(runs []brokerapi.RunSummary) brokerapi.RunSummary {
	if len(runs) == 0 {
		return brokerapi.RunSummary{}
	}
	return runs[0]
}

func renderDashboardSafetyAlerts(data dashboardData) string {
	alerts := []string{}
	run := primaryDashboardRun(data.runs)
	if strings.ToLower(strings.TrimSpace(run.ProvisioningPosture)) == "tofu" {
		alerts = append(alerts, provisioningDegradedBadge("ALERT_TOFU_PROVISIONING")+" TOFU isolate key provisioning in effect")
	}
	if strings.ToLower(strings.TrimSpace(run.IsolationAssuranceLevel)) == "unknown" || strings.ToLower(strings.TrimSpace(run.IsolationAssuranceLevel)) == "unavailable" {
		alerts = append(alerts, dangerBadge("ALERT_RUNTIME_POSTURE_UNAVAILABLE")+" authoritative runtime isolation posture degraded/unavailable")
	}
	if strings.ToLower(strings.TrimSpace(data.audit.Summary.AnchoringStatus)) == "degraded" || data.audit.Summary.CurrentlyDegraded {
		alerts = append(alerts, auditDegradedBadge("ALERT_AUDIT_UNANCHORED")+" audit posture unanchored/degraded")
	}
	if len(alerts) == 0 {
		return "Safety alerts: " + successBadge("NO_ACTIVE_DEGRADATION")
	}
	return "Safety alerts: " + strings.Join(alerts, " ")
}
