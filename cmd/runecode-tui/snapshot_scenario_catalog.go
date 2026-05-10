//go:build runecode_tui_snapshot

package main

func allSnapshotScenarios() []snapshotScenarioState {
	return []snapshotScenarioState{
		buildHealthyEmptySnapshot(),
		buildApprovalWaitingSnapshot(),
		buildBlockedSnapshot(),
		buildDegradedSnapshot(),
		buildToastSnapshot(),
		buildCommandModeDraftSnapshot(),
		buildCommandModeErrorSnapshot(),
		buildCommandPaletteSnapshot(),
		buildSessionSwitcherSnapshot(),
		buildQuitConfirmChatComposeSnapshot(),
		buildQuitConfirmCommandEntrySnapshot(),
		buildQuitConfirmProviderSecretSnapshot(),
		buildNarrowSidebarOverlaySnapshot(),
		buildNarrowInspectorOverlaySnapshot(),
		buildNarrowSidebarOverlayCompactSnapshot(),
		buildNarrowInspectorOverlayCompactSnapshot(),
		buildDashboardFocusNavSnapshot(),
		buildLeaderHelpSnapshot(),
		buildChatSnapshot(),
		buildChatComposeOpenSnapshot(),
		buildRunsSnapshot(),
		buildRunsFocusInspectorSnapshot(),
		buildRunsRawSnapshot(),
		buildApprovalsSnapshot(),
		buildApprovalsStructuredSnapshot(),
		buildActionCenterSnapshot(),
		buildArtifactsSnapshot(),
		buildArtifactsStructuredSnapshot(),
		buildAuditSnapshot(),
		buildAuditRawSnapshot(),
		buildStatusSnapshot(),
		buildStatusDegradedSnapshot(),
		buildProviderSetupSnapshot(),
		buildProviderSecretEntrySnapshot(),
		buildGitSetupSnapshot(),
		buildGitSetupProviderLinkedSnapshot(),
		buildGitRemoteSnapshot(),
		buildGitRemoteExecutedSnapshot(),
		buildGitRemoteFailClosedSnapshot(),
	}
}

func dashboardSnapshotScenarios() []snapshotScenarioState {
	return []snapshotScenarioState{
		buildHealthyEmptySnapshot(),
		buildApprovalWaitingSnapshot(),
		buildBlockedSnapshot(),
		buildDegradedSnapshot(),
		buildToastSnapshot(),
		buildDashboardFocusNavSnapshot(),
		buildLeaderHelpSnapshot(),
	}
}
