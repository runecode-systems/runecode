package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type keyBinding struct {
	Keys        []string
	Description string
}

func (k keyBinding) matches(msg tea.KeyMsg) bool {
	pressed := msg.String()
	for _, key := range k.Keys {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "space":
			if msg.Type == tea.KeySpace || pressed == " " {
				return true
			}
			continue
		}
		if pressed == key {
			return true
		}
	}
	return false
}

func (k keyBinding) label() string {
	return strings.Join(k.Keys, "/")
}

type shellKeyMap struct {
	Quit                          keyBinding
	LeaderStart                   keyBinding
	CycleFocusNext                keyBinding
	CycleFocusPrev                keyBinding
	ToggleSidebar                 keyBinding
	BackRoute                     keyBinding
	CopyIdentity                  keyBinding
	CopyRouteAction               keyBinding
	ToggleSelectionMode           keyBinding
	RunCommand                    keyBinding
	CycleTheme                    keyBinding
	LayoutSidebarWider            keyBinding
	LayoutSidebarNarrower         keyBinding
	LayoutInspectorWider          keyBinding
	LayoutInspectorNarrower       keyBinding
	LayoutToggleSidebarCollapse   keyBinding
	LayoutToggleInspectorCollapse keyBinding
	RouteNext                     keyBinding
	RoutePrev                     keyBinding
	RouteOpen                     keyBinding
	SessionOpen                   keyBinding
	SessionNext                   keyBinding
	SessionPrev                   keyBinding
	SessionPin                    keyBinding
	QuickJump                     keyBinding
	OpenPalette                   keyBinding
	OpenSessionQuickSwitch        keyBinding
	SessionQuickSwitchClose       keyBinding
	SessionQuickSwitchPick        keyBinding
	SessionQuickSwitchNext        keyBinding
	SessionQuickSwitchPrev        keyBinding
	PaletteClose                  keyBinding
	PalettePick                   keyBinding
	PaletteNext                   keyBinding
	PalettePrev                   keyBinding
	ScrollDown                    keyBinding
	ScrollUp                      keyBinding
}

func defaultShellKeyMap() shellKeyMap {
	leaderStart, err := shellLeaderStartKeyBinding("space")
	if err != nil {
		leaderStart = keyBinding{Keys: []string{"space"}, Description: "Enter leader mode"}
	}
	return shellKeyMap{
		Quit:                          keyBinding{Keys: []string{"ctrl+c"}, Description: "Emergency quit"},
		LeaderStart:                   leaderStart,
		CycleFocusNext:                keyBinding{Keys: []string{"tab"}, Description: "Next focus area"},
		CycleFocusPrev:                keyBinding{Keys: []string{"shift+tab"}, Description: "Previous focus area"},
		ToggleSidebar:                 keyBinding{Description: "Toggle sidebar"},
		BackRoute:                     keyBinding{Description: "Back to previous route"},
		CopyIdentity:                  keyBinding{Description: "Copy current identity"},
		CopyRouteAction:               keyBinding{Description: "Copy next route copy action"},
		ToggleSelectionMode:           keyBinding{Description: "Toggle selection mode (mouse capture)"},
		RunCommand:                    keyBinding{Description: "Run shell command"},
		CycleTheme:                    keyBinding{Description: "Cycle theme preset"},
		LayoutSidebarWider:            keyBinding{Description: "Widen sidebar pane"},
		LayoutSidebarNarrower:         keyBinding{Description: "Narrow sidebar pane"},
		LayoutInspectorWider:          keyBinding{Description: "Widen inspector pane"},
		LayoutInspectorNarrower:       keyBinding{Description: "Narrow inspector pane"},
		LayoutToggleSidebarCollapse:   keyBinding{Description: "Collapse/expand sidebar pane"},
		LayoutToggleInspectorCollapse: keyBinding{Description: "Collapse/expand inspector pane"},
		RouteNext:                     keyBinding{Keys: []string{"down"}, Description: "Move sidebar cursor down"},
		RoutePrev:                     keyBinding{Keys: []string{"up"}, Description: "Move sidebar cursor up"},
		RouteOpen:                     keyBinding{Keys: []string{"enter"}, Description: "Open selected route"},
		SessionOpen:                   keyBinding{Keys: []string{"enter"}, Description: "Open selected session"},
		SessionNext:                   keyBinding{Keys: []string{"down"}, Description: "Move session selection down"},
		SessionPrev:                   keyBinding{Keys: []string{"up"}, Description: "Move session selection up"},
		SessionPin:                    keyBinding{Description: "Pin/unpin selected session"},
		QuickJump:                     keyBinding{Description: "Open route by number"},
		OpenPalette:                   keyBinding{Keys: []string{"ctrl+p"}, Description: "Open quick jump palette"},
		OpenSessionQuickSwitch:        keyBinding{Keys: []string{"ctrl+j"}, Description: "Open session quick switcher"},
		SessionQuickSwitchClose:       keyBinding{Keys: []string{"esc"}, Description: "Close session quick switcher"},
		SessionQuickSwitchPick:        keyBinding{Keys: []string{"enter"}, Description: "Switch to selected session"},
		SessionQuickSwitchNext:        keyBinding{Keys: []string{"down", "ctrl+n"}, Description: "Next session match"},
		SessionQuickSwitchPrev:        keyBinding{Keys: []string{"up", "ctrl+p"}, Description: "Previous session match"},
		PaletteClose:                  keyBinding{Keys: []string{"esc"}, Description: "Close palette"},
		PalettePick:                   keyBinding{Keys: []string{"enter"}, Description: "Open selected match"},
		PaletteNext:                   keyBinding{Keys: []string{"down", "ctrl+n"}, Description: "Next palette match"},
		PalettePrev:                   keyBinding{Keys: []string{"up", "ctrl+p"}, Description: "Previous palette match"},
		ScrollDown:                    keyBinding{Description: "Scroll content down"},
		ScrollUp:                      keyBinding{Description: "Scroll content up"},
	}
}

func (k shellKeyMap) helpBindings(paletteOpen bool) []keyBinding {
	base := nonEmptyBindings(
		k.LeaderStart,
		k.CycleFocusNext,
		k.CycleFocusPrev,
		k.OpenPalette,
		k.OpenSessionQuickSwitch,
		k.Quit,
	)
	if !paletteOpen {
		base = append(base, nonEmptyBindings(k.RoutePrev, k.RouteNext, k.RouteOpen)...)
		return base
	}
	return append(base, nonEmptyBindings(
		k.PalettePrev,
		k.PaletteNext,
		k.PalettePick,
		k.PaletteClose,
		k.SessionQuickSwitchPrev,
		k.SessionQuickSwitchNext,
		k.SessionQuickSwitchPick,
		k.SessionQuickSwitchClose,
	)...)
}

func shellLeaderStartKeyBinding(configured string) (keyBinding, error) {
	normalized := strings.ToLower(strings.TrimSpace(configured))
	if normalized == "" {
		normalized = "space"
	}
	if unsafe := map[string]struct{}{"enter": {}, "esc": {}, "ctrl+c": {}}; containsKey(unsafe, normalized) {
		return keyBinding{}, fmt.Errorf("unsafe leader key %q is not allowed", normalized)
	}
	allowlist := map[string]keyBinding{
		"space":     {Keys: []string{"space"}, Description: "Enter leader mode"},
		"comma":     {Keys: []string{","}, Description: "Enter leader mode"},
		"backslash": {Keys: []string{"\\"}, Description: "Enter leader mode"},
	}
	binding, ok := allowlist[normalized]
	if !ok {
		return keyBinding{}, fmt.Errorf("leader key %q is not in allowlist", normalized)
	}
	return binding, nil
}

func containsKey(set map[string]struct{}, value string) bool {
	_, ok := set[value]
	return ok
}

func nonEmptyBindings(bindings ...keyBinding) []keyBinding {
	out := make([]keyBinding, 0, len(bindings))
	for _, binding := range bindings {
		if len(binding.Keys) == 0 {
			continue
		}
		out = append(out, binding)
	}
	return out
}
