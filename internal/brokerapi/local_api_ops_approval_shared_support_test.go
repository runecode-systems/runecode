package brokerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func setupServiceWithApprovalFixture(t *testing.T) (*Service, artifacts.ArtifactReference, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
	return setupServiceWithApprovalFixtureAndOutcome(t, "approve")
}

func setupServiceWithApprovalFixtureAndOutcome(t *testing.T, outcome string) (*Service, artifacts.ArtifactReference, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
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
	unapproved, err := s.Put(artifacts.PutRequest{Payload: []byte("private excerpt"), ContentType: "text/plain", DataClass: artifacts.DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64), CreatedByRole: "workspace", RunID: "run-approval", StepID: "step-1"})
	if err != nil {
		t.Fatalf("Put unapproved returned error: %v", err)
	}
	requestEnv, decisionEnv, verifiers := signedApprovalArtifactsForBrokerTestsWithOutcome(t, "human", unapproved.Digest, outcome)
	seedPendingApprovalForSignedRequest(t, s, "run-approval", "step-1", unapproved.Digest, *requestEnv)
	for _, verifier := range verifiers {
		if putErr := putTrustedVerifierRecordForService(s, verifier); putErr != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", putErr)
		}
	}
	return s, unapproved, requestEnv, decisionEnv
}

func seedPendingApprovalForSignedRequest(t *testing.T, s *Service, runID, stepID, sourceDigest string, requestEnv trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	approvalID, err := approvalIDFromRequest(requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	payload := decodeApprovalPayloadMap(requestEnv.Payload)
	seedPolicyDecisionForPendingApproval(t, s, runID, payload)
	record := pendingPromotionApprovalRecordForRequest(runID, stepID, sourceDigest, approvalID, payload, requestEnv)
	if err := s.RecordApproval(record); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
}

func seedPolicyDecisionForPendingApproval(t *testing.T, s *Service, runID string, payload map[string]any) {
	t.Helper()
	manifestHash := digestFromPayloadField(payload, "manifest_hash")
	actionHash := digestFromPayloadField(payload, "action_request_hash")
	if err := s.RecordPolicyDecision(runID, "", policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        policyengine.DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           manifestHash,
		ActionRequestHash:      actionHash,
		PolicyInputHashes:      []string{manifestHash},
		RelevantArtifactHashes: relevantArtifactHashesFromPayload(payload),
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details:                map[string]any{"precedence": "test_seed"},
	}); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
}

func decodeApprovalPayloadMap(payload []byte) map[string]any {
	out := map[string]any{}
	if err := json.Unmarshal(payload, &out); err != nil {
		panic(err)
	}
	return out
}

func pendingPromotionApprovalRecordForRequest(runID, stepID, sourceDigest, approvalID string, payload map[string]any, requestEnv trustpolicy.SignedObjectEnvelope) artifacts.ApprovalRecord {
	expiresAt := parseRFC3339OrNow(payload, "expires_at")
	return artifacts.ApprovalRecord{
		ApprovalID:             approvalID,
		Status:                 "pending",
		WorkspaceID:            workspaceIDForRun(runID),
		RunID:                  runID,
		StageID:                "artifact_flow",
		StepID:                 stepID,
		ActionKind:             policyengine.ActionKindPromotion,
		RequestedAt:            parseRFC3339OrNow(payload, "requested_at"),
		ExpiresAt:              &expiresAt,
		ApprovalTriggerCode:    stringFieldFromPayload(payload, "approval_trigger_code", "excerpt_promotion"),
		ChangesIfApproved:      stringFieldFromPayload(payload, "changes_if_approved", approvalChangesIfApprovedDefault),
		ApprovalAssuranceLevel: stringFieldFromPayload(payload, "approval_assurance_level", "session_authenticated"),
		PresenceMode:           stringFieldFromPayload(payload, "presence_mode", "os_confirmation"),
		ManifestHash:           digestFromPayloadField(payload, "manifest_hash"),
		ActionRequestHash:      digestFromPayloadField(payload, "action_request_hash"),
		RelevantArtifactHashes: relevantArtifactHashesFromPayload(payload),
		RequestDigest:          approvalID,
		SourceDigest:           sourceDigest,
		RequestEnvelope:        &requestEnv,
	}
}

func relevantArtifactHashesFromPayload(payload map[string]any) []string {
	items, ok := payload["relevant_artifact_hashes"].([]any)
	if !ok {
		return nil
	}
	relevant := make([]string, 0, len(items))
	for _, item := range items {
		digestObj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		hashAlg, _ := digestObj["hash_alg"].(string)
		hash, _ := digestObj["hash"].(string)
		if hashAlg != "" && hash != "" {
			relevant = append(relevant, fmt.Sprintf("%s:%s", hashAlg, hash))
		}
	}
	return relevant
}

func promotionActionHashForBrokerTests(digest, repoPath, commit, extractorVersion, approver string) string {
	actionHash, err := artifacts.CanonicalPromotionActionRequestHash(artifacts.PromotionRequest{UnapprovedDigest: digest, Approver: approver, RepoPath: repoPath, Commit: commit, ExtractorToolVersion: extractorVersion})
	if err != nil {
		panic(err)
	}
	return actionHash
}

func createPendingApprovalFromPolicyDecision(t *testing.T, s *Service, runID, stepID, sourceDigest string) string {
	t.Helper()
	manifestHash := "sha256:" + strings.Repeat("1", 64)
	actionHash := promotionActionHashForBrokerTests(sourceDigest, "repo/file.txt", "abc123", "tool-v1", "human")
	if err := s.RecordPolicyDecision(runID, "", policyengine.PolicyDecision{SchemaID: "runecode.protocol.v0.PolicyDecision", SchemaVersion: "0.3.0", DecisionOutcome: policyengine.DecisionRequireHumanApproval, PolicyReasonCode: "approval_required", ManifestHash: manifestHash, PolicyInputHashes: []string{"sha256:" + strings.Repeat("2", 64)}, ActionRequestHash: actionHash, RelevantArtifactHashes: []string{sourceDigest}, DetailsSchemaID: "runecode.protocol.details.policy.evaluation.v0", Details: map[string]any{"precedence": "approval_profile_moderate"}, RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.moderate.workspace_write.v0", RequiredApproval: map[string]any{"approval_trigger_code": "excerpt_promotion", "approval_assurance_level": "session_authenticated", "presence_mode": "os_confirmation", "scope": map[string]any{"schema_id": "runecode.protocol.v0.ApprovalBoundScope", "schema_version": "0.1.0", "workspace_id": workspaceIDForRun(runID), "run_id": runID, "stage_id": "artifact_flow", "step_id": stepID, "action_kind": "promotion"}, "changes_if_approved": "Promote reviewed file excerpts for downstream use.", "approval_ttl_seconds": 1800}}); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	for _, rec := range s.ApprovalList() {
		if rec.RunID == runID && rec.Status == "pending" {
			return rec.ApprovalID
		}
	}
	t.Fatalf("missing pending approval for run %q", runID)
	return ""
}

func putTrustedVerifierRecordForService(service *Service, record trustpolicy.VerifierRecord) error {
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	ref, err := service.Put(artifacts.PutRequest{Payload: b, ContentType: "application/json", DataClass: artifacts.DataClassAuditVerificationReport, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "auditd", TrustedSource: true})
	if err != nil {
		return err
	}
	return service.AppendTrustedAuditEvent(artifacts.TrustedContractImportAuditEventType, "brokerapi", map[string]interface{}{artifacts.TrustedContractImportKindDetailKey: artifacts.TrustedContractImportKindVerifierRecord, artifacts.TrustedContractImportArtifactDigestDetailKey: ref.Digest, artifacts.TrustedContractImportProvenanceDetailKey: "sha256:" + strings.Repeat("1", 64)})
}

func setupServiceWithStageSignOffApprovalFixture(t *testing.T) (*Service, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
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
	requestEnv, decisionEnv, verifiers := signedStageSummaryApprovalArtifactsForBrokerTests(t, "human", "run-stage", "stage-1", "stage-signoff-a", 1, "approve")
	for _, verifier := range verifiers {
		if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
		}
	}
	seedPendingStageSignOffApprovalForSignedRequest(t, s, "run-stage", "stage-1", *requestEnv)
	return s, requestEnv, decisionEnv
}

func setupServiceWithSupersededStageSignOffApprovals(t *testing.T) (*Service, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, string) {
	s, oldReq, oldDec := setupServiceWithStageSignOffApprovalFixture(t)
	newReq, _, _ := signedStageSummaryApprovalArtifactsForBrokerTests(t, "human", "run-stage", "stage-1", "stage-signoff-b", 2, "approve")
	newApprovalID := seedPendingStageSignOffApprovalForSignedRequest(t, s, "run-stage", "stage-1", *newReq)
	return s, oldReq, oldDec, newApprovalID
}

func setupServiceWithPlanScopedSupersededStageSignOffApprovals(t *testing.T) (*Service, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, string) {
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
	baseNow := time.Now().UTC()
	oldReq, oldDec, verifiers := signedStageSummaryApprovalArtifactsForBrokerTestsWithPlanAt(t, "human", "run-stage", "plan-a", "stage-1", "stage-signoff-plan", 1, "approve", baseNow)
	for _, verifier := range verifiers {
		if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
		}
	}
	_ = seedPendingStageSignOffApprovalForSignedRequest(t, s, "run-stage", "stage-1", *oldReq)
	newReq, _, _ := signedStageSummaryApprovalArtifactsForBrokerTestsWithPlanAt(t, "human", "run-stage", "plan-b", "stage-1", "stage-signoff-plan", 1, "approve", baseNow.Add(time.Minute))
	newApprovalID := seedPendingStageSignOffApprovalForSignedRequest(t, s, "run-stage", "stage-1", *newReq)
	return s, oldReq, oldDec, newApprovalID
}

func setupServiceWithBackendPostureApprovalFixture(t *testing.T) (*Service, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
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
	requestEnv, decisionEnv, verifiers := signedBackendPostureApprovalArtifactsForBrokerTests(t, "human", "container", "explicit_selection", "select_backend", "reduce_assurance", "exact_action_approval", "approve")
	for _, verifier := range verifiers {
		if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
		}
	}
	seedPendingBackendPostureApprovalForSignedRequest(t, s, *requestEnv)
	return s, requestEnv, decisionEnv
}

func seedPendingStageSignOffApprovalForSignedRequest(t *testing.T, s *Service, runID, stageID string, requestEnv trustpolicy.SignedObjectEnvelope) string {
	t.Helper()
	approvalID, err := approvalIDFromRequest(requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(requestEnv.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal approval request payload returned error: %v", err)
	}
	seedPolicyDecisionForPendingApproval(t, s, runID, payload)
	requestedAt := parseRFC3339OrNow(payload, "requested_at")
	expiresAt := parseRFC3339OrNow(payload, "expires_at")
	if err := s.RecordApproval(artifacts.ApprovalRecord{ApprovalID: approvalID, Status: "pending", WorkspaceID: workspaceIDForRun(runID), RunID: runID, StageID: stageID, ActionKind: policyengine.ActionKindStageSummarySign, RequestedAt: requestedAt, ExpiresAt: &expiresAt, ApprovalTriggerCode: stringFieldFromPayload(payload, "approval_trigger_code", "stage_sign_off"), ChangesIfApproved: stringFieldFromPayload(payload, "changes_if_approved", "Stage summary is signed off for this exact summary hash."), ApprovalAssuranceLevel: stringFieldFromPayload(payload, "approval_assurance_level", "session_authenticated"), PresenceMode: stringFieldFromPayload(payload, "presence_mode", "os_confirmation"), ManifestHash: digestFromPayloadField(payload, "manifest_hash"), ActionRequestHash: digestFromPayloadField(payload, "action_request_hash"), RequestDigest: approvalID, SourceDigest: "", RequestEnvelope: &requestEnv}); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	return approvalID
}

func stageSignOffActionHashForBrokerTests(runID, planID, stageID string, stageSummary map[string]any, stageSummaryDigest string, summaryRevision int64) string {
	manifestHash, err := digestFromIdentity("sha256:" + strings.Repeat("1", 64))
	if err != nil {
		panic(err)
	}
	stageSummaryHash, err := digestFromIdentity(stageSummaryDigest)
	if err != nil {
		panic(err)
	}
	action, err := policyengine.NewStageSummarySignOffAction(policyengine.StageSummarySignOffActionInput{ActionEnvelope: policyengine.ActionEnvelope{CapabilityID: "cap_stage", Actor: policyengine.ActionActor{ActorKind: "daemon", RoleFamily: "workspace", RoleKind: "workspace-edit"}}, RunID: runID, PlanID: planID, StageID: stageID, ManifestHash: manifestHash, StageSummaryHash: stageSummaryHash, ApprovalProfile: "moderate", SummaryRevision: &summaryRevision, StageSummary: stageSummary})
	if err != nil {
		panic(err)
	}
	hash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		panic(err)
	}
	return hash
}

func seedPendingBackendPostureApprovalForSignedRequest(t *testing.T, s *Service, requestEnv trustpolicy.SignedObjectEnvelope) string {
	t.Helper()
	approvalID, err := approvalIDFromRequest(requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	payload := mustApprovalPayloadForSeed(t, requestEnv)
	runID := "run-backend"
	seedPolicyDecisionForPendingApproval(t, s, runID, payload)
	record := pendingBackendPostureApprovalRecord(runID, approvalID, payload, requestEnv)
	if err := s.RecordApproval(record); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	backfillBackendPostureApprovalBinding(t, s, approvalID, requestEnv)
	return approvalID
}

func mustApprovalPayloadForSeed(t *testing.T, requestEnv trustpolicy.SignedObjectEnvelope) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(requestEnv.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal approval request payload returned error: %v", err)
	}
	return payload
}

func pendingBackendPostureApprovalRecord(runID, approvalID string, payload map[string]any, requestEnv trustpolicy.SignedObjectEnvelope) artifacts.ApprovalRecord {
	requestedAt := parseRFC3339OrNow(payload, "requested_at")
	expiresAt := parseRFC3339OrNow(payload, "expires_at")
	return artifacts.ApprovalRecord{
		ApprovalID:             approvalID,
		Status:                 "pending",
		WorkspaceID:            workspaceIDForRun(runID),
		RunID:                  runID,
		ActionKind:             policyengine.ActionKindBackendPosture,
		RequestedAt:            requestedAt,
		ExpiresAt:              &expiresAt,
		ApprovalTriggerCode:    stringFieldFromPayload(payload, "approval_trigger_code", "reduced_assurance_backend"),
		ChangesIfApproved:      stringFieldFromPayload(payload, "changes_if_approved", "Reduced-assurance backend posture change may be applied."),
		ApprovalAssuranceLevel: stringFieldFromPayload(payload, "approval_assurance_level", "reauthenticated"),
		PresenceMode:           stringFieldFromPayload(payload, "presence_mode", "hardware_touch"),
		ManifestHash:           digestFromPayloadField(payload, "manifest_hash"),
		ActionRequestHash:      digestFromPayloadField(payload, "action_request_hash"),
		RequestDigest:          approvalID,
		SourceDigest:           "",
		RequestEnvelope:        &requestEnv,
	}
}

func backfillBackendPostureApprovalBinding(t *testing.T, s *Service, approvalID string, requestEnv trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	rec, ok := s.ApprovalGet(approvalID)
	if !ok {
		return
	}
	instanceID := backendPostureTargetInstanceFromPayload(requestEnv.Payload)
	if instanceID == "" {
		return
	}
	rec.InstanceID = instanceID
	rec.PolicyDecisionHash = policyDecisionHashForStoredApproval(t, s, approvalID)
	if err := s.RecordApproval(rec); err != nil {
		t.Fatalf("RecordApproval(instance backfill) returned error: %v", err)
	}
}

func backendPostureTargetInstanceFromPayload(payload []byte) string {
	details := map[string]any{}
	if err := json.Unmarshal(payload, &details); err != nil {
		return ""
	}
	requestDetails, _ := details["details"].(map[string]any)
	instanceID, _ := requestDetails["target_instance_id"].(string)
	return instanceID
}

func backendPostureActionHashForBrokerTests(backendKind, selectionMode, changeKind, assuranceChangeKind, optInKind string) string {
	action := policyengine.NewBackendPostureChangeAction(policyengine.BackendPostureChangeActionInput{ActionEnvelope: policyengine.ActionEnvelope{CapabilityID: "cap_stage", Actor: policyengine.ActionActor{ActorKind: "daemon", RoleFamily: "workspace", RoleKind: "workspace-edit"}}, RunID: "instance-control:launcher-instance-1", TargetInstanceID: "launcher-instance-1", TargetBackendKind: backendKind, SelectionMode: selectionMode, ChangeKind: changeKind, AssuranceChangeKind: assuranceChangeKind, OptInKind: optInKind, ReducedAssuranceAcknowledged: true, Reason: "operator_requested_reduced_assurance_backend_opt_in"})
	hash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		panic(err)
	}
	return hash
}

func assertApprovalAndAuditReadEndpoints(t *testing.T, s *Service, approvalID string) {
	t.Helper()
	_, _ = s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get", ApprovalID: approvalID}, RequestContext{})
}

func policyDecisionHashForStoredApproval(t *testing.T, s *Service, approvalID string) string {
	t.Helper()
	rec, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	if rec.PolicyDecisionHash == "" {
		t.Fatalf("ApprovalGet(%q) missing policy_decision_hash", approvalID)
	}
	return rec.PolicyDecisionHash
}

func assertVersionAndLogEndpoints(t *testing.T, s *Service) {
	t.Helper()
	assertReadinessAndVersionEndpoints(t, s)
	assertLogStreamEndpoints(t, s)
}

func assertReadinessAndVersionEndpoints(t *testing.T, s *Service) { t.Helper() }
func assertLogStreamEndpoints(t *testing.T, s *Service)           { t.Helper() }
