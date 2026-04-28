package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestAuditReadinessAndVerificationCommands(t *testing.T) {
	root := setBrokerServiceForTest(t)
	if err := seedLedgerForBrokerCommandTest(filepath.Join(root, "audit-ledger")); err != nil {
		t.Fatalf("seedLedgerForBrokerCommandTest returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if err := run([]string{"audit-readiness"}, stdout, stderr); err != nil {
		t.Fatalf("audit-readiness returned error: %v", err)
	}
	readiness := trustpolicy.AuditdReadiness{}
	if err := json.Unmarshal(stdout.Bytes(), &readiness); err != nil {
		t.Fatalf("audit-readiness output parse error: %v", err)
	}
	if !readiness.Ready {
		t.Fatal("readiness.ready = false, want true")
	}

	stdout.Reset()
	if err := run([]string{"audit-verification", "--limit", "5"}, stdout, stderr); err != nil {
		t.Fatalf("audit-verification returned error: %v", err)
	}
	surface := brokerapi.AuditVerificationSurface{}
	if err := json.Unmarshal(stdout.Bytes(), &surface); err != nil {
		t.Fatalf("audit-verification output parse error: %v", err)
	}
	if len(surface.Views) == 0 {
		t.Fatal("audit-verification views empty, want default operational view entries")
	}

	stdout.Reset()
	if err := run([]string{"audit-finalize-verify"}, stdout, stderr); err != nil {
		t.Fatalf("audit-finalize-verify returned error: %v", err)
	}
	finalize := brokerapi.AuditFinalizeVerifyResponse{}
	if err := json.Unmarshal(stdout.Bytes(), &finalize); err != nil {
		t.Fatalf("audit-finalize-verify output parse error: %v", err)
	}
	if finalize.ActionStatus != "ok" {
		t.Fatalf("audit-finalize-verify action_status = %q, want ok", finalize.ActionStatus)
	}
}

func TestAuditRecordGetCommand(t *testing.T) {
	root := setBrokerServiceForTest(t)
	if err := seedLedgerForBrokerCommandTest(filepath.Join(root, "audit-ledger")); err != nil {
		t.Fatalf("seedLedgerForBrokerCommandTest returned error: %v", err)
	}
	service, err := brokerServiceFactory(defaultBrokerServiceRoots())
	if err != nil {
		t.Fatalf("brokerServiceFactory returned error: %v", err)
	}
	surface, err := service.LatestAuditVerificationSurface(1)
	if err != nil {
		t.Fatalf("LatestAuditVerificationSurface returned error: %v", err)
	}
	if len(surface.Views) == 0 {
		t.Fatal("seed surface views empty, want at least one record")
	}
	digestID, err := surface.Views[0].RecordDigest.Identity()
	if err != nil {
		t.Fatalf("record digest identity error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"audit-record-get", "--record-digest", digestID}, stdout, stderr); err != nil {
		t.Fatalf("audit-record-get returned error: %v", err)
	}
	record := brokerapi.AuditRecordDetail{}
	if err := json.Unmarshal(stdout.Bytes(), &record); err != nil {
		t.Fatalf("audit-record-get output parse error: %v", err)
	}
	if record.RecordFamily == "" || record.Summary == "" || record.OccurredAt == "" {
		t.Fatalf("audit-record-get record missing core fields: %+v", record)
	}
	if len(record.LinkedReferences) == 0 {
		t.Fatal("audit-record-get linked_references empty, want projected links")
	}
}

func TestPromoteExcerptRequiresSignedApprovalInputs(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	err := run([]string{"promote-excerpt", "--unapproved-digest", unapproved.Digest, "--approver", "human", "--repo-path", "repo/file.txt", "--commit", "abc123", "--extractor-version", "tool-v1", "--full-content-visible"}, stdout, stderr)
	if err == nil {
		t.Fatal("promote-excerpt expected usage error when signed approval inputs are missing")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestPromoteExcerptRejectsSelfProvidedVerifierRecord(t *testing.T) {
	root := setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	approvalRequestPath, approvalEnvelopePath, _ := writeApprovalFixtures(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	_, _, verifierRecords := signedApprovalArtifactsForCLITests(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	for index := range verifierRecords {
		payload, err := json.Marshal(verifierRecords[index])
		if err != nil {
			t.Fatalf("Marshal verifier error: %v", err)
		}
		payloadPath := writeTempFile(t, "verifier-non-auditd.json", string(payload))
		nibble := string('a' + rune(index%6))
		err = run([]string{"--state-root", root, "put-artifact", "--file", payloadPath, "--content-type", "application/json", "--data-class", "audit_verification_report", "--provenance-hash", testDigest(nibble), "--role", "workspace"}, stdout, stderr)
		if err != nil {
			t.Fatalf("put-artifact verifier record returned error: %v", err)
		}
	}
	err := run([]string{"promote-excerpt", "--unapproved-digest", unapproved.Digest, "--approver", "human", "--approval-request", approvalRequestPath, "--approval-envelope", approvalEnvelopePath, "--repo-path", "repo/file.txt", "--commit", "abc123", "--extractor-version", "tool-v1", "--full-content-visible"}, stdout, stderr)
	if err == nil {
		t.Fatal("promote-excerpt expected error when verifier records are not auditd-owned")
	}
}

func TestImportTrustedContractAllowsPromotionWorkflow(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	approvalRequestPath, approvalEnvelopePath, verifierRecords := writeApprovalFixtures(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	for index := range verifierRecords {
		payload, err := json.Marshal(verifierRecords[index])
		if err != nil {
			t.Fatalf("Marshal verifier record error: %v", err)
		}
		verifierPath := filepath.Join(t.TempDir(), "verifier-record.json")
		if err := os.WriteFile(verifierPath, payload, 0o600); err != nil {
			t.Fatalf("WriteFile verifier record error: %v", err)
		}
		evidencePath := writeTrustedImportEvidenceFixture(t, "verifier-record")
		if err := run([]string{"import-trusted-contract", "--kind", "verifier-record", "--file", verifierPath, "--evidence", evidencePath}, stdout, stderr); err != nil {
			t.Fatalf("import-trusted-contract returned error: %v", err)
		}
	}
	approved := promoteViaCLI(t, stdout, stderr, unapproved.Digest, approvalRequestPath, approvalEnvelopePath)
	if approved.DataClass != "approved_file_excerpts" {
		t.Fatalf("approved data_class = %q, want approved_file_excerpts", approved.DataClass)
	}
}

func TestImportTrustedContractRejectsUnsupportedKind(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"import-trusted-contract", "--kind", "unknown", "--file", writeTempFile(t, "noop.json", "{}"), "--evidence", writeTrustedImportEvidenceFixture(t, "unknown")}, stdout, stderr)
	if err == nil {
		t.Fatal("import-trusted-contract expected usage error for unsupported kind")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestImportTrustedContractRequiresEvidence(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"import-trusted-contract", "--kind", "verifier-record", "--file", writeTempFile(t, "verifier.json", "{}")}, stdout, stderr)
	if err == nil {
		t.Fatal("import-trusted-contract expected usage error when --evidence is missing")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestBackendPostureCommandsAndGenericApprovalResolveViaLocalRPC(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	requestedOps := make([]string, 0, 6)
	originalDispatch := localRPCDispatch
	localRPCDispatch = backendPostureLocalRPCDispatchForTest(t, &requestedOps)
	t.Cleanup(func() { localRPCDispatch = originalDispatch })

	runBackendPostureCommandSequence(t, stdout, stderr)
	assertBackendPostureRequestedOps(t, requestedOps)
}

func TestGenericApprovalResolveRequestRejectsUnsupportedActionKind(t *testing.T) {
	_, err := genericApprovalResolveRequest(
		"sha256:"+strings.Repeat("a", 64),
		brokerapi.ApprovalBoundScope{ActionKind: "__unsupported_test_kind__"},
		brokerapi.ApprovalGetResponse{},
		trustpolicy.SignedObjectEnvelope{},
		trustpolicy.SignedObjectEnvelope{},
	)
	if err == nil {
		t.Fatal("expected usage error for unsupported action kind")
	}
	usage, ok := err.(*usageError)
	if !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
	if got := usage.Error(); !strings.Contains(got, "approval-resolve does not support this action kind") {
		t.Fatalf("usage error = %q", got)
	}
}

func backendPostureLocalRPCDispatchForTest(t *testing.T, requestedOps *[]string) func(*brokerapi.Service, context.Context, localRPCRequest, brokerapi.RequestContext) localRPCResponse {
	t.Helper()
	return func(_ *brokerapi.Service, _ context.Context, wire localRPCRequest, _ brokerapi.RequestContext) localRPCResponse {
		*requestedOps = append(*requestedOps, wire.Operation)
		switch wire.Operation {
		case "backend_posture_get":
			return mustOKLocalRPCResponse(t, testBackendPostureGetResponse())
		case "backend_posture_change":
			return mustOKLocalRPCResponse(t, testBackendPostureChangeResponse())
		case "approval_get":
			return mustOKLocalRPCResponse(t, testBackendPostureApprovalGetResponse(t, wire.Request))
		case "approval_resolve":
			return mustOKLocalRPCResponse(t, testBackendPostureApprovalResolveResponse(t, wire.Request))
		default:
			return localRPCResponse{OK: false}
		}
	}
}

func runBackendPostureCommandSequence(t *testing.T, stdout, stderr *bytes.Buffer) {
	t.Helper()
	if err := run([]string{"backend-posture-get"}, stdout, stderr); err != nil {
		t.Fatalf("backend-posture-get returned error: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"backend-posture-change", "--target-backend-kind", "container", "--reduced-assurance-acknowledged"}, stdout, stderr); err != nil {
		t.Fatalf("backend-posture-change returned error: %v", err)
	}
	approvalRequestPath, approvalEnvelopePath, _ := writeApprovalFixtures(t, "human", testDigest("2"), "repo/file.txt", "abc123", "tool-v1")
	stdout.Reset()
	if err := run([]string{"approval-resolve", "--approval-request", approvalRequestPath, "--approval-envelope", approvalEnvelopePath}, stdout, stderr); err != nil {
		t.Fatalf("approval-resolve returned error: %v", err)
	}
}

func assertBackendPostureRequestedOps(t *testing.T, requestedOps []string) {
	t.Helper()
	wantOps := []string{"backend_posture_get", "backend_posture_get", "backend_posture_change", "approval_get", "approval_get", "approval_resolve"}
	if len(requestedOps) != len(wantOps) {
		t.Fatalf("requested operations = %v, want %v", requestedOps, wantOps)
	}
	for i := range wantOps {
		if requestedOps[i] != wantOps[i] {
			t.Fatalf("operation[%d] = %q, want %q", i, requestedOps[i], wantOps[i])
		}
	}
}

func testBackendPostureGetResponse() brokerapi.BackendPostureGetResponse {
	return brokerapi.BackendPostureGetResponse{
		SchemaID:      "runecode.protocol.v0.BackendPostureGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-posture-get",
		Posture: brokerapi.BackendPostureState{
			SchemaID:             "runecode.protocol.v0.BackendPostureState",
			SchemaVersion:        "0.1.0",
			InstanceID:           "launcher-instance-1",
			BackendKind:          "microvm",
			PreferredBackendKind: "microvm",
			Availability: []brokerapi.BackendPostureAvailability{
				{SchemaID: "runecode.protocol.v0.BackendPostureAvailability", SchemaVersion: "0.1.0", BackendKind: "microvm", Available: true},
				{SchemaID: "runecode.protocol.v0.BackendPostureAvailability", SchemaVersion: "0.1.0", BackendKind: "container", Available: true},
			},
		},
	}
}

func testBackendPostureChangeResponse() brokerapi.BackendPostureChangeResponse {
	return brokerapi.BackendPostureChangeResponse{
		SchemaID:      "runecode.protocol.v0.BackendPostureChangeResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-posture-change",
		Outcome: brokerapi.BackendPostureChangeOutcome{
			SchemaID:          "runecode.protocol.v0.BackendPostureChangeOutcome",
			SchemaVersion:     "0.1.0",
			Outcome:           "approval_required",
			OutcomeReasonCode: "approval_required",
			ApprovalID:        testDigest("a"),
		},
		Posture: brokerapi.BackendPostureState{
			SchemaID:          "runecode.protocol.v0.BackendPostureState",
			SchemaVersion:     "0.1.0",
			InstanceID:        "launcher-instance-1",
			BackendKind:       "microvm",
			PendingApproval:   true,
			PendingApprovalID: testDigest("a"),
		},
	}
}

func testBackendPostureApprovalGetResponse(t *testing.T, raw json.RawMessage) brokerapi.ApprovalGetResponse {
	t.Helper()
	request := brokerapi.ApprovalGetRequest{}
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("Unmarshal approval_get request error: %v", err)
	}
	return brokerapi.ApprovalGetResponse{
		SchemaID:      "runecode.protocol.v0.ApprovalGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-approval-get",
		Approval: brokerapi.ApprovalSummary{
			SchemaID:      "runecode.protocol.v0.ApprovalSummary",
			SchemaVersion: "0.1.0",
			ApprovalID:    request.ApprovalID,
			BoundScope: brokerapi.ApprovalBoundScope{
				SchemaID:           "runecode.protocol.v0.ApprovalBoundScope",
				SchemaVersion:      "0.1.0",
				WorkspaceID:        "ws-1",
				RunID:              "run-backend",
				InstanceID:         "launcher-instance-1",
				ActionKind:         "backend_posture_change",
				PolicyDecisionHash: testDigest("b"),
			},
		},
		ApprovalDetail: testBackendPostureApprovalDetail(request.ApprovalID),
	}
}

func testBackendPostureApprovalDetail(approvalID string) brokerapi.ApprovalDetail {
	return brokerapi.ApprovalDetail{
		SchemaID:         "runecode.protocol.v0.ApprovalDetail",
		SchemaVersion:    "0.1.0",
		ApprovalID:       approvalID,
		BindingKind:      "exact_action",
		PolicyReasonCode: "requires_human_review",
		LifecycleDetail: brokerapi.ApprovalLifecycleDetail{
			SchemaID:            "runecode.protocol.v0.ApprovalLifecycleDetail",
			SchemaVersion:       "0.1.0",
			LifecycleState:      "pending",
			LifecycleReasonCode: "awaiting_decision",
		},
		WhatChangesIfApproved: brokerapi.ApprovalWhatChangesIfApproved{
			SchemaID:      "runecode.protocol.v0.ApprovalWhatChangesIfApproved",
			SchemaVersion: "0.1.0",
			Summary:       "Apply reduced-assurance backend posture.",
			EffectKind:    "backend_posture_selection",
		},
		BlockedWorkScope: brokerapi.ApprovalBlockedWorkScope{
			SchemaID:      "runecode.protocol.v0.ApprovalBlockedWorkScope",
			SchemaVersion: "0.1.0",
			ScopeKind:     "action_kind",
			ActionKind:    "backend_posture_change",
		},
		BoundIdentity: brokerapi.ApprovalBoundIdentity{
			SchemaID:              "runecode.protocol.v0.ApprovalBoundIdentity",
			SchemaVersion:         "0.1.0",
			ApprovalRequestDigest: approvalID,
			ManifestHash:          testDigest("c"),
			PolicyDecisionHash:    testDigest("b"),
			BindingKind:           "exact_action",
			BoundActionHash:       testDigest("d"),
		},
		BackendPostureSelection: &brokerapi.ApprovalBackendPostureSelection{
			SchemaID:                     "runecode.protocol.v0.ApprovalBackendPostureSelection",
			SchemaVersion:                "0.1.0",
			TargetInstanceID:             "launcher-instance-1",
			TargetBackendKind:            "container",
			SelectionMode:                "explicit_selection",
			ChangeKind:                   "select_backend",
			AssuranceChangeKind:          "reduce_assurance",
			OptInKind:                    "exact_action_approval",
			ReducedAssuranceAcknowledged: true,
		},
	}
}

func testBackendPostureApprovalResolveResponse(t *testing.T, raw json.RawMessage) brokerapi.ApprovalResolveResponse {
	t.Helper()
	request := brokerapi.ApprovalResolveRequest{}
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("Unmarshal approval_resolve request error: %v", err)
	}
	if request.ResolutionDetails.BackendPostureSelection == nil {
		t.Fatalf("approval_resolve missing backend_posture_selection details: %+v", request)
	}
	return brokerapi.ApprovalResolveResponse{
		SchemaID:             "runecode.protocol.v0.ApprovalResolveResponse",
		SchemaVersion:        "0.1.0",
		RequestID:            "req-approval-resolve",
		ResolutionStatus:     "resolved",
		ResolutionReasonCode: "approval_consumed",
		Approval: brokerapi.ApprovalSummary{
			SchemaID:      "runecode.protocol.v0.ApprovalSummary",
			SchemaVersion: "0.1.0",
			ApprovalID:    request.ApprovalID,
			Status:        "consumed",
		},
	}
}
