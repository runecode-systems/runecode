package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestGitRemoteMutationRouteLoadsPreparedReviewState(t *testing.T) {
	model := newGitRemoteMutationRouteModel(routeDefinition{ID: routeGitRemote, Label: "Git Remote"}, &fakeBrokerClient{})
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeGitRemote})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	for _, want := range []string{
		"Git Remote Mutation",
		"Review-centric broker flow over canonical prepare/get/execute contracts",
		"Stable identities: typed_request_hash=",
		"Approval binding:",
		"Fail-closed: execute requires required approval bindings and a broker-issued provider credential lease bound to this prepared mutation.",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q in %q", want, view)
		}
	}
}

func TestGitRemoteMutationRouteExecuteUsesTypedContract(t *testing.T) {
	recording := newRecordingBrokerClient(&fakeBrokerClient{})
	model := newGitRemoteMutationRouteModel(routeDefinition{ID: routeGitRemote, Label: "Git Remote"}, recording)
	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeGitRemote})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("expected issue-execute-lease command")
	}
	updated, cmd = updated.Update(cmd())
	if cmd == nil {
		t.Fatal("expected execute command after lease issuance")
	}
	updated, cmd = updated.Update(cmd())
	if cmd == nil {
		t.Fatal("expected post-execute reload command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "Execute completed") {
		t.Fatalf("expected execute completion status in %q", view)
	}
	assertStringSliceEqual(t, recording.Calls(), []string{"GitRemoteMutationGet", "GitRemoteMutationIssueExecuteLease", "GitRemoteMutationExecute", "GitRemoteMutationGet"})
}

func TestGitRemoteMutationRouteExecuteFailsClosedWithoutApprovalBinding(t *testing.T) {
	model := newGitRemoteMutationRouteModel(routeDefinition{ID: routeGitRemote, Label: "Git Remote"}, &fakeBrokerClient{})
	prepared := fakePreparedGitRemoteMutationState("sha256:" + strings.Repeat("8", 64))
	prepared.RequiredApprovalID = ""
	prepared.RequiredApprovalRequestHash = nil
	prepared.RequiredApprovalDecisionHash = nil
	updated, _ := model.Update(gitRemoteMutationLoadedMsg{resp: brokerapi.GitRemoteMutationGetResponse{Prepared: prepared}, seq: 0})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd != nil {
		t.Fatal("expected execute to remain fail-closed without approval binding")
	}
	view := updated.View(120, 40, focusContent)
	if !strings.Contains(view, "required approval binding is incomplete") {
		t.Fatalf("expected fail-closed status in view, got %q", view)
	}
}
