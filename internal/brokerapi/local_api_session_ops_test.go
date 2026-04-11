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
	seedSessionForOpsTest(t, s, "run-session-1", "sess-alpha")
	listResp := mustSessionList(t, s, "req-session-list")
	summary := requireSingleSessionSummary(t, listResp, "sess-alpha")
	assertSessionSummaryProjection(t, summary)
	getResp := mustSessionGet(t, s, "req-session-get", "sess-alpha")
	assertSessionDetailProjection(t, getResp, "sess-alpha", "run-session-1")
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
	seedSessionForOpsTest(t, s, "run-session-send", "sess-send")
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
	ack1 := mustSessionSendMessage(t, s, baseReq)
	assertInitialSessionSendAck(t, ack1)
	replayReq := baseReq
	replayReq.RequestID = "req-session-send-2"
	ack2 := mustSessionSendMessage(t, s, replayReq)
	assertSessionSendReplayAck(t, ack1, ack2)
	nextReq := SessionSendMessageRequest{SchemaID: "runecode.protocol.v0.SessionSendMessageRequest", SchemaVersion: "0.1.0", RequestID: "req-session-send-3", SessionID: "sess-send", Role: "user", ContentText: "second"}
	ack3 := mustSessionSendMessage(t, s, nextReq)
	assertSecondDistinctSessionSendAck(t, ack1, ack3)
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

func seedSessionForOpsTest(t *testing.T, s *Service, runID, sessionID string) {
	t.Helper()
	putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	if err := s.RecordRuntimeFacts(runID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: runID, SessionID: sessionID}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
}

func mustSessionList(t *testing.T, s *Service, requestID string) SessionListResponse {
	t.Helper()
	resp, errResp := s.HandleSessionList(context.Background(), SessionListRequest{SchemaID: "runecode.protocol.v0.SessionListRequest", SchemaVersion: "0.1.0", RequestID: requestID, Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionList error response: %+v", errResp)
	}
	return resp
}

func requireSingleSessionSummary(t *testing.T, resp SessionListResponse, wantSessionID string) SessionSummary {
	t.Helper()
	if len(resp.Sessions) != 1 {
		t.Fatalf("session list len = %d, want 1", len(resp.Sessions))
	}
	summary := resp.Sessions[0]
	if summary.Identity.SessionID != wantSessionID {
		t.Fatalf("identity.session_id = %q, want %s", summary.Identity.SessionID, wantSessionID)
	}
	return summary
}

func assertSessionSummaryProjection(t *testing.T, summary SessionSummary) {
	t.Helper()
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
}

func mustSessionGet(t *testing.T, s *Service, requestID, sessionID string) SessionGetResponse {
	t.Helper()
	resp, errResp := s.HandleSessionGet(context.Background(), SessionGetRequest{SchemaID: "runecode.protocol.v0.SessionGetRequest", SchemaVersion: "0.1.0", RequestID: requestID, SessionID: sessionID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionGet error response: %+v", errResp)
	}
	return resp
}

func assertSessionDetailProjection(t *testing.T, resp SessionGetResponse, wantSessionID, wantRunID string) {
	t.Helper()
	if resp.Session.Summary.Identity.SessionID != wantSessionID {
		t.Fatalf("session.summary.identity.session_id = %q, want %s", resp.Session.Summary.Identity.SessionID, wantSessionID)
	}
	assertSessionDetailLinks(t, resp.Session, wantRunID)
	assertSingleSessionTurnProjection(t, resp.Session.TranscriptTurns, wantRunID)
	if len(resp.Session.LinkedAuditRecordDigests) != 0 {
		t.Fatalf("linked_audit_record_digests = %+v, want empty in minimal substrate", resp.Session.LinkedAuditRecordDigests)
	}
}

func assertSessionDetailLinks(t *testing.T, detail SessionDetail, wantRunID string) {
	t.Helper()
	if len(detail.LinkedRunIDs) != 1 || detail.LinkedRunIDs[0] != wantRunID {
		t.Fatalf("linked_run_ids = %+v, want [%s]", detail.LinkedRunIDs, wantRunID)
	}
	if len(detail.LinkedArtifactDigests) != 1 || !strings.HasPrefix(detail.LinkedArtifactDigests[0], "sha256:") {
		t.Fatalf("linked_artifact_digests = %+v, want single sha256 digest", detail.LinkedArtifactDigests)
	}
	if len(detail.LinkedAuditRecordDigests) != 0 {
		t.Fatalf("linked_audit_record_digests = %+v, want empty in minimal substrate", detail.LinkedAuditRecordDigests)
	}
}

func assertSingleSessionTurnProjection(t *testing.T, turns []SessionTranscriptTurn, wantRunID string) {
	t.Helper()
	if len(turns) != 1 {
		t.Fatalf("transcript_turns len = %d, want 1", len(turns))
	}
	turn := turns[0]
	if turn.TurnIndex != 1 {
		t.Fatalf("transcript_turns[0].turn_index = %d, want 1", turn.TurnIndex)
	}
	if len(turn.Messages) != 1 {
		t.Fatalf("transcript_turns[0].messages len = %d, want 1", len(turn.Messages))
	}
	if len(turn.Messages[0].RelatedLinks.RunIDs) != 1 || turn.Messages[0].RelatedLinks.RunIDs[0] != wantRunID {
		t.Fatalf("transcript_turns[0].messages[0].related_links.run_ids = %+v, want [%s]", turn.Messages[0].RelatedLinks.RunIDs, wantRunID)
	}
}

func mustSessionSendMessage(t *testing.T, s *Service, req SessionSendMessageRequest) SessionSendMessageResponse {
	t.Helper()
	resp, errResp := s.HandleSessionSendMessage(context.Background(), req, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionSendMessage error response: %+v", errResp)
	}
	return resp
}

func assertInitialSessionSendAck(t *testing.T, ack SessionSendMessageResponse) {
	t.Helper()
	if ack.EventType != "session_message_ack" {
		t.Fatalf("event_type = %q, want session_message_ack", ack.EventType)
	}
	if ack.StreamID != "session-sess-send" {
		t.Fatalf("stream_id = %q, want session-sess-send", ack.StreamID)
	}
	if ack.Seq != 1 {
		t.Fatalf("seq = %d, want 1", ack.Seq)
	}
	if ack.Message.SessionID != "sess-send" || ack.Turn.SessionID != "sess-send" {
		t.Fatalf("ack session identities mismatch: message=%q turn=%q", ack.Message.SessionID, ack.Turn.SessionID)
	}
	if ack.Message.ContentText != "hello" {
		t.Fatalf("message content_text = %q, want hello", ack.Message.ContentText)
	}
	if len(ack.Message.RelatedLinks.RunIDs) != 1 || ack.Message.RelatedLinks.RunIDs[0] != "run-session-send" {
		t.Fatalf("related_links.run_ids = %+v, want [run-session-send]", ack.Message.RelatedLinks.RunIDs)
	}
	if ack.Turn.TurnIndex != 2 {
		t.Fatalf("turn.turn_index = %d, want 2 based on existing summary turn count", ack.Turn.TurnIndex)
	}
}

func assertSessionSendReplayAck(t *testing.T, first, replay SessionSendMessageResponse) {
	t.Helper()
	if replay.Seq != first.Seq {
		t.Fatalf("idempotent replay seq = %d, want %d", replay.Seq, first.Seq)
	}
	if replay.Message.MessageID != first.Message.MessageID {
		t.Fatalf("idempotent replay message_id = %q, want %q", replay.Message.MessageID, first.Message.MessageID)
	}
	if replay.RequestID != "req-session-send-2" {
		t.Fatalf("replay request_id = %q, want req-session-send-2", replay.RequestID)
	}
}

func assertSecondDistinctSessionSendAck(t *testing.T, first, second SessionSendMessageResponse) {
	t.Helper()
	if second.Seq != 2 {
		t.Fatalf("second distinct seq = %d, want 2", second.Seq)
	}
	if second.Turn.TurnIndex != 3 {
		t.Fatalf("second distinct turn.turn_index = %d, want 3", second.Turn.TurnIndex)
	}
	if second.Message.MessageID == first.Message.MessageID {
		t.Fatalf("second distinct message_id = %q, want different than first %q", second.Message.MessageID, first.Message.MessageID)
	}
}
