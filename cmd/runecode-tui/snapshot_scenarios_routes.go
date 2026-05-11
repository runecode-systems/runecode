//go:build runecode_tui_snapshot

package main

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

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

func buildChatComposeOpenSnapshot() snapshotScenarioState {
	chatState := buildChatSnapshot()
	chat := chatState.Surface.(chatRouteModel)
	chat.composeOn = true
	chat.composer.SetValue("Summarize blocked work and next approval.")
	chat.composer.Focus()
	chatState.Name = "chat-compose-open"
	chatState.Surface = chat
	return chatState
}

func buildRunsSnapshot() snapshotScenarioState {
	watch := snapshotWatchApprovalWaiting()
	active := &brokerapi.RunDetail{Summary: brokerapi.RunSummary{RunID: "run-1", WorkspaceID: "ws-1", WorkflowKind: "change_draft", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", ProvisioningPosture: "attested", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok"}, Coordination: brokerapi.RunCoordinationSummary{Blocked: true, WaitReasonCode: "approval_wait", CoordinationMode: "stage_gate"}, StageSummaries: []brokerapi.RunStageSummary{{StageID: "stage-1", PendingApprovalCount: 1, ArtifactCount: 2}}, PendingApprovalIDs: []string{"ap-1"}, ArtifactCountsByClass: map[string]int{"diffs": 2}, AuthoritativeState: map[string]any{"workflow_projection_reason": "plan_authoritative"}}
	runs := runsRouteModel{def: routeDefinition{ID: routeRuns, Label: "Runs"}, runs: []brokerapi.RunSummary{active.Summary}, selected: 0, active: active, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
	return snapshotScenarioState{Name: "runs-active-detail", RouteID: routeRuns, Surface: runs, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildRunsFocusInspectorSnapshot() snapshotScenarioState {
	runsState := buildRunsSnapshot()
	runsState.Name = "runs-focus-inspector"
	runsState.Focus = focusInspector
	return runsState
}

func buildRunsRawSnapshot() snapshotScenarioState {
	runsState := buildRunsSnapshot()
	runs := runsState.Surface.(runsRouteModel)
	runs.presentation = presentationRaw
	runs.syncDetailDocument()
	runsState.Name = "runs-raw-detail"
	runsState.Surface = runs
	return runsState
}

func buildApprovalsSnapshot() snapshotScenarioState {
	watch := snapshotWatchApprovalWaiting()
	active := &brokerapi.ApprovalGetResponse{Approval: brokerapi.ApprovalSummary{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{WorkspaceID: "ws-1", RunID: "run-1", StageID: "stage-1", ActionKind: "promotion"}}, ApprovalDetail: brokerapi.ApprovalDetail{BindingKind: "exact_action", PolicyReasonCode: "requires_human_review", LifecycleDetail: brokerapi.ApprovalLifecycleDetail{LifecycleState: "pending", LifecycleReasonCode: "awaiting_decision"}, WhatChangesIfApproved: brokerapi.ApprovalWhatChangesIfApproved{Summary: "Promotion continues", EffectKind: "unblock_next_stage"}, BlockedWorkScope: brokerapi.ApprovalBlockedWorkScope{ScopeKind: "stage", RunID: "run-1", StageID: "stage-1", ActionKind: "promotion"}, BoundIdentity: brokerapi.ApprovalBoundIdentity{ApprovalRequestDigest: "sha256:req", ManifestHash: "sha256:manifest", PolicyDecisionHash: "sha256:policy"}}}
	approvals := approvalsRouteModel{def: routeDefinition{ID: routeApprovals, Label: "Approvals"}, items: []brokerapi.ApprovalSummary{active.Approval}, selected: 0, active: active, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
	return snapshotScenarioState{Name: "approvals-pending-detail", RouteID: routeApprovals, Surface: approvals, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildApprovalsStructuredSnapshot() snapshotScenarioState {
	approvalState := buildApprovalsSnapshot()
	approvals := approvalState.Surface.(approvalsRouteModel)
	approvals.presentation = presentationStructured
	approvals.syncDetailDocument()
	approvalState.Name = "approvals-structured-detail"
	approvalState.Surface = approvals
	return approvalState
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

func buildArtifactsStructuredSnapshot() snapshotScenarioState {
	artifactState := buildArtifactsSnapshot()
	artifacts := artifactState.Surface.(artifactsRouteModel)
	artifacts.presentation = presentationStructured
	artifacts.syncDetailDocument()
	artifactState.Name = "artifacts-structured-detail"
	artifactState.Surface = artifacts
	return artifactState
}

func buildAuditSnapshot() snapshotScenarioState {
	watch := snapshotWatchActionCenter()
	verify := &brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "failed", AnchoringStatus: "degraded", CurrentlyDegraded: true, HardFailures: []string{"anchor_receipt_invalid"}}}
	recordDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("c", 64)}
	record := &brokerapi.AuditRecordGetResponse{Record: brokerapi.AuditRecordDetail{RecordDigest: recordDigest, RecordFamily: "audit_event", EventType: "run_state", Summary: "Run state changed", OccurredAt: "2026-05-09T12:20:00Z", LinkedReferences: []brokerapi.AuditRecordLinkedReference{{ReferenceKind: "run", ReferenceID: "run-1"}}, VerificationPosture: &brokerapi.AuditRecordVerificationPosture{Status: "degraded", ReasonCodes: []string{"anchor_delayed"}}}}
	audit := auditRouteModel{def: routeDefinition{ID: routeAudit, Label: "Audit"}, verify: verify, active: record, timeline: []brokerapi.AuditTimelineViewEntry{{RecordDigest: recordDigest, EventType: "run_state", Summary: "Run state changed"}}, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
	return snapshotScenarioState{Name: "audit-degraded-detail", RouteID: routeAudit, Surface: audit, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildAuditRawSnapshot() snapshotScenarioState {
	auditState := buildAuditSnapshot()
	audit := auditState.Surface.(auditRouteModel)
	audit.presentation = presentationRaw
	audit.syncDetailDocument()
	auditState.Name = "audit-raw-detail"
	auditState.Surface = audit
	return auditState
}

func buildStatusSnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	status := statusRouteModel{def: routeDefinition{ID: routeStatus, Label: "Status"}, data: statusLoadedMsg{readiness: brokerapi.BrokerReadiness{Ready: true, LocalOnly: true, RecoveryComplete: true, AppendPositionStable: true, CurrentSegmentWritable: true, VerifierMaterialAvailable: true, DerivedIndexCaughtUp: true}, version: brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0", BuildRevision: "deadbeef", BuildTime: "2026-05-09T00:00:00Z", ProtocolBundleVersion: "v0", ProtocolBundleManifestHash: "sha256:bundle", APIFamily: "local", APIVersion: "0.1.0"}, lifecycle: brokerapi.BrokerProductLifecyclePosture{SchemaID: "runecode.protocol.v0.BrokerProductLifecyclePosture", LifecyclePosture: "normal", AttachMode: "managed", Attachable: true, NormalOperationAllowed: true}, project: brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported", NormalOperationAllowed: true}, RemediationGuidance: []string{"No remediation required"}}, posture: brokerapi.BackendPostureState{InstanceID: "instance-1", BackendKind: "workspace"}}}
	return snapshotScenarioState{Name: "status-ready-overview", RouteID: routeStatus, Surface: status, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildStatusDegradedSnapshot() snapshotScenarioState {
	watch := snapshotWatchDegraded()
	status := statusRouteModel{def: routeDefinition{ID: routeStatus, Label: "Status"}, status: "Project setup validation could not be refreshed. Review the broker guidance below.", data: statusLoadedMsg{readiness: brokerapi.BrokerReadiness{Ready: false, LocalOnly: true, RecoveryComplete: false, AppendPositionStable: true, CurrentSegmentWritable: false, VerifierMaterialAvailable: true, DerivedIndexCaughtUp: false}, version: brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0", BuildRevision: "deadbeef", BuildTime: "2026-05-09T00:00:00Z", ProtocolBundleVersion: "v0", ProtocolBundleManifestHash: "sha256:bundle", APIFamily: "local", APIVersion: "0.1.0"}, lifecycle: brokerapi.BrokerProductLifecyclePosture{SchemaID: "runecode.protocol.v0.BrokerProductLifecyclePosture", SchemaVersion: "0.1.0", ProductInstanceID: "repo-test", LifecycleGeneration: "gen-degraded", LifecyclePosture: "degraded", AttachMode: "managed", Attachable: true, NormalOperationAllowed: false, DegradedReasonCodes: []string{"verifier_lagging"}}, project: brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "degraded", CompatibilityPosture: "supported_with_upgrade_available", NormalOperationAllowed: true}, RemediationGuidance: []string{"Refresh managed setup validation"}}, posture: brokerapi.BackendPostureState{InstanceID: "instance-1", BackendKind: "workspace"}}}
	return snapshotScenarioState{Name: "status-degraded-overview", RouteID: routeStatus, Surface: status, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildProviderSetupSnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	profile := brokerapi.ProviderProfile{ProviderProfileID: "provider-profile-test", DisplayLabel: "OpenAI Compatible", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", CurrentAuthMode: "direct_credential", SupportedAuthModes: []string{"direct_credential"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{CredentialState: "missing", EffectiveReadiness: "not_ready"}}
	provider := providerSetupRouteModel{def: routeDefinition{ID: routeProviders, Label: "Model Providers"}, selected: providerSetupDefaultsFor("openai_compatible"), status: "Press s to start direct-credential setup. Press f to switch provider family.", profiles: []brokerapi.ProviderProfile{profile}}
	return snapshotScenarioState{Name: "model-providers-credential-needed", RouteID: routeProviders, Surface: provider, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildProviderSecretEntrySnapshot() snapshotScenarioState {
	providerState := buildProviderSetupSnapshot()
	provider := providerState.Surface.(providerSetupRouteModel)
	provider.entryActive = true
	provider.secretRunes = []rune("sk-live-123456")
	provider.status = "Credential entry is active. Press Enter to submit or Esc to cancel."
	providerState.Name = "model-providers-secret-entry"
	providerState.Surface = provider
	return providerState
}

func buildGitSetupSnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	git := gitSetupRouteModel{def: routeDefinition{ID: routeGitSetup, Label: "Git Setup"}, provider: "github", data: brokerapi.GitSetupGetResponse{ProviderAccount: brokerapi.GitProviderAccountState{Provider: "github", Linked: true, AccountUsername: "runecode-dev"}, IdentityProfiles: []brokerapi.GitCommitIdentityProfile{{ProfileID: "default", DisplayName: "RuneCode Dev", AuthorName: "RuneCode Dev", AuthorEmail: "dev@example.com", CommitterName: "RuneCode Dev", CommitterEmail: "dev@example.com", DefaultProfile: true}}, AuthPosture: brokerapi.GitAuthPostureState{AuthStatus: "linked", BootstrapMode: "browser", HeadlessBootstrapSupported: true}, ControlPlaneState: brokerapi.GitControlPlaneState{DefaultIdentityProfileID: "default", LastSetupView: "overview"}, PolicySurface: brokerapi.GitPolicySurfaceState{ArtifactManagedOnly: true, DirectMutationSupport: false}}}
	return snapshotScenarioState{Name: "git-setup-identity-needed", RouteID: routeGitSetup, Surface: git, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildGitSetupProviderLinkedSnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	git := gitSetupRouteModel{def: routeDefinition{ID: routeGitSetup, Label: "Git Setup"}, provider: "github", status: "The provider account is linked, but commit identity still needs setup.", data: brokerapi.GitSetupGetResponse{ProviderAccount: brokerapi.GitProviderAccountState{Provider: "github", Linked: true, AccountUsername: "runecode-dev"}, AuthPosture: brokerapi.GitAuthPostureState{AuthStatus: "linked", BootstrapMode: "browser", HeadlessBootstrapSupported: true}, ControlPlaneState: brokerapi.GitControlPlaneState{DefaultIdentityProfileID: "", LastSetupView: "overview"}, PolicySurface: brokerapi.GitPolicySurfaceState{ArtifactManagedOnly: true, DirectMutationSupport: false}}}
	return snapshotScenarioState{Name: "git-setup-provider-linked", RouteID: routeGitSetup, Surface: git, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
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

func buildGitRemoteExecutedSnapshot() snapshotScenarioState {
	gitState := buildGitRemoteSnapshot()
	git := gitState.Surface.(gitRemoteMutationRouteModel)
	git.prepared.ExecutionState = "completed"
	git.status = "Remote execution completed. Review the refreshed state to confirm the remote change landed as expected."
	gitState.Name = "git-remote-executed"
	gitState.Surface = git
	return gitState
}

func buildGitRemoteFailClosedSnapshot() snapshotScenarioState {
	gitState := buildGitRemoteSnapshot()
	git := gitState.Surface.(gitRemoteMutationRouteModel)
	git.prepared.RequiredApprovalID = ""
	git.prepared.RequiredApprovalRequestHash = nil
	git.prepared.RequiredApprovalDecisionHash = nil
	git.errText = "required approval binding is incomplete in prepared state; execute remains fail-closed"
	git.status = ""
	gitState.Name = "git-remote-fail-closed"
	gitState.Surface = git
	return gitState
}
