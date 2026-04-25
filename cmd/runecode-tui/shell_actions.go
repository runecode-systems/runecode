package main

import (
	"fmt"
	"sort"
	"strings"
)

type commandModeParseResult struct {
	actionID string
	args     []string
}

type shellActionScope string

const (
	shellActionScopeGlobal         shellActionScope = "global"
	shellActionScopeRouteSensitive shellActionScope = "route_sensitive"
)

type shellActionDefinition struct {
	ID             string
	Title          string
	Description    string
	CommandAliases []string
	LeaderPath     []string
	LeaderGroup    string
	PaletteShow    bool
	PaletteSearch  string
	HelpText       string
	Scope          shellActionScope
	Resolve        func(shellModel) (paletteActionMsg, bool)
	Available      func(shellModel) bool
}

type shellActionGraph struct {
	actions []shellActionDefinition
	byID    map[string]shellActionDefinition
	aliases map[string]string
}

func newShellActionGraph(routes []routeDefinition, commands shellCommandRegistry) shellActionGraph {
	graph := shellActionGraph{byID: map[string]shellActionDefinition{}, aliases: map[string]string{}}
	registerCommandBackedActions(&graph, commands)
	registerShellSurfaceActions(&graph)
	registerRouteJumpActions(&graph, routes)
	return graph
}

func registerCommandBackedActions(graph *shellActionGraph, commands shellCommandRegistry) {
	for _, cmd := range commands.List() {
		graph.add(commandActionDefinition(cmd))
	}
}

func commandActionDefinition(cmd shellCommand) shellActionDefinition {
	action := shellActionDefinition{
		ID:             strings.TrimSpace(cmd.ID),
		Title:          strings.TrimSpace(cmd.Title),
		Description:    strings.TrimSpace(cmd.Description),
		CommandAliases: append([]string(nil), cmd.Aliases...),
		LeaderPath:     normalizeActionPath(cmd.LeaderPath),
		LeaderGroup:    strings.TrimSpace(cmd.LeaderGroup),
		PaletteShow:    cmd.PaletteShow,
		PaletteSearch:  strings.TrimSpace(cmd.PaletteText),
		HelpText:       strings.TrimSpace(cmd.HelpText),
		Scope:          cmd.Scope,
		Available:      cmd.Available,
	}
	commandID := action.ID
	commandScope := action.Scope
	action.Resolve = func(_ shellModel) (paletteActionMsg, bool) {
		if strings.TrimSpace(commandID) == "" {
			return paletteActionMsg{}, false
		}
		return paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: commandID, CommandArgs: nil}}, true
	}
	if commandScope == shellActionScopeRouteSensitive && action.Available == nil {
		action.Available = func(shell shellModel) bool {
			return shell.keyboardOwnership() == routeKeyboardOwnershipNormal
		}
	}
	return action
}

func registerShellSurfaceActions(graph *shellActionGraph) {
	for _, action := range []shellActionDefinition{
		{ID: "shell.back", Title: "Back", Description: "Back to previous location", CommandAliases: []string{"back"}, LeaderPath: []string{"q", "b"}, LeaderGroup: "Quit/Back", PaletteShow: true, PaletteSearch: "back jump previous route", HelpText: "back — navigate to previous location", Scope: shellActionScopeGlobal, Resolve: func(_ shellModel) (paletteActionMsg, bool) { return paletteActionMsg{Verb: verbBack}, true }},
		{ID: "shell.quit", Title: "Quit RuneCode", Description: "Exit the TUI", CommandAliases: []string{"q", "quit"}, LeaderPath: []string{"q", "q"}, LeaderGroup: "Quit/Back", PaletteShow: true, PaletteSearch: "quit exit close", HelpText: "q, quit — exit runecode-tui", Scope: shellActionScopeGlobal, Resolve: func(_ shellModel) (paletteActionMsg, bool) { return paletteActionMsg{Verb: verbQuit}, true }},
		{ID: "shell.copy_identity", Title: "Copy Current Identity", Description: "Copy current route/object identity", CommandAliases: []string{"copy identity"}, LeaderPath: []string{"c", "i"}, LeaderGroup: "Copy", PaletteShow: true, PaletteSearch: "copy identity breadcrumb route object", HelpText: "copy identity — copy current object identity", Scope: shellActionScopeRouteSensitive, Resolve: func(_ shellModel) (paletteActionMsg, bool) {
			return paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: "shell.copy_identity"}}, true
		}},
		{ID: "shell.copy_route_action", Title: "Copy Next Route Action", Description: "Cycle and execute route copy action", CommandAliases: []string{"copy next", "copy action"}, LeaderPath: []string{"c", "a"}, LeaderGroup: "Copy", PaletteShow: true, PaletteSearch: "copy next route action", HelpText: "copy next — cycle and execute route copy action", Scope: shellActionScopeRouteSensitive, Resolve: func(_ shellModel) (paletteActionMsg, bool) {
			return paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: "shell.copy_route_action"}}, true
		}},
		{ID: "shell.open_route", Title: "Open Route", Description: "Jump to route by id/alias", CommandAliases: []string{"open"}, LeaderPath: []string{"o", "o"}, LeaderGroup: "Open/Jump", PaletteShow: false, PaletteSearch: "open route jump", HelpText: "open <route> — jump to route", Scope: shellActionScopeGlobal, Resolve: func(_ shellModel) (paletteActionMsg, bool) {
			return paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: "shell.open_route"}}, true
		}},
		{ID: "shell.set_leader", Title: "Set Leader Key", Description: "Set shell leader key in local workbench preferences", CommandAliases: []string{"set leader"}, LeaderPath: []string{"w", "l"}, LeaderGroup: "Workbench", PaletteShow: true, PaletteSearch: "set leader key preference", HelpText: "set leader <space|comma|backslash|default> — configure persisted leader key", Scope: shellActionScopeGlobal, Resolve: func(_ shellModel) (paletteActionMsg, bool) {
			return paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: "shell.set_leader"}}, true
		}},
		{ID: "shell.open_approvals", Title: "Open Approvals", Description: "Jump to approvals queue", CommandAliases: []string{"approvals", "open approvals"}, LeaderPath: []string{"a", "p"}, LeaderGroup: "Approvals/Action Center", PaletteShow: true, PaletteSearch: "approvals queue pending decisions", HelpText: "open approvals — jump to approvals", Scope: shellActionScopeGlobal, Resolve: func(_ shellModel) (paletteActionMsg, bool) {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeApprovals}}, true
		}},
		{ID: "shell.open_action_center", Title: "Open Action Center", Description: "Jump to action-center triage", CommandAliases: []string{"action center", "open action-center", "open action center"}, LeaderPath: []string{"a", "a"}, LeaderGroup: "Approvals/Action Center", PaletteShow: true, PaletteSearch: "action center triage queues", HelpText: "open action-center — jump to action center", Scope: shellActionScopeGlobal, Resolve: func(_ shellModel) (paletteActionMsg, bool) {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAction}}, true
		}},
		{ID: "shell.open_palette", Title: "Open Command Discovery", Description: "Open fuzzy command/object palette", CommandAliases: []string{"search", "discover", "palette"}, LeaderPath: []string{"s", "p"}, LeaderGroup: "Search/Discovery", PaletteShow: false, PaletteSearch: "search discovery palette ctrl+p", HelpText: "search — open command discovery palette", Scope: shellActionScopeGlobal, Resolve: func(_ shellModel) (paletteActionMsg, bool) {
			return paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: "shell.open_palette"}}, true
		}},
		{ID: "shell.open_sessions", Title: "Open Session Switcher", Description: "Open fuzzy session quick switcher", CommandAliases: []string{"sessions", "session switch", "search sessions"}, LeaderPath: []string{"s", "s"}, LeaderGroup: "Search/Discovery", PaletteShow: true, PaletteSearch: "search discover sessions quick switch", HelpText: "sessions — open session quick switcher", Scope: shellActionScopeGlobal, Resolve: func(_ shellModel) (paletteActionMsg, bool) {
			return paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "command", CommandID: "shell.open_sessions"}}, true
		}},
	} {
		graph.add(action)
	}
}

func registerRouteJumpActions(graph *shellActionGraph, routes []routeDefinition) {
	for _, route := range routes {
		graph.add(routeJumpActionDefinition(route))
	}
}

func routeJumpActionDefinition(route routeDefinition) shellActionDefinition {
	leader := []string{"o", strings.ToLower(strings.TrimSpace(route.QuickJumpKey))}
	if strings.TrimSpace(route.QuickJumpKey) == "" {
		leader = nil
	}
	routeID := route.ID
	label := strings.TrimSpace(route.Label)
	return shellActionDefinition{
		ID:            "route.jump." + strings.TrimSpace(string(route.ID)),
		Title:         "Open " + label,
		Description:   strings.TrimSpace(route.Description),
		LeaderPath:    leader,
		LeaderGroup:   "Open/Jump",
		PaletteShow:   true,
		PaletteSearch: strings.ToLower(strings.TrimSpace(route.Label) + " " + strings.TrimSpace(route.Description) + " route jump open"),
		HelpText:      fmt.Sprintf("open %s — jump to %s", strings.ToLower(label), label),
		Scope:         shellActionScopeGlobal,
		Resolve: func(_ shellModel) (paletteActionMsg, bool) {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeID}}, true
		},
	}
}

func (g *shellActionGraph) add(action shellActionDefinition) {
	id := strings.TrimSpace(action.ID)
	if id == "" || action.Resolve == nil {
		return
	}
	action.ID = id
	if existing, ok := g.byID[id]; ok {
		action = mergeShellActionDefinition(existing, action)
	}
	if strings.TrimSpace(action.HelpText) == "" {
		action.HelpText = action.Description
	}
	if action.Scope == "" {
		action.Scope = shellActionScopeGlobal
	}
	replaced := false
	for i := range g.actions {
		if g.actions[i].ID == id {
			g.actions[i] = action
			replaced = true
			break
		}
	}
	if !replaced {
		g.actions = append(g.actions, action)
	}
	g.byID[id] = action
	for _, alias := range action.CommandAliases {
		normalized := normalizeCommandAlias(alias)
		if normalized == "" {
			continue
		}
		g.aliases[normalized] = id
	}
}

func mergeShellActionDefinition(existing shellActionDefinition, override shellActionDefinition) shellActionDefinition {
	merged := existing
	merged.ID = override.ID
	if title := strings.TrimSpace(override.Title); title != "" {
		merged.Title = title
	}
	if description := strings.TrimSpace(override.Description); description != "" {
		merged.Description = description
	}
	if len(override.CommandAliases) > 0 {
		merged.CommandAliases = append([]string(nil), override.CommandAliases...)
	}
	if len(override.LeaderPath) > 0 {
		merged.LeaderPath = append([]string(nil), override.LeaderPath...)
	}
	if group := strings.TrimSpace(override.LeaderGroup); group != "" {
		merged.LeaderGroup = group
	}
	if override.PaletteShow {
		merged.PaletteShow = true
	}
	if search := strings.TrimSpace(override.PaletteSearch); search != "" {
		merged.PaletteSearch = search
	}
	if help := strings.TrimSpace(override.HelpText); help != "" {
		merged.HelpText = help
	}
	if override.Scope != "" {
		merged.Scope = override.Scope
	}
	if override.Resolve != nil {
		merged.Resolve = override.Resolve
	}
	if override.Available != nil {
		merged.Available = override.Available
	}
	return merged
}

func (g shellActionGraph) resolveByID(id string, model shellModel) (paletteActionMsg, bool) {
	action, ok := g.byID[strings.TrimSpace(id)]
	if !ok {
		return paletteActionMsg{}, false
	}
	if action.Available != nil && !action.Available(model) {
		return paletteActionMsg{}, false
	}
	return action.Resolve(model)
}

func (g shellActionGraph) definitionByID(id string) (shellActionDefinition, bool) {
	action, ok := g.byID[strings.TrimSpace(id)]
	if !ok {
		return shellActionDefinition{}, false
	}
	return action, true
}

func (g shellActionGraph) resolveCommandModeDraft(draft string, model shellModel) (paletteActionMsg, error) {
	raw := normalizeCommandAlias(draft)
	if raw == "" {
		return paletteActionMsg{}, fmt.Errorf("empty command")
	}
	if strings.HasPrefix(raw, "command ") {
		parts := strings.Fields(raw)
		if len(parts) < 2 {
			return paletteActionMsg{}, fmt.Errorf("usage: command <id>")
		}
		action, ok := g.resolveByID(parts[1], model)
		if !ok {
			return paletteActionMsg{}, fmt.Errorf("execution failed: unknown command %s", parts[1])
		}
		return action, nil
	}
	parsed := parseCommandAlias(raw, g)
	actionID, ok := g.aliases[parsed.actionID]
	if !ok {
		return paletteActionMsg{}, fmt.Errorf("unknown command")
	}
	action, ok := g.resolveByID(actionID, model)
	if !ok {
		return paletteActionMsg{}, fmt.Errorf("execution failed: command unavailable")
	}
	if action.Target.Kind == "command" {
		action.Target.CommandArgs = append([]string(nil), parsed.args...)
	}
	return action, nil
}

func parseCommandAlias(raw string, graph shellActionGraph) commandModeParseResult {
	result := commandModeParseResult{actionID: raw}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return result
	}
	for size := len(fields); size >= 1; size-- {
		candidate := strings.Join(fields[:size], " ")
		if _, ok := graph.aliases[candidate]; !ok {
			continue
		}
		result.actionID = candidate
		if len(fields) > size {
			result.args = append(result.args, fields[size:]...)
		}
		return result
	}
	return result
}

func (g shellActionGraph) leaderBindings(model shellModel) []shellLeaderBinding {
	bindings := make([]shellLeaderBinding, 0, len(g.actions))
	for _, action := range g.actions {
		if len(action.LeaderPath) == 0 {
			continue
		}
		if action.Available != nil && !action.Available(model) {
			continue
		}
		resolved, ok := action.Resolve(model)
		if !ok {
			continue
		}
		bindings = append(bindings, shellLeaderBinding{
			Sequence:    append([]string(nil), action.LeaderPath...),
			Group:       strings.TrimSpace(action.LeaderGroup),
			Label:       strings.TrimSpace(action.Title),
			Description: strings.TrimSpace(action.Description),
			Action:      resolved,
		})
	}
	return bindings
}

func (g shellActionGraph) appendPaletteEntries(add func(string, string, string, paletteActionMsg), model shellModel) {
	for _, action := range g.actions {
		if !action.PaletteShow {
			continue
		}
		if action.Available != nil && !action.Available(model) {
			continue
		}
		resolved, ok := action.Resolve(model)
		if !ok {
			continue
		}
		search := strings.TrimSpace(action.PaletteSearch)
		if search == "" {
			search = strings.ToLower(strings.TrimSpace(action.Title + " " + action.Description + " " + strings.Join(action.CommandAliases, " ")))
		}
		add(strings.ToLower(strings.TrimSpace(action.Title)), action.Description, search, resolved)
	}
}

func (g shellActionGraph) helpEntries(limit int) []string {
	if limit <= 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for _, action := range g.actions {
		text := strings.TrimSpace(action.HelpText)
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	sort.Strings(out)
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func normalizeActionPath(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		normalized := normalizeLeaderToken(token)
		if normalized == "" {
			continue
		}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeCommandAlias(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(value, ":") {
		value = strings.TrimSpace(strings.TrimPrefix(value, ":"))
	}
	return strings.Join(strings.Fields(value), " ")
}
