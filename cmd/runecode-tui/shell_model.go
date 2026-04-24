package main

import (
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const (
	shellMediumMinWidth    = 90
	shellWideMinWidth      = 130
	emergencyQuitArmWindow = 1500 * time.Millisecond
)

type shellOverlayID string

const (
	overlayIDQuickJump   shellOverlayID = "quick-jump"
	overlayIDSessions    shellOverlayID = "session-switcher"
	overlayIDSidebar     shellOverlayID = "sidebar-drawer"
	overlayIDInspector   shellOverlayID = "inspector-sheet"
	overlayIDLeader      shellOverlayID = "leader-which-key"
	overlayIDQuitConfirm shellOverlayID = "quit-confirm"
)

type focusArea int

const (
	focusNav focusArea = iota
	focusContent
	focusInspector
	focusPalette
)

func (f focusArea) Label() string {
	switch f {
	case focusNav:
		return "sidebar"
	case focusContent:
		return "main"
	case focusInspector:
		return "inspector"
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

type shellEmergencyQuitTimeoutMsg struct {
	token uint64
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
	location    shellWorkbenchLocation
	history     []shellWorkbenchLocation

	focusManager   shellFocusManager
	overlayManager shellOverlayManager
	commands       shellCommandRegistry
	actions        shellActionGraph
	clipboard      shellClipboardService
	workbench      shellWorkbenchStateStore
	workbenchScope string
	toasts         shellToastService
	objectIndex    shellDiscoverabilityIndex

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
	overlays        []shellOverlayID
	overlayReturn   focusArea

	sessionItems     []brokerapi.SessionSummary
	sidebarCursor    int
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
	watch            shellWatchManager
	activityFrame    int
	selectionMode    bool
	copyActionIndex  int

	leader           shellLeaderState
	leaderKeyConfig  string
	leaderKeyInvalid string
	commandMode      shellCommandModeState
	emergencyQuit    shellEmergencyQuitState
	quitConfirm      shellQuitConfirmState
}

type shellEmergencyQuitState struct {
	pending bool
	token   uint64
}

type shellQuitConfirmState struct {
	active bool
	reason string
}

func newShellModel() shellModel {
	routes := shellRoutes()
	models := newRouteModels(routes)
	defaultRoute := routeChat
	commands := defaultShellCommandRegistry()
	actions := newShellActionGraph(routes, commands)
	workbench := newDefaultWorkbenchStateStore()
	binaryPath := strings.ToLower(strings.TrimSpace(os.Args[0]))
	if strings.HasSuffix(binaryPath, ".test") || strings.HasSuffix(binaryPath, ".test.exe") {
		workbench = &memoryWorkbenchStateStore{}
	}
	scope := logicalBrokerTargetKey()
	initialState := workbenchLocalState{SidebarVisible: true, InspectorVisible: true, InspectorMode: presentationRendered, ThemePreset: themePresetDark, LastRouteID: defaultRoute, ViewedActivity: map[string]string{}, LastSessionByWS: map[string]string{}, SidebarPaneRatio: 0.22, InspectorPaneRatio: 0.30}
	if existing := workbench.Read(scope); isZeroWorkbenchState(existing) {
		workbench.Write(scope, initialState)
	}
	appTheme = newTheme(themePresetDark)
	m := shellModel{
		keys:           defaultShellKeyMap(),
		routes:         routes,
		nav:            newPrimaryNavModel(routes),
		palette:        newPaletteModel(nil),
		sessions:       newSessionSwitcherModel(),
		focus:          focusNav,
		client:         newLocalBrokerClient(),
		focusManager:   newShellFocusManager(focusNav),
		overlayManager: shellOverlayManager{},
		commands:       commands,
		actions:        actions,
		clipboard:      newShellClipboardService(),
		workbench:      workbench,
		workbenchScope: scope,
		toasts:         newShellToastService(),
		routeModels:    models,
		location: shellWorkbenchLocation{
			Primary: shellObjectLocation{RouteID: defaultRoute, Object: workbenchObjectRef{Kind: "route", ID: string(defaultRoute)}},
		},
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
		watch:            newShellWatchManager(),
		objectIndex:      newShellDiscoverabilityIndex(routes),
		overlayReturn:    focusContent,
		leader:           newShellLeaderState(actions.leaderBindings(shellModel{})),
		leaderKeyConfig:  "space",
	}
	_ = m.setLeaderKey(m.leaderKeyConfig)
	m.restoreWorkbenchState()
	m.syncSidebarCursorToLocation()
	return m
}

func (m shellModel) Init() tea.Cmd {
	return tea.Batch(m.activateCurrentRouteCmd(), m.loadSessionWorkspaceCmd(), m.loadObjectIndexCmd(), m.startWatchPollCmd(), m.mouseCaptureCmd())
}

func (m shellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd, handled := m.handleQuitMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleKeyboardOwnershipMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleQuitShortcutMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleWindowSize(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleOverlayMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleShellMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	return m.updateActiveRoute(msg)
}

func (m shellModel) activateCurrentRouteCmd() tea.Cmd {
	active := m.currentRouteID()
	activeSessionID := m.activeSessionID
	inspectorVisible := m.inspectorOn
	preferredMode := normalizePresentationMode(m.preferredMode)
	return func() tea.Msg {
		return routeActivatedMsg{RouteID: active, ActiveSessionID: activeSessionID, InspectorVisible: inspectorVisible, InspectorSet: true, PreferredMode: preferredMode}
	}
}

func (m shellModel) updateActiveRoute(msg tea.Msg) (tea.Model, tea.Cmd) {
	active := m.routeModels[m.currentRouteID()]
	if active == nil {
		return m, nil
	}
	updated, cmd := active.Update(msg)
	m.routeModels[m.currentRouteID()] = updated
	return m, cmd
}

func (m *shellModel) publishShellPreferencesToCurrentRoute() {
	activeID := m.currentRouteID()
	active := m.routeModels[activeID]
	if active == nil {
		return
	}
	updated, _ := active.Update(routeShellPreferencesMsg{RouteID: activeID, InspectorVisible: m.inspectorOn, PreferredMode: normalizePresentationMode(m.preferredMode)})
	m.routeModels[activeID] = updated
}

func (m shellModel) activeShellSurface() routeSurface {
	active := m.routeModels[m.currentRouteID()]
	if active == nil {
		return routeSurface{
			Regions: routeSurfaceRegions{
				Main: routeSurfaceRegion{Body: "Route not available"},
			},
			Capabilities: routeSurfaceCapabilities{},
			Chrome:       routeSurfaceChrome{Breadcrumbs: []string{"Home", string(m.currentRouteID())}},
		}
	}
	baseCtx := routeShellContext{Width: m.width, Height: m.availableShellHeight(), Focus: m.focus, Focused: m.focusedRouteRegion(), Breakpoint: m.breakpoint(), Render: routeShellRenderPreferences{PreferredPresentation: normalizePresentationMode(m.preferredMode), ThemePreset: normalizeThemePreset(m.themePreset)}}
	surface := active.ShellSurface(baseCtx)
	layout := m.planShellLayout(surface)
	ctx := baseCtx
	ctx.Regions = layout.Regions
	ctx.Breakpoint = layout.Breakpoint
	surface = active.ShellSurface(ctx)
	return m.withLocationChrome(surface)
}

func (m shellModel) availableShellHeight() int {
	_, viewportHeight := normalizedShellViewport(m.width, m.height)
	if viewportHeight <= 0 {
		return viewportHeight
	}
	if overlayHeight := m.activeOverlayHeight(viewportHeight); overlayHeight > 0 {
		available := viewportHeight - overlayHeight
		if available < 1 {
			return 1
		}
		return available
	}
	return viewportHeight
}

func (m shellModel) focusedRouteRegion() routeRegionFocus {
	if m.palette.IsOpen() || m.sessions.IsOpen() || m.leader.Active() || m.quitConfirm.active {
		return routeRegionOverlay
	}
	if m.narrowInspectOn {
		return routeRegionInspector
	}
	if m.focus == focusInspector {
		return routeRegionInspector
	}
	return routeRegionMain
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
	return m.planShellLayout(surface).InspectorVisible
}

func (m shellModel) mouseCaptureCmd() tea.Cmd {
	if m.selectionMode {
		return tea.DisableMouse
	}
	return tea.EnableMouseCellMotion
}
