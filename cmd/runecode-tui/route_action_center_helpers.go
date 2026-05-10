package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type actionCenterSummary struct {
	State      routeLoadState
	Title      string
	Message    string
	Reason     string
	NextAction string
}

func (m actionCenterRouteModel) familyBuckets() map[actionCenterFamily][]actionCenterItem {
	return m.familyBucketsAt(time.Now().UTC())
}

func (m actionCenterRouteModel) familyBucketsAt(now time.Time) map[actionCenterFamily][]actionCenterItem {
	buckets := map[actionCenterFamily][]actionCenterItem{
		actionCenterFamilyApprovals: buildApprovalActionItems(m.approvals, now),
		actionCenterFamilyOps:       buildOperationalAttentionItems(m.audit, m.auditErr, m.watch, m.watchHealth, m.runs, m.project),
		actionCenterFamilyBlocked:   buildBlockedImpactItems(m.runs, m.approvals),
	}
	for family, items := range buckets {
		sort.SliceStable(items, func(i, j int) bool {
			return actionCenterUrgencyRank(items[i].Urgency) > actionCenterUrgencyRank(items[j].Urgency)
		})
		buckets[family] = items
	}
	return buckets
}

func buildApprovalActionItems(items []brokerapi.ApprovalSummary, now time.Time) []actionCenterItem {
	now = now.UTC()
	out := make([]actionCenterItem, 0, len(items))
	for _, ap := range items {
		approvalID := strings.TrimSpace(ap.ApprovalID)
		if approvalID == "" {
			continue
		}
		expiryCue, urgency := approvalExpiryUrgency(ap.ExpiresAt, now)
		staleCue, urgency := approvalStalenessUrgency(ap, urgency)
		urgency = pendingApprovalUrgency(ap.Status, urgency)
		state := routeLoadStateApprovalRequired
		if urgency == "critical" {
			state = routeLoadStateBlocked
		}
		reasons := []string{fmt.Sprintf("Approval is %s", humanizeExecutionToken(ap.Status))}
		if trigger := strings.TrimSpace(ap.ApprovalTriggerCode); trigger != "" {
			reasons = append(reasons, fmt.Sprintf("triggered by %s", humanizeExecutionToken(trigger)))
		}
		if staleCue != "fresh" {
			reasons = append(reasons, humanizeExecutionToken(staleCue))
		}
		if expiryCue != "no_expiry" {
			reasons = append(reasons, humanizeExecutionToken(expiryCue))
		}
		impact := fmt.Sprintf("Workflow progress stays gated for run %s stage %s until this decision is resolved.", valueOrNA(ap.BoundScope.RunID), valueOrNA(ap.BoundScope.StageID))
		targetLabel := fmt.Sprintf("Approvals › %s", approvalID)
		evidence := fmt.Sprintf("source=approval_summary approval_id=%s run=%s", approvalID, valueOrNA(ap.BoundScope.RunID))
		out = append(out, actionCenterItem{
			Title:          fmt.Sprintf("approval %s", approvalID),
			State:          state,
			Urgency:        urgency,
			Reason:         strings.Join(reasons, "; "),
			Impact:         impact,
			Owner:          "owner: operator decision",
			RequiredAction: "Open Approvals, review the exact gated action and evidence, then approve or reject.",
			TargetLabel:    targetLabel,
			EvidenceCue:    evidence,
			Target:         paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: approvalID},
		})
	}
	if len(out) == 0 {
		return []actionCenterItem{{Title: "no pending approvals", State: routeLoadStateEmpty, Urgency: "low", Reason: "The broker approval queue is currently empty.", Impact: "No workflow work is waiting on approval decisions from the current broker surfaces.", Owner: "owner: none", RequiredAction: "No approval follow-up is needed right now.", TargetLabel: "Approvals", EvidenceCue: "source=approval_list empty"}}
	}
	return out
}

func approvalExpiryUrgency(expiresAt string, now time.Time) (string, string) {
	expiryCue := "no_expiry"
	urgency := "normal"
	ts := parseTimestamp(expiresAt)
	if ts.IsZero() {
		return expiryCue, urgency
	}
	switch {
	case ts.Before(now):
		return "expired", "critical"
	case ts.Before(now.Add(actionCenterExpirySoonWindow)):
		return "expiring_soon", "high"
	default:
		return "expires_later", urgency
	}
}

func approvalStalenessUrgency(ap brokerapi.ApprovalSummary, urgency string) (string, string) {
	statusLower := strings.ToLower(strings.TrimSpace(ap.Status))
	if strings.TrimSpace(ap.SupersededByApprovalID) != "" || strings.Contains(statusLower, "supersed") {
		return "superseded", lowerUrgencyUnlessCritical(urgency, "low")
	}
	if strings.Contains(statusLower, "stale") {
		return "stale", lowerUrgencyUnlessCritical(urgency, "medium")
	}
	return "fresh", urgency
}

func pendingApprovalUrgency(status string, urgency string) string {
	statusLower := strings.ToLower(strings.TrimSpace(status))
	if strings.Contains(statusLower, "pending") || strings.Contains(statusLower, "requested") {
		if urgency == "normal" {
			return "high"
		}
	}
	return urgency
}

func lowerUrgencyUnlessCritical(current string, next string) string {
	if current == "critical" {
		return current
	}
	return next
}

func buildOperationalAttentionItems(audit *brokerapi.AuditVerificationGetResponse, auditErr string, watch dashboardLiveActivity, health shellSyncHealth, runs []brokerapi.RunSummary, project brokerapi.ProjectSubstratePostureGetResponse) []actionCenterItem {
	items := []actionCenterItem{}
	items = append(items, operationalSyncHealthItem(health)...)
	items = append(items, operationalWatchFamilyItems(watch)...)
	items = append(items, operationalAuditItems(audit, auditErr)...)
	items = append(items, operationalRunItems(runs)...)
	items = append(items, operationalProjectSetupItems(project)...)
	if len(items) == 0 {
		return []actionCenterItem{{Title: "no active operational attention", State: routeLoadStateReady, Urgency: "low", Reason: "Audit evidence, watch sync, runtime posture, and setup posture are nominal on current broker surfaces.", Impact: "No degraded operational follow-up is visible here.", Owner: "owner: none", RequiredAction: "No operational remediation is needed right now.", TargetLabel: "Action Center", EvidenceCue: "source=operational_attention empty"}}
	}
	return items
}

func operationalSyncHealthItem(health shellSyncHealth) []actionCenterItem {
	if health.State != shellSyncStateDisconnected && health.State != shellSyncStateDegraded && health.State != shellSyncStateReconnecting {
		return nil
	}
	urgency := "high"
	if health.State == shellSyncStateDisconnected {
		urgency = "critical"
	}
	return []actionCenterItem{{
		Title:          "shell watch sync health",
		State:          routeLoadStateDegraded,
		Urgency:        urgency,
		Reason:         fmt.Sprintf("Watch sync is %s.", humanizeExecutionToken(string(health.State))),
		Impact:         "Live operator follow-up coverage is degraded until shell watch sync recovers.",
		Owner:          "owner: operator checks broker connectivity",
		RequiredAction: "Open Status to confirm broker connectivity and sync posture before relying on live updates.",
		TargetLabel:    "Status",
		EvidenceCue:    fmt.Sprintf("source=shell_sync_health state=%s", health.State),
		Target:         paletteTarget{Kind: "route", RouteID: routeStatus},
	}}
}

func operationalWatchFamilyItems(watch dashboardLiveActivity) []actionCenterItem {
	items := make([]actionCenterItem, 0, 3)
	for _, family := range []watchFamilySummary{watch.runWatch, watch.approvalWatch, watch.sessionWatch} {
		if family.errorCount == 0 && strings.EqualFold(strings.TrimSpace(family.lastStatus), "ok") {
			continue
		}
		urgency := "medium"
		if family.errorCount > 0 {
			urgency = "high"
		}
		items = append(items, actionCenterItem{
			Title:          fmt.Sprintf("watch family %s", valueOrNA(family.family)),
			State:          routeLoadStateDegraded,
			Urgency:        urgency,
			Reason:         fmt.Sprintf("%s reported %d error(s); last status %s.", humanizeExecutionToken(family.family), family.errorCount, humanizeExecutionToken(family.lastStatus)),
			Impact:         "Operator follow-up cues from this watch family may be incomplete or stale.",
			Owner:          "owner: operator verifies watch health",
			RequiredAction: "Use Dashboard live activity and Status to confirm whether watch degradation is transient or ongoing.",
			TargetLabel:    "Dashboard",
			EvidenceCue:    fmt.Sprintf("source=watch_family family=%s", valueOrNA(family.family)),
			Target:         paletteTarget{Kind: "route", RouteID: routeDashboard},
		})
	}
	return items
}

func operationalAuditItems(audit *brokerapi.AuditVerificationGetResponse, auditErr string) []actionCenterItem {
	if strings.TrimSpace(auditErr) != "" {
		return []actionCenterItem{{
			Title:          "audit verification unavailable",
			State:          routeLoadStateDegraded,
			Urgency:        "high",
			Reason:         fmt.Sprintf("Audit verification could not be loaded: %s", auditErr),
			Impact:         "Evidence posture is shown with degraded fallback until broker verification becomes available again.",
			Owner:          "owner: operator verifies evidence health",
			RequiredAction: "Open Audit to confirm the fallback posture and verify whether the broker audit surface has recovered.",
			TargetLabel:    "Audit",
			EvidenceCue:    "source=audit_verification fallback",
			Target:         paletteTarget{Kind: "route", RouteID: routeAudit},
		}}
	}
	if audit == nil {
		return nil
	}
	s := audit.Summary
	if !s.CurrentlyDegraded && !strings.EqualFold(strings.TrimSpace(s.AnchoringStatus), "degraded") && !strings.EqualFold(strings.TrimSpace(s.IntegrityStatus), "failed") {
		return nil
	}
	urgency := "high"
	if strings.EqualFold(strings.TrimSpace(s.IntegrityStatus), "failed") {
		urgency = "critical"
	}
	return []actionCenterItem{{
		Title:          "audit verification posture",
		State:          routeLoadStateDegraded,
		Urgency:        urgency,
		Reason:         fmt.Sprintf("Integrity is %s, receipts are %s, and evidence needs review.", humanizeExecutionToken(s.IntegrityStatus), humanizeExecutionToken(s.AnchoringStatus)),
		Impact:         fmt.Sprintf("Evidence confidence is reduced; %d hard failure(s) and %d degraded reason(s) are available in Audit.", len(s.HardFailures), len(s.DegradedReasons)),
		Owner:          "owner: operator reviews evidence posture",
		RequiredAction: "Open Audit to inspect the degraded or failed evidence details before treating the trail as healthy.",
		TargetLabel:    "Audit",
		EvidenceCue:    fmt.Sprintf("source=audit_verification summary findings=%d", s.FindingCount),
		Target:         paletteTarget{Kind: "route", RouteID: routeAudit},
	}}
}

func operationalRunItems(runs []brokerapi.RunSummary) []actionCenterItem {
	items := make([]actionCenterItem, 0, len(runs))
	for _, run := range runs {
		state := strings.ToLower(strings.TrimSpace(run.LifecycleState))
		failedLifecycle := strings.Contains(state, "fail") || strings.Contains(state, "error")
		degradedLifecycle := strings.Contains(state, "degrad")
		if !run.RuntimePostureDegraded && !run.AuditCurrentlyDegraded && !failedLifecycle && !degradedLifecycle {
			continue
		}
		urgency := "high"
		if failedLifecycle {
			urgency = "critical"
		}
		reasonParts := []string{fmt.Sprintf("Lifecycle is %s", humanizeExecutionToken(run.LifecycleState))}
		if run.RuntimePostureDegraded {
			reasonParts = append(reasonParts, "runtime posture needs review")
		}
		if run.AuditCurrentlyDegraded {
			reasonParts = append(reasonParts, "audit posture needs review")
		}
		items = append(items, actionCenterItem{
			Title:          fmt.Sprintf("run %s operational posture", valueOrNA(run.RunID)),
			State:          routeLoadStateDegraded,
			Urgency:        urgency,
			Reason:         strings.Join(reasonParts, "; ") + ".",
			Impact:         fmt.Sprintf("Execution confidence is reduced for run %s. Runtime proof remains available in Runs.", valueOrNA(run.RunID)),
			Owner:          "owner: operator reviews run posture",
			RequiredAction: "Open Runs to inspect the degraded run posture and confirm linked evidence before continuing.",
			TargetLabel:    fmt.Sprintf("Runs › %s", valueOrNA(run.RunID)),
			EvidenceCue:    fmt.Sprintf("source=run_summary run_id=%s", valueOrNA(run.RunID)),
			Target:         paletteTarget{Kind: "run", RouteID: routeRuns, RunID: run.RunID},
		})
	}
	return items
}

func operationalProjectSetupItems(project brokerapi.ProjectSubstratePostureGetResponse) []actionCenterItem {
	if strings.TrimSpace(project.PostureSummary.SchemaID) == "" || !dashboardProjectSubstrateNeedsAttention(project) {
		return nil
	}
	state := routeLoadStateApprovalRequired
	urgency := "medium"
	if dashboardProjectSubstrateBlocked(project) {
		state = routeLoadStateBlocked
		urgency = "high"
	}
	return []actionCenterItem{{
		Title:          "project setup remediation",
		State:          state,
		Urgency:        urgency,
		Reason:         dashboardProjectSubstrateReason(project),
		Impact:         "Normal product work may stay blocked or require operator remediation until setup posture is brought back into supported shape.",
		Owner:          "owner: operator remediates project setup",
		RequiredAction: "Open Status for broker-owned setup guidance, preview/apply availability, and post-remediation validation.",
		TargetLabel:    "Status",
		EvidenceCue:    fmt.Sprintf("source=project_substrate_posture compatibility=%s", valueOrNA(project.PostureSummary.CompatibilityPosture)),
		Target:         paletteTarget{Kind: "route", RouteID: routeStatus},
	}}
}

func buildBlockedImpactItems(runs []brokerapi.RunSummary, approvals []brokerapi.ApprovalSummary) []actionCenterItem {
	items := make([]actionCenterItem, 0, len(runs))
	byRun := approvalCountsByRun(approvals)
	for _, run := range runs {
		if !runHasBlockedImpact(run) {
			continue
		}
		items = append(items, blockedImpactItem(run, linkedApprovalCount(run, byRun)))
	}
	if len(items) == 0 {
		return []actionCenterItem{{Title: "no blocked work impact", State: routeLoadStateReady, Urgency: "low", Reason: "No run currently reports blocking posture.", Impact: "No workflow work is blocked on the current broker surfaces.", Owner: "owner: none", RequiredAction: "No blocked-work follow-up is needed right now.", TargetLabel: "Runs", EvidenceCue: "source=run_list no_blocked_items"}}
	}
	return items
}

func (m *actionCenterRouteModel) normalizeSelection() {
	buckets := m.snapshot().Families
	for _, family := range []actionCenterFamily{actionCenterFamilyApprovals, actionCenterFamilyOps, actionCenterFamilyBlocked} {
		max := len(buckets[family])
		if max <= 0 {
			m.selected[family] = 0
			continue
		}
		if m.selected[family] < 0 {
			m.selected[family] = 0
		}
		if m.selected[family] >= max {
			m.selected[family] = max - 1
		}
	}
}

func (m actionCenterRouteModel) moveSelection(delta int) {
	buckets := m.snapshot().Families
	items := buckets[m.family]
	if len(items) == 0 {
		m.selected[m.family] = 0
		return
	}
	if delta > 0 {
		m.selected[m.family] = (m.selected[m.family] + 1) % len(items)
		return
	}
	m.selected[m.family]--
	if m.selected[m.family] < 0 {
		m.selected[m.family] = len(items) - 1
	}
}

func (m actionCenterRouteModel) selectedItem() (actionCenterItem, bool) {
	buckets := m.snapshot().Families
	items := buckets[m.family]
	if len(items) == 0 {
		return actionCenterItem{}, false
	}
	idx := m.selectedIndex(m.family, len(items))
	return items[idx], true
}

func (m actionCenterRouteModel) selectedIndex(family actionCenterFamily, count int) int {
	if count <= 0 {
		return 0
	}
	idx := m.selected[family]
	if idx < 0 {
		return 0
	}
	if idx >= count {
		return count - 1
	}
	return idx
}

func (m actionCenterRouteModel) nextFamily() actionCenterFamily {
	order := []actionCenterFamily{actionCenterFamilyApprovals, actionCenterFamilyOps, actionCenterFamilyBlocked}
	for i, family := range order {
		if family == m.family {
			return order[(i+1)%len(order)]
		}
	}
	return actionCenterFamilyApprovals
}

func (m actionCenterRouteModel) prevFamily() actionCenterFamily {
	order := []actionCenterFamily{actionCenterFamilyApprovals, actionCenterFamilyOps, actionCenterFamilyBlocked}
	for i, family := range order {
		if family == m.family {
			if i == 0 {
				return order[len(order)-1]
			}
			return order[i-1]
		}
	}
	return actionCenterFamilyApprovals
}

func actionCenterUrgencyRank(urgency string) int {
	switch strings.ToLower(strings.TrimSpace(urgency)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "normal":
		return 1
	default:
		return 0
	}
}
