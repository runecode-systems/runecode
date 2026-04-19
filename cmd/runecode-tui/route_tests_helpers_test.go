package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

var testSignedApprovalRequest = &trustpolicy.SignedObjectEnvelope{
	SchemaID:             trustpolicy.EnvelopeSchemaID,
	SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
	PayloadSchemaID:      trustpolicy.ApprovalRequestSchemaID,
	PayloadSchemaVersion: trustpolicy.ApprovalRequestSchemaVersion,
	Payload:              []byte(`{"schema_id":"runecode.protocol.v0.ApprovalRequest","schema_version":"0.3.0"}`),
	SignatureInput:       trustpolicy.SignatureInputProfile,
	Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: "c2ln"},
}

var testSignedApprovalDecision = &trustpolicy.SignedObjectEnvelope{
	SchemaID:             trustpolicy.EnvelopeSchemaID,
	SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
	PayloadSchemaID:      trustpolicy.ApprovalDecisionSchemaID,
	PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion,
	Payload:              []byte(`{"schema_id":"runecode.protocol.v0.ApprovalDecision","schema_version":"0.3.0"}`),
	SignatureInput:       trustpolicy.SignatureInputProfile,
	Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("b", 64), Signature: "c2ln"},
}

type fakeBrokerClient struct{}

type reloadAwareBrokerClient struct{}

type backendResolveReadyBrokerClient struct{ *fakeBrokerClient }

type recordingBrokerClient struct {
	base  localBrokerClient
	calls []string
}

func newRecordingBrokerClient(base localBrokerClient) *recordingBrokerClient {
	return &recordingBrokerClient{base: base}
}

func (r *recordingBrokerClient) record(call string) {
	r.calls = append(r.calls, call)
}

func (r *recordingBrokerClient) Calls() []string {
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

func (r *recordingBrokerClient) RunList(ctx context.Context, limit int) (brokerapi.RunListResponse, error) {
	r.record("RunList")
	return r.base.RunList(ctx, limit)
}

func (r *recordingBrokerClient) RunGet(ctx context.Context, runID string) (brokerapi.RunGetResponse, error) {
	r.record("RunGet")
	return r.base.RunGet(ctx, runID)
}

func (r *recordingBrokerClient) RunWatch(ctx context.Context, req brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, error) {
	r.record("RunWatch")
	return r.base.RunWatch(ctx, req)
}

func (r *recordingBrokerClient) SessionList(ctx context.Context, limit int) (brokerapi.SessionListResponse, error) {
	r.record("SessionList")
	return r.base.SessionList(ctx, limit)
}

func (r *recordingBrokerClient) SessionGet(ctx context.Context, sessionID string) (brokerapi.SessionGetResponse, error) {
	r.record("SessionGet")
	return r.base.SessionGet(ctx, sessionID)
}

func (r *recordingBrokerClient) SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, error) {
	r.record("SessionSendMessage")
	return r.base.SessionSendMessage(ctx, req)
}

func (r *recordingBrokerClient) SessionWatch(ctx context.Context, req brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, error) {
	r.record("SessionWatch")
	return r.base.SessionWatch(ctx, req)
}

func (r *recordingBrokerClient) ApprovalList(ctx context.Context, limit int) (brokerapi.ApprovalListResponse, error) {
	r.record("ApprovalList")
	return r.base.ApprovalList(ctx, limit)
}

func (r *recordingBrokerClient) ApprovalGet(ctx context.Context, approvalID string) (brokerapi.ApprovalGetResponse, error) {
	r.record("ApprovalGet")
	return r.base.ApprovalGet(ctx, approvalID)
}

func (r *recordingBrokerClient) ApprovalResolve(ctx context.Context, req brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, error) {
	r.record("ApprovalResolve")
	return r.base.ApprovalResolve(ctx, req)
}

func (r *recordingBrokerClient) ApprovalWatch(ctx context.Context, req brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, error) {
	r.record("ApprovalWatch")
	return r.base.ApprovalWatch(ctx, req)
}

func (r *recordingBrokerClient) BackendPostureGet(ctx context.Context) (brokerapi.BackendPostureGetResponse, error) {
	r.record("BackendPostureGet")
	return r.base.BackendPostureGet(ctx)
}

func (r *recordingBrokerClient) BackendPostureChange(ctx context.Context, req brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, error) {
	r.record("BackendPostureChange")
	return r.base.BackendPostureChange(ctx, req)
}

func (r *recordingBrokerClient) ArtifactList(ctx context.Context, limit int, dataClass string) (brokerapi.LocalArtifactListResponse, error) {
	r.record("ArtifactList")
	return r.base.ArtifactList(ctx, limit, dataClass)
}

func (r *recordingBrokerClient) ArtifactHead(ctx context.Context, digest string) (brokerapi.LocalArtifactHeadResponse, error) {
	r.record("ArtifactHead")
	return r.base.ArtifactHead(ctx, digest)
}

func (r *recordingBrokerClient) ArtifactRead(ctx context.Context, req brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, error) {
	r.record("ArtifactRead")
	return r.base.ArtifactRead(ctx, req)
}

func (r *recordingBrokerClient) LLMInvoke(ctx context.Context, req brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, error) {
	r.record("LLMInvoke")
	return r.base.LLMInvoke(ctx, req)
}

func (r *recordingBrokerClient) LLMStream(ctx context.Context, req brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, error) {
	r.record("LLMStream")
	return r.base.LLMStream(ctx, req)
}

func (r *recordingBrokerClient) AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error) {
	r.record("AuditTimeline")
	return r.base.AuditTimeline(ctx, limit, cursor)
}

func (r *recordingBrokerClient) AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error) {
	r.record("AuditVerificationGet")
	return r.base.AuditVerificationGet(ctx, viewLimit)
}

func (r *recordingBrokerClient) AuditFinalizeVerify(ctx context.Context) (brokerapi.AuditFinalizeVerifyResponse, error) {
	r.record("AuditFinalizeVerify")
	return r.base.AuditFinalizeVerify(ctx)
}

func (r *recordingBrokerClient) AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error) {
	r.record("AuditRecordGet")
	return r.base.AuditRecordGet(ctx, digest)
}

func (r *recordingBrokerClient) AuditAnchorPreflightGet(ctx context.Context, req brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, error) {
	r.record("AuditAnchorPreflightGet")
	return r.base.AuditAnchorPreflightGet(ctx, req)
}

func (r *recordingBrokerClient) AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, error) {
	r.record("AuditAnchorPresenceGet")
	return r.base.AuditAnchorPresenceGet(ctx, req)
}

func (r *recordingBrokerClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error) {
	r.record("AuditAnchorSegment")
	return r.base.AuditAnchorSegment(ctx, req)
}

func (r *recordingBrokerClient) GitSetupGet(ctx context.Context, provider string) (brokerapi.GitSetupGetResponse, error) {
	r.record("GitSetupGet")
	return r.base.GitSetupGet(ctx, provider)
}

func (r *recordingBrokerClient) GitSetupAuthBootstrap(ctx context.Context, req brokerapi.GitSetupAuthBootstrapRequest) (brokerapi.GitSetupAuthBootstrapResponse, error) {
	r.record("GitSetupAuthBootstrap")
	return r.base.GitSetupAuthBootstrap(ctx, req)
}

func (r *recordingBrokerClient) GitSetupIdentityUpsert(ctx context.Context, req brokerapi.GitSetupIdentityUpsertRequest) (brokerapi.GitSetupIdentityUpsertResponse, error) {
	r.record("GitSetupIdentityUpsert")
	return r.base.GitSetupIdentityUpsert(ctx, req)
}

func (r *recordingBrokerClient) ProviderSetupSessionBegin(ctx context.Context, req brokerapi.ProviderSetupSessionBeginRequest) (brokerapi.ProviderSetupSessionBeginResponse, error) {
	r.record("ProviderSetupSessionBegin")
	return r.base.ProviderSetupSessionBegin(ctx, req)
}

func (r *recordingBrokerClient) ProviderSetupSecretIngressPrepare(ctx context.Context, req brokerapi.ProviderSetupSecretIngressPrepareRequest) (brokerapi.ProviderSetupSecretIngressPrepareResponse, error) {
	r.record("ProviderSetupSecretIngressPrepare")
	return r.base.ProviderSetupSecretIngressPrepare(ctx, req)
}

func (r *recordingBrokerClient) ProviderSetupSecretIngressSubmit(ctx context.Context, req brokerapi.ProviderSetupSecretIngressSubmitRequest, secret []byte) (brokerapi.ProviderSetupSecretIngressSubmitResponse, error) {
	r.record("ProviderSetupSecretIngressSubmit")
	return r.base.ProviderSetupSecretIngressSubmit(ctx, req, secret)
}

func (r *recordingBrokerClient) ProviderCredentialLeaseIssue(ctx context.Context, req brokerapi.ProviderCredentialLeaseIssueRequest) (brokerapi.ProviderCredentialLeaseIssueResponse, error) {
	r.record("ProviderCredentialLeaseIssue")
	return r.base.ProviderCredentialLeaseIssue(ctx, req)
}

func (r *recordingBrokerClient) ProviderProfileList(ctx context.Context) (brokerapi.ProviderProfileListResponse, error) {
	r.record("ProviderProfileList")
	return r.base.ProviderProfileList(ctx)
}

func (r *recordingBrokerClient) ProviderProfileGet(ctx context.Context, providerProfileID string) (brokerapi.ProviderProfileGetResponse, error) {
	r.record("ProviderProfileGet")
	return r.base.ProviderProfileGet(ctx, providerProfileID)
}

func (r *recordingBrokerClient) ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error) {
	r.record("ReadinessGet")
	return r.base.ReadinessGet(ctx)
}

func (r *recordingBrokerClient) VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error) {
	r.record("VersionInfoGet")
	return r.base.VersionInfoGet(ctx)
}

func (f *fakeBrokerClient) RunList(ctx context.Context, limit int) (brokerapi.RunListResponse, error) {
	_ = ctx
	_ = limit
	return brokerapi.RunListResponse{Runs: []brokerapi.RunSummary{{RunID: "run-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", PendingApprovalCount: 1, ProvisioningPosture: "ok", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "degraded"}}}, nil
}

func (f *fakeBrokerClient) RunGet(ctx context.Context, runID string) (brokerapi.RunGetResponse, error) {
	_ = ctx
	if runID == "" {
		return brokerapi.RunGetResponse{}, fmt.Errorf("run id required")
	}
	return brokerapi.RunGetResponse{Run: brokerapi.RunDetail{Summary: brokerapi.RunSummary{RunID: runID, BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", ProvisioningPosture: "ok", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "degraded"}, Coordination: brokerapi.RunCoordinationSummary{Blocked: true, WaitReasonCode: "approval_wait", CoordinationMode: "stage_gate"}, StageSummaries: []brokerapi.RunStageSummary{{StageID: "stage-1", PendingApprovalCount: 1}, {StageID: "stage-2", PendingApprovalCount: 0}}, RoleSummaries: []brokerapi.RunRoleSummary{{RoleInstanceID: "role-1", WaitReasonCode: "approval_wait"}, {RoleInstanceID: "role-2"}}, PendingApprovalIDs: []string{"ap-1"}, ActiveManifestHashes: []string{"sha256:manifest"}, LatestPolicyDecisionRefs: []string{"sha256:policy"}, AuthoritativeState: map[string]any{"phase": "active"}, AdvisoryState: map[string]any{"runner": "active"}}}, nil
}

func (f *fakeBrokerClient) RunWatch(ctx context.Context, req brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, error) {
	_ = ctx
	if req.StreamID == "" {
		return nil, fmt.Errorf("stream id required")
	}
	run := brokerapi.RunSummary{RunID: "run-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", PendingApprovalCount: 1}
	return []brokerapi.RunWatchEvent{
		{EventType: "run_watch_snapshot", Seq: 1, Run: &run},
		{EventType: "run_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
	}, nil
}

func (f *fakeBrokerClient) SessionList(ctx context.Context, limit int) (brokerapi.SessionListResponse, error) {
	_ = ctx
	_ = limit
	return brokerapi.SessionListResponse{Sessions: []brokerapi.SessionSummary{{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", TurnCount: 2}, {Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}, Status: "active", TurnCount: 3}}}, nil
}

func (f *fakeBrokerClient) SessionGet(ctx context.Context, sessionID string) (brokerapi.SessionGetResponse, error) {
	_ = ctx
	if sessionID == "" {
		return brokerapi.SessionGetResponse{}, fmt.Errorf("session id required")
	}
	turnTwo := brokerapi.SessionTranscriptTurn{TurnID: "turn-2", TurnIndex: 2, Status: "completed", Messages: []brokerapi.SessionTranscriptMessage{{MessageID: "msg-2", MessageIndex: 2, Role: "assistant", ContentText: "world"}}}
	turnOne := brokerapi.SessionTranscriptTurn{TurnID: "turn-1", TurnIndex: 1, Status: "completed", Messages: []brokerapi.SessionTranscriptMessage{{MessageID: "msg-1", MessageIndex: 1, Role: "user", ContentText: "hello", RelatedLinks: brokerapi.SessionTranscriptLinks{RunIDs: []string{"run-1"}}}}}
	if strings.Contains(sessionID, "2") {
		turnTwo.Messages[0].RelatedLinks = brokerapi.SessionTranscriptLinks{ApprovalIDs: []string{"ap-1"}, ArtifactDigests: []string{"sha256:bbbb"}, AuditRecordDigests: []string{"sha256:aaaa"}}
	}
	return brokerapi.SessionGetResponse{Session: brokerapi.SessionDetail{Summary: brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: sessionID, WorkspaceID: "ws-1"}}, TranscriptTurns: []brokerapi.SessionTranscriptTurn{turnTwo, turnOne}, LinkedRunIDs: []string{"run-1"}, LinkedApprovalIDs: []string{"ap-1"}, LinkedArtifactDigests: []string{"sha256:bbbb"}, LinkedAuditRecordDigests: []string{"sha256:aaaa"}}}, nil
}

func (f *fakeBrokerClient) SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, error) {
	_ = ctx
	if req.SessionID == "" {
		return brokerapi.SessionSendMessageResponse{}, fmt.Errorf("session id required")
	}
	if strings.TrimSpace(req.ContentText) == "" {
		return brokerapi.SessionSendMessageResponse{}, fmt.Errorf("content required")
	}
	msg := brokerapi.SessionTranscriptMessage{MessageID: "msg-ack-1", TurnID: "turn-ack-1", SessionID: req.SessionID, MessageIndex: 1, Role: req.Role, ContentText: req.ContentText}
	turn := brokerapi.SessionTranscriptTurn{TurnID: "turn-ack-1", SessionID: req.SessionID, TurnIndex: 99, Status: "in_progress", Messages: []brokerapi.SessionTranscriptMessage{msg}}
	return brokerapi.SessionSendMessageResponse{SessionID: req.SessionID, Turn: turn, Message: msg, EventType: "session.message.appended", StreamID: "session", Seq: 1}, nil
}

func (f *fakeBrokerClient) SessionWatch(ctx context.Context, req brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, error) {
	_ = ctx
	if req.StreamID == "" {
		return nil, fmt.Errorf("stream id required")
	}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", TurnCount: 2}
	return []brokerapi.SessionWatchEvent{
		{EventType: "session_watch_snapshot", Seq: 1, Session: &session},
		{EventType: "session_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
	}, nil
}

func (f *fakeBrokerClient) ApprovalList(ctx context.Context, limit int) (brokerapi.ApprovalListResponse, error) {
	_ = ctx
	_ = limit
	return brokerapi.ApprovalListResponse{Approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{ActionKind: "promotion"}}}}, nil
}

func (f *fakeBrokerClient) ApprovalGet(ctx context.Context, approvalID string) (brokerapi.ApprovalGetResponse, error) {
	_ = ctx
	if approvalID == "" {
		return brokerapi.ApprovalGetResponse{}, fmt.Errorf("approval id required")
	}
	return brokerapi.ApprovalGetResponse{Approval: brokerapi.ApprovalSummary{ApprovalID: approvalID, Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{WorkspaceID: "ws-1", RunID: "run-1", StageID: "stage-1", ActionKind: "promotion"}}, ApprovalDetail: brokerapi.ApprovalDetail{BindingKind: "exact_action", PolicyReasonCode: "requires_human_review", LifecycleDetail: brokerapi.ApprovalLifecycleDetail{LifecycleState: "pending", LifecycleReasonCode: "awaiting_decision", Stale: true, StaleReasonCode: "policy_recomputed"}, WhatChangesIfApproved: brokerapi.ApprovalWhatChangesIfApproved{Summary: "Promotion continues", EffectKind: "unblock_next_stage"}, BlockedWorkScope: brokerapi.ApprovalBlockedWorkScope{ScopeKind: "stage", RunID: "run-1", StageID: "stage-1", ActionKind: "promotion"}, BoundIdentity: brokerapi.ApprovalBoundIdentity{ApprovalRequestDigest: "sha256:req", ManifestHash: "sha256:manifest", PolicyDecisionHash: "sha256:policy"}}, SignedApprovalRequest: testSignedApprovalRequest, SignedApprovalDecision: testSignedApprovalDecision}, nil
}

func (f *fakeBrokerClient) ApprovalResolve(ctx context.Context, req brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, error) {
	_ = ctx
	if req.ApprovalID == "" {
		return brokerapi.ApprovalResolveResponse{}, fmt.Errorf("approval id required")
	}
	return brokerapi.ApprovalResolveResponse{
		Approval:             brokerapi.ApprovalSummary{ApprovalID: req.ApprovalID, Status: "consumed", ApprovalTriggerCode: "policy_gate", BoundScope: req.BoundScope},
		ResolutionStatus:     "resolved",
		ResolutionReasonCode: "approval_consumed",
	}, nil
}

func (f *backendResolveReadyBrokerClient) ApprovalList(ctx context.Context, limit int) (brokerapi.ApprovalListResponse, error) {
	_ = ctx
	_ = limit
	return brokerapi.ApprovalListResponse{Approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{ActionKind: "promotion"}}, {ApprovalID: "ap-2", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{WorkspaceID: "ws-1", RunID: "run-1", ActionKind: "backend_posture_change"}}}}, nil
}

func (f *backendResolveReadyBrokerClient) ApprovalGet(ctx context.Context, approvalID string) (brokerapi.ApprovalGetResponse, error) {
	if approvalID == "ap-2" {
		return brokerapi.ApprovalGetResponse{Approval: brokerapi.ApprovalSummary{ApprovalID: approvalID, Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: "ws-1", RunID: "run-1", InstanceID: "launcher-instance-1", ActionKind: "backend_posture_change", PolicyDecisionHash: "sha256:policy"}}, ApprovalDetail: brokerapi.ApprovalDetail{BindingKind: "exact_action", PolicyReasonCode: "requires_human_review", LifecycleDetail: brokerapi.ApprovalLifecycleDetail{LifecycleState: "pending", LifecycleReasonCode: "awaiting_decision"}, WhatChangesIfApproved: brokerapi.ApprovalWhatChangesIfApproved{Summary: "Backend posture changes", EffectKind: "backend_posture_selection"}, BlockedWorkScope: brokerapi.ApprovalBlockedWorkScope{ScopeKind: "action_kind", ActionKind: "backend_posture_change"}, BoundIdentity: brokerapi.ApprovalBoundIdentity{ApprovalRequestDigest: "sha256:req", ManifestHash: "sha256:manifest", PolicyDecisionHash: "sha256:policy"}, BackendPostureSelection: &brokerapi.ApprovalBackendPostureSelection{TargetInstanceID: "launcher-instance-1", TargetBackendKind: "container", SelectionMode: "explicit_selection", ChangeKind: "select_backend", AssuranceChangeKind: "reduce_assurance", OptInKind: "exact_action_approval", ReducedAssuranceAcknowledged: true}}, SignedApprovalRequest: testSignedApprovalRequest, SignedApprovalDecision: testSignedApprovalDecision}, nil
	}
	return f.fakeBrokerClient.ApprovalGet(ctx, approvalID)
}

func (f *fakeBrokerClient) BackendPostureGet(ctx context.Context) (brokerapi.BackendPostureGetResponse, error) {
	_ = ctx
	return brokerapi.BackendPostureGetResponse{Posture: brokerapi.BackendPostureState{InstanceID: "launcher-instance-1", BackendKind: "microvm", PreferredBackendKind: "microvm", Availability: []brokerapi.BackendPostureAvailability{{BackendKind: "microvm", Available: true}, {BackendKind: "container", Available: true}}}}, nil
}

func (f *fakeBrokerClient) BackendPostureChange(ctx context.Context, req brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.TargetInstanceID) == "" {
		return brokerapi.BackendPostureChangeResponse{}, fmt.Errorf("target instance id required")
	}
	if req.TargetBackendKind == "container" {
		return brokerapi.BackendPostureChangeResponse{Outcome: brokerapi.BackendPostureChangeOutcome{Outcome: "approval_required", OutcomeReasonCode: "approval_required", ApprovalID: "sha256:" + strings.Repeat("a", 64)}, Posture: brokerapi.BackendPostureState{InstanceID: req.TargetInstanceID, BackendKind: "microvm", PreferredBackendKind: "microvm", PendingApproval: true, PendingApprovalID: "sha256:" + strings.Repeat("a", 64), Availability: []brokerapi.BackendPostureAvailability{{BackendKind: "microvm", Available: true}, {BackendKind: "container", Available: true}}}}, nil
	}
	return brokerapi.BackendPostureChangeResponse{Outcome: brokerapi.BackendPostureChangeOutcome{Outcome: "applied", OutcomeReasonCode: "policy_allow"}, Posture: brokerapi.BackendPostureState{InstanceID: req.TargetInstanceID, BackendKind: req.TargetBackendKind, PreferredBackendKind: "microvm", Availability: []brokerapi.BackendPostureAvailability{{BackendKind: "microvm", Available: true}, {BackendKind: "container", Available: true}}}}, nil
}

func (f *reloadAwareBrokerClient) RunList(ctx context.Context, limit int) (brokerapi.RunListResponse, error) {
	_ = ctx
	_ = limit
	return brokerapi.RunListResponse{Runs: []brokerapi.RunSummary{{RunID: "run-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", PendingApprovalCount: 1, ProvisioningPosture: "ok", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "degraded"}, {RunID: "run-2", LifecycleState: "blocked", BackendKind: "container", IsolationAssuranceLevel: "reduced", PendingApprovalCount: 0, ProvisioningPosture: "degraded", AuditIntegrityStatus: "degraded", AuditAnchoringStatus: "degraded"}}}, nil
}

func (f *reloadAwareBrokerClient) RunGet(ctx context.Context, runID string) (brokerapi.RunGetResponse, error) {
	_ = ctx
	summary := brokerapi.RunSummary{RunID: runID, BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", ProvisioningPosture: "ok", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "degraded"}
	coordination := brokerapi.RunCoordinationSummary{Blocked: true, WaitReasonCode: "approval_wait", CoordinationMode: "stage_gate"}
	if runID == "run-2" {
		summary = brokerapi.RunSummary{RunID: runID, BackendKind: "container", IsolationAssuranceLevel: "reduced", ProvisioningPosture: "degraded", AuditIntegrityStatus: "degraded", AuditAnchoringStatus: "degraded"}
		coordination = brokerapi.RunCoordinationSummary{Blocked: false, WaitReasonCode: "", CoordinationMode: "free"}
	}
	return brokerapi.RunGetResponse{Run: brokerapi.RunDetail{Summary: summary, Coordination: coordination}}, nil
}

func (f *reloadAwareBrokerClient) RunWatch(ctx context.Context, req brokerapi.RunWatchRequest) ([]brokerapi.RunWatchEvent, error) {
	return (&fakeBrokerClient{}).RunWatch(ctx, req)
}

func (f *reloadAwareBrokerClient) SessionList(ctx context.Context, limit int) (brokerapi.SessionListResponse, error) {
	return (&fakeBrokerClient{}).SessionList(ctx, limit)
}

func (f *reloadAwareBrokerClient) SessionGet(ctx context.Context, sessionID string) (brokerapi.SessionGetResponse, error) {
	return (&fakeBrokerClient{}).SessionGet(ctx, sessionID)
}

func (f *reloadAwareBrokerClient) SessionSendMessage(ctx context.Context, req brokerapi.SessionSendMessageRequest) (brokerapi.SessionSendMessageResponse, error) {
	return (&fakeBrokerClient{}).SessionSendMessage(ctx, req)
}

func (f *reloadAwareBrokerClient) SessionWatch(ctx context.Context, req brokerapi.SessionWatchRequest) ([]brokerapi.SessionWatchEvent, error) {
	return (&fakeBrokerClient{}).SessionWatch(ctx, req)
}

func (f *reloadAwareBrokerClient) ApprovalList(ctx context.Context, limit int) (brokerapi.ApprovalListResponse, error) {
	_ = ctx
	_ = limit
	return brokerapi.ApprovalListResponse{Approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{WorkspaceID: "ws-1", RunID: "run-1", StageID: "stage-1", ActionKind: "promotion"}}, {ApprovalID: "ap-2", Status: "pending", ApprovalTriggerCode: "stage_sign_off", BoundScope: brokerapi.ApprovalBoundScope{WorkspaceID: "ws-1", RunID: "run-2", StageID: "stage-2", ActionKind: "stage_summary_sign_off"}}}}, nil
}

func (f *reloadAwareBrokerClient) ApprovalGet(ctx context.Context, approvalID string) (brokerapi.ApprovalGetResponse, error) {
	_ = ctx
	resp, err := (&fakeBrokerClient{}).ApprovalGet(ctx, approvalID)
	if err != nil {
		return brokerapi.ApprovalGetResponse{}, err
	}
	if approvalID == "ap-2" {
		resp.Approval.BoundScope = brokerapi.ApprovalBoundScope{WorkspaceID: "ws-1", RunID: "run-2", StageID: "stage-2", ActionKind: "stage_summary_sign_off"}
		resp.ApprovalDetail.BindingKind = "stage_sign_off"
		resp.ApprovalDetail.PolicyReasonCode = "stage_sign_off_required"
		resp.ApprovalDetail.BoundStageSummaryHash = "sha256:stage"
		resp.ApprovalDetail.BoundIdentity.PolicyDecisionHash = "sha256:policy-2"
	}
	return resp, nil
}

func (f *reloadAwareBrokerClient) ApprovalResolve(ctx context.Context, req brokerapi.ApprovalResolveRequest) (brokerapi.ApprovalResolveResponse, error) {
	return (&fakeBrokerClient{}).ApprovalResolve(ctx, req)
}

func (f *reloadAwareBrokerClient) BackendPostureGet(ctx context.Context) (brokerapi.BackendPostureGetResponse, error) {
	return (&fakeBrokerClient{}).BackendPostureGet(ctx)
}

func (f *reloadAwareBrokerClient) BackendPostureChange(ctx context.Context, req brokerapi.BackendPostureChangeRequest) (brokerapi.BackendPostureChangeResponse, error) {
	return (&fakeBrokerClient{}).BackendPostureChange(ctx, req)
}

func (f *reloadAwareBrokerClient) ApprovalWatch(ctx context.Context, req brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, error) {
	return (&fakeBrokerClient{}).ApprovalWatch(ctx, req)
}

func (f *reloadAwareBrokerClient) ArtifactList(ctx context.Context, limit int, dataClass string) (brokerapi.LocalArtifactListResponse, error) {
	_ = ctx
	_ = limit
	_ = dataClass
	refOne := brokerapi.ArtifactSummary{}.Reference
	refOne.Digest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	refOne.ContentType = "text/plain"
	refOne.DataClass = "diffs"
	refOne.SizeBytes = 128
	refOne.ProvenanceReceiptHash = "sha256:receipt-1"
	refTwo := brokerapi.ArtifactSummary{}.Reference
	refTwo.Digest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	refTwo.ContentType = "text/plain"
	refTwo.DataClass = "build_logs"
	refTwo.SizeBytes = 256
	refTwo.ProvenanceReceiptHash = "sha256:receipt-2"
	return brokerapi.LocalArtifactListResponse{Artifacts: []brokerapi.ArtifactSummary{{Reference: refOne, RunID: "run-1"}, {Reference: refTwo, RunID: "run-2"}}}, nil
}

func (f *reloadAwareBrokerClient) ArtifactHead(ctx context.Context, digest string) (brokerapi.LocalArtifactHeadResponse, error) {
	_ = ctx
	ref := brokerapi.ArtifactSummary{}.Reference
	ref.Digest = digest
	ref.ContentType = "text/plain"
	ref.DataClass = "diffs"
	ref.SizeBytes = 128
	ref.ProvenanceReceiptHash = "sha256:receipt-1"
	if strings.Contains(digest, "cccc") {
		ref.DataClass = "build_logs"
		ref.SizeBytes = 256
		ref.ProvenanceReceiptHash = "sha256:receipt-2"
	}
	return brokerapi.LocalArtifactHeadResponse{Artifact: brokerapi.ArtifactSummary{Reference: ref}}, nil
}

func (f *reloadAwareBrokerClient) ArtifactRead(ctx context.Context, req brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, error) {
	_ = ctx
	content := "diff preview"
	if strings.Contains(req.Digest, "cccc") {
		content = "build log preview"
	}
	chunk := base64.StdEncoding.EncodeToString([]byte(content))
	return []brokerapi.ArtifactStreamEvent{{EventType: "artifact_stream_chunk", ChunkBase64: chunk}, {EventType: "artifact_stream_terminal", Terminal: true, TerminalStatus: "completed"}}, nil
}

func (f *reloadAwareBrokerClient) LLMInvoke(ctx context.Context, req brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, error) {
	return (&fakeBrokerClient{}).LLMInvoke(ctx, req)
}

func (f *reloadAwareBrokerClient) LLMStream(ctx context.Context, req brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, error) {
	return (&fakeBrokerClient{}).LLMStream(ctx, req)
}

func (f *reloadAwareBrokerClient) AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error) {
	return (&fakeBrokerClient{}).AuditTimeline(ctx, limit, cursor)
}

func (f *reloadAwareBrokerClient) AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error) {
	return (&fakeBrokerClient{}).AuditVerificationGet(ctx, viewLimit)
}

func (f *reloadAwareBrokerClient) AuditFinalizeVerify(ctx context.Context) (brokerapi.AuditFinalizeVerifyResponse, error) {
	return (&fakeBrokerClient{}).AuditFinalizeVerify(ctx)
}

func (f *reloadAwareBrokerClient) AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error) {
	return (&fakeBrokerClient{}).AuditRecordGet(ctx, digest)
}

func (f *reloadAwareBrokerClient) AuditAnchorPreflightGet(ctx context.Context, req brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, error) {
	return (&fakeBrokerClient{}).AuditAnchorPreflightGet(ctx, req)
}

func (f *reloadAwareBrokerClient) AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, error) {
	return (&fakeBrokerClient{}).AuditAnchorPresenceGet(ctx, req)
}

func (f *reloadAwareBrokerClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error) {
	return (&fakeBrokerClient{}).AuditAnchorSegment(ctx, req)
}

func (f *reloadAwareBrokerClient) GitSetupGet(ctx context.Context, provider string) (brokerapi.GitSetupGetResponse, error) {
	return (&fakeBrokerClient{}).GitSetupGet(ctx, provider)
}

func (f *reloadAwareBrokerClient) ProviderSetupSessionBegin(ctx context.Context, req brokerapi.ProviderSetupSessionBeginRequest) (brokerapi.ProviderSetupSessionBeginResponse, error) {
	return (&fakeBrokerClient{}).ProviderSetupSessionBegin(ctx, req)
}

func (f *reloadAwareBrokerClient) ProviderSetupSecretIngressPrepare(ctx context.Context, req brokerapi.ProviderSetupSecretIngressPrepareRequest) (brokerapi.ProviderSetupSecretIngressPrepareResponse, error) {
	return (&fakeBrokerClient{}).ProviderSetupSecretIngressPrepare(ctx, req)
}

func (f *reloadAwareBrokerClient) ProviderSetupSecretIngressSubmit(ctx context.Context, req brokerapi.ProviderSetupSecretIngressSubmitRequest, secret []byte) (brokerapi.ProviderSetupSecretIngressSubmitResponse, error) {
	return (&fakeBrokerClient{}).ProviderSetupSecretIngressSubmit(ctx, req, secret)
}

func (f *reloadAwareBrokerClient) ProviderCredentialLeaseIssue(ctx context.Context, req brokerapi.ProviderCredentialLeaseIssueRequest) (brokerapi.ProviderCredentialLeaseIssueResponse, error) {
	return (&fakeBrokerClient{}).ProviderCredentialLeaseIssue(ctx, req)
}

func (f *reloadAwareBrokerClient) ProviderProfileList(ctx context.Context) (brokerapi.ProviderProfileListResponse, error) {
	return (&fakeBrokerClient{}).ProviderProfileList(ctx)
}

func (f *reloadAwareBrokerClient) ProviderProfileGet(ctx context.Context, providerProfileID string) (brokerapi.ProviderProfileGetResponse, error) {
	return (&fakeBrokerClient{}).ProviderProfileGet(ctx, providerProfileID)
}

func (f *reloadAwareBrokerClient) GitSetupAuthBootstrap(ctx context.Context, req brokerapi.GitSetupAuthBootstrapRequest) (brokerapi.GitSetupAuthBootstrapResponse, error) {
	return (&fakeBrokerClient{}).GitSetupAuthBootstrap(ctx, req)
}

func (f *reloadAwareBrokerClient) GitSetupIdentityUpsert(ctx context.Context, req brokerapi.GitSetupIdentityUpsertRequest) (brokerapi.GitSetupIdentityUpsertResponse, error) {
	return (&fakeBrokerClient{}).GitSetupIdentityUpsert(ctx, req)
}

func (f *reloadAwareBrokerClient) ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error) {
	return (&fakeBrokerClient{}).ReadinessGet(ctx)
}

func (f *reloadAwareBrokerClient) VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error) {
	return (&fakeBrokerClient{}).VersionInfoGet(ctx)
}

func (f *fakeBrokerClient) ApprovalWatch(ctx context.Context, req brokerapi.ApprovalWatchRequest) ([]brokerapi.ApprovalWatchEvent, error) {
	_ = ctx
	if req.StreamID == "" {
		return nil, fmt.Errorf("stream id required")
	}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate"}
	return []brokerapi.ApprovalWatchEvent{
		{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval},
		{EventType: "approval_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"},
	}, nil
}

func (f *fakeBrokerClient) ArtifactList(ctx context.Context, limit int, dataClass string) (brokerapi.LocalArtifactListResponse, error) {
	_ = ctx
	_ = limit
	_ = dataClass
	ref := brokerapi.ArtifactSummary{}.Reference
	ref.Digest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	ref.ContentType = "text/plain"
	ref.DataClass = "diffs"
	ref.SizeBytes = 128
	ref.ProvenanceReceiptHash = "sha256:receipt"
	return brokerapi.LocalArtifactListResponse{Artifacts: []brokerapi.ArtifactSummary{{Reference: ref, RunID: "run-1"}}}, nil
}

func (f *fakeBrokerClient) ArtifactHead(ctx context.Context, digest string) (brokerapi.LocalArtifactHeadResponse, error) {
	_ = ctx
	if digest == "" {
		return brokerapi.LocalArtifactHeadResponse{}, fmt.Errorf("digest required")
	}
	ref := brokerapi.ArtifactSummary{}.Reference
	ref.Digest = digest
	ref.ContentType = "text/plain"
	ref.DataClass = "diffs"
	ref.SizeBytes = 128
	ref.ProvenanceReceiptHash = "sha256:receipt"
	return brokerapi.LocalArtifactHeadResponse{Artifact: brokerapi.ArtifactSummary{Reference: ref}}, nil
}

func (f *fakeBrokerClient) ArtifactRead(ctx context.Context, req brokerapi.ArtifactReadRequest) ([]brokerapi.ArtifactStreamEvent, error) {
	_ = ctx
	if req.Digest == "" {
		return nil, fmt.Errorf("digest required")
	}
	content := "diff --git a/file b/file\n+new line\n-result: success\ntoken=super-secret-token\n"
	chunk := base64.StdEncoding.EncodeToString([]byte(content))
	return []brokerapi.ArtifactStreamEvent{
		{EventType: "artifact_stream_chunk", ChunkBase64: chunk},
		{EventType: "artifact_stream_terminal", Terminal: true, TerminalStatus: "completed"},
	}, nil
}

func (f *fakeBrokerClient) LLMInvoke(ctx context.Context, req brokerapi.LLMInvokeRequest) (brokerapi.LLMInvokeResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.RunID) == "" {
		return brokerapi.LLMInvokeResponse{}, fmt.Errorf("run id required")
	}
	if req.LLMRequest == nil {
		return brokerapi.LLMInvokeResponse{}, fmt.Errorf("llm_request required")
	}
	return brokerapi.LLMInvokeResponse{
		SchemaID:      "runecode.protocol.v0.LLMInvokeResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-llm-invoke",
		RunID:         req.RunID,
		RequestDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)},
		Response: map[string]any{
			"schema_id":      "runecode.protocol.v0.LLMResponse",
			"schema_version": "0.1.0",
			"response_id":    "resp-1",
			"status":         "completed",
			"output_text":    "stubbed response",
		},
	}, nil
}

func (f *fakeBrokerClient) LLMStream(ctx context.Context, req brokerapi.LLMStreamRequest) (brokerapi.LLMStreamEnvelope, error) {
	_ = ctx
	if strings.TrimSpace(req.RunID) == "" {
		return brokerapi.LLMStreamEnvelope{}, fmt.Errorf("run id required")
	}
	if req.LLMRequest == nil {
		return brokerapi.LLMStreamEnvelope{}, fmt.Errorf("llm_request required")
	}
	return brokerapi.LLMStreamEnvelope{
		SchemaID:      "runecode.protocol.v0.LLMStreamEnvelope",
		SchemaVersion: "0.1.0",
		RequestID:     "req-llm-stream",
		RunID:         req.RunID,
		RequestDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)},
		Events: []brokerapi.LLMStreamAny{
			map[string]any{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.1.0", "event_type": "response.delta", "seq": 1, "delta_text": "stub"},
			map[string]any{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.1.0", "event_type": "response.completed", "seq": 2},
		},
	}, nil
}

func (f *fakeBrokerClient) AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error) {
	_ = ctx
	_ = limit
	if cursor == "page-2" {
		entry := brokerapi.AuditTimelineViewEntry{RecordDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, EventType: "approval", Summary: "Approval consumed", VerificationPosture: &brokerapi.AuditRecordVerificationPosture{Status: "failed", ReasonCodes: []string{"anchor_receipt_invalid"}}}
		return brokerapi.AuditTimelineResponse{Views: []brokerapi.AuditTimelineViewEntry{entry}}, nil
	}
	entry := brokerapi.AuditTimelineViewEntry{RecordDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, EventType: "run_state", Summary: "Run state changed", VerificationPosture: &brokerapi.AuditRecordVerificationPosture{Status: "degraded", ReasonCodes: []string{"anchor_receipt_missing"}}}
	return brokerapi.AuditTimelineResponse{Views: []brokerapi.AuditTimelineViewEntry{entry}, NextCursor: "page-2"}, nil
}

func (f *fakeBrokerClient) AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error) {
	_ = ctx
	_ = viewLimit
	report := trustpolicy.AuditVerificationReportPayload{
		AnchoringStatus: "degraded",
		DegradedReasons: []string{"anchor_receipt_missing"},
		HardFailures:    []string{"anchor_receipt_invalid"},
		Findings: []trustpolicy.AuditVerificationFinding{
			{Code: "anchor_receipt_missing", Dimension: trustpolicy.AuditVerificationDimensionAnchoring, Severity: trustpolicy.AuditVerificationSeverityWarning, Message: "anchor receipt pending"},
			{Code: "anchor_receipt_invalid", Dimension: trustpolicy.AuditVerificationDimensionAnchoring, Severity: trustpolicy.AuditVerificationSeverityError, Message: "anchor receipt invalid"},
		},
	}
	return brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "degraded", CurrentlyDegraded: true, FindingCount: 2}, Report: report}, nil
}

func (f *fakeBrokerClient) AuditFinalizeVerify(ctx context.Context) (brokerapi.AuditFinalizeVerifyResponse, error) {
	_ = ctx
	report := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("e", 64)}
	return brokerapi.AuditFinalizeVerifyResponse{
		SchemaID:      "runecode.protocol.v0.AuditFinalizeVerifyResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-finalize",
		ActionStatus:  "ok",
		SegmentID:     "segment-000001",
		ReportDigest:  &report,
	}, nil
}

func (f *fakeBrokerClient) AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error) {
	_ = ctx
	if digest == "" {
		return brokerapi.AuditRecordGetResponse{}, fmt.Errorf("digest required")
	}
	return brokerapi.AuditRecordGetResponse{Record: brokerapi.AuditRecordDetail{RecordFamily: "audit_event", EventType: "run_state", OccurredAt: "2026-01-01T00:00:00Z", LinkedReferences: []brokerapi.AuditRecordLinkedReference{{ReferenceKind: "run", ReferenceID: "run-1"}}, VerificationPosture: &brokerapi.AuditRecordVerificationPosture{Status: "degraded", ReasonCodes: []string{"anchor_delayed"}}}}, nil
}

func (f *fakeBrokerClient) AuditAnchorPresenceGet(ctx context.Context, req brokerapi.AuditAnchorPresenceGetRequest) (brokerapi.AuditAnchorPresenceGetResponse, error) {
	_ = ctx
	if _, err := req.SealDigest.Identity(); err != nil {
		return brokerapi.AuditAnchorPresenceGetResponse{}, fmt.Errorf("invalid seal digest")
	}
	return brokerapi.AuditAnchorPresenceGetResponse{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPresenceGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-presence",
		SealDigest:    req.SealDigest,
		PresenceMode:  "os_confirmation",
		PresenceAttestation: &brokerapi.AuditAnchorPresenceAttestation{
			Challenge:           "presence-challenge-0123456789abcdef",
			AcknowledgmentToken: strings.Repeat("a", 64),
		},
	}, nil
}

func (f *fakeBrokerClient) AuditAnchorPreflightGet(ctx context.Context, req brokerapi.AuditAnchorPreflightGetRequest) (brokerapi.AuditAnchorPreflightGetResponse, error) {
	_ = ctx
	_ = req
	seal := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("e", 64)}
	return brokerapi.AuditAnchorPreflightGetResponse{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPreflightGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-preflight",
		LatestAnchorableSeal: &brokerapi.AuditAnchorableSealRef{
			SegmentID:  "segment-000001",
			SealDigest: seal,
		},
		SignerReadiness:      brokerapi.AuditAnchorSignerReadiness{Ready: true, PresenceMode: "os_confirmation", SignerLogicalScope: "node"},
		VerifierReadiness:    brokerapi.AuditAnchorVerifierReadiness{Ready: true},
		PresenceRequirements: brokerapi.AuditAnchorPresenceRequirements{Required: true, AttestationMode: "os_confirmation", AttestationReady: true},
		ApprovalRequirements: brokerapi.AuditAnchorApprovalRequirements{Required: false, ReasonCode: "approval_not_required", Message: "no approval requirement declared"},
	}, nil
}

func (f *fakeBrokerClient) AuditAnchorSegment(ctx context.Context, req brokerapi.AuditAnchorSegmentRequest) (brokerapi.AuditAnchorSegmentResponse, error) {
	_ = ctx
	if _, err := req.SealDigest.Identity(); err != nil {
		return brokerapi.AuditAnchorSegmentResponse{}, fmt.Errorf("invalid seal digest")
	}
	receipt := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("c", 64)}
	report := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("d", 64)}
	return brokerapi.AuditAnchorSegmentResponse{
		SchemaID:                 "runecode.protocol.v0.AuditAnchorSegmentResponse",
		SchemaVersion:            "0.1.0",
		RequestID:                "req-anchor",
		SealDigest:               req.SealDigest,
		ReceiptDigest:            &receipt,
		VerificationReportDigest: &report,
		AnchoringStatus:          "ok",
	}, nil
}

func (f *fakeBrokerClient) GitSetupGet(ctx context.Context, provider string) (brokerapi.GitSetupGetResponse, error) {
	_ = ctx
	resolved := strings.TrimSpace(provider)
	if resolved == "" {
		resolved = "github"
	}
	profile := brokerapi.GitCommitIdentityProfile{SchemaID: "runecode.protocol.v0.GitCommitIdentityProfile", SchemaVersion: "0.1.0", ProfileID: "default", DisplayName: "Default identity", AuthorName: "RuneCode Operator", AuthorEmail: "operator@example.invalid", CommitterName: "RuneCode Operator", CommitterEmail: "operator@example.invalid", SignoffName: "RuneCode Operator", SignoffEmail: "operator@example.invalid", DefaultProfile: true}
	return brokerapi.GitSetupGetResponse{SchemaID: "runecode.protocol.v0.GitSetupGetResponse", SchemaVersion: "0.1.0", RequestID: "req-git-setup-get", ProviderAccount: brokerapi.GitProviderAccountState{SchemaID: "runecode.protocol.v0.GitProviderAccountState", SchemaVersion: "0.1.0", Provider: resolved, AccountID: "not_linked", AccountUsername: "not_linked", Linked: false, Source: "restored_state"}, IdentityProfiles: []brokerapi.GitCommitIdentityProfile{profile}, AuthPosture: brokerapi.GitAuthPostureState{SchemaID: "runecode.protocol.v0.GitAuthPostureState", SchemaVersion: "0.1.0", Provider: resolved, AuthStatus: "not_linked", BootstrapMode: "browser", HeadlessBootstrapSupported: true, InteractiveTokenFallbackSupport: true}, ControlPlaneState: brokerapi.GitControlPlaneState{SchemaID: "runecode.protocol.v0.GitControlPlaneState", SchemaVersion: "0.1.0", Provider: resolved, DefaultIdentityProfileID: "default", LastSetupView: "overview", RecentRepositories: []string{}}, PolicySurface: brokerapi.GitPolicySurfaceState{ArtifactManagedOnly: true, InspectionSupported: true, PrepareChangesSupport: true, DirectMutationSupport: false}}, nil
}

func (f *fakeBrokerClient) ProviderSetupSessionBegin(ctx context.Context, req brokerapi.ProviderSetupSessionBeginRequest) (brokerapi.ProviderSetupSessionBeginResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.ProviderFamily) == "" {
		return brokerapi.ProviderSetupSessionBeginResponse{}, fmt.Errorf("provider family required")
	}
	profile := brokerapi.ProviderProfile{SchemaID: "runecode.protocol.v0.ProviderProfile", SchemaVersion: "0.1.0", ProviderProfileID: "provider-profile-test", DisplayLabel: "Test", ProviderFamily: req.ProviderFamily, AdapterKind: req.AdapterKind, CurrentAuthMode: "direct_credential", SupportedAuthModes: []string{"direct_credential"}, AuthMaterial: brokerapi.ProviderAuthMaterial{SchemaID: "runecode.protocol.v0.ProviderAuthMaterial", SchemaVersion: "0.1.0", MaterialKind: "direct_credential", MaterialState: "missing"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{SchemaID: "runecode.protocol.v0.ProviderReadinessPosture", SchemaVersion: "0.1.0", ConfigurationState: "configured", CredentialState: "missing", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"}}
	session := brokerapi.ProviderSetupSession{SchemaID: "runecode.protocol.v0.ProviderSetupSession", SchemaVersion: "0.1.0", SetupSessionID: "provider-setup-session-test", ProviderProfileID: profile.ProviderProfileID, ProviderFamily: req.ProviderFamily, CurrentPhase: "metadata_configured", CurrentAuthMode: "direct_credential", SecretIngressReady: false, CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}
	return brokerapi.ProviderSetupSessionBeginResponse{SchemaID: "runecode.protocol.v0.ProviderSetupSessionBeginResponse", SchemaVersion: "0.1.0", RequestID: "req-provider-setup-begin", SetupSession: session, Profile: profile}, nil
}

func (f *fakeBrokerClient) ProviderSetupSecretIngressPrepare(ctx context.Context, req brokerapi.ProviderSetupSecretIngressPrepareRequest) (brokerapi.ProviderSetupSecretIngressPrepareResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.SetupSessionID) == "" {
		return brokerapi.ProviderSetupSecretIngressPrepareResponse{}, fmt.Errorf("setup session id required")
	}
	session := brokerapi.ProviderSetupSession{SchemaID: "runecode.protocol.v0.ProviderSetupSession", SchemaVersion: "0.1.0", SetupSessionID: req.SetupSessionID, ProviderProfileID: "provider-profile-test", ProviderFamily: "openai_compatible", CurrentPhase: "secret_ingress_ready", CurrentAuthMode: "direct_credential", SecretIngressReady: true, CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}
	return brokerapi.ProviderSetupSecretIngressPrepareResponse{SchemaID: "runecode.protocol.v0.ProviderSetupSecretIngressPrepareResponse", SchemaVersion: "0.1.0", RequestID: "req-provider-setup-prepare", SetupSession: session, SecretIngressToken: "provider-secret-ingress-test", ExpiresAt: "2026-01-01T00:05:00Z"}, nil
}

func (f *fakeBrokerClient) ProviderSetupSecretIngressSubmit(ctx context.Context, req brokerapi.ProviderSetupSecretIngressSubmitRequest, secret []byte) (brokerapi.ProviderSetupSecretIngressSubmitResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.SecretIngressToken) == "" {
		return brokerapi.ProviderSetupSecretIngressSubmitResponse{}, fmt.Errorf("secret ingress token required")
	}
	if len(secret) == 0 {
		return brokerapi.ProviderSetupSecretIngressSubmitResponse{}, fmt.Errorf("secret required")
	}
	profile := brokerapi.ProviderProfile{SchemaID: "runecode.protocol.v0.ProviderProfile", SchemaVersion: "0.1.0", ProviderProfileID: "provider-profile-test", DisplayLabel: "Test", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", CurrentAuthMode: "direct_credential", SupportedAuthModes: []string{"direct_credential"}, AuthMaterial: brokerapi.ProviderAuthMaterial{SchemaID: "runecode.protocol.v0.ProviderAuthMaterial", SchemaVersion: "0.1.0", MaterialKind: "direct_credential", MaterialState: "present", SecretRef: "secrets/model-providers/provider-profile-test/direct-credential", LeasePolicyRef: "secretsd://lease-policy/model-provider-default"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{SchemaID: "runecode.protocol.v0.ProviderReadinessPosture", SchemaVersion: "0.1.0", ConfigurationState: "configured", CredentialState: "present", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"}}
	session := brokerapi.ProviderSetupSession{SchemaID: "runecode.protocol.v0.ProviderSetupSession", SchemaVersion: "0.1.0", SetupSessionID: "provider-setup-session-test", ProviderProfileID: profile.ProviderProfileID, ProviderFamily: profile.ProviderFamily, CurrentPhase: "configured", CurrentAuthMode: "direct_credential", SecretIngressReady: false, CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:01:00Z"}
	return brokerapi.ProviderSetupSecretIngressSubmitResponse{SchemaID: "runecode.protocol.v0.ProviderSetupSecretIngressSubmitResponse", SchemaVersion: "0.1.0", RequestID: "req-provider-setup-submit", SetupSession: session, Profile: profile}, nil
}

func (f *fakeBrokerClient) ProviderCredentialLeaseIssue(ctx context.Context, req brokerapi.ProviderCredentialLeaseIssueRequest) (brokerapi.ProviderCredentialLeaseIssueResponse, error) {
	_ = ctx
	if strings.TrimSpace(req.ProviderProfileID) == "" || strings.TrimSpace(req.RunID) == "" {
		return brokerapi.ProviderCredentialLeaseIssueResponse{}, fmt.Errorf("provider profile id and run id required")
	}
	lease := fakeProviderCredentialLease(req.RunID)
	return brokerapi.ProviderCredentialLeaseIssueResponse{SchemaID: "runecode.protocol.v0.ProviderCredentialLeaseIssueResponse", SchemaVersion: "0.1.0", RequestID: "req-provider-lease", ProviderProfileID: req.ProviderProfileID, ProviderAuthLeaseID: lease.LeaseID, Lease: lease}, nil
}

func (f *fakeBrokerClient) ProviderProfileList(ctx context.Context) (brokerapi.ProviderProfileListResponse, error) {
	_ = ctx
	profile := brokerapi.ProviderProfile{SchemaID: "runecode.protocol.v0.ProviderProfile", SchemaVersion: "0.1.0", ProviderProfileID: "provider-profile-test", DisplayLabel: "Test", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", DestinationRef: "model_endpoint://api.openai.com/v1", SupportedAuthModes: []string{"direct_credential"}, CurrentAuthMode: "direct_credential", AllowlistedModelIDs: []string{"gpt-4o-mini"}, ModelCatalogPosture: brokerapi.ProviderModelCatalogPosture{SchemaID: "runecode.protocol.v0.ProviderModelCatalogPosture", SchemaVersion: "0.1.0", SelectionAuthority: "manual_allowlist_canonical", DiscoveryPosture: "advisory", CompatibilityProbePosture: "advisory"}, CompatibilityPosture: "unverified", QuotaProfileKind: "hybrid", RequestBindingKind: "canonical_llm_request_digest", SurfaceChannel: "broker_local_api", AuthMaterial: brokerapi.ProviderAuthMaterial{SchemaID: "runecode.protocol.v0.ProviderAuthMaterial", SchemaVersion: "0.1.0", MaterialKind: "direct_credential", MaterialState: "present", SecretRef: "secrets/model-providers/provider-profile-test/direct-credential", LeasePolicyRef: "secretsd://lease-policy/model-provider-default"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{SchemaID: "runecode.protocol.v0.ProviderReadinessPosture", SchemaVersion: "0.1.0", ConfigurationState: "configured", CredentialState: "present", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"}, Lifecycle: brokerapi.ProviderLifecycleMetadata{CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}}
	return brokerapi.ProviderProfileListResponse{SchemaID: "runecode.protocol.v0.ProviderProfileListResponse", SchemaVersion: "0.1.0", RequestID: "req-provider-profile-list", Profiles: []brokerapi.ProviderProfile{profile}}, nil
}

func (f *fakeBrokerClient) ProviderProfileGet(ctx context.Context, providerProfileID string) (brokerapi.ProviderProfileGetResponse, error) {
	_ = ctx
	if strings.TrimSpace(providerProfileID) == "" {
		return brokerapi.ProviderProfileGetResponse{}, fmt.Errorf("provider profile id required")
	}
	list, _ := f.ProviderProfileList(ctx)
	profile := list.Profiles[0]
	profile.ProviderProfileID = strings.TrimSpace(providerProfileID)
	return brokerapi.ProviderProfileGetResponse{SchemaID: "runecode.protocol.v0.ProviderProfileGetResponse", SchemaVersion: "0.1.0", RequestID: "req-provider-profile-get", Profile: profile}, nil
}

func fakeProviderCredentialLease(runID string) secretsd.Lease {
	return secretsd.Lease{LeaseID: "lease-provider-credential", SecretRef: "secrets/model-providers/provider-profile-test/direct-credential", ConsumerID: "principal:gateway:model:" + runID, RoleKind: "model-gateway", Scope: "run:" + runID, DeliveryKind: "model_gateway", Status: "active"}
}

func (f *fakeBrokerClient) GitSetupAuthBootstrap(ctx context.Context, req brokerapi.GitSetupAuthBootstrapRequest) (brokerapi.GitSetupAuthBootstrapResponse, error) {
	_ = ctx
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = "github"
	}
	status := "pending"
	account := brokerapi.GitProviderAccountState{SchemaID: "runecode.protocol.v0.GitProviderAccountState", SchemaVersion: "0.1.0", Provider: provider, AccountID: "pending", AccountUsername: "pending", Linked: false, Source: "auth_bootstrap"}
	auth := brokerapi.GitAuthPostureState{SchemaID: "runecode.protocol.v0.GitAuthPostureState", SchemaVersion: "0.1.0", Provider: provider, AuthStatus: "not_linked", BootstrapMode: req.Mode, HeadlessBootstrapSupported: true, InteractiveTokenFallbackSupport: true}
	deviceURI := ""
	deviceCode := ""
	nextPoll := 0
	if req.Mode == "device_code" {
		deviceURI = "https://github.com/login/device"
		deviceCode = "RUNE-CODE"
		nextPoll = 5
	}
	return brokerapi.GitSetupAuthBootstrapResponse{SchemaID: "runecode.protocol.v0.GitSetupAuthBootstrapResponse", SchemaVersion: "0.1.0", RequestID: "req-git-auth-bootstrap", Provider: provider, Mode: req.Mode, Status: status, DeviceVerificationURI: deviceURI, DeviceUserCode: deviceCode, NextPollAfterSeconds: nextPoll, AccountState: account, AuthPosture: auth}, nil
}

func (f *fakeBrokerClient) GitSetupIdentityUpsert(ctx context.Context, req brokerapi.GitSetupIdentityUpsertRequest) (brokerapi.GitSetupIdentityUpsertResponse, error) {
	_ = ctx
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = "github"
	}
	profile := req.Profile
	profile.SchemaID = "runecode.protocol.v0.GitCommitIdentityProfile"
	profile.SchemaVersion = "0.1.0"
	if strings.TrimSpace(profile.ProfileID) == "" {
		return brokerapi.GitSetupIdentityUpsertResponse{}, fmt.Errorf("profile id required")
	}
	return brokerapi.GitSetupIdentityUpsertResponse{SchemaID: "runecode.protocol.v0.GitSetupIdentityUpsertResponse", SchemaVersion: "0.1.0", RequestID: "req-git-identity-upsert", Provider: provider, Profile: profile, ControlPlaneState: brokerapi.GitControlPlaneState{SchemaID: "runecode.protocol.v0.GitControlPlaneState", SchemaVersion: "0.1.0", Provider: provider, DefaultIdentityProfileID: profile.ProfileID, LastSetupView: "identity", RecentRepositories: []string{}}}, nil
}

func (f *fakeBrokerClient) ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error) {
	_ = ctx
	return brokerapi.ReadinessGetResponse{Readiness: brokerapi.BrokerReadiness{Ready: true, LocalOnly: true, RecoveryComplete: true, AppendPositionStable: true, CurrentSegmentWritable: true, VerifierMaterialAvailable: false, DerivedIndexCaughtUp: true}}, nil
}

func (f *fakeBrokerClient) VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error) {
	_ = ctx
	return brokerapi.VersionInfoGetResponse{VersionInfo: brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0", BuildRevision: "abc123", BuildTime: "2026-01-01T00:00:00Z", ProtocolBundleVersion: "0.9.0", ProtocolBundleManifestHash: "sha256:xyz", APIFamily: "broker_local_api", APIVersion: "v0"}}, nil
}
