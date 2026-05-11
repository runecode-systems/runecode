package main

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func chatMustContainAll(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			t.Fatalf("expected %q in view, got %q", needle, haystack)
		}
	}
}

type chatBrokerClientSpy struct {
	fakeBrokerClient
	sentReq   *brokerapi.SessionExecutionTriggerRequest
	watchReq  *brokerapi.SessionTurnExecutionWatchRequest
	watchResp []brokerapi.SessionTurnExecutionWatchEvent
}

func (s *chatBrokerClientSpy) SessionExecutionTrigger(ctx context.Context, req brokerapi.SessionExecutionTriggerRequest) (brokerapi.SessionExecutionTriggerResponse, error) {
	_ = ctx
	reqCopy := req
	s.sentReq = &reqCopy
	return brokerapi.SessionExecutionTriggerResponse{
		SessionID:              req.SessionID,
		TriggerID:              "trigger-send",
		TriggerSource:          req.TriggerSource,
		RequestedOperation:     req.RequestedOperation,
		UserMessageContentText: req.UserMessageContentText,
	}, nil
}

func (s *chatBrokerClientSpy) SessionTurnExecutionWatch(ctx context.Context, req brokerapi.SessionTurnExecutionWatchRequest) ([]brokerapi.SessionTurnExecutionWatchEvent, error) {
	_ = ctx
	reqCopy := req
	s.watchReq = &reqCopy
	if s.watchResp != nil {
		return s.watchResp, nil
	}
	exec := brokerapi.SessionTurnExecution{TurnID: "turn-1", SessionID: req.SessionID, ExecutionIndex: 1, TriggerID: "trigger-send", TriggerSource: "interactive_user", RequestedOperation: "start", ExecutionState: "running", ApprovalProfile: "moderate", AutonomyPosture: "balanced", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}
	return []brokerapi.SessionTurnExecutionWatchEvent{
		{EventType: "session_turn_execution_watch_snapshot", Seq: 1, TurnExecution: &exec},
		{EventType: "session_turn_execution_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
	}, nil
}

func TestChatRouteKeepsStableActiveSessionIdentityAcrossReload(t *testing.T) {
	model := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, &fakeBrokerClient{})

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeChat})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		t.Fatal("did not expect command on list navigation")
	}
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected load command on enter")
	}
	updated, _ = updated.Update(cmd())

	chat, ok := updated.(chatRouteModel)
	if !ok {
		t.Fatalf("expected chatRouteModel, got %T", updated)
	}
	if chat.activeID != "session-2" {
		t.Fatalf("expected active session-2, got %q", chat.activeID)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected reload command")
	}
	updated, _ = updated.Update(cmd())
	chat, ok = updated.(chatRouteModel)
	if !ok {
		t.Fatalf("expected chatRouteModel, got %T", updated)
	}
	if chat.activeID != "session-2" {
		t.Fatalf("expected stable active session-2 after reload, got %q", chat.activeID)
	}
	if chat.selected != 1 {
		t.Fatalf("expected selected index 1 for session-2, got %d", chat.selected)
	}
}

func TestChatRouteRendersOrderedTranscriptAndLinkedReferences(t *testing.T) {
	model := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, &fakeBrokerClient{})

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeChat})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	surface := updated.ShellSurface(routeShellContext{Width: 120, Height: 40, Focus: focusContent, Breakpoint: shellBreakpointWide})
	inspector := surface.Regions.Inspector.Body

	turnOnePos := strings.Index(inspector, "turn[1] turn-1")
	turnTwoPos := strings.Index(inspector, "turn[2] turn-2")
	if turnOnePos < 0 || turnTwoPos < 0 || turnOnePos > turnTwoPos {
		t.Fatalf("expected ordered transcript turns in inspector, got %q", inspector)
	}
	if !strings.Contains(inspector, "Linked runs: run-1") {
		t.Fatalf("expected linked run reference in inspector, got %q", inspector)
	}
	if !strings.Contains(inspector, "Linked approvals: ap-1") {
		t.Fatalf("expected linked approval reference in inspector, got %q", inspector)
	}
	if !strings.Contains(inspector, "Linked artifacts: sha256:bbbb") {
		t.Fatalf("expected linked artifact reference in inspector, got %q", inspector)
	}
	if !strings.Contains(inspector, "Linked audit: sha256:aaaa") {
		t.Fatalf("expected linked audit reference in inspector, got %q", inspector)
	}
	chatMustContainAll(t, inspector,
		"Summary:",
		"Identity: session=session-1 workspace=ws-1",
		"Local actions: jump:session-run | jump:runs | jump:approvals | jump:artifacts | jump:audit | copy:session_id",
		"Copy actions: session id | workspace id | transcript excerpt | linked references",
		"Long-form transcript:",
	)
	if strings.Contains(view, "Long-form transcript:") {
		t.Fatalf("expected transcript detail to render only in inspector region, got %q", view)
	}
}

func TestChatRouteComposeSendsTypedSessionMessageRequest(t *testing.T) {
	spy := &chatBrokerClientSpy{}
	model := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, spy)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeChat})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	if cmd == nil {
		t.Fatal("expected send command from compose alt+enter")
	}
	updated, _ = updated.Update(cmd())

	assertChatSendAndWatchRequests(t, spy)

	view := updated.View(120, 40, focusContent)
	chatMustContainAll(t, view,
		"Workflow is waiting for approval.",
		"Current broker state: running",
		"Approval is still required before the broker can continue this workflow.",
		"Evidence: 1 linked run • 1 approval • 3 artifacts • 1 audit record",
		"Active session session-1 · workspace ws-1",
		"Session directory",
		"Composer closed. Press c to continue the conversation.",
	)
	if strings.Contains(view, "SessionExecutionTrigger") {
		t.Fatalf("expected product language instead of protocol phrasing in primary view, got %q", view)
	}
	if strings.Contains(view, "Modes:") {
		t.Fatalf("expected no duplicate mode tabs in main chat view, got %q", view)
	}
	if strings.Contains(view, "status=") || strings.Contains(view, "work=") {
		t.Fatalf("expected calmer session directory rows without dump-style labels, got %q", view)
	}
}

func assertChatSendAndWatchRequests(t *testing.T, spy *chatBrokerClientSpy) {
	t.Helper()
	if spy.sentReq == nil {
		t.Fatal("expected SessionExecutionTrigger request to be captured")
	}
	if spy.sentReq.SessionID != "session-1" || spy.sentReq.TriggerSource != "interactive_user" || spy.sentReq.RequestedOperation != "start" || spy.sentReq.UserMessageContentText != "hi" {
		t.Fatalf("unexpected send request: %+v", *spy.sentReq)
	}
	if spy.sentReq.WorkflowRouting == nil || spy.sentReq.WorkflowRouting.WorkflowFamily != "runecontext" || spy.sentReq.WorkflowRouting.WorkflowOperation != "change_draft" {
		t.Fatalf("unexpected workflow routing: %+v", spy.sentReq.WorkflowRouting)
	}
	if spy.watchReq == nil {
		t.Fatal("expected SessionTurnExecutionWatch request to be captured")
	}
	if spy.watchReq.SessionID != "session-1" || !spy.watchReq.IncludeSnapshot || !spy.watchReq.Follow {
		t.Fatalf("unexpected watch request: %+v", *spy.watchReq)
	}
}

func TestChatRouteLoadIncludesExecutionTruthAndEvidenceCounts(t *testing.T) {
	model := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, &fakeBrokerClient{})

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeChat})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(120, 40, focusContent)
	chatMustContainAll(t, view,
		"Current broker state: waiting / wait waiting approval",
		"Broker-known stages: waiting • plan compiled • runner active • checkpoint received • approval required • artifact ready",
		"Evidence: 1 linked run • 1 approval • 3 artifacts • 1 audit record",
	)
}

func TestChatRouteBlockedWorkflowUsesStatusRemediationLanguage(t *testing.T) {
	spy := &chatBrokerClientSpy{}
	blocked := brokerapi.SessionTurnExecution{
		TurnID:             "turn-1",
		SessionID:          "session-1",
		ExecutionIndex:     1,
		TriggerID:          "trigger-send",
		TriggerSource:      "interactive_user",
		RequestedOperation: "start",
		ExecutionState:     "blocked",
		WaitKind:           "project_blocked",
		WaitState:          "waiting_project_blocked",
		ApprovalProfile:    "moderate",
		AutonomyPosture:    "balanced",
		CreatedAt:          "2026-01-01T00:00:00Z",
		UpdatedAt:          "2026-01-01T00:00:00Z",
	}
	spy.watchResp = []brokerapi.SessionTurnExecutionWatchEvent{
		{EventType: "session_turn_execution_watch_snapshot", Seq: 1, TurnExecution: &blocked},
		{EventType: "session_turn_execution_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
	}
	model := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, spy)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeChat})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	if cmd == nil {
		t.Fatal("expected send command from compose alt+enter")
	}
	updated, _ = updated.Update(cmd())

	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Project setup is blocking this workflow. Open Status") {
		t.Fatalf("expected Status remediation in view, got %q", view)
	}
	if !strings.Contains(view, "Follow-up: Project setup is blocking this workflow") || !strings.Contains(view, "Remediation:") {
		t.Fatalf("expected posture guidance in view, got %q", view)
	}
}

func TestChatRouteWatchPollingSkipsRepeatedSessionListRefreshes(t *testing.T) {
	spy := &chatBrokerClientSpy{}
	model := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, spy)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeChat})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	if cmd == nil {
		t.Fatal("expected send command from compose alt+enter")
	}
	updated, cmd = updated.Update(cmd())
	if cmd == nil {
		t.Fatal("expected watch polling command after send")
	}

	chat := updated.(chatRouteModel)
	watchMsg, ok := cmd().(chatExecutionWatchPollMsg)
	if !ok {
		t.Fatalf("expected chatExecutionWatchPollMsg, got %T", cmd())
	}
	updated, cmd = chat.Update(watchMsg)
	if cmd == nil {
		t.Fatal("expected watch load command")
	}
	updated, _ = updated.Update(cmd())
	chat = updated.(chatRouteModel)
	if chat.watching {
		t.Fatal("expected terminal watch envelope to stop polling")
	}
	if chat.watchPollCount != 0 {
		t.Fatalf("watchPollCount = %d, want reset after terminal", chat.watchPollCount)
	}
	if chat.sessions == nil || len(chat.sessions) == 0 {
		t.Fatal("expected sessions retained during watch updates")
	}
}

func TestChatRouteComposeStartsContinuousExecutionWatchPolling(t *testing.T) {
	spy := &chatBrokerClientSpy{}
	model := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, spy)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeChat})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	if cmd == nil {
		t.Fatal("expected send command from compose alt+enter")
	}
	updated, cmd = updated.Update(cmd())
	if cmd == nil {
		t.Fatal("expected watch polling command after send")
	}
	chat := updated.(chatRouteModel)
	if !chat.watching {
		t.Fatal("expected continuous watch polling enabled")
	}
	if chat.watchTrigger != "trigger-send" {
		t.Fatalf("watch trigger = %q, want trigger-send", chat.watchTrigger)
	}
	if chat.watchSession != "session-1" {
		t.Fatalf("watch session = %q, want session-1", chat.watchSession)
	}
}

func TestChatRouteComposeSupportsMultilineBracketedPaste(t *testing.T) {
	model := newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, &fakeBrokerClient{})

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeChat})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("first line\nsecond line"), Paste: true})

	chat := updated.(chatRouteModel)
	if got := chat.composer.Value(); got != "first line\nsecond line" {
		t.Fatalf("expected multiline pasted draft, got %q", got)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected plain enter to remain newline in compose mode")
	}
	chat = updated.(chatRouteModel)
	if !strings.Contains(chat.composer.Value(), "\n") {
		t.Fatalf("expected newline retained in composer, got %q", chat.composer.Value())
	}
}
