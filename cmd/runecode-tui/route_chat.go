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
	loadSeq      uint64
}

func newChatRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return chatRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered}
}

func (m chatRouteModel) ID() routeID { return m.def.ID }

func (m chatRouteModel) Title() string { return m.def.Label }

func (m chatRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		return m.reload()
	case tea.KeyMsg:
		return m.handleKey(typed)
	case chatLoadedMsg:
		return m.applyLoaded(typed)
	case chatMessageSentMsg:
		return m.applySent(typed)
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
		return "Loading chat route from broker session contracts..."
	}
	if m.sending {
		return "Sending message via broker SessionSendMessage..."
	}
	if m.errText != "" {
		return compactLines("Chat", "Load failed: "+m.errText, "Press r to retry.")
	}
	active := "none"
	if m.activeID != "" {
		active = m.activeID
	}
	body := []string{
		sectionTitle("Chat") + " " + focusBadge(focus),
		fmt.Sprintf("Sessions: %d active=%s", len(m.sessions), active),
		fmt.Sprintf("Composer: %s", composerState(m.composeOn)),
		fmt.Sprintf("Presentation mode=%s", normalizePresentationMode(m.presentation)),
		tableHeader("Session list"),
		renderSessionList(m.sessions, m.selected),
		renderComposer(m.composeOn, m.draft),
	}
	if m.statusText != "" {
		body = append(body, "Status: "+m.statusText)
	}
	if m.inspectorOn {
		body = append(body, tableHeader("Inspector")+" "+appTheme.InspectorHint.Render("(linked refs + ordered transcript)"))
		body = append(body, renderSessionInspector(m.active, m.presentation))
	}
	body = append(body, keyHint("Route keys: j/k move, enter load detail, i toggle inspector, c compose, v cycle rendered/raw/structured, r reload"))
	return compactLines(body...)
}

func (m chatRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	if m.composeOn {
		return m.handleComposeKey(key)
	}
	switch key.String() {
	case "r":
		return m.reload()
	case "i":
		m.inspectorOn = !m.inspectorOn
		return m, nil
	case "c":
		if m.activeID == "" {
			m.statusText = "Select a session first before composing."
			return m, nil
		}
		m.composeOn = true
		m.statusText = ""
		return m, nil
	case "v":
		m.presentation = nextPresentationMode(m.presentation)
		return m, nil
	case "j", "down":
		if len(m.sessions) == 0 {
			return m, nil
		}
		m.selected = (m.selected + 1) % len(m.sessions)
		return m, nil
	case "k", "up":
		if len(m.sessions) == 0 {
			return m, nil
		}
		m.selected--
		if m.selected < 0 {
			m.selected = len(m.sessions) - 1
		}
		return m, nil
	case "enter":
		if len(m.sessions) == 0 {
			return m, nil
		}
		m.statusText = ""
		m.loading = true
		m.errText = ""
		m.loadSeq++
		return m, m.loadCmd(m.sessions[m.selected].Identity.SessionID, m.loadSeq)
	default:
		return m, nil
	}
}

func (m chatRouteModel) handleComposeKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.composeOn = false
		m.statusText = "Compose canceled."
		return m, nil
	case "backspace", "ctrl+h":
		if len(m.draft) > 0 {
			runes := []rune(m.draft)
			m.draft = string(runes[:len(runes)-1])
		}
		return m, nil
	case "enter":
		content := strings.TrimSpace(m.draft)
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
		return m, m.sendCmd(m.activeID, content)
	default:
		if key.Type == tea.KeyRunes {
			m.draft += string(key.Runes)
			return m, nil
		}
		return m, nil
	}
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

func renderSessionList(sessions []brokerapi.SessionSummary, selected int) string {
	if len(sessions) == 0 {
		return "  - no sessions"
	}
	line := ""
	for i, s := range sessions {
		marker := " "
		if i == selected {
			marker = ">"
		}
		line += selectedLine(i == selected, fmt.Sprintf("  %s %s %s turns=%d", marker, s.Identity.SessionID, stateBadgeWithLabel("status", s.Status), s.TurnCount)) + "\n"
	}
	return line
}

func renderSessionInspector(detail *brokerapi.SessionDetail, presentation contentPresentationMode) string {
	if detail == nil {
		return "  Select a session and press enter to load transcript."
	}
	presentation = normalizePresentationMode(presentation)
	transcript := renderTranscriptTurns(detail.TranscriptTurns)
	if presentation == presentationRaw {
		transcript = renderTranscriptRaw(detail.TranscriptTurns)
	}
	if presentation == presentationStructured {
		transcript = renderTranscriptStructured(detail.TranscriptTurns)
	}
	body := []string{
		fmt.Sprintf("  Workspace: %s", detail.Summary.Identity.WorkspaceID),
		renderLinkedReferenceLine("  Linked runs", detail.LinkedRunIDs),
		renderLinkedReferenceLine("  Linked approvals", detail.LinkedApprovalIDs),
		renderLinkedReferenceLine("  Linked artifacts", detail.LinkedArtifactDigests),
		renderLinkedReferenceLine("  Linked audit", detail.LinkedAuditRecordDigests),
		"  Transcript:",
		transcript,
	}
	return compactLines(body...)
}
