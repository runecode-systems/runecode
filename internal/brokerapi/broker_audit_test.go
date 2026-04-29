package brokerapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestBrokerRejectionPathsAreAudited(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxMessageBytes: 2048, MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})

	_, _ = s.HandleArtifactList(context.Background(), DefaultArtifactListRequest("req-auth"), RequestContext{AdmissionErr: context.Canceled})
	_, _ = s.HandleArtifactHead(context.Background(), ArtifactHeadRequest{SchemaID: "runecode.protocol.v0.BrokerArtifactHeadRequest", SchemaVersion: "0.1.0", RequestID: "req-schema", Digest: "not-a-digest"}, RequestContext{})
	oversized := DefaultArtifactPutRequest("req-size", []byte(strings.Repeat("a", 4000)), "text/plain", "spec_text", "sha256:"+strings.Repeat("1", 64), "workspace", "run-1", "step-1")
	_, _ = s.HandleArtifactPut(context.Background(), oversized, RequestContext{})
	release, err := s.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	_, _ = s.HandleArtifactList(context.Background(), DefaultArtifactListRequest("req-limit"), RequestContext{ClientID: "client-a", LaneID: "lane-a"})
	release()

	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}

	assertBrokerRejectionAuditEvent(t, events, "req-auth", "broker_api_auth_admission_denied")
	assertBrokerRejectionAuditEvent(t, events, "req-schema", "broker_validation_schema_invalid")
	assertBrokerRejectionAuditEvent(t, events, "req-size", "broker_limit_message_size_exceeded")
	assertBrokerRejectionAuditEvent(t, events, "req-limit", "broker_limit_in_flight_exceeded")
}

func TestBrokerApprovalResolveAuditsResolution(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := brokerAuditResolveRequest(approvalID, policyDecisionHash, unapproved.Digest, requestEnv, decisionEnv)
	if _, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{}); errResp != nil {
		t.Fatalf("HandleApprovalResolve returned error response: %+v", errResp)
	}
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	if !hasBrokerApprovalResolutionEvent(events, "req-approval-audit", approvalID, "consumed") {
		t.Fatal("expected broker_approval_resolution audit event for approval resolve")
	}
}

func brokerAuditResolveRequest(approvalID, policyDecisionHash, digest string, requestEnv, decisionEnv *trustpolicy.SignedObjectEnvelope) ApprovalResolveRequest {
	return ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-audit", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
}

func hasBrokerApprovalResolutionEvent(events []artifacts.AuditEvent, requestID, approvalID, status string) bool {
	for _, event := range events {
		if event.Type != brokerAuditEventTypeApprovalResolution {
			continue
		}
		if event.Details["request_id"] != requestID || event.Details["approval_id"] != approvalID {
			continue
		}
		return event.Details["approval_status"] == status
	}
	return false
}

func TestBrokerRejectionFailsClosedWhenAuditPersistFails(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	if err := s.store.AppendTrustedAuditEvent("prime", "brokerapi", map[string]interface{}{"x": "y"}); err != nil {
		t.Fatalf("prime audit append returned error: %v", err)
	}
	if err := s.store.SetPolicy(artifacts.Policy{}); err == nil {
		// keep state touched to ensure store writable before forcing failure path
	}

	if s.store != nil {
		s.store = nil
	}
	errResp := s.makeError("req-audit-fail", "broker_limit_in_flight_exceeded", "transport", true, "limit")
	if errResp.Error.Code != "gateway_failure" {
		t.Fatalf("error code = %q, want gateway_failure", errResp.Error.Code)
	}
}

func TestShouldAuditErrorCodeIncludesPolicyRejected(t *testing.T) {
	if !shouldAuditErrorCode("broker_limit_policy_rejected") {
		t.Fatal("broker_limit_policy_rejected must be auditable")
	}
}

func TestRecordRuntimeFactsEmitsBrokerOwnedLauncherRuntimeAuditEvents(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_ = putRunScopedArtifactForLocalOpsTest(t, s, "run-runtime-audit", "step-1")
	facts := launcherRuntimeFactsFixture()
	facts.LaunchReceipt.LaunchFailureReasonCode = ""
	if err := s.RecordRuntimeFacts("run-runtime-audit", facts); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	_, evidence, _, _, ok := s.store.RuntimeEvidenceState("run-runtime-audit")
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime evidence")
	}
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	assertLauncherRuntimeAuditEvent(t, events, "runtime_launch_admission", evidence.Launch.EvidenceDigest, evidence.Hardening.EvidenceDigest, evidence.Session.EvidenceDigest)
	assertLauncherRuntimeAuditEvent(t, events, "isolate_session_started", evidence.Launch.EvidenceDigest, evidence.Hardening.EvidenceDigest, evidence.Session.EvidenceDigest)
	assertLauncherRuntimeAuditEvent(t, events, "isolate_session_bound", evidence.Launch.EvidenceDigest, evidence.Hardening.EvidenceDigest, evidence.Session.EvidenceDigest)
}

func TestRuntimeLaunchAdmissionAuditPayloadOmitsEmptyToolchainDigest(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_ = putRunScopedArtifactForLocalOpsTest(t, s, "run-runtime-admission-no-toolchain", "step-1")
	facts := launcherRuntimeFactsFixture()
	facts.LaunchReceipt.RunID = "run-runtime-admission-no-toolchain"
	facts.LaunchReceipt.LaunchFailureReasonCode = ""
	facts.LaunchReceipt.RuntimeToolchainDescriptorDigest = ""
	if err := s.RecordRuntimeFacts("run-runtime-admission-no-toolchain", facts); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	event := findLauncherRuntimeAuditEvent(t, events, "runtime_launch_admission")
	payload := launcherRuntimeAuditEventPayload(t, event)
	if _, ok := payload["runtime_toolchain_descriptor_digest"]; ok {
		t.Fatal("runtime_toolchain_descriptor_digest present in launch admission payload, want omitted when empty")
	}
}

func TestRecordRuntimeFactsEmitsLaunchDeniedAuditWithoutSessionBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	facts := launcherRuntimeFactsFixture()
	facts.LaunchReceipt.RunID = "run-runtime-denied-pre-session"
	facts.LaunchReceipt.StageID = "stage-runtime-denied"
	facts.LaunchReceipt.SessionID = ""
	facts.LaunchReceipt.SessionNonce = ""
	facts.LaunchReceipt.HandshakeTranscriptHash = ""
	facts.LaunchReceipt.IsolateSessionKeyIDValue = ""
	facts.LaunchReceipt.LaunchContextDigest = ""
	facts.LaunchReceipt.IsolateID = ""
	if err := s.RecordRuntimeFacts("run-runtime-denied-pre-session", facts); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	_, evidence, _, _, ok := s.store.RuntimeEvidenceState("run-runtime-denied-pre-session")
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime evidence")
	}
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	assertLauncherRuntimeAuditEvent(t, events, "runtime_launch_denied", evidence.Launch.EvidenceDigest, evidence.Hardening.EvidenceDigest, "")
}

func TestRuntimeFactsAuditEventsAreNotReemittedForSameEvidenceDigest(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_ = putRunScopedArtifactForLocalOpsTest(t, s, "run-runtime-dedupe", "step-1")
	facts := launcherRuntimeFactsFixture()
	if err := s.RecordRuntimeFacts("run-runtime-dedupe", facts); err != nil {
		t.Fatalf("first RecordRuntimeFacts returned error: %v", err)
	}
	before, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents(before) returned error: %v", err)
	}
	if err := s.RecordRuntimeFacts("run-runtime-dedupe", facts); err != nil {
		t.Fatalf("second RecordRuntimeFacts returned error: %v", err)
	}
	after, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents(after) returned error: %v", err)
	}
	if countLauncherRuntimeAuditEvents(after) != countLauncherRuntimeAuditEvents(before) {
		t.Fatalf("launcher runtime event count changed after duplicate facts: before=%d after=%d", countLauncherRuntimeAuditEvents(before), countLauncherRuntimeAuditEvents(after))
	}
}

func TestBuildRuntimeEventOperationIDForDeniedPreSessionUsesLaunchEvidenceDigest(t *testing.T) {
	evidenceA := launcherbackend.RuntimeEvidenceSnapshot{Launch: launcherbackend.LaunchRuntimeEvidence{RunID: "run-denied-pre-session", EvidenceDigest: "sha256:" + strings.Repeat("a", 64)}}
	evidenceB := launcherbackend.RuntimeEvidenceSnapshot{Launch: launcherbackend.LaunchRuntimeEvidence{RunID: "run-denied-pre-session", EvidenceDigest: "sha256:" + strings.Repeat("b", 64)}}

	opIDa := buildRuntimeEventOperationID("runtime_launch_denied", evidenceA)
	opIDb := buildRuntimeEventOperationID("runtime_launch_denied", evidenceB)

	if opIDa == opIDb {
		t.Fatalf("operation IDs must differ for unique launch evidence digests, got identical %q", opIDa)
	}
	if !strings.HasPrefix(opIDa, "runtime-launch-denied:") {
		t.Fatalf("operation_id = %q, want runtime-launch-denied prefix", opIDa)
	}
	if !strings.Contains(opIDa, evidenceA.Launch.EvidenceDigest) {
		t.Fatalf("operation_id = %q, want launch evidence digest %q", opIDa, evidenceA.Launch.EvidenceDigest)
	}
	if !strings.Contains(opIDb, evidenceB.Launch.EvidenceDigest) {
		t.Fatalf("operation_id = %q, want launch evidence digest %q", opIDb, evidenceB.Launch.EvidenceDigest)
	}
}

func TestBuildRuntimeEventOperationIDForLaunchAdmissionStaysLaunchDigestScoped(t *testing.T) {
	evidence := launcherbackend.RuntimeEvidenceSnapshot{
		Launch:    launcherbackend.LaunchRuntimeEvidence{RunID: "run-launch-admission", SessionID: " session-1 ", EvidenceDigest: "sha256:" + strings.Repeat("c", 64)},
		Hardening: launcherbackend.HardeningRuntimeEvidence{EvidenceDigest: "sha256:" + strings.Repeat("d", 64)},
		Session:   &launcherbackend.SessionRuntimeEvidence{EvidenceDigest: "sha256:" + strings.Repeat("e", 64)},
	}
	opID := buildRuntimeEventOperationID("runtime_launch_admission", evidence)
	if opID != "runtime-launch-admission:"+evidence.Launch.EvidenceDigest {
		t.Fatalf("operation_id = %q, want launch-digest scoped id", opID)
	}
}

func TestRuntimeSessionAuditIdentityKeyIncludesLaunchAndHardeningDigests(t *testing.T) {
	evidence := launcherbackend.RuntimeEvidenceSnapshot{
		Launch:    launcherbackend.LaunchRuntimeEvidence{EvidenceDigest: "sha256:" + strings.Repeat("1", 64)},
		Hardening: launcherbackend.HardeningRuntimeEvidence{EvidenceDigest: "sha256:" + strings.Repeat("2", 64)},
		Session:   &launcherbackend.SessionRuntimeEvidence{EvidenceDigest: "sha256:" + strings.Repeat("3", 64)},
	}
	key := runtimeSessionAuditIdentityKey(evidence)
	parts := strings.Split(key, ":")
	if len(parts) != 6 {
		t.Fatalf("session audit identity parts = %d, want 6 for three sha256 digests", len(parts))
	}
	if !strings.Contains(key, evidence.Launch.EvidenceDigest) || !strings.Contains(key, evidence.Hardening.EvidenceDigest) || !strings.Contains(key, evidence.Session.EvidenceDigest) {
		t.Fatalf("session audit identity = %q, want launch, hardening, and session digests included", key)
	}
}

func assertLauncherRuntimeAuditEvent(t *testing.T, events []artifacts.AuditEvent, runtimeEventType, launchDigest, hardeningDigest, sessionDigest string) {
	t.Helper()
	event := findLauncherRuntimeAuditEvent(t, events, runtimeEventType)
	assertLauncherRuntimeAuditDigests(t, event, launchDigest, hardeningDigest, sessionDigest)
}

func findLauncherRuntimeAuditEvent(t *testing.T, events []artifacts.AuditEvent, runtimeEventType string) artifacts.AuditEvent {
	t.Helper()
	for _, event := range events {
		if event.Type != brokerAuditEventTypeLauncherRuntime {
			continue
		}
		if event.Details["runtime_event_type"] != runtimeEventType {
			continue
		}
		return event
	}
	t.Fatalf("missing %s launcher runtime audit event", runtimeEventType)
	return artifacts.AuditEvent{}
}

func launcherRuntimeAuditEventPayload(t *testing.T, event artifacts.AuditEvent) map[string]any {
	t.Helper()
	rawPayload, ok := event.Details["event_payload"]
	if !ok {
		t.Fatal("event_payload missing from launcher runtime audit details")
	}
	switch payload := rawPayload.(type) {
	case map[string]any:
		return payload
	case json.RawMessage:
		var decoded map[string]any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			t.Fatalf("json.Unmarshal(event_payload RawMessage) returned error: %v", err)
		}
		return decoded
	case []byte:
		var decoded map[string]any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			t.Fatalf("json.Unmarshal(event_payload bytes) returned error: %v", err)
		}
		return decoded
	case string:
		var decoded map[string]any
		if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
			t.Fatalf("json.Unmarshal(event_payload string) returned error: %v", err)
		}
		return decoded
	default:
		t.Fatalf("event_payload = %T, want object or JSON payload", rawPayload)
		return nil
	}
}

func assertLauncherRuntimeAuditDigests(t *testing.T, event artifacts.AuditEvent, launchDigest, hardeningDigest, sessionDigest string) {
	t.Helper()
	digests := launcherRuntimeAuditDigests(t, event)
	assertLauncherRuntimeAuditDigestValue(t, digests, "launch_receipt", launchDigest)
	assertLauncherRuntimeAuditDigestValue(t, digests, "hardening_posture", hardeningDigest)
	assertLauncherRuntimeAuditDigestValue(t, digests, "session_binding", sessionDigest)
}

func launcherRuntimeAuditDigests(t *testing.T, event artifacts.AuditEvent) map[string]any {
	t.Helper()
	digests, ok := event.Details["stored_runtime_fact_digests"].(map[string]any)
	if !ok {
		t.Fatalf("stored_runtime_fact_digests = %T, want map", event.Details["stored_runtime_fact_digests"])
	}
	return digests
}

func assertLauncherRuntimeAuditDigestValue(t *testing.T, digests map[string]any, name, want string) {
	t.Helper()
	if digests[name] != want {
		t.Fatalf("%s digest = %v, want %q", name, digests[name], want)
	}
}

func countLauncherRuntimeAuditEvents(events []artifacts.AuditEvent) int {
	count := 0
	for _, event := range events {
		if event.Type == brokerAuditEventTypeLauncherRuntime {
			count++
		}
	}
	return count
}

func assertBrokerRejectionAuditEvent(t *testing.T, events []artifacts.AuditEvent, requestID, reasonCode string) {
	t.Helper()
	for _, event := range events {
		if event.Type != brokerAuditEventTypeRejection {
			continue
		}
		if event.Details["request_id"] != requestID {
			continue
		}
		if event.Details["reason_code"] != reasonCode {
			t.Fatalf("reason_code = %v, want %s", event.Details["reason_code"], reasonCode)
		}
		return
	}
	t.Fatalf("missing broker rejection audit event for request_id=%s reason_code=%s", requestID, reasonCode)
}
