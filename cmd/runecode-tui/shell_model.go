package main

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const (
	shellMediumMinWidth = 90
	shellWideMinWidth   = 130
	overlayIDQuickJump  = "quick-jump"
	overlayIDSessions   = "session-switcher"
	overlayIDSidebar    = "sidebar-drawer"
	overlayIDInspector  = "inspector-sheet"
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
		return "sidebar"
	case focusContent:
		return "main"
	case focusPalette:
		return "overlay"
	default:
		return "unknown"
	}
}

type sessionWorkspaceLoadedMsg struct {
	sessions []brokerapi.SessionSummary
	err      error
}

type shellModel struct {
	quitting bool
	width    int
	height   int

	keys     shellKeyMap
	routes   []routeDefinition
	nav      primaryNavModel
	palette  paletteModel
	sessions sessionSwitcherModel
	focus    focusArea
	client   localBrokerClient

	routeModels map[routeID]routeModel
	currentID   routeID
	backstack   []routeID
	scroll      int

	focusManager   shellFocusManager
	overlayManager shellOverlayManager
	commands       shellCommandRegistry
	clipboard      shellClipboardService
	workbench      shellWorkbenchStateStore
	workbenchScope string
	toasts         shellToastService

	sidebarVisible  bool
	inspectorOn     bool
	themePreset     themePreset
	preferredMode   contentPresentationMode
	sidebarRatio    float64
	inspectorRatio  float64
	sidebarFolded   bool
	inspectorFolded bool
	narrowSidebarOn bool
	narrowInspectOn bool
	overlays        []string

	sessionItems     []brokerapi.SessionSummary
	sessionSelected  int
	activeSessionID  string
	sessionLoadError string
	sessionLoading   bool
	pinnedSessions   map[string]struct{}
	recentSessions   []string
	lastSessionByWS  map[string]string
	recentObjects    []workbenchObjectRef
	sessionWorkspace map[string]string
	viewedActivity   map[string]string
	watchCache       shellLiveActivityCache
	watchHealth      shellSyncHealth
	activity         shellActivitySemantics
	activityFrame    int
	selectionMode    bool
	copyActionIndex  int
}

func newShellModel() shellModel {
	routes := shellRoutes()
	models := newRouteModels(routes)
	defaultRoute := routeChat
	workbench := newDefaultWorkbenchStateStore()
	if strings.HasSuffix(os.Args[0], ".test") {
		workbench = &memoryWorkbenchStateStore{}
	}
	scope := logicalBrokerTargetKey()
	initialState := workbenchLocalState{SidebarVisible: true, InspectorVisible: true, InspectorMode: presentationRendered, ThemePreset: themePresetDark, LastRouteID: defaultRoute, ViewedActivity: map[string]string{}, LastSessionByWS: map[string]string{}, SidebarPaneRatio: 0.22, InspectorPaneRatio: 0.30}
	if existing := workbench.Read(scope); isZeroWorkbenchState(existing) {
		workbench.Write(scope, initialState)
	}
	appTheme = newTheme(themePresetDark)
	m := shellModel{
		keys:             defaultShellKeyMap(),
		routes:           routes,
		nav:              newPrimaryNavModel(routes),
		palette:          newPaletteModel(nil),
		sessions:         newSessionSwitcherModel(),
		focus:            focusNav,
		client:           newLocalBrokerClient(),
		focusManager:     newShellFocusManager(focusNav),
		overlayManager:   shellOverlayManager{},
		commands:         defaultShellCommandRegistry(),
		clipboard:        newShellClipboardService(),
		workbench:        workbench,
		workbenchScope:   scope,
		toasts:           newShellToastService(),
		routeModels:      models,
		currentID:        defaultRoute,
		sidebarVisible:   true,
		inspectorOn:      true,
		themePreset:      themePresetDark,
		preferredMode:    presentationRendered,
		sidebarRatio:     0.22,
		inspectorRatio:   0.30,
		sessionLoading:   true,
		pinnedSessions:   map[string]struct{}{},
		lastSessionByWS:  map[string]string{},
		recentObjects:    nil,
		sessionWorkspace: map[string]string{},
		viewedActivity:   map[string]string{},
		watchHealth:      shellSyncHealth{State: shellSyncStateLoading},
		activity:         shellActivitySemantics{State: shellActivityStateLoading},
	}
	m.restoreWorkbenchState()
	return m
}

func (m shellModel) Init() tea.Cmd {
	return tea.Batch(m.activateCurrentRouteCmd(), m.loadSessionWorkspaceCmd(), m.startWatchPollCmd(), m.mouseCaptureCmd())
}

func (m shellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if updated, cmd, handled := m.handleQuitMessage(msg); handled {
		return updated, cmd
	}
	if updated, cmd, handled := m.handleTextEntryMessage(msg); handled {
		return updated, cmd
	}
	if updated, cmd, handled := m.handleQuitShortcutMessage(msg); handled {
		return updated, cmd
	}
	if updated, cmd, handled := m.handleWindowSize(msg); handled {
		return updated, cmd
	}
	if updated, cmd, handled := m.handleOverlayMessage(msg); handled {
		return updated, cmd
	}
	if updated, cmd, handled := m.handleShellMessage(msg); handled {
		return updated, cmd
	}
	return m.updateActiveRoute(msg)
}

func (m shellModel) activateCurrentRouteCmd() tea.Cmd {
	active := m.currentID
	activeSessionID := m.activeSessionID
	return func() tea.Msg {
		return routeActivatedMsg{RouteID: active, ActiveSessionID: activeSessionID}
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

func (m shellModel) activeShellSurface() routeSurface {
	active := m.routeModels[m.currentID]
	if active == nil {
		return routeSurface{Main: "Route not available", Breadcrumbs: []string{"Home", string(m.currentID)}}
	}
	ctx := routeShellContext{Width: m.width, Height: m.height, Focus: m.focus, Breakpoint: m.breakpoint()}
	return active.ShellSurface(ctx)
}

func (m shellModel) breakpoint() shellBreakpoint {
	return shellBreakpointForWidth(m.width)
}

func shellBreakpointForWidth(width int) shellBreakpoint {
	if width <= 0 {
		return shellBreakpointWide
	}
	if width < shellMediumMinWidth {
		return shellBreakpointNarrow
	}
	if width < shellWideMinWidth {
		return shellBreakpointMedium
	}
	return shellBreakpointWide
}

func (m shellModel) effectiveSidebarVisible() bool {
	if m.breakpoint() == shellBreakpointNarrow {
		return false
	}
	if m.sidebarFolded {
		return false
	}
	return m.sidebarVisible
}

func (m shellModel) navigationSurfaceVisible() bool {
	if m.effectiveSidebarVisible() {
		return true
	}
	return m.breakpoint() == shellBreakpointNarrow && m.narrowSidebarOn
}

func (m shellModel) shouldShowInspector(surface routeSurface) bool {
	if strings.TrimSpace(surface.Inspector) == "" {
		return false
	}
	if !m.inspectorOn || m.inspectorFolded {
		return false
	}
	return m.breakpoint() == shellBreakpointWide
}

func (m shellModel) isTextEntryActive() bool {
	active := m.routeModels[m.currentID]
	chat, ok := active.(chatRouteModel)
	if !ok {
		return false
	}
	return chat.composeOn
}

func (m shellModel) mouseCaptureCmd() tea.Cmd {
	if m.selectionMode {
		return tea.DisableMouse
	}
	return tea.EnableMouseCellMotion
}
