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
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestCLIAdoptionRoutesRunApprovalVersionAndLogThroughLocalRPC(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	requestedOps := make([]string, 0, 8)

	installRunApprovalVersionLogDispatchStub(t, &requestedOps)

	llmRequestPath := writeLLMRequestFile(t)
	commands := [][]string{{"run-list"}, {"run-get", "--run-id", "run-1"}, {"run-watch"}, {"backend-posture-get"}, {"backend-posture-change", "--target-backend-kind", "container", "--reduced-assurance-acknowledged"}, {"session-list"}, {"session-get", "--session-id", "sess-1"}, {"session-send-message", "--session-id", "sess-1", "--content", "hello"}, {"session-watch"}, {"approval-list"}, {"approval-get", "--approval-id", testDigest("a")}, {"approval-watch"}, {"version-info"}, {"stream-logs"}, {"stream-logs", "--stream-id", "custom-stream"}, {"llm-invoke", "--run-id", "run-1", "--request-file", llmRequestPath}, {"llm-stream", "--run-id", "run-1", "--request-file", llmRequestPath, "--stream-id", "llm-s-1"}}
	for _, args := range commands {
		stdout.Reset()
		if err := run(args, stdout, stderr); err != nil {
			t.Fatalf("run(%v) error: %v", args, err)
		}
	}

	want := []string{"run_list", "run_get", "run_watch", "backend_posture_get", "backend_posture_get", "backend_posture_change", "session_list", "session_get", "session_send_message", "session_watch", "approval_list", "approval_get", "approval_watch", "version_info_get", "log_stream", "log_stream", "llm_invoke", "llm_stream"}
	assertRequestedOps(t, requestedOps, want)
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
		switch wire.Operation {
		case "run_list":
			return mustOKLocalRPCResponse(t, brokerapi.RunListResponse{SchemaID: "runecode.protocol.v0.RunListResponse", SchemaVersion: "0.1.0", RequestID: "req-run-list"})
		case "run_get":
			return mustOKLocalRPCResponse(t, brokerapi.RunGetResponse{SchemaID: "runecode.protocol.v0.RunGetResponse", SchemaVersion: "0.1.0", RequestID: "req-run-get", Run: brokerapi.RunDetail{SchemaID: "runecode.protocol.v0.RunDetail", SchemaVersion: "0.2.0", Summary: brokerapi.RunSummary{SchemaID: "runecode.protocol.v0.RunSummary", SchemaVersion: "0.2.0", RunID: "run-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z", LifecycleState: "pending", PendingApprovalCount: 0, ApprovalProfile: "unknown", BackendKind: "unknown", IsolationAssuranceLevel: "unknown", ProvisioningPosture: "unknown", AssuranceLevel: "unknown", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok", AuditCurrentlyDegraded: false, RuntimePostureDegraded: false}}})
		case "run_watch":
			return mustOKLocalRPCResponse(t, []brokerapi.RunWatchEvent{{SchemaID: "runecode.protocol.v0.RunWatchEvent", SchemaVersion: "0.1.0", StreamID: "rw-1", RequestID: "req-run-watch", Seq: 1, EventType: "run_watch_snapshot", Run: &brokerapi.RunSummary{SchemaID: "runecode.protocol.v0.RunSummary", SchemaVersion: "0.2.0", RunID: "run-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z", LifecycleState: "pending", PendingApprovalCount: 0, ApprovalProfile: "unknown", BackendKind: "unknown", IsolationAssuranceLevel: "unknown", ProvisioningPosture: "unknown", AssuranceLevel: "unknown", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok", AuditCurrentlyDegraded: false, RuntimePostureDegraded: false}}, {SchemaID: "runecode.protocol.v0.RunWatchEvent", SchemaVersion: "0.1.0", StreamID: "rw-1", RequestID: "req-run-watch", Seq: 2, EventType: "run_watch_terminal", Terminal: true, TerminalStatus: "completed"}})
		case "backend_posture_get":
			return mustOKLocalRPCResponse(t, brokerapi.BackendPostureGetResponse{SchemaID: "runecode.protocol.v0.BackendPostureGetResponse", SchemaVersion: "0.1.0", RequestID: "req-backend-posture-get", Posture: brokerapi.BackendPostureState{SchemaID: "runecode.protocol.v0.BackendPostureState", SchemaVersion: "0.1.0", InstanceID: "launcher-instance-1", BackendKind: "microvm", PreferredBackendKind: "microvm", Availability: []brokerapi.BackendPostureAvailability{{SchemaID: "runecode.protocol.v0.BackendPostureAvailability", SchemaVersion: "0.1.0", BackendKind: "microvm", Available: true}, {SchemaID: "runecode.protocol.v0.BackendPostureAvailability", SchemaVersion: "0.1.0", BackendKind: "container", Available: true}}}})
		case "backend_posture_change":
			return mustOKLocalRPCResponse(t, brokerapi.BackendPostureChangeResponse{SchemaID: "runecode.protocol.v0.BackendPostureChangeResponse", SchemaVersion: "0.1.0", RequestID: "req-backend-posture-change", Outcome: brokerapi.BackendPostureChangeOutcome{SchemaID: "runecode.protocol.v0.BackendPostureChangeOutcome", SchemaVersion: "0.1.0", Outcome: "approval_required", OutcomeReasonCode: "approval_required", ApprovalID: testDigest("a")}, Posture: brokerapi.BackendPostureState{SchemaID: "runecode.protocol.v0.BackendPostureState", SchemaVersion: "0.1.0", InstanceID: "launcher-instance-1", BackendKind: "microvm", PendingApproval: true, PendingApprovalID: testDigest("a")}})
		case "session_list":
			return mustOKLocalRPCResponse(t, brokerapi.SessionListResponse{SchemaID: "runecode.protocol.v0.SessionListResponse", SchemaVersion: "0.1.0", RequestID: "req-session-list", Order: "updated_at_desc", Sessions: []brokerapi.SessionSummary{{SchemaID: "runecode.protocol.v0.SessionSummary", SchemaVersion: "0.1.0", Identity: brokerapi.SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z"}, UpdatedAt: "2026-01-01T00:00:00Z", Status: "active", LastActivityKind: "run_progress", TurnCount: 0, LinkedRunCount: 1, LinkedApprovalCount: 0, LinkedArtifactCount: 0, LinkedAuditEventCount: 0, HasIncompleteTurn: false}}})
		case "session_get":
			return mustOKLocalRPCResponse(t, brokerapi.SessionGetResponse{SchemaID: "runecode.protocol.v0.SessionGetResponse", SchemaVersion: "0.1.0", RequestID: "req-session-get", Session: brokerapi.SessionDetail{SchemaID: "runecode.protocol.v0.SessionDetail", SchemaVersion: "0.1.0", Summary: brokerapi.SessionSummary{SchemaID: "runecode.protocol.v0.SessionSummary", SchemaVersion: "0.1.0", Identity: brokerapi.SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z"}, UpdatedAt: "2026-01-01T00:00:00Z", Status: "active", LastActivityKind: "run_progress", TurnCount: 0, LinkedRunCount: 1, LinkedApprovalCount: 0, LinkedArtifactCount: 0, LinkedAuditEventCount: 0, HasIncompleteTurn: false}, LinkedRunIDs: []string{"run-1"}, LinkedApprovalIDs: []string{}, LinkedArtifactDigests: []string{}, LinkedAuditRecordDigests: []string{}}})
		case "session_send_message":
			return mustOKLocalRPCResponse(t, brokerapi.SessionSendMessageResponse{SchemaID: "runecode.protocol.v0.SessionSendMessageResponse", SchemaVersion: "0.1.0", RequestID: "req-session-send", SessionID: "sess-1", Turn: brokerapi.SessionTranscriptTurn{SchemaID: "runecode.protocol.v0.SessionTranscriptTurn", SchemaVersion: "0.1.0", TurnID: "sess-1.turn.000001", SessionID: "sess-1", TurnIndex: 1, StartedAt: "2026-01-01T00:00:00Z", CompletedAt: "2026-01-01T00:00:00Z", Status: "completed", Messages: []brokerapi.SessionTranscriptMessage{{SchemaID: "runecode.protocol.v0.SessionTranscriptMessage", SchemaVersion: "0.1.0", MessageID: "sess-1.turn.000001.msg.000001", TurnID: "sess-1.turn.000001", SessionID: "sess-1", MessageIndex: 1, Role: "user", CreatedAt: "2026-01-01T00:00:00Z", ContentText: "hello", RelatedLinks: brokerapi.SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{}, ApprovalIDs: []string{}, ArtifactDigests: []string{}, AuditRecordDigests: []string{}}}}}, Message: brokerapi.SessionTranscriptMessage{SchemaID: "runecode.protocol.v0.SessionTranscriptMessage", SchemaVersion: "0.1.0", MessageID: "sess-1.turn.000001.msg.000001", TurnID: "sess-1.turn.000001", SessionID: "sess-1", MessageIndex: 1, Role: "user", CreatedAt: "2026-01-01T00:00:00Z", ContentText: "hello", RelatedLinks: brokerapi.SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{}, ApprovalIDs: []string{}, ArtifactDigests: []string{}, AuditRecordDigests: []string{}}}, EventType: "session_message_ack", StreamID: "session-sess-1", Seq: 1})
		case "session_watch":
			return mustOKLocalRPCResponse(t, []brokerapi.SessionWatchEvent{{SchemaID: "runecode.protocol.v0.SessionWatchEvent", SchemaVersion: "0.1.0", StreamID: "sw-1", RequestID: "req-session-watch", Seq: 1, EventType: "session_watch_snapshot", Session: &brokerapi.SessionSummary{SchemaID: "runecode.protocol.v0.SessionSummary", SchemaVersion: "0.1.0", Identity: brokerapi.SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z"}, UpdatedAt: "2026-01-01T00:00:00Z", Status: "active", LastActivityKind: "chat_message", TurnCount: 1, LinkedRunCount: 1, LinkedApprovalCount: 0, LinkedArtifactCount: 0, LinkedAuditEventCount: 0, HasIncompleteTurn: false}}, {SchemaID: "runecode.protocol.v0.SessionWatchEvent", SchemaVersion: "0.1.0", StreamID: "sw-1", RequestID: "req-session-watch", Seq: 2, EventType: "session_watch_terminal", Terminal: true, TerminalStatus: "completed"}})
		case "approval_list":
			return mustOKLocalRPCResponse(t, brokerapi.ApprovalListResponse{SchemaID: "runecode.protocol.v0.ApprovalListResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-list"})
		case "approval_get":
			return mustOKLocalRPCResponse(t, brokerapi.ApprovalGetResponse{SchemaID: "runecode.protocol.v0.ApprovalGetResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-get", Approval: brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: testDigest("a")}, ApprovalDetail: brokerapi.ApprovalDetail{SchemaID: "runecode.protocol.v0.ApprovalDetail", SchemaVersion: "0.1.0", ApprovalID: testDigest("a"), LifecycleDetail: brokerapi.ApprovalLifecycleDetail{SchemaID: "runecode.protocol.v0.ApprovalLifecycleDetail", SchemaVersion: "0.1.0", LifecycleState: "pending", LifecycleReasonCode: "approval_pending", Stale: false}, BindingKind: "exact_action", BoundActionHash: testDigest("e"), WhatChangesIfApproved: brokerapi.ApprovalWhatChangesIfApproved{SchemaID: "runecode.protocol.v0.ApprovalWhatChangesIfApproved", SchemaVersion: "0.1.0", Summary: "Promote reviewed file excerpts for downstream use.", EffectKind: "promotion"}, BlockedWorkScope: brokerapi.ApprovalBlockedWorkScope{SchemaID: "runecode.protocol.v0.ApprovalBlockedWorkScope", SchemaVersion: "0.1.0", ScopeKind: "step", WorkspaceID: "workspace-local", RunID: "run-1", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion"}, BoundIdentity: brokerapi.ApprovalBoundIdentity{SchemaID: "runecode.protocol.v0.ApprovalBoundIdentity", SchemaVersion: "0.1.0", ApprovalRequestDigest: testDigest("a"), ManifestHash: testDigest("f"), BindingKind: "exact_action", BoundActionHash: testDigest("e")}}})
		case "approval_watch":
			return mustOKLocalRPCResponse(t, []brokerapi.ApprovalWatchEvent{{SchemaID: "runecode.protocol.v0.ApprovalWatchEvent", SchemaVersion: "0.1.0", StreamID: "aw-1", RequestID: "req-approval-watch", Seq: 1, EventType: "approval_watch_snapshot", Approval: &brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: testDigest("a"), Status: "pending", RequestedAt: "2026-01-01T00:00:00Z", ApprovalTriggerCode: "manual_approval_required", ChangesIfApproved: "Promote reviewed file excerpts for downstream use.", ApprovalAssuranceLevel: "session_authenticated", PresenceMode: "os_confirmation", BoundScope: brokerapi.ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", ActionKind: "promotion"}}}, {SchemaID: "runecode.protocol.v0.ApprovalWatchEvent", SchemaVersion: "0.1.0", StreamID: "aw-1", RequestID: "req-approval-watch", Seq: 2, EventType: "approval_watch_terminal", Terminal: true, TerminalStatus: "completed"}})
		case "version_info_get":
			return mustOKLocalRPCResponse(t, brokerapi.VersionInfoGetResponse{SchemaID: "runecode.protocol.v0.VersionInfoGetResponse", SchemaVersion: "0.1.0", RequestID: "req-version", VersionInfo: brokerapi.BrokerVersionInfo{SchemaID: "runecode.protocol.v0.BrokerVersionInfo", SchemaVersion: "0.1.0"}})
		case "log_stream":
			return handleLogStreamDispatchForTest(t, wire, len(*requestedOps))
		case "llm_invoke":
			return mustOKLocalRPCResponse(t, brokerapi.LLMInvokeResponse{SchemaID: "runecode.protocol.v0.LLMInvokeResponse", SchemaVersion: "0.1.0", RequestID: "req-llm-invoke", RunID: "run-1", RequestDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}, Response: map[string]any{"schema_id": "runecode.protocol.v0.LLMResponse", "schema_version": "0.3.0"}})
		case "llm_stream":
			return mustOKLocalRPCResponse(t, brokerapi.LLMStreamEnvelope{SchemaID: "runecode.protocol.v0.LLMStreamEnvelope", SchemaVersion: "0.1.0", RequestID: "req-llm-stream", RunID: "run-1", RequestDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("1", 64)}, Events: []brokerapi.LLMStreamAny{{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": "llm-s-1", "request_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)}, "seq": 1.0, "event_type": "response_start", "emitter": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "role_instance", "principal_id": "brokerapi", "instance_id": "brokerapi-1", "role_family": "gateway", "role_kind": "model-gateway"}}, {"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": "llm-s-1", "request_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)}, "seq": 2.0, "event_type": "response_terminal", "terminal_status": "success", "final_response_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("2", 64)}, "emitter": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "role_instance", "principal_id": "brokerapi", "instance_id": "brokerapi-1", "role_family": "gateway", "role_kind": "model-gateway"}}}})
		default:
			return localRPCResponse{OK: false}
		}
	}
	t.Cleanup(func() { localRPCDispatch = originalDispatch })
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

func handleLogStreamDispatchForTest(t *testing.T, wire localRPCRequest, opCount int) localRPCResponse {
	t.Helper()
	request := brokerapi.LogStreamRequest{}
	if err := json.Unmarshal(wire.Request, &request); err != nil {
		t.Fatalf("Unmarshal log stream request error: %v", err)
	}
	if opCount == 17 && (request.StreamID == "" || request.StreamID == request.RequestID) {
		t.Fatalf("default stream-logs request stream_id = %q, want derived non-empty stream id", request.StreamID)
	}
	if opCount == 18 && request.StreamID != "custom-stream" {
		t.Fatalf("explicit stream-logs request stream_id = %q, want custom-stream", request.StreamID)
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
