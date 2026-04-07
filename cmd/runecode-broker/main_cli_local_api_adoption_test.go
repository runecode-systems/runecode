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

	originalDispatch := localRPCDispatch
	localRPCDispatch = func(_ *brokerapi.Service, _ context.Context, wire localRPCRequest, _ brokerapi.RequestContext) localRPCResponse {
		requestedOps = append(requestedOps, wire.Operation)
		switch wire.Operation {
		case "run_list":
			return mustOKLocalRPCResponse(t, brokerapi.RunListResponse{SchemaID: "runecode.protocol.v0.RunListResponse", SchemaVersion: "0.1.0", RequestID: "req-run-list"})
		case "run_get":
			return mustOKLocalRPCResponse(t, brokerapi.RunGetResponse{SchemaID: "runecode.protocol.v0.RunGetResponse", SchemaVersion: "0.1.0", RequestID: "req-run-get", Run: brokerapi.RunDetail{SchemaID: "runecode.protocol.v0.RunDetail", SchemaVersion: "0.1.0", Summary: brokerapi.RunSummary{SchemaID: "runecode.protocol.v0.RunSummary", SchemaVersion: "0.1.0", RunID: "run-1"}}})
		case "approval_list":
			return mustOKLocalRPCResponse(t, brokerapi.ApprovalListResponse{SchemaID: "runecode.protocol.v0.ApprovalListResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-list"})
		case "approval_get":
			return mustOKLocalRPCResponse(t, brokerapi.ApprovalGetResponse{SchemaID: "runecode.protocol.v0.ApprovalGetResponse", SchemaVersion: "0.1.0", RequestID: "req-approval-get", Approval: brokerapi.ApprovalSummary{SchemaID: "runecode.protocol.v0.ApprovalSummary", SchemaVersion: "0.1.0", ApprovalID: testDigest("a")}})
		case "version_info_get":
			return mustOKLocalRPCResponse(t, brokerapi.VersionInfoGetResponse{SchemaID: "runecode.protocol.v0.VersionInfoGetResponse", SchemaVersion: "0.1.0", RequestID: "req-version", VersionInfo: brokerapi.BrokerVersionInfo{SchemaID: "runecode.protocol.v0.BrokerVersionInfo", SchemaVersion: "0.1.0"}})
		case "log_stream":
			return mustOKLocalRPCResponse(t, []brokerapi.LogStreamEvent{{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-log", Seq: 1, EventType: "log_stream_start"}, {SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "req-log", Seq: 2, EventType: "log_stream_terminal", Terminal: true, TerminalStatus: "completed"}})
		default:
			return localRPCResponse{OK: false}
		}
	}
	t.Cleanup(func() { localRPCDispatch = originalDispatch })

	commands := [][]string{{"run-list"}, {"run-get", "--run-id", "run-1"}, {"approval-list"}, {"approval-get", "--approval-id", testDigest("a")}, {"version-info"}, {"stream-logs"}}
	for _, args := range commands {
		stdout.Reset()
		if err := run(args, stdout, stderr); err != nil {
			t.Fatalf("run(%v) error: %v", args, err)
		}
	}

	want := []string{"run_list", "run_get", "approval_list", "approval_get", "version_info_get", "log_stream"}
	assertRequestedOps(t, requestedOps, want)
}

func TestCLIAdoptionRoutesArtifactAuditAndResolveThroughLocalRPC(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	requestedOps := make([]string, 0, 8)

	installArtifactAuditResolveDispatchStub(t, &requestedOps)
	runArtifactAuditResolveCommands(t, stdout, stderr)

	want := []string{"artifact_list", "artifact_head", "artifact_read", "approval_resolve", "readiness_get", "audit_verification_get"}
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
