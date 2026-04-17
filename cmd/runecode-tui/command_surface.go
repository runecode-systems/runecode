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
	m.objectIndex.appendPaletteEntries(add)
	m.appendActionCenterPaletteEntries(add)
	m.appendActiveSurfaceActionEntries(add)

	return entries
}

func (m shellModel) appendActiveSurfaceActionEntries(add func(string, string, string, paletteActionMsg)) {
	surface := m.activeShellSurface()
	for _, action := range surface.Actions.LocalActions {
		label := strings.TrimSpace(action.Label)
		if label == "" {
			continue
		}
		add(
			"action "+label,
			"execute local inspector action",
			"action local "+strings.ToLower(label),
			action.Action,
		)
	}
	for _, action := range surface.Actions.ReferenceActions {
		label := strings.TrimSpace(action.Label)
		if label == "" {
			continue
		}
		add(
			"reference "+label,
			"open linked reference target",
			"reference linked "+strings.ToLower(label),
			action.Action,
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
