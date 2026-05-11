package main

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func (m chatRouteModel) loadCmd(preferredSessionID string, seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		return m.loadChatInitialState(ctx, preferredSessionID, seq)
	}
}

func (m chatRouteModel) loadChatInitialState(ctx context.Context, preferredSessionID string, seq uint64) tea.Msg {
	listResp, err := m.client.SessionList(ctx, 20)
	if err != nil {
		return chatLoadedMsg{err: err, seq: seq}
	}
	target := selectedChatSessionID(listResp.Sessions, preferredSessionID)
	if target == "" {
		return chatLoadedMsg{sessions: listResp.Sessions, activeSessionID: "", seq: seq}
	}
	state, err := m.loadChatSessionState(ctx, target)
	if err != nil {
		return chatLoadedMsg{err: err, seq: seq}
	}
	state.sessions = listResp.Sessions
	return state.loadedMsg(seq)
}

func selectedChatSessionID(sessions []brokerapi.SessionSummary, preferredSessionID string) string {
	target := strings.TrimSpace(preferredSessionID)
	if target != "" && selectedSessionIndex(sessions, target) >= 0 {
		return target
	}
	if len(sessions) == 0 {
		return ""
	}
	return sessions[0].Identity.SessionID
}

type chatSessionLoadState struct {
	sessions  []brokerapi.SessionSummary
	detail    *brokerapi.SessionDetail
	runDetail *brokerapi.RunDetail
	posture   *brokerapi.ProjectSubstratePostureGetResponse
	activeID  string
}

func (s chatSessionLoadState) loadedMsg(seq uint64) chatLoadedMsg {
	return chatLoadedMsg{sessions: s.sessions, detail: s.detail, runDetail: s.runDetail, posture: s.posture, activeSessionID: s.activeID, seq: seq}
}

func (m chatRouteModel) loadChatSessionState(ctx context.Context, sessionID string) (chatSessionLoadState, error) {
	getResp, err := m.client.SessionGet(ctx, sessionID)
	if err != nil {
		return chatSessionLoadState{}, err
	}
	posture, err := m.client.ProjectSubstratePostureGet(ctx)
	if err != nil {
		return chatSessionLoadState{}, err
	}
	runDetail, err := m.loadChatRunDetail(ctx, getResp.Session)
	if err != nil {
		return chatSessionLoadState{}, err
	}
	return chatSessionLoadState{detail: &getResp.Session, runDetail: runDetail, posture: &posture, activeID: sessionID}, nil
}

func (m chatRouteModel) sendCmd(sessionID, content string) tea.Cmd {
	m.watchStreamID = newRequestID("chat-session-turn-watch")
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		sendResp, err := m.client.SessionExecutionTrigger(ctx, brokerapi.SessionExecutionTriggerRequest{SessionID: sessionID, TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: defaultSessionWorkflowRouting(), UserMessageContentText: content})
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		watchEvents, err := m.client.SessionTurnExecutionWatch(ctx, brokerapi.SessionTurnExecutionWatchRequest{StreamID: m.watchStreamID, SessionID: sessionID, Follow: true, IncludeSnapshot: true})
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		turnExecution := matchingTurnExecutionFromWatch(watchEvents, sendResp.TriggerID)
		state, err := m.loadChatSessionState(ctx, sessionID)
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		listResp, err := m.client.SessionList(ctx, 20)
		if err != nil {
			return chatMessageSentMsg{err: err}
		}
		return chatMessageSentMsg{sessions: listResp.Sessions, detail: state.detail, ack: &sendResp, turnExecution: turnExecution, posture: state.posture, runDetail: state.runDetail}
	}
}

func defaultSessionWorkflowRouting() *brokerapi.SessionWorkflowPackRouting {
	return &brokerapi.SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "change_draft"}
}

func (m chatRouteModel) watchLoadCmd(sessionID, triggerID string, seq uint64, refreshSessions bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		return m.loadChatWatchState(ctx, sessionID, triggerID, seq, refreshSessions)
	}
}

func (m chatRouteModel) loadChatWatchState(ctx context.Context, sessionID, triggerID string, seq uint64, refreshSessions bool) tea.Msg {
	watchEvents, err := m.client.SessionTurnExecutionWatch(ctx, brokerapi.SessionTurnExecutionWatchRequest{StreamID: m.watchStreamID, SessionID: sessionID, Follow: true, IncludeSnapshot: true})
	if err != nil {
		return chatExecutionWatchLoadedMsg{seq: seq, err: err}
	}
	turnExecution, terminalSeen := watchExecutionState(watchEvents, triggerID)
	state, err := m.loadChatSessionState(ctx, sessionID)
	if err != nil {
		return chatExecutionWatchLoadedMsg{seq: seq, err: err}
	}
	continueWatch := chatShouldContinueWatch(turnExecution, terminalSeen)
	if !continueWatch || refreshSessions {
		listResp, err := m.client.SessionList(ctx, 20)
		if err != nil {
			return chatExecutionWatchLoadedMsg{seq: seq, err: err}
		}
		state.sessions = listResp.Sessions
	}
	return chatExecutionWatchLoadedMsg{seq: seq, sessionID: sessionID, triggerID: triggerID, turnExecution: turnExecution, detail: state.detail, runDetail: state.runDetail, sessions: state.sessions, posture: state.posture, continueWatch: continueWatch}
}

func chatShouldContinueWatch(turnExecution *brokerapi.SessionTurnExecution, terminalSeen bool) bool {
	if terminalSeen {
		return false
	}
	if turnExecution == nil {
		return true
	}
	return !chatExecutionTerminal(*turnExecution)
}

func (m chatRouteModel) loadChatRunDetail(ctx context.Context, detail brokerapi.SessionDetail) (*brokerapi.RunDetail, error) {
	runID := ""
	if exec := chatVisibleExecution(&detail); exec != nil {
		runID = chatRunIDFromExecution(*exec)
	}
	if strings.TrimSpace(runID) == "" && len(detail.LinkedRunIDs) > 0 {
		runID = strings.TrimSpace(detail.LinkedRunIDs[0])
	}
	if strings.TrimSpace(runID) == "" {
		return nil, nil
	}
	resp, err := m.client.RunGet(ctx, runID)
	if err != nil {
		return nil, err
	}
	run := resp.Run
	return &run, nil
}

func matchingTurnExecutionFromWatch(events []brokerapi.SessionTurnExecutionWatchEvent, triggerID string) *brokerapi.SessionTurnExecution {
	turnExecution, _ := watchExecutionState(events, triggerID)
	return turnExecution
}

func watchExecutionState(events []brokerapi.SessionTurnExecutionWatchEvent, triggerID string) (*brokerapi.SessionTurnExecution, bool) {
	terminalSeen := false
	for i := len(events) - 1; i >= 0; i-- {
		terminalSeen = terminalSeen || events[i].Terminal
		if events[i].TurnExecution == nil {
			continue
		}
		if strings.TrimSpace(events[i].TurnExecution.TriggerID) != strings.TrimSpace(triggerID) {
			continue
		}
		v := *events[i].TurnExecution
		return &v, terminalSeen
	}
	return nil, terminalSeen
}
