package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
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
		return m, nil
	}
	m.errText = ""
	m.statusText = ""
	m.sessions = typed.sessions
	m.selected = selectedSessionIndex(m.sessions, typed.activeSessionID)
	m.activeID = typed.activeSessionID
	m.active = typed.detail
	m.syncDetailDocument()
	return m, nil
}

func (m chatRouteModel) applySent(typed chatMessageSentMsg) (routeModel, tea.Cmd) {
	m.sending = false
	if typed.err != nil {
		m.errText = safeUIErrorText(typed.err)
		return m, nil
	}
	m.errText = ""
	m.statusText = "Execution trigger accepted; waiting on broker-owned turn execution state."
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
	if m.active != nil && typed.turnExecution != nil {
		exec := *typed.turnExecution
		m.active.CurrentTurnExecution = &exec
		m.active.LatestTurnExecution = &exec
		state := strings.TrimSpace(exec.ExecutionState)
		if state != "" {
			m.statusText = "Execution progress: " + state
			if wait := strings.TrimSpace(exec.WaitState); wait != "" {
				m.statusText += " (" + wait + ")"
			}
		}
	}
	if typed.sessions != nil {
		m.sessions = typed.sessions
		m.selected = selectedSessionIndex(m.sessions, m.activeID)
	}
	m.syncDetailDocument()
	return m, nil
}

func (m chatRouteModel) reload() (routeModel, tea.Cmd) {
	m.loading = true
	m.errText = ""
	m.statusText = ""
	m.loadSeq++
	return m, m.loadCmd(m.activeID, m.loadSeq)
}

func (m *chatRouteModel) syncDetailDocument() {
	if m.active == nil {
		m.detailDoc.SetDocument(workbenchObjectRef{Kind: "session", ID: "none"}, inspectorContentTranscript, "transcript", "")
		return
	}
	presentation := normalizePresentationMode(m.presentation)
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
		if target != "" && selectedSessionIndex(listResp.Sessions, target) < 0 {
			target = ""
		}
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
		sendResp, err := m.client.SessionExecutionTrigger(ctx, brokerapi.SessionExecutionTriggerRequest{
			SessionID:              sessionID,
			TriggerSource:          "interactive_user",
			RequestedOperation:     "start",
			UserMessageContentText: content,
		})
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		watchEvents, err := m.client.SessionTurnExecutionWatch(ctx, brokerapi.SessionTurnExecutionWatchRequest{SessionID: sessionID, Follow: true, IncludeSnapshot: true})
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		turnExecution := matchingTurnExecutionFromWatch(watchEvents, sendResp.TriggerID)
		getResp, err := m.client.SessionGet(ctx, sessionID)
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		listResp, err := m.client.SessionList(ctx, 20)
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		return chatMessageSentMsg{sessions: listResp.Sessions, detail: &getResp.Session, ack: &sendResp, turnExecution: turnExecution}
	}
}

func matchingTurnExecutionFromWatch(events []brokerapi.SessionTurnExecutionWatchEvent, triggerID string) *brokerapi.SessionTurnExecution {
	for i := range events {
		if events[i].TurnExecution == nil {
			continue
		}
		if strings.TrimSpace(events[i].TurnExecution.TriggerID) != strings.TrimSpace(triggerID) {
			continue
		}
		v := *events[i].TurnExecution
		return &v
	}
	return nil
}
