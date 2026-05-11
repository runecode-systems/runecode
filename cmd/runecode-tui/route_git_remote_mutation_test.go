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
		"Guarded remote review",
		"Planned change:",
		"Target:",
		"Approval check:",
		"Execution access:",
		"Next safe action:",
		"Safety: RuneCode keeps execution blocked",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q in %q", want, view)
		}
	}
	for _, unwanted := range []string{
		"Structured/raw detail can show prepared=",
		"Route keys: r reload prepared state, e execute prepared mutation",
		"repository=",
		"approval sha256:",
	} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("view unexpectedly contained %q in %q", unwanted, view)
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
	if !strings.Contains(view, "Remote execution completed") {
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

func TestGitRemoteMutationRouteClearsLeaseWhenReloadDropsApprovalBinding(t *testing.T) {
	prepared := fakePreparedGitRemoteMutationState("sha256:" + strings.Repeat("8", 64))
	refreshed := prepared
	refreshed.RequiredApprovalID = ""
	refreshed.RequiredApprovalRequestHash = nil
	refreshed.RequiredApprovalDecisionHash = nil
	model := gitRemoteMutationRouteModel{def: routeDefinition{ID: routeGitRemote, Label: "Git Remote"}, prepared: prepared, providerAuthLeaseID: "lease-git-provider", loadSeq: 1}

	updated, _ := model.Update(gitRemoteMutationLoadedMsg{resp: brokerapi.GitRemoteMutationGetResponse{Prepared: refreshed}, seq: 1})
	shell := updated.(gitRemoteMutationRouteModel)
	if shell.providerAuthLeaseID != "" {
		t.Fatalf("expected lease cleared when refreshed approval binding is incomplete, got %q", shell.providerAuthLeaseID)
	}
	view := shell.View(120, 40, focusContent)
	if strings.Contains(view, "Execution access is ready") || strings.Contains(view, "Press e to execute") {
		t.Fatalf("expected refreshed incomplete binding to remain not-ready, got %q", view)
	}
}
