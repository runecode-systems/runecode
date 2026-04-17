package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type chatLoadedMsg struct {
	sessions        []brokerapi.SessionSummary
	detail          *brokerapi.SessionDetail
	activeSessionID string
	err             error
	seq             uint64
}

type chatMessageSentMsg struct {
	sessions []brokerapi.SessionSummary
	detail   *brokerapi.SessionDetail
	ack      *brokerapi.SessionSendMessageResponse
	err      error
}

type chatSelectSessionMsg struct {
	SessionID string
}

type chatRouteModel struct {
	def          routeDefinition
	client       localBrokerClient
	loading      bool
	sending      bool
	errText      string
	statusText   string
	sessions     []brokerapi.SessionSummary
	selected     int
	active       *brokerapi.SessionDetail
	activeID     string
	inspectorOn  bool
	composeOn    bool
	presentation contentPresentationMode
	draft        string
	composer     composeTextarea
	loadSeq      uint64
}

func newChatRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return chatRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered, composer: newComposeTextarea()}
}

func (m chatRouteModel) ID() routeID { return m.def.ID }

func (m chatRouteModel) Title() string { return m.def.Label }

func (m chatRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		if strings.TrimSpace(typed.ActiveSessionID) != "" {
			m.activeID = strings.TrimSpace(typed.ActiveSessionID)
		}
		return m.reload()
	case tea.KeyMsg:
		return m.handleKey(typed)
	case chatLoadedMsg:
		return m.applyLoaded(typed)
	case chatMessageSentMsg:
		return m.applySent(typed)
	case chatSelectSessionMsg:
		if strings.TrimSpace(typed.SessionID) == "" {
			return m, nil
		}
		m.statusText = ""
		m.loading = true
		m.errText = ""
		m.loadSeq++
		return m, m.loadCmd(strings.TrimSpace(typed.SessionID), m.loadSeq)
	default:
		return m, nil
	}
}

func (m chatRouteModel) applyLoaded(typed chatLoadedMsg) (routeModel, tea.Cmd) {
	if typed.seq != m.loadSeq {
		return m, nil
	}
	m.loading = false
	if typed.err != nil {
		m.errText = safeUIErrorText(typed.err)
		m.statusText = ""
		return m, nil
	}
	m.errText = ""
	m.statusText = ""
	m.sessions = typed.sessions
	m.selected = selectedSessionIndex(m.sessions, typed.activeSessionID)
	m.activeID = typed.activeSessionID
	m.active = typed.detail
	return m, nil
}

func (m chatRouteModel) applySent(typed chatMessageSentMsg) (routeModel, tea.Cmd) {
	m.sending = false
	if typed.err != nil {
		m.errText = safeUIErrorText(typed.err)
		return m, nil
	}
	m.errText = ""
	m.statusText = "Message appended to canonical transcript."
	m.draft = ""
	m.composer.SetValue("")
	m.composer.Blur()
	m.composeOn = false
	if typed.ack != nil {
		m.activeID = typed.ack.SessionID
	}
	if typed.detail != nil {
		m.active = typed.detail
	}
	if typed.sessions != nil {
		m.sessions = typed.sessions
		m.selected = selectedSessionIndex(m.sessions, m.activeID)
	}
	return m, nil
}

func (m chatRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Chat", "Loading chat route from broker session contracts...")
	}
	if m.sending {
		return renderStateCard(routeLoadStateLoading, "Chat", "Sending message via broker SessionSendMessage...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Chat", "Load failed: "+m.errText+" (press r to retry)")
	}
	active := "none"
	if m.activeID != "" {
		active = m.activeID
	}
	body := []string{
		sectionTitle("Chat") + " " + focusBadge(focus),
		fmt.Sprintf("Sessions: %d active=%s", len(m.sessions), active),
		"Main pane default: one active canonical session",
		fmt.Sprintf("Composer: %s", composerState(m.composeOn)),
		renderModeSwitchTabs([]string{string(presentationRendered), string(presentationRaw), string(presentationStructured)}, string(normalizePresentationMode(m.presentation))),
		renderStateCard(routeLoadStateReady, "Active session", activeSessionSummaryLine(m.active)),
		renderComposer(m.composeOn, m.draft, m.composer.View()),
	}
	if m.statusText != "" {
		body = append(body, "Status: "+m.statusText)
	}
	if m.inspectorOn {
		body = append(body, renderSessionInspector(m.active, m.presentation))
	}
	body = append(body, keyHint("Route keys: j/k move, enter load detail, i toggle inspector, c compose, ctrl+enter send, enter newline, v cycle rendered/raw/structured, r reload"))
	return compactLines(body...)
}

func (m chatRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	breadcrumbs := []string{"Home", m.def.Label}
	if strings.TrimSpace(m.activeID) != "" {
		breadcrumbs = append(breadcrumbs, m.activeID)
	}
	status := strings.TrimSpace(m.statusText)
	if status == "" && strings.TrimSpace(m.errText) != "" {
		status = "Load failed: " + strings.TrimSpace(m.errText)
	}
	return routeSurface{
		Main:           m.View(ctx.Width, ctx.Height, ctx.Focus),
		Inspector:      renderSessionInspector(m.active, m.presentation),
		BottomStrip:    keyHint("Route keys: j/k move, enter load detail, i toggle inspector, c compose, ctrl+enter send, enter newline, v cycle rendered/raw/structured, r reload"),
		Status:         status,
		Breadcrumbs:    breadcrumbs,
		MainTitle:      "Chat workspace",
		InspectorTitle: "Session inspector",
		ModeTabs:       []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveTab:      string(normalizePresentationMode(m.presentation)),
		CopyActions:    chatRouteCopyActions(m.active),
	}
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
	if key.Type == tea.KeyEnter && (key.Alt || key.String() == "ctrl+enter") {
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
		m.draft = content
		return m, m.sendCmd(m.activeID, content)
	}
	m.composer.BubbleUpdate(key)
	m.draft = m.composer.Value()
	return m, nil
}

func (m chatRouteModel) reload() (routeModel, tea.Cmd) {
	m.loading = true
	m.errText = ""
	m.statusText = ""
	m.loadSeq++
	return m, m.loadCmd(m.activeID, m.loadSeq)
}

func (m chatRouteModel) loadCmd(preferredSessionID string, seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		listResp, err := m.client.SessionList(ctx, 20)
		if err != nil {
			return chatLoadedMsg{err: err, seq: seq}
		}
		target := preferredSessionID
		if target == "" && len(listResp.Sessions) > 0 {
			target = listResp.Sessions[0].Identity.SessionID
		}
		if target == "" {
			return chatLoadedMsg{sessions: listResp.Sessions, activeSessionID: "", seq: seq}
		}
		getResp, err := m.client.SessionGet(ctx, target)
		if err != nil {
			return chatLoadedMsg{err: err, seq: seq}
		}
		return chatLoadedMsg{sessions: listResp.Sessions, detail: &getResp.Session, activeSessionID: target, seq: seq}
	}
}

func (m chatRouteModel) sendCmd(sessionID, content string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		sendResp, err := m.client.SessionSendMessage(ctx, brokerapi.SessionSendMessageRequest{
			SessionID:   sessionID,
			Role:        "user",
			ContentText: content,
		})
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		getResp, err := m.client.SessionGet(ctx, sessionID)
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		listResp, err := m.client.SessionList(ctx, 20)
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		return chatMessageSentMsg{sessions: listResp.Sessions, detail: &getResp.Session, ack: &sendResp}
	}
}
