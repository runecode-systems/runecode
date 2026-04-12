package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type fakeBrokerClient struct{}

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

func (r *recordingBrokerClient) AuditTimeline(ctx context.Context, limit int, cursor string) (brokerapi.AuditTimelineResponse, error) {
	r.record("AuditTimeline")
	return r.base.AuditTimeline(ctx, limit, cursor)
}

func (r *recordingBrokerClient) AuditVerificationGet(ctx context.Context, viewLimit int) (brokerapi.AuditVerificationGetResponse, error) {
	r.record("AuditVerificationGet")
	return r.base.AuditVerificationGet(ctx, viewLimit)
}

func (r *recordingBrokerClient) AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error) {
	r.record("AuditRecordGet")
	return r.base.AuditRecordGet(ctx, digest)
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
	return brokerapi.ApprovalGetResponse{Approval: brokerapi.ApprovalSummary{ApprovalID: approvalID, Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{WorkspaceID: "ws-1", RunID: "run-1", StageID: "stage-1", ActionKind: "promotion"}}, ApprovalDetail: brokerapi.ApprovalDetail{BindingKind: "exact_action", PolicyReasonCode: "requires_human_review", LifecycleDetail: brokerapi.ApprovalLifecycleDetail{LifecycleState: "pending", LifecycleReasonCode: "awaiting_decision", Stale: true, StaleReasonCode: "policy_recomputed"}, WhatChangesIfApproved: brokerapi.ApprovalWhatChangesIfApproved{Summary: "Promotion continues", EffectKind: "unblock_next_stage"}, BlockedWorkScope: brokerapi.ApprovalBlockedWorkScope{ScopeKind: "stage", RunID: "run-1", StageID: "stage-1", ActionKind: "promotion"}, BoundIdentity: brokerapi.ApprovalBoundIdentity{ApprovalRequestDigest: "sha256:req", ManifestHash: "sha256:manifest", PolicyDecisionHash: "sha256:policy"}}}, nil
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

func (f *fakeBrokerClient) AuditRecordGet(ctx context.Context, digest string) (brokerapi.AuditRecordGetResponse, error) {
	_ = ctx
	if digest == "" {
		return brokerapi.AuditRecordGetResponse{}, fmt.Errorf("digest required")
	}
	return brokerapi.AuditRecordGetResponse{Record: brokerapi.AuditRecordDetail{RecordFamily: "audit_event", EventType: "run_state", OccurredAt: "2026-01-01T00:00:00Z", LinkedReferences: []brokerapi.AuditRecordLinkedReference{{ReferenceKind: "run", ReferenceID: "run-1"}}, VerificationPosture: &brokerapi.AuditRecordVerificationPosture{Status: "degraded", ReasonCodes: []string{"anchor_delayed"}}}}, nil
}

func (f *fakeBrokerClient) ReadinessGet(ctx context.Context) (brokerapi.ReadinessGetResponse, error) {
	_ = ctx
	return brokerapi.ReadinessGetResponse{Readiness: brokerapi.BrokerReadiness{Ready: true, LocalOnly: true, RecoveryComplete: true, AppendPositionStable: true, CurrentSegmentWritable: true, VerifierMaterialAvailable: false, DerivedIndexCaughtUp: true}}, nil
}

func (f *fakeBrokerClient) VersionInfoGet(ctx context.Context) (brokerapi.VersionInfoGetResponse, error) {
	_ = ctx
	return brokerapi.VersionInfoGetResponse{VersionInfo: brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0", BuildRevision: "abc123", BuildTime: "2026-01-01T00:00:00Z", ProtocolBundleVersion: "0.9.0", ProtocolBundleManifestHash: "sha256:xyz", APIFamily: "broker_local_api", APIVersion: "v0"}}, nil
}
