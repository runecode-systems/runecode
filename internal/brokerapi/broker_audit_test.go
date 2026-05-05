package brokerapi

import (
	"context"
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
	facts := launcherRuntimeFactsFixtureForRun("run-runtime-audit")
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
	facts := launcherRuntimeFactsFixtureForRun("run-runtime-dedupe")
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

func TestRuntimeLaunchDeniedAuditDedupeIncludesReasonCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-runtime-denied-dedupe-reason", "step-1")

	facts := launcherRuntimeFactsFixture()
	facts.LaunchReceipt.RunID = "run-runtime-denied-dedupe-reason"
	facts.LaunchReceipt.LaunchFailureReasonCode = launcherbackend.BackendErrorCodeAccelerationUnavailable
	if err := s.RecordRuntimeFacts("run-runtime-denied-dedupe-reason", facts); err != nil {
		t.Fatalf("first RecordRuntimeFacts returned error: %v", err)
	}

	facts.LaunchReceipt.LaunchFailureReasonCode = launcherbackend.BackendErrorCodeHypervisorLaunchFailed
	if err := s.RecordRuntimeFacts("run-runtime-denied-dedupe-reason", facts); err != nil {
		t.Fatalf("second RecordRuntimeFacts returned error: %v", err)
	}

	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}

	if countRuntimeLaunchDeniedEventsByReasonCode(t, events, launcherbackend.BackendErrorCodeAccelerationUnavailable) != 1 {
		t.Fatalf("expected exactly one runtime_launch_denied event with reason %q", launcherbackend.BackendErrorCodeAccelerationUnavailable)
	}
	if countRuntimeLaunchDeniedEventsByReasonCode(t, events, launcherbackend.BackendErrorCodeHypervisorLaunchFailed) != 1 {
		t.Fatalf("expected exactly one runtime_launch_denied event with reason %q", launcherbackend.BackendErrorCodeHypervisorLaunchFailed)
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
		Launch:                  launcherbackend.LaunchRuntimeEvidence{EvidenceDigest: "sha256:" + strings.Repeat("1", 64)},
		Hardening:               launcherbackend.HardeningRuntimeEvidence{EvidenceDigest: "sha256:" + strings.Repeat("2", 64)},
		Session:                 &launcherbackend.SessionRuntimeEvidence{EvidenceDigest: "sha256:" + strings.Repeat("3", 64)},
		Attestation:             &launcherbackend.IsolateAttestationEvidence{EvidenceDigest: "sha256:" + strings.Repeat("4", 64)},
		AttestationVerification: &launcherbackend.IsolateAttestationVerificationRecord{VerificationDigest: "sha256:" + strings.Repeat("5", 64), VerificationResult: launcherbackend.AttestationVerificationResultValid},
	}
	key := runtimeSessionAuditIdentityKey(evidence)
	parts := strings.Split(key, ":")
	if len(parts) != 11 {
		t.Fatalf("session audit identity parts = %d, want 11 for five sha256 digests plus posture marker", len(parts))
	}
	if !strings.Contains(key, evidence.Launch.EvidenceDigest) || !strings.Contains(key, evidence.Hardening.EvidenceDigest) || !strings.Contains(key, evidence.Session.EvidenceDigest) || !strings.Contains(key, evidence.Attestation.EvidenceDigest) || !strings.Contains(key, evidence.AttestationVerification.VerificationDigest) || !strings.Contains(key, launcherbackend.AttestationPostureUnknown) {
		t.Fatalf("session audit identity = %q, want launch/hardening/session/attestation digests and posture included", key)
	}
}

func TestRuntimeSessionAuditPayloadIncludesAttestationEvidenceDigestAdditively(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_ = putRunScopedArtifactForLocalOpsTest(t, s, "run-runtime-attestation-audit", "step-1")
	facts := attestationAuditRuntimeFacts()

	if err := s.RecordRuntimeFacts("run-runtime-attestation-audit", facts); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	evidence := requirePersistedAttestationEvidence(t, s, "run-runtime-attestation-audit")

	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	startedEvent := findLauncherRuntimeAuditEvent(t, events, "isolate_session_started")
	assertRuntimeAttestationAuditPayload(t, startedEvent, evidence.Attestation.EvidenceDigest)
	boundEvent := findLauncherRuntimeAuditEvent(t, events, "isolate_session_bound")
	assertRuntimeAttestationAuditPayload(t, boundEvent, evidence.Attestation.EvidenceDigest)
	assertLauncherRuntimeAuditDigestValue(t, launcherRuntimeAuditDigests(t, startedEvent), "attestation_evidence", evidence.Attestation.EvidenceDigest)
}

func TestRuntimeSessionBoundAuditTracksPersistedAttestationStateTransitions(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-runtime-attestation-transition"
	_ = putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")

	recordRuntimeFactsForAuditTransition(t, s, runID, preAttestationRuntimeFacts(runID), "pre-attestation")
	assertBoundAuditBeforePersistedAttestation(t, s)

	attestedFacts := attestedRuntimeFacts(runID)
	recordRuntimeFactsForAuditTransition(t, s, runID, attestedFacts, "attested")
	assertBoundAuditAfterPersistedAttestation(t, s, runID)

	recordRuntimeFactsForAuditTransition(t, s, runID, attestedFacts, "attested repeat")
	assertBoundAuditEventCount(t, s, 2, "after repeat")
}

func preAttestationRuntimeFacts(runID string) launcherbackend.RuntimeFactsSnapshot {
	facts := launcherRuntimeFactsFixture()
	facts.LaunchReceipt.RunID = runID
	facts.LaunchReceipt.LaunchFailureReasonCode = ""
	facts.LaunchReceipt.AttestationEvidenceSourceKind = launcherbackend.AttestationSourceKindUnknown
	facts.LaunchReceipt.AttestationMeasurementProfile = ""
	facts.LaunchReceipt.AttestationFreshnessMaterial = nil
	facts.LaunchReceipt.AttestationFreshnessBindingClaims = nil
	facts.LaunchReceipt.AttestationEvidenceClaimsDigest = ""
	facts.LaunchReceipt.AttestationVerifierPolicyID = ""
	facts.LaunchReceipt.AttestationVerifierPolicyDigest = ""
	facts.LaunchReceipt.AttestationVerificationRulesVersion = ""
	facts.LaunchReceipt.AttestationVerificationTimestamp = ""
	facts.LaunchReceipt.AttestationVerificationResult = ""
	facts.LaunchReceipt.AttestationVerificationReasonCodes = nil
	facts.LaunchReceipt.AttestationReplayVerdict = ""
	facts.PostHandshakeAttestationInput = nil
	return facts
}

func attestedRuntimeFacts(runID string) launcherbackend.RuntimeFactsSnapshot {
	facts := launcherRuntimeFactsFixture()
	facts.LaunchReceipt.RunID = runID
	facts.LaunchReceipt.LaunchFailureReasonCode = ""
	facts.LaunchReceipt.AttestationVerifierPolicyID = "runtime_asset_admission_identity"
	facts.LaunchReceipt.AttestationVerifierPolicyDigest = facts.LaunchReceipt.AuthorityStateDigest
	facts.LaunchReceipt.AttestationVerificationResult = launcherbackend.AttestationVerificationResultValid
	facts.LaunchReceipt.AttestationReplayVerdict = launcherbackend.AttestationReplayVerdictOriginal
	facts.PostHandshakeAttestationInput = runtimeFactsPostHandshakeAttestationInput(facts.LaunchReceipt)
	facts.PostHandshakeAttestationInput.VerifierPolicyID = facts.LaunchReceipt.AttestationVerifierPolicyID
	facts.PostHandshakeAttestationInput.VerifierPolicyDigest = facts.LaunchReceipt.AttestationVerifierPolicyDigest
	facts.PostHandshakeAttestationInput.VerificationResult = facts.LaunchReceipt.AttestationVerificationResult
	facts.PostHandshakeAttestationInput.ReplayVerdict = facts.LaunchReceipt.AttestationReplayVerdict
	return facts
}

func recordRuntimeFactsForAuditTransition(t *testing.T, s *Service, runID string, facts launcherbackend.RuntimeFactsSnapshot, label string) {
	t.Helper()
	if err := s.RecordRuntimeFacts(runID, facts); err != nil {
		t.Fatalf("RecordRuntimeFacts(%s) returned error: %v", label, err)
	}
}

func assertBoundAuditBeforePersistedAttestation(t *testing.T, s *Service) {
	t.Helper()
	boundEvents := requireBoundAuditEvents(t, s, 1, "before persisted attestation evidence")
	firstPayload := launcherRuntimeAuditEventPayload(t, boundEvents[0])
	if _, ok := firstPayload["attestation_evidence_digest"]; ok {
		t.Fatalf("first isolate_session_bound payload attestation_evidence_digest = %v, want omitted before persisted attestation", firstPayload["attestation_evidence_digest"])
	}
}

func assertBoundAuditAfterPersistedAttestation(t *testing.T, s *Service, runID string) {
	t.Helper()
	boundEvents := requireBoundAuditEvents(t, s, 2, "after attestation transition")
	latestBound := boundEvents[len(boundEvents)-1]
	latestPayload := launcherRuntimeAuditEventPayload(t, latestBound)
	persistedEvidence := requirePersistedAttestationVerificationEvidence(t, s, runID)
	if latestPayload["attestation_evidence_digest"] != persistedEvidence.Attestation.EvidenceDigest {
		t.Fatalf("latest isolate_session_bound payload attestation_evidence_digest = %v, want %q", latestPayload["attestation_evidence_digest"], persistedEvidence.Attestation.EvidenceDigest)
	}
	if latestBound.Details["attestation_posture"] != launcherbackend.AttestationPostureValid {
		t.Fatalf("latest isolate_session_bound details attestation_posture = %v, want %q", latestBound.Details["attestation_posture"], launcherbackend.AttestationPostureValid)
	}
}

func assertBoundAuditEventCount(t *testing.T, s *Service, want int, label string) {
	t.Helper()
	requireBoundAuditEvents(t, s, want, label)
}

func requireBoundAuditEvents(t *testing.T, s *Service, want int, label string) []artifacts.AuditEvent {
	t.Helper()
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents(%s) returned error: %v", label, err)
	}
	boundEvents := launcherRuntimeAuditEventsByRuntimeType(events, "isolate_session_bound")
	if len(boundEvents) != want {
		t.Fatalf("isolate_session_bound event count %s = %d, want %d", label, len(boundEvents), want)
	}
	return boundEvents
}

func requirePersistedAttestationVerificationEvidence(t *testing.T, s *Service, runID string) launcherbackend.RuntimeEvidenceSnapshot {
	t.Helper()
	_, persistedEvidence, _, _, ok := s.store.RuntimeEvidenceState(runID)
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime evidence")
	}
	if persistedEvidence.Attestation == nil || persistedEvidence.AttestationVerification == nil {
		t.Fatalf("persisted evidence missing attestation/verification after attested facts: %#v", persistedEvidence)
	}
	return persistedEvidence
}

func attestationAuditRuntimeFacts() launcherbackend.RuntimeFactsSnapshot {
	facts := launcherRuntimeFactsFixture()
	facts.LaunchReceipt.RunID = "run-runtime-attestation-audit"
	facts.LaunchReceipt.AttestationEvidenceSourceKind = launcherbackend.AttestationSourceKindTPMQuote
	facts.LaunchReceipt.AttestationMeasurementProfile = "microvm-boot-v1"
	facts.LaunchReceipt.AttestationFreshnessMaterial = []string{"quote_nonce"}
	facts.LaunchReceipt.AttestationFreshnessBindingClaims = []string{"session_nonce", "handshake_transcript_hash"}
	facts.LaunchReceipt.AttestationEvidenceClaimsDigest = runtimeFactsMeasurementDigests(facts.LaunchReceipt)[0]
	facts.LaunchReceipt.AttestationVerifierPolicyID = "runtime_asset_admission_identity"
	facts.LaunchReceipt.AttestationVerifierPolicyDigest = facts.LaunchReceipt.AuthorityStateDigest
	facts.LaunchReceipt.AttestationVerificationResult = launcherbackend.AttestationVerificationResultValid
	facts.LaunchReceipt.AttestationReplayVerdict = launcherbackend.AttestationReplayVerdictOriginal
	facts.PostHandshakeAttestationInput = runtimeFactsPostHandshakeAttestationInput(facts.LaunchReceipt)
	facts.PostHandshakeAttestationInput.VerifierPolicyID = facts.LaunchReceipt.AttestationVerifierPolicyID
	facts.PostHandshakeAttestationInput.VerifierPolicyDigest = facts.LaunchReceipt.AttestationVerifierPolicyDigest
	facts.PostHandshakeAttestationInput.VerificationResult = facts.LaunchReceipt.AttestationVerificationResult
	facts.PostHandshakeAttestationInput.ReplayVerdict = facts.LaunchReceipt.AttestationReplayVerdict
	return facts
}

func requirePersistedAttestationEvidence(t *testing.T, s *Service, runID string) launcherbackend.RuntimeEvidenceSnapshot {
	t.Helper()
	_, evidence, _, _, ok := s.store.RuntimeEvidenceState(runID)
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime evidence")
	}
	if evidence.Attestation == nil || evidence.Attestation.EvidenceDigest == "" {
		t.Fatalf("attestation evidence missing from persisted runtime evidence: %#v", evidence.Attestation)
	}
	return evidence
}

func assertRuntimeAttestationAuditPayload(t *testing.T, event artifacts.AuditEvent, expectedDigest string) {
	t.Helper()
	payload := launcherRuntimeAuditEventPayload(t, event)
	if payload["attestation_evidence_digest"] != expectedDigest {
		t.Fatalf("attestation_evidence_digest = %v, want %q", payload["attestation_evidence_digest"], expectedDigest)
	}
	if event.Details["provisioning_posture"] != launcherbackend.ProvisioningPostureAttested {
		t.Fatalf("details provisioning_posture = %v, want %q", event.Details["provisioning_posture"], launcherbackend.ProvisioningPostureAttested)
	}
	if event.Details["attestation_posture"] != launcherbackend.AttestationPostureValid {
		t.Fatalf("details attestation_posture = %v, want %q", event.Details["attestation_posture"], launcherbackend.AttestationPostureValid)
	}
	if event.Details["attestation_verifier_class"] != launcherbackend.AttestationVerifierClassHardwareRooted {
		t.Fatalf("details attestation_verifier_class = %v, want %q", event.Details["attestation_verifier_class"], launcherbackend.AttestationVerifierClassHardwareRooted)
	}
	if event.Details["supported_runtime_requirements_satisfied"] != true {
		t.Fatalf("details supported_runtime_requirements_satisfied = %v, want true", event.Details["supported_runtime_requirements_satisfied"])
	}
}

func assertLauncherRuntimeAuditEvent(t *testing.T, events []artifacts.AuditEvent, runtimeEventType, launchDigest, hardeningDigest, sessionDigest string) {
	t.Helper()
	event := findLauncherRuntimeAuditEvent(t, events, runtimeEventType)
	assertLauncherRuntimeAuditDigests(t, event, launchDigest, hardeningDigest, sessionDigest)
}
