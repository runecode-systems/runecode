package main

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type chatBrokerClientSpy struct {
	fakeBrokerClient
	sentReq *brokerapi.SessionSendMessageRequest
}

func (s *chatBrokerClientSpy) SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, error) {
	_ = ctx
	reqCopy := req
	s.sentReq = &reqCopy
	return brokerapi.SessionSendMessageResponse{
		SessionID: req.SessionID,
		Turn:      brokerapi.SessionTranscriptTurn{TurnID: "turn-send", SessionID: req.SessionID, TurnIndex: 100, Status: "in_progress"},
		Message:   brokerapi.SessionTranscriptMessage{MessageID: "msg-send", TurnID: "turn-send", SessionID: req.SessionID, MessageIndex: 1, Role: req.Role, ContentText: req.ContentText},
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

	turnOnePos := strings.Index(view, "turn[1] turn-1")
	turnTwoPos := strings.Index(view, "turn[2] turn-2")
	if turnOnePos < 0 || turnTwoPos < 0 || turnOnePos > turnTwoPos {
		t.Fatalf("expected ordered transcript turns in view, got %q", view)
	}
	if !strings.Contains(view, "Linked runs: run-1") {
		t.Fatalf("expected linked run reference in view, got %q", view)
	}
	if !strings.Contains(view, "Linked approvals: ap-1") {
		t.Fatalf("expected linked approval reference in view, got %q", view)
	}
	if !strings.Contains(view, "Linked artifacts: sha256:bbbb") {
		t.Fatalf("expected linked artifact reference in view, got %q", view)
	}
	if !strings.Contains(view, "Linked audit: sha256:aaaa") {
		t.Fatalf("expected linked audit reference in view, got %q", view)
	}
	mustContainAll(t, view,
		"Summary:",
		"Identity: session=session-1 workspace=ws-1",
		"Local actions: jump:runs | jump:approvals | jump:artifacts | jump:audit | copy:session_id",
		"Copy actions: session id | workspace id | transcript excerpt | linked references",
		"Long-form transcript:",
	)
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

	if spy.sentReq == nil {
		t.Fatal("expected SessionSendMessage request to be captured")
	}
	if spy.sentReq.SessionID != "session-1" {
		t.Fatalf("expected session-1 send target, got %q", spy.sentReq.SessionID)
	}
	if spy.sentReq.Role != "user" {
		t.Fatalf("expected user role, got %q", spy.sentReq.Role)
	}
	if spy.sentReq.ContentText != "hi" {
		t.Fatalf("expected content hi, got %q", spy.sentReq.ContentText)
	}

	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Status: Message appended to canonical transcript.") {
		t.Fatalf("expected send ack status in view, got %q", view)
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
