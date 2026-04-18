package main

import (
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
	return shellKeyMap{
		Quit:                          keyBinding{Keys: []string{"q", "ctrl+c"}, Description: "Quit"},
		CycleFocusNext:                keyBinding{Keys: []string{"tab"}, Description: "Next focus area"},
		CycleFocusPrev:                keyBinding{Keys: []string{"shift+tab"}, Description: "Previous focus area"},
		ToggleSidebar:                 keyBinding{Keys: []string{"s"}, Description: "Toggle sidebar"},
		BackRoute:                     keyBinding{Keys: []string{"b", "alt+left"}, Description: "Back to previous route"},
		CopyIdentity:                  keyBinding{Keys: []string{"y"}, Description: "Copy current identity"},
		CopyRouteAction:               keyBinding{Keys: []string{"Y"}, Description: "Copy next route copy action"},
		ToggleSelectionMode:           keyBinding{Keys: []string{"ctrl+t"}, Description: "Toggle selection mode (mouse capture)"},
		RunCommand:                    keyBinding{Keys: []string{"ctrl+k"}, Description: "Run shell command"},
		CycleTheme:                    keyBinding{Keys: []string{"t"}, Description: "Cycle theme preset"},
		LayoutSidebarWider:            keyBinding{Keys: []string{"]"}, Description: "Widen sidebar pane"},
		LayoutSidebarNarrower:         keyBinding{Keys: []string{"["}, Description: "Narrow sidebar pane"},
		LayoutInspectorWider:          keyBinding{Keys: []string{"}"}, Description: "Widen inspector pane"},
		LayoutInspectorNarrower:       keyBinding{Keys: []string{"{"}, Description: "Narrow inspector pane"},
		LayoutToggleSidebarCollapse:   keyBinding{Keys: []string{"S"}, Description: "Collapse/expand sidebar pane"},
		LayoutToggleInspectorCollapse: keyBinding{Keys: []string{"I"}, Description: "Collapse/expand inspector pane"},
		RouteNext:                     keyBinding{Keys: []string{"j", "down"}, Description: "Move sidebar cursor down"},
		RoutePrev:                     keyBinding{Keys: []string{"k", "up"}, Description: "Move sidebar cursor up"},
		RouteOpen:                     keyBinding{Keys: []string{"enter"}, Description: "Open selected route"},
		SessionOpen:                   keyBinding{Keys: []string{"enter"}, Description: "Open selected session"},
		SessionNext:                   keyBinding{Keys: []string{"j", "down"}, Description: "Move session selection down"},
		SessionPrev:                   keyBinding{Keys: []string{"k", "up"}, Description: "Move session selection up"},
		SessionPin:                    keyBinding{Keys: []string{"p"}, Description: "Pin/unpin selected session"},
		QuickJump:                     keyBinding{Keys: []string{"0-9"}, Description: "Open route by number"},
		OpenPalette:                   keyBinding{Keys: []string{":", "ctrl+p"}, Description: "Open quick jump palette"},
		OpenSessionQuickSwitch:        keyBinding{Keys: []string{"ctrl+j"}, Description: "Open session quick switcher"},
		SessionQuickSwitchClose:       keyBinding{Keys: []string{"esc"}, Description: "Close session quick switcher"},
		SessionQuickSwitchPick:        keyBinding{Keys: []string{"enter"}, Description: "Switch to selected session"},
		SessionQuickSwitchNext:        keyBinding{Keys: []string{"down", "ctrl+n"}, Description: "Next session match"},
		SessionQuickSwitchPrev:        keyBinding{Keys: []string{"up", "ctrl+p"}, Description: "Previous session match"},
		PaletteClose:                  keyBinding{Keys: []string{"esc"}, Description: "Close palette"},
		PalettePick:                   keyBinding{Keys: []string{"enter"}, Description: "Open selected match"},
		PaletteNext:                   keyBinding{Keys: []string{"down", "ctrl+n"}, Description: "Next palette match"},
		PalettePrev:                   keyBinding{Keys: []string{"up", "ctrl+p"}, Description: "Previous palette match"},
		ScrollDown:                    keyBinding{Keys: []string{"pgdown"}, Description: "Scroll content down"},
		ScrollUp:                      keyBinding{Keys: []string{"pgup"}, Description: "Scroll content up"},
	}
}

func (k shellKeyMap) helpBindings(paletteOpen bool) []keyBinding {
	base := []keyBinding{
		k.CycleFocusNext,
		k.CycleFocusPrev,
		k.ToggleSidebar,
		k.BackRoute,
		k.CopyIdentity,
		k.CopyRouteAction,
		k.ToggleSelectionMode,
		k.RunCommand,
		k.CycleTheme,
		k.LayoutSidebarNarrower,
		k.LayoutSidebarWider,
		k.LayoutInspectorNarrower,
		k.LayoutInspectorWider,
		k.LayoutToggleSidebarCollapse,
		k.LayoutToggleInspectorCollapse,
		k.OpenPalette,
		k.OpenSessionQuickSwitch,
		k.QuickJump,
		k.ScrollUp,
		k.ScrollDown,
		k.Quit,
	}
	if !paletteOpen {
		base = append(base, k.RoutePrev, k.RouteNext, k.RouteOpen)
		return base
	}
	return append(base, k.PalettePrev, k.PaletteNext, k.PalettePick, k.PaletteClose, k.SessionQuickSwitchPrev, k.SessionQuickSwitchNext, k.SessionQuickSwitchPick, k.SessionQuickSwitchClose)
}
