package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestCLIAdoptionRoutesRunApprovalVersionAndLogThroughLocalRPC(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	requestedOps := make([]string, 0, 8)

	installRunApprovalVersionLogDispatchStub(t, &requestedOps)

	llmRequestPath := writeLLMRequestFile(t)
	prepareReqPath, getReqPath, leaseReqPath, executeReqPath := writeGitRemoteMutationRequestFiles(t)
	commands := [][]string{{"run-list"}, {"run-get", "--run-id", "run-1"}, {"run-watch"}, {"backend-posture-get"}, {"backend-posture-change", "--target-backend-kind", "container", "--reduced-assurance-acknowledged"}, {"session-list"}, {"session-get", "--session-id", "sess-1"}, {"session-send-message", "--session-id", "sess-1", "--content", "hello"}, {"session-watch"}, {"approval-list"}, {"approval-get", "--approval-id", testDigest("a")}, {"approval-watch"}, {"git-setup-get", "--provider", "github"}, {"git-setup-auth-bootstrap", "--provider", "github", "--mode", "browser"}, {"git-setup-identity-upsert", "--provider", "github", "--profile-id", "default", "--display-name", "Default identity", "--author-name", "RuneCode Operator", "--author-email", "operator@example.invalid", "--committer-name", "RuneCode Operator", "--committer-email", "operator@example.invalid", "--signoff-name", "RuneCode Operator", "--signoff-email", "operator@example.invalid", "--default-profile"}, {"provider-profile-list"}, {"provider-profile-get", "--provider-profile-id", "provider-profile-test"}, {"provider-credential-lease-issue", "--provider-profile-id", "provider-profile-test", "--run-id", "run-1", "--ttl-seconds", "900"}, {"project-substrate-get"}, {"project-substrate-posture-get"}, {"project-substrate-adopt"}, {"project-substrate-init-preview"}, {"project-substrate-init-apply"}, {"project-substrate-upgrade-preview"}, {"project-substrate-upgrade-apply", "--expected-preview-digest", testDigest("u")}, {"git-remote-mutation-prepare", "--request-file", prepareReqPath}, {"git-remote-mutation-get", "--request-file", getReqPath}, {"git-remote-mutation-issue-execute-lease", "--request-file", leaseReqPath}, {"git-remote-mutation-execute", "--request-file", executeReqPath}, {"version-info"}, {"stream-logs"}, {"stream-logs", "--stream-id", "custom-stream"}, {"llm-invoke", "--run-id", "run-1", "--request-file", llmRequestPath}, {"llm-stream", "--run-id", "run-1", "--request-file", llmRequestPath, "--stream-id", "llm-s-1"}}
	for _, args := range commands {
		stdout.Reset()
		if err := run(args, stdout, stderr); err != nil {
			t.Fatalf("run(%v) error: %v", args, err)
		}
	}

	want := []string{"run_list", "run_get", "run_watch", "backend_posture_get", "backend_posture_get", "backend_posture_change", "session_list", "session_get", "session_send_message", "session_watch", "approval_list", "approval_get", "approval_watch", "git_setup_get", "git_setup_auth_bootstrap", "git_setup_identity_upsert", "provider_profile_list", "provider_profile_get", "provider_credential_lease_issue", "project_substrate_get", "project_substrate_posture_get", "project_substrate_adopt", "project_substrate_init_preview", "project_substrate_init_apply", "project_substrate_upgrade_preview", "project_substrate_upgrade_apply", "git_remote_mutation_prepare", "git_remote_mutation_get", "git_remote_mutation_issue_execute_lease", "git_remote_mutation_execute", "version_info_get", "log_stream", "log_stream", "llm_invoke", "llm_stream"}
	assertRequestedOps(t, requestedOps, want)
}

func TestGitRemoteMutationPrepareRequiresRequestFile(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"git-remote-mutation-prepare"}, stdout, stderr)
	if err == nil {
		t.Fatal("git-remote-mutation-prepare expected usage error when request file is missing")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("git-remote-mutation-prepare error type = %T, want *usageError", err)
	}
}

func TestSessionSendMessageRejectsInvalidRole(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"session-send-message", "--session-id", "sess-1", "--content", "hello", "--role", "invalid"}, stdout, stderr)
	if err == nil {
		t.Fatal("session-send-message expected usage error for invalid role")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("session-send-message error type = %T, want *usageError", err)
	}
}

func installRunApprovalVersionLogDispatchStub(t *testing.T, requestedOps *[]string) {
	t.Helper()
	originalDispatch := localRPCDispatch
	localRPCDispatch = func(_ *brokerapi.Service, _ context.Context, wire localRPCRequest, _ brokerapi.RequestContext) localRPCResponse {
		*requestedOps = append(*requestedOps, wire.Operation)
		opCount := len(*requestedOps)
		if resp, ok := handleRunApprovalVersionLogStub(t, wire); ok {
			return resp
		}
		return handleSessionGitLLMStub(t, wire, opCount)
	}
	t.Cleanup(func() { localRPCDispatch = originalDispatch })
}

func handleRunApprovalVersionLogStub(t *testing.T, wire localRPCRequest) (localRPCResponse, bool) {
	t.Helper()
	switch wire.Operation {
	case "run_list":
		return mustOKLocalRPCResponse(t, brokerapi.RunListResponse{SchemaID: "runecode.protocol.v0.RunListResponse", SchemaVersion: "0.1.0", RequestID: "req-run-list"}), true
	case "run_get":
		return mustOKLocalRPCResponse(t, brokerapi.RunGetResponse{SchemaID: "runecode.protocol.v0.RunGetResponse", SchemaVersion: "0.1.0", RequestID: "req-run-get", Run: brokerapi.RunDetail{SchemaID: "runecode.protocol.v0.RunDetail", SchemaVersion: "0.2.0", Summary: brokerapi.RunSummary{SchemaID: "runecode.protocol.v0.RunSummary", SchemaVersion: "0.2.0", RunID: "run-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z", LifecycleState: "pending", PendingApprovalCount: 0, ApprovalProfile: "unknown", BackendKind: "unknown", IsolationAssuranceLevel: "unknown", ProvisioningPosture: "unknown", AssuranceLevel: "unknown", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok", AuditCurrentlyDegraded: false, RuntimePostureDegraded: false}}}), true
	case "run_watch":
		return mustOKLocalRPCResponse(t, []brokerapi.RunWatchEvent{{SchemaID: "runecode.protocol.v0.RunWatchEvent", SchemaVersion: "0.1.0", StreamID: "rw-1", RequestID: "req-run-watch", Seq: 1, EventType: "run_watch_snapshot", Run: &brokerapi.RunSummary{SchemaID: "runecode.protocol.v0.RunSummary", SchemaVersion: "0.2.0", RunID: "run-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z", LifecycleState: "pending", PendingApprovalCount: 0, ApprovalProfile: "unknown", BackendKind: "unknown", IsolationAssuranceLevel: "unknown", ProvisioningPosture: "unknown", AssuranceLevel: "unknown", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok", AuditCurrentlyDegraded: false, RuntimePostureDegraded: false}}, {SchemaID: "runecode.protocol.v0.RunWatchEvent", SchemaVersion: "0.1.0", StreamID: "rw-1", RequestID: "req-run-watch", Seq: 2, EventType: "run_watch_terminal", Terminal: true, TerminalStatus: "completed"}}), true
	case "backend_posture_get":
		return mustOKLocalRPCResponse(t, brokerapi.BackendPostureGetResponse{SchemaID: "runecode.protocol.v0.BackendPostureGetResponse", SchemaVersion: "0.1.0", RequestID: "req-backend-posture-get", Posture: brokerapi.BackendPostureState{SchemaID: "runecode.protocol.v0.BackendPostureState", SchemaVersion: "0.1.0", InstanceID: "launcher-instance-1", BackendKind: "microvm", PreferredBackendKind: "microvm", Availability: []brokerapi.BackendPostureAvailability{{SchemaID: "runecode.protocol.v0.BackendPostureAvailability", SchemaVersion: "0.1.0", BackendKind: "microvm", Available: true}, {SchemaID: "runecode.protocol.v0.BackendPostureAvailability", SchemaVersion: "0.1.0", BackendKind: "container", Available: true}}}}), true
	case "backend_posture_change":
		return mustOKLocalRPCResponse(t, brokerapi.BackendPostureChangeResponse{SchemaID: "runecode.protocol.v0.BackendPostureChangeResponse", SchemaVersion: "0.1.0", RequestID: "req-backend-posture-change", Outcome: brokerapi.BackendPostureChangeOutcome{SchemaID: "runecode.protocol.v0.BackendPostureChangeOutcome", SchemaVersion: "0.1.0", Outcome: "approval_required", OutcomeReasonCode: "approval_required", ApprovalID: testDigest("a")}, Posture: brokerapi.BackendPostureState{SchemaID: "runecode.protocol.v0.BackendPostureState", SchemaVersion: "0.1.0", InstanceID: "launcher-instance-1", BackendKind: "microvm", PendingApproval: true, PendingApprovalID: testDigest("a")}}), true
	default:
		return localRPCResponse{}, false
	}
}

func handleSessionGitLLMStub(t *testing.T, wire localRPCRequest, opCount int) localRPCResponse {
	t.Helper()
	if resp, ok := handleSessionRPCStub(t, wire); ok {
		return resp
	}
	if resp, ok := handleApprovalRPCStub(t, wire); ok {
		return resp
	}
	if resp, ok := handleProviderProfileRPCStub(t, wire); ok {
		return resp
	}
	if resp, ok := handleProjectSubstrateRPCStub(t, wire); ok {
		return resp
	}
	if resp, ok := handleGitRemoteAndLLMRPCStub(t, wire, opCount); ok {
		return resp
	}
	return localRPCResponse{OK: false}
}

func handleSessionRPCStub(t *testing.T, wire localRPCRequest) (localRPCResponse, bool) {
	t.Helper()
	switch wire.Operation {
	case "session_list":
		return mustOKLocalRPCResponse(t, brokerapi.SessionListResponse{SchemaID: "runecode.protocol.v0.SessionListResponse", SchemaVersion: "0.1.0", RequestID: "req-session-list", Order: "updated_at_desc", Sessions: []brokerapi.SessionSummary{{SchemaID: "runecode.protocol.v0.SessionSummary", SchemaVersion: "0.1.0", Identity: brokerapi.SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z"}, UpdatedAt: "2026-01-01T00:00:00Z", Status: "active", LastActivityKind: "run_progress", TurnCount: 0, LinkedRunCount: 1, LinkedApprovalCount: 0, LinkedArtifactCount: 0, LinkedAuditEventCount: 0, HasIncompleteTurn: false}}}), true
	case "session_get":
		return mustOKLocalRPCResponse(t, brokerapi.SessionGetResponse{SchemaID: "runecode.protocol.v0.SessionGetResponse", SchemaVersion: "0.1.0", RequestID: "req-session-get", Session: brokerapi.SessionDetail{SchemaID: "runecode.protocol.v0.SessionDetail", SchemaVersion: "0.1.0", Summary: brokerapi.SessionSummary{SchemaID: "runecode.protocol.v0.SessionSummary", SchemaVersion: "0.1.0", Identity: brokerapi.SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z"}, UpdatedAt: "2026-01-01T00:00:00Z", Status: "active", LastActivityKind: "run_progress", TurnCount: 0, LinkedRunCount: 1, LinkedApprovalCount: 0, LinkedArtifactCount: 0, LinkedAuditEventCount: 0, HasIncompleteTurn: false}, LinkedRunIDs: []string{"run-1"}, LinkedApprovalIDs: []string{}, LinkedArtifactDigests: []string{}, LinkedAuditRecordDigests: []string{}}}), true
	case "session_send_message":
		return mustOKLocalRPCResponse(t, brokerapi.SessionSendMessageResponse{SchemaID: "runecode.protocol.v0.SessionSendMessageResponse", SchemaVersion: "0.1.0", RequestID: "req-session-send", SessionID: "sess-1", Turn: brokerapi.SessionTranscriptTurn{SchemaID: "runecode.protocol.v0.SessionTranscriptTurn", SchemaVersion: "0.1.0", TurnID: "sess-1.turn.000001", SessionID: "sess-1", TurnIndex: 1, StartedAt: "2026-01-01T00:00:00Z", CompletedAt: "2026-01-01T00:00:00Z", Status: "completed", Messages: []brokerapi.SessionTranscriptMessage{{SchemaID: "runecode.protocol.v0.SessionTranscriptMessage", SchemaVersion: "0.1.0", MessageID: "sess-1.turn.000001.msg.000001", TurnID: "sess-1.turn.000001", SessionID: "sess-1", MessageIndex: 1, Role: "user", CreatedAt: "2026-01-01T00:00:00Z", ContentText: "hello", RelatedLinks: brokerapi.SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{}, ApprovalIDs: []string{}, ArtifactDigests: []string{}, AuditRecordDigests: []string{}}}}}, Message: brokerapi.SessionTranscriptMessage{SchemaID: "runecode.protocol.v0.SessionTranscriptMessage", SchemaVersion: "0.1.0", MessageID: "sess-1.turn.000001.msg.000001", TurnID: "sess-1.turn.000001", SessionID: "sess-1", MessageIndex: 1, Role: "user", CreatedAt: "2026-01-01T00:00:00Z", ContentText: "hello", RelatedLinks: brokerapi.SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{}, ApprovalIDs: []string{}, ArtifactDigests: []string{}, AuditRecordDigests: []string{}}}, EventType: "session_message_ack", StreamID: "session-sess-1", Seq: 1}), true
	case "session_watch":
		return mustOKLocalRPCResponse(t, []brokerapi.SessionWatchEvent{{SchemaID: "runecode.protocol.v0.SessionWatchEvent", SchemaVersion: "0.1.0", StreamID: "sw-1", RequestID: "req-session-watch", Seq: 1, EventType: "session_watch_snapshot", Session: &brokerapi.SessionSummary{SchemaID: "runecode.protocol.v0.SessionSummary", SchemaVersion: "0.1.0", Identity: brokerapi.SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z"}, UpdatedAt: "2026-01-01T00:00:00Z", Status: "active", LastActivityKind: "chat_message", TurnCount: 1, LinkedRunCount: 1, LinkedApprovalCount: 0, LinkedArtifactCount: 0, LinkedAuditEventCount: 0, HasIncompleteTurn: false}}, {SchemaID: "runecode.protocol.v0.SessionWatchEvent", SchemaVersion: "0.1.0", StreamID: "sw-1", RequestID: "req-session-watch", Seq: 2, EventType: "session_watch_terminal", Terminal: true, TerminalStatus: "completed"}}), true
	default:
		return localRPCResponse{}, false
	}
}

func handleApprovalRPCStub(t *testing.T, wire localRPCRequest) (localRPCResponse, bool) {
	t.Helper()
	switch wire.Operation {
	case "approval_list":
		return mustOKLocalRPCResponse(t, brokerapi.ApprovalListResponse{SchemaID: "runecode.protocol.v0.ApprovalListResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-list"}), true
	case "approval_get":
		return mustOKLocalRPCResponse(t, brokerapi.ApprovalGetResponse{SchemaID: "runecode.protocol.v0.ApprovalGetResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-get", Approval: brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: testDigest("a")}, ApprovalDetail: brokerapi.ApprovalDetail{SchemaID: "runecode.protocol.v0.ApprovalDetail", SchemaVersion: "0.1.0", ApprovalID: testDigest("a"), LifecycleDetail: brokerapi.ApprovalLifecycleDetail{SchemaID: "runecode.protocol.v0.ApprovalLifecycleDetail", SchemaVersion: "0.1.0", LifecycleState: "pending", LifecycleReasonCode: "approval_pending", Stale: false}, BindingKind: "exact_action", BoundActionHash: testDigest("e"), WhatChangesIfApproved: brokerapi.ApprovalWhatChangesIfApproved{SchemaID: "runecode.protocol.v0.ApprovalWhatChangesIfApproved", SchemaVersion: "0.1.0", Summary: "Promote reviewed file excerpts for downstream use.", EffectKind: "promotion"}, BlockedWorkScope: brokerapi.ApprovalBlockedWorkScope{SchemaID: "runecode.protocol.v0.ApprovalBlockedWorkScope", SchemaVersion: "0.1.0", ScopeKind: "step", WorkspaceID: "workspace-local", RunID: "run-1", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion"}, BoundIdentity: brokerapi.ApprovalBoundIdentity{SchemaID: "runecode.protocol.v0.ApprovalBoundIdentity", SchemaVersion: "0.1.0", ApprovalRequestDigest: testDigest("a"), ManifestHash: testDigest("f"), BindingKind: "exact_action", BoundActionHash: testDigest("e")}}}), true
	case "approval_watch":
		return mustOKLocalRPCResponse(t, []brokerapi.ApprovalWatchEvent{{SchemaID: "runecode.protocol.v0.ApprovalWatchEvent", SchemaVersion: "0.1.0", StreamID: "aw-1", RequestID: "req-approval-watch", Seq: 1, EventType: "approval_watch_snapshot", Approval: &brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: testDigest("a"), Status: "pending", RequestedAt: "2026-01-01T00:00:00Z", ApprovalTriggerCode: "manual_approval_required", ChangesIfApproved: "Promote reviewed file excerpts for downstream use.", ApprovalAssuranceLevel: "session_authenticated", PresenceMode: "os_confirmation", BoundScope: brokerapi.ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", ActionKind: "promotion"}}}, {SchemaID: "runecode.protocol.v0.ApprovalWatchEvent", SchemaVersion: "0.1.0", StreamID: "aw-1", RequestID: "req-approval-watch", Seq: 2, EventType: "approval_watch_terminal", Terminal: true, TerminalStatus: "completed"}}), true
	default:
		return localRPCResponse{}, false
	}
}

func handleProviderProfileRPCStub(t *testing.T, wire localRPCRequest) (localRPCResponse, bool) {
	t.Helper()
	switch wire.Operation {
	case "git_setup_get":
		return mustOKLocalRPCResponse(t, brokerapi.GitSetupGetResponse{SchemaID: "runecode.protocol.v0.GitSetupGetResponse", SchemaVersion: "0.1.0", RequestID: "req-git-setup-get", ProviderAccount: brokerapi.GitProviderAccountState{SchemaID: "runecode.protocol.v0.GitProviderAccountState", SchemaVersion: "0.1.0", Provider: "github", AccountID: "not_linked", AccountUsername: "not_linked", Linked: false, Source: "restored_state"}, IdentityProfiles: []brokerapi.GitCommitIdentityProfile{{SchemaID: "runecode.protocol.v0.GitCommitIdentityProfile", SchemaVersion: "0.1.0", ProfileID: "default", DisplayName: "Default identity", AuthorName: "RuneCode Operator", AuthorEmail: "operator@example.invalid", CommitterName: "RuneCode Operator", CommitterEmail: "operator@example.invalid", SignoffName: "RuneCode Operator", SignoffEmail: "operator@example.invalid", DefaultProfile: true}}, AuthPosture: brokerapi.GitAuthPostureState{SchemaID: "runecode.protocol.v0.GitAuthPostureState", SchemaVersion: "0.1.0", Provider: "github", AuthStatus: "not_linked", BootstrapMode: "browser", HeadlessBootstrapSupported: true, InteractiveTokenFallbackSupport: true}, ControlPlaneState: brokerapi.GitControlPlaneState{SchemaID: "runecode.protocol.v0.GitControlPlaneState", SchemaVersion: "0.1.0", Provider: "github", DefaultIdentityProfileID: "default", LastSetupView: "overview", RecentRepositories: []string{}}, PolicySurface: brokerapi.GitPolicySurfaceState{ArtifactManagedOnly: true, InspectionSupported: true, PrepareChangesSupport: true, DirectMutationSupport: false}}), true
	case "git_setup_auth_bootstrap":
		return mustOKLocalRPCResponse(t, brokerapi.GitSetupAuthBootstrapResponse{SchemaID: "runecode.protocol.v0.GitSetupAuthBootstrapResponse", SchemaVersion: "0.1.0", RequestID: "req-git-setup-auth", Provider: "github", Mode: "browser", Status: "pending", AccountState: brokerapi.GitProviderAccountState{SchemaID: "runecode.protocol.v0.GitProviderAccountState", SchemaVersion: "0.1.0", Provider: "github", AccountID: "pending", AccountUsername: "pending", Linked: false, Source: "auth_bootstrap"}, AuthPosture: brokerapi.GitAuthPostureState{SchemaID: "runecode.protocol.v0.GitAuthPostureState", SchemaVersion: "0.1.0", Provider: "github", AuthStatus: "not_linked", BootstrapMode: "browser", HeadlessBootstrapSupported: true, InteractiveTokenFallbackSupport: true}}), true
	case "git_setup_identity_upsert":
		return mustOKLocalRPCResponse(t, brokerapi.GitSetupIdentityUpsertResponse{SchemaID: "runecode.protocol.v0.GitSetupIdentityUpsertResponse", SchemaVersion: "0.1.0", RequestID: "req-git-setup-identity", Provider: "github", Profile: brokerapi.GitCommitIdentityProfile{SchemaID: "runecode.protocol.v0.GitCommitIdentityProfile", SchemaVersion: "0.1.0", ProfileID: "default", DisplayName: "Default identity", AuthorName: "RuneCode Operator", AuthorEmail: "operator@example.invalid", CommitterName: "RuneCode Operator", CommitterEmail: "operator@example.invalid", SignoffName: "RuneCode Operator", SignoffEmail: "operator@example.invalid", DefaultProfile: true}, ControlPlaneState: brokerapi.GitControlPlaneState{SchemaID: "runecode.protocol.v0.GitControlPlaneState", SchemaVersion: "0.1.0", Provider: "github", DefaultIdentityProfileID: "default", LastSetupView: "identity", RecentRepositories: []string{}}}), true
	case "provider_profile_list":
		return mustOKLocalRPCResponse(t, brokerapi.ProviderProfileListResponse{SchemaID: "runecode.protocol.v0.ProviderProfileListResponse", SchemaVersion: "0.1.0", RequestID: "req-provider-list", Profiles: []brokerapi.ProviderProfile{{SchemaID: "runecode.protocol.v0.ProviderProfile", SchemaVersion: "0.1.0", ProviderProfileID: "provider-profile-test", DisplayLabel: "Provider Test", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", DestinationRef: "model_endpoint://api.openai.com/v1", SupportedAuthModes: []string{"direct_credential"}, CurrentAuthMode: "direct_credential", AllowlistedModelIDs: []string{}, ModelCatalogPosture: brokerapi.ProviderModelCatalogPosture{SchemaID: "runecode.protocol.v0.ProviderModelCatalogPosture", SchemaVersion: "0.1.0", SelectionAuthority: "manual_allowlist_canonical", DiscoveryPosture: "advisory", CompatibilityProbePosture: "advisory"}, CompatibilityPosture: "unverified", QuotaProfileKind: "hybrid", RequestBindingKind: "canonical_llm_request_digest", SurfaceChannel: "broker_local_api", AuthMaterial: brokerapi.ProviderAuthMaterial{SchemaID: "runecode.protocol.v0.ProviderAuthMaterial", SchemaVersion: "0.1.0", MaterialKind: "direct_credential", MaterialState: "present"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{SchemaID: "runecode.protocol.v0.ProviderReadinessPosture", SchemaVersion: "0.1.0", ConfigurationState: "configured", CredentialState: "present", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"}, Lifecycle: brokerapi.ProviderLifecycleMetadata{CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}}}}), true
	case "provider_profile_get":
		return mustOKLocalRPCResponse(t, brokerapi.ProviderProfileGetResponse{SchemaID: "runecode.protocol.v0.ProviderProfileGetResponse", SchemaVersion: "0.1.0", RequestID: "req-provider-get", Profile: brokerapi.ProviderProfile{SchemaID: "runecode.protocol.v0.ProviderProfile", SchemaVersion: "0.1.0", ProviderProfileID: "provider-profile-test", DisplayLabel: "Provider Test", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", DestinationRef: "model_endpoint://api.openai.com/v1", SupportedAuthModes: []string{"direct_credential"}, CurrentAuthMode: "direct_credential", AllowlistedModelIDs: []string{}, ModelCatalogPosture: brokerapi.ProviderModelCatalogPosture{SchemaID: "runecode.protocol.v0.ProviderModelCatalogPosture", SchemaVersion: "0.1.0", SelectionAuthority: "manual_allowlist_canonical", DiscoveryPosture: "advisory", CompatibilityProbePosture: "advisory"}, CompatibilityPosture: "unverified", QuotaProfileKind: "hybrid", RequestBindingKind: "canonical_llm_request_digest", SurfaceChannel: "broker_local_api", AuthMaterial: brokerapi.ProviderAuthMaterial{SchemaID: "runecode.protocol.v0.ProviderAuthMaterial", SchemaVersion: "0.1.0", MaterialKind: "direct_credential", MaterialState: "present"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{SchemaID: "runecode.protocol.v0.ProviderReadinessPosture", SchemaVersion: "0.1.0", ConfigurationState: "configured", CredentialState: "present", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"}, Lifecycle: brokerapi.ProviderLifecycleMetadata{CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}}}), true
	case "provider_credential_lease_issue":
		return mustOKLocalRPCResponse(t, brokerapi.ProviderCredentialLeaseIssueResponse{SchemaID: "runecode.protocol.v0.ProviderCredentialLeaseIssueResponse", SchemaVersion: "0.1.0", RequestID: "req-provider-lease", ProviderProfileID: "provider-profile-test", ProviderAuthLeaseID: "lease-1", Lease: secretsd.Lease{LeaseID: "lease-1", SecretRef: "secrets/model-providers/provider-profile-test/direct-credential", ConsumerID: "principal:gateway:model:1", RoleKind: "model-gateway", Scope: "run:run-1", DeliveryKind: "model_gateway", Status: "active"}}), true
	default:
		return localRPCResponse{}, false
	}
}

func handleProjectSubstrateRPCStub(t *testing.T, wire localRPCRequest) (localRPCResponse, bool) {
	t.Helper()
	switch wire.Operation {
	case "project_substrate_get":
		return mustOKLocalRPCResponse(t, brokerapi.ProjectSubstrateGetResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateGetResponse", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-get", RepositoryRoot: "/repo", Contract: brokerapi.ProjectSubstrateGetResponse{}.Contract, Snapshot: brokerapi.ProjectSubstrateGetResponse{}.Snapshot}), true
	case "project_substrate_posture_get":
		return mustOKLocalRPCResponse(t, brokerapi.ProjectSubstratePostureGetResponse{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureGetResponse", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-posture-get", RepositoryRoot: "/repo", Contract: brokerapi.ProjectSubstrateGetResponse{}.Contract, Snapshot: brokerapi.ProjectSubstrateGetResponse{}.Snapshot, PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", SchemaVersion: "0.1.0", ActiveContractID: "runecode.runecontext.project_substrate.v0", ActiveContractVersion: "v0", ContractID: "runecode.runecontext.project_substrate.v0", ContractVersion: "v0", ValidationState: "valid", CompatibilityPosture: "supported_current", NormalOperationAllowed: true, SupportedContractVersionMin: "v0", SupportedContractVersionMax: "v0", RecommendedContractVersion: "v0", SupportedRuneContextMin: "0.1.0-alpha.13", SupportedRuneContextMax: "0.1.0-alpha.16", RecommendedRuneContextTarget: "0.1.0-alpha.14"}, Adoption: brokerapi.ProjectSubstrateAdoptResponse{}.Adoption, InitPreview: brokerapi.ProjectSubstrateInitPreviewResponse{}.Preview, UpgradePreview: brokerapi.ProjectSubstrateUpgradePreviewResponse{}.Preview}), true
	case "project_substrate_adopt":
		return mustOKLocalRPCResponse(t, brokerapi.ProjectSubstrateAdoptResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateAdoptResponse", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-adopt", Adoption: brokerapi.ProjectSubstrateAdoptResponse{}.Adoption}), true
	case "project_substrate_init_preview":
		return mustOKLocalRPCResponse(t, brokerapi.ProjectSubstrateInitPreviewResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateInitPreviewResponse", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-init-preview", Preview: brokerapi.ProjectSubstrateInitPreviewResponse{}.Preview}), true
	case "project_substrate_init_apply":
		return mustOKLocalRPCResponse(t, brokerapi.ProjectSubstrateInitApplyResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateInitApplyResponse", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-init-apply", ApplyResult: brokerapi.ProjectSubstrateInitApplyResponse{}.ApplyResult}), true
	case "project_substrate_upgrade_preview":
		return mustOKLocalRPCResponse(t, brokerapi.ProjectSubstrateUpgradePreviewResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradePreviewResponse", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-upgrade-preview", Preview: brokerapi.ProjectSubstrateUpgradePreviewResponse{}.Preview}), true
	case "project_substrate_upgrade_apply":
		return mustOKLocalRPCResponse(t, brokerapi.ProjectSubstrateUpgradeApplyResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradeApplyResponse", SchemaVersion: "0.1.0", RequestID: "req-project-substrate-upgrade-apply", ApplyResult: brokerapi.ProjectSubstrateUpgradeApplyResponse{}.ApplyResult}), true
	default:
		return localRPCResponse{}, false
	}
}

func handleGitRemoteAndLLMRPCStub(t *testing.T, wire localRPCRequest, opCount int) (localRPCResponse, bool) {
	t.Helper()
	switch wire.Operation {
	case "git_remote_mutation_prepare":
		return mustOKLocalRPCResponse(t, brokerapi.GitRemoteMutationPrepareResponse{}), true
	case "git_remote_mutation_get":
		return mustOKLocalRPCResponse(t, brokerapi.GitRemoteMutationGetResponse{}), true
	case "git_remote_mutation_issue_execute_lease":
		return mustOKLocalRPCResponse(t, brokerapi.GitRemoteMutationIssueExecuteLeaseResponse{ProviderAuthLeaseID: "lease-1", Lease: secretsd.Lease{LeaseID: "lease-1", SecretRef: "secrets/prod/git/provider-token", ConsumerID: "principal:gateway:git:1", RoleKind: "git-gateway", Scope: "run:run-1", DeliveryKind: "git_gateway", Status: "active"}}), true
	case "git_remote_mutation_execute":
		return mustOKLocalRPCResponse(t, brokerapi.GitRemoteMutationExecuteResponse{}), true
	case "version_info_get":
		return mustOKLocalRPCResponse(t, brokerapi.VersionInfoGetResponse{SchemaID: "runecode.protocol.v0.VersionInfoGetResponse", SchemaVersion: "0.1.0", RequestID: "req-version", VersionInfo: brokerapi.BrokerVersionInfo{SchemaID: "runecode.protocol.v0.BrokerVersionInfo", SchemaVersion: "0.1.0"}}), true
	case "log_stream":
		return handleLogStreamDispatchForTest(t, wire, opCount), true
	case "llm_invoke":
		return mustOKLocalRPCResponse(t, brokerapi.LLMInvokeResponse{SchemaID: "runecode.protocol.v0.LLMInvokeResponse", SchemaVersion: "0.1.0", RequestID: "req-llm-invoke", RunID: "run-1", RequestDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}, Response: map[string]any{"schema_id": "runecode.protocol.v0.LLMResponse", "schema_version": "0.3.0"}}), true
	case "llm_stream":
		return mustOKLocalRPCResponse(t, brokerapi.LLMStreamEnvelope{SchemaID: "runecode.protocol.v0.LLMStreamEnvelope", SchemaVersion: "0.1.0", RequestID: "req-llm-stream", RunID: "run-1", RequestDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}, Events: []brokerapi.LLMStreamAny{{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": "llm-s-1", "request_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)}, "seq": 1.0, "event_type": "response_start", "emitter": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "role_instance", "principal_id": "brokerapi", "instance_id": "brokerapi-1", "role_family": "gateway", "role_kind": "model-gateway"}}, {"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": "llm-s-1", "request_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)}, "seq": 2.0, "event_type": "response_terminal", "terminal_status": "success", "final_response_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("2", 64)}, "emitter": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "role_instance", "principal_id": "brokerapi", "instance_id": "brokerapi-1", "role_family": "gateway", "role_kind": "model-gateway"}}}}), true
	default:
		return localRPCResponse{}, false
	}
}

func writeLLMRequestFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "llm-request.json")
	requestDigest := strings.Repeat("1", 64)
	provenanceDigest := strings.Repeat("2", 64)
	payload := map[string]any{
		"schema_id":        "runecode.protocol.v0.LLMRequest",
		"schema_version":   "0.3.0",
		"selection_source": "signed_allowlist",
		"provider":         "provider-test",
		"model":            "model-test",
		"tool_allowlist": []any{
			map[string]any{
				"tool_name":                "noop",
				"arguments_schema_id":      "runecode.protocol.tools.noop.args",
				"arguments_schema_version": "0.1.0",
			},
		},
		"input_artifacts": []any{
			map[string]any{
				"schema_id":      "runecode.protocol.v0.ArtifactReference",
				"schema_version": "0.3.0",
				"digest":         map[string]any{"hash_alg": "sha256", "hash": requestDigest},
				"size_bytes":     5,
				"content_type":   "text/plain",
				"data_class":     "spec_text",
				"provenance_receipt_hash": map[string]any{
					"hash_alg": "sha256",
					"hash":     provenanceDigest,
				},
			},
		},
		"response_mode":  "text",
		"streaming_mode": "stream",
		"request_limits": map[string]any{"max_request_bytes": 262144, "max_tool_calls": 8, "max_total_tool_call_argument_bytes": 65536, "max_structured_output_bytes": 262144, "max_streamed_bytes": 16777216, "max_stream_chunk_bytes": 65536, "stream_idle_timeout_ms": 15000},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal llm request payload error: %v", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("WriteFile llm request payload error: %v", err)
	}
	return path
}

func writeGitRemoteMutationRequestFiles(t *testing.T) (string, string, string, string) {
	t.Helper()
	dir := t.TempDir()
	preparePath := filepath.Join(dir, "git-remote-prepare.json")
	getPath := filepath.Join(dir, "git-remote-get.json")
	leasePath := filepath.Join(dir, "git-remote-issue-execute-lease.json")
	executePath := filepath.Join(dir, "git-remote-execute.json")
	preparePayload := map[string]any{
		"schema_id":      "runecode.protocol.v0.GitRemoteMutationPrepareRequest",
		"schema_version": "0.1.0",
		"request_id":     "req-git-prepare",
		"run_id":         "run-1",
		"provider":       "github",
		"typed_request": map[string]any{
			"schema_id":                         "runecode.protocol.v0.GitRefUpdateRequest",
			"schema_version":                    "0.1.0",
			"request_kind":                      "git_ref_update",
			"target_ref":                        "refs/heads/main",
			"repository_identity":               map[string]any{"canonical_host": "github.com", "canonical_path_prefix": "runecode-ai/runecode", "git_repository_identity": "github.com/runecode-ai/runecode"},
			"expected_old_ref_hash":             map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("2", 64)},
			"referenced_patch_artifact_digests": []any{map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("3", 64)}},
			"expected_result_tree_hash":         map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("4", 64)},
		},
	}
	getPayload := map[string]any{
		"schema_id":            "runecode.protocol.v0.GitRemoteMutationGetRequest",
		"schema_version":       "0.1.0",
		"request_id":           "req-git-get",
		"prepared_mutation_id": testDigest("1"),
	}
	leasePayload := map[string]any{
		"schema_id":            "runecode.protocol.v0.GitRemoteMutationIssueExecuteLeaseRequest",
		"schema_version":       "0.1.0",
		"request_id":           "req-git-issue-execute-lease",
		"prepared_mutation_id": testDigest("1"),
	}
	executePayload := map[string]any{
		"schema_id":              "runecode.protocol.v0.GitRemoteMutationExecuteRequest",
		"schema_version":         "0.1.0",
		"request_id":             "req-git-execute",
		"prepared_mutation_id":   testDigest("1"),
		"approval_id":            testDigest("a"),
		"approval_request_hash":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("5", 64)},
		"approval_decision_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("6", 64)},
		"provider_auth_lease_id": "lease-1",
	}
	writeJSONFixtureFile(t, preparePath, preparePayload)
	writeJSONFixtureFile(t, getPath, getPayload)
	writeJSONFixtureFile(t, leasePath, leasePayload)
	writeJSONFixtureFile(t, executePath, executePayload)
	return preparePath, getPath, leasePath, executePath
}

func writeJSONFixtureFile(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal(%q) error: %v", path, err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error: %v", path, err)
	}
}

func handleLogStreamDispatchForTest(t *testing.T, wire localRPCRequest, opCount int) localRPCResponse {
	t.Helper()
	_ = opCount
	request := brokerapi.LogStreamRequest{}
	if err := json.Unmarshal(wire.Request, &request); err != nil {
		t.Fatalf("Unmarshal log stream request error: %v", err)
	}
	if request.StreamID == "custom-stream" {
		return mustOKLocalRPCResponse(t, []brokerapi.LogStreamEvent{{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-log", Seq: 1, EventType: "log_stream_start"}, {SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-log", Seq: 2, EventType: "log_stream_terminal", Terminal: true, TerminalStatus: "completed"}})
	}
	if request.StreamID == "" || request.StreamID == request.RequestID {
		t.Fatalf("default stream-logs request stream_id = %q, want derived non-empty stream id", request.StreamID)
	}
	return mustOKLocalRPCResponse(t, []brokerapi.LogStreamEvent{{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-log", Seq: 1, EventType: "log_stream_start"}, {SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-log", Seq: 2, EventType: "log_stream_terminal", Terminal: true, TerminalStatus: "completed"}})
}

func TestCLIAdoptionRoutesArtifactAuditAndResolveThroughLocalRPC(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	requestedOps := make([]string, 0, 8)

	installArtifactAuditResolveDispatchStub(t, &requestedOps)
	runArtifactAuditResolveCommands(t, stdout, stderr)

	want := []string{"artifact_list", "artifact_head", "artifact_read", "approval_get", "approval_resolve", "readiness_get", "audit_verification_get", "audit_finalize_verify", "audit_record_get", "audit_anchor_preflight_get", "audit_anchor_presence_get", "audit_anchor_segment"}
	assertRequestedOps(t, requestedOps, want)
}

func installArtifactAuditResolveDispatchStub(t *testing.T, requestedOps *[]string) {
	t.Helper()
	originalDispatch := localRPCDispatch
	localRPCDispatch = func(_ *brokerapi.Service, _ context.Context, wire localRPCRequest, _ brokerapi.RequestContext) localRPCResponse {
		*requestedOps = append(*requestedOps, wire.Operation)
		if resp, ok := artifactAuditResolveStaticResponse(t, wire.Operation); ok {
			return resp
		}
		return artifactAuditResolveDynamicResponse(t, wire)
	}
	t.Cleanup(func() { localRPCDispatch = originalDispatch })
}

func artifactAuditResolveStaticResponse(t *testing.T, operation string) (localRPCResponse, bool) {
	t.Helper()
	switch operation {
	case "artifact_list":
		return mustOKLocalRPCResponse(t, brokerapi.LocalArtifactListResponse{SchemaID: "runecode.protocol.v0.ArtifactListResponse", SchemaVersion: "0.1.0", RequestID: "req-art-list", Artifacts: []brokerapi.ArtifactSummary{{SchemaID: "runecode.protocol.v0.ArtifactSummary", SchemaVersion: "0.1.0", Reference: artifacts.ArtifactReference{Digest: testDigest("b"), DataClass: artifacts.DataClassSpecText}, CreatedAt: "2026-01-01T00:00:00Z", CreatedByRole: "workspace"}}}), true
	case "artifact_head":
		return mustOKLocalRPCResponse(t, brokerapi.LocalArtifactHeadResponse{SchemaID: "runecode.protocol.v0.ArtifactHeadResponse", SchemaVersion: "0.1.0", RequestID: "req-art-head", Artifact: brokerapi.ArtifactSummary{SchemaID: "runecode.protocol.v0.ArtifactSummary", SchemaVersion: "0.1.0", Reference: artifacts.ArtifactReference{Digest: testDigest("b"), DataClass: artifacts.DataClassSpecText}, CreatedAt: "2026-01-01T00:00:00Z", CreatedByRole: "workspace"}}), true
	case "artifact_read":
		return mustOKLocalRPCResponse(t, []brokerapi.ArtifactStreamEvent{{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-art-read", Seq: 1, EventType: "artifact_stream_start", Digest: testDigest("b"), DataClass: "spec_text"}, {SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-art-read", Seq: 2, EventType: "artifact_stream_chunk", Digest: testDigest("b"), DataClass: "spec_text", ChunkBase64: base64.StdEncoding.EncodeToString([]byte("hello")), ChunkBytes: 5}, {SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-art-read", Seq: 3, EventType: "artifact_stream_terminal", Digest: testDigest("b"), DataClass: "spec_text", Terminal: true, TerminalStatus: "completed"}}), true
	case "approval_resolve":
		return mustOKLocalRPCResponse(t, brokerapi.ApprovalResolveResponse{SchemaID: "runecode.protocol.v0.ApprovalResolveResponse", SchemaVersion: "0.1.0", RequestID: "req-resolve", ResolutionStatus: "resolved", ResolutionReasonCode: "approval_approved", Approval: brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: testDigest("c")}, ApprovedArtifact: &brokerapi.ArtifactSummary{SchemaID: "runecode.protocol.v0.ArtifactSummary", SchemaVersion: "0.1.0", Reference: artifacts.ArtifactReference{Digest: testDigest("d"), DataClass: artifacts.DataClassApprovedFileExcerpts}, CreatedAt: "2026-01-01T00:00:00Z", CreatedByRole: "workspace"}}), true
	case "readiness_get":
		return mustOKLocalRPCResponse(t, brokerapi.ReadinessGetResponse{SchemaID: "runecode.protocol.v0.ReadinessGetResponse", SchemaVersion: "0.1.0", RequestID: "req-readiness", Readiness: brokerapi.BrokerReadiness{SchemaID: "runecode.protocol.v0.BrokerReadiness", SchemaVersion: "0.1.0", Ready: true, LocalOnly: true, ConsumptionChannel: "broker_local_api", RecoveryComplete: true, AppendPositionStable: true, CurrentSegmentWritable: true, VerifierMaterialAvailable: true, DerivedIndexCaughtUp: true}}), true
	case "audit_verification_get":
		return mustOKLocalRPCResponse(t, brokerapi.AuditVerificationGetResponse{SchemaID: "runecode.protocol.v0.AuditVerificationGetResponse", SchemaVersion: "0.1.0", RequestID: "req-audit"}), true
	case "audit_finalize_verify":
		report := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("e", 64)}
		return mustOKLocalRPCResponse(t, brokerapi.AuditFinalizeVerifyResponse{SchemaID: "runecode.protocol.v0.AuditFinalizeVerifyResponse", SchemaVersion: "0.1.0", RequestID: "req-audit-finalize", ActionStatus: "ok", SegmentID: "segment-000001", ReportDigest: &report}), true
	default:
		return localRPCResponse{}, false
	}
}

func artifactAuditResolveDynamicResponse(t *testing.T, wire localRPCRequest) localRPCResponse {
	t.Helper()
	switch wire.Operation {
	case "approval_get":
		return artifactAuditResolveApprovalGetResponse(t, wire)
	case "audit_record_get":
		return mustOKLocalRPCResponse(t, brokerapi.AuditRecordGetResponse{SchemaID: "runecode.protocol.v0.AuditRecordGetResponse", SchemaVersion: "0.1.0", RequestID: "req-audit-record", Record: brokerapi.AuditRecordDetail{SchemaID: "runecode.protocol.v0.AuditRecordDetail", SchemaVersion: "0.1.0", RecordDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)}, RecordFamily: "audit_event", OccurredAt: "2026-01-01T00:00:00Z", EventType: "isolate_session_bound", Summary: "Audit event isolate_session_bound recorded.", LinkedReferences: []brokerapi.AuditRecordLinkedReference{}}})
	case "audit_anchor_preflight_get":
		return artifactAuditResolveAnchorPreflightResponse(t, wire)
	case "audit_anchor_segment":
		return artifactAuditResolveAnchorSegmentResponse(t, wire)
	case "audit_anchor_presence_get":
		return artifactAuditResolveAnchorPresenceResponse(t, wire)
	default:
		return localRPCResponse{OK: false}
	}
}

func artifactAuditResolveApprovalGetResponse(t *testing.T, wire localRPCRequest) localRPCResponse {
	t.Helper()
	request := brokerapi.ApprovalGetRequest{}
	if err := json.Unmarshal(wire.Request, &request); err != nil {
		t.Fatalf("Unmarshal approval_get request error: %v", err)
	}
	return mustOKLocalRPCResponse(t, brokerapi.ApprovalGetResponse{SchemaID: "runecode.protocol.v0.ApprovalGetResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-get", Approval: brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: request.ApprovalID, BoundScope: brokerapi.ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", ActionKind: "promotion"}}, ApprovalDetail: brokerapi.ApprovalDetail{SchemaID: "runecode.protocol.v0.ApprovalDetail", SchemaVersion: "0.1.0", ApprovalID: request.ApprovalID, PolicyReasonCode: "approval_required", LifecycleDetail: brokerapi.ApprovalLifecycleDetail{SchemaID: "runecode.protocol.v0.ApprovalLifecycleDetail", SchemaVersion: "0.1.0", LifecycleState: "pending", LifecycleReasonCode: "approval_pending", Stale: false}, BindingKind: "exact_action", BoundActionHash: testDigest("e"), WhatChangesIfApproved: brokerapi.ApprovalWhatChangesIfApproved{SchemaID: "runecode.protocol.v0.ApprovalWhatChangesIfApproved", SchemaVersion: "0.1.0", Summary: "Promote reviewed file excerpts for downstream use.", EffectKind: "promotion"}, BlockedWorkScope: brokerapi.ApprovalBlockedWorkScope{SchemaID: "runecode.protocol.v0.ApprovalBlockedWorkScope", SchemaVersion: "0.1.0", ScopeKind: "action_kind", ActionKind: "promotion"}, BoundIdentity: brokerapi.ApprovalBoundIdentity{SchemaID: "runecode.protocol.v0.ApprovalBoundIdentity", SchemaVersion: "0.1.0", ApprovalRequestDigest: request.ApprovalID, ManifestHash: testDigest("f"), BindingKind: "exact_action", BoundActionHash: testDigest("e")}}})
}

func artifactAuditResolveAnchorSegmentResponse(t *testing.T, wire localRPCRequest) localRPCResponse {
	t.Helper()
	request := brokerapi.AuditAnchorSegmentRequest{}
	if err := json.Unmarshal(wire.Request, &request); err != nil {
		t.Fatalf("Unmarshal audit_anchor_segment request error: %v", err)
	}
	assertAuditAnchorPresenceAttestationForCLI(t, request.PresenceAttestation)
	receipt := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("c", 64)}
	report := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("d", 64)}
	return mustOKLocalRPCResponse(t, brokerapi.AuditAnchorSegmentResponse{SchemaID: "runecode.protocol.v0.AuditAnchorSegmentResponse", SchemaVersion: "0.1.0", RequestID: "req-audit-anchor", SealDigest: request.SealDigest, ReceiptDigest: &receipt, VerificationReportDigest: &report, AnchoringStatus: "ok"})
}

func artifactAuditResolveAnchorPresenceResponse(t *testing.T, wire localRPCRequest) localRPCResponse {
	t.Helper()
	request := brokerapi.AuditAnchorPresenceGetRequest{}
	if err := json.Unmarshal(wire.Request, &request); err != nil {
		t.Fatalf("Unmarshal audit_anchor_presence_get request error: %v", err)
	}
	if _, err := request.SealDigest.Identity(); err != nil {
		t.Fatalf("audit_anchor_presence_get invalid seal_digest: %v", err)
	}
	return mustOKLocalRPCResponse(t, brokerapi.AuditAnchorPresenceGetResponse{SchemaID: "runecode.protocol.v0.AuditAnchorPresenceGetResponse", SchemaVersion: "0.1.0", RequestID: "req-audit-presence", SealDigest: request.SealDigest, PresenceMode: "os_confirmation", PresenceAttestation: &brokerapi.AuditAnchorPresenceAttestation{Challenge: "presence-challenge-0123456789abcdef", AcknowledgmentToken: strings.Repeat("a", 64)}})
}

func artifactAuditResolveAnchorPreflightResponse(t *testing.T, wire localRPCRequest) localRPCResponse {
	t.Helper()
	request := brokerapi.AuditAnchorPreflightGetRequest{}
	if err := json.Unmarshal(wire.Request, &request); err != nil {
		t.Fatalf("Unmarshal audit_anchor_preflight_get request error: %v", err)
	}
	seal := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)}
	return mustOKLocalRPCResponse(t, brokerapi.AuditAnchorPreflightGetResponse{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPreflightGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-preflight",
		LatestAnchorableSeal: &brokerapi.AuditAnchorableSealRef{
			SegmentID:  "segment-000001",
			SealDigest: seal,
		},
		SignerReadiness:      brokerapi.AuditAnchorSignerReadiness{Ready: true, PresenceMode: "os_confirmation", SignerLogicalScope: "node"},
		VerifierReadiness:    brokerapi.AuditAnchorVerifierReadiness{Ready: true},
		PresenceRequirements: brokerapi.AuditAnchorPresenceRequirements{Required: true, AttestationMode: "os_confirmation", AttestationReady: true},
		ApprovalRequirements: brokerapi.AuditAnchorApprovalRequirements{Required: false, ReasonCode: "approval_not_required", Message: "no approval requirement declared"},
	})
}

func assertAuditAnchorPresenceAttestationForCLI(t *testing.T, att *brokerapi.AuditAnchorPresenceAttestation) {
	t.Helper()
	if att == nil {
		t.Fatal("audit_anchor_segment request missing presence attestation")
	}
	if strings.TrimSpace(att.Challenge) == "" {
		t.Fatal("audit_anchor_segment presence challenge is empty")
	}
	if len(att.AcknowledgmentToken) != 64 {
		t.Fatalf("audit_anchor_segment presence acknowledgment token length = %d, want 64", len(att.AcknowledgmentToken))
	}
}

func runArtifactAuditResolveCommands(t *testing.T, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	t.Helper()
	outPath := filepath.Join(t.TempDir(), "artifact.out")
	if err := run([]string{"list-artifacts"}, stdout, stderr); err != nil {
		t.Fatalf("list-artifacts returned error: %v", err)
	}
	if err := run([]string{"head-artifact", "--digest", testDigest("b")}, stdout, stderr); err != nil {
		t.Fatalf("head-artifact returned error: %v", err)
	}
	if err := run([]string{"get-artifact", "--digest", testDigest("b"), "--producer", "workspace", "--consumer", "model_gateway", "--out", outPath}, stdout, stderr); err != nil {
		t.Fatalf("get-artifact returned error: %v", err)
	}
	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", outPath, err)
	}
	if string(payload) != "hello" {
		t.Fatalf("artifact payload = %q, want hello", string(payload))
	}

	approvalRequestPath, approvalEnvelopePath, _ := writeApprovalFixtures(t, "human", testDigest("2"), "repo/file.txt", "abc123", "tool-v1")
	seedPendingPromotionApprovalForCLI(t, testDigest("2"), approvalRequestPath)
	if err := run([]string{"promote-excerpt", "--unapproved-digest", testDigest("2"), "--approver", "human", "--approval-request", approvalRequestPath, "--approval-envelope", approvalEnvelopePath, "--repo-path", "repo/file.txt", "--commit", "abc123", "--extractor-version", "tool-v1", "--full-content-visible"}, stdout, stderr); err != nil {
		t.Fatalf("promote-excerpt returned error: %v", err)
	}
	if err := run([]string{"audit-readiness"}, stdout, stderr); err != nil {
		t.Fatalf("audit-readiness returned error: %v", err)
	}
	if err := run([]string{"audit-verification"}, stdout, stderr); err != nil {
		t.Fatalf("audit-verification returned error: %v", err)
	}
	if err := run([]string{"audit-finalize-verify"}, stdout, stderr); err != nil {
		t.Fatalf("audit-finalize-verify returned error: %v", err)
	}
	if err := run([]string{"audit-record-get", "--record-digest", testDigest("a")}, stdout, stderr); err != nil {
		t.Fatalf("audit-record-get returned error: %v", err)
	}
	if err := run([]string{"audit-anchor-segment", "--seal-digest", testDigest("a")}, stdout, stderr); err != nil {
		t.Fatalf("audit-anchor-segment returned error: %v", err)
	}
}

func mustOKLocalRPCResponse(t *testing.T, value any) localRPCResponse {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal local RPC payload error: %v", err)
	}
	return localRPCResponse{OK: true, Response: json.RawMessage(b)}
}

func assertRequestedOps(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("requested operations = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("operation[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
