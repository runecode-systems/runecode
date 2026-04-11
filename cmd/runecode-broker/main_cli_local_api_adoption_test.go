package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestCLIAdoptionRoutesRunApprovalVersionAndLogThroughLocalRPC(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	requestedOps := make([]string, 0, 8)

	installRunApprovalVersionLogDispatchStub(t, &requestedOps)

	commands := [][]string{{"run-list"}, {"run-get", "--run-id", "run-1"}, {"session-list"}, {"session-get", "--session-id", "sess-1"}, {"session-send-message", "--session-id", "sess-1", "--content", "hello"}, {"approval-list"}, {"approval-get", "--approval-id", testDigest("a")}, {"version-info"}, {"stream-logs"}, {"stream-logs", "--stream-id", "custom-stream"}}
	for _, args := range commands {
		stdout.Reset()
		if err := run(args, stdout, stderr); err != nil {
			t.Fatalf("run(%v) error: %v", args, err)
		}
	}

	want := []string{"run_list", "run_get", "session_list", "session_get", "session_send_message", "approval_list", "approval_get", "version_info_get", "log_stream", "log_stream"}
	assertRequestedOps(t, requestedOps, want)
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
			return mustOKLocalRPCResponse(t, brokerapi.RunGetResponse{SchemaID: "runecode.protocol.v0.RunGetResponse", SchemaVersion: "0.1.0", RequestID: "req-run-get", Run: brokerapi.RunDetail{SchemaID: "runecode.protocol.v0.RunDetail", SchemaVersion: "0.2.0", Summary: brokerapi.RunSummary{SchemaID: "runecode.protocol.v0.RunSummary", SchemaVersion: "0.2.0", RunID: "run-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z", LifecycleState: "pending", PendingApprovalCount: 0, ApprovalProfile: "unknown", BackendKind: "unknown", IsolationAssuranceLevel: "unknown", ProvisioningPosture: "unknown", AssuranceLevel: "unknown", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok", AuditCurrentlyDegraded: false}}})
		case "session_list":
			return mustOKLocalRPCResponse(t, brokerapi.SessionListResponse{SchemaID: "runecode.protocol.v0.SessionListResponse", SchemaVersion: "0.1.0", RequestID: "req-session-list", Order: "updated_at_desc", Sessions: []brokerapi.SessionSummary{{SchemaID: "runecode.protocol.v0.SessionSummary", SchemaVersion: "0.1.0", Identity: brokerapi.SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z"}, UpdatedAt: "2026-01-01T00:00:00Z", Status: "active", LastActivityKind: "run_progress", TurnCount: 0, LinkedRunCount: 1, LinkedApprovalCount: 0, LinkedArtifactCount: 0, LinkedAuditEventCount: 0, HasIncompleteTurn: false}}})
		case "session_get":
			return mustOKLocalRPCResponse(t, brokerapi.SessionGetResponse{SchemaID: "runecode.protocol.v0.SessionGetResponse", SchemaVersion: "0.1.0", RequestID: "req-session-get", Session: brokerapi.SessionDetail{SchemaID: "runecode.protocol.v0.SessionDetail", SchemaVersion: "0.1.0", Summary: brokerapi.SessionSummary{SchemaID: "runecode.protocol.v0.SessionSummary", SchemaVersion: "0.1.0", Identity: brokerapi.SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: "sess-1", WorkspaceID: "workspace-local", CreatedAt: "2026-01-01T00:00:00Z"}, UpdatedAt: "2026-01-01T00:00:00Z", Status: "active", LastActivityKind: "run_progress", TurnCount: 0, LinkedRunCount: 1, LinkedApprovalCount: 0, LinkedArtifactCount: 0, LinkedAuditEventCount: 0, HasIncompleteTurn: false}, LinkedRunIDs: []string{"run-1"}, LinkedApprovalIDs: []string{}, LinkedArtifactDigests: []string{}, LinkedAuditRecordDigests: []string{}}})
		case "session_send_message":
			return mustOKLocalRPCResponse(t, brokerapi.SessionSendMessageResponse{SchemaID: "runecode.protocol.v0.SessionSendMessageResponse", SchemaVersion: "0.1.0", RequestID: "req-session-send", SessionID: "sess-1", Turn: brokerapi.SessionTranscriptTurn{SchemaID: "runecode.protocol.v0.SessionTranscriptTurn", SchemaVersion: "0.1.0", TurnID: "sess-1.turn.000001", SessionID: "sess-1", TurnIndex: 1, StartedAt: "2026-01-01T00:00:00Z", CompletedAt: "2026-01-01T00:00:00Z", Status: "completed", Messages: []brokerapi.SessionTranscriptMessage{{SchemaID: "runecode.protocol.v0.SessionTranscriptMessage", SchemaVersion: "0.1.0", MessageID: "sess-1.turn.000001.msg.000001", TurnID: "sess-1.turn.000001", SessionID: "sess-1", MessageIndex: 1, Role: "user", CreatedAt: "2026-01-01T00:00:00Z", ContentText: "hello", RelatedLinks: brokerapi.SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{}, ApprovalIDs: []string{}, ArtifactDigests: []string{}, AuditRecordDigests: []string{}}}}}, Message: brokerapi.SessionTranscriptMessage{SchemaID: "runecode.protocol.v0.SessionTranscriptMessage", SchemaVersion: "0.1.0", MessageID: "sess-1.turn.000001.msg.000001", TurnID: "sess-1.turn.000001", SessionID: "sess-1", MessageIndex: 1, Role: "user", CreatedAt: "2026-01-01T00:00:00Z", ContentText: "hello", RelatedLinks: brokerapi.SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{}, ApprovalIDs: []string{}, ArtifactDigests: []string{}, AuditRecordDigests: []string{}}}, EventType: "session_message_ack", StreamID: "session-sess-1", Seq: 1})
		case "approval_list":
			return mustOKLocalRPCResponse(t, brokerapi.ApprovalListResponse{SchemaID: "runecode.protocol.v0.ApprovalListResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-list"})
		case "approval_get":
			return mustOKLocalRPCResponse(t, brokerapi.ApprovalGetResponse{SchemaID: "runecode.protocol.v0.ApprovalGetResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-get", Approval: brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: testDigest("a")}})
		case "version_info_get":
			return mustOKLocalRPCResponse(t, brokerapi.VersionInfoGetResponse{SchemaID: "runecode.protocol.v0.VersionInfoGetResponse", SchemaVersion: "0.1.0", RequestID: "req-version", VersionInfo: brokerapi.BrokerVersionInfo{SchemaID: "runecode.protocol.v0.BrokerVersionInfo", SchemaVersion: "0.1.0"}})
		case "log_stream":
			return handleLogStreamDispatchForTest(t, wire, len(*requestedOps))
		default:
			return localRPCResponse{OK: false}
		}
	}
	t.Cleanup(func() { localRPCDispatch = originalDispatch })
}

func handleLogStreamDispatchForTest(t *testing.T, wire localRPCRequest, opCount int) localRPCResponse {
	t.Helper()
	request := brokerapi.LogStreamRequest{}
	if err := json.Unmarshal(wire.Request, &request); err != nil {
		t.Fatalf("Unmarshal log stream request error: %v", err)
	}
	if opCount == 9 && (request.StreamID == "" || request.StreamID == request.RequestID) {
		t.Fatalf("default stream-logs request stream_id = %q, want derived non-empty stream id", request.StreamID)
	}
	if opCount == 10 && request.StreamID != "custom-stream" {
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

	want := []string{"artifact_list", "artifact_head", "artifact_read", "approval_get", "approval_resolve", "readiness_get", "audit_verification_get"}
	assertRequestedOps(t, requestedOps, want)
}

func installArtifactAuditResolveDispatchStub(t *testing.T, requestedOps *[]string) {
	t.Helper()
	originalDispatch := localRPCDispatch
	localRPCDispatch = func(_ *brokerapi.Service, _ context.Context, wire localRPCRequest, _ brokerapi.RequestContext) localRPCResponse {
		*requestedOps = append(*requestedOps, wire.Operation)
		switch wire.Operation {
		case "artifact_list":
			return mustOKLocalRPCResponse(t, brokerapi.LocalArtifactListResponse{SchemaID: "runecode.protocol.v0.ArtifactListResponse", SchemaVersion: "0.1.0", RequestID: "req-art-list", Artifacts: []brokerapi.ArtifactSummary{{SchemaID: "runecode.protocol.v0.ArtifactSummary", SchemaVersion: "0.1.0", Reference: artifacts.ArtifactReference{Digest: testDigest("b"), DataClass: artifacts.DataClassSpecText}, CreatedAt: "2026-01-01T00:00:00Z", CreatedByRole: "workspace"}}})
		case "artifact_head":
			return mustOKLocalRPCResponse(t, brokerapi.LocalArtifactHeadResponse{SchemaID: "runecode.protocol.v0.ArtifactHeadResponse", SchemaVersion: "0.1.0", RequestID: "req-art-head", Artifact: brokerapi.ArtifactSummary{SchemaID: "runecode.protocol.v0.ArtifactSummary", SchemaVersion: "0.1.0", Reference: artifacts.ArtifactReference{Digest: testDigest("b"), DataClass: artifacts.DataClassSpecText}, CreatedAt: "2026-01-01T00:00:00Z", CreatedByRole: "workspace"}})
		case "artifact_read":
			return mustOKLocalRPCResponse(t, []brokerapi.ArtifactStreamEvent{{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-art-read", Seq: 1, EventType: "artifact_stream_start", Digest: testDigest("b"), DataClass: "spec_text"}, {SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-art-read", Seq: 2, EventType: "artifact_stream_chunk", Digest: testDigest("b"), DataClass: "spec_text", ChunkBase64: base64.StdEncoding.EncodeToString([]byte("hello")), ChunkBytes: 5}, {SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-art-read", Seq: 3, EventType: "artifact_stream_terminal", Digest: testDigest("b"), DataClass: "spec_text", Terminal: true, TerminalStatus: "completed"}})
		case "approval_get":
			request := brokerapi.ApprovalGetRequest{}
			if err := json.Unmarshal(wire.Request, &request); err != nil {
				t.Fatalf("Unmarshal approval_get request error: %v", err)
			}
			return mustOKLocalRPCResponse(t, brokerapi.ApprovalGetResponse{SchemaID: "runecode.protocol.v0.ApprovalGetResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-get", Approval: brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: request.ApprovalID, BoundScope: brokerapi.ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", ActionKind: "promotion"}}})
		case "approval_resolve":
			return mustOKLocalRPCResponse(t, brokerapi.ApprovalResolveResponse{SchemaID: "runecode.protocol.v0.ApprovalResolveResponse", SchemaVersion: "0.1.0", RequestID: "req-resolve", ResolutionStatus: "resolved", ResolutionReasonCode: "approval_approved", Approval: brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: testDigest("c")}, ApprovedArtifact: &brokerapi.ArtifactSummary{SchemaID: "runecode.protocol.v0.ArtifactSummary", SchemaVersion: "0.1.0", Reference: artifacts.ArtifactReference{Digest: testDigest("d"), DataClass: artifacts.DataClassApprovedFileExcerpts}, CreatedAt: "2026-01-01T00:00:00Z", CreatedByRole: "workspace"}})
		case "readiness_get":
			return mustOKLocalRPCResponse(t, brokerapi.ReadinessGetResponse{SchemaID: "runecode.protocol.v0.ReadinessGetResponse", SchemaVersion: "0.1.0", RequestID: "req-readiness", Readiness: brokerapi.BrokerReadiness{SchemaID: "runecode.protocol.v0.BrokerReadiness", SchemaVersion: "0.1.0", Ready: true, LocalOnly: true, ConsumptionChannel: "broker_local_api", RecoveryComplete: true, AppendPositionStable: true, CurrentSegmentWritable: true, VerifierMaterialAvailable: true, DerivedIndexCaughtUp: true}})
		case "audit_verification_get":
			return mustOKLocalRPCResponse(t, brokerapi.AuditVerificationGetResponse{SchemaID: "runecode.protocol.v0.AuditVerificationGetResponse", SchemaVersion: "0.1.0", RequestID: "req-audit"})
		default:
			return localRPCResponse{OK: false}
		}
	}
	t.Cleanup(func() { localRPCDispatch = originalDispatch })
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
