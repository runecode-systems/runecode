package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func (m actionCenterRouteModel) familyBuckets() map[actionCenterFamily][]actionCenterItem {
	buckets := map[actionCenterFamily][]actionCenterItem{
		actionCenterFamilyApprovals: buildApprovalActionItems(m.approvals),
		actionCenterFamilyOps:       buildOperationalAttentionItems(m.audit, m.watch, m.watchHealth, m.runs),
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

func buildApprovalActionItems(items []brokerapi.ApprovalSummary) []actionCenterItem {
	now := time.Now().UTC()
	out := make([]actionCenterItem, 0, len(items))
	for _, ap := range items {
		approvalID := strings.TrimSpace(ap.ApprovalID)
		if approvalID == "" {
			continue
		}
		expiryCue, urgency := approvalExpiryUrgency(ap.ExpiresAt, now)
		staleCue, urgency := approvalStalenessUrgency(ap, urgency)
		urgency = pendingApprovalUrgency(ap.Status, urgency)
		impact := fmt.Sprintf("run=%s stage=%s action=%s", valueOrNA(ap.BoundScope.RunID), valueOrNA(ap.BoundScope.StageID), valueOrNA(ap.BoundScope.ActionKind))
		out = append(out, actionCenterItem{
			Title:     fmt.Sprintf("approval %s", approvalID),
			Detail:    fmt.Sprintf("status=%s trigger=%s", valueOrNA(ap.Status), valueOrNA(ap.ApprovalTriggerCode)),
			Urgency:   urgency,
			ExpiryCue: expiryCue,
			StaleCue:  staleCue,
			Impact:    impact,
			Target:    paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: approvalID},
		})
	}
	if len(out) == 0 {
		return []actionCenterItem{{Title: "no pending approvals", Detail: "canonical approval queue is currently empty", Urgency: "low", ExpiryCue: "n/a", StaleCue: "n/a", Impact: "none"}}
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

func buildOperationalAttentionItems(audit *brokerapi.AuditVerificationGetResponse, watch dashboardLiveActivity, health shellSyncHealth, runs []brokerapi.RunSummary) []actionCenterItem {
	items := []actionCenterItem{}
	items = append(items, operationalSyncHealthItem(health)...)
	items = append(items, operationalWatchFamilyItems(watch)...)
	items = append(items, operationalAuditItems(audit)...)
	items = append(items, operationalRunItems(runs)...)
	if len(items) == 0 {
		return []actionCenterItem{{Title: "no active operational attention", Detail: "audit anchoring, watch sync, and run posture all nominal", Urgency: "low", ExpiryCue: "n/a", StaleCue: "n/a", Impact: "none"}}
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
		Title:     "shell watch sync health",
		Detail:    fmt.Sprintf("state=%s error=%s", health.State, defaultPlaceholder(health.ErrorText, "n/a")),
		Urgency:   urgency,
		ExpiryCue: "n/a",
		StaleCue:  "n/a",
		Impact:    "live activity coverage degraded",
		Target:    paletteTarget{Kind: "route", RouteID: routeStatus},
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
			Title:     fmt.Sprintf("watch family %s", valueOrNA(family.family)),
			Detail:    fmt.Sprintf("errors=%d last_status=%s last_subject=%s", family.errorCount, valueOrNA(family.lastStatus), valueOrNA(family.lastSubject)),
			Urgency:   urgency,
			ExpiryCue: "n/a",
			StaleCue:  "n/a",
			Impact:    "operator follow stream may be degraded",
			Target:    paletteTarget{Kind: "route", RouteID: routeDashboard},
		})
	}
	return items
}

func operationalAuditItems(audit *brokerapi.AuditVerificationGetResponse) []actionCenterItem {
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
		Title:     "audit verification posture",
		Detail:    fmt.Sprintf("integrity=%s anchoring=%s degraded=%t", valueOrNA(s.IntegrityStatus), valueOrNA(s.AnchoringStatus), s.CurrentlyDegraded),
		Urgency:   urgency,
		ExpiryCue: "n/a",
		StaleCue:  "n/a",
		Impact:    fmt.Sprintf("hard_failures=%d degraded_reasons=%d", len(s.HardFailures), len(s.DegradedReasons)),
		Target:    paletteTarget{Kind: "route", RouteID: routeAudit},
	}}
}

func operationalRunItems(runs []brokerapi.RunSummary) []actionCenterItem {
	items := make([]actionCenterItem, 0, len(runs))
	for _, run := range runs {
		if !run.RuntimePostureDegraded && !run.AuditCurrentlyDegraded {
			continue
		}
		items = append(items, actionCenterItem{
			Title:     fmt.Sprintf("run %s operational posture", valueOrNA(run.RunID)),
			Detail:    fmt.Sprintf("runtime_degraded=%t audit_degraded=%t", run.RuntimePostureDegraded, run.AuditCurrentlyDegraded),
			Urgency:   "high",
			ExpiryCue: "n/a",
			StaleCue:  "n/a",
			Impact:    fmt.Sprintf("backend=%s isolation=%s", valueOrNA(run.BackendKind), valueOrNA(run.IsolationAssuranceLevel)),
			Target:    paletteTarget{Kind: "run", RouteID: routeRuns, RunID: run.RunID},
		})
	}
	return items
}

func buildBlockedImpactItems(runs []brokerapi.RunSummary, approvals []brokerapi.ApprovalSummary) []actionCenterItem {
	items := make([]actionCenterItem, 0, len(runs))
	byRun := map[string]int{}
	for _, ap := range approvals {
		runID := strings.TrimSpace(ap.BoundScope.RunID)
		if runID != "" {
			byRun[runID]++
		}
	}
	for _, run := range runs {
		state := strings.ToLower(strings.TrimSpace(run.LifecycleState))
		isBlocked := strings.Contains(state, "block") || strings.Contains(state, "wait") || run.PendingApprovalCount > 0 || strings.TrimSpace(run.BlockingReasonCode) != ""
		if !isBlocked {
			continue
		}
		urgency := "medium"
		if run.PendingApprovalCount > 0 {
			urgency = "high"
		}
		blockedCount := run.PendingApprovalCount
		if blockedCount == 0 {
			blockedCount = byRun[run.RunID]
		}
		items = append(items, actionCenterItem{
			Title:     fmt.Sprintf("run %s blocked impact", valueOrNA(run.RunID)),
			Detail:    fmt.Sprintf("lifecycle=%s reason=%s", valueOrNA(run.LifecycleState), valueOrNA(run.BlockingReasonCode)),
			Urgency:   urgency,
			ExpiryCue: "n/a",
			StaleCue:  "n/a",
			Impact:    fmt.Sprintf("pending_approvals=%d linked_queue_items=%d", run.PendingApprovalCount, blockedCount),
			Target:    paletteTarget{Kind: "run", RouteID: routeRuns, RunID: run.RunID},
		})
	}
	if len(items) == 0 {
		return []actionCenterItem{{Title: "no blocked work impact", Detail: "no run currently reports blocking posture", Urgency: "low", ExpiryCue: "n/a", StaleCue: "n/a", Impact: "none"}}
	}
	return items
}

func renderActionCenterItems(items []actionCenterItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("%s | %s | urgency=%s expiry=%s stale/superseded=%s | impact=%s", item.Title, item.Detail, valueOrNA(item.Urgency), valueOrNA(item.ExpiryCue), valueOrNA(item.StaleCue), valueOrNA(item.Impact)))
	}
	return out
}

func renderActionCenterInspector(family actionCenterFamily, items []actionCenterItem, selected int) string {
	if len(items) == 0 || selected < 0 || selected >= len(items) {
		return renderInspectorHeader("Action Center inspector", appTheme.InspectorHint.Render("triage detail")) + "\n  No item selected."
	}
	item := items[selected]
	return compactLines(
		renderInspectorHeader("Action Center inspector", appTheme.InspectorHint.Render("triage detail")),
		fmt.Sprintf("  family=%s", family),
		fmt.Sprintf("  title=%s", item.Title),
		fmt.Sprintf("  urgency=%s", valueOrNA(item.Urgency)),
		fmt.Sprintf("  expiry=%s", valueOrNA(item.ExpiryCue)),
		fmt.Sprintf("  stale_or_superseded=%s", valueOrNA(item.StaleCue)),
		fmt.Sprintf("  impact=%s", valueOrNA(item.Impact)),
		fmt.Sprintf("  drill_down_target=%s/%s", valueOrNA(item.Target.Kind), valueOrNA(string(item.Target.RouteID))),
	)
}

func (m *actionCenterRouteModel) normalizeSelection() {
	buckets := m.familyBuckets()
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
	buckets := m.familyBuckets()
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
	buckets := m.familyBuckets()
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
