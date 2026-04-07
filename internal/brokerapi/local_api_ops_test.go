package brokerapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestRunAndArtifactLocalTypedOperations(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	digest := putRunScopedArtifactForLocalOpsTest(t, s, "run-123", "step-1")
	assertRunListAndDetailForLocalOps(t, s)
	assertArtifactListAndHeadForLocalOps(t, s, digest)
	assertArtifactReadStreamCompletes(t, s, digest)
}

func assertRunListAndDetailForLocalOps(t *testing.T, s *Service) {
	t.Helper()
	runList, errResp := s.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "req-run-list", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunList error response: %+v", errResp)
	}
	if len(runList.Runs) != 1 || runList.Runs[0].RunID != "run-123" {
		t.Fatalf("run list = %+v, want run-123", runList.Runs)
	}
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-get", RunID: "run-123"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if runGet.Run.Summary.RunID != "run-123" {
		t.Fatalf("run detail run_id = %q, want run-123", runGet.Run.Summary.RunID)
	}
}

func assertArtifactListAndHeadForLocalOps(t *testing.T, s *Service, digest string) {
	t.Helper()
	artList, errResp := s.HandleArtifactListV0(context.Background(), LocalArtifactListRequest{SchemaID: "runecode.protocol.v0.ArtifactListRequest", SchemaVersion: "0.1.0", RequestID: "req-art-list", Order: "created_at_desc", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactListV0 error response: %+v", errResp)
	}
	if len(artList.Artifacts) != 1 || artList.Artifacts[0].RunID != "run-123" {
		t.Fatalf("artifact list = %+v, want run-scoped artifact", artList.Artifacts)
	}
	headReq := LocalArtifactHeadRequest{SchemaID: "runecode.protocol.v0.ArtifactHeadRequest", SchemaVersion: "0.1.0", RequestID: "req-art-head", Digest: artList.Artifacts[0].Reference.Digest}
	headResp, errResp := s.HandleArtifactHeadV0(context.Background(), headReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactHeadV0 error response: %+v", errResp)
	}
	if headResp.Artifact.Reference.Digest != digest {
		t.Fatalf("artifact head digest = %q, want %q", headResp.Artifact.Reference.Digest, digest)
	}
}

func putRunScopedArtifactForLocalOpsTest(t *testing.T, s *Service, runID, stepID string) string {
	t.Helper()
	ref, putErr := s.Put(artifacts.PutRequest{
		Payload:               []byte("artifact-a"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64),
		CreatedByRole:         "workspace",
		RunID:                 runID,
		StepID:                stepID,
	})
	if putErr != nil {
		t.Fatalf("Put returned error: %v", putErr)
	}
	return ref.Digest
}

func assertArtifactReadStreamCompletes(t *testing.T, s *Service, digest string) {
	t.Helper()
	readReq := ArtifactReadRequest{SchemaID: "runecode.protocol.v0.ArtifactReadRequest", SchemaVersion: "0.1.0", RequestID: "req-art-read", Digest: digest, ProducerRole: "workspace", ConsumerRole: "model_gateway", ChunkBytes: 128}
	readResp, errResp := s.HandleArtifactRead(context.Background(), readReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactRead error response: %+v", errResp)
	}
	if readResp.StreamID == "" || readResp.Reader == nil {
		t.Fatalf("artifact read response invalid: %+v", readResp)
	}
	events, err := s.StreamArtifactReadEvents(readResp)
	if err != nil {
		t.Fatalf("StreamArtifactReadEvents error: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("artifact stream events = %d, want at least start+terminal", len(events))
	}
	if events[0].EventType != "artifact_stream_start" {
		t.Fatalf("first stream event_type = %q, want artifact_stream_start", events[0].EventType)
	}
	assertStreamSeqMonotonic(t, events)
	assertSingleArtifactTerminalEvent(t, events)
	if got := events[len(events)-1].TerminalStatus; got != "completed" {
		t.Fatalf("terminal_status = %q, want completed", got)
	}
}

func assertStreamSeqMonotonic(t *testing.T, events []ArtifactStreamEvent) {
	t.Helper()
	for i := 1; i < len(events); i++ {
		if events[i].Seq <= events[i-1].Seq {
			t.Fatalf("stream seq not monotonic: prev=%d curr=%d", events[i-1].Seq, events[i].Seq)
		}
	}
}

func assertSingleArtifactTerminalEvent(t *testing.T, events []ArtifactStreamEvent) {
	t.Helper()
	terminalCount := 0
	for _, event := range events {
		if event.EventType != "artifact_stream_terminal" {
			continue
		}
		terminalCount++
		if event.TerminalStatus == "" {
			t.Fatal("artifact terminal event missing in-band terminal_status")
		}
	}
	if terminalCount != 1 {
		t.Fatalf("terminal event count = %d, want 1", terminalCount)
	}
}

func TestArtifactReadRejectsRangeRequestsInMVP(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref, putErr := s.Put(artifacts.PutRequest{
		Payload:               []byte("artifact-a"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64),
		CreatedByRole:         "workspace",
	})
	if putErr != nil {
		t.Fatalf("Put returned error: %v", putErr)
	}
	rangeStart := int64(0)
	_, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactReadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-range",
		Digest:        ref.Digest,
		ProducerRole:  "workspace",
		ConsumerRole:  "model_gateway",
		RangeStart:    &rangeStart,
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleArtifactRead expected range rejection error")
	}
	if errResp.Error.Code != "broker_validation_range_not_supported" {
		t.Fatalf("error code = %q, want broker_validation_range_not_supported", errResp.Error.Code)
	}
}

func TestArtifactReadRequiresManifestOptInForApprovedExcerpt(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	approved := setupApprovedExcerptArtifactForReadTests(t, s)

	t.Run("reject without manifest opt-in", func(t *testing.T) {
		_, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{
			SchemaID:      "runecode.protocol.v0.ArtifactReadRequest",
			SchemaVersion: "0.1.0",
			RequestID:     "req-approved-deny",
			Digest:        approved.Digest,
			ProducerRole:  "workspace",
			ConsumerRole:  "model_gateway",
			DataClass:     string(artifacts.DataClassApprovedFileExcerpts),
		}, RequestContext{})
		if errResp == nil {
			t.Fatal("HandleArtifactRead expected manifest opt-in denial")
		}
		if errResp.Error.Code != "broker_limit_policy_rejected" {
			t.Fatalf("error code = %q, want broker_limit_policy_rejected", errResp.Error.Code)
		}
	})

	t.Run("allow with manifest opt-in", func(t *testing.T) {
		handle, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{
			SchemaID:      "runecode.protocol.v0.ArtifactReadRequest",
			SchemaVersion: "0.1.0",
			RequestID:     "req-approved-allow",
			Digest:        approved.Digest,
			ProducerRole:  "workspace",
			ConsumerRole:  "model_gateway",
			DataClass:     string(artifacts.DataClassApprovedFileExcerpts),
			ManifestOptIn: true,
		}, RequestContext{})
		if errResp != nil {
			t.Fatalf("HandleArtifactRead with manifest opt-in error response: %+v", errResp)
		}
		events, streamErr := s.StreamArtifactReadEvents(handle)
		if streamErr != nil {
			t.Fatalf("StreamArtifactReadEvents returned error: %v", streamErr)
		}
		assertArtifactStreamDecodedPayload(t, events, "approved:\nprivate excerpt")
	})
}

func setupApprovedExcerptArtifactForReadTests(t *testing.T, s *Service) artifacts.ArtifactReference {
	t.Helper()
	unapproved, err := s.Put(artifacts.PutRequest{
		Payload:               []byte("private excerpt"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64),
		CreatedByRole:         "workspace",
	})
	if err != nil {
		t.Fatalf("Put unapproved returned error: %v", err)
	}
	requestEnv, decisionEnv, verifiers := signedApprovalArtifactsForBrokerTests(t, "human", unapproved.Digest)
	for _, verifier := range verifiers {
		if putErr := putTrustedVerifierRecordForService(s, verifier); putErr != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", putErr)
		}
	}
	approved, err := s.PromoteApprovedExcerpt(artifacts.PromotionRequest{
		UnapprovedDigest:     unapproved.Digest,
		Approver:             "human",
		ApprovalRequest:      requestEnv,
		ApprovalDecision:     decisionEnv,
		RepoPath:             "repo/file.txt",
		Commit:               "abc123",
		ExtractorToolVersion: "tool-v1",
		FullContentVisible:   true,
	})
	if err != nil {
		t.Fatalf("PromoteApprovedExcerpt returned error: %v", err)
	}
	return approved
}

func assertArtifactStreamDecodedPayload(t *testing.T, events []ArtifactStreamEvent, want string) {
	t.Helper()
	decoded := ""
	for _, event := range events {
		if event.EventType != "artifact_stream_chunk" {
			continue
		}
		chunk, decodeErr := base64.StdEncoding.DecodeString(event.ChunkBase64)
		if decodeErr != nil {
			t.Fatalf("chunk decode error: %v", decodeErr)
		}
		decoded += string(chunk)
	}
	if decoded != want {
		t.Fatalf("decoded artifact stream payload = %q, want %q", decoded, want)
	}
}

func TestArtifactReadGatewayFlowDeniedForMismatchedProducerRole(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref, putErr := s.Put(artifacts.PutRequest{
		Payload:               []byte("artifact-a"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64),
		CreatedByRole:         "workspace",
	})
	if putErr != nil {
		t.Fatalf("Put returned error: %v", putErr)
	}
	_, errResp := s.HandleArtifactRead(context.Background(), ArtifactReadRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactReadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-role-mismatch",
		Digest:        ref.Digest,
		ProducerRole:  "auditd",
		ConsumerRole:  "model_gateway",
		DataClass:     string(artifacts.DataClassSpecText),
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleArtifactRead expected policy rejection for mismatched producer")
	}
	if errResp.Error.Code != "broker_limit_policy_rejected" {
		t.Fatalf("error code = %q, want broker_limit_policy_rejected", errResp.Error.Code)
	}
}

func TestArtifactStreamEventsCloseWithSingleTerminalOnReadFailure(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	events, err := s.StreamArtifactReadEvents(ArtifactReadHandle{
		RequestID:  "req-stream-fail",
		Digest:     "sha256:" + strings.Repeat("a", 64),
		DataClass:  artifacts.DataClassSpecText,
		StreamID:   "stream-fail",
		ChunkBytes: 8,
		Reader:     io.NopCloser(&alwaysErrReader{}),
	})
	if err != nil {
		t.Fatalf("StreamArtifactReadEvents returned error: %v", err)
	}
	terminal := 0
	for _, event := range events {
		if event.EventType == "artifact_stream_terminal" {
			terminal++
			if event.TerminalStatus != "failed" {
				t.Fatalf("terminal_status = %q, want failed", event.TerminalStatus)
			}
			if event.Error == nil {
				t.Fatal("terminal failure event missing typed error envelope")
			}
		}
	}
	if terminal != 1 {
		t.Fatalf("terminal event count = %d, want 1", terminal)
	}
}

func TestArtifactStreamSemanticsRejectNonMonotonicSeq(t *testing.T) {
	err := validateArtifactStreamSemantics([]ArtifactStreamEvent{
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "artifact_stream_start", Digest: "sha256:" + strings.Repeat("a", 64), DataClass: "spec_text"},
		{SchemaID: "runecode.protocol.v0.ArtifactStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "artifact_stream_terminal", Digest: "sha256:" + strings.Repeat("a", 64), DataClass: "spec_text", Terminal: true, TerminalStatus: "completed"},
	})
	if err == nil {
		t.Fatal("validateArtifactStreamSemantics expected non-monotonic seq error")
	}
}

func TestLogStreamSemanticsRejectMultipleTerminalEvents(t *testing.T) {
	err := validateLogStreamSemantics([]LogStreamEvent{
		{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 1, EventType: "log_stream_start"},
		{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 2, EventType: "log_stream_terminal", Terminal: true, TerminalStatus: "completed"},
		{SchemaID: "runecode.protocol.v0.LogStreamEvent", SchemaVersion: "0.1.0", StreamID: "s-1", RequestID: "r-1", Seq: 3, EventType: "log_stream_terminal", Terminal: true, TerminalStatus: "completed"},
	})
	if err == nil {
		t.Fatal("validateLogStreamSemantics expected multiple terminal error")
	}
}

type alwaysErrReader struct{}

func (r *alwaysErrReader) Read(_ []byte) (int, error) {
	return 0, errors.New("forced stream read failure")
}

func TestHandleRunListRejectsAdmissionFailure(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleRunList(
		context.Background(),
		RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "req-admission", Limit: 10},
		RequestContext{AdmissionErr: errors.New("peer credentials unavailable")},
	)
	if errResp == nil {
		t.Fatal("HandleRunList error = nil, want typed auth admission error")
	}
	if errResp.Error.Code != "broker_api_auth_admission_denied" {
		t.Fatalf("error code = %q, want broker_api_auth_admission_denied", errResp.Error.Code)
	}
}

func TestApprovalResolveAndAuditReadinessVersionOperations(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)

	resolveReq := ApprovalResolveRequest{
		SchemaID:               "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-approval-resolve",
		ApprovalID:             approvalID,
		BoundScope:             ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: "workspace-1", RunID: "run-approval", StageID: "artifact_flow", ActionKind: "artifact_promotion"},
		UnapprovedDigest:       unapproved.Digest,
		Approver:               "human",
		RepoPath:               "repo/file.txt",
		Commit:                 "abc123",
		ExtractorToolVersion:   "tool-v1",
		FullContentVisible:     true,
		ExplicitViewFull:       false,
		BulkRequest:            false,
		BulkApprovalConfirmed:  false,
		SignedApprovalRequest:  *requestEnv,
		SignedApprovalDecision: *decisionEnv,
	}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.ResolutionStatus != "resolved" {
		t.Fatalf("resolution_status = %q, want resolved", resolveResp.ResolutionStatus)
	}

	assertApprovalAndAuditReadEndpoints(t, s, approvalID)
	assertVersionAndLogEndpoints(t, s)
}

func setupServiceWithApprovalFixture(t *testing.T) (*Service, artifacts.ArtifactReference, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	root := t.TempDir()
	ledgerRoot := root + "/audit-ledger"
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	s, err := NewServiceWithConfig(root, ledgerRoot, APIConfig{})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	unapproved, err := s.Put(artifacts.PutRequest{
		Payload:               []byte("private excerpt"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64),
		CreatedByRole:         "workspace",
		RunID:                 "run-approval",
		StepID:                "step-1",
	})
	if err != nil {
		t.Fatalf("Put unapproved returned error: %v", err)
	}
	requestEnv, decisionEnv, verifiers := signedApprovalArtifactsForBrokerTests(t, "human", unapproved.Digest)
	for _, verifier := range verifiers {
		if putErr := putTrustedVerifierRecordForService(s, verifier); putErr != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", putErr)
		}
	}
	return s, unapproved, requestEnv, decisionEnv
}

func approvalIDForBrokerTest(t *testing.T, requestEnv *trustpolicy.SignedObjectEnvelope) string {
	t.Helper()
	approvalID, idErr := approvalIDFromRequest(*requestEnv)
	if idErr != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", idErr)
	}
	return approvalID
}

func assertApprovalAndAuditReadEndpoints(t *testing.T, s *Service, approvalID string) {
	t.Helper()

	getResp, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get", ApprovalID: approvalID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalGet error response: %+v", errResp)
	}
	if getResp.Approval.ApprovalID != approvalID || getResp.SignedApprovalDecision == nil {
		t.Fatalf("approval get response invalid: %+v", getResp)
	}

	listResp, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-list", Order: "pending_first_newest_within_status", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalList error response: %+v", errResp)
	}
	if len(listResp.Approvals) == 0 {
		t.Fatal("approval list empty, want at least resolved approval")
	}

	auditResp, errResp := s.HandleAuditVerificationGet(context.Background(), AuditVerificationGetRequest{SchemaID: "runecode.protocol.v0.AuditVerificationGetRequest", SchemaVersion: "0.1.0", RequestID: "req-audit-ver", ViewLimit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditVerificationGet error response: %+v", errResp)
	}
	if auditResp.Summary.IntegrityStatus == "" {
		t.Fatal("audit summary integrity_status empty")
	}

	timelineResp, errResp := s.HandleAuditTimeline(context.Background(), AuditTimelineRequest{SchemaID: "runecode.protocol.v0.AuditTimelineRequest", SchemaVersion: "0.1.0", RequestID: "req-audit-timeline", Order: "operational_seq_asc", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditTimeline error response: %+v", errResp)
	}
	if timelineResp.Order != "operational_seq_asc" {
		t.Fatalf("timeline order = %q, want operational_seq_asc", timelineResp.Order)
	}
}

func assertVersionAndLogEndpoints(t *testing.T, s *Service) {
	t.Helper()
	assertReadinessAndVersionEndpoints(t, s)
	assertLogStreamEndpoints(t, s)
}

func assertReadinessAndVersionEndpoints(t *testing.T, s *Service) {
	t.Helper()
	readyResp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{SchemaID: "runecode.protocol.v0.ReadinessGetRequest", SchemaVersion: "0.1.0", RequestID: "req-ready"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet error response: %+v", errResp)
	}
	if readyResp.Readiness.ConsumptionChannel != "broker_local_api" {
		t.Fatalf("readiness consumption_channel = %q, want broker_local_api", readyResp.Readiness.ConsumptionChannel)
	}

	versionResp, errResp := s.HandleVersionInfoGet(context.Background(), VersionInfoGetRequest{SchemaID: "runecode.protocol.v0.VersionInfoGetRequest", SchemaVersion: "0.1.0", RequestID: "req-version"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleVersionInfoGet error response: %+v", errResp)
	}
	if versionResp.VersionInfo.APIFamily != "broker_local_api" {
		t.Fatalf("api_family = %q, want broker_local_api", versionResp.VersionInfo.APIFamily)
	}
}

func assertLogStreamEndpoints(t *testing.T, s *Service) {
	t.Helper()
	logReq, errResp := s.HandleLogStreamRequest(context.Background(), LogStreamRequest{SchemaID: "runecode.protocol.v0.LogStreamRequest", SchemaVersion: "0.1.0", RequestID: "req-log-stream", StreamID: "", Follow: true, IncludeBacklog: true}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleLogStreamRequest error response: %+v", errResp)
	}
	if logReq.StreamID == "" {
		t.Fatal("log stream request ack stream_id empty")
	}

	logEvents, streamErr := s.StreamLogEvents(logReq)
	if streamErr != nil {
		t.Fatalf("StreamLogEvents returned error: %v", streamErr)
	}
	if len(logEvents) < 2 {
		t.Fatalf("log stream events = %d, want at least start+terminal", len(logEvents))
	}
	if logEvents[0].EventType != "log_stream_start" {
		t.Fatalf("first log event_type = %q, want log_stream_start", logEvents[0].EventType)
	}
	assertLogStreamSeqMonotonic(t, logEvents)
	assertSingleLogTerminalEvent(t, logEvents)
	if got := logEvents[len(logEvents)-1].TerminalStatus; got != "completed" {
		t.Fatalf("log terminal_status = %q, want completed", got)
	}
}

func assertLogStreamSeqMonotonic(t *testing.T, events []LogStreamEvent) {
	t.Helper()
	for i := 1; i < len(events); i++ {
		if events[i].Seq <= events[i-1].Seq {
			t.Fatalf("log stream seq not monotonic: prev=%d curr=%d", events[i-1].Seq, events[i].Seq)
		}
	}
}

func assertSingleLogTerminalEvent(t *testing.T, events []LogStreamEvent) {
	t.Helper()
	terminalCount := 0
	for _, event := range events {
		if event.EventType != "log_stream_terminal" {
			continue
		}
		terminalCount++
		if event.TerminalStatus == "" {
			t.Fatal("log terminal event missing in-band terminal_status")
		}
	}
	if terminalCount != 1 {
		t.Fatalf("log terminal event count = %d, want 1", terminalCount)
	}
}

func TestLogStreamEventsCarryTypedErrorOnFailedTerminal(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ack, errResp := s.HandleLogStreamRequest(context.Background(), LogStreamRequest{
		SchemaID:       "runecode.protocol.v0.LogStreamRequest",
		SchemaVersion:  "0.1.0",
		RequestID:      "req-log-fail",
		StreamID:       "",
		StartCursor:    "force_failure",
		Follow:         false,
		IncludeBacklog: true,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleLogStreamRequest error response: %+v", errResp)
	}
	events, err := s.StreamLogEvents(ack)
	if err != nil {
		t.Fatalf("StreamLogEvents returned error: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("log stream events = %d, want at least start+terminal", len(events))
	}
	terminal := events[len(events)-1]
	if terminal.EventType != "log_stream_terminal" {
		t.Fatalf("last event_type = %q, want log_stream_terminal", terminal.EventType)
	}
	if terminal.TerminalStatus != "failed" {
		t.Fatalf("terminal_status = %q, want failed", terminal.TerminalStatus)
	}
	if terminal.Error == nil {
		t.Fatal("failed terminal event missing typed error envelope")
	}
}

func signedApprovalArtifactsForBrokerTests(t *testing.T, approver string, digest string) (*trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, []trustpolicy.VerifierRecord) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	keyID := sha256.Sum256(publicKey)
	keyIDValue := hex.EncodeToString(keyID[:])
	actionHash := promotionActionHashForBrokerTests(digest, "repo/file.txt", "abc123", "tool-v1", approver)
	requestPayload := map[string]any{
		"schema_id":                trustpolicy.ApprovalRequestSchemaID,
		"schema_version":           trustpolicy.ApprovalRequestSchemaVersion,
		"approval_profile":         "moderate",
		"requester":                map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1"},
		"approval_trigger_code":    "artifact_promotion",
		"manifest_hash":            map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)},
		"action_request_hash":      map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(actionHash, "sha256:")},
		"relevant_artifact_hashes": []any{map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(digest, "sha256:")}},
		"details_schema_id":        "runecode.protocol.details.approval.excerpt-promotion.v0",
		"details":                  map[string]any{"repo_path": "repo/file.txt", "commit": "abc123"},
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"requested_at":             "2026-03-13T12:00:00Z",
		"expires_at":               "2026-03-13T12:30:00Z",
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      "Promote reviewed file excerpts for downstream use.",
		"signatures":               []any{map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": keyIDValue, "signature": "c2ln"}},
	}
	reqBytes, _ := json.Marshal(requestPayload)
	reqCanonical, _ := jsoncanonicalizer.Transform(reqBytes)
	reqSig := ed25519.Sign(privateKey, reqCanonical)
	requestEnv := &trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.ApprovalRequestSchemaID, PayloadSchemaVersion: trustpolicy.ApprovalRequestSchemaVersion, Payload: reqBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: base64.StdEncoding.EncodeToString(reqSig)}}

	requestDigest, _ := approvalIDFromRequest(*requestEnv)
	decisionPayload := map[string]any{
		"schema_id":                trustpolicy.ApprovalDecisionSchemaID,
		"schema_version":           trustpolicy.ApprovalDecisionSchemaVersion,
		"approval_request_hash":    map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(requestDigest, "sha256:")},
		"approver":                 map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": approver, "instance_id": "approval-session"},
		"decision_outcome":         "approve",
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"key_protection_posture":   "hardware_backed",
		"identity_binding_posture": "attested",
		"approval_assertion_hash":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)},
		"decided_at":               "2026-03-13T12:05:00Z",
		"consumption_posture":      "single_use",
		"signatures":               []any{map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": strings.Repeat("a", 64), "signature": "c2ln"}},
	}
	decisionBytes, _ := json.Marshal(decisionPayload)
	decisionCanonical, _ := jsoncanonicalizer.Transform(decisionBytes)
	decisionSig := ed25519.Sign(privateKey, decisionCanonical)
	decisionEnv := &trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.ApprovalDecisionSchemaID, PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion, Payload: decisionBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: base64.StdEncoding.EncodeToString(decisionSig)}}

	verifier := trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "approval_authority", LogicalScope: "user", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "user", PrincipalID: approver, InstanceID: "approval-session"}, KeyProtectionPosture: "hardware_backed", IdentityBindingPosture: "attested", PresenceMode: "hardware_touch", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}
	return requestEnv, decisionEnv, []trustpolicy.VerifierRecord{verifier}
}

func promotionActionHashForBrokerTests(digest, repoPath, commit, extractorVersion, approver string) string {
	payload, err := json.Marshal(struct {
		Approver             string `json:"approver"`
		Commit               string `json:"commit"`
		ExtractorToolVersion string `json:"extractor_tool_version"`
		RepoPath             string `json:"repo_path"`
		UnapprovedDigest     string `json:"unapproved_digest"`
	}{
		Approver:             approver,
		Commit:               commit,
		ExtractorToolVersion: extractorVersion,
		RepoPath:             repoPath,
		UnapprovedDigest:     digest,
	})
	if err != nil {
		panic(err)
	}
	canonical, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func putTrustedVerifierRecordForService(service *Service, record trustpolicy.VerifierRecord) error {
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = service.Put(artifacts.PutRequest{Payload: b, ContentType: "application/json", DataClass: artifacts.DataClassAuditVerificationReport, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "auditd", TrustedSource: true})
	return err
}
