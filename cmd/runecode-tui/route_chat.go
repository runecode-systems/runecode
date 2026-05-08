package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type chatLoadedMsg struct {
	sessions        []brokerapi.SessionSummary
	detail          *brokerapi.SessionDetail
	runDetail       *brokerapi.RunDetail
	posture         *brokerapi.ProjectSubstratePostureGetResponse
	activeSessionID string
	err             error
	seq             uint64
}

type chatMessageSentMsg struct {
	sessions      []brokerapi.SessionSummary
	detail        *brokerapi.SessionDetail
	runDetail     *brokerapi.RunDetail
	ack           *brokerapi.SessionExecutionTriggerResponse
	turnExecution *brokerapi.SessionTurnExecution
	posture       *brokerapi.ProjectSubstratePostureGetResponse
	err           error
}

type chatExecutionWatchPollMsg struct {
	seq uint64
}

type chatExecutionWatchLoadedMsg struct {
	seq           uint64
	sessionID     string
	triggerID     string
	turnExecution *brokerapi.SessionTurnExecution
	detail        *brokerapi.SessionDetail
	runDetail     *brokerapi.RunDetail
	sessions      []brokerapi.SessionSummary
	posture       *brokerapi.ProjectSubstratePostureGetResponse
	continueWatch bool
	err           error
}

type chatSelectSessionMsg struct {
	SessionID string
}

type chatRouteModel struct {
	def           routeDefinition
	client        localBrokerClient
	loading       bool
	sending       bool
	errText       string
	statusText    string
	sessions      []brokerapi.SessionSummary
	selected      int
	active        *brokerapi.SessionDetail
	activeID      string
	inspectorOn   bool
	composeOn     bool
	presentation  contentPresentationMode
	draft         string
	composer      composeTextarea
	actionText    string
	loadSeq       uint64
	watchSeq      uint64
	watchStreamID string
	watching      bool
	watchSession  string
	watchTrigger  string
	posture       *brokerapi.ProjectSubstratePostureGetResponse
	runDetail     *brokerapi.RunDetail
	detailDoc     longFormDocumentState
}

func newChatRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return chatRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered, composer: newComposeTextarea(), detailDoc: newLongFormDocumentState()}
}

func (m chatRouteModel) ID() routeID { return m.def.ID }

func (m chatRouteModel) Title() string { return m.def.Label }

func (m chatRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case chatLoadedMsg:
		return m.applyLoaded(typed)
	case chatMessageSentMsg:
		return m.applySent(typed)
	case chatSelectSessionMsg:
		return m.handleSessionSelect(typed)
	case chatExecutionWatchPollMsg:
		return m.handleExecutionWatchPoll(typed)
	case chatExecutionWatchLoadedMsg:
		return m.applyExecutionWatchLoaded(typed)
	case routeViewportScrollMsg:
		return m.handleViewportScroll(typed)
	case routeViewportResizeMsg:
		return m.handleViewportResize(typed)
	case routeShellPreferencesMsg:
		return m.handleShellPreferences(typed)
	default:
		return m, nil
	}
}

func (m chatRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Chat", "Loading sessions and workflow evidence...")
	}
	if m.sending {
		return renderStateCard(routeLoadStateLoading, "Chat", "Starting the broker-owned workflow from this session...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Chat", "Load failed: "+m.errText+" (press r to retry)")
	}
	activeLine := activeSessionSummaryLine(m.active)
	if activeLine == "" {
		activeLine = "No active session selected."
	}
	composerSummary := "Composer is ready for a follow-up prompt."
	if !m.composeOn {
		composerSummary = "Composer is closed. Press c when you want to continue the conversation."
	}
	body := []string{
		sectionTitle("Chat") + " " + focusBadge(focus),
		renderStateCardSpec(stateCardSpec{State: routeLoadStateReady, Title: "Active session", Message: activeLine, Reason: fmt.Sprintf("Canonical session directory: %d session(s) available.", len(m.sessions)), NextAction: "Review the session below, or move through the directory to switch context.", ShortcutCue: "j/k move • enter review", RouteCue: "Chat"}),
		renderStateCardSpec(chatExecutionStateCard(m.active, m.posture, m.runDetail)),
		renderStateCardSpec(stateCardSpec{State: routeLoadStateReady, Title: "Composer", Message: fmt.Sprintf("Composer is %s.", composerState(m.composeOn)), Reason: composerSummary, NextAction: "Write a prompt here when you want the broker to start the next workflow turn.", ShortcutCue: "c compose • alt+enter send", RouteCue: "Chat"}),
		renderModeSwitchTabs([]string{string(presentationRendered), string(presentationRaw), string(presentationStructured)}, string(normalizePresentationMode(m.presentation))),
		renderDirectory("Session directory", renderSessionDirectoryItems(m.sessions), m.selected),
		renderComposer(m.composeOn, m.draft, m.composer.View()),
	}
	if m.active == nil {
		body = append(body, muted("Select a canonical session to review the transcript and start the next workflow."))
	}
	if m.statusText != "" {
		body = append(body, "Status: "+m.statusText)
	}
	if strings.TrimSpace(m.actionText) != "" {
		body = append(body, "Follow-up: "+m.actionText)
	}
	body = append(body, muted("Transcript remains the durable conversation record. Inspectors keep raw detail and evidence links available."))
	body = append(body, keyHint("Keys: j/k move • enter review • c compose • alt+enter send • i inspector • v mode • r reload"))
	return compactLines(body...)
}

func (m chatRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	breadcrumbs := []string{"Home", m.def.Label}
	if strings.TrimSpace(m.activeID) != "" {
		breadcrumbs = append(breadcrumbs, m.activeID)
	}
	status := strings.TrimSpace(m.statusText)
	if status == "" && strings.TrimSpace(m.errText) != "" {
		status = "Load failed: " + strings.TrimSpace(m.errText)
	}
	if status == "" && strings.TrimSpace(m.actionText) != "" {
		status = strings.TrimSpace(m.actionText)
	}
	inspector := ""
	if m.inspectorOn {
		inspector = renderSessionInspector(m.active, m.presentation, &m.detailDoc)
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:      routeSurfaceRegion{Title: "Chat workspace", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Inspector: routeSurfaceRegion{Title: "Session inspector", Body: inspector},
			Bottom:    routeSurfaceRegion{Body: keyHint("Keys: j/k move • enter review • c compose • alt+enter send • i inspector • v mode • r reload")},
			Status:    routeSurfaceRegion{Body: status},
		},
		Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: m.inspectorOn}},
		Chrome:       routeSurfaceChrome{Breadcrumbs: breadcrumbs},
		Actions: routeSurfaceActions{
			ModeTabs:         []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
			ActiveTab:        string(normalizePresentationMode(m.presentation)),
			CopyActions:      chatRouteCopyActions(m.active),
			ReferenceActions: chatInspectorReferenceActions(m.active),
			LocalActions:     chatInspectorLocalActions(),
		},
	}
}

func (m chatRouteModel) watchPollCmd(seq uint64, after time.Duration) tea.Cmd {
	if after <= 0 {
		after = 900 * time.Millisecond
	}
	return tea.Tick(after, func(time.Time) tea.Msg {
		return chatExecutionWatchPollMsg{seq: seq}
	})
}

func (m chatRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	if m.composeOn {
		return m.handleComposeKey(key)
	}
	for _, handler := range []func(tea.KeyMsg) (routeModel, tea.Cmd, bool){
		m.handleReloadKey,
		m.handleToggleInspectorKey,
		m.handleComposeToggleKey,
		m.handleCyclePresentationKey,
		m.handleSessionNextKey,
		m.handleSessionPrevKey,
		m.handleSessionOpenKey,
	} {
		if updated, cmd, handled := handler(key); handled {
			return updated, cmd
		}
	}
	return m, nil
}

func (m chatRouteModel) KeyboardOwnership() routeKeyboardOwnership {
	if m.composeOn {
		return routeKeyboardOwnershipTextEntry
	}
	return routeKeyboardOwnershipNormal
}

func (m chatRouteModel) handleReloadKey(key tea.KeyMsg) (routeModel, tea.Cmd, bool) {
	if key.String() != "r" {
		return m, nil, false
	}
	updated, cmd := m.reload()
	return updated, cmd, true
}

func (m chatRouteModel) handleToggleInspectorKey(key tea.KeyMsg) (routeModel, tea.Cmd, bool) {
	if key.String() != "i" {
		return m, nil, false
	}
	m.inspectorOn = !m.inspectorOn
	return m, nil, true
}

func (m chatRouteModel) handleComposeToggleKey(key tea.KeyMsg) (routeModel, tea.Cmd, bool) {
	if key.String() != "c" {
		return m, nil, false
	}
	if m.activeID == "" {
		m.statusText = "Select a session first before composing."
		return m, nil, true
	}
	m.composeOn = true
	m.composer.Focus()
	m.statusText = ""
	return m, nil, true
}

func (m chatRouteModel) handleCyclePresentationKey(key tea.KeyMsg) (routeModel, tea.Cmd, bool) {
	if key.String() != "v" {
		return m, nil, false
	}
	m.presentation = nextPresentationMode(m.presentation)
	m.syncDetailDocument()
	return m, nil, true
}

func (m chatRouteModel) handleSessionNextKey(key tea.KeyMsg) (routeModel, tea.Cmd, bool) {
	if key.String() != "j" && key.String() != "down" {
		return m, nil, false
	}
	if len(m.sessions) == 0 {
		return m, nil, true
	}
	m.selected = (m.selected + 1) % len(m.sessions)
	return m, nil, true
}

func (m chatRouteModel) handleSessionPrevKey(key tea.KeyMsg) (routeModel, tea.Cmd, bool) {
	if key.String() != "k" && key.String() != "up" {
		return m, nil, false
	}
	if len(m.sessions) == 0 {
		return m, nil, true
	}
	m.selected--
	if m.selected < 0 {
		m.selected = len(m.sessions) - 1
	}
	return m, nil, true
}

func (m chatRouteModel) handleSessionOpenKey(key tea.KeyMsg) (routeModel, tea.Cmd, bool) {
	if key.String() != "enter" {
		return m, nil, false
	}
	if len(m.sessions) == 0 {
		return m, nil, true
	}
	m.statusText = ""
	m.loading = true
	m.errText = ""
	m.loadSeq++
	return m, m.loadCmd(m.sessions[m.selected].Identity.SessionID, m.loadSeq), true
}

func (m chatRouteModel) handleComposeKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	if key.String() == "esc" {
		m.composeOn = false
		m.composer.Blur()
		m.statusText = "Compose canceled."
		return m, nil
	}
	if key.Type == tea.KeyEnter && key.Alt {
		content := strings.TrimSpace(m.composer.Value())
		if content == "" {
			m.statusText = "Draft is empty; type a message or press esc."
			return m, nil
		}
		if m.activeID == "" {
			m.statusText = "No active session selected."
			return m, nil
		}
		m.sending = true
		m.errText = ""
		m.statusText = ""
		m.actionText = ""
		m.watching = false
		m.watchSession = ""
		m.watchTrigger = ""
		m.draft = content
		return m, m.sendCmd(m.activeID, content)
	}
	m.composer.BubbleUpdate(key)
	m.draft = m.composer.Value()
	return m, nil
}
