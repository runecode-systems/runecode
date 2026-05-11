package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func (m chatRouteModel) handleRouteActivated(msg routeActivatedMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	if msg.InspectorSet {
		m.inspectorOn = msg.InspectorVisible
	}
	m.presentation = normalizePresentationMode(msg.PreferredMode)
	if strings.TrimSpace(msg.ActiveSessionID) != "" {
		m.activeID = strings.TrimSpace(msg.ActiveSessionID)
	}
	return m.reload()
}

func (m chatRouteModel) handleSessionSelect(msg chatSelectSessionMsg) (routeModel, tea.Cmd) {
	sessionID := strings.TrimSpace(msg.SessionID)
	if sessionID == "" {
		return m, nil
	}
	m.watching = false
	m.watchSession = ""
	m.watchTrigger = ""
	m.actionText = ""
	m.statusText = ""
	m.loading = true
	m.errText = ""
	m.loadSeq++
	return m, m.loadCmd(sessionID, m.loadSeq)
}

func (m chatRouteModel) handleViewportScroll(msg routeViewportScrollMsg) (routeModel, tea.Cmd) {
	if msg.Region == routeRegionInspector {
		m.detailDoc.Scroll(msg.Delta)
	}
	return m, nil
}

func (m chatRouteModel) handleViewportResize(msg routeViewportResizeMsg) (routeModel, tea.Cmd) {
	width, height := longFormViewportSizeForShell(msg.Width, msg.Height)
	m.detailDoc.Resize(width, height)
	return m, nil
}

func (m chatRouteModel) handleShellPreferences(msg routeShellPreferencesMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	m.inspectorOn = msg.InspectorVisible
	m.presentation = normalizePresentationMode(msg.PreferredMode)
	m.syncDetailDocument()
	return m, nil
}

func (m chatRouteModel) applyLoaded(typed chatLoadedMsg) (routeModel, tea.Cmd) {
	if typed.seq != m.loadSeq {
		return m, nil
	}
	m.loading = false
	if typed.err != nil {
		m.errText = safeUIErrorText(typed.err)
		m.statusText = ""
		m.actionText = ""
		return m, nil
	}
	m.errText = ""
	m.statusText = ""
	m.actionText = ""
	m.sessions = typed.sessions
	m.selected = selectedSessionIndex(m.sessions, typed.activeSessionID)
	m.activeID = typed.activeSessionID
	m.active = typed.detail
	m.runDetail = typed.runDetail
	m.posture = typed.posture
	m.watching = false
	m.watchPollCount = 0
	m.watchSession = ""
	m.watchTrigger = ""
	m.syncDetailDocument()
	return m, nil
}

func (m chatRouteModel) applySent(typed chatMessageSentMsg) (routeModel, tea.Cmd) {
	m.sending = false
	if typed.err != nil {
		m.errText = safeUIErrorText(typed.err)
		m.actionText = ""
		return m, nil
	}
	m.resetSendSuccessState()
	m.applySentActiveState(typed)
	m.applySentExecutionState(typed)
	m.applySentSessionList(typed)
	if cmd := m.startSentExecutionWatch(typed); cmd != nil {
		m.syncDetailDocument()
		return m, cmd
	}
	m.syncDetailDocument()
	return m, nil
}

func (m *chatRouteModel) resetSendSuccessState() {
	m.errText = ""
	m.statusText = "Execution trigger accepted; waiting on broker-owned turn execution state."
	m.actionText = ""
	m.draft = ""
	m.composer.SetValue("")
	m.composer.Blur()
	m.composeOn = false
	m.watching = false
	m.watchPollCount = 0
	m.watchSession = ""
	m.watchTrigger = ""
}

func (m *chatRouteModel) applySentActiveState(typed chatMessageSentMsg) {
	if typed.ack != nil {
		m.activeID = typed.ack.SessionID
	}
	if typed.detail != nil {
		m.active = typed.detail
	}
}

func (m *chatRouteModel) applySentExecutionState(typed chatMessageSentMsg) {
	m.runDetail = typed.runDetail
	m.posture = typed.posture
	if m.active != nil && typed.turnExecution != nil {
		exec := *typed.turnExecution
		m.active.CurrentTurnExecution = &exec
		m.active.LatestTurnExecution = &exec
		m.statusText, m.actionText = chatExecutionStatusAndAction(exec)
	}
	if typed.posture != nil && typed.turnExecution != nil {
		m.actionText = chooseActionTextByPosture(m.actionText, *typed.posture, *typed.turnExecution)
	}
	if strings.TrimSpace(m.actionText) == "" && typed.posture != nil && !typed.posture.PostureSummary.NormalOperationAllowed {
		m.actionText = chooseActionTextByPosture(m.actionText, *typed.posture, brokerapi.SessionTurnExecution{WaitKind: "project_blocked"})
	}
}

func (m *chatRouteModel) applySentSessionList(typed chatMessageSentMsg) {
	if typed.sessions != nil {
		m.sessions = typed.sessions
		m.selected = selectedSessionIndex(m.sessions, m.activeID)
	}
}

func (m *chatRouteModel) startSentExecutionWatch(typed chatMessageSentMsg) tea.Cmd {
	if typed.ack == nil {
		return nil
	}
	sessionID := strings.TrimSpace(typed.ack.SessionID)
	triggerID := strings.TrimSpace(typed.ack.TriggerID)
	if sessionID == "" || triggerID == "" {
		return nil
	}
	if typed.turnExecution != nil && chatExecutionTerminal(*typed.turnExecution) {
		return nil
	}
	m.watchSeq++
	m.watching = true
	m.watchPollCount = 0
	m.watchSession = sessionID
	m.watchTrigger = triggerID
	m.watchStreamID = newRequestID("chat-session-turn-watch")
	return m.watchPollCmd(m.watchSeq, 700*time.Millisecond)
}

func (m chatRouteModel) handleExecutionWatchPoll(msg chatExecutionWatchPollMsg) (routeModel, tea.Cmd) {
	if msg.seq != m.watchSeq || !m.watching {
		return m, nil
	}
	sessionID := strings.TrimSpace(m.watchSession)
	triggerID := strings.TrimSpace(m.watchTrigger)
	if sessionID == "" || triggerID == "" {
		m.watching = false
		return m, nil
	}
	nextPoll := m.watchPollCount + 1
	refreshSessions := nextPoll%4 == 0
	return m, m.watchLoadCmd(sessionID, triggerID, msg.seq, refreshSessions)
}

func (m chatRouteModel) applyExecutionWatchLoaded(msg chatExecutionWatchLoadedMsg) (routeModel, tea.Cmd) {
	if msg.seq != m.watchSeq {
		return m, nil
	}
	if msg.err != nil {
		m.watching = false
		m.errText = safeUIErrorText(msg.err)
		return m, nil
	}
	if strings.TrimSpace(msg.sessionID) == "" || strings.TrimSpace(msg.triggerID) == "" {
		m.watching = false
		return m, nil
	}
	m.errText = ""
	m.activeID = strings.TrimSpace(msg.sessionID)
	m.runDetail = msg.runDetail
	m.posture = msg.posture
	m.watchPollCount++
	if msg.detail != nil {
		m.active = msg.detail
	}
	if msg.sessions != nil {
		m.sessions = msg.sessions
		m.selected = selectedSessionIndex(m.sessions, m.activeID)
	}
	if m.active != nil && msg.turnExecution != nil {
		exec := *msg.turnExecution
		m.active.CurrentTurnExecution = &exec
		m.active.LatestTurnExecution = &exec
		m.statusText, m.actionText = chatExecutionStatusAndAction(exec)
		if msg.posture != nil {
			m.actionText = chooseActionTextByPosture(m.actionText, *msg.posture, exec)
		}
	}
	if !msg.continueWatch {
		m.watching = false
		m.watchPollCount = 0
		m.watchSession = ""
		m.watchTrigger = ""
		m.syncDetailDocument()
		return m, nil
	}
	m.syncDetailDocument()
	return m, m.watchPollCmd(msg.seq, 900*time.Millisecond)
}

func (m chatRouteModel) reload() (routeModel, tea.Cmd) {
	m.loading = true
	m.errText = ""
	m.actionText = ""
	m.statusText = ""
	m.watching = false
	m.watchPollCount = 0
	m.watchSession = ""
	m.watchTrigger = ""
	m.loadSeq++
	return m, m.loadCmd(m.activeID, m.loadSeq)
}

func (m *chatRouteModel) syncDetailDocument() {
	if m.active == nil {
		m.detailDocSession = nil
		m.detailDocMode = normalizePresentationMode(m.presentation)
		m.detailDoc.SetDocument(workbenchObjectRef{Kind: "session", ID: "none"}, inspectorContentTranscript, "transcript", "")
		return
	}
	presentation := normalizePresentationMode(m.presentation)
	if m.detailDocSession == m.active && m.detailDocMode == presentation {
		return
	}
	transcript := renderTranscriptTurns(m.active.TranscriptTurns)
	kind := inspectorContentTranscript
	if presentation == presentationRaw {
		transcript = renderTranscriptRaw(m.active.TranscriptTurns)
		kind = inspectorContentRaw
	}
	if presentation == presentationStructured {
		transcript = renderTranscriptStructured(m.active.TranscriptTurns)
		kind = inspectorContentStructured
	}
	summary := m.active.Summary
	ref := workbenchObjectRef{Kind: "session", ID: strings.TrimSpace(summary.Identity.SessionID), WorkspaceID: strings.TrimSpace(summary.Identity.WorkspaceID), SessionID: strings.TrimSpace(summary.Identity.SessionID)}
	m.detailDoc.SetDocument(ref, kind, "transcript", transcript)
	m.detailDocSession = m.active
	m.detailDocMode = presentation
}
