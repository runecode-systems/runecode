package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestSessionListAndGetProjectCanonicalSessionIdentity(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-session-1", "step-1")
	if err := s.RecordRuntimeFacts("run-session-1", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-session-1", SessionID: "sess-alpha"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}

	listResp, errResp := s.HandleSessionList(context.Background(), SessionListRequest{SchemaID: "runecode.protocol.v0.SessionListRequest", SchemaVersion: "0.1.0", RequestID: "req-session-list", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionList error response: %+v", errResp)
	}
	if len(listResp.Sessions) != 1 {
		t.Fatalf("session list len = %d, want 1", len(listResp.Sessions))
	}
	summary := listResp.Sessions[0]
	if summary.Identity.SessionID != "sess-alpha" {
		t.Fatalf("identity.session_id = %q, want sess-alpha", summary.Identity.SessionID)
	}
	if summary.Identity.SchemaID != "runecode.protocol.v0.SessionIdentity" {
		t.Fatalf("identity.schema_id = %q, want SessionIdentity", summary.Identity.SchemaID)
	}
	if summary.Identity.WorkspaceID != "workspace-local" {
		t.Fatalf("identity.workspace_id = %q, want workspace-local", summary.Identity.WorkspaceID)
	}
	if summary.LinkedRunCount != 1 {
		t.Fatalf("linked_run_count = %d, want 1", summary.LinkedRunCount)
	}
	if summary.TurnCount != 1 {
		t.Fatalf("turn_count = %d, want 1 for minimal ordered transcript substrate", summary.TurnCount)
	}

	getResp, getErr := s.HandleSessionGet(context.Background(), SessionGetRequest{SchemaID: "runecode.protocol.v0.SessionGetRequest", SchemaVersion: "0.1.0", RequestID: "req-session-get", SessionID: "sess-alpha"}, RequestContext{})
	if getErr != nil {
		t.Fatalf("HandleSessionGet error response: %+v", getErr)
	}
	if getResp.Session.Summary.Identity.SessionID != "sess-alpha" {
		t.Fatalf("session.summary.identity.session_id = %q, want sess-alpha", getResp.Session.Summary.Identity.SessionID)
	}
	if len(getResp.Session.LinkedRunIDs) != 1 || getResp.Session.LinkedRunIDs[0] != "run-session-1" {
		t.Fatalf("linked_run_ids = %+v, want [run-session-1]", getResp.Session.LinkedRunIDs)
	}
	if len(getResp.Session.LinkedArtifactDigests) != 1 || !strings.HasPrefix(getResp.Session.LinkedArtifactDigests[0], "sha256:") {
		t.Fatalf("linked_artifact_digests = %+v, want single sha256 digest", getResp.Session.LinkedArtifactDigests)
	}
	if len(getResp.Session.TranscriptTurns) != 1 {
		t.Fatalf("transcript_turns len = %d, want 1", len(getResp.Session.TranscriptTurns))
	}
	if getResp.Session.TranscriptTurns[0].TurnIndex != 1 {
		t.Fatalf("transcript_turns[0].turn_index = %d, want 1", getResp.Session.TranscriptTurns[0].TurnIndex)
	}
	if len(getResp.Session.TranscriptTurns[0].Messages) != 1 {
		t.Fatalf("transcript_turns[0].messages len = %d, want 1", len(getResp.Session.TranscriptTurns[0].Messages))
	}
	if len(getResp.Session.TranscriptTurns[0].Messages[0].RelatedLinks.RunIDs) != 1 || getResp.Session.TranscriptTurns[0].Messages[0].RelatedLinks.RunIDs[0] != "run-session-1" {
		t.Fatalf("transcript_turns[0].messages[0].related_links.run_ids = %+v, want [run-session-1]", getResp.Session.TranscriptTurns[0].Messages[0].RelatedLinks.RunIDs)
	}
	if len(getResp.Session.LinkedAuditRecordDigests) != 0 {
		t.Fatalf("linked_audit_record_digests = %+v, want empty in minimal substrate", getResp.Session.LinkedAuditRecordDigests)
	}
}

func TestSessionGetNotFoundUsesSessionSpecificCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleSessionGet(context.Background(), SessionGetRequest{SchemaID: "runecode.protocol.v0.SessionGetRequest", SchemaVersion: "0.1.0", RequestID: "req-session-missing", SessionID: "sess-missing"}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleSessionGet expected not-found error")
	}
	if errResp.Error.Code != "broker_not_found_session" {
		t.Fatalf("error code = %q, want broker_not_found_session", errResp.Error.Code)
	}
}

func TestSessionSendMessageReturnsTypedAckAndSupportsIdempotency(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-session-send", "step-1")
	if err := s.RecordRuntimeFacts("run-session-send", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-session-send", SessionID: "sess-send"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	baseReq := SessionSendMessageRequest{
		SchemaID:       "runecode.protocol.v0.SessionSendMessageRequest",
		SchemaVersion:  "0.1.0",
		RequestID:      "req-session-send-1",
		SessionID:      "sess-send",
		Role:           "user",
		ContentText:    "hello",
		IdempotencyKey: "idem-1",
		RelatedLinks:   &SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{"run-session-send"}, ApprovalIDs: []string{}, ArtifactDigests: []string{}, AuditRecordDigests: []string{}},
	}
	ack1, errResp := s.HandleSessionSendMessage(context.Background(), baseReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionSendMessage first error response: %+v", errResp)
	}
	if ack1.EventType != "session_message_ack" {
		t.Fatalf("event_type = %q, want session_message_ack", ack1.EventType)
	}
	if ack1.StreamID != "session-sess-send" {
		t.Fatalf("stream_id = %q, want session-sess-send", ack1.StreamID)
	}
	if ack1.Seq != 1 {
		t.Fatalf("seq = %d, want 1", ack1.Seq)
	}
	if ack1.Message.SessionID != "sess-send" || ack1.Turn.SessionID != "sess-send" {
		t.Fatalf("ack session identities mismatch: message=%q turn=%q", ack1.Message.SessionID, ack1.Turn.SessionID)
	}
	if ack1.Message.ContentText != "hello" {
		t.Fatalf("message content_text = %q, want hello", ack1.Message.ContentText)
	}
	if len(ack1.Message.RelatedLinks.RunIDs) != 1 || ack1.Message.RelatedLinks.RunIDs[0] != "run-session-send" {
		t.Fatalf("related_links.run_ids = %+v, want [run-session-send]", ack1.Message.RelatedLinks.RunIDs)
	}
	if ack1.Turn.TurnIndex != 2 {
		t.Fatalf("turn.turn_index = %d, want 2 based on existing summary turn count", ack1.Turn.TurnIndex)
	}
	replayReq := baseReq
	replayReq.RequestID = "req-session-send-2"
	ack2, errResp := s.HandleSessionSendMessage(context.Background(), replayReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionSendMessage replay error response: %+v", errResp)
	}
	if ack2.Seq != ack1.Seq {
		t.Fatalf("idempotent replay seq = %d, want %d", ack2.Seq, ack1.Seq)
	}
	if ack2.Message.MessageID != ack1.Message.MessageID {
		t.Fatalf("idempotent replay message_id = %q, want %q", ack2.Message.MessageID, ack1.Message.MessageID)
	}
	if ack2.RequestID != "req-session-send-2" {
		t.Fatalf("replay request_id = %q, want req-session-send-2", ack2.RequestID)
	}
	nextReq := SessionSendMessageRequest{SchemaID: "runecode.protocol.v0.SessionSendMessageRequest", SchemaVersion: "0.1.0", RequestID: "req-session-send-3", SessionID: "sess-send", Role: "user", ContentText: "second"}
	ack3, errResp := s.HandleSessionSendMessage(context.Background(), nextReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionSendMessage second distinct error response: %+v", errResp)
	}
	if ack3.Seq != 2 {
		t.Fatalf("second distinct seq = %d, want 2", ack3.Seq)
	}
	if ack3.Turn.TurnIndex != 3 {
		t.Fatalf("second distinct turn.turn_index = %d, want 3", ack3.Turn.TurnIndex)
	}
	if ack3.Message.MessageID == ack1.Message.MessageID {
		t.Fatalf("second distinct message_id = %q, want different than first %q", ack3.Message.MessageID, ack1.Message.MessageID)
	}
}

func TestBuildSessionTranscriptTurnsCapsToSchemaLimits(t *testing.T) {
	summary := SessionSummary{TurnCount: 3000, UpdatedAt: "2026-01-01T00:00:00Z", LastActivityPreview: "preview"}
	runs := map[string]struct{}{}
	approvals := map[string]struct{}{}
	artifactsByDigest := map[string]struct{}{}
	audit := map[string]struct{}{}
	for i := 0; i < 2000; i++ {
		digest := fmt.Sprintf("sha256:%064x", i)
		runs[fmt.Sprintf("run-%d", i)] = struct{}{}
		approvals[digest] = struct{}{}
		artifactsByDigest[digest] = struct{}{}
		audit[digest] = struct{}{}
	}

	turns := buildSessionTranscriptTurns("sess-cap", summary, runs, approvals, artifactsByDigest, audit)
	if len(turns) != 2048 {
		t.Fatalf("turn count = %d, want 2048", len(turns))
	}
}
