package main

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const (
	objectIndexSessionListLimit  = 50
	objectIndexRunListLimit      = 40
	objectIndexApprovalListLimit = 40
	objectIndexArtifactListLimit = 40
	objectIndexAuditListLimit    = 20
)

type shellObjectIndexLoadedMsg struct {
	sessions     []brokerapi.SessionSummary
	sessionErr   error
	runs         []brokerapi.RunSummary
	runErr       error
	approvals    []brokerapi.ApprovalSummary
	approvalErr  error
	artifacts    []brokerapi.ArtifactSummary
	artifactErr  error
	auditRecords []brokerapi.AuditTimelineViewEntry
	auditErr     error
}

type shellDiscoverabilityIndex struct {
	routes        []routeDefinition
	commands      []shellCommand
	sessions      map[string]brokerapi.SessionSummary
	runs          map[string]brokerapi.RunSummary
	approvals     map[string]brokerapi.ApprovalSummary
	artifacts     map[string]brokerapi.ArtifactSummary
	auditRecords  map[string]brokerapi.AuditTimelineViewEntry
	recentObjects []workbenchObjectRef
	recentSession []string
	activeSession string
	sessionWS     map[string]string
}

func newShellDiscoverabilityIndex(routes []routeDefinition, commands []shellCommand) shellDiscoverabilityIndex {
	idx := shellDiscoverabilityIndex{
		routes:       append([]routeDefinition(nil), routes...),
		commands:     append([]shellCommand(nil), commands...),
		sessions:     map[string]brokerapi.SessionSummary{},
		runs:         map[string]brokerapi.RunSummary{},
		approvals:    map[string]brokerapi.ApprovalSummary{},
		artifacts:    map[string]brokerapi.ArtifactSummary{},
		auditRecords: map[string]brokerapi.AuditTimelineViewEntry{},
		sessionWS:    map[string]string{},
	}
	return idx
}

func (m shellModel) loadObjectIndexCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()

		sessionResp, sessionErr := m.client.SessionList(ctx, objectIndexSessionListLimit)
		runResp, runErr := m.client.RunList(ctx, objectIndexRunListLimit)
		approvalResp, approvalErr := m.client.ApprovalList(ctx, objectIndexApprovalListLimit)
		artifactResp, artifactErr := m.client.ArtifactList(ctx, objectIndexArtifactListLimit, "")
		auditResp, auditErr := m.client.AuditTimeline(ctx, objectIndexAuditListLimit, "")

		return shellObjectIndexLoadedMsg{
			sessions:     sessionResp.Sessions,
			sessionErr:   sessionErr,
			runs:         runResp.Runs,
			runErr:       runErr,
			approvals:    approvalResp.Approvals,
			approvalErr:  approvalErr,
			artifacts:    artifactResp.Artifacts,
			artifactErr:  artifactErr,
			auditRecords: auditResp.Views,
			auditErr:     auditErr,
		}
	}
}

func (m *shellModel) applyObjectIndexLoaded(msg shellObjectIndexLoadedMsg) {
	if msg.sessionErr == nil {
		m.objectIndex.ingestSessions(msg.sessions)
	}
	if msg.runErr == nil {
		m.objectIndex.ingestRuns(msg.runs)
	}
	if msg.approvalErr == nil {
		m.objectIndex.ingestApprovals(msg.approvals)
	}
	if msg.artifactErr == nil {
		m.objectIndex.ingestArtifacts(msg.artifacts)
	}
	if msg.auditErr == nil {
		m.objectIndex.ingestAuditRecords(msg.auditRecords)
	}
	m.refreshObjectIndexFromShellState()
}

func (m *shellModel) refreshObjectIndexFromShellState() {
	m.objectIndex.activeSession = strings.TrimSpace(m.activeSessionID)
	m.objectIndex.recentObjects = append([]workbenchObjectRef(nil), m.recentObjects...)
	m.objectIndex.recentSession = append([]string(nil), m.recentSessions...)
	m.objectIndex.sessionWS = cloneSessionMap(m.sessionWorkspace)
	m.objectIndex.ingestSessions(m.sessionItems)
	m.objectIndex.ingestRuns(shellRunSummariesFromWatch(m.watch.reduction.runs))
	m.objectIndex.ingestApprovals(shellApprovalSummariesFromWatch(m.watch.reduction.approvals))
	m.objectIndex.ingestSessions(shellSessionSummariesFromWatch(m.watch.reduction.sessions))
	if m.palette.IsOpen() {
		m.palette = m.palette.UpdateEntries(m.buildPaletteEntries())
	}
}

func shellRunSummariesFromWatch(items map[string]brokerapi.RunSummary) []brokerapi.RunSummary {
	if len(items) == 0 {
		return nil
	}
	out := make([]brokerapi.RunSummary, 0, len(items))
	for _, summary := range items {
		out = append(out, summary)
	}
	return out
}

func shellApprovalSummariesFromWatch(items map[string]brokerapi.ApprovalSummary) []brokerapi.ApprovalSummary {
	if len(items) == 0 {
		return nil
	}
	out := make([]brokerapi.ApprovalSummary, 0, len(items))
	for _, summary := range items {
		out = append(out, summary)
	}
	return out
}

func shellSessionSummariesFromWatch(items map[string]brokerapi.SessionSummary) []brokerapi.SessionSummary {
	if len(items) == 0 {
		return nil
	}
	out := make([]brokerapi.SessionSummary, 0, len(items))
	for _, summary := range items {
		out = append(out, summary)
	}
	return out
}

func (idx *shellDiscoverabilityIndex) ingestSessions(items []brokerapi.SessionSummary) {
	for _, item := range items {
		sid := strings.TrimSpace(item.Identity.SessionID)
		if sid == "" {
			continue
		}
		idx.sessions[sid] = item
		if ws := strings.TrimSpace(item.Identity.WorkspaceID); ws != "" {
			idx.sessionWS[sid] = ws
		}
	}
}

func (idx *shellDiscoverabilityIndex) ingestRuns(items []brokerapi.RunSummary) {
	for _, item := range items {
		runID := strings.TrimSpace(item.RunID)
		if runID == "" {
			continue
		}
		idx.runs[runID] = item
	}
}

func (idx *shellDiscoverabilityIndex) ingestApprovals(items []brokerapi.ApprovalSummary) {
	for _, item := range items {
		approvalID := strings.TrimSpace(item.ApprovalID)
		if approvalID == "" {
			continue
		}
		idx.approvals[approvalID] = item
	}
}

func (idx *shellDiscoverabilityIndex) ingestArtifacts(items []brokerapi.ArtifactSummary) {
	for _, item := range items {
		digest := strings.TrimSpace(item.Reference.Digest)
		if digest == "" {
			continue
		}
		idx.artifacts[digest] = item
	}
}

func (idx *shellDiscoverabilityIndex) ingestAuditRecords(items []brokerapi.AuditTimelineViewEntry) {
	for _, item := range items {
		digest, err := item.RecordDigest.Identity()
		if err != nil {
			continue
		}
		digest = strings.TrimSpace(digest)
		if digest == "" {
			continue
		}
		idx.auditRecords[digest] = item
	}
}

func (idx shellDiscoverabilityIndex) appendPaletteEntries(add func(string, string, string, paletteActionMsg)) {
	idx.appendRecentEntries(add)
	idx.appendRouteEntries(add)
	idx.appendSessionEntries(add)
	idx.appendRunEntries(add)
	idx.appendApprovalEntries(add)
	idx.appendArtifactEntries(add)
	idx.appendAuditEntries(add)
	idx.appendCommandEntries(add)
}

func (idx shellDiscoverabilityIndex) appendRouteEntries(add func(string, string, string, paletteActionMsg)) {
	for _, route := range idx.routes {
		add(
			fmt.Sprintf("jump route %s", strings.ToLower(route.Label)),
			route.Description,
			fmt.Sprintf("route %s %s", strings.ToLower(route.Label), strings.ToLower(string(route.ID))),
			paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: route.ID}},
		)
	}
}

func (idx shellDiscoverabilityIndex) appendSessionEntries(add func(string, string, string, paletteActionMsg)) {
	for _, sid := range sortedKeys(idx.sessions) {
		s := idx.sessions[sid]
		ws := strings.TrimSpace(s.Identity.WorkspaceID)
		if ws == "" {
			ws = strings.TrimSpace(idx.sessionWS[sid])
		}
		preview := truncateText(s.LastActivityPreview, 56)
		add(
			"open session "+sid,
			fmt.Sprintf("ws=%s status=%s activity=%s", defaultPlaceholder(ws, "n/a"), defaultPlaceholder(s.Status, "n/a"), defaultPlaceholder(s.LastActivityKind, "n/a")),
			fmt.Sprintf("session %s %s %s", sid, ws, preview),
			paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "session", SessionID: sid}},
		)
	}
}

func (idx shellDiscoverabilityIndex) appendRunEntries(add func(string, string, string, paletteActionMsg)) {
	for _, runID := range sortedKeys(idx.runs) {
		run := idx.runs[runID]
		add(
			"inspect run "+runID,
			fmt.Sprintf("state=%s approvals=%d ws=%s", defaultPlaceholder(run.LifecycleState, "n/a"), run.PendingApprovalCount, defaultPlaceholder(run.WorkspaceID, "n/a")),
			fmt.Sprintf("run %s %s %s", runID, run.WorkspaceID, run.LifecycleState),
			paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "run", RouteID: routeRuns, RunID: runID}},
		)
	}
}

func (idx shellDiscoverabilityIndex) appendApprovalEntries(add func(string, string, string, paletteActionMsg)) {
	for _, approvalID := range sortedKeys(idx.approvals) {
		approval := idx.approvals[approvalID]
		add(
			"inspect approval "+approvalID,
			fmt.Sprintf("status=%s trigger=%s", defaultPlaceholder(approval.Status, "n/a"), defaultPlaceholder(approval.ApprovalTriggerCode, "n/a")),
			fmt.Sprintf("approval %s %s %s", approvalID, approval.BoundScope.WorkspaceID, approval.BoundScope.ActionKind),
			paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: approvalID}},
		)
	}
}

func (idx shellDiscoverabilityIndex) appendArtifactEntries(add func(string, string, string, paletteActionMsg)) {
	for _, digest := range sortedKeys(idx.artifacts) {
		artifact := idx.artifacts[digest]
		add(
			"inspect artifact "+digest,
			fmt.Sprintf("class=%s bytes=%d run=%s", defaultPlaceholder(fmt.Sprintf("%v", artifact.Reference.DataClass), "n/a"), artifact.Reference.SizeBytes, defaultPlaceholder(artifact.RunID, "n/a")),
			fmt.Sprintf("artifact %s %v %s", digest, artifact.Reference.DataClass, artifact.RunID),
			paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "artifact", RouteID: routeArtifacts, Digest: digest}},
		)
	}
}

func (idx shellDiscoverabilityIndex) appendAuditEntries(add func(string, string, string, paletteActionMsg)) {
	for _, digest := range sortedKeys(idx.auditRecords) {
		record := idx.auditRecords[digest]
		add(
			"inspect audit "+digest,
			fmt.Sprintf("event=%s summary=%s", defaultPlaceholder(record.EventType, "n/a"), defaultPlaceholder(record.Summary, "n/a")),
			fmt.Sprintf("audit %s %s", digest, record.EventType),
			paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "audit", RouteID: routeAudit, Digest: digest}},
		)
	}
}

func (idx shellDiscoverabilityIndex) appendCommandEntries(add func(string, string, string, paletteActionMsg)) {
	for _, cmd := range idx.commands {
		add(
			"open command "+cmd.Title,
			cmd.Description,
			"command "+cmd.ID+" "+cmd.Title+" "+cmd.Description,
			paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: cmd.ID}},
		)
	}
}

func (idx shellDiscoverabilityIndex) appendRecentEntries(add func(string, string, string, paletteActionMsg)) {
	for _, ref := range idx.recentObjects {
		target, ok := paletteTargetFromObjectRef(ref)
		if !ok {
			continue
		}
		id := strings.TrimSpace(ref.ID)
		if id == "" {
			continue
		}
		ws := strings.TrimSpace(ref.WorkspaceID)
		if ws == "" && ref.Kind == "session" {
			ws = strings.TrimSpace(idx.sessionWS[id])
		}
		add(
			fmt.Sprintf("recent %s %s", ref.Kind, id),
			fmt.Sprintf("ws=%s session=%s", defaultPlaceholder(ws, "n/a"), defaultPlaceholder(ref.SessionID, "n/a")),
			fmt.Sprintf("recent %s %s %s", ref.Kind, id, ws),
			paletteActionMsg{Verb: verbOpen, Target: target},
		)
	}
	for _, sid := range idx.recentSession {
		sid = strings.TrimSpace(sid)
		if sid == "" {
			continue
		}
		add(
			"recent session "+sid,
			fmt.Sprintf("ws=%s", defaultPlaceholder(idx.sessionWS[sid], "n/a")),
			"recent session "+sid+" "+idx.sessionWS[sid],
			paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "session", SessionID: sid}},
		)
	}
}

func paletteTargetFromObjectRef(ref workbenchObjectRef) (paletteTarget, bool) {
	switch strings.TrimSpace(strings.ToLower(ref.Kind)) {
	case "route":
		return paletteTarget{Kind: "route", RouteID: routeID(strings.TrimSpace(ref.ID))}, true
	case "session":
		return paletteTarget{Kind: "session", SessionID: strings.TrimSpace(ref.ID)}, true
	case "run":
		return paletteTarget{Kind: "run", RouteID: routeRuns, RunID: strings.TrimSpace(ref.ID)}, true
	case "approval":
		return paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: strings.TrimSpace(ref.ID)}, true
	case "artifact":
		return paletteTarget{Kind: "artifact", RouteID: routeArtifacts, Digest: strings.TrimSpace(ref.ID)}, true
	case "audit":
		return paletteTarget{Kind: "audit", RouteID: routeAudit, Digest: strings.TrimSpace(ref.ID)}, true
	default:
		return paletteTarget{}, false
	}
}

func sortedKeys[T any](items map[string]T) []string {
	out := make([]string, 0, len(items))
	for key := range items {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
