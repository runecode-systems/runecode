package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type routeActivationCase struct {
	name     string
	routeID  routeID
	newModel func(client localBrokerClient) routeModel
	expected []string
}

var routeActivationCaseData = []routeActivationCase{
	{
		name:    "dashboard uses typed status/run/approval/audit contracts",
		routeID: routeDashboard,
		newModel: func(client localBrokerClient) routeModel {
			return newDashboardRouteModel(routeDefinition{ID: routeDashboard, Label: "Dashboard"}, client)
		},
		expected: []string{"ReadinessGet", "VersionInfoGet", "RunList", "ApprovalList", "AuditVerificationGet"},
	},
	{
		name:    "chat uses typed session contracts",
		routeID: routeChat,
		newModel: func(client localBrokerClient) routeModel {
			return newChatRouteModel(routeDefinition{ID: routeChat, Label: "Chat"}, client)
		},
		expected: []string{"SessionList", "SessionGet"},
	},
	{
		name:    "runs uses typed run contracts",
		routeID: routeRuns,
		newModel: func(client localBrokerClient) routeModel {
			return newRunsRouteModel(routeDefinition{ID: routeRuns, Label: "Runs"}, client)
		},
		expected: []string{"RunList", "RunGet"},
	},
	{
		name:    "approvals uses typed approval contracts",
		routeID: routeApprovals,
		newModel: func(client localBrokerClient) routeModel {
			return newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, client)
		},
		expected: []string{"ApprovalList", "ApprovalGet"},
	},
	{
		name:    "action center uses typed approval/run/audit contracts",
		routeID: routeAction,
		newModel: func(client localBrokerClient) routeModel {
			return newActionCenterRouteModel(routeDefinition{ID: routeAction, Label: "Action Center"}, client)
		},
		expected: []string{"ApprovalList", "RunList", "AuditVerificationGet"},
	},
	{
		name:    "artifacts uses typed artifact contracts",
		routeID: routeArtifacts,
		newModel: func(client localBrokerClient) routeModel {
			return newArtifactsRouteModel(routeDefinition{ID: routeArtifacts, Label: "Artifacts"}, client)
		},
		expected: []string{"ArtifactList", "ArtifactHead", "ArtifactRead"},
	},
	{
		name:    "audit uses typed timeline verification contracts",
		routeID: routeAudit,
		newModel: func(client localBrokerClient) routeModel {
			return newAuditRouteModel(routeDefinition{ID: routeAudit, Label: "Audit"}, client)
		},
		expected: []string{"AuditTimeline", "AuditVerificationGet"},
	},
	{
		name:    "status uses typed readiness/version contracts",
		routeID: routeStatus,
		newModel: func(client localBrokerClient) routeModel {
			return newStatusRouteModel(routeDefinition{ID: routeStatus, Label: "Status"}, client)
		},
		expected: []string{"ReadinessGet", "VersionInfoGet", "BackendPostureGet"},
	},
}

func routeActivationCases() []routeActivationCase {
	out := make([]routeActivationCase, len(routeActivationCaseData))
	copy(out, routeActivationCaseData)
	return out
}

func TestRouteActivationUsesTypedBrokerContractsOnly(t *testing.T) {
	for _, tc := range routeActivationCases() {
		t.Run(tc.name, func(t *testing.T) {
			recording := newRecordingBrokerClient(&fakeBrokerClient{})
			model := tc.newModel(recording)
			updated, cmd := model.Update(routeActivatedMsg{RouteID: tc.routeID})
			if cmd == nil {
				t.Fatal("expected activation load command")
			}
			loaded := cmd()
			_, _ = updated.Update(loaded)
			assertStringSliceEqual(t, recording.Calls(), tc.expected)
		})
	}
}

func TestApprovalsResolveUsesTypedBrokerContract(t *testing.T) {
	recording := newRecordingBrokerClient(&fakeBrokerClient{})
	model := newApprovalsRouteModel(routeDefinition{ID: routeApprovals, Label: "Approvals"}, recording)

	updated, cmd := model.Update(routeActivatedMsg{RouteID: routeApprovals})
	if cmd == nil {
		t.Fatal("expected activation load command")
	}
	updated, _ = updated.Update(cmd())

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatal("expected promotion approval resolve to stay fail-closed")
	}

	view := updated.View(120, 40, focusContent)
	if want := "promotion approvals must be resolved via promote-excerpt to preserve exact promotion binding"; !containsSubstring(view, want) {
		t.Fatalf("expected fail-closed status %q in view, got %q", want, view)
	}

	assertStringSliceEqual(t, recording.Calls(), []string{"ApprovalList", "ApprovalGet"})
}

func containsSubstring(text, want string) bool {
	return strings.Contains(text, want)
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d calls, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d mismatch: expected %q, got %q (all calls: %v)", i, want[i], got[i], got)
		}
	}
}
