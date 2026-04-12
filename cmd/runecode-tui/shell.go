package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	wideTerminalWidth = 100
	navLinePrefix     = "Primary navigation: "
)

type focusArea int

const (
	focusNav focusArea = iota
	focusContent
	focusPalette
)

func (f focusArea) Label() string {
	switch f {
	case focusNav:
		return "primary-nav"
	case focusContent:
		return "content"
	case focusPalette:
		return "palette"
	default:
		return "unknown"
	}
}

type routeSwitchMsg struct {
	RouteID routeID
}

type shellModel struct {
	quitting bool
	width    int
	height   int

	keys    shellKeyMap
	routes  []routeDefinition
	nav     primaryNavModel
	palette paletteModel
	focus   focusArea

	routeModels map[routeID]routeModel
	currentID   routeID
	scroll      int
}

func newShellModel() shellModel {
	routes := shellRoutes()
	models := newRouteModels(routes)
	defaultRoute := routeDashboard
	return shellModel{
		keys:        defaultShellKeyMap(),
		routes:      routes,
		nav:         newPrimaryNavModel(routes),
		palette:     newPaletteModel(routes),
		focus:       focusNav,
		routeModels: models,
		currentID:   defaultRoute,
	}
}

func (m shellModel) Init() tea.Cmd {
	return func() tea.Msg {
		return routeActivatedMsg{RouteID: m.currentID}
	}
}

func (m shellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && m.keys.Quit.matches(key) {
		m.quitting = true
		return m, tea.Quit
	}

	if typed, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = typed.Width
		m.height = typed.Height
		if m.palette.IsOpen() {
			return m, nil
		}
	}

	if m.palette.IsOpen() {
		switch typed := msg.(type) {
		case tea.MouseMsg:
			updatedPalette, routeMsg, changed := m.palette.UpdateMouse(typed, m.paletteStartY(), 0)
			m.palette = updatedPalette
			if !m.palette.IsOpen() && m.focus == focusPalette {
				m.focus = focusNav
			}
			if changed {
				return m, func() tea.Msg { return routeMsg }
			}
			return m.updateActiveRoute(msg)
		case tea.KeyMsg:
			updatedPalette, routeMsg, changed := m.palette.Update(msg, m.keys)
			m.palette = updatedPalette
			if !m.palette.IsOpen() && m.focus == focusPalette {
				m.focus = focusNav
			}
			if changed {
				return m, func() tea.Msg { return routeMsg }
			}
			return m, nil
		default:
			return m.updateActiveRoute(msg)
		}
	}

	switch typed := msg.(type) {
	case routeSwitchMsg:
		m.currentID = typed.RouteID
		m.nav.SelectByRouteID(typed.RouteID)
		m.focus = focusContent
		return m, m.activateCurrentRouteCmd()
	case tea.MouseMsg:
		return m.handleMouse(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	}

	return m.updateActiveRoute(msg)
}

func (m shellModel) View() string {
	if m.quitting {
		return "Goodbye from runecode-tui.\n"
	}

	b := strings.Builder{}
	for _, line := range m.headerLines() {
		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.palette.IsOpen() {
		b.WriteString(m.renderPalette())
		b.WriteString("\n")
	} else {
		b.WriteString(m.renderActiveRoute())
		b.WriteString("\n")
	}

	b.WriteString(renderHelp(m.keys, m.palette.IsOpen()))
	b.WriteString("\n")
	b.WriteString(muted("Mouse: click nav to open route, click content to focus, wheel to scroll"))
	b.WriteString("\n")
	b.WriteString(muted("Keyboard equivalents: 1-7/open palette/enter/tab/j/k"))
	b.WriteString("\n")
	b.WriteString(muted(localBrokerBoundaryPosture()))
	b.WriteString("\n")
	b.WriteString(muted("Trust boundary: typed broker contracts only; no CLI scraping or daemon-private path modeling."))
	b.WriteString("\n")
	return b.String()
}

func (m shellModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.keys.OpenPalette.matches(key) {
		m.palette = m.palette.Open()
		m.focus = focusPalette
		return m, nil
	}
	if route, ok := routeByQuickJumpKey(key.String(), m.routes); ok {
		m.currentID = route.ID
		m.nav.SelectByRouteID(route.ID)
		m.focus = focusContent
		return m, m.activateCurrentRouteCmd()
	}
	if m.keys.CycleFocusNext.matches(key) {
		m.focus = m.nextFocus()
		return m, nil
	}
	if m.keys.CycleFocusPrev.matches(key) {
		m.focus = m.prevFocus()
		return m, nil
	}

	if m.focus == focusNav {
		if m.keys.RouteNext.matches(key) {
			m.nav.MoveNext()
			return m, nil
		}
		if m.keys.RoutePrev.matches(key) {
			m.nav.MovePrev()
			return m, nil
		}
		if m.keys.RouteOpen.matches(key) {
			route := m.nav.Selected()
			m.currentID = route.ID
			m.focus = focusContent
			return m, m.activateCurrentRouteCmd()
		}
	}

	if m.keys.ScrollDown.matches(key) {
		m.scroll++
		return m, nil
	}
	if m.keys.ScrollUp.matches(key) {
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil
	}

	return m.updateActiveRoute(key)
}

func (m shellModel) handleMouse(mouse tea.MouseMsg) (tea.Model, tea.Cmd) {
	if mouse.Button == tea.MouseButtonLeft && (mouse.Action == tea.MouseActionPress || mouse.Action == tea.MouseActionRelease) {
		if navX, ok := m.navOffsetAtMouse(mouse.X, mouse.Y); ok {
			_, boxes := m.nav.Render(m.width >= wideTerminalWidth)
			if routeID, ok := navRouteAtX(boxes, navX); ok {
				m.currentID = routeID
				m.nav.SelectByRouteID(routeID)
				m.focus = focusContent
				return m, m.activateCurrentRouteCmd()
			}
			m.focus = focusNav
			return m, nil
		}
		m.focus = focusContent
		return m, nil
	}

	if mouse.Button == tea.MouseButtonWheelUp || mouse.Button == tea.MouseButtonWheelDown {
		switch mouse.Button {
		case tea.MouseButtonWheelUp:
			if m.scroll > 0 {
				m.scroll--
			}
		case tea.MouseButtonWheelDown:
			m.scroll++
		}
		m.focus = focusContent
		return m, nil
	}

	return m, nil
}

func (m shellModel) navYRange() (startY int, endY int) {
	return 1, 1
}

func (m shellModel) paletteStartY() int {
	return len(m.headerLines())
}

func (m shellModel) navOffsetAtMouse(mouseX int, mouseY int) (int, bool) {
	startY, endY := m.navYRange()
	if mouseY < startY || mouseY > endY {
		return 0, false
	}
	if mouseX < len(navLinePrefix) {
		return 0, false
	}
	return mouseX - len(navLinePrefix), true
}

func (m shellModel) headerLines() []string {
	wide := m.width >= wideTerminalWidth
	navLine, _ := m.nav.Render(wide)
	return []string{
		appTheme.AppTitle.Render("Runecode TUI α shell") + " " + neutralBadge("THEME "+string(themePresetDark)),
		navLinePrefix + navLine,
		appTheme.FocusLine.Render("Focus: ") + focusBadge(m.focus),
	}
}

func (m shellModel) renderActiveRoute() string {
	active := m.routeModels[m.currentID]
	if active == nil {
		return "Route not available"
	}
	body := active.View(m.width, m.height, m.focus)
	return fmt.Sprintf("Route: %s\nScroll offset: %d\n\n%s", active.Title(), m.scroll, body)
}

func (m shellModel) renderPalette() string {
	b := strings.Builder{}
	b.WriteString("Quick Jump Palette (: / ctrl+p)\n")
	b.WriteString(fmt.Sprintf("Query: %q\n", m.palette.query))
	if len(m.palette.matches) == 0 {
		b.WriteString("No matches. Press esc to close.\n")
		return b.String()
	}
	b.WriteString("Matches:\n")
	for i, route := range m.palette.matches {
		b.WriteString(paletteMatchLine(route, i == m.palette.selectedIndex))
		b.WriteString("\n")
	}
	return b.String()
}

func (m shellModel) nextFocus() focusArea {
	if m.palette.IsOpen() {
		return focusPalette
	}
	if m.focus == focusNav {
		return focusContent
	}
	return focusNav
}

func (m shellModel) prevFocus() focusArea {
	if m.palette.IsOpen() {
		return focusPalette
	}
	if m.focus == focusContent {
		return focusNav
	}
	return focusContent
}

func (m shellModel) activateCurrentRouteCmd() tea.Cmd {
	active := m.currentID
	return func() tea.Msg {
		return routeActivatedMsg{RouteID: active}
	}
}

func (m shellModel) updateActiveRoute(msg tea.Msg) (tea.Model, tea.Cmd) {
	active := m.routeModels[m.currentID]
	if active == nil {
		return m, nil
	}
	updated, cmd := active.Update(msg)
	m.routeModels[m.currentID] = updated
	return m, cmd
}
