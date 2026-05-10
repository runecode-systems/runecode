//go:build runecode_tui_snapshot

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type snapshotScenarioState struct {
	Name     string
	RouteID  routeID
	Surface  routeModel
	Sessions []brokerapi.SessionSummary
	Watch    shellWatchTransportLoadedMsg
	Theme    themePreset
	Focus    focusArea
}

func resolveSnapshotScenarios(name string) ([]snapshotScenarioState, error) {
	scenarios := []snapshotScenarioState{
		buildHealthyEmptySnapshot(),
		buildApprovalWaitingSnapshot(),
		buildBlockedSnapshot(),
		buildDegradedSnapshot(),
		buildChatSnapshot(),
		buildRunsSnapshot(),
		buildApprovalsSnapshot(),
		buildActionCenterSnapshot(),
		buildArtifactsSnapshot(),
		buildAuditSnapshot(),
		buildStatusSnapshot(),
		buildProviderSetupSnapshot(),
		buildGitSetupSnapshot(),
		buildGitRemoteSnapshot(),
	}
	switch strings.TrimSpace(name) {
	case "", "all":
		return scenarios, nil
	case "dashboard":
		return scenarios[:4], nil
	case "action-center":
		return []snapshotScenarioState{buildActionCenterSnapshot()}, nil
	default:
		for _, scenario := range scenarios {
			if scenario.Name == name {
				return []snapshotScenarioState{scenario}, nil
			}
		}
		return nil, fmt.Errorf("unknown snapshot scenario %q", name)
	}
}

func resolveSnapshotBundleScenarios(name string) ([]snapshotScenarioState, error) {
	bundle, err := resolveSnapshotBundle(name)
	if err != nil {
		return nil, err
	}
	out := make([]snapshotScenarioState, 0, len(bundle.Scenarios))
	for _, name := range bundle.Scenarios {
		scenarios, err := resolveSnapshotScenarios(name)
		if err != nil {
			return nil, err
		}
		out = append(out, scenarios...)
	}
	return out, nil
}

func renderSnapshotScenario(state snapshotScenarioState, cfg tuiSnapshotConfig) (string, routeID, error) {
	snapshotRenderMu.Lock()
	defer snapshotRenderMu.Unlock()

	return withSnapshotColorProfile(func() (string, routeID, error) {
		forceMemoryWorkbenchState = true
		defer func() { forceMemoryWorkbenchState = false }()
		theme := cfg.theme
		if theme == "" {
			theme = state.Theme
		}
		appTheme = newTheme(theme)
		m := newShellModel()
		m.width = cfg.width
		m.height = cfg.height
		m.themePreset = theme
		m.preferredMode = presentationRendered
		m.location.Primary = shellObjectLocation{RouteID: state.RouteID, Object: workbenchObjectRef{Kind: "route", ID: string(state.RouteID)}}
		m.nav.SelectByRouteID(state.RouteID)
		m.routeModels[state.RouteID] = state.Surface
		m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: append([]brokerapi.SessionSummary(nil), state.Sessions...)})
		m.applyWatchTransport(state.Watch)
		m.publishWatchStateToRoutes()
		m.setFocus(state.Focus)
		m.syncSidebarCursorToLocation()
		return m.View(), state.RouteID, nil
	})
}

func defaultSnapshotSessions() []brokerapi.SessionSummary {
	return []brokerapi.SessionSummary{
		{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", LastActivityPreview: "Draft change proposal", LinkedRunCount: 1, LinkedApprovalCount: 1, TurnCount: 2},
		{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}, Status: "waiting", LastActivityPreview: "Review blocked work", LinkedRunCount: 1, LinkedApprovalCount: 0, TurnCount: 3},
	}
}

func buildHealthyEmptySnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: true, RecoveryComplete: true},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project: brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{
				SchemaID:               "runecode.protocol.v0.ProjectSubstratePostureSummary",
				ValidationState:        "valid",
				CompatibilityPosture:   "supported",
				NormalOperationAllowed: true,
			}},
			live:  watchProjection(watch).Live,
			audit: brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "ok", CurrentlyDegraded: false}},
		},
	}
	return snapshotScenarioState{Name: "dashboard-healthy-empty", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildApprovalWaitingSnapshot() snapshotScenarioState {
	watch := snapshotWatchApprovalWaiting()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: true, RecoveryComplete: true},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported_with_upgrade_available", NormalOperationAllowed: true}},
			runs:      []brokerapi.RunSummary{{RunID: "run-1", WorkspaceID: "ws-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", PendingApprovalCount: 1, ProvisioningPosture: "attested", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok"}},
			approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-1", StageID: "stage-2", ActionKind: "promotion"}}},
			live:      watchProjection(watch).Live,
			audit:     brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "ok", CurrentlyDegraded: false}},
		},
	}
	return snapshotScenarioState{Name: "dashboard-approval-waiting", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildBlockedSnapshot() snapshotScenarioState {
	watch := snapshotWatchBlocked()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: true, RecoveryComplete: true},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported", NormalOperationAllowed: true}},
			runs:      []brokerapi.RunSummary{{RunID: "run-blocked", WorkspaceID: "ws-1", LifecycleState: "blocked", BlockingReasonCode: "approval_wait", PendingApprovalCount: 1, BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", ProvisioningPosture: "attested", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok"}},
			approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-blocked", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-blocked", StageID: "stage-1", ActionKind: "promotion"}}},
			live:      watchProjection(watch).Live,
			audit:     brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "ok", CurrentlyDegraded: false}},
		},
	}
	return snapshotScenarioState{Name: "dashboard-blocked", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildDegradedSnapshot() snapshotScenarioState {
	watch := snapshotWatchDegraded()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: false, RecoveryComplete: false},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported_with_upgrade_available", NormalOperationAllowed: true}},
			runs:      []brokerapi.RunSummary{{RunID: "run-degraded", WorkspaceID: "ws-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "unavailable", RuntimePostureDegraded: true, ProvisioningPosture: "tofu", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "degraded", AuditCurrentlyDegraded: true}},
			live:      watchProjection(watch).Live,
			audit:     brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "degraded", CurrentlyDegraded: true, FindingCount: 2}},
		},
	}
	return snapshotScenarioState{Name: "dashboard-degraded", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildActionCenterSnapshot() snapshotScenarioState {
	watch := snapshotWatchActionCenter()
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	audit := brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "failed", AnchoringStatus: "degraded", CurrentlyDegraded: true, FindingCount: 3, HardFailures: []string{"anchor_receipt_invalid"}}}
	action := actionCenterRouteModel{
		def:         routeDefinition{ID: routeAction, Label: "Action Center"},
		now:         func() time.Time { return now },
		inspectorOn: true,
		family:      actionCenterFamilyApprovals,
		selected: map[actionCenterFamily]int{
			actionCenterFamilyApprovals: 0,
			actionCenterFamilyOps:       0,
			actionCenterFamilyBlocked:   0,
		},
		watch:       watchProjection(watch).Live,
		watchHealth: watchProjection(watch).Health,
		runs: []brokerapi.RunSummary{
			{RunID: "run-approval", WorkspaceID: "ws-1", LifecycleState: "waiting", PendingApprovalCount: 1, BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed"},
			{RunID: "run-degraded", WorkspaceID: "ws-1", LifecycleState: "active", RuntimePostureDegraded: true, BackendKind: "workspace", IsolationAssuranceLevel: "reduced", AuditAnchoringStatus: "degraded", AuditCurrentlyDegraded: true},
			{RunID: "run-blocked", WorkspaceID: "ws-1", LifecycleState: "blocked", BlockingReasonCode: "project_blocked", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed"},
		},
		approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-urgent", Status: "pending", ApprovalTriggerCode: "policy_gate", ExpiresAt: now.Add(45 * time.Minute).Format(time.RFC3339), BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-approval", StageID: "stage-2", ActionKind: "promotion"}}},
		project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "incompatible", CompatibilityPosture: "unsupported", NormalOperationAllowed: false}, BlockedExplanation: "project setup blocks normal operation", RemediationGuidance: []string{"review project setup", "apply supported substrate"}},
		audit:     &audit,
	}
	return snapshotScenarioState{Name: "action-center-triage", RouteID: routeAction, Surface: action, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildChatSnapshot() snapshotScenarioState {
	watch := snapshotWatchApprovalWaiting()
	active := &brokerapi.SessionDetail{
		Summary:              brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", LastActivityPreview: "Draft change proposal", LinkedRunCount: 1, LinkedArtifactCount: 1, LinkedApprovalCount: 1},
		CurrentTurnExecution: &brokerapi.SessionTurnExecution{TurnID: "turn-2", SessionID: "session-1", TriggerSource: "operator", RequestedOperation: "change_draft", ExecutionState: "waiting", PrimaryRunID: "run-1", PendingApprovalID: "ap-1"},
		LinkedRunIDs:         []string{"run-1"},
		LinkedApprovalIDs:    []string{"ap-1"},
	}
	run := &brokerapi.RunDetail{Summary: brokerapi.RunSummary{RunID: "run-1", WorkspaceID: "ws-1", WorkflowKind: "change_draft", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed"}, Coordination: brokerapi.RunCoordinationSummary{Blocked: true, WaitReasonCode: "approval_wait"}, PendingApprovalIDs: []string{"ap-1"}, ArtifactCountsByClass: map[string]int{"diffs": 1}}
	chat := chatRouteModel{def: routeDefinition{ID: routeChat, Label: "Chat"}, sessions: defaultSnapshotSessions(), selected: 0, active: active, activeID: "session-1", inspectorOn: true, presentation: presentationRendered, posture: &brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported", NormalOperationAllowed: true}}, runDetail: run, detailDoc: newLongFormDocumentState(), detailDocSession: active}
	return snapshotScenarioState{Name: "chat-active-session", RouteID: routeChat, Surface: chat, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildRunsSnapshot() snapshotScenarioState {
	watch := snapshotWatchApprovalWaiting()
	active := &brokerapi.RunDetail{Summary: brokerapi.RunSummary{RunID: "run-1", WorkspaceID: "ws-1", WorkflowKind: "change_draft", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", ProvisioningPosture: "attested", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok"}, Coordination: brokerapi.RunCoordinationSummary{Blocked: true, WaitReasonCode: "approval_wait", CoordinationMode: "stage_gate"}, StageSummaries: []brokerapi.RunStageSummary{{StageID: "stage-1", PendingApprovalCount: 1, ArtifactCount: 2}}, PendingApprovalIDs: []string{"ap-1"}, ArtifactCountsByClass: map[string]int{"diffs": 2}, AuthoritativeState: map[string]any{"workflow_projection_reason": "plan_authoritative"}}
	runs := runsRouteModel{def: routeDefinition{ID: routeRuns, Label: "Runs"}, runs: []brokerapi.RunSummary{active.Summary}, selected: 0, active: active, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
	return snapshotScenarioState{Name: "runs-active-detail", RouteID: routeRuns, Surface: runs, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildApprovalsSnapshot() snapshotScenarioState {
	watch := snapshotWatchApprovalWaiting()
	active := &brokerapi.ApprovalGetResponse{Approval: brokerapi.ApprovalSummary{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{WorkspaceID: "ws-1", RunID: "run-1", StageID: "stage-1", ActionKind: "promotion"}}, ApprovalDetail: brokerapi.ApprovalDetail{BindingKind: "exact_action", PolicyReasonCode: "requires_human_review", LifecycleDetail: brokerapi.ApprovalLifecycleDetail{LifecycleState: "pending", LifecycleReasonCode: "awaiting_decision"}, WhatChangesIfApproved: brokerapi.ApprovalWhatChangesIfApproved{Summary: "Promotion continues", EffectKind: "unblock_next_stage"}, BlockedWorkScope: brokerapi.ApprovalBlockedWorkScope{ScopeKind: "stage", RunID: "run-1", StageID: "stage-1", ActionKind: "promotion"}, BoundIdentity: brokerapi.ApprovalBoundIdentity{ApprovalRequestDigest: "sha256:req", ManifestHash: "sha256:manifest", PolicyDecisionHash: "sha256:policy"}}}
	approvals := approvalsRouteModel{def: routeDefinition{ID: routeApprovals, Label: "Approvals"}, items: []brokerapi.ApprovalSummary{active.Approval}, selected: 0, active: active, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
	return snapshotScenarioState{Name: "approvals-pending-detail", RouteID: routeApprovals, Surface: approvals, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildArtifactsSnapshot() snapshotScenarioState {
	watch := snapshotWatchApprovalWaiting()
	selected := brokerapi.ArtifactSummary{RunID: "run-1"}
	selected.Reference.Digest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	selected.Reference.ContentType = "text/plain"
	selected.Reference.DataClass = "diffs"
	selected.Reference.SizeBytes = 128
	selected.Reference.ProvenanceReceiptHash = "sha256:receipt"
	other := brokerapi.ArtifactSummary{RunID: "run-1"}
	other.Reference.Digest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	other.Reference.ContentType = "text/plain"
	other.Reference.DataClass = "build_logs"
	other.Reference.SizeBytes = 256
	other.Reference.ProvenanceReceiptHash = "sha256:receipt-2"
	artifacts := artifactsRouteModel{
		def:          routeDefinition{ID: routeArtifacts, Label: "Artifacts"},
		items:        []brokerapi.ArtifactSummary{selected, other},
		selected:     0,
		active:       &brokerapi.LocalArtifactHeadResponse{Artifact: selected},
		content:      "diff --git a/docs/plan.md b/docs/plan.md\n+Evidence trail copy now points reviewers to Audit\n+Anchoring receipt attached: sha256:receipt\n-result: pending approval\n",
		mode:         artifactModeDiff,
		presentation: presentationRendered,
		inspectorOn:  true,
		detailDoc:    newLongFormDocumentState(),
	}
	artifacts.syncDetailDocument()
	return snapshotScenarioState{Name: "artifacts-evidence-detail", RouteID: routeArtifacts, Surface: artifacts, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildAuditSnapshot() snapshotScenarioState {
	watch := snapshotWatchActionCenter()
	verify := &brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "failed", AnchoringStatus: "degraded", CurrentlyDegraded: true, HardFailures: []string{"anchor_receipt_invalid"}}}
	recordDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("c", 64)}
	record := &brokerapi.AuditRecordGetResponse{Record: brokerapi.AuditRecordDetail{RecordDigest: recordDigest, RecordFamily: "audit_event", EventType: "run_state", Summary: "Run state changed", OccurredAt: "2026-05-09T12:20:00Z", LinkedReferences: []brokerapi.AuditRecordLinkedReference{{ReferenceKind: "run", ReferenceID: "run-1"}}, VerificationPosture: &brokerapi.AuditRecordVerificationPosture{Status: "degraded", ReasonCodes: []string{"anchor_delayed"}}}}
	audit := auditRouteModel{def: routeDefinition{ID: routeAudit, Label: "Audit"}, verify: verify, active: record, timeline: []brokerapi.AuditTimelineViewEntry{{RecordDigest: recordDigest, EventType: "run_state", Summary: "Run state changed"}}, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
	return snapshotScenarioState{Name: "audit-degraded-detail", RouteID: routeAudit, Surface: audit, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildStatusSnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	status := statusRouteModel{def: routeDefinition{ID: routeStatus, Label: "Status"}, data: statusLoadedMsg{readiness: brokerapi.BrokerReadiness{Ready: true, LocalOnly: true, RecoveryComplete: true, AppendPositionStable: true, CurrentSegmentWritable: true, VerifierMaterialAvailable: true, DerivedIndexCaughtUp: true}, version: brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0", BuildRevision: "deadbeef", BuildTime: "2026-05-09T00:00:00Z", ProtocolBundleVersion: "v0", ProtocolBundleManifestHash: "sha256:bundle", APIFamily: "local", APIVersion: "0.1.0"}, lifecycle: brokerapi.BrokerProductLifecyclePosture{SchemaID: "runecode.protocol.v0.BrokerProductLifecyclePosture", LifecyclePosture: "normal", AttachMode: "managed", Attachable: true, NormalOperationAllowed: true}, project: brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported", NormalOperationAllowed: true}, RemediationGuidance: []string{"No remediation required"}}, posture: brokerapi.BackendPostureState{InstanceID: "instance-1", BackendKind: "workspace"}}}
	return snapshotScenarioState{Name: "status-ready-overview", RouteID: routeStatus, Surface: status, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildProviderSetupSnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	profile := brokerapi.ProviderProfile{ProviderProfileID: "provider-profile-test", DisplayLabel: "OpenAI Compatible", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", CurrentAuthMode: "direct_credential", SupportedAuthModes: []string{"direct_credential"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{CredentialState: "missing", EffectiveReadiness: "not_ready"}}
	provider := providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, selected: providerSetupDefaultsFor("openai_compatible"), status: "Press s to start direct-credential setup. Press f to switch provider family.", profiles: []brokerapi.ProviderProfile{profile}}
	return snapshotScenarioState{Name: "model-providers-credential-needed", RouteID: routeProviders, Surface: provider, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildGitSetupSnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	git := gitSetupRouteModel{def: routeDefinition{ID: routeGitSetup, Label: "Git Setup"}, provider: "github", data: brokerapi.GitSetupGetResponse{ProviderAccount: brokerapi.GitProviderAccountState{Provider: "github", Linked: true, AccountUsername: "runecode-dev"}, IdentityProfiles: []brokerapi.GitCommitIdentityProfile{{ProfileID: "default", DisplayName: "RuneCode Dev", AuthorName: "RuneCode Dev", AuthorEmail: "dev@example.com", CommitterName: "RuneCode Dev", CommitterEmail: "dev@example.com", DefaultProfile: true}}, AuthPosture: brokerapi.GitAuthPostureState{AuthStatus: "linked", BootstrapMode: "browser", HeadlessBootstrapSupported: true}, ControlPlaneState: brokerapi.GitControlPlaneState{DefaultIdentityProfileID: "default", LastSetupView: "overview"}, PolicySurface: brokerapi.GitPolicySurfaceState{ArtifactManagedOnly: true, DirectMutationSupport: false}}}
	return snapshotScenarioState{Name: "git-setup-identity-needed", RouteID: routeGitSetup, Surface: git, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildGitRemoteSnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	requestHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}
	actionHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("2", 64)}
	decisionHash := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("3", 64)}
	approvalRequest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("4", 64)}
	approvalDecision := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("5", 64)}
	patchDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("6", 64)}
	expectedTree := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("7", 64)}
	prepared := brokerapi.GitRemoteMutationPreparedState{PreparedMutationID: "sha256:" + strings.Repeat("9", 64), RunID: "run-1", Provider: "github", DestinationRef: "github.com/runecode-ai/runecode", RequestKind: "git_ref_update", TypedRequestHash: requestHash, ActionRequestHash: actionHash, PolicyDecisionHash: decisionHash, RequiredApprovalID: "sha256:" + strings.Repeat("a", 64), RequiredApprovalRequestHash: &approvalRequest, RequiredApprovalDecisionHash: &approvalDecision, LifecycleState: "prepared", ExecutionState: "not_started", DerivedSummary: brokerapi.GitRemoteMutationDerivedSummary{RepositoryIdentity: "github.com/runecode-ai/runecode", TargetRefs: []string{"refs/heads/main"}, ReferencedPatchArtifactHashes: []trustpolicy.Digest{patchDigest}, ExpectedResultTreeHash: expectedTree, CommitSubject: "Apply reviewed patch"}}
	git := gitRemoteMutationRouteModel{def: routeDefinition{ID: routeGitRemote, Label: "Git Remote"}, prepared: prepared}
	return snapshotScenarioState{Name: "git-remote-approval-ready", RouteID: routeGitRemote, Surface: git, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func watchProjection(msg shellWatchTransportLoadedMsg) shellWatchProjectionState {
	manager := newShellWatchManager()
	manager.applyTransport(msg)
	return manager.projection
}

func snapshotWatchHealthyEmpty() shellWatchTransportLoadedMsg {
	return shellWatchTransportLoadedMsg{ObservedAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)}
}

func snapshotWatchApprovalWaiting() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 5, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", PendingApprovalCount: 1}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-1", Status: "pending"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", LastActivityPreview: "Awaiting approval"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}}}}
}

func snapshotWatchBlocked() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 10, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-blocked", LifecycleState: "blocked", BlockingReasonCode: "approval_wait", PendingApprovalCount: 1}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-blocked", Status: "pending"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}, Status: "blocked", LastActivityPreview: "Blocked on approval"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}}}}
}

func snapshotWatchDegraded() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 15, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-degraded", LifecycleState: "active", RuntimePostureDegraded: true, AuditCurrentlyDegraded: true}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-quiet", Status: "consumed"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "degraded", LastActivityPreview: "Runtime posture degraded"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}, {EventType: "run_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}, {EventType: "session_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"}}}}
}

func snapshotWatchActionCenter() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 20, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-degraded", LifecycleState: "active", RuntimePostureDegraded: true}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-urgent", Status: "pending"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}, Status: "active", LastActivityPreview: "Triage action center"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}}}}
}
