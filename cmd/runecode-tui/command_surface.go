package main

import (
	"fmt"
	"strings"
)

type navigationVerb string

const (
	verbOpen    navigationVerb = "open"
	verbInspect navigationVerb = "inspect"
	verbJump    navigationVerb = "jump"
	verbBack    navigationVerb = "back"
)

type paletteTarget struct {
	Kind       string
	RouteID    routeID
	SessionID  string
	RunID      string
	ApprovalID string
	Digest     string
	CommandID  string
}

type paletteActionMsg struct {
	Verb   navigationVerb
	Target paletteTarget
}

type paletteEntry struct {
	Index       int
	Label       string
	Description string
	Search      string
	Action      paletteActionMsg
}

func (m shellModel) buildPaletteEntries() []paletteEntry {
	entries := make([]paletteEntry, 0, 64)
	idx := 1
	add := func(label, description, search string, action paletteActionMsg) {
		entries = append(entries, paletteEntry{Index: idx, Label: label, Description: description, Search: search, Action: action})
		idx++
	}

	add("back", "Back to previous location", "back jump previous route", paletteActionMsg{Verb: verbBack})
	m.appendRoutePaletteEntries(add)
	m.appendSessionPaletteEntries(add)
	m.appendRunPaletteEntries(add)
	m.appendApprovalPaletteEntries(add)
	m.appendActionCenterPaletteEntries(add)
	m.appendArtifactPaletteEntries(add)
	m.appendAuditPaletteEntries(add)
	m.appendCommandPaletteEntries(add)

	return entries
}

func (m shellModel) appendRoutePaletteEntries(add func(string, string, string, paletteActionMsg)) {
	for _, r := range m.routes {
		add(
			fmt.Sprintf("jump route %s", strings.ToLower(r.Label)),
			r.Description,
			fmt.Sprintf("jump route %s %s", strings.ToLower(r.Label), strings.ToLower(string(r.ID))),
			paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: r.ID}},
		)
	}
}

func (m shellModel) appendSessionPaletteEntries(add func(string, string, string, paletteActionMsg)) {
	for _, s := range m.sessionItems {
		sid := strings.TrimSpace(s.Identity.SessionID)
		if sid == "" {
			continue
		}
		add(
			"open session "+sid,
			fmt.Sprintf("ws=%s activity=%s/%s", s.Identity.WorkspaceID, defaultPlaceholder(s.LastActivityAt, "n/a"), defaultPlaceholder(s.LastActivityKind, "n/a")),
			fmt.Sprintf("open session %s %s %s", sid, s.Identity.WorkspaceID, truncateText(s.LastActivityPreview, 64)),
			paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "session", SessionID: sid}},
		)
	}
}

func (m shellModel) appendRunPaletteEntries(add func(string, string, string, paletteActionMsg)) {
	runsModel, ok := m.routeModels[routeRuns].(runsRouteModel)
	if !ok {
		return
	}
	for _, run := range runsModel.runs {
		runID := strings.TrimSpace(run.RunID)
		if runID == "" {
			continue
		}
		add(
			"inspect run "+runID,
			fmt.Sprintf("state=%s approvals=%d", run.LifecycleState, run.PendingApprovalCount),
			fmt.Sprintf("inspect run %s %s", runID, run.WorkspaceID),
			paletteActionMsg{Verb: verbInspect, Target: paletteTarget{Kind: "run", RouteID: routeRuns, RunID: runID}},
		)
	}
}

func (m shellModel) appendApprovalPaletteEntries(add func(string, string, string, paletteActionMsg)) {
	approvalsModel, ok := m.routeModels[routeApprovals].(approvalsRouteModel)
	if !ok {
		return
	}
	for _, ap := range approvalsModel.items {
		approvalID := strings.TrimSpace(ap.ApprovalID)
		if approvalID == "" {
			continue
		}
		add(
			"inspect approval "+approvalID,
			fmt.Sprintf("status=%s trigger=%s", ap.Status, ap.ApprovalTriggerCode),
			fmt.Sprintf("inspect approval %s %s", approvalID, ap.BoundScope.ActionKind),
			paletteActionMsg{Verb: verbInspect, Target: paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: approvalID}},
		)
	}
}

func (m shellModel) appendActionCenterPaletteEntries(add func(string, string, string, paletteActionMsg)) {
	actionModel, ok := m.routeModels[routeAction].(actionCenterRouteModel)
	if !ok {
		return
	}
	for family, items := range actionModel.familyBuckets() {
		for _, item := range items {
			if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Detail) == "" {
				continue
			}
			target := item.Target
			if strings.TrimSpace(target.Kind) == "" {
				target = paletteTarget{Kind: "route", RouteID: routeAction}
			}
			add(
				fmt.Sprintf("triage %s %s", family, item.Title),
				fmt.Sprintf("urgency=%s impact=%s", valueOrNA(item.Urgency), valueOrNA(item.Impact)),
				fmt.Sprintf("triage action center %s %s %s %s %s", family, item.Title, item.Detail, item.Impact, item.ExpiryCue),
				paletteActionMsg{Verb: verbJump, Target: target},
			)
		}
	}
}

func (m shellModel) appendArtifactPaletteEntries(add func(string, string, string, paletteActionMsg)) {
	artifactsModel, ok := m.routeModels[routeArtifacts].(artifactsRouteModel)
	if !ok {
		return
	}
	for _, item := range artifactsModel.items {
		digest := strings.TrimSpace(item.Reference.Digest)
		if digest == "" {
			continue
		}
		add(
			"inspect artifact "+digest,
			fmt.Sprintf("class=%s bytes=%d", item.Reference.DataClass, item.Reference.SizeBytes),
			fmt.Sprintf("inspect artifact %s %s", digest, item.Reference.DataClass),
			paletteActionMsg{Verb: verbInspect, Target: paletteTarget{Kind: "artifact", RouteID: routeArtifacts, Digest: digest}},
		)
	}
}

func (m shellModel) appendAuditPaletteEntries(add func(string, string, string, paletteActionMsg)) {
	auditModel, ok := m.routeModels[routeAudit].(auditRouteModel)
	if !ok {
		return
	}
	for _, item := range auditModel.timeline {
		digest, err := item.RecordDigest.Identity()
		if err != nil || strings.TrimSpace(digest) == "" {
			continue
		}
		add(
			"inspect audit "+digest,
			fmt.Sprintf("event=%s summary=%s", item.EventType, item.Summary),
			fmt.Sprintf("inspect audit %s", digest),
			paletteActionMsg{Verb: verbInspect, Target: paletteTarget{Kind: "audit", RouteID: routeAudit, Digest: digest}},
		)
	}
}

func (m shellModel) appendCommandPaletteEntries(add func(string, string, string, paletteActionMsg)) {
	for _, cmd := range m.commands.List() {
		add(
			"open command "+cmd.Title,
			cmd.Description,
			"open command "+cmd.ID+" "+cmd.Title+" "+cmd.Description,
			paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: cmd.ID}},
		)
	}
}
